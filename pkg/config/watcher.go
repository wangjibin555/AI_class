package config

import (
	"AI_class/pkg/logger"
	"AI_class/pkg/provider"
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	ErrConfigClientNotInitialized = fmt.Errorf("config client not initialized")
	ErrWatchNamespace             = fmt.Errorf("fail to Watch Namespace")
	ErrNamespaceNotRegistered     = "namespace %s not registered"
)

// 配置热更新的桥接层面，实现配置更新时候自动感知
// 命名空间配置变更回调,当配置文件变更时，会接收到解析后的配置对象
type NamespaceWatchCallback func(namespace string, oldConfig, newConfig interface{})

// 监听错误处理器
type NamespaceWatchErrorHandler func(namespace string, err error)

type WatchOption func(*watchOptions)

type watchOptions struct {
	errorHandler NamespaceWatchErrorHandler
	debounce     time.Duration
	async        bool
}

func WithErrorHandler(handler NamespaceWatchErrorHandler) WatchOption {
	return func(options *watchOptions) {
		options.errorHandler = handler
	}
}

func withDebounce(duration time.Duration) WatchOption {
	return func(options *watchOptions) {
		options.debounce = duration
	}
}

func WithSync() WatchOption {
	return func(options *watchOptions) {
		options.async = false
	}
}

func WithAsync() WatchOption {
	return func(options *watchOptions) {
		options.async = true
	}
}

// 全局监听管理
func WatchNamespace(ctx context.Context, namespace string, callback NamespaceWatchCallback, options ...WatchOption) error {
	if gloableClient == nil {
		return ErrConfigClientNotInitialized
	}
	return gloableClient.WatchNamespace(ctx, namespace, callback, options...)
}

func UnwatchNamespace(namespace string) error {
	if gloableClient == nil {
		return ErrConfigClientNotInitialized
	}
	return gloableClient.UnwatchNamespace(namespace)
}

func ReloadNamespace(ctx context.Context, namespace string) error {
	if gloableClient == nil {
		return fmt.Errorf("config client not initialized")
	}
	return gloableClient.ReloadNamespace(ctx, namespace)
}

func StopAllWatch() {
	if gloableClient != nil {
		gloableClient.StopAllWatchers()
	}
}

func GetWatchNamespace() []string {
	if gloableClient == nil {
		return nil
	}
	return gloableClient.GetWatchedNamespaces()
}

func WatchAllNamespaces(ctx context.Context, callback NamespaceWatchCallback, options ...WatchOption) error {
	namespaces := GetRegisteredNamespaces()
	successCount := 0
	for _, ns := range namespaces {
		if err := WatchNamespace(ctx, ns, callback, options...); err != nil {
			logger.Error("Fail to watch namespace %s:%v", ns, err)
		} else {
			successCount++
		}
	}
	if successCount == 0 && len(namespaces) > 0 {
		return ErrWatchNamespace
	}
	logger.Info("Started watching %d/%d namespaces", successCount, len(namespaces))
	return nil
}

// 监听命名空间配置更新
func (c *ConfigClient) WatchNamespace(ctx context.Context, namespace string, callback NamespaceWatchCallback, options ...WatchOption) error {
	opts := &watchOptions{
		async: true, //默认异步回调
	}
	for _, opt := range options {
		opt(opts)
	}

	if !isNamespaceRegistered(namespace) {
		return fmt.Errorf(ErrNamespaceNotRegistered, namespace)
	}

	c.mu.Lock()
	if c.cancelFuncs == nil {
		c.cancelFuncs = make(map[string]provider.CancelFunc)
	}
	if c.watcher == nil {
		c.watcher = make(map[string][]NamespaceWatchCallback)
	}

	if _, exists := c.cancelFuncs[namespace]; exists {
		//添加回调
		c.watcher[namespace] = append(c.watcher[namespace], callback)
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	//创建provider层回调
	providerCallback := c.createProviderCallback(ctx, namespace, opts)

	//调用provider层监听
	cancelFunc, err := c.provider.Watch(ctx, namespace, providerCallback)
	if err != nil {
		logger.Error("Failed to Watch %s Namespace", namespace)
		return err
	}

	//保存回调和取消函数
	c.mu.Lock()
	c.watcher[namespace] = append(c.watcher[namespace], callback)
	c.cancelFuncs[namespace] = cancelFunc
	c.mu.Unlock()

	logger.Info("Started watching namespace: %s", namespace)
	return nil
}

// 创建provider层回调
func (c *ConfigClient) createProviderCallback(ctx context.Context, namespace string, opts *watchOptions) provider.WatchCallback {
	if opts.debounce > 0 {
		return c.createProviderCallback(ctx, namespace, opts)
	}

	return func(event *provider.WatchEvent) {
		c.handleConfigChange(ctx, namespace, event, opts)
	}
}

func (c *ConfigClient) createdebounceCallback(ctx context.Context, namespace string, opts *watchOptions) provider.WatchCallback {
	var (
		timer     *time.Timer
		mu        sync.Mutex
		lastEvent *provider.WatchEvent
		pending   bool
	)
	return func(event *provider.WatchEvent) {
		mu.Lock()
		defer mu.Unlock()

		lastEvent = event
		pending = true

		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(opts.debounce, func() {
			mu.Lock()
			if pending && lastEvent != nil {
				c.handleConfigChange(ctx, namespace, lastEvent, opts)
			}
			mu.Unlock()
		})
	}
}

// 处理资源位变更
func (c *ConfigClient) handleConfigChange(ctx context.Context, namespace string, event *provider.WatchEvent, opts *watchOptions) {
	c.mu.Lock()
	oldConfig := c.namespaces[namespace]
	c.mu.Unlock()

	//解析配置
	newConfig, err := parseNamespaceConfig(namespace, event.NewValue)
	if err != nil {
		logger.Error("Failed to parse namespace %s config: %v", namespace, err)
		if opts.errorHandler != nil {
			opts.errorHandler(namespace, fmt.Errorf("parse config failed: %v", err))
		}
		return
	}

	if c.cache != nil {
		cacheKey := BuildConfigKey(namespace, "")
		if err = c.cache.Set(ctx, cacheKey, event.NewValue, 0); err != nil {
			logger.Error("Failed to cache namespace %s config: %v", namespace, err)
		}
	}

	// 4. 更新内存中的配置
	c.mu.Lock()
	c.namespaces[namespace] = newConfig
	c.mu.Unlock()

	// 5. 触发所有回调
	c.mu.RLock()
	callbacks := make([]NamespaceWatchCallback, len(c.watcher[namespace]))
	copy(callbacks, c.watcher[namespace])
	c.mu.RUnlock()

	for _, callback := range callbacks {
		if opts.async {
			// 异步执行回调
			go c.safeCallCallback(callback, namespace, oldConfig, newConfig)
		} else {
			// 同步执行回调
			c.safeCallCallback(callback, namespace, oldConfig, newConfig)
		}
	}
}

// safeCallCallback 安全地调用回调函数（防止 panic）
func (c *ConfigClient) safeCallCallback(callback NamespaceWatchCallback, namespace string, oldConfig, newConfig interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Namespace %s callback panic: %v", namespace, r)
		}
	}()

	callback(namespace, oldConfig, newConfig)
}

// UnwatchNamespace 停止监听命名空间
func (c *ConfigClient) UnwatchNamespace(namespace string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFuncs == nil {
		return fmt.Errorf("no watchers registered")
	}

	cancelFunc, exists := c.cancelFuncs[namespace]
	if !exists {
		return fmt.Errorf("namespace %s not watched", namespace)
	}

	// 取消 Provider 层监听
	cancelFunc()

	// 清理数据
	delete(c.cancelFuncs, namespace)
	delete(c.watcher, namespace)

	logger.Info("Stopped watching namespace: %s", namespace)
	return nil
}

// StopAllWatchers 停止所有监听器
func (c *ConfigClient) StopAllWatchers() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFuncs == nil {
		return
	}

	for namespace, cancelFunc := range c.cancelFuncs {
		cancelFunc()
		logger.Info("Stopped watching namespace: %s", namespace)
	}

	c.cancelFuncs = make(map[string]provider.CancelFunc)
	c.watcher = make(map[string][]NamespaceWatchCallback)
}

// ReloadNamespace 手动重新加载命名空间配置
func (c *ConfigClient) ReloadNamespace(ctx context.Context, namespace string) error {
	logger.Info("Manually reloading namespace: %s", namespace)
	return c.loadNamespace(ctx, namespace)
}

// ReloadAllNamespaces 手动重新加载所有命名空间配置
func (c *ConfigClient) ReloadAllNamespaces(ctx context.Context) error {
	logger.Info("Manually reloading all namespaces")
	return c.loadAllNamespace(ctx)
}

// GetWatchedNamespaces 获取当前正在监听的命名空间列表
func (c *ConfigClient) GetWatchedNamespaces() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cancelFuncs == nil {
		return nil
	}

	namespaces := make([]string, 0, len(c.cancelFuncs))
	for ns := range c.cancelFuncs {
		namespaces = append(namespaces, ns)
	}

	return namespaces
}

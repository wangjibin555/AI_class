package config

import (
	"AI_class/pkg/logger"
	"AI_class/pkg/provider"
	"context"
	"fmt"
	"sync"
	"time"
)

// 客户端核心
var (
	ErrGloableClient = "Config: namespace %s not found"
)

type ConfigClient struct {
	provider   provider.ConfigProvider
	cache      Cache
	namespaces map[string]interface{}
	mu         sync.RWMutex
	opts       *ClientOptions

	//监听器
	watcher     map[string][]NamespaceWatchCallback
	cancelFuncs map[string]provider.CancelFunc
}

type ClientOptions struct {
	EnableCache  bool
	CacheOptions []CacheOption
	Environment  string
	Region       string
}

var (
	gloableClient *ConfigClient
	initOnce      sync.Once
)

// 按照配置模式初始化
func Init(ctx context.Context, p provider.ConfigProvider, opts ...ClientOption) error {
	var err error
	initOnce.Do(func() {
		option := &ClientOptions{
			Environment: "dev",
			EnableCache: true,
		}
		for _, opt := range opts {
			opt(option)
		}

		var cache Cache
		// 这个地方指定当前究竟使用的是哪个实现层
		if option.EnableCache {
			cache = NewMemoryCache(option.CacheOptions...)
		}
		gloableClient = &ConfigClient{
			provider:   p,
			cache:      cache,
			namespaces: make(map[string]interface{}),
			opts:       option,
		}

		err = gloableClient.loadAllNamespace(ctx)
	})
	return err
}

// ClientOption 客户端选项函数
type ClientOption func(*ClientOptions)

func Get(namespace string) (interface{}, error) {
	if gloableClient == nil {
		return nil, fmt.Errorf(ErrGloableClient, namespace)
	}
	return gloableClient.GetNamespace(namespace)
}

func (c *ConfigClient) GetNamespace(namespace string) (interface{}, error) {
	c.mu.RLock()
	config, exists := c.namespaces[namespace]
	defer c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf(ErrGloableClient, namespace)
	}
	return config, nil
}

// 加载所有命名空间
func (c *ConfigClient) loadAllNamespace(ctx context.Context) error {
	namespaces := GetRegisteredNamespaces()

	for _, ns := range namespaces {
		if err := c.loadNamespace(ctx, ns); err != nil {
			return err
		}
	}
	return nil
}

// 依据给定命名空间，加载单个命名空间
func (c *ConfigClient) loadNamespace(ctx context.Context, namespace string) error {
	//先从缓存中获取
	if c.cache != nil {
		cacheKey := BuildConfigKey(namespace, "")
		if data, err := c.cache.Get(ctx, cacheKey); err != nil {
			return err
		} else {
			config, err := parseNamespaceConfig(namespace, data)
			if err == nil {
				c.mu.Lock()
				c.namespaces[namespace] = config
				c.mu.Unlock()
				return nil
			}
		}
	}

	//缓存中失败则从provider中获取
	data, err := c.provider.GetRaw(ctx, namespace)
	if err != nil {
		return err
	}

	//解析对应获取到的配置
	config, err := parseNamespaceConfig(namespace, data)
	if err != nil {
		return err
	}

	//保存配置到缓存
	if c.cache != nil {
		cacheKey := BuildConfigKey(namespace, "")
		c.cache.Set(ctx, cacheKey, data, 0)
	}
	//保存配置到内存
	c.mu.Lock()
	c.namespaces[namespace] = config
	c.mu.Unlock()
	return nil
}

func WithCache(opts ...CacheOption) ClientOption {
	return func(o *ClientOptions) {
		o.EnableCache = true
		o.CacheOptions = opts
	}
}

// WithEnvironment 设置环境
func WithEnvironment(env string) ClientOption {
	return func(o *ClientOptions) {
		o.Environment = env
	}
}

// Shutdown 关闭配置客户端，清理所有资源
// 应该在服务关闭时调用，确保所有监听器和缓存正确清理
//
// 参数：
// timeout: 清理超时时间，0 表示无限等待
func Shutdown(timeout time.Duration) {
	if gloableClient == nil {
		return
	}

	logger.Info("Shutting down config client...")

	// 1. 停止所有监听器
	gloableClient.StopAllWatchers()
	logger.Info("All watchers stopped")

	// 2. 清理缓存
	if gloableClient.cache != nil {
		ctx := context.Background()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		if err := gloableClient.cache.Clear(ctx); err != nil {
			logger.Error("Failed to clear cache: %v", err)
		} else {
			logger.Info("Cache cleared")
		}
	}

	// 3. 关闭 Provider
	if closer, ok := gloableClient.provider.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Failed to close provider: %v", err)
		} else {
			logger.Info("Provider closed")
		}
	}
	logger.Info("Config client shutdown completed")
}

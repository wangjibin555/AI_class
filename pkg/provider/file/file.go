package file

import (
	"AI_class/pkg/logger"
	"AI_class/pkg/provider"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	_ "time"
)

type providerState int32

const (
	stateUninitialized providerState = iota
	stateInitialized
	stateClosed
)

// FileProvider 本地文件配置提供者
// 【说明】适用于开发环境和测试环境，生产环境建议使用 Nacos/Consul
type FileProvider struct {
	// 配置选项
	opts *provider.ProviderOptions

	// 配置源路径（文件或目录）
	configPath string

	// 已加载的配置数据
	// key: 配置文件相对路径（不含扩展名），value: 解析后的数据
	data map[string]interface{}

	// 扁平化的配置数据（用于快速查找）
	// key: 完整的配置键路径，如 "redis.host"
	flatData map[string]interface{}

	// 并发控制
	mu sync.RWMutex

	// 提供者状态
	state atomic.Int32

	logger *logger.ContextLogger

	// 监听相关字段
	watchers       map[string][]provider.WatchCallback // 键 -> 回调列表
	prefixWatchers map[string][]provider.WatchCallback // 前缀 -> 回调列表
	watcherMutex   sync.RWMutex                        // 监听器并发控制
	stopWatcher    chan struct{}                       // 停止监听信号
	watcherWg      sync.WaitGroup                      // 等待监听 goroutine 结束
}

// NewFileProvider 创建文件配置提供者
// configPath: 配置文件路径（单个文件）或配置目录（多个文件）
func NewFileProvider(configPath string) *FileProvider {
	return &FileProvider{
		configPath:     configPath,
		data:           make(map[string]interface{}),
		flatData:       make(map[string]interface{}),
		watchers:       make(map[string][]provider.WatchCallback), // 初始化监听器映射
		prefixWatchers: make(map[string][]provider.WatchCallback), // 初始化前缀监听器映射
		stopWatcher:    make(chan struct{}),
	}
}

// Init 初始化提供者
// 1. 应用配置选项
// 2. 验证配置路径
// 3. 加载配置文件
// 4. 【未来】启动文件监听
func (p *FileProvider) Init(ctx context.Context, opts ...provider.Option) error {
	//CAS进行状态流转
	if !p.state.CompareAndSwap(int32(stateUninitialized), int32(stateInitialized)) {
		stage := providerState(p.state.Load())
		//幂等性，当前状态已经是目标状态
		if stage == stateInitialized {
			return nil
		}
		if p.logger != nil {
			p.logger.Warn("Init failed: provider already closed")
		}
		return provider.NewConfigError("Init", "", provider.ErrProviderClosed)
	}
	//更新配置信息
	p.opts = provider.ApplyOptions(opts...)

	// 初始化 logger
	p.logger = logger.WithContext(ctx)

	//进行校验配置路径
	info, err := os.Stat(p.configPath)
	if err != nil {
		p.state.Store(int32(stateUninitialized))
		if p.logger != nil {
			p.logger.Error("Init failed: config path not found, path=%s, error=%v", p.configPath, err)
		}
		return provider.NewConfigError("Init", p.configPath, fmt.Errorf("path not found: %w", err))
	}

	//进行拉取配置
	if info.IsDir() {
		err = p.loadDirectory(ctx)
	} else {
		err = p.loadFile(ctx, p.configPath)
	}

	//拉取配置失败，回滚
	if err != nil {
		p.state.Store(int32(stateUninitialized))
		if p.logger != nil {
			p.logger.Error("Init failed: load config error, path=%s, error=%v", p.configPath, err)
		}
		return err
	}

	// 如果监听模式不是 Push 模式，则启动轮询监听器
	if p.opts.WatchMode != provider.WatchModePush { // Push 模式需要外部配置中心支持
		if err := p.startPollingWatcher(ctx); err != nil {
			// 监听启动失败不影响初始化，只记录警告
			// 【未来扩展】记录日志
		}
	}

	return nil
}

// Close 关闭提供者
// 1. 停止文件监听
// 2. 清理资源
func (p *FileProvider) Close(ctx context.Context) error {
	// 检查状态
	if !p.state.CompareAndSwap(int32(stateInitialized), int32(stateClosed)) {
		return nil // 幂等：未初始化或已关闭则直接返回
	}

	// 停止监听器
	close(p.stopWatcher)

	// 等待监听 goroutine 结束（设置超时避免无限等待）
	done := make(chan struct{})
	go func() {
		p.watcherWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 正常结束
	case <-time.After(5 * time.Second):
		// 超时，强制结束
		// 【未来扩展】记录超时警告
	}

	// 清理监听器数据
	p.watcherMutex.Lock()
	p.watchers = nil
	p.prefixWatchers = nil
	p.watcherMutex.Unlock()

	// 清理数据
	p.mu.Lock()
	p.data = nil
	p.flatData = nil
	p.mu.Unlock()

	return nil
}

// HealthCheck 健康检查
// 检查配置文件是否仍然可访问
func (p *FileProvider) HealthCheck(ctx context.Context) error {
	// 检查状态
	if providerState(p.state.Load()) != stateInitialized {
		logger.Warn("当前文件配置中心未初始化")
		return provider.NewConfigError("HealthCheck", "", provider.ErrNotInitialized)
	}

	// 检查配置路径是否可访问
	_, err := os.Stat(p.configPath)
	if err != nil {
		if p.logger != nil {
			p.logger.Error("HealthCheck failed: config path not accessible, path=%s, error=%v", p.configPath, err)
		}
		return provider.NewConfigError("HealthCheck", p.configPath, provider.ErrHealthCheck)
	}

	// 【未来扩展】检查文件内容是否有变化
	// 【未来扩展】检查文件权限

	return nil
}

// IsInitialized 检查是否已初始化
func (p *FileProvider) IsInitialized() bool {
	return providerState(p.state.Load()) == stateInitialized
}

// ============ 配置重载 ============

// Reload 重新加载配置
// 【未来扩展】支持热重载
func (p *FileProvider) Reload(ctx context.Context) error {
	if err := p.ensureInitialized(); err != nil {
		return err
	}

	// 清空旧数据
	p.mu.Lock()
	p.data = make(map[string]interface{})
	p.flatData = make(map[string]interface{})
	p.mu.Unlock()

	// 重新加载
	info, err := os.Stat(p.configPath)
	if err != nil {
		return provider.NewConfigError("Reload", p.configPath, err)
	}

	if info.IsDir() {
		return p.loadDirectory(ctx)
	}
	return p.loadFile(ctx, p.configPath)
}

// GetRaw 获取原始配置值（JSON 字节）
func (p *FileProvider) GetRaw(ctx context.Context, key string) ([]byte, error) {
	if err := p.ensureInitialized(); err != nil {
		return nil, provider.NewConfigError("GetRaw", key, err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// 查找配置值
	value, found := p.flatData[key]
	if !found {
		// 尝试从嵌套结构中查找
		value, found = provider.GetNestedValue(p.data, key)
		if !found {
			return nil, provider.NewConfigError("GetRaw", key, provider.ErrKeyNotFound)
		}
	}

	// 序列化为 JSON
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, provider.NewConfigError("GetRaw", key, fmt.Errorf("marshal value: %w", err))
	}

	return bytes, nil
}

// GetRawWithDefault 获取原始配置值，不存在时返回默认值
func (p *FileProvider) GetRawWithDefault(ctx context.Context, key string, defaultVal []byte) []byte {
	val, err := p.GetRaw(ctx, key)
	if err != nil {
		return defaultVal
	}
	return val
}

// ============ 配置获取 - 类型安全 ============

// GetString 获取字符串配置
func (p *FileProvider) GetString(ctx context.Context, key string) (string, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return "", err
	}
	result, err := provider.ToString(val)
	if err != nil {
		return "", provider.NewConfigError("GetString", key, err)
	}
	return result, nil
}

// GetInt 获取整数配置
func (p *FileProvider) GetInt(ctx context.Context, key string) (int, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return 0, err
	}
	result, err := provider.ToInt(val)
	if err != nil {
		return 0, provider.NewConfigError("GetInt", key, err)
	}
	return result, nil
}

// GetBool 获取布尔配置
func (p *FileProvider) GetBool(ctx context.Context, key string) (bool, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return false, err
	}
	result, err := provider.ToBool(val)
	if err != nil {
		return false, provider.NewConfigError("GetBool", key, err)
	}
	return result, nil
}

// GetFloat64 获取浮点数配置
func (p *FileProvider) GetFloat64(ctx context.Context, key string) (float64, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return 0, err
	}
	result, err := provider.ToFloat64(val)
	if err != nil {
		return 0, provider.NewConfigError("GetFloat64", key, err)
	}
	return result, nil
}

// GetDuration 获取时间间隔配置
// 支持格式: "5s", "1m30s", "2h", 或毫秒数
func (p *FileProvider) GetDuration(ctx context.Context, key string) (time.Duration, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return 0, err
	}
	result, err := provider.ToDuration(val)
	if err != nil {
		return 0, provider.NewConfigError("GetDuration", key, err)
	}
	return result, nil
}

// GetStringSlice 获取字符串数组配置
func (p *FileProvider) GetStringSlice(ctx context.Context, key string) ([]string, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return nil, err
	}
	result, err := provider.ToStringSlice(val)
	if err != nil {
		return nil, provider.NewConfigError("GetStringSlice", key, err)
	}
	return result, nil
}

// GetStringMap 获取字符串映射配置
func (p *FileProvider) GetStringMap(ctx context.Context, key string) (map[string]string, error) {
	val, err := p.getValue(ctx, key)
	if err != nil {
		return nil, err
	}
	result, err := provider.ToStringMap(val)
	if err != nil {
		return nil, provider.NewConfigError("GetStringMap", key, err)
	}
	return result, nil
}

// Unmarshal 将配置绑定到结构体
// key: 配置键路径，如 "infrastructure.redis"
// target: 目标结构体指针
func (p *FileProvider) Unmarshal(ctx context.Context, key string, target interface{}) error {
	if err := p.ensureInitialized(); err != nil {
		return provider.NewConfigError("Unmarshal", key, err)
	}

	// 获取原始数据
	data, err := p.GetRaw(ctx, key)
	if err != nil {
		return err
	}

	// 反序列化到目标结构体
	if err := json.Unmarshal(data, target); err != nil {
		return provider.NewConfigError("Unmarshal", key, fmt.Errorf("unmarshal: %w", err))
	}

	return nil
}

// UnmarshalKey 将指定键的配置绑定到结构体（Unmarshal 的别名）
// 【说明】保持接口兼容性，功能与 Unmarshal 相同
func (p *FileProvider) UnmarshalKey(ctx context.Context, key string, target interface{}) error {
	return p.Unmarshal(ctx, key, target)
}

// GetMetadata 获取配置的元数据
// 【说明】文件提供者的元数据相对简单，主要包含：
// - 键名、内容类型、大小
// - 版本和时间戳（文件提供者中简化处理）
func (p *FileProvider) GetMetadata(ctx context.Context, key string) (*provider.ConfigMetadata, error) {
	if err := p.ensureInitialized(); err != nil {
		return nil, provider.NewConfigError("GetMetadata", key, err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// 检查键是否存在
	value, found := p.flatData[key]
	if !found {
		// 尝试从嵌套结构中查找
		value, found = provider.GetNestedValue(p.data, key)
		if !found {
			return nil, provider.NewConfigError("GetMetadata", key, provider.ErrKeyNotFound)
		}
	}

	// 序列化为 JSON 以计算大小和 MD5
	jsonData, err := json.Marshal(value)
	if err != nil {
		return nil, provider.NewConfigError("GetMetadata", key, fmt.Errorf("marshal value: %w", err))
	}

	// 计算 MD5
	hash := md5.Sum(jsonData)
	md5Str := fmt.Sprintf("%x", hash)

	// 推断内容类型
	contentType := p.inferContentType(key, value)

	// 构建元数据
	metadata := &provider.ConfigMetadata{
		Key:         key,
		Version:     0,           // 【未来扩展】文件提供者可以基于文件修改时间或内容哈希生成版本号
		CreateTime:  time.Time{}, // 【未来扩展】可以从文件系统获取创建时间
		UpdateTime:  time.Now(),  // 【简化】使用当前时间，实际应该从文件系统获取
		ContentType: contentType,
		Size:        int64(len(jsonData)),
		MD5:         md5Str,
	}

	// 【未来扩展】如果配置来自文件，可以获取文件的实际修改时间
	// if filePath := p.getFilePathForKey(key); filePath != "" {
	// 	if info, err := os.Stat(filePath); err == nil {
	// 		metadata.UpdateTime = info.ModTime()
	// 		metadata.CreateTime = p.getFileCreateTime(filePath) // 需要系统调用
	// 		metadata.Version = info.ModTime().Unix() // 使用修改时间作为版本号
	// 	}
	// }

	return metadata, nil
}

// 公共监听方法
func (p *FileProvider) Watch(ctx context.Context, key string, callback provider.WatchCallback) (provider.CancelFunc, error) {
	// 检查当前配置中心是否初始化
	if err := p.ensureInitialized(); err != nil {
		return nil, provider.NewConfigError("Watch", key, err)
	}

	// 检查键是否存在
	exists, err := p.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, provider.NewConfigError("Watch", key, provider.ErrKeyNotFound)
	}

	// 注册监听器
	p.watcherMutex.Lock()
	defer p.watcherMutex.Unlock()
	if _, exists := p.watchers[key]; !exists {
		p.watchers[key] = make([]provider.WatchCallback, 0)
	}
	p.watchers[key] = append(p.watchers[key], callback)

	// 如果监听器未启动，进行启动
	p.ensureWatcherStarted(ctx)

	// 返回当前监听器取消函数
	return func() {
		p.unregisterWatcher(key, callback)
	}, nil

}

// 指定前缀变更监听器
func (p *FileProvider) WatchWithPrefix(ctx context.Context, prefix string, callback provider.WatchCallback) (provider.CancelFunc, error) {
	// 检查当前配置中心是否初始化
	if err := p.ensureInitialized(); err != nil {
		return nil, provider.NewConfigError("WatchWithPrefix", prefix, err)
	}

	// 获取全部匹配的键列表
	keys, err := p.Keys(ctx, prefix)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, provider.NewConfigError("WatchWithPrefix", prefix, provider.ErrKeyNotFound)
	}

	// 注册监听器
	p.watcherMutex.Lock()
	defer p.watcherMutex.Unlock()
	if _, exists := p.prefixWatchers[prefix]; !exists {
		p.prefixWatchers[prefix] = make([]provider.WatchCallback, 0)
	}
	p.prefixWatchers[prefix] = append(p.prefixWatchers[prefix], callback)

	// 如果监听器未启动，进行启动
	p.ensureWatcherStarted(ctx)

	// 返回当前监听器取消函数
	return func() {
		p.unregisterWatcher(prefix, callback)
	}, nil
}

// ==================================
// ============ 内部方法  ============
// ==================================

// ensureInitialized 确保提供者已初始化
func (p *FileProvider) ensureInitialized() error {
	state := providerState(p.state.Load())
	switch state {
	case stateUninitialized:
		return provider.ErrNotInitialized
	case stateClosed:
		return provider.ErrProviderClosed
	default:
		return nil
	}
}

// loadDirectory 加载目录下的所有配置文件
func (p *FileProvider) loadDirectory(ctx context.Context) error {
	return filepath.Walk(p.configPath, func(path string, info os.FileInfo, err error) error {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			if p.logger != nil {
				p.logger.Warn("loadDirectory cancelled: context done, path=%s", path)
			}
			return ctx.Err()
		default:
		}

		if err != nil {
			if p.logger != nil {
				p.logger.Error("loadDirectory failed: walk error, path=%s, error=%v", path, err)
			}
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理支持的文件类型
		ext := filepath.Ext(path)
		if !isSupportedExt(ext) {
			return nil
		}

		return p.loadFile(ctx, path)
	})
}

// loadFile 加载单个配置文件
func (p *FileProvider) loadFile(ctx context.Context, path string) error {
	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}

	// 解析文件内容
	var data interface{}
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		// 使用 json.Number 保持数字精度
		decoder := json.NewDecoder(strings.NewReader(string(content)))
		decoder.UseNumber()
		if err := decoder.Decode(&data); err != nil {
			return fmt.Errorf("parse json %s: %w", path, err)
		}
	case ".yaml", ".yml":
		// 【未来扩展】YAML 支持
		// if err := yaml.Unmarshal(content, &data); err != nil {
		// 	return fmt.Errorf("parse yaml %s: %w", path, err)
		// }
		return fmt.Errorf("yaml not supported yet, file: %s", path)
	case ".toml":
		// 【未来扩展】TOML 支持
		return fmt.Errorf("toml not supported yet, file: %s", path)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	// 计算命名空间（相对路径，不含扩展名）
	namespace := p.getNamespace(path)

	// 存储配置数据
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data[namespace] = data

	// 扁平化数据
	if m, ok := data.(map[string]interface{}); ok {
		p.flattenMap(namespace, m)
	}

	return nil
}

// getNamespace 获取配置的命名空间
// 例如: /config/redis.json -> redis
// 例如: /config/db/mysql.json -> db/mysql (如果 configPath 是 /config)
func (p *FileProvider) getNamespace(path string) string {
	// 获取相对路径
	relPath, err := filepath.Rel(p.configPath, path)
	if err != nil {
		relPath = filepath.Base(path)
	}

	// 移除扩展名
	ext := filepath.Ext(relPath)
	return strings.TrimSuffix(relPath, ext)
}

// flattenMap 将嵌套 map 扁平化
// 例如: {"redis": {"host": "localhost"}} -> {"redis.host": "localhost"}
func (p *FileProvider) flattenMap(prefix string, data map[string]interface{}) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// 递归处理嵌套 map
			p.flattenMap(fullKey, v)
			// 同时保存中间节点（允许获取整个子树）
			p.flatData[fullKey] = v
		default:
			p.flatData[fullKey] = v
		}
	}
}

// isSupportedExt 检查是否是支持的文件扩展名
func isSupportedExt(ext string) bool {
	switch ext {
	case ".json", ".yaml", ".yml", ".toml":
		return true
	default:
		return false
	}
}

// getValue 获取配置值（内部使用，不做类型转换）
func (p *FileProvider) getValue(ctx context.Context, key string) (interface{}, error) {
	if err := p.ensureInitialized(); err != nil {
		return nil, provider.NewConfigError("getValue", key, err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// 先从扁平数据中查找
	if val, found := p.flatData[key]; found {
		return val, nil
	}

	// 再从嵌套结构中查找
	val, found := provider.GetNestedValue(p.data, key)
	if !found {
		return nil, provider.NewConfigError("getValue", key, provider.ErrKeyNotFound)
	}

	return val, nil
}

// inferContentType 推断配置内容类型
// 【说明】根据键名和值类型推断，主要用于调试和展示
func (p *FileProvider) inferContentType(key string, value interface{}) string {
	// 根据值类型推断
	switch value.(type) {
	case map[string]interface{}:
		return "json" // 嵌套对象通常是 JSON
	case []interface{}:
		return "json" // 数组通常是 JSON
	case string:
		// 检查是否是 JSON 字符串
		var temp interface{}
		if err := json.Unmarshal([]byte(value.(string)), &temp); err == nil {
			return "json"
		}
		return "string"
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case float32, float64:
		return "number"
	case json.Number:
		return "number"
	default:
		return "unknown"
	}
}

// ============ 批量元数据获取 ============
// 【未来扩展】批量获取多个键的元数据，提高性能

// Exists 检查配置键是否存在
func (p *FileProvider) Exists(ctx context.Context, key string) (bool, error) {
	if err := p.ensureInitialized(); err != nil {
		return false, provider.NewConfigError("Exists", key, err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// 先从扁平数据中查找（快速路径）
	if _, found := p.flatData[key]; found {
		return true, nil
	}

	// 再从嵌套结构中查找
	_, found := provider.GetNestedValue(p.data, key)
	return found, nil
}

// 遍历 flatData 收集所有键，获取对应前缀的键列表，如果prefix为空，则返回全部键
func (p *FileProvider) Keys(ctx context.Context, prefix string) ([]string, error) {
	//检验是否初始化
	if err := p.ensureInitialized(); err != nil {
		return nil, provider.NewConfigError("Keys", prefix, err)
	}

	//外部方法，需要加锁
	p.mu.RLock()
	defer p.mu.RUnlock()

	//遍历 flatData 收集所有键
	keys := make([]string, 0, len(p.flatData))
	//特殊判断处理
	if prefix == "" {
		for key := range p.flatData {
			keys = append(keys, key)
		}
		return keys, nil
	} else {
		for key := range p.flatData {
			if strings.HasPrefix(key, prefix) {
				keys = append(keys, key)
			}
		}
	}
	return keys, nil
}

// 内部使用，获取全部键，不加锁
func (p *FileProvider) getAllKeys(ctx context.Context) ([]string, error) {
	//检验是否初始化（消耗小，防御性编程）
	if err := p.ensureInitialized(); err != nil {
		return nil, err
	}

	//遍历返回全部键
	keys := make([]string, 0, len(p.flatData))
	for key := range p.flatData {
		keys = append(keys, key)
	}
	return keys, nil
}

// 内部使用，获取对应前缀值键，不加锁
func (p *FileProvider) getKeysWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	//检验是否初始化
	if err := p.ensureInitialized(); err != nil {
		return nil, err
	}

	//遍历返回对应前缀值键
	keys := make([]string, 0, len(p.flatData))
	for key := range p.flatData {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

/****************   监听  *****************/
// 文件快照，只做标识不建议加Data存储信息
type fileSnapshot struct {
	ModTime time.Time
	Size    int64
	MD5     string
}

// 获取文件快照变更
// 通过校验时间，大小和MD5来判断是否发生变化
func (p *FileProvider) getFileSnapshot(path string) (*fileSnapshot, error) {
	//获取文件信息
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	//读取文件内容，计算MD5
	context, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	md5 := md5.Sum(context)
	md5Str := fmt.Sprintf("%x", md5)
	return &fileSnapshot{
		ModTime: info.ModTime(),
		Size:    info.Size(),
		MD5:     md5Str,
	}, nil
}

// 获取全部需要监听的文件列表
func (p *FileProvider) getAllWatchFiles() ([]string, error) {
	var files []string

	//获取对应路径下文件信息
	info, err := os.Stat(p.configPath)
	if err != nil {
		return nil, err
	}

	// 检查是否是文件
	if info.IsDir() {
		// walk遍历目，收集全部文件路径
		err = filepath.Walk(p.configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// 不是目录，只有单个文件
		files = append(files, p.configPath)
	}
	return files, nil
}

// 对比快照，检查数据变更
func (p *FileProvider) compareSnapshots(oldSnapshots map[string]*fileSnapshot, lastSnapshots map[string]*fileSnapshot) []string {
	var changedFiles []string

	//检查新增或者更新
	for path, oldSnapshot := range oldSnapshots {
		//检查当前文件有的，原有文件是否存在
		lastPath, exit := lastSnapshots[path]
		if !exit {
			changedFiles = append(changedFiles, path)

			continue
		}
		//对比时间，大小和MD5
		if oldSnapshot.ModTime != lastPath.ModTime || oldSnapshot.Size != lastPath.Size || oldSnapshot.MD5 != lastPath.MD5 {
			changedFiles = append(changedFiles, path)
		}
	}

	//检查是否有文件删除
	for filePath := range lastSnapshots {
		if _, exists := oldSnapshots[filePath]; !exists {
			changedFiles = append(changedFiles, filePath)
		}
	}
	return changedFiles
}

// 轮询监听器启动
func (p *FileProvider) startPollingWatcher(ctx context.Context) error {

	//检查监听模式
	if p.opts.WatchMode != provider.WatchModePoll {
		return nil
	}

	//获取快照文件
	lastSnapshots := make(map[string]*fileSnapshot)
	files, err := p.getAllWatchFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		snapshot, err := p.getFileSnapshot(file)
		if err == nil {
			lastSnapshots[file] = snapshot
		} else {
			logger.Error("获取快照失败", err)
			return err
		}
	}

	//轮询监听器
	p.watcherWg.Add(1)
	go func() {
		defer p.watcherWg.Done()

		//定时时间设定
		ticker := time.NewTicker(p.opts.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopWatcher:
				//停止信号
				return
			case <-ctx.Done():
				//上下文取消
				return
			case <-ticker.C:
				//定时检查
				p.pollWatchFiles(ctx, lastSnapshots)
			}
		}
	}()

	return nil
}

// 轮询变更检查
func (p *FileProvider) pollWatchFiles(ctx context.Context, lastSnapshots map[string]*fileSnapshot) {
	// 获取当前文件快照
	currentSnapshots := make(map[string]*fileSnapshot)
	//获取全部需要监听的文件列表
	files, err := p.getAllWatchFiles()
	if err != nil {
		logger.Error("获取当前文件快照失败", err)
		return
	}
	//将获取文件放入快照
	for _, file := range files {
		snapshot, err := p.getFileSnapshot(file)
		if err == nil {
			currentSnapshots[file] = snapshot
		}
	}

	changeLists := p.compareSnapshots(lastSnapshots, currentSnapshots)
	if len(changeLists) > 0 {
		// 有资源变更，重新加载资源并且触发回调
		p.handleChangeLists(ctx, changeLists)

		// 更新快照
		for filePath, snap := range currentSnapshots {
			lastSnapshots[filePath] = snap
		}

		// 移除已经删除的文件快照
		for filePath := range lastSnapshots {
			if _, exists := currentSnapshots[filePath]; !exists {
				delete(lastSnapshots, filePath)
			}
		}
	}
}

// 处理资源变更
func (p *FileProvider) handleChangeLists(ctx context.Context, changeLists []string) {
	// 重新加载变更的文件
	for _, filePath := range changeLists {
		// 检查文件是否还存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// 文件被删除，触发删除事件
			p.notifyKeyDeleted(filePath)
			continue
		}

		// 重新加载文件
		if err := p.loadFile(ctx, filePath); err != nil {
			// 加载失败，记录错误但不中断其他文件处理
			continue
		}

		// 获取该文件对应的命名空间和所有配置键
		namespace := p.getNamespace(filePath)
		// 更新事件
		p.notifyKeyChanged(namespace, filePath)
	}
}

// 配置变更事件处理
func (p *FileProvider) notifyKeyChanged(namespace string, filePath string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 获取该命名空间下的所有配置键
	var affectedKeys []string
	for key := range p.flatData {
		if strings.HasPrefix(key, namespace) {
			affectedKeys = append(affectedKeys, key)
		}
	}

	// 触发精确匹配的监听器
	for _, key := range affectedKeys {
		p.triggerWatchers(key, provider.EventTypePut)
	}

	// 触发前缀匹配的监听器
	p.triggerPrefixWatchers(affectedKeys)
}

// 配置删除事件处理
func (p *FileProvider) notifyKeyDeleted(filePath string) {
	namespace := p.getNamespace(filePath)

	p.mu.RLock()
	defer p.mu.RUnlock()

	// 这里简化处理，实际应该维护键到文件的映射关系
	var affectedKeys []string
	for key := range p.flatData {
		if strings.HasPrefix(key, namespace) {
			affectedKeys = append(affectedKeys, key)
		}
	}
	// 触发前缀匹配的监听器
	p.triggerPrefixWatchers(affectedKeys)
}

// 触发指定键监听器
func (p *FileProvider) triggerWatchers(key string, eventType provider.EventType) {
	p.watcherMutex.RLock()
	callbacks, exists := p.watchers[key]
	p.watcherMutex.RUnlock()

	if !exists || len(callbacks) == 0 {
		return
	}

	// 获取旧值和新值
	var oldValue, newValue []byte
	p.mu.RLock()
	if val, found := p.flatData[key]; found {
		if bytes, err := json.Marshal(val); err == nil {
			newValue = bytes
		}
	}
	p.mu.RUnlock()

	// 创建事件
	event := &provider.WatchEvent{
		Key:       key,
		OldValue:  oldValue, // 【简化】轮询模式下难以获取旧值，实际应该缓存
		NewValue:  newValue,
		EventType: eventType,
		Timestamp: time.Now(),
		Version:   0, // 【未来扩展】使用文件修改时间作为版本号
	}

	// 异步触发所有回调（避免阻塞）
	for _, callback := range callbacks {
		cb := callback // 捕获循环变量
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 回调 panic 不影响其他回调
					// 【未来扩展】记录 panic 日志
				}
			}()
			cb(event)
		}()
	}
}

// triggerPrefixWatchers 触发前缀匹配的监听器
func (p *FileProvider) triggerPrefixWatchers(affectedKeys []string) {
	p.watcherMutex.RLock()
	defer p.watcherMutex.RUnlock()

	// 遍历所有前缀监听器
	for prefix, callbacks := range p.prefixWatchers {
		// 检查是否有键匹配该前缀
		for _, key := range affectedKeys {
			if strings.HasPrefix(key, prefix) {
				// 触发该前缀的所有回调
				var oldValue, newValue []byte
				p.mu.RLock()
				if val, found := p.flatData[key]; found {
					if bytes, err := json.Marshal(val); err == nil {
						newValue = bytes
					}
				}
				p.mu.RUnlock()

				event := &provider.WatchEvent{
					Key:       key,
					OldValue:  oldValue,
					NewValue:  newValue,
					EventType: provider.EventTypePut,
					Timestamp: time.Now(),
					Version:   0,
				}

				for _, callback := range callbacks {
					cb := callback
					go func() {
						defer func() {
							if r := recover(); r != nil {
								// 忽略 panic
							}
						}()
						cb(event)
					}()
				}
				break // 每个前缀只触发一次（即使有多个键匹配）
			}
		}
	}
}

// 检验监听器是否启动
func (p *FileProvider) ensureWatcherStarted(ctx context.Context) {
	//检查监听器是否启动，如果已经启动，则直接返回
	select {
	case <-p.stopWatcher:
		// 已经关闭，需要重新创建
		p.stopWatcher = make(chan struct{})
	default:
		return
	}

	if err := p.startPollingWatcher(ctx); err != nil {
		logger.Error("启动轮询监听器失败", err)
		return
	}
}

// 取消注册监听器
func (p *FileProvider) unregisterWatcher(key string, callback provider.WatchCallback) {
	p.watcherMutex.Lock()
	defer p.watcherMutex.Unlock()

	// 检查回调函数是否存在
	callbacks, exit := p.watchers[key]
	if !exit {
		return
	}
	// 回调表中移除
	for i, cb := range callbacks {
		if &cb == &callback {
			p.watchers[key] = append(callbacks[:i], callbacks[i+1:]...)
			break
		}
	}

	if len(p.watchers[key]) == 0 {
		delete(p.watchers, key)
	}
}

// 取消注册前缀监听器
func (p *FileProvider) unregisterPrefixWatcher(prefix string, callback provider.WatchCallback) {
	p.watcherMutex.Lock()
	defer p.watcherMutex.Unlock()

	callbacks, exit := p.prefixWatchers[prefix]
	if !exit {
		return
	}
	for i, cb := range callbacks {
		if &cb == &callback {
			p.prefixWatchers[prefix] = append(callbacks[:i], callbacks[i+1:]...)
			break
		}
	}

	if len(p.prefixWatchers[prefix]) == 0 {
		delete(p.prefixWatchers, prefix)
	}
}

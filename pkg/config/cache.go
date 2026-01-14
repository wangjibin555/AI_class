package config

import (
	"AI_class/pkg/logger"
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrCacheNotFound    = errors.New("cache: key not found")
	ErrCacheExpired     = errors.New("cache: key expired")
	ErrCacheFull        = errors.New("cache: cache is full")
	ErrInvalidKey       = errors.New("cache: invalid key")
	ErrInvalidValue     = errors.New("cache: invalid value")
	ErrCacheClosed      = errors.New("cache: cache is closed")
	ErrInvalidTTL       = errors.New("cache: invalid TTL")
	ErrInvalidNamespace = errors.New("cache: invalid namespace")
	ErrNamespaceExists  = errors.New("cache: namespace no exists")
	ErrDeletedNamespace = errors.New("cache: delete namespace error")
)

const (
	// 默认配置
	DefaultMaxSize       = 1000            // 默认最大缓存条目数
	DefaultCleanupTicker = 5 * time.Minute // 默认清理间隔
	DefaultTTL           = 5 * time.Minute // 默认 TTL
	MinTTL               = 1 * time.Second // 最小 TTL
	MaxTTL               = 24 * time.Hour  // 最大 TTL

	// 键命名规范前缀
	KeyPrefixConfig       = "config:"         // 配置键前缀
	KeyPrefixNamespace    = "ns:"             // 命名空间前缀，进行隔离普通配置
	KeyPrefixMetadata     = "meta:"           // 元数据前缀
	KeySeparator          = ":"               // 键分隔符
	DefaultEvictionPolicy = EvictionPolicyLRU // 默认淘汰策略
)

// 缓存淘汰策略
type EvictionPolicy string

const (
	EvictionPolicyLRU    EvictionPolicy = "lru"    // 最近最少使用
	EvictionPolicyLFU    EvictionPolicy = "lfu"    // 最少使用
	EvictionPolicyRandom EvictionPolicy = "random" // 随机
	EvictionPolicyNone   EvictionPolicy = "none"   // 不淘汰
)

// 针对缓存操作接口
type Cache interface {
	BasicOptions
	//	PipelineOptions
	NamespaceOptions
	TTLOptions
	MetricsOptions
	Close() error
}

type BasicOptions interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
}

type PipelineOptions interface {
	MGet(ctx context.Context, keys ...string) (map[string][]byte, error)
	MSet(ctx context.Context, values map[string][]byte, ttl time.Duration) error
	MDelete(ctx context.Context, keys ...string) error
}

type NamespaceOptions interface {
	//获取正常存储的命名空间，如果keys为空则返回整个namespace的配置
	GetWithNamespace(ctx context.Context, namespace string, keys ...string) (map[string][]byte, error)
	//设置命名空间
	SetWithNamespace(ctx context.Context, namespace string, keys map[string][]byte, ttl time.Duration) error
	//删除命名空间内对应键
	DeleteWithNamespace(ctx context.Context, namespace string, keys ...string) error
	//获取命名空间下的键列表
	GetNamespaces(ctx context.Context, namespace string) ([]string, error)
	//获取全部命名空间
	GetAllNamespaces(ctx context.Context) ([]string, error)
	//创建命名空间
	CreateNamespace(ctx context.Context, namespace string) error
	//删除命名空间，包含其全部键
	DeleteNamespace(ctx context.Context, namespace string) error
}

type TTLOptions interface {
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	SetTTL(ctx context.Context, key string, ttl time.Duration) error
	DeleteTTL(ctx context.Context, key string) error
}

type MetricsOptions interface {
	Stats(ctx context.Context) *CacheStats
	Size(ctx context.Context) int
	Clear(ctx context.Context) error
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits          int64          // 命中次数
	Misses        int64          // 未命中次数
	Sets          int64          // 设置次数
	Deletes       int64          // 删除次数
	Evictions     int64          // 淘汰次数
	Expired       int64          // 过期次数
	Size          int            // 当前条目数
	MaxSize       int            // 最大容量
	HitRate       float64        // 命中率
	MemoryUsage   int64          // 内存使用（字节）
	LastCleanup   time.Time      // 上次清理时间
	NamespaceSize map[string]int // 各命名空间大小
}

// CacheEntry缓存条目
type CacheEntry struct {
	Key         string
	Value       []byte
	Namespace   string
	ExpireAt    time.Time     //过期时间
	CreateAt    time.Time     //创建时间
	AccessAt    time.Time     //最后访问时间（更新时间也是最近）
	Size        int64         //缓存条目大小
	AccessCount int64         //访问次数
	TTL         time.Duration //TTL时间
}

// 检查当前条目是否过期
func (e *CacheEntry) IsExpired() bool {
	if e.ExpireAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpireAt) //返回是否在对应时间后
}

// 操作时候更新最近访问时间
func (e *CacheEntry) UpdateAccessAt() {
	e.AccessAt = time.Now()
	e.AccessCount++
}

// ==============缓存选项，配置模式==============
// 缓存配置选项
type CacheOptions struct {
	MaxSize         int                            //最大缓存条目数
	EvictionPolicy  EvictionPolicy                 //缓存淘汰策略
	CleanupInterval time.Duration                  //清理间隔
	DefaultTTL      time.Duration                  //默认TTL时间
	EnableStats     bool                           // 是否启用统计
	OnEvicted       func(key string, value []byte) // 淘汰回调
}

type CacheOption func(*CacheOptions)

// 创建默认配置
func DefaultCacheOptions() *CacheOptions {
	return &CacheOptions{
		MaxSize:         DefaultMaxSize,
		EvictionPolicy:  EvictionPolicyLRU,
		CleanupInterval: DefaultCleanupTicker,
		DefaultTTL:      DefaultTTL,
		EnableStats:     true,
		OnEvicted:       nil,
	}
}

// 设置对应最大大小
func WithMaxSize(size int) CacheOption {
	if size == 0 {
		return func(opts *CacheOptions) {
			logger.Warn("WithMaxSize: Invalid size, using default")
			opts.MaxSize = DefaultMaxSize
		}
	}
	return func(opts *CacheOptions) {
		opts.MaxSize = size
	}
}

// 设置淘汰策略
func WithEvictionPolicy(policy EvictionPolicy) CacheOption {
	//前置校验
	if policy == "" {
		return func(opts *CacheOptions) {
			logger.Warn("WithEvictionPolicy: Invalid eviction policy, using default")
			opts.EvictionPolicy = DefaultEvictionPolicy
		}
	}

	switch policy {
	case EvictionPolicyLRU:
		return func(opts *CacheOptions) {
			opts.EvictionPolicy = EvictionPolicyLRU
		}
	case EvictionPolicyLFU:
		return func(opts *CacheOptions) {
			opts.EvictionPolicy = EvictionPolicyLFU
		}
	case EvictionPolicyRandom:
		return func(opts *CacheOptions) {
			opts.EvictionPolicy = EvictionPolicyRandom
		}
	case EvictionPolicyNone:
		return func(opts *CacheOptions) {
			opts.EvictionPolicy = EvictionPolicyNone
		}
	default:
		return func(opts *CacheOptions) {
			opts.EvictionPolicy = DefaultEvictionPolicy
		}
	}
}

// 设置清除间隔
func WithCleanupInterval(interval *time.Duration) CacheOption {
	if interval == nil || *interval <= 0 {
		return func(opts *CacheOptions) {
			logger.Warn("Cleanup interval is invalid, using default")
			opts.CleanupInterval = DefaultCleanupTicker
		}
	}
	return func(opts *CacheOptions) {
		opts.CleanupInterval = *interval
	}
}

// 设置TTL
func WithDefaultTTL(ttl time.Duration) CacheOption {
	if err := VaildateTTL(ttl); err != nil {
		return func(opts *CacheOptions) {
			logger.Warn("Default TTL is invalid, using default")
			opts.DefaultTTL = DefaultTTL
		}
	}
	return func(opts *CacheOptions) {
		opts.DefaultTTL = ttl
	}
}

// 设置启用或禁用统计
func WithEnableStats(enable bool) CacheOption {
	return func(opts *CacheOptions) {
		opts.EnableStats = enable
	}
}

// 设置淘汰回调
func WithOnEvicted(callback func(key string, value []byte)) CacheOption {
	return func(opts *CacheOptions) {
		opts.OnEvicted = callback
	}
}

// ===========缓存键命名规范===========
// 缓存键构建器
type CacheBuilder struct {
	parts []string
	mu    sync.RWMutex
}

func NewCacheKeyBuilder() *CacheBuilder {
	return &CacheBuilder{
		parts: make([]string, 0),
		mu:    sync.RWMutex{},
	}
}

// 追加前缀
func (b *CacheBuilder) AddPrefix(parts string) *CacheBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.parts = append(b.parts, parts)
	return b
}

// 添加命名空间
func (b *CacheBuilder) AddNamespace(namespace string) *CacheBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.parts = append(b.parts, KeyPrefixNamespace+namespace)
	return b
}

// 添加键
func (b *CacheBuilder) AddKey(key string) *CacheBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.parts = append(b.parts, key)
	return b
}

// 构建最终键
func (b *CacheBuilder) Build() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.parts) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, part := range b.parts {
		if i > 0 {
			sb.WriteString(KeySeparator)
		}
		sb.WriteString(part)
	}
	return sb.String()
}

// Reset 重置构建器
func (b *CacheBuilder) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.parts = make([]string, 0)
}

// ===============快捷方法===============
// 快捷构建配置键
func BuildConfigKey(namespace string, key string) string {
	return NewCacheKeyBuilder().AddPrefix(KeyPrefixConfig).AddNamespace(namespace).AddKey(key).Build()
}

// 快捷构建元数据键
func BuildMetadataKey(namespace, key string) string {
	return NewCacheKeyBuilder().AddPrefix(KeyPrefixMetadata).AddNamespace(namespace).AddKey(key).Build()
}

// 快捷解析对应缓存键
func ParseCacheKey(fullKey string) (prefix, namespace, key string) {
	parts := strings.Split(fullKey, KeySeparator)
	if len(parts) < 3 {
		logger.Warn("ParseCacheKey: Invalid cache key: %s", fullKey)
		return "", "", fullKey
	}

	//解析缓存键，第一个是类型前缀，第二个就是ns，第三个是命名空间，后续就是键
	prefix = parts[0]
	for i := 1; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], KeyPrefixNamespace) {
			namespace = strings.TrimPrefix(parts[i], KeyPrefixNamespace)
			if i+1 < len(parts) {
				key = strings.Join(parts[i+1:], KeySeparator)
			}
			break
		}
	}
	return prefix, namespace, key
}

// 检查键是否有效
func VaildateCacheKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	parts := strings.Split(key, KeySeparator)
	if len(parts) > 256 {
		return ErrInvalidKey
	}

	return nil
}

// 校验TTL
func VaildateTTL(ttl time.Duration) error {
	// 使用默认ttl
	if ttl == 0 {
		return nil
	}
	if ttl < MinTTL || ttl > MaxTTL {
		return ErrInvalidTTL
	}
	return nil
}

// 计算命中率
func CalculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

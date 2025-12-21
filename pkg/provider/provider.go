package provider

import (
	"context"
	"time"
)

// ============ 核心接口 ============

// ConfigProvider 配置提供者接口（扩展版）
type ConfigProvider interface {
	// 生命周期管理
	Init(ctx context.Context, opts ...Option) error
	Close(ctx context.Context) error
	HealthCheck(ctx context.Context) error

	// 配置获取 - 原始值
	GetRaw(ctx context.Context, key string) ([]byte, error)
	GetRawWithDefault(ctx context.Context, key string, defaultVal []byte) []byte

	// 配置获取 - 类型安全
	GetString(ctx context.Context, key string) (string, error)
	GetInt(ctx context.Context, key string) (int, error)
	GetBool(ctx context.Context, key string) (bool, error)
	GetFloat64(ctx context.Context, key string) (float64, error)
	GetDuration(ctx context.Context, key string) (time.Duration, error)
	GetStringSlice(ctx context.Context, key string) ([]string, error)
	GetStringMap(ctx context.Context, key string) (map[string]string, error)

	// 配置获取 - 结构体绑定
	Unmarshal(ctx context.Context, key string, target interface{}) error
	UnmarshalKey(ctx context.Context, key string, target interface{}) error

	// 配置监听
	Watch(ctx context.Context, key string, callback WatchCallback) (CancelFunc, error)
	WatchPrefix(ctx context.Context, prefix string, callback WatchCallback) (CancelFunc, error)

	// 配置元信息
	Exists(ctx context.Context, key string) (bool, error)
	Keys(ctx context.Context, prefix string) ([]string, error)
	GetMetadata(ctx context.Context, key string) (*ConfigMetadata, error)
}

// ============ 辅助类型 ============

// WatchCallback 配置变更回调
type WatchCallback func(event *WatchEvent)

// CancelFunc 取消监听函数
type CancelFunc func()

// WatchEvent 配置变更事件
type WatchEvent struct {
	Key       string
	OldValue  []byte
	NewValue  []byte
	EventType EventType
	Timestamp time.Time
	Version   int64
}

// EventType 事件类型
type EventType int

const (
	EventTypePut EventType = iota
	EventTypeDelete
	EventTypeExpire
)

// ConfigMetadata 配置元数据
type ConfigMetadata struct {
	Key         string
	Version     int64
	CreateTime  time.Time
	UpdateTime  time.Time
	ContentType string // json/yaml/toml
	Size        int64
	MD5         string
}

// Option 配置选项函数
type Option func(*ProviderOptions)

// ProviderOptions 提供者配置选项
type ProviderOptions struct {
	// 基础配置
	Namespace   string        // 命名空间
	Environment string        // 环境: dev/staging/prod
	Region      string        // 区域
	Timeout     time.Duration // 超时时间

	// 缓存配置
	EnableCache   bool
	CacheTTL      time.Duration
	LocalCacheDir string // 本地缓存目录

	// 重试配置
	MaxRetries       int
	RetryInterval    time.Duration
	RetryBackoffRate float64 // 退避倍率

	// 监听配置
	WatchMode    WatchMode
	PollInterval time.Duration // 轮询间隔（轮询模式）

	// 安全配置
	EnableEncryption bool
	KMSKeyID         string

	// 可观测性
	EnableMetrics bool
	MetricsPrefix string
}

// WatchMode 监听模式
type WatchMode int

const (
	WatchModePush   WatchMode = iota // 推送模式（配置中心主动推送）
	WatchModePoll                    // 轮询模式
	WatchModeHybrid                  // 混合模式（推送为主，轮询兜底）
)

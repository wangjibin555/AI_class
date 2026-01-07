package config

import (
	"AI_class/pkg/provider"
	"sync"
)

//客户端核心

type ConfigClient struct {
	provider  provider.ConfigProvider
	cache     Cache
	namespace map[string]interface{}
	mu        sync.RWMutex
	opts      *ClientOptions
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
//func Init(ctx context.Context, p provider.ConfigProvider, opts ...ClientOption) {
//	initOnce.Do(func() {
//		option := &ClientOptions{
//			Environment: "dev",
//			EnableCache: true,
//		}
//
//		for _, opt := range opts {
//			opt(option)
//		}
//
//		var cache Cache
//		if option.EnableCache {
//			cache = NewMemoryCache(option.CacheOptions...)
//		}
//	})
//}

// ClientOption 客户端选项函数
type ClientOption func(*ClientOptions)

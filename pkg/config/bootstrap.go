package config

import (
	"AI_class/def"
	"AI_class/pkg/graceful"
	"AI_class/pkg/logger"
	"AI_class/pkg/provider"
	"AI_class/pkg/provider/file"
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	ErrConfigPath             = errors.New("config path is required")
	ErrFileProviderInitFailed = "failed to init file provider: %w"
)

type BootstrapOptions struct {
	// 配置文件路径（文件或目录）
	ConfigPath string

	// 环境（dev/test/prod）
	Environment string

	// 是否启用缓存
	EnableCache bool

	// 缓存选项
	CacheOptions []CacheOption

	// Provider 选项
	ProviderOptions []provider.Option

	// 服务是否自动注册平滑关闭
	AutoRegisterShutdown bool

	// 平滑关闭超时时间
	ShutdownTimeout time.Duration
}

type BootstrapOption func(*BootstrapOptions)

func WithConfigpath(path string) BootstrapOption {
	return func(options *BootstrapOptions) {
		options.ConfigPath = path
	}
}

func WithEnvionment(env string) BootstrapOption {
	return func(options *BootstrapOptions) {
		options.Environment = env
	}
}

func WithEnableCache(enable bool) BootstrapOption {
	return func(options *BootstrapOptions) {
		options.EnableCache = enable
	}
}

func WithCacheOptions(options ...CacheOption) BootstrapOption {
	return func(bootstrapOptions *BootstrapOptions) {
		bootstrapOptions.CacheOptions = options
	}
}

func WithProviderOptions(options ...provider.Option) BootstrapOption {
	return func(bootstrapOptions *BootstrapOptions) {
		bootstrapOptions.ProviderOptions = options
	}
}

func WithAutoRegisterShutdown(autoRegisterShutdown bool) BootstrapOption {
	return func(bootstrapOptions *BootstrapOptions) {
		bootstrapOptions.AutoRegisterShutdown = autoRegisterShutdown
	}
}

func WithShutdownTimeout(timeout time.Duration) BootstrapOption {
	return func(bootstrapOptions *BootstrapOptions) {
		bootstrapOptions.ShutdownTimeout = timeout
	}
}

// 初始化
func Bootstrap(ctx context.Context, opts ...BootstrapOption) error {
	options := &BootstrapOptions{
		ConfigPath:           getDefaultConfigPath(),
		Environment:          getDefaultEnvironment(),
		EnableCache:          true,
		AutoRegisterShutdown: true,
		ShutdownTimeout:      def.ShutdownTimeout,
	}

	for _, opt := range opts {
		opt(options)
	}

	//校验是否被初始化
	if options.ConfigPath == "" {
		return ErrConfigPath
	}

	//创建provider并进行初始化
	fileProvider := file.NewFileProvider(options.ConfigPath)
	if err := fileProvider.Init(ctx, options.ProviderOptions...); err != nil {
		return fmt.Errorf(ErrFileProviderInitFailed, err)
	}

	//构建客户端核心
	clientOpts := []ClientOption{
		WithEnvironment(options.Environment),
	}
	if options.EnableCache {
		clientOpts = append(clientOpts, WithCache(options.CacheOptions...))
	}

	logger.Info("Config center initialized successfully")
	logger.Info("Config path: %s", options.ConfigPath)
	logger.Info("Environment: %s", options.Environment)
	logger.Info("Cache enabled: %v", options.EnableCache)

	//自动注册平滑关闭
	if options.AutoRegisterShutdown {
		graceful.RegisterAllFunc(func() {
			Shutdown(options.ShutdownTimeout)
		})
		logger.Info("Auto-registered config shutdown handler")
	}
	return nil
}

func BootstrapWithProvider(ctx context.Context, p provider.ConfigProvider, opts ...BootstrapOption) error {
	options := &BootstrapOptions{
		Environment:          getDefaultEnvironment(),
		EnableCache:          true,
		AutoRegisterShutdown: true,
		ShutdownTimeout:      30 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	// 构建配置中心客户端选项
	clientOpts := []ClientOption{
		WithEnvironment(options.Environment),
	}

	if options.EnableCache {
		clientOpts = append(clientOpts, WithCache(options.CacheOptions...))
	}

	// 初始化配置中心
	if err := Init(ctx, p, clientOpts...); err != nil {
		return fmt.Errorf("failed to init config client: %w", err)
	}

	logger.Info("Config center initialized with custom provider")
	logger.Info("Environment: %s", options.Environment)
	logger.Info("Cache enabled: %v", options.EnableCache)

	// 自动注册平滑关闭
	if options.AutoRegisterShutdown {
		graceful.RegisterAllFunc(func() {
			Shutdown(options.ShutdownTimeout)
		})
		logger.Info("Auto-registered config shutdown handler")
	}
	return nil
}

func getDefaultConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	//找常见路径
	paths := []string{
		"./config",
		"./configs",
		"../config",
		"../../config",
	}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	//返回当前目录下的config
	return "./config"
}

func getDefaultEnvironment() string {
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	return "dev"
}

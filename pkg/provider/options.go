package provider

import (
	"errors"
	"time"
)

var (
	ErrInvalidNamespace        = errors.New("namespace is required")
	ErrInvalidEnvironment      = errors.New("environment is required")
	ErrInvalidRegion           = errors.New("region is required")
	ErrInvalidTimeout          = errors.New("timeout is required")
	ErrInvalidCacheTTL         = errors.New("cache TTL is required")
	ErrInvalidMaxRetries       = errors.New("max retries is required")
	ErrInvalidRetryInterval    = errors.New("retry interval is required")
	ErrInvalidRetryBackoffRate = errors.New("retry backoff rate is required")
	ErrInvalidPollInterval     = errors.New("poll interval is required")
	ErrInvalidWatchMode        = errors.New("watch mode is required")
	ErrInvalidEnableEncryption = errors.New("enable encryption is required")
	ErrInvalidKMSKeyID         = errors.New("KMS key ID is required")
	ErrInvalidEnableMetrics    = errors.New("enable metrics is required")
	ErrInvalidMetricsPrefix    = errors.New("metrics prefix is required")
	ErrInvalidEnableCache      = errors.New("enable cache is required")
	ErrInvalidLocalCacheDir    = errors.New("local cache dir is required")
)

// 常量
const (
	DefaultTimeout          = 5 * time.Second
	DefaultCacheTTL         = 5 * time.Minute
	DefaultMaxRetries       = 3
	DefaultRetryInterval    = 1 * time.Second
	DefaultRetryBackoffRate = 2.0
	DefaultPollInterval     = 30 * time.Second
)

// 返回默认配置选项
func DefaultOptions() *ProviderOptions {
	return &ProviderOptions{
		//基础配置
		Namespace:   "default",
		Environment: "dev",
		Region:      "",
		Timeout:     DefaultTimeout,

		//缓存配置
		EnableCache:   true,
		CacheTTL:      DefaultCacheTTL,
		LocalCacheDir: "", //不需要文件缓存

		//重试配置
		MaxRetries:       DefaultMaxRetries,
		RetryInterval:    DefaultRetryInterval,
		RetryBackoffRate: DefaultRetryBackoffRate,

		//监听配置
		PollInterval: DefaultPollInterval,
		WatchMode:    WatchModePoll,

		//安全配置
		EnableEncryption: false,
		KMSKeyID:         "",

		//可观测性
		EnableMetrics: false,
		MetricsPrefix: "config_provider",
	}
}

// ApplyOptions 应用选项到默认配置，统一入口，其实就是相当于给当前配置赋值
func ApplyOptions(opts ...Option) *ProviderOptions {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options) //将选项应用到默认配置
	}
	return options
}

// 设置命名空间
func SetNamespace(namespace string) (Option, error) {
	if namespace == "" {
		return nil, ErrInvalidNamespace
	}

	return func(opts *ProviderOptions) {
		opts.Namespace = namespace
	}, nil
}

func SetEnvironment(environment string) (Option, error) {
	if environment == "" {
		return nil, ErrInvalidEnvironment
	}

	return func(opts *ProviderOptions) {
		opts.Environment = environment
	}, nil
}

func SetRegion(region string) (Option, error) {
	if region == "" {
		return nil, ErrInvalidRegion
	}

	return func(opts *ProviderOptions) {
		opts.Region = region
	}, nil
}

func SetTimeout(timeout time.Duration) (Option, error) {
	if timeout <= 0 {
		return nil, ErrInvalidTimeout
	}

	return func(opts *ProviderOptions) {
		opts.Timeout = timeout
	}, nil
}

func SetCacheTTL(cacheTTL time.Duration) (Option, error) {
	if cacheTTL <= 0 {
		return nil, ErrInvalidCacheTTL
	}

	return func(opts *ProviderOptions) {
		opts.CacheTTL = cacheTTL
	}, nil
}

func SetLocalCacheDir(localCacheDir string) (Option, error) {
	if localCacheDir == "" {
		return nil, ErrInvalidLocalCacheDir
	}

	return func(opts *ProviderOptions) {
		opts.LocalCacheDir = localCacheDir
	}, nil
}

func SetMaxRetries(maxRetries int) (Option, error) {
	if maxRetries <= 0 {
		return nil, ErrInvalidMaxRetries
	}

	return func(opts *ProviderOptions) {
		opts.MaxRetries = maxRetries
	}, nil
}

func SetRetryInterval(retryInterval time.Duration) (Option, error) {
	if retryInterval <= 0 {
		return nil, ErrInvalidRetryInterval
	}

	return func(opts *ProviderOptions) {
		opts.RetryInterval = retryInterval
	}, nil
}

func SetRetryBackoffRate(retryBackoffRate float64) (Option, error) {
	if retryBackoffRate <= 0 {
		return nil, ErrInvalidRetryBackoffRate
	}

	return func(opts *ProviderOptions) {
		opts.RetryBackoffRate = retryBackoffRate
	}, nil
}

func SetPollInterval(pollInterval time.Duration) (Option, error) {
	if pollInterval <= 0 {
		return nil, ErrInvalidPollInterval
	}

	return func(opts *ProviderOptions) {
		opts.PollInterval = pollInterval
	}, nil
}

func SetWatchMode(watchMode WatchMode) (Option, error) {
	return func(opts *ProviderOptions) {
		opts.WatchMode = watchMode
	}, nil
}

// 设置加密
func SetEnableEncryption(enableEncryption bool) (Option, error) {
	return func(opts *ProviderOptions) {
		opts.EnableEncryption = enableEncryption
	}, nil
}

// 加密key id
func SetKMSKeyID(kmsKeyID string) (Option, error) {
	return func(opts *ProviderOptions) {
		opts.KMSKeyID = kmsKeyID
	}, nil
}

// 监控，设置数据采集
func SetEnableMetrics(enableMetrics bool) (Option, error) {
	return func(opts *ProviderOptions) {
		opts.EnableMetrics = enableMetrics
	}, nil
}

// 监控，设置数据采集前缀
func SetMetricsPrefix(metricsPrefix string) (Option, error) {
	return func(opts *ProviderOptions) {
		opts.MetricsPrefix = metricsPrefix
	}, nil
}

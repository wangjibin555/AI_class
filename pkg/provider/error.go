package provider

import (
	"errors"
	"fmt"
)

var (
	// 配置不存在
	ErrKeyNotFound = errors.New("config key not found")

	// 配置类型转换失败
	ErrTypeMismatch = errors.New("config type mismatch")

	// 提供者未初始化
	ErrNotInitialized = errors.New("provider not initialized")

	// 提供者已关闭
	ErrProviderClosed = errors.New("provider already closed")

	// 配置解析失败
	ErrParseConfig = errors.New("failed to parse config")

	// 配置验证失败
	ErrValidation = errors.New("config validation failed")

	// 健康检查失败
	ErrHealthCheck = errors.New("health check failed")

	// 监听已存在
	ErrWatcherExists = errors.New("watcher already exists for this key")
)

// 未来扩展 带上下文的错误

// 包装错误类，扩展错误，携带更多数据
type ConfigError struct {
	Op  string // 操作名称（init,GetString等）
	Key string // 配置键名
	Err error  // 错误
}

// 错误
func (e *ConfigError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("config error: %s, key: %s, err: %v", e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("config error: %s, err: %v", e.Op, e.Err)
}

// 解包错误
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// 创建配置错误
func NewConfigError(op, key string, err error) *ConfigError {
	return &ConfigError{Op: op, Key: key, Err: err}
}

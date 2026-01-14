package database

import (
	"fmt"
	"time"
)

var (
	ErrMysqlHostIsEmpty     = fmt.Errorf("mysql host is empty")
	ErrMysqlPortIsError     = fmt.Errorf("mysql port is empty")
	ErrMysqlUserNameIsEmpty = fmt.Errorf("mysql username is empty")
	ErrMysqlPasswordIsEmpty = fmt.Errorf("mysql password is empty")
)

type MysqlConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	UserName        string `json:"username"`
	Password        string `json:"password"`
	DataBase        string `json:"database"`
	Charset         string `json:"chartset"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
	ConnMaxIdletime int    `json:"conn_max_idle_time"`
	ConnTimeout     int    `json:"conn_timeout"`
	ReadTimeout     int    `json:"read_timeout"`
	WriteTimeout    int    `json:"write_timeout"`
	LogLevel        string `json:"log_level"`
	SlowThreshold   int    `json:"slow_threshold"`
}

// GetHost 获取 Host
func (m *MysqlConfig) GetHost() string {
	return m.Host
}

// GetPort 获取 Port
func (m *MysqlConfig) GetPort() int {
	return m.Port
}

// GetUserName 获取 UserName
func (m *MysqlConfig) GetUserName() string {
	return m.UserName
}

// GetPassword 获取 Password
func (m *MysqlConfig) GetPassword() string {
	return m.Password
}

// GetDataBase 获取 DataBase
func (m *MysqlConfig) GetDataBase() string {
	return m.DataBase
}

// GetCharset 获取 Charset
func (m *MysqlConfig) GetCharset() string {
	if m.Charset == "" {
		return "utf8mb4"
	}
	return m.Charset
}

// GetMaxOpenConns 获取 MaxOpenConns
func (m *MysqlConfig) GetMaxOpenConns() int {
	if m.MaxOpenConns <= 0 {
		return 100
	}
	return m.MaxOpenConns
}

// GetMaxIdleConns 获取 MaxIdleConns
func (m *MysqlConfig) GetMaxIdleConns() int {
	if m.MaxIdleConns <= 0 {
		return 10
	}
	return m.MaxIdleConns
}

// GetConnMaxLifetime 获取 ConnMaxLifetime
func (m *MysqlConfig) GetConnMaxLifetime() time.Duration {
	if m.ConnMaxLifetime <= 0 {
		return time.Duration(3600) * time.Second
	}
	return time.Duration(m.ConnMaxLifetime) * time.Second
}

// GetConnMaxIdletime 获取 ConnMaxIdletime
func (m *MysqlConfig) GetConnMaxIdletime() time.Duration {
	if m.ConnMaxIdletime <= 0 {
		return time.Duration(600) * time.Second
	}
	return time.Duration(m.ConnMaxIdletime) * time.Second
}

// GetConnTimeout 获取 ConnTimeout
func (m *MysqlConfig) GetConnTimeout() time.Duration {
	if m.ConnTimeout <= 0 {
		return time.Duration(10) * time.Second
	}
	return time.Duration(m.ConnTimeout) * time.Second
}

// GetReadTimeout 获取 ReadTimeout
func (m *MysqlConfig) GetReadTimeout() time.Duration {
	if m.ReadTimeout <= 0 {
		return time.Duration(30) * time.Second
	}
	return time.Duration(m.ReadTimeout) * time.Second
}

// GetWriteTimeout 获取 WriteTimeout
func (m *MysqlConfig) GetWriteTimeout() time.Duration {
	if m.WriteTimeout <= 0 {
		return time.Duration(30) * time.Second
	}
	return time.Duration(m.WriteTimeout) * time.Second
}

// GetLogLevel 获取 LogLevel
func (m *MysqlConfig) GetLogLevel() string {
	return m.LogLevel
}

// GetSlowThreshold 获取 SlowThreshold
func (m *MysqlConfig) GetSlowThreshold() int {
	if m.SlowThreshold <= 0 {
		return 200
	}
	return m.SlowThreshold
}

// GetAddr 获取完整地址
func (m *MysqlConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

// HasPassword 判断是否配置了密码
func (m *MysqlConfig) HasPassword() bool {
	return m.Password != ""
}

func (c *MysqlConfig) Vaildate() error {
	if c.Host == "" {
		return ErrMysqlHostIsEmpty
	}

	if c.Port <= 0 || c.Port > 65535 {
		return ErrMysqlPortIsError
	}

	if c.UserName == "" {
		return ErrMysqlUserNameIsEmpty
	}
	if c.Password == "" {
		return ErrMysqlPasswordIsEmpty
	}

	return nil
}

package database

import (
	"fmt"
	"time"
)

var (
	ErrRedisHostEmpty   = fmt.Errorf("redis host is empty")
	ErrRedisPortEmpty   = fmt.Errorf("redis port is empty")
	ErrRedisDbEmpty     = fmt.Errorf("redis db is empty")
	ErrRedisClusterMode = fmt.Errorf("cluster nodes are required in cluster mode")
)

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	UserName string `json:"username"`
	Password string `json:"password"`
	Db       int    `json:"db"`

	//连接池配置
	MaxIdle     int   `json:"max_idle"`
	MaxActive   int   `json:"max_active"`
	IdleTimeout int64 `json:"idle_timeout"`
	Wait        bool  `json:"wait"`

	//超时设置
	DialTimeout  int `json:"dial_timeout"`
	ReadTimeout  int `json:"read_timeout"`
	WriteTimeout int `json:"write_timeout"`

	//TLS配置
	EnableTLS bool `json:"enable_tls"`

	//集群配置
	ClusterMode   bool     `json:"cluster_mode"`
	ClusterNodes  []string `json:"cluster_nodes"`  //集群节点列表
	ClusterPrefix string   `json:"cluster_prefix"` //集群键前缀

	//其他配置
	KeyPrefix string `json:"key_prefix"`
}

func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *RedisConfig) GetMaxIdle() int {
	if c.MaxIdle <= 0 {
		return 10
	}
	return c.MaxIdle
}

func (c *RedisConfig) GetMaxActive() int {
	if c.MaxActive <= 0 {
		return 500
	}
	return c.MaxActive
}

func (c *RedisConfig) GetIdleTimeout() time.Duration {
	if c.IdleTimeout <= 0 {
		return time.Duration(240)
	}
	return time.Duration(c.IdleTimeout)
}

func (c *RedisConfig) GetDialTimeout() time.Duration {
	if c.DialTimeout <= 0 {
		return time.Duration(5)
	}
	return time.Duration(c.DialTimeout)
}

func (c *RedisConfig) GetReadTimeout() time.Duration {
	if c.ReadTimeout <= 0 {
		return time.Duration(20)
	}
	return time.Duration(c.ReadTimeout)
}

func (c *RedisConfig) GetWriteTimeout() time.Duration {
	if c.WriteTimeout <= 0 {
		return time.Duration(20)
	}
	return time.Duration(c.WriteTimeout)
}

func (c *RedisConfig) IsClusterMode() bool {
	return c.ClusterMode
}

func (c *RedisConfig) HashPassword() bool {
	return c.Password != ""
}

func (c *RedisConfig) Vaildate() error {
	if c.Host == "" {
		return ErrRedisHostEmpty
	}

	if c.Port <= 0 || c.Port > 65535 {
		return ErrRedisPortEmpty
	}

	if c.Db < 0 || c.Db > 15 {
		return ErrRedisDbEmpty
	}

	if c.ClusterMode && len(c.ClusterNodes) == 0 {
		return ErrRedisClusterMode
	}

	return nil
}

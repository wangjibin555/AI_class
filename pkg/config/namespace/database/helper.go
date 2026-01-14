package database

import (
	mysqlClient "AI_class/infrastructure/mysql"
	redisInfra "AI_class/infrastructure/redis"
	"context"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

// 桥接方法,依据配置中心配置进行创建对应Redis连接
func (c *RedisConfig) NewRedisClient() (*redisInfra.RedisClient, error) {
	// 验证配置
	if err := c.Vaildate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	pool := &redis.Pool{
		MaxIdle:     c.GetMaxIdle(),
		MaxActive:   c.GetMaxActive(),
		IdleTimeout: c.GetIdleTimeout(),
		Wait:        c.Wait,

		Dial: func() (redis.Conn, error) {
			opts := []redis.DialOption{
				redis.DialConnectTimeout(c.GetDialTimeout()),
				redis.DialReadTimeout(c.GetReadTimeout()),
				redis.DialWriteTimeout(c.GetWriteTimeout()),
				redis.DialDatabase(c.Db),
			}

			if c.HashPassword() {
				opts = append(opts, redis.DialPassword(c.Password))
			}

			if c.EnableTLS {
				opts = append(opts, redis.DialUseTLS(true))
				// TODO: 添加 TLS 配置
			}
			opts = append(opts, redis.DialKeepAlive(c.GetReadTimeout()+time.Second))

			conn, err := redis.Dial("tcp", c.GetAddr(), opts...)
			if err != nil {
				return nil, fmt.Errorf("failed to dial redis: %w", err)
			}

			return conn, nil
		},

		// 健康检查
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if time.Since(t) < c.GetReadTimeout()+time.Second {
				return nil
			}
			_, err := conn.Do("PING")
			return err
		},
	}
	client := redisInfra.NewRedisClientWithContext(
		context.Background(),
		pool,
	)
	return client, nil
}

func (mq *MysqlConfig) NewMysqlClient() (*mysqlClient.MysqlClient, error) {
	if err := mq.Vaildate(); err != nil {
		return nil, fmt.Errorf("invalid mysql config: %w", err)
	}
	mysqlConfig := &mysqlClient.MysqlConfig{
		Host:            mq.GetHost(),
		Port:            mq.GetPort(),
		UserName:        mq.GetUserName(),
		Password:        mq.GetPassword(),
		Database:        mq.GetDataBase(),
		Charset:         mq.GetCharset(),
		MaxIdleconns:    mq.GetMaxIdleConns(),
		MaxOpenConns:    mq.GetMaxOpenConns(),
		ConnMaxLifetime: mq.GetConnMaxLifetime(),
		ConnMaxIdletime: mq.GetConnMaxIdletime(),
		ConnTimeout:     mq.GetConnTimeout(),
		ReadTimeout:     mq.GetReadTimeout(),
		WriteTimeout:    mq.GetWriteTimeout(),
		LogLevel:        mq.GetLogLevel(),
		SlowThreshold:   time.Duration(mq.GetSlowThreshold()),
	}

	mysqlClient, err := mysqlClient.NewMysqlClient(mysqlConfig)
	if err != nil {
		return nil, err
	}

	return mysqlClient, nil
}

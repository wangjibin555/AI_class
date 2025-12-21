package redis

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
)

// 连接结构体
type RedisClient struct {
	ctx  context.Context
	pool *redis.Pool // 非集群模式下redis客户端
}

// 定义常量
const (
	//redis连接
	dialTimeout        = time.Second * 5  //直连超时时间
	clientReadTimeout  = time.Second * 20 //客户端读取操作时间超时
	clientWirteTimeout = time.Second * 20 //客户端写入操作时间超时
	//设置连接超时时间比客户端操作时间大一秒，避免客户端操作失败
	turnValidConnTimeout = clientReadTimeout + time.Second //连接健康检查域值

	//连接池
	redisMaxActiveNum = 500               // 最大活跃连接数量
	redisMaxIdleNum   = 10                //最大空闲连接数
	IdleTimeout       = 240 * time.Second //空闲连接超时时间
)

// 管道类
type RedisPipeline struct {
	client   *RedisClient
	conn     redis.Conn // redis连接
	count    int
	err      error
	commands []*SendCommand
	replys   []*PipeApplyPair //Pipeline的操作结果与错误信息
	Done     bool
}

// 获取连接池
func (c *RedisClient) GetRedisClient() *redis.Pool {
	return c.pool
}

// 关闭连接池
func (c *RedisClient) ClosePool() error {
	return c.pool.Close()
}

// 获取连接池状态
func (c *RedisClient) RedisPoolStatus() redis.PoolStats {
	return c.pool.Stats()
}

func (c *RedisClient) NewRedisClientWithName(name, addr, password string) {
	client := newRedisClient(name, addr, password)
	*c = *client
}

func newRedisClient(name, addr, password string) *RedisClient {
	pool := &redis.Pool{
		MaxIdle:     redisMaxIdleNum,
		MaxActive:   redisMaxActiveNum,
		IdleTimeout: IdleTimeout,

		//连接池耗尽是否等待（true表示等待）
		Wait: true,

		//连接建立
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr,
				redis.DialPassword(password),
				redis.DialConnectTimeout(dialTimeout),
				redis.DialReadTimeout(clientReadTimeout),
				redis.DialWriteTimeout(clientWirteTimeout),
				redis.DialDatabase(0),
				redis.DialTLSConfig(nil),
				redis.DialKeepAlive(turnValidConnTimeout))
			if err != nil {
				return nil, err
			}
			return c, nil
		},

		// 从连接池中获取连接的健康检查
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			// 如果连接最近使用过，跳过PING
			if time.Since(t) < turnValidConnTimeout {
				return nil
			}
			// PING检查连接是否健康
			_, err := c.Do("PING")
			return err
		},
	}

	return &RedisClient{
		ctx:  context.Background(),
		pool: pool,
	}
}

func (c *RedisClient) Get(key string) (string, error) {
	//获取连接
	conn := c.pool.Get()
	//确保连接会自动放回连接池
	defer conn.Close()
	//执行Redis操作
	return redis.String(conn.Do("GET", key))
}

func (c *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	//错误处理
	if err := conn.Err(); err != nil {
		return err
	}
	//处理设置国旗时间的场景
	if expiration > 0 {
		_, err := conn.Do("SET", key, value, "EX", int(expiration.Seconds()))
		return err
	}
	//这个是处理不设置过期时间
	_, err := conn.Do("SET", key, value)
	return err
}

func (c *RedisClient) Del(key string) error {
	conn := c.pool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		return err
	}
	_, err := conn.Do("DEL", key)
	return err
}

func (c *RedisClient) Ping() error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	return err
}

// Pipeline相关操作
func (c *RedisClient) NewPipeline() *RedisPipeline {
	client := c.pool.Get()
	return &RedisPipeline{
		client:   c,
		conn:     client,
		count:    0,
		err:      nil,
		commands: make([]*SendCommand, 0),
		replys:   make([]*PipeApplyPair, 0),
		Done:     false,
	}
}

// Send 添加命令到Pipeline（不立即执行）
func (p *RedisPipeline) Send(commandName string, args ...interface{}) *RedisPipeline {
	// 如果Pipeline已经执行完毕或已有错误，不再添加命令
	if p.Done || p.err != nil {
		return p
	}

	// 记录命令信息（用于调试和跟踪）
	cmd := &SendCommand{
		CommandName: commandName,
		Args:        args,
		Idx:         p.count,
	}
	p.commands = append(p.commands, cmd)
	p.count++

	// 将命令发送到连接的缓冲区
	err := p.conn.Send(commandName, args...)
	if err != nil {
		p.err = err
	}

	return p // 支持链式调用
}

// Exec 执行Pipeline中的所有命令并获取结果
func (p *RedisPipeline) Exec() ([]*PipeApplyPair, error) {
	// 如果已经执行过，直接返回之前的结果
	if p.Done {
		return p.replys, p.err
	}

	// 确保连接在执行完后会被释放
	defer func() {
		p.Done = true
		p.conn.Close()
	}()

	// 如果在Send过程中就出错了，直接返回错误
	if p.err != nil {
		return nil, p.err
	}

	// 没有命令，直接返回
	if p.count == 0 {
		return p.replys, nil
	}

	// Flush：将缓冲区中的所有命令发送到Redis服务器
	err := p.conn.Flush()
	if err != nil {
		p.err = err
		return nil, err
	}

	// Receive：接收所有命令的响应
	for i := 0; i < p.count; i++ {
		reply, err := p.conn.Receive()
		pair := &PipeApplyPair{
			val: reply,
			err: err,
		}
		p.replys = append(p.replys, pair)
	}

	return p.replys, nil
}

// Close 关闭Pipeline连接（如果还没执行Exec，会放弃所有命令）
func (p *RedisPipeline) Close() {
	if !p.Done {
		p.Done = true
		p.conn.Close()
	}
}

// GetResults 获取Pipeline执行结果
func (p *RedisPipeline) GetResults() []*PipeApplyPair {
	return p.replys
}

// GetError 获取Pipeline的错误
func (p *RedisPipeline) GetError() error {
	return p.err
}

// Count 获取Pipeline中命令的数量
func (p *RedisPipeline) Count() int {
	return p.count
}

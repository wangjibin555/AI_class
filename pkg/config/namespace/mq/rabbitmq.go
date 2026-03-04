package mq

import (
	"AI_class/pkg/logx"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

var (
	ErrorRabbitMQClientCreate  = fmt.Errorf("rabbitmq client create error")
	ErrorRabbitMQChannelCreate = fmt.Errorf("rabbitm1 client channel craete error")
)

type RabbitMQConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	VirtualHost    string
	MaxConnections int
	MaxChannels    int //虚拟连接，相当于工作携程，进行执行具体发布、消费、声明队列
	ConnectTimeout time.Duration
	HeartBeat      time.Duration
}

type RabbitMQClient struct {
	config   *RabbitMQConfig
	conn     *amqp091.Connection
	channel  []*amqp091.Channel
	ctx      context.Context
	mu       sync.RWMutex
	isClosed bool //是否关闭
}

func NewRabbitMQClient(config *RabbitMQConfig) (*RabbitMQClient, error) {
	if config.VirtualHost != "" {
		config.VirtualHost = `/`
	}
	if config.MaxConnections <= 0 {
		config.MaxConnections = 10
	}
	if config.MaxChannels <= 0 {
		config.MaxChannels = 10
	}
	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = 10 * time.Second
	}
	if config.HeartBeat <= 0 {
		config.HeartBeat = 10 * time.Second
	}

	url := fmt.Sprintf("amqp://%s:%s@%s:%d%s", config.User, config.Password, config.Host, config.Port, config.VirtualHost)

	conn, err := amqp091.DialConfig(url, amqp091.Config{
		Heartbeat: config.HeartBeat,
		Locale:    "en_US",
	})
	if err != nil {
		return nil, ErrorRabbitMQClientCreate
	}

	client := &RabbitMQClient{
		config:   config,
		conn:     conn,
		channel:  make([]*amqp091.Channel, 0),
		ctx:      context.Background(),
		isClosed: false,
	}

	//监听连接关闭
	go client.handleConnectionClose()

	return client, nil
}

func (c *RabbitMQClient) NewChannel() (*amqp091.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return nil, fmt.Errorf("connection is closed")
	}

	channel, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	c.channel = append(c.channel, channel)
	return channel, nil
}

func (c *RabbitMQClient) GetChannel() (*amqp091.Channel, error) {
	c.mu.RLock()

	if len(c.channel) > 0 {
		ch := c.channel[0]
		c.mu.RUnlock()
		return ch, nil
	}
	c.mu.RUnlock()
	return c.NewChannel()
}

// 主动关闭，实际关闭操作执行
func (c *RabbitMQClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return nil
	}

	c.isClosed = true
	var errs []error
	for _, ch := range c.channel {
		if err := ch.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := c.conn.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close connection: %v", errs)
	}

	return nil
}

// 负责监听并且释放当前Conection内部关注的Channel列表
func (c *RabbitMQClient) handleConnectionClose() {
	closeChan := c.conn.NotifyClose(make(chan *amqp091.Error, 1))

	err := <-closeChan

	c.mu.Lock()
	if c.isClosed {
		c.mu.Unlock()
		return
	}

	c.isClosed = true
	c.channel = nil
	c.mu.Unlock()

	if err != nil {
		logx.Error("[ERROR] RabbitMQ connection closed unexpectedly: %v (code: %d, reason: %s)\n", err, err.Code, err.Reason)
	} else {
		logx.Info("[INFO] RabbitMQ connection closed")
	}
}

package server

import (
	"fmt"
	"net/http"
	"time"
)

// 服务器构建器实现
type serverBuilder struct {
	router      Router
	middlewares []Middleware
	config      *ServerConfig
}

// 创建服务器构建器
func NewServerBuilder() ServerBuilder {
	return &serverBuilder{
		middlewares: []Middleware{},
	}
}

func (b *serverBuilder) WithRouter(router Router) ServerBuilder {
	b.router = router
	return b
}

func (b *serverBuilder) WithMiddleware(middleware Middleware) ServerBuilder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

func (b *serverBuilder) WithConfig(config *ServerConfig) ServerBuilder {
	b.config = config
	return b
}

func (b *serverBuilder) Build() (*http.Server, error) {
	if b.router == nil {
		return nil, fmt.Errorf("router is required")
	}
	handler := b.router.Handler()

	//全局中间件
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		handler = b.middlewares[i].Handle(handler)
	}

	config := b.getConfig()

	server := &http.Server{
		Addr:              config.Addr,
		Handler:           handler,
		ReadTimeout:       time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(config.WriteTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(config.ReadHeaderTimeout) * time.Second,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}

	return server, nil
}

func (b *serverBuilder) getConfig() *ServerConfig {
	if b.config != nil {
		return b.config
	}

	// 返回默认配置
	return &ServerConfig{
		Addr:              ":8080",
		ReadTimeout:       30,
		WriteTimeout:      30,
		ReadHeaderTimeout: 45,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}

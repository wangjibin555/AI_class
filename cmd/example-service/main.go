package main

import (
	"AI_class/def"
	"AI_class/pkg/config"
	"AI_class/pkg/config/namespace/database"
	"AI_class/pkg/config/namespace/server"
	"AI_class/pkg/graceful"
	httpServer "AI_class/pkg/http/server"
	"AI_class/pkg/http/server/middleware"
	routes "AI_class/pkg/http/server/route"
	"AI_class/pkg/logger"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudflare/tableflip"
)

func main() {
	upg, err := tableflip.New(tableflip.Options{})
	if err != nil {
		panic(err)
	}
	defer upg.Stop()

	ctx := context.Background()

	// 1. 初始化配置中心
	err = config.Bootstrap(ctx,
		config.WithConfigpath("./config"),
		config.WithEnvionment("dev"),
		config.WithEnableCache(true),
		config.WithCacheOptions(
			config.WithMaxSize(2000),
			config.WithEvictionPolicy(config.EvictionPolicyLRU),
		),
		config.WithAutoRegisterShutdown(true),
		config.WithShutdownTimeout(def.ShutdownTimeout),
	)
	if err != nil {
		logger.Error("Bootstrap failed: %v", err)
	}

	// 2. 初始化基础设施
	if err := initInfrastructure(ctx); err != nil {
		logger.Error("Init infrastructure failed: %v", err)
	}

	// 3. 构建 HTTP 服务器
	httpServer, err := buildHTTPServer(ctx)
	if err != nil {
		logger.Error("Failed to build HTTP server: %v", err)
	}

	// 4. 启动服务器
	ln, err := upg.Listen("tcp", getListenAddress(ctx))
	if err != nil {
		logger.Error("Listen failed: %v", err)
	}

	go func() {
		logger.Info("HTTP server starting on %s", ln.Addr().String())
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	// 5. 注册关闭函数
	graceful.RegisterAllFunc(func() {
		logger.Info("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP server shutdown error: %v", err)
		}
	})

	// 6. 通知 tableflip 准备就绪
	if err = upg.Ready(); err != nil {
		panic(err)
	}
	logger.Info("Service ready, PID: %d", os.Getpid())

	// 7. 等待退出信号
	//<-upg.Exit()
	waitForShutdown(upg)

	// 8. 执行平滑关闭
	logger.Info("Shutting down service...")
	graceful.ClearAllFunc(30 * time.Second)

	logger.Info("Service shutdown completed")
}

// buildHTTPServer 构建 HTTP 服务器（使用新架构）
func buildHTTPServer(ctx context.Context) (*http.Server, error) {
	// 1. 创建路由器
	router := httpServer.NewRouter()

	// 2. 创建中间件管理器
	middlewareMgr := middleware.NewManager(&configProvider{})

	// 3. 创建路由注册器
	routeRegistry := routes.NewRegistry()

	// 注册各个路由模块
	routeRegistry.Register(routes.NewHealthRoutes())
	routeRegistry.Register(routes.NewAPIRoutes(nil)) // 可以注入依赖

	// 注册所有路由
	if err := routeRegistry.RegisterAll(router); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	// 4. 获取服务器配置
	serverConfig := getServerConfig(ctx)

	// 5. 构建服务器
	server, err := httpServer.NewServerBuilder().
		WithRouter(router).
		WithMiddleware(middlewareMgr.Recovery()).       // 最外层：恢复
		WithMiddleware(middlewareMgr.RequestLogging()). // 中间层：日志
		WithMiddleware(middlewareMgr.CORS()).           // 内层：CORS
		WithConfig(serverConfig).
		Build()

	if err != nil {
		return nil, err
	}

	return server, nil
}

// configProvider 配置提供者实现
type configProvider struct{}

func (p *configProvider) Get(key string) (interface{}, error) {
	return config.Get(key)
}

// getServerConfig 获取服务器配置
func getServerConfig(ctx context.Context) *httpServer.ServerConfig {
	cfg, err := config.Get("server")
	if err != nil {
		return &httpServer.ServerConfig{
			Addr:              ":8080",
			ReadTimeout:       30,
			WriteTimeout:      30,
			ReadHeaderTimeout: 45,
			MaxHeaderBytes:    1 << 20,
		}
	}

	serverConfig := cfg.(*server.ServerConfig)
	httpConfig := serverConfig.HTTP

	return &httpServer.ServerConfig{
		Addr:              fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port),
		ReadTimeout:       httpConfig.ReadTimeout,
		WriteTimeout:      httpConfig.WriteTimeout,
		ReadHeaderTimeout: 45,
		MaxHeaderBytes:    httpConfig.MaxHeaderBytes,
	}
}

func getListenAddress(ctx context.Context) string {
	cfg, err := config.Get("server")
	if err == nil {
		serverConfig := cfg.(*server.ServerConfig)
		httpConfig := serverConfig.HTTP
		if httpConfig.Host != "" && httpConfig.Port > 0 {
			return fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port)
		}
	}
	return ":8080"
}

func initInfrastructure(ctx context.Context) error {
	cfg, err := config.Get("database")
	if err != nil {
		return fmt.Errorf("failed to get database config: %w", err)
	}

	dbConfig := cfg.(*database.DataBaseConfig)

	//初始化Redis
	redisClient, err := dbConfig.Redis.NewRedisClient()
	if err != nil {
		return fmt.Errorf("failed to init redis: %w", err)
	}
	logger.Info("Redis client initialized")

	//传入关闭函数，让系统接管关闭函数调用实现优雅关闭
	graceful.RegisterAllFunc(func() {
		if err = redisClient.ClosePool(); err != nil {
			logger.Error("failed to close redis pool: %w", err)
		}
	})
	return nil
}

func waitForShutdown(upg *tableflip.Upgrader) {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)

	// 注册要监听的信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建退出通道
	exitChan := make(chan struct{})

	// 启动 goroutine 监听 tableflip 退出信号
	go func() {
		<-upg.Exit()
		logger.Info("Received tableflip exit signal")
		close(exitChan)
	}()

	// 启动 goroutine 监听系统信号
	go func() {
		sig := <-sigChan
		logger.Info("Received system signal: %v", sig)
		close(exitChan)
	}()

	// 等待任一退出信号
	<-exitChan
	logger.Info("Exit signal received, starting graceful shutdown...")
}

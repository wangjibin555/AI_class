package main

import (
	"AI_class/def"
	"AI_class/pkg/config"
	"AI_class/pkg/config/namespace/database"
	"AI_class/pkg/config/namespace/server"
	"AI_class/pkg/graceful"
	"AI_class/pkg/logger"
	"context"
	"fmt"
	"net/http"
	"os"
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

	//初始化基础设施
	if err := initInfrastructure(ctx); err != nil {
		logger.Error("Init infrastructure failed: %v", err)
	}

	//启动服务
	ln, err := upg.Listen("tcp", "8080")
	if err != nil {
		logger.Error("Listen failed: %v", err)
	}
	defer ln.Close()

	//在goroutine中启动HTTP服务器
	httpServer, err := createHTTPServer(ctx)
	if err != nil {
		logger.Error("Failed to create HTTP server: %v", err)
	}
	go func() {
		logger.Info("HTTP server starting on %s", ln.Addr().String())
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	//通知tableflip准备就绪
	if err = upg.Ready(); err != nil {
		panic(err)
	}
	logger.Info("Service ready, PID: %d", os.Getpid())

	//等待退出信号
	<-upg.Exit()

	//执行平滑关闭
	logger.Info("Shutting down service...")
	graceful.ClearAllFunc(30 * time.Second)

	logger.Info("Service shutdown completed")
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

// createHTTPServer 创建 HTTP 服务器
func createHTTPServer(ctx context.Context) (*http.Server, error) {
	// 1. 创建路由处理器
	r := createRouter(ctx)

	// 2. 从配置中心获取服务器配置
	cfg, err := config.Get("server")
	if err != nil {
		logger.Warn("Failed to get server config, using defaults: %v", err)
		return createDefaultHTTPServer(r), nil
	}

	serverConfig := cfg.(*server.ServerConfig)
	httpConfig := serverConfig.HTTP

	// 3. 创建 HTTP 服务器（使用配置中心的配置）
	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port),
		Handler:           r,
		ReadTimeout:       time.Duration(httpConfig.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(httpConfig.WriteTimeout) * time.Second,
		ReadHeaderTimeout: 45 * time.Second, // 固定值，防止慢客户端攻击
		MaxHeaderBytes:    httpConfig.MaxHeaderBytes,
	}

	logger.Info("HTTP server configured: %s:%d", httpConfig.Host, httpConfig.Port)
	return httpServer, nil
}

// createDefaultHTTPServer 创建默认 HTTP 服务器（配置中心不可用时使用）
func createDefaultHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 45 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}

// createRouter 创建路由处理器
func createRouter(ctx context.Context) http.Handler {
	mux := http.NewServeMux()

	// ============ 健康检查接口 ============
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"example-service"}`))
	})

	// ============ 就绪检查接口（用于 Kubernetes）============
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// 可以在这里检查依赖服务（Redis、MySQL等）是否就绪
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// ============ 存活检查接口（用于 Kubernetes）============
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})

	// ============ API 路由 ============
	apiHandler := createAPIHandler(ctx)
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiHandler))

	// ============ 添加中间件 ============
	// 1. 添加 CORS 支持（如果配置启用）
	handler := addCORSIfEnabled(mux, ctx)

	// 2. 添加请求日志中间件
	handler = addRequestLogging(handler)

	// 3. 添加 HTTP/3 Alt-Svc 头（如果需要）
	handler = addAltSvcHeader(handler)

	return handler
}

// createAPIHandler 创建 API 处理器
func createAPIHandler(ctx context.Context) http.Handler {
	mux := http.NewServeMux()

	// 示例 API 路由
	mux.HandleFunc("/example", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"example endpoint","method":"` + r.Method + `"}`))
	})

	// 配置信息接口（可选，生产环境建议移除或加权限）
	mux.HandleFunc("/config/info", func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.Get("server")
		if err != nil {
			http.Error(w, `{"error":"config not available"}`, http.StatusInternalServerError)
			return
		}

		serverConfig := cfg.(*server.ServerConfig)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"name":"%s","env":"%s"}`,
			serverConfig.Name, serverConfig.Env)))
	})

	return mux
}

// addCORSIfEnabled 如果配置启用 CORS，则添加 CORS 中间件
func addCORSIfEnabled(handler http.Handler, ctx context.Context) http.Handler {
	cfg, err := config.Get("server")
	if err != nil {
		return handler
	}

	serverConfig := cfg.(*server.ServerConfig)
	if !serverConfig.HTTP.EnableCORS {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// addRequestLogging 添加请求日志中间件
func addRequestLogging(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建响应写入器，用于记录状态码
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		handler.ServeHTTP(rw, r)

		duration := time.Since(start)
		logger.Info("HTTP %s %s %d %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

// responseWriter 包装 http.ResponseWriter，用于记录状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// addAltSvcHeader 添加 HTTP/3 Alt-Svc 头
func addAltSvcHeader(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 添加 HTTP/3 支持头（如果需要）
		w.Header().Set("Alt-Svc", "h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000")
		handler.ServeHTTP(w, r)
	})
}

// getListenAddress 获取监听地址
func getListenAddress(ctx context.Context) string {
	cfg, err := config.Get("server")
	if err == nil {
		serverConfig := cfg.(*server.ServerConfig)
		httpConfig := serverConfig.HTTP
		if httpConfig.Host != "" && httpConfig.Port > 0 {
			return fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port)
		}
	}

	// 从环境变量获取
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}

	// 默认地址
	return ":8080"
}

package middleware

import (
	"AI_class/pkg/http/server"
	"AI_class/pkg/logger"
	"net/http"
	"time"
)

type Manager struct {
	configProvider ConfigProvider
}

type ConfigProvider interface {
	Get(key string) (interface{}, error)
}

func NewManager(configProvider ConfigProvider) *Manager {
	return &Manager{
		configProvider: configProvider,
	}
}

// CORS中间件创建
func (m *Manager) CORS() server.Middleware {
	return server.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}).WithName("cors")
}

func (m *Manager) RequestLogging() server.Middleware {
	return server.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			logger.Info("request", "method", r.Method, "path", r.URL.Path, "status", rw.statusCode, "duration", duration)
		})
	}).WithName("request-logging")
}

// Recovery 创建恢复中间件（防止 panic）
func (m *Manager) Recovery() server.Middleware {
	return server.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					// 记录错误日志
				}
			}()
			next.ServeHTTP(w, r)
		})
	}).WithName("recovery")
}

// Timeout 创建超时中间件
func (m *Manager) Timeout(timeout time.Duration) server.Middleware {
	return server.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout").(http.Handler)
	}).WithName("timeout")
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type namedMiddleware struct {
	server.Middleware
	name string
}

func (m *namedMiddleware) Name() string {
	return m.name
}

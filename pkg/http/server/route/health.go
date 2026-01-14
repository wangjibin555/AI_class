package route

import (
	"AI_class/pkg/http/server"
	"net/http"
)

// HealthRoutes 健康检查路由
type HealthRoutes struct{}

func NewHealthRoutes() *HealthRoutes {
	return &HealthRoutes{}
}

func (r *HealthRoutes) RegisterRoutes(router server.Router) error {
	router.Register("GET", "/health", r.Health)
	router.Register("GET", "/ready", r.Ready)
	router.Register("GET", "/live", r.Live)
	return nil
}

func (r *HealthRoutes) Health(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (r *HealthRoutes) Ready(w http.ResponseWriter, req *http.Request) {
	// 可以检查依赖服务
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

func (r *HealthRoutes) Live(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive"}`))
}

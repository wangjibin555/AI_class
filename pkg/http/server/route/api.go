package route

import (
	"AI_class/pkg/http/server"
	"net/http"
)

// APIRoutes API 路由
type APIRoutes struct {
	// 可以注入依赖
	deps interface{}
}

func NewAPIRoutes(deps interface{}) *APIRoutes {
	return &APIRoutes{
		deps: deps,
	}
}

func (r *APIRoutes) RegisterRoutes(router server.Router) error {
	// 创建 API 路由组
	apiGroup := router.Group("/api/v1")

	apiGroup.Register("GET", "/example", r.Example)
	apiGroup.Register("GET", "/users", r.GetUsers)
	apiGroup.Register("POST", "/user", r.CreateUser)

	return nil
}

func (r *APIRoutes) Example(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"example"}`))
}

func (r *APIRoutes) GetUsers(w http.ResponseWriter, req *http.Request) {
	// 业务逻辑
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"users":[]}`))
}

func (r *APIRoutes) CreateUser(w http.ResponseWriter, req *http.Request) {
	// 业务逻辑
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"id":1}`))
}

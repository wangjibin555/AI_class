package route

import "AI_class/pkg/http/server"

// Registry 路由注册器管理器
type Registry struct {
	registrars []server.RouteRegister
}

func NewRegistry() *Registry {
	return &Registry{
		registrars: []server.RouteRegister{},
	}
}

// Register 注册路由注册器
func (r *Registry) Register(registrar server.RouteRegister) {
	r.registrars = append(r.registrars, registrar)
}

// RegisterAll 注册所有路由
func (r *Registry) RegisterAll(router server.Router) error {
	for _, registrar := range r.registrars {
		if err := registrar.RegisterRoutes(router); err != nil {
			return err
		}
	}
	return nil
}

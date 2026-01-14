package server

import "net/http"

// 路由接口
type Router interface {
	//路由注册
	Register(method, path string, handler http.HandlerFunc, middlewares ...Middleware)

	//路由组创建
	Group(prefix string, middlewares ...Middleware) RouterGroup

	//返回HTTP Handler
	Handler() http.Handler
}

// 路由组接口
type RouterGroup interface {
	Register(method, path string, handler http.HandlerFunc, middlewares ...Middleware)
	Group(prefix string, middlewares ...Middleware) RouterGroup
}

// 中间件接口
type Middleware interface {
	//请求处理
	Handle(next http.Handler) http.Handler

	//返回中间件名称
	Name() string
}

// 中间价函数类型
type MiddlewareFunc func(next http.Handler) http.Handler

// 中间件处理
func (f MiddlewareFunc) Handle(next http.Handler) http.Handler {
	return f(next)
}

func (f MiddlewareFunc) Name() string {
	return "anonymous"
}

// WithName 为中间件函数设置名称
func (f MiddlewareFunc) WithName(name string) Middleware {
	return &namedMiddleware{
		Middleware: f,
		name:       name,
	}
}

// namedMiddleware 带名称的中间件包装器
type namedMiddleware struct {
	Middleware
	name string
}

// Name 返回中间件名称
func (n *namedMiddleware) Name() string {
	return n.name
}

// 服务器构建器接口
type ServerBuilder interface {
	//设置路由器
	WithRouter(router Router) ServerBuilder

	//添加全局中间件
	WithMiddleware(middlewares Middleware) ServerBuilder

	//设置配置
	WithConfig(config *ServerConfig) ServerBuilder

	//构建服务器
	Build() (*http.Server, error)
}

// 服务器配置
type ServerConfig struct {
	Addr              string
	ReadTimeout       int
	WriteTimeout      int
	ReadHeaderTimeout int
	MaxHeaderBytes    int
}

type RouteRegister interface {
	RegisterRoutes(router Router) error
}

type DependencyContainer interface {
	//获取依赖
	Get(key string) (interface{}, error)

	//设置依赖
	Set(key string, value interface{})
}

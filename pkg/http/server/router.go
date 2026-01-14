package server

import (
	"net/http"
)

// 基于 http.ServeMux 的路由实现
type muxRouter struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

// 创建新路由器
func NewRouter() Router {
	return &muxRouter{
		mux:         http.NewServeMux(),
		middlewares: []Middleware{},
	}
}

// 实现 Router.Register 方法
func (r *muxRouter) Register(method, path string, handler http.HandlerFunc, middlewares ...Middleware) {
	// 包装处理器（应用中间件）
	finalHandler := r.wrapHandler(handler, middlewares...)

	// 如果方法不是 GET，需要检查方法
	if method != "GET" {
		finalHandler = r.methodCheck(method, finalHandler)
	}

	// 注册到 ServeMux
	r.mux.HandleFunc(path, finalHandler)
}

// 实现 Router.Group 方法
func (r *muxRouter) Group(prefix string, middlewares ...Middleware) RouterGroup {
	return &routerGroup{
		router:      r,
		prefix:      prefix,
		middlewares: append(r.middlewares, middlewares...),
	}
}

// 实现 Router.Handler 方法
func (r *muxRouter) Handler() http.Handler {
	return r.mux
}

// 包装处理器，应用中间件
func (r *muxRouter) wrapHandler(handler http.HandlerFunc, routeMiddlewares ...Middleware) http.HandlerFunc {
	h := http.HandlerFunc(handler)

	// 1. 先应用路由组中间件（从外到内）
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i].Handle(h).(http.HandlerFunc)
	}

	// 2. 再应用路由级中间件
	for i := len(routeMiddlewares) - 1; i >= 0; i-- {
		h = routeMiddlewares[i].Handle(h).(http.HandlerFunc)
	}

	return h
}

// 检查 HTTP 方法
func (r *muxRouter) methodCheck(method string, handler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handler.ServeHTTP(w, req)
	})
}

// 路由组实现
type routerGroup struct {
	router      *muxRouter
	prefix      string
	middlewares []Middleware
}

// 实现 RouterGroup.Register 方法
func (g *routerGroup) Register(method, path string, handler http.HandlerFunc, middlewares ...Middleware) {
	// 构建完整路径
	fullPath := g.prefix + path

	// 合并路由组中间件和路由级中间件
	allMiddlewares := append(g.middlewares, middlewares...)

	// 调用底层路由器的 Register 方法
	g.router.Register(method, fullPath, handler, allMiddlewares...)
}

// 实现 RouterGroup.Group 方法
func (g *routerGroup) Group(prefix string, middlewares ...Middleware) RouterGroup {
	return &routerGroup{
		router:      g.router,                              // 指向同一个底层路由器
		prefix:      g.prefix + prefix,                     // 拼接前缀
		middlewares: append(g.middlewares, middlewares...), // 合并中间件
	}
}

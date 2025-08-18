package routes

import (
	"ai-classroom/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupWebViewRoutes 设置WebView代理路由
func SetupWebViewRoutes(router *gin.RouterGroup, handler *handlers.WebViewProxyHandler) {
	webviewGroup := router.Group("/webview")
	{
		// WebView代理HTML文件服务
		// 这个路由专门用于小程序web-view，绕过域名限制
		webviewGroup.GET("/html/*filepath", handler.ProxyHTMLFile)
		webviewGroup.HEAD("/html/*filepath", handler.ProxyHTMLFile)

		// WebView信息和调试
		webviewGroup.GET("/info", handler.GetWebViewInfo)
	}
}

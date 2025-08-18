package routes

import (
	"ai-classroom/internal/handlers"
	"ai-classroom/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupTMPHTMLRoutes 设置.tmp文件HTML转换路由
func SetupTMPHTMLRoutes(router *gin.RouterGroup, handler *handlers.TMPHTMLHandler) {
	tmpGroup := router.Group("/tmp")
	{
		// .tmp文件转换相关
		tmpGroup.POST("/convert-to-html", handler.ConvertTMPToHTML)
		tmpGroup.GET("/html/:taskId", handler.GetTMPHTMLPresentation)
		tmpGroup.GET("/html/:taskId/preview", handler.PreviewTMPHTMLPresentation)

		// 🆕 静态HTML文件服务 - 使用专门的处理器和中间件
		// 方案1：使用专门的处理器（主要方案）
		tmpGroup.GET("/html/preview/*filepath", handler.ServeHTMLFile)
		tmpGroup.HEAD("/html/preview/*filepath", handler.ServeHTMLFile)

		// 方案2：使用静态文件中间件（备用方案）
		// 可以通过配置启用，提供更好的静态文件服务
		staticOpts := middleware.StaticFileOptions{
			Root:       "output/html",
			EnableLogs: true,
			Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		}
		tmpGroup.Use(middleware.StaticFileMiddleware(staticOpts))

		// 文件管理
		tmpGroup.GET("/files", handler.ListTMPFiles)
		tmpGroup.GET("/files/:filename/info", handler.GetTMPFileInfo)
		tmpGroup.DELETE("/files/:filename", handler.DeleteTMPFile)

		// 转换状态和监控
		tmpGroup.GET("/conversion/:taskId/status", handler.GetConversionStatus)
		tmpGroup.GET("/conversion/:taskId/progress", handler.GetConversionProgress)

		// 资源文件
		tmpGroup.GET("/resources/:taskId/:filename", handler.ServeTMPResource)

		// 集成到现有Coze服务
		tmpGroup.POST("/process/:taskId", handler.ProcessTMPFile)
	}
}

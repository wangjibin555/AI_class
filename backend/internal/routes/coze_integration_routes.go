package routes

import (
	"ai-classroom/internal/handlers"
	"ai-classroom/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCozeIntegrationRoutes 设置Coze集成相关路由
func SetupCozeIntegrationRoutes(router *gin.RouterGroup, integrationHandler *handlers.CozeIntegrationHandler) {
	// Coze集成路由组
	cozeIntegrationGroup := router.Group("/coze-integration")
	cozeIntegrationGroup.Use(middleware.AuthRequired()) // 需要认证
	{
		// 直接传入HTML内容进行集成
		cozeIntegrationGroup.POST("/html-content", integrationHandler.IntegrateHTMLContent)

		// 从文件路径集成HTML（如果需要）
		cozeIntegrationGroup.POST("/html-file", integrationHandler.IntegrateHTMLFile)

		// 从上传文件集成HTML
		cozeIntegrationGroup.POST("/html-upload", integrationHandler.IntegrateHTMLFromUpload)
	}
}

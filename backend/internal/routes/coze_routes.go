package routes

import (
	"ai-classroom/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupCozeRoutes 设置Coze路由
func SetupCozeRoutes(router *gin.Engine, cozeHandler *handlers.CozeHandler) {
	// Coze API路由组
	cozeGroup := router.Group("/api/v1/coze")
	{
		// 健康检查
		cozeGroup.GET("/health", cozeHandler.HealthCheck)

		// PPT生成相关
		cozeGroup.POST("/generate-ppt", cozeHandler.GeneratePPT)

		// 🆕 工作流PPT生成
		cozeGroup.POST("/workflow/generate-ppt", cozeHandler.GeneratePPTWithWorkflow)

		// 任务管理
		cozeGroup.GET("/tasks/:task_id/status", cozeHandler.GetTaskStatus)
		cozeGroup.GET("/tasks/:task_id/result", cozeHandler.GetTaskResult)
		cozeGroup.DELETE("/tasks/:task_id", cozeHandler.CleanupTask)
		cozeGroup.GET("/tasks", cozeHandler.GetAllTasks) // 调试用接口

		// 🆕 HTML预览相关
		cozeGroup.GET("/tasks/:task_id/html-preview", cozeHandler.GetTaskHTMLPreview)

		// Coze链接处理
		cozeGroup.POST("/process-link", cozeHandler.ProcessCozeLink)

		// 工具接口
		cozeGroup.GET("/validate-url", cozeHandler.ValidateURL)
		cozeGroup.GET("/bot-config", cozeHandler.GetBotConfig)

		// 文件下载
		cozeGroup.GET("/download/:filename", cozeHandler.DownloadFile)
	}

	// 通用AI引擎接口
	aiGroup := router.Group("/api/v1")
	{
		// 智能体管理
		aiGroup.GET("/ai-engines", cozeHandler.GetAIEngines)
		aiGroup.GET("/ai-engines/:engine_id/config", cozeHandler.GetEngineConfig)
	}
}

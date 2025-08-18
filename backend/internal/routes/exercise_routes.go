package routes

import (
	"ai-classroom/internal/handlers"
	"ai-classroom/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupExerciseRoutes 设置练习生成相关路由
func SetupExerciseRoutes(router *gin.RouterGroup, exerciseHandler *handlers.ExerciseHandler) {
	// 练习生成路由组，需要认证
	exerciseGroup := router.Group("/exercises")
	exerciseGroup.Use(middleware.AuthRequired()) // 需要登录
	{
		// 检查课程是否已有练习
		exerciseGroup.GET("/check/:courseId", exerciseHandler.CheckCourseQuiz)

		// 生成练习
		exerciseGroup.POST("/generate", exerciseHandler.GenerateExercise)

		// 获取生成记录列表
		exerciseGroup.GET("/generation", exerciseHandler.GetUserGenerationRecords)

		// 获取指定生成记录详情
		exerciseGroup.GET("/generation/:id", exerciseHandler.GetGenerationRecord)

		// 获取生成状态
		exerciseGroup.GET("/generation/:id/status", exerciseHandler.GetGenerationStatus)
	}
}

package routes

import (
	"ai-classroom/internal/handlers"
	"ai-classroom/internal/middleware"
	"ai-classroom/internal/repositories"
	"ai-classroom/pkg/storage"
	"ai-classroom/pkg/tts"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// RegisterAudioRoutes 注册音频相关路由
func RegisterAudioRoutes(r *gin.Engine, db *gorm.DB, ttsClient *tts.AliyunTTSClient) {
	// 创建存储服务
	// 从配置中获取文件服务器地址
	fileBaseURL := viper.GetString("server.file_base_url")
	if fileBaseURL == "" {
		fileBaseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	storageService := storage.NewLocalStorageService(
		"./storage", // 本地存储路径
		fileBaseURL, // 访问URL
	)

	// 创建repositories
	courseRepo := repositories.NewCourseRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// 创建服务
	audioHandler := handlers.NewAudioHandler(
		ttsClient,
		storageService,
		db,
		courseRepo,
		userRepo,
	)

	// 创建学习处理器
	learningHandler := handlers.NewLearningHandler(db, courseRepo)

	// API路由组
	api := r.Group("/api/v1")
	{
		// 音频生成相关路由
		audio := api.Group("/audio")
		{
			audio.POST("/generate/:courseId", middleware.AuthRequired(), audioHandler.GenerateCourseAudio)
			audio.GET("/status/:courseId", middleware.AuthRequired(), audioHandler.GetAudioStatus)
			audio.POST("/preview", middleware.AuthRequired(), audioHandler.PreviewVoice)
			// 测试用接口，不需要认证
			audio.POST("/test-preview", audioHandler.PreviewVoice)
		}

		// 学习记录相关路由
		learning := api.Group("/learning")
		{
			learning.POST("/start", middleware.AuthRequired(), learningHandler.StartLearning)
			learning.POST("/progress", middleware.AuthRequired(), learningHandler.UpdateProgress)
			learning.GET("/record/:recordId", middleware.AuthRequired(), learningHandler.GetLearningRecord)
			learning.GET("/course/:courseId", middleware.AuthRequired(), learningHandler.GetCourseLearningRecord)
			learning.GET("/records", middleware.AuthRequired(), learningHandler.GetLearningRecords)
			learning.POST("/end", middleware.AuthRequired(), learningHandler.EndLearning)
			learning.DELETE("/record/:recordId", middleware.AuthRequired(), learningHandler.DeleteLearningRecord)
			learning.GET("/stats", middleware.AuthRequired(), learningHandler.GetLearningStats)
		}
	}
}

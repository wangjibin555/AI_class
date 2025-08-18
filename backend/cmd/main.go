package main

import (
	"ai-classroom/internal/coze"
	"ai-classroom/internal/database"
	"ai-classroom/internal/handlers"
	"ai-classroom/internal/middleware"
	"ai-classroom/internal/repositories"
	"ai-classroom/internal/routes"
	"ai-classroom/internal/services"
	"ai-classroom/internal/websocket"
	"ai-classroom/pkg/ai"
	mysqldb "ai-classroom/pkg/database"
	"ai-classroom/pkg/redis"
	"ai-classroom/pkg/storage"
	"ai-classroom/pkg/tts"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func init() {
	// 初始化配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("⚠️ 读取配置文件失败: %v", err)
		log.Println("使用默认配置")
		setDefaultConfig()
	} else {
		log.Printf("✅ 成功读取配置文件: %s", viper.ConfigFileUsed())
		log.Printf("🔍 配置文件中coze配置存在: %v", viper.IsSet("coze"))
	}
}

// setDefaultConfig 设置默认配置
func setDefaultConfig() {
	// 数据库配置
	viper.SetDefault("database.host", "summer-camp-dev.rwlb.rds.aliyuncs.com")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.username", "wangjibin")
	viper.SetDefault("database.password", "nuGM8Vb9QBRhtH#zmM")
	viper.SetDefault("database.database", "wangjibin")
	viper.SetDefault("database.charset", "utf8mb4")

	// Redis配置
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// 应用配置
	viper.SetDefault("app.mode", "development")
	viper.SetDefault("app.name", "AI课堂")
	viper.SetDefault("app.version", "1.0.0")

	// 服务器配置
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8088")       // 开发环境HTTP端口
	viper.SetDefault("server.https_port", "9000") // 生产环境HTTPS端口
	viper.SetDefault("server.external_host", "wangjibin-sc.wepie.com")
	viper.SetDefault("server.external_port", "9000") // 生产环境端口
	viper.SetDefault("server.ws_host", "wangjibin-sc.wepie.com")
	viper.SetDefault("server.ws_port", "9000") // 生产环境WebSocket端口
	viper.SetDefault("server.file_base_url", "https://wangjibin-sc.wepie.com:9000")
	viper.SetDefault("server.cert_file", "certs/cert.pem")
	viper.SetDefault("server.key_file", "certs/key.pem")
}

func main() {
	// 初始化数据库 - 使用viper配置
	db, err := mysqldb.InitDB()
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 自动迁移数据库表
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化Redis
	rdb, err := redis.InitRedis()
	if err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}

	// 设置Gin模式
	if viper.GetString("app.mode") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// 静态文件服务 - 提供PPT HTML文件
	r.Static("/ppt", "./ppt")

	// 静态文件服务 - 提供音频文件
	r.Static("/audio", "../storage/audio")

	// 初始化WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 初始化AI客户端
	var aiClient *ai.DashScopeClient

	// 首先尝试从环境变量读取API密钥
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		// 如果环境变量没有，尝试从配置文件读取
		apiKey = viper.GetString("aliyun.dashscope.api_key")
	}

	// 生产环境：确保API配置完整，否则退出
	if apiKey == "" || apiKey == "mock_key" || len(strings.TrimSpace(apiKey)) <= 10 {
		log.Printf("❌ 错误：阿里云DashScope API Key配置不完整或无效")
		log.Printf("🔍 当前API Key: %s", apiKey)
		log.Printf("💡 请检查配置文件 configs/config.yaml 中的 aliyun.dashscope.api_key")
		log.Printf("📋 或设置环境变量 DASHSCOPE_API_KEY")
		log.Fatal("生产环境必须配置有效的API Key，程序退出")
	}

	// 使用真实API配置
	aiClient = ai.NewDashScopeClient(
		apiKey,
		viper.GetString("aliyun.dashscope.base_url"),
		viper.GetString("aliyun.dashscope.model"),
		viper.GetInt("aliyun.dashscope.max_tokens"),
		viper.GetFloat64("aliyun.dashscope.temperature"),
		time.Duration(viper.GetInt("aliyun.dashscope.timeout"))*time.Second,
	)
	log.Printf("✅ AI客户端初始化成功，使用真实API模型: %s", viper.GetString("aliyun.dashscope.model"))
	log.Printf("📝 API Key状态: %s...%s", apiKey[:8], apiKey[len(apiKey)-4:])

	// 初始化数据访问层
	courseRepo := repositories.NewCourseRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// 验证TTS配置
	ttsAccessKeyID := viper.GetString("aliyun.access_key_id")
	ttsAccessKeySecret := viper.GetString("aliyun.access_key_secret")
	ttsAppKey := viper.GetString("aliyun.tts.app_key")

	if ttsAccessKeyID == "" || ttsAccessKeySecret == "" || ttsAppKey == "" {
		log.Printf("❌ 错误：阿里云TTS配置不完整")
		log.Printf("🔍 AccessKeyID: %s", ttsAccessKeyID)
		log.Printf("🔍 AccessKeySecret: %s", ttsAccessKeySecret)
		log.Printf("🔍 AppKey: %s", ttsAppKey)
		log.Fatal("生产环境必须配置完整的TTS参数，程序退出")
	}

	// 初始化TTS客户端（增强版）
	enhancedTTSClient := tts.NewEnhancedAliyunTTSClient(
		ttsAccessKeyID,
		ttsAccessKeySecret,
		ttsAppKey,
		time.Duration(viper.GetInt("aliyun.tts.timeout"))*time.Second,
	)

	// 配置增强版TTS参数
	enhancedTTSClient.SetMaxRetries(3)
	enhancedTTSClient.SetRetryDelay(2 * time.Second)
	enhancedTTSClient.SetLogLevel(true) // 启用详细日志

	log.Printf("✅ TTS客户端初始化成功，AppKey: %s", ttsAppKey)

	// 为了兼容性，保留原始TTS客户端接口
	ttsClient := enhancedTTSClient

	// 初始化存储服务（本地存储）
	fileBaseURL := viper.GetString("server.file_base_url")
	if fileBaseURL == "" {
		fileBaseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	storageService := storage.NewLocalStorageService("../storage", fileBaseURL)

	// 初始化服务层
	ttsService := services.NewTTSService(ttsClient, storageService, db)
	courseService := services.NewCourseService(courseRepo, aiClient, ttsService)
	userService := services.NewUserService(userRepo)

	// 初始化PPT生成服务
	// ✅ 创建关键词提取服务
	keywordService := services.NewKeywordExtractionService(aiClient)
	pptGenerationService := services.NewPPTGenerationService(aiClient, keywordService)

	// 初始化增强PPT服务
	enhancedPPTService := services.NewEnhancedPPTService(aiClient, db, "./ppt", ttsService)

	// 初始化AI内容总结服务
	contentSummaryService := services.NewContentSummaryService(aiClient, db)

	// 🆕 初始化内容服务（用于AI分析）
	contentService := services.NewContentService(db, aiClient)

	// 🆕 初始化AI内容分析服务
	aiContentAnalysisService := services.NewAIContentAnalysisService(aiClient, db, contentService)

	// 🆕 初始化Coze课程集成服务
	cozeIntegrationService := services.NewCozeCourseIntegrationService(db, aiContentAnalysisService)
	log.Printf("✅ Coze课程集成服务初始化完成")

	// 🆕 初始化Coze服务
	cozeConfig := coze.DefaultCozeConfig()

	// 从配置文件读取Coze配置
	if viper.IsSet("coze") {
		cozeConfig.APIBase = viper.GetString("coze.api_base")
		cozeConfig.Token = viper.GetString("coze.token")
		cozeConfig.APIKey = viper.GetString("coze.api_key")

		// 工作流配置
		if viper.IsSet("coze.workflow") {
			cozeConfig.Workflow.WorkflowID = viper.GetString("coze.workflow.workflow_id")
			cozeConfig.Workflow.PollInterval = viper.GetDuration("coze.workflow.poll_interval")
			cozeConfig.Workflow.MaxPollTime = viper.GetDuration("coze.workflow.max_poll_time")
		}

		// Bot配置
		if viper.IsSet("coze.bot") {
			cozeConfig.Bot.ID = viper.GetString("coze.bot.id")
			cozeConfig.Bot.CozeBotID = viper.GetString("coze.bot.coze_bot_id")
		}

		log.Printf("✅ 从配置文件加载Coze配置: APIBase=%s, Token长度=%d", cozeConfig.APIBase, len(cozeConfig.Token))
	}

	cozeAPIKey := os.Getenv("COZE_API_KEY")
	if cozeAPIKey == "" {
		cozeAPIKey = cozeConfig.APIKey // 使用配置文件中的APIKey
	}
	cozeService := services.NewCozeService(cozeConfig, cozeAPIKey)
	log.Printf("✅ Coze服务初始化完成")

	// 🆕 初始化练习生成工作流客户端和服务
	exerciseWorkflowClient := coze.NewExerciseWorkflowClient(cozeConfig)
	exerciseGenerationService := services.NewExerciseGenerationService(db, exerciseWorkflowClient)
	log.Printf("✅ 练习生成服务初始化完成")

	// 验证微信配置
	wechatAppID := viper.GetString("wechat.app_id")
	wechatAppSecret := viper.GetString("wechat.app_secret")
	appMode := viper.GetString("app.mode")

	if wechatAppID == "" || wechatAppSecret == "" {
		if appMode == "production" {
			log.Fatalf("❌ 生产环境必须配置微信小程序参数")
		} else {
			log.Printf("⚠️  微信配置不完整，将在认证处理器中使用Mock客户端进行测试")
			log.Printf("🔍 微信配置状态 - AppID: %s, AppSecret: %s, Mode: %s", wechatAppID, wechatAppSecret, appMode)
			log.Printf("💡 如需使用真实微信登录，请在配置文件中设置 wechat.app_id 和 wechat.app_secret")
		}
	} else {
		log.Printf("✅ 微信配置验证通过，AppID: %s, Mode: %s", wechatAppID, appMode)
	}

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(db, rdb)
	userHandler := handlers.NewUserHandler(db, rdb, userService)
	courseHandler := handlers.NewCourseHandler(db, rdb, hub, courseService)
	quizHandler := handlers.NewQuizHandler(db, rdb, aiClient)
	contentHandler := handlers.NewContentHandler(db, aiClient, contentSummaryService, pptGenerationService, enhancedPPTService)
	pptGenerationHandler := handlers.NewPPTGenerationHandler(pptGenerationService, db)

	// 🆕 初始化AI内容处理器
	aiContentHandler := handlers.NewAIContentHandler(aiContentAnalysisService, db)
	log.Printf("✅ AI内容处理器初始化完成: %+v", aiContentHandler != nil)

	// 🆕 初始化Coze处理器
	cozeHandler := handlers.NewCozeHandler(cozeService)
	log.Printf("✅ Coze处理器初始化完成")

	// 🆕 初始化练习生成处理器
	exerciseHandler := handlers.NewExerciseHandler(exerciseGenerationService)
	log.Printf("✅ 练习生成处理器初始化完成")

	// 🆕 初始化Coze集成处理器
	cozeIntegrationHandler := handlers.NewCozeIntegrationHandler(cozeIntegrationService)
	log.Printf("✅ Coze集成处理器初始化完成")

	// 🆕 初始化.tmp文件HTML转换处理器
	convertConfig := &coze.ConvertConfig{
		CDNBaseURL:   "https://your-cdn-domain.com", // 可以从配置文件读取
		TemplateType: "reveal",
		OutputFormat: "html",
		EnableCache:  true,
		CacheTimeout: 3600,
	}
	tmpConverter := coze.NewTMPToHTMLConverter(convertConfig, cozeConfig)
	fileStorage := coze.NewFileStorage(cozeConfig)
	tmpHTMLHandler := handlers.NewTMPHTMLHandler(tmpConverter, fileStorage)
	log.Printf("✅ .tmp文件HTML转换处理器初始化完成")

	// 创建音频处理器 - 使用增强版TTS客户端
	audioTTSClient := enhancedTTSClient

	// 创建增强TTS服务
	audioHandler := handlers.NewAudioHandler(audioTTSClient, storageService, db, courseRepo, userRepo)
	learningHandler := handlers.NewLearningHandler(db, courseRepo)

	// 创建配置处理器
	configHandler := handlers.NewConfigHandler()

	// 注册路由
	log.Printf("🔍 开始注册路由，AI内容处理器: %+v", aiContentHandler != nil)
	setupRoutes(r, authHandler, userHandler, courseHandler, quizHandler, contentHandler, pptGenerationHandler, audioHandler, learningHandler, aiContentHandler, cozeHandler, exerciseHandler, cozeIntegrationHandler, tmpHTMLHandler, configHandler, hub, db, courseRepo, storageService, audioTTSClient)

	// 创建HTTP服务器
	httpServer := &http.Server{
		Addr:         ":" + viper.GetString("server.port"),
		Handler:      r,
		ReadTimeout:  300 * time.Second, // 5分钟读取超时
		WriteTimeout: 300 * time.Second, // 5分钟写入超时
		IdleTimeout:  120 * time.Second, // 2分钟空闲超时
	}

	// 创建HTTPS服务器（用于开发环境）
	httpsPort := viper.GetString("server.https_port")
	if httpsPort == "" {
		httpsPort = "3001"
	}
	httpsServer := &http.Server{
		Addr:         ":" + httpsPort,
		Handler:      r,
		ReadTimeout:  300 * time.Second, // 5分钟读取超时
		WriteTimeout: 300 * time.Second, // 5分钟写入超时
		IdleTimeout:  120 * time.Second, // 2分钟空闲超时
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 启动HTTP服务器
	go func() {
		log.Printf("HTTP服务器启动在端口 %s", viper.GetString("server.port"))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 启动HTTPS服务器（用于开发环境）
	go func() {
		certFile := viper.GetString("server.cert_file")
		keyFile := viper.GetString("server.key_file")
		if certFile == "" {
			certFile = "certs/cert.pem"
		}
		if keyFile == "" {
			keyFile = "certs/key.pem"
		}
		log.Printf("HTTPS服务器启动在端口 %s", httpsPort)
		if err := httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTPS服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-quit
	log.Println("正在关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP服务器关闭失败: %v", err)
	}

	// 关闭HTTPS服务器
	if err := httpsServer.Shutdown(ctx); err != nil {
		log.Printf("HTTPS服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}

// setupRoutes 设置路由
func setupRoutes(r *gin.Engine,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	courseHandler *handlers.CourseHandler,
	quizHandler *handlers.QuizHandler,
	contentHandler *handlers.ContentHandler,
	pptGenerationHandler *handlers.PPTGenerationHandler,
	audioHandler *handlers.AudioHandler,
	learningHandler *handlers.LearningHandler,
	aiContentHandler *handlers.AIContentHandler,
	cozeHandler *handlers.CozeHandler,
	exerciseHandler *handlers.ExerciseHandler,
	cozeIntegrationHandler *handlers.CozeIntegrationHandler,
	tmpHTMLHandler *handlers.TMPHTMLHandler,
	configHandler *handlers.ConfigHandler,
	hub *websocket.Hub,
	db *gorm.DB,
	courseRepo repositories.CourseRepository,
	audioStorageService *storage.LocalStorageService,
	audioTTSClient tts.TTSClient) {

	log.Println("setupRoutes called")
	// API v1 路由组
	api := r.Group("/api/v1")

	// 认证相关路由
	auth := api.Group("/auth")
	{
		// 公开接口
		auth.POST("/wechat", authHandler.WechatLogin)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.GET("/validate", authHandler.ValidateToken)

		// 需要认证的接口
		auth.POST("/logout", middleware.AuthRequired(), authHandler.Logout)
		auth.GET("/profile", middleware.AuthRequired(), authHandler.GetProfile)
		auth.PUT("/profile", middleware.AuthRequired(), authHandler.UpdateProfile)
		auth.GET("/vip", middleware.AuthRequired(), authHandler.GetVIPStatus)
		auth.GET("/credits", middleware.AuthRequired(), authHandler.GetCredits)
		auth.POST("/credits/consume", middleware.AuthRequired(), authHandler.ConsumeCredits)

		// 管理员接口
		auth.GET("/users/:id", middleware.AuthRequired(), authHandler.GetUserByID)
	}

	// 用户相关路由
	users := api.Group("/users", middleware.AuthRequired())
	{
		users.GET("/profile", userHandler.GetProfile)
		users.PUT("/profile", userHandler.UpdateProfile)
		users.GET("/stats", userHandler.GetStats)
		users.GET("/usage", userHandler.GetUsage)
		users.POST("/credits/consume", userHandler.ConsumeCredits)
		users.POST("/credits/add", userHandler.AddCredits)
		users.GET("/:id", userHandler.GetUserInfo)
		users.PUT("/:id/vip", userHandler.UpdateUserVIP)
	}

	// 内容处理相关路由
	content := api.Group("/content")
	{
		// 公开接口
		content.GET("/platforms", contentHandler.GetSupportedPlatforms)
		content.GET("/types", contentHandler.GetContentTypes)
		content.POST("/validate-url", contentHandler.ValidateURL)
		content.POST("/test-tts", func(c *gin.Context) {
			// 简单的TTS测试端点
			var request struct {
				Text      string `json:"text"`
				VoiceType string `json:"voice_type"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "请求参数错误",
					"error":   err.Error(),
				})
				return
			}

			// 测试TTS客户端
			if audioTTSClient != nil {
				// 模拟测试响应
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "TTS服务配置正常",
					"data": gin.H{
						"text":       request.Text,
						"voice_type": request.VoiceType,
						"status":     "test_success",
					},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "TTS服务未初始化",
				})
			}
		})

		// 需要认证的接口
		content.POST("/extract", middleware.AuthRequired(), contentHandler.ExtractWebContent)
		content.POST("/text", middleware.AuthRequired(), contentHandler.ProcessTextContent)
		content.POST("/upload", middleware.AuthRequired(), contentHandler.UploadDocument)
		content.POST("/batch", middleware.AuthRequired(), contentHandler.BatchProcessContent)
		content.GET("/preview/:id", middleware.AuthRequired(), contentHandler.PreviewContent)
		content.GET("/stats", middleware.AuthRequired(), contentHandler.GetProcessStats)

		// 课程生成接口
		content.POST("/generate", middleware.AuthRequired(), contentHandler.GenerateCourse)

		// AI总结和增强PPT生成接口
		content.POST("/ai-summary", middleware.AuthRequired(), contentHandler.GenerateAISummary)
		content.POST("/enhanced-ppt", middleware.AuthRequired(), contentHandler.GenerateEnhancedPPT)

		// 新增增强PPT生成接口（带页数限制）
		content.POST("/enhanced-ppt-limits", middleware.AuthRequired(), contentHandler.GenerateEnhancedPPTWithLimits)
		content.GET("/slide-limits", middleware.AuthRequired(), contentHandler.GetSlideCountLimits)
		content.POST("/process-file", middleware.AuthRequired(), contentHandler.ProcessFileContent)
		content.GET("/supported-types", contentHandler.GetSupportedFileTypes)

		// 音频生成进度查询接口
		content.GET("/audio-progress/:courseId", middleware.AuthRequired(), contentHandler.GetAudioProgress)
	}

	// PPT生成相关路由
	ppt := api.Group("/ppt")
	{
		// 公开接口
		ppt.GET("/templates", pptGenerationHandler.GetTemplates)
		ppt.POST("/validate-url", pptGenerationHandler.ValidateURL)

		// 需要认证的接口
		ppt.POST("/generate", middleware.AuthRequired(), pptGenerationHandler.GeneratePPT)
		ppt.GET("/status/:id", middleware.AuthRequired(), pptGenerationHandler.GetGenerationStatus)
		ppt.POST("/upload", middleware.AuthRequired(), pptGenerationHandler.UploadDocument)

		// PPT文件下载和预览接口
		ppt.GET("/download/:filename", pptGenerationHandler.DownloadPPT)
		ppt.GET("/preview/:filename", pptGenerationHandler.PreviewPPT)
		ppt.GET("/info/:filename", pptGenerationHandler.GetPPTInfo)
	}

	// AI助手相关路由
	aiAssistant := api.Group("/ai-assistant")
	{
		// 公开接口
		aiAssistant.GET("/popular-questions", contentHandler.GetPopularQuestions)
		aiAssistant.GET("/suggested-topics", contentHandler.GetSuggestedTopics)

		// 需要认证的接口
		aiAssistant.POST("/chat", middleware.AuthRequired(), contentHandler.AIAssistantChat)
		aiAssistant.GET("/history", middleware.AuthRequired(), contentHandler.GetAIAssistantHistory)
		aiAssistant.DELETE("/history", middleware.AuthRequired(), contentHandler.ClearAIAssistantHistory)
		aiAssistant.GET("/stats", middleware.AuthRequired(), contentHandler.GetAIAssistantStats)
	}

	// 🔍 直接注册AI路由测试
	log.Printf("🔍 测试直接注册AI路由")
	api.GET("/ai-content-direct-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "直接AI路由测试成功",
			"method":  "direct_registration",
		})
	})
	log.Printf("✅ 直接AI路由注册完成")

	// 🆕 AI内容分析相关路由
	log.Printf("🔍 注册AI内容路由，处理器状态: %+v", aiContentHandler != nil)
	aiContent := api.Group("/ai-content")
	{
		// 测试路由（不需要认证）
		aiContent.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "AI内容测试路由正常工作",
				"status":  "success",
			})
		})
		log.Printf("✅ AI内容测试路由注册完成")

		// 需要认证的接口
		aiContent.POST("/url-analysis", middleware.AuthRequired(), aiContentHandler.AIURLAnalysis)
		aiContent.POST("/generate-ppt", middleware.AuthRequired(), aiContentHandler.AIGeneratePPT)
		aiContent.GET("/analysis-history", middleware.AuthRequired(), aiContentHandler.GetAnalysisHistory)
		aiContent.DELETE("/analysis/:id", middleware.AuthRequired(), aiContentHandler.DeleteAnalysis)

		// 引擎状态接口（预留扩展）
		aiContent.GET("/engine-status", aiContentHandler.GetEngineStatus)
		log.Printf("✅ AI引擎状态路由注册完成")
	}

	// 音频生成相关路由
	audio := api.Group("/audio")
	{
		audio.POST("/generate/:courseId", middleware.AuthRequired(), audioHandler.GenerateCourseAudio)
		audio.GET("/status/:courseId", middleware.AuthRequired(), audioHandler.GetAudioStatus)
		audio.POST("/preview", middleware.AuthRequired(), audioHandler.PreviewVoice)
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

	// 🆕 Coze智能体相关路由
	routes.SetupCozeRoutes(r, cozeHandler)

	// 🆕 Coze集成相关路由
	routes.SetupCozeIntegrationRoutes(api, cozeIntegrationHandler)

	// 🆕 .tmp文件HTML转换路由
	routes.SetupTMPHTMLRoutes(api, tmpHTMLHandler)

	// 🆕 WebView代理路由（解决小程序域名限制问题）
	webviewProxyHandler := handlers.NewWebViewProxyHandler()
	routes.SetupWebViewRoutes(api, webviewProxyHandler)
	log.Printf("✅ WebView代理路由初始化完成")

	// 🆕 练习生成相关路由
	routes.SetupExerciseRoutes(api, exerciseHandler)

	// PPT处理相关路由 - 课程相关
	pptCourse := api.Group("/ppt-course")
	{
		// 创建PPT处理器实例
		pptHandler := handlers.NewPPTHandler(db, courseRepo, audioStorageService)

		pptCourse.GET("/preview/:courseId", middleware.AuthRequired(), pptHandler.PreviewPPT)
		pptCourse.GET("/download/:courseId", middleware.AuthRequired(), pptHandler.DownloadPPT)
		pptCourse.POST("/generate/:courseId", middleware.AuthRequired(), pptHandler.GeneratePPTFile)
		pptCourse.GET("/status/:courseId", middleware.AuthRequired(), pptHandler.GetPPTStatus)
	}

	// 调试路由
	log.Println("注册的路由:")
	log.Println("- GET /api/v1/ai-assistant/popular-questions")
	log.Println("- GET /api/v1/ai-assistant/suggested-topics")
	log.Println("- POST /api/v1/ai-assistant/chat")
	log.Println("- GET /api/v1/ai-assistant/history")
	log.Println("- DELETE /api/v1/ai-assistant/history")
	log.Println("- GET /api/v1/ai-assistant/stats")

	// 测试路由
	api.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "测试路由正常工作",
		})
	})

	// 公开课件路由（不需要认证）- 必须在课件路由组之前注册
	api.GET("/courses/public", courseHandler.GetPublicCourses)

	// 课件相关路由（需要认证）
	courses := api.Group("/courses", middleware.AuthRequired())
	{
		courses.GET("", courseHandler.GetCourses)
		courses.POST("", courseHandler.CreateCourse)
		courses.POST("/upload", courseHandler.UploadCourse)

		// 搜索路由（放在具体ID路由之前）
		courses.GET("/search", courseHandler.SearchCourses)

		// 学习记录相关路由（放在具体ID路由之前）
		courses.GET("/learning-records", courseHandler.GetLearningRecords)
		courses.GET("/learning-records/stats", courseHandler.GetLearningStats)
		courses.POST("/learning-records/:id/reset", courseHandler.ResetLearningProgress)
		courses.DELETE("/learning-records/:id", courseHandler.DeleteLearningRecord)

		// 特殊路由（放在具体ID路由之前）
		courses.GET("/latest", courseHandler.GetLatestCourse)

		// 具体ID的路由（全部统一为:id参数）
		courses.GET("/:id/generation-status", courseHandler.GetGenerationStatus)
		courses.GET("/:id/progress", courseHandler.GetProgress)
		courses.PUT("/:id/progress", courseHandler.UpdateLearningProgress)
		courses.POST("/:id/complete", courseHandler.CompleteCourse)
		courses.POST("/:id/cancel", courseHandler.CancelGeneration)
		courses.POST("/:id/regenerate", courseHandler.RegenerateCourse)
		courses.GET("/:id", courseHandler.GetCourse)
		courses.PUT("/:id", courseHandler.UpdateCourse)
		courses.DELETE("/:id", courseHandler.DeleteCourse)
	}

	// 练习相关路由
	quiz := api.Group("/quiz", middleware.AuthRequired())
	{
		// 根据课件生成练习
		quiz.POST("/generate/:id", quizHandler.GenerateQuiz)

		// 练习操作
		quiz.GET("/:id", quizHandler.GetQuiz)
		quiz.POST("/:id/start", quizHandler.StartQuiz)
		quiz.GET("/:id/stats", quizHandler.GetQuizStats)

		// 练习答题
		quiz.POST("/attempts/:id/answer", quizHandler.SubmitAnswer)
		quiz.POST("/attempts/:id/submit", quizHandler.SubmitQuiz)
		quiz.GET("/attempts/:id", quizHandler.GetAttempt)
	}

	// WebSocket路由
	r.GET("/ws", websocket.HandleWebSocket(hub))
	r.GET("/ws/generation/:id", websocket.HandleGenerationWebSocket(hub))

	// 配置相关路由（公开访问，不需要认证）
	config := api.Group("/config")
	{
		config.GET("/client", configHandler.GetClientConfig)
	}

	// 健康检查
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"message": "AI课堂服务运行正常",
		})
	})
}

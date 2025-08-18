package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"ai-classroom/internal/middleware"
	"ai-classroom/internal/models"
	"ai-classroom/internal/services"
	"ai-classroom/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIContentHandler struct {
	aiAnalysisService *services.AIContentAnalysisService
	db                *gorm.DB
}

func NewAIContentHandler(
	aiAnalysisService *services.AIContentAnalysisService,
	db *gorm.DB,
) *AIContentHandler {
	return &AIContentHandler{
		aiAnalysisService: aiAnalysisService,
		db:                db,
	}
}

// AIURLAnalysisRequest AI URL分析请求
type AIURLAnalysisRequest struct {
	URL          string `json:"url" binding:"required"`
	AnalysisType string `json:"analysis_type"` // comprehensive, summary, technical
	EngineType   string `json:"engine_type"`   // dashscope, coze, hybrid
	Language     string `json:"language"`      // zh-CN, en-US
}

// AIGeneratePPTRequest AI生成PPT请求
type AIGeneratePPTRequest struct {
	AIContent        *models.AIAnalysisResponse    `json:"ai_content" binding:"required"`
	GenerationParams *services.PPTGenerationParams `json:"generation_params"`
}

// AIURLAnalysis AI URL分析接口
func (h *AIContentHandler) AIURLAnalysis(c *gin.Context) {
	log.Printf("🔍 [API] 收到AI URL分析请求")

	var req AIURLAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ [API] 请求参数绑定失败: %v", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	log.Printf("🔍 [API] 请求参数: URL=%s, 分析类型=%s, 引擎类型=%s, 语言=%s",
		req.URL, req.AnalysisType, req.EngineType, req.Language)

	// 获取用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		log.Printf("❌ [API] 用户未认证")
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	log.Printf("🔍 [API] 用户ID: %d", userID)

	// 验证URL格式
	if !h.isValidURL(req.URL) {
		log.Printf("❌ [API] 无效的URL格式: %s", req.URL)
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的URL格式")
		return
	}

	// 设置默认值
	if req.AnalysisType == "" {
		req.AnalysisType = "comprehensive"
	}
	if req.EngineType == "" {
		req.EngineType = "dashscope"
	}
	if req.Language == "" {
		req.Language = "zh-CN"
	}

	log.Printf("🔍 [API] 使用默认值: 分析类型=%s, 引擎类型=%s, 语言=%s",
		req.AnalysisType, req.EngineType, req.Language)

	// 构建分析请求
	analysisReq := &services.AnalysisRequest{
		URL:          req.URL,
		AnalysisType: req.AnalysisType,
		EngineType:   services.AIEngineType(req.EngineType),
		Language:     req.Language,
	}

	log.Printf("🔍 [API] 开始调用AI分析服务")
	// 调用AI分析服务
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := h.aiAnalysisService.AnalyzeURL(ctx, analysisReq)
	if err != nil {
		log.Printf("❌ [API] AI分析失败: %v", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "AI分析失败", err.Error())

		// 记录错误分析历史（异步）
		go h.recordErrorAnalysisHistory(userID, req, err.Error())
		return
	}

	log.Printf("✅ [API] AI分析成功，结果标题: %s", result.Title)
	// 记录成功的分析历史
	go h.recordSuccessAnalysisHistory(userID, analysisReq, result)

	utils.SuccessResponse(c, "AI分析完成", result)
}

// AIGeneratePPT AI生成PPT接口
func (h *AIContentHandler) AIGeneratePPT(c *gin.Context) {
	log.Printf("🔍 [API] 收到AI生成PPT请求")

	var req AIGeneratePPTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ [API] PPT生成请求参数绑定失败: %v", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	log.Printf("🔍 [API] AI内容标题: %s", req.AIContent.Title)

	// 获取用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		log.Printf("❌ [API] 用户未认证")
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	log.Printf("🔍 [API] 用户ID: %d", userID)

	// 设置默认生成参数
	if req.GenerationParams == nil {
		log.Printf("🔍 [API] 使用默认生成参数")
		req.GenerationParams = &services.PPTGenerationParams{
			SlideCount:           15,
			Style:                "professional",
			IncludeCodeExamples:  true,
			IncludeBestPractices: true,
			Template:             "technical",
			Language:             "zh-CN",
		}
	} else {
		log.Printf("🔍 [API] 使用自定义生成参数: 幻灯片数量=%d, 样式=%s",
			req.GenerationParams.SlideCount, req.GenerationParams.Style)
	}

	log.Printf("🔍 [API] 开始生成课程")
	// 生成课程
	course, err := h.aiAnalysisService.GenerateCourseFromAnalysis(
		userID,
		req.AIContent,
		req.GenerationParams,
	)
	if err != nil {
		log.Printf("❌ [API] 生成课程失败: %v", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成课程失败", err.Error())
		return
	}

	log.Printf("✅ [API] 课程生成成功，课程ID: %d", course.ID)

	// 重新查询课程以获取完整信息
	log.Printf("🔍 [API] 重新查询课程完整信息")
	var fullCourse models.Course
	if err := h.db.Preload("Slides").First(&fullCourse, course.ID).Error; err != nil {
		log.Printf("❌ [API] 获取课程信息失败: %v", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取课程信息失败", err.Error())
		return
	}

	log.Printf("✅ [API] 课程信息查询成功: 标题=%s, 幻灯片数量=%d",
		fullCourse.Title, len(fullCourse.Slides))

	utils.SuccessResponse(c, "PPT生成成功", gin.H{
		"course_id":    fullCourse.ID,
		"title":        fullCourse.Title,
		"description":  fullCourse.Description,
		"slides_count": len(fullCourse.Slides),
		"source_url":   fullCourse.SourceURL,
		"status":       fullCourse.Status,
		"created_at":   fullCourse.CreatedAt,
	})
}

// GetAnalysisHistory 获取分析历史
func (h *AIContentHandler) GetAnalysisHistory(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	engineType := c.Query("engine_type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 构建查询
	query := h.db.Where("user_id = ?", userID)
	if engineType != "" {
		query = query.Where("engine_type = ?", engineType)
	}

	// 获取总数
	var total int64
	query.Model(&models.AIAnalysisRecord{}).Count(&total)

	// 获取记录
	var records []models.AIAnalysisRecord
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取历史记录失败", err.Error())
		return
	}

	// 统计引擎使用情况
	var engineStats []map[string]interface{}
	h.db.Raw(`
		SELECT engine_type, COUNT(*) as count, AVG(process_time) as avg_time 
		FROM ai_analysis_records 
		WHERE user_id = ? 
		GROUP BY engine_type
	`, userID).Scan(&engineStats)

	utils.SuccessResponse(c, "获取分析历史成功", gin.H{
		"records":      records,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"engine_stats": engineStats,
	})
}

// DeleteAnalysis 删除分析记录
func (h *AIContentHandler) DeleteAnalysis(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	analysisID := c.Param("id")
	if analysisID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "分析ID不能为空")
		return
	}

	// 检查记录是否存在且属于当前用户
	var record models.AIAnalysisRecord
	if err := h.db.Where("id = ? AND user_id = ?", analysisID, userID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "分析记录不存在")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询记录失败", err.Error())
		}
		return
	}

	// 删除记录
	if err := h.db.Delete(&record).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除记录失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "删除成功", nil)
}

// GetEngineStatus 获取引擎状态（预留接口）
func (h *AIContentHandler) GetEngineStatus(c *gin.Context) {
	status := gin.H{
		"dashscope": gin.H{
			"available": true,
			"status":    "online",
			"features":  []string{"url_analysis", "content_generation", "ppt_creation"},
		},
		"coze": gin.H{
			"available": false,
			"status":    "coming_soon",
			"features":  []string{"specialized_analysis", "domain_expertise", "enhanced_quality"},
		},
		"hybrid": gin.H{
			"available": false,
			"status":    "coming_soon",
			"features":  []string{"best_of_both", "parallel_processing", "result_fusion"},
		},
	}

	utils.SuccessResponse(c, "引擎状态获取成功", status)
}

// recordSuccessAnalysisHistory 记录成功的分析历史
func (h *AIContentHandler) recordSuccessAnalysisHistory(
	userID uint,
	req *services.AnalysisRequest,
	result *models.AIAnalysisResponse,
) {
	// 计算处理时间（如果有的话）
	processTime := 0
	if result.Metadata != nil && !result.Metadata.AnalysisTime.IsZero() {
		processTime = int(time.Since(result.Metadata.AnalysisTime).Milliseconds())
	}

	record := &models.AIAnalysisRecord{
		UserID:       userID,
		URL:          req.URL,
		AnalysisType: req.AnalysisType,
		EngineType:   string(req.EngineType),
		Status:       "completed",
		ProcessTime:  processTime,
		CreatedAt:    time.Now(),
	}

	if err := h.db.Create(record).Error; err != nil {
		// 记录日志但不影响主流程
		utils.LogError("保存分析记录失败", err)
	}
}

// recordErrorAnalysisHistory 记录错误的分析历史
func (h *AIContentHandler) recordErrorAnalysisHistory(
	userID uint,
	req AIURLAnalysisRequest,
	errorMessage string,
) {
	record := &models.AIAnalysisRecord{
		UserID:       userID,
		URL:          req.URL,
		AnalysisType: req.AnalysisType,
		EngineType:   req.EngineType,
		Status:       "failed",
		ErrorMessage: errorMessage,
		CreatedAt:    time.Now(),
	}

	if err := h.db.Create(record).Error; err != nil {
		utils.LogError("保存错误分析记录失败", err)
	}
}

// isValidURL 验证URL格式
func (h *AIContentHandler) isValidURL(url string) bool {
	// 简单的URL格式验证
	return len(url) > 0 && (len(url) > 7 && url[:7] == "http://" ||
		len(url) > 8 && url[:8] == "https://")
}

package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ai-classroom/internal/models"
	"ai-classroom/internal/services"
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/parser"
	"ai-classroom/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ContentHandler 内容处理器
type ContentHandler struct {
	db                    *gorm.DB
	contentService        *services.ContentService
	aiAssistantService    *services.AIAssistantService
	contentSummaryService *services.ContentSummaryService
	pptGenerationService  *services.PPTGenerationService
	enhancedPPTService    *services.EnhancedPPTService       // 新增增强PPT服务
	keywordService        *services.KeywordExtractionService // ✅ 新增关键词提取服务
	slideService          *services.SlideCreationService     // ✅ 新增幻灯片创建服务
}

// NewContentHandler 创建内容处理器
func NewContentHandler(
	db *gorm.DB,
	aiClient *ai.DashScopeClient,
	contentSummaryService *services.ContentSummaryService,
	pptGenerationService *services.PPTGenerationService,
	enhancedPPTService *services.EnhancedPPTService, // 新增参数
) *ContentHandler {
	// ✅ 创建关键词提取服务和幻灯片创建服务
	keywordService := services.NewKeywordExtractionService(aiClient)
	slideService := services.NewSlideCreationService(db, keywordService)

	return &ContentHandler{
		db:                    db,
		contentService:        services.NewContentService(db, aiClient),
		aiAssistantService:    services.NewAIAssistantService(db, aiClient),
		contentSummaryService: contentSummaryService,
		pptGenerationService:  pptGenerationService,
		enhancedPPTService:    enhancedPPTService,
		keywordService:        keywordService, // ✅ 新增字段
		slideService:          slideService,   // ✅ 新增字段
	}
}

// ExtractWebContent 提取网页内容
// @Summary 提取网页内容
// @Description 从指定URL提取网页内容，支持多个主流平台
// @Tags content
// @Accept json
// @Produce json
// @Param request body ExtractWebContentRequest true "提取请求"
// @Success 200 {object} utils.Response{data=services.ContentResult} "提取成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/extract [post]
func (h *ContentHandler) ExtractWebContent(c *gin.Context) {
	var req ExtractWebContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 构建内容请求
	contentReq := &services.ContentRequest{
		Type:           "url",
		URL:            req.URL,
		CourseSettings: req.CourseSettings,
	}

	// 处理内容
	result, err := h.contentService.ProcessContent(contentReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "内容提取失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "内容提取成功", result)
}

// ProcessTextContent 处理文本内容
// @Summary 处理文本内容
// @Description 处理用户输入的文本内容
// @Tags content
// @Accept json
// @Produce json
// @Param request body ProcessTextContentRequest true "文本处理请求"
// @Success 200 {object} utils.Response{data=services.ContentResult} "处理成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/text [post]
func (h *ContentHandler) ProcessTextContent(c *gin.Context) {
	var req ProcessTextContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 构建内容请求
	contentReq := &services.ContentRequest{
		Type:           "text",
		Title:          req.Title,
		Content:        req.Content,
		CourseSettings: req.CourseSettings,
	}

	// 处理内容
	result, err := h.contentService.ProcessContent(contentReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "文本处理失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "文本处理成功", result)
}

// UploadDocument 上传文档
// @Summary 上传文档
// @Description 上传PDF、Word等文档并解析内容
// @Tags content
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文档文件"
// @Success 200 {object} utils.Response{data=services.ContentResult} "上传成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/upload [post]
func (h *ContentHandler) UploadDocument(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件上传失败", err.Error())
		return
	}

	// 检查文件大小（限制为10MB）
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if file.Size > maxFileSize {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件过大", "文件大小不能超过10MB")
		return
	}

	// 检查文件类型
	allowedTypes := []string{".pdf", ".doc", ".docx", ".txt", ".md"}
	if !isAllowedFileType(file.Filename, allowedTypes) {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件类型不支持", "只支持PDF、Word、TXT、Markdown文件")
		return
	}

	// 保存文件到临时目录
	tempDir := "./storage/temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建临时目录失败", err.Error())
		return
	}

	tempFilePath := filepath.Join(tempDir, file.Filename)
	if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存文件失败", err.Error())
		return
	}

	// 解析文档内容
	content, err := h.contentService.ProcessDocumentFile(file)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "文档解析失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "文档上传并解析成功", content)
}

// PreviewContent 预览内容
// @Summary 预览内容
// @Description 获取已处理内容的预览
// @Tags content
// @Accept json
// @Produce json
// @Param id path string true "内容ID"
// @Success 200 {object} utils.Response{data=services.ContentResult} "获取成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 404 {object} utils.Response "内容不存在"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/preview/{id} [get]
func (h *ContentHandler) PreviewContent(c *gin.Context) {
	contentID := c.Param("id")
	if contentID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", "内容ID不能为空")
		return
	}

	// TODO: 实现内容预览功能
	// 当前返回提示信息
	utils.ErrorResponse(c, http.StatusNotImplemented, "功能开发中", "内容预览功能正在开发中")
}

// ValidateURL 验证URL
// @Summary 验证URL
// @Description 验证URL是否支持内容提取
// @Tags content
// @Accept json
// @Produce json
// @Param request body ValidateURLRequest true "URL验证请求"
// @Success 200 {object} utils.Response{data=crawler.ValidationResult} "验证成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Router /api/v1/content/validate-url [post]
func (h *ContentHandler) ValidateURL(c *gin.Context) {
	var req ValidateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 验证URL
	result := h.contentService.ValidateURL(req.URL)

	if result.Valid {
		utils.SuccessResponse(c, "URL验证成功", result)
	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, "URL验证失败", result.Reason)
	}
}

// GetSupportedPlatforms 获取支持的平台列表
// @Summary 获取支持的平台列表
// @Description 获取所有支持内容提取的平台信息
// @Tags content
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=[]crawler.PlatformInfo} "获取成功"
// @Router /api/v1/content/platforms [get]
func (h *ContentHandler) GetSupportedPlatforms(c *gin.Context) {
	platforms := h.contentService.GetSupportedPlatforms()
	utils.SuccessResponse(c, "获取平台列表成功", platforms)
}

// GetProcessStats 获取处理统计
// @Summary 获取处理统计
// @Description 获取内容处理的统计信息
// @Tags content
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=services.ProcessStats} "获取成功"
// @Router /api/v1/content/stats [get]
func (h *ContentHandler) GetProcessStats(c *gin.Context) {
	stats := h.contentService.GetProcessStats()
	utils.SuccessResponse(c, "获取统计信息成功", stats)
}

// GetContentTypes 获取支持的内容类型
// @Summary 获取支持的内容类型
// @Description 获取所有支持的内容输入类型
// @Tags content
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=[]ContentTypeInfo} "获取成功"
// @Router /api/v1/content/types [get]
func (h *ContentHandler) GetContentTypes(c *gin.Context) {
	types := []ContentTypeInfo{
		{
			Type:        "url",
			Name:        "网页链接",
			Description: "从网页URL提取内容",
			Examples:    []string{"https://zhuanlan.zhihu.com/p/123456", "https://mp.weixin.qq.com/s/abc123"},
			Supported:   true,
		},
		{
			Type:        "text",
			Name:        "文本输入",
			Description: "直接输入文本内容",
			Examples:    []string{"手动输入的文章内容", "复制粘贴的文本"},
			Supported:   true,
		},
		{
			Type:        "file",
			Name:        "文档上传",
			Description: "上传PDF、Word、TXT等文档",
			Examples:    []string{"document.pdf", "article.docx", "notes.txt"},
			Supported:   true, // 已实现
		},
	}

	utils.SuccessResponse(c, "获取内容类型成功", types)
}

// BatchProcessContent 批量处理内容
// @Summary 批量处理内容
// @Description 批量处理多个内容请求
// @Tags content
// @Accept json
// @Produce json
// @Param request body BatchProcessRequest true "批量处理请求"
// @Success 200 {object} utils.Response{data=BatchProcessResponse} "处理成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/batch [post]
func (h *ContentHandler) BatchProcessContent(c *gin.Context) {
	var req BatchProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 检查请求数量限制
	if len(req.Requests) > 10 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求过多", "单次批量处理最多支持10个请求")
		return
	}

	// 批量处理
	results := make([]*services.ContentResult, 0, len(req.Requests))
	errors := make([]string, 0)

	for _, contentReq := range req.Requests {
		result, err := h.contentService.ProcessContent(contentReq)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			results = append(results, result)
		}
	}

	response := &BatchProcessResponse{
		Results:      results,
		SuccessCount: len(results),
		FailCount:    len(errors),
		Errors:       errors,
	}

	utils.SuccessResponse(c, "批量处理完成", response)
}

// AIAssistantChat AI助手对话
// @Summary AI助手对话
// @Description 与AI助手进行对话
// @Tags ai-assistant
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body AIAssistantChatRequest true "对话请求"
// @Success 200 {object} utils.Response{data=AIAssistantChatResponse} "对话成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 401 {object} utils.Response "用户未登录"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/ai-assistant/chat [post]
func (h *ContentHandler) AIAssistantChat(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	var req AIAssistantChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 调用AI助手服务
	chatReq := &services.ChatRequest{
		Message: req.Message,
		Context: req.Context,
	}

	response, err := h.aiAssistantService.Chat(userID.(uint), chatReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "AI对话失败", err.Error())
		return
	}

	// 转换为响应格式
	chatResponse := &AIAssistantChatResponse{
		Response: response.Response,
		Context:  response.Context,
	}

	utils.SuccessResponse(c, "AI对话成功", chatResponse)
}

// GetAIAssistantHistory 获取AI助手对话历史
// @Summary 获取AI助手对话历史
// @Description 获取用户的AI助手对话历史记录
// @Tags ai-assistant
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(50)
// @Success 200 {object} utils.Response{data=services.GetChatHistoryResponse} "获取成功"
// @Failure 401 {object} utils.Response "用户未登录"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/ai-assistant/history [get]
func (h *ContentHandler) GetAIAssistantHistory(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	// 构建请求
	req := &services.GetChatHistoryRequest{
		Page:  page,
		Limit: limit,
	}

	// 获取对话历史
	response, err := h.aiAssistantService.GetChatHistory(userID.(uint), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取对话历史失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "获取对话历史成功", response)
}

// ClearAIAssistantHistory 清空AI助手对话历史
// @Summary 清空AI助手对话历史
// @Description 清空用户的AI助手对话历史记录
// @Tags ai-assistant
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utils.Response "清空成功"
// @Failure 401 {object} utils.Response "用户未登录"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/ai-assistant/history [delete]
func (h *ContentHandler) ClearAIAssistantHistory(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	// 清空对话历史
	if err := h.aiAssistantService.ClearChatHistory(userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清空对话历史失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "清空对话历史成功", nil)
}

// GetAIAssistantStats 获取AI助手统计信息
// @Summary 获取AI助手统计信息
// @Description 获取用户的AI助手使用统计
// @Tags ai-assistant
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utils.Response{data=map[string]interface{}} "获取成功"
// @Failure 401 {object} utils.Response "用户未登录"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/ai-assistant/stats [get]
func (h *ContentHandler) GetAIAssistantStats(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	// 获取统计信息
	stats, err := h.aiAssistantService.GetChatStats(userID.(uint))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取统计信息失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "获取统计信息成功", stats)
}

// GetPopularQuestions 获取热门问题
// @Summary 获取热门问题
// @Description 获取AI助手的热门问题列表
// @Tags ai-assistant
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=[]string} "获取成功"
// @Router /api/v1/ai-assistant/popular-questions [get]
func (h *ContentHandler) GetPopularQuestions(c *gin.Context) {
	questions := h.aiAssistantService.GetPopularQuestions()
	utils.SuccessResponse(c, "获取热门问题成功", questions)
}

// GetSuggestedTopics 获取建议话题
// @Summary 获取建议话题
// @Description 获取AI助手的建议话题列表
// @Tags ai-assistant
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=[]map[string]string} "获取成功"
// @Router /api/v1/ai-assistant/suggested-topics [get]
func (h *ContentHandler) GetSuggestedTopics(c *gin.Context) {
	topics := h.aiAssistantService.GetSuggestedTopics()
	utils.SuccessResponse(c, "获取建议话题成功", topics)
}

// GenerateCourse 生成课件
// @Summary 生成课件
// @Description 根据内容生成完整的课件
// @Tags content
// @Accept json
// @Produce json
// @Param request body GenerateCourseRequest true "生成请求"
// @Success 200 {object} utils.Response{data=services.CourseGenerationResult} "生成成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/generate [post]
func (h *ContentHandler) GenerateCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	var req GenerateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 生成课件
	result, err := h.contentService.GenerateCourse(req.Content, userID.(uint))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "课件生成失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "课件生成成功", result)
}

// GenerateAISummary AI内容总结
// @Summary 生成AI内容总结
// @Description 使用AI对内容进行智能分析和总结
// @Tags content
// @Accept json
// @Produce json
// @Param request body GenerateAISummaryRequest true "AI总结请求"
// @Success 200 {object} utils.Response{data=services.SummaryResult} "总结成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// @Router /api/v1/content/ai-summary [post]
func (h *ContentHandler) GenerateAISummary(c *gin.Context) {
	var req GenerateAISummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 根据来源类型处理内容
	var contentToAnalyze string
	var sourceURL string

	switch req.SourceType {
	case "url":
		// 如果是URL类型，先爬取内容
		if req.Content == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "URL不能为空")
			return
		}

		// 使用ContentService的URL处理功能
		contentReq := &services.ContentRequest{
			Type: "url",
			URL:  req.Content,
		}

		contentResult, err := h.contentService.ProcessContent(contentReq)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "URL内容处理失败", err.Error())
			return
		}

		if contentResult.Status != "success" {
			utils.ErrorResponse(c, http.StatusInternalServerError, "URL内容处理失败", contentResult.Error)
			return
		}

		// 使用处理后的内容进行AI总结
		contentToAnalyze = contentResult.Content
		sourceURL = req.Content

		// 记录日志
		fmt.Printf("URL内容爬取成功，URL: %s, 内容长度: %d\n", sourceURL, len(contentToAnalyze))

	case "text":
		contentToAnalyze = req.Content
	case "file":
		// 文件内容已经在之前处理过了
		contentToAnalyze = req.Content
	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "不支持的内容类型")
		return
	}

	// 验证内容
	if contentToAnalyze == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "内容不能为空")
		return
	}

	// 构建AI总结请求
	summaryReq := &services.SummaryRequest{
		Content:        contentToAnalyze,
		SummaryLevel:   req.SummaryLevel,
		TargetAudience: req.TargetAudience,
		SourceType:     req.SourceType,
	}

	// 生成AI总结
	result, err := h.contentSummaryService.GenerateSummary(summaryReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "AI总结生成失败", err.Error())
		return
	}

	// 如果是从URL爬取的内容，在结果中添加源URL信息
	if sourceURL != "" {
		// 可以在结果中添加源URL信息
		fmt.Printf("AI总结完成，源URL: %s, 总结长度: %d\n", sourceURL, len(result.Summary))
	}

	utils.SuccessResponse(c, "AI总结生成成功", result)
}

// GenerateEnhancedPPT 生成增强PPT
// @Summary 生成增强PPT
// @Description 基于AI摘要生成增强的PPT
// @Tags content
// @Accept json
// @Produce json
// @Param request body GenerateEnhancedPPTRequest true "增强PPT生成请求"
// @Success 200 {object} utils.Response{data=EnhancedPPTResponse} "生成成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 500 {object} utils.Response "服务器错误"
// GenerateEnhancedPPT 兼容性方法，重定向到带限制的版本
func (h *ContentHandler) GenerateEnhancedPPT(c *gin.Context) {
	// 直接调用带限制的版本
	h.GenerateEnhancedPPTWithLimits(c)
}

// GenerateEnhancedPPTWithLimits 生成增强PPT（带页数限制）
func (h *ContentHandler) GenerateEnhancedPPTWithLimits(c *gin.Context) {
	var req GenerateEnhancedPPTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取用户信息（用于页数限制）
	userIDInterface, exists := c.Get("user_id")
	var userType string = "regular" // 默认为普通用户
	var userID string = "anonymous"

	if exists {
		switch v := userIDInterface.(type) {
		case uint:
			userID = strconv.FormatUint(uint64(v), 10)
		case int:
			userID = strconv.Itoa(v)
		case int64:
			userID = strconv.FormatInt(v, 10)
		case string:
			userID = v
		default:
			userID = fmt.Sprintf("%v", v)
		}
	}

	// 获取用户类型（VIP等级）
	vipLevelInterface, exists := c.Get("vip_level")
	if exists {
		if vipLevel, ok := vipLevelInterface.(int); ok && vipLevel > 0 {
			userType = "vip"
		}
	}

	// 获取页数限制信息
	limits := h.enhancedPPTService.GetSlideCountLimits(userType)

	// 验证请求的页数是否超出限制
	requestedSlides := req.SlideCount
	if requestedSlides <= 0 {
		requestedSlides = limits["default"]
	}

	maxAllowed := limits["user_max"]
	if requestedSlides > maxAllowed {
		utils.ErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("请求的幻灯片数量(%d)超过您的配额限制(%d页)。%s用户最多可生成%d页PPT",
				requestedSlides, maxAllowed, userType, maxAllowed))
		return
	}

	// 构建增强PPT生成请求
	enhancedReq := &services.EnhancedGenerationRequest{
		SourceType:    req.SourceType,
		Content:       req.Content,
		Title:         req.Title,
		SlideCount:    requestedSlides,
		Template:      req.Template,
		Style:         req.Style,
		Audience:      req.Audience,
		Difficulty:    req.Difficulty,
		Language:      req.Language,
		UserType:      userType,
		UserID:        userID,
		IncludeImages: req.IncludeImages,
		IncludeNotes:  req.IncludeNotes,
		AutoOptimize:  req.AutoOptimize,
		GenerateAudio: req.GenerateAudio,
		VoiceType:     req.VoiceType,
	}

	// 生成PPT
	result, err := h.enhancedPPTService.GeneratePPT(enhancedReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成PPT失败: "+err.Error())
		return
	}

	if !result.Success {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成PPT失败: "+result.Error)
		return
	}

	// 如果需要创建课程
	var courseID uint
	if req.CreateCourse {
		courseID, err = h.createCourseFromPPT(result, req, userID)
		if err != nil {
			// 即使课程创建失败，PPT已经生成，所以返回警告而不是错误
			result.Warnings = append(result.Warnings, "PPT生成成功，但创建课程失败: "+err.Error())
		}
	}

	// 构建响应
	response := map[string]interface{}{
		"success":              result.Success,
		"title":                result.Title,
		"total_slides":         result.TotalSlides,
		"actual_slides":        result.ActualSlides,
		"requested_slides":     result.RequestedSlides,
		"limited_by_quota":     result.LimitedByQuota,
		"generation_time":      result.GenerationTime.String(),
		"ppt_file_path":        result.PPTFilePath,
		"ppt_file_url":         result.PPTFileURL,
		"source_info":          result.SourceInfo,
		"warnings":             result.Warnings,
		"slides":               result.Slides,
		"slide_limits":         limits,
		"user_type":            userType,
		"audio_generated":      result.AudioGenerated,
		"audio_files":          result.AudioFiles,
		"total_audio_duration": result.TotalAudioDuration,
	}

	if courseID > 0 {
		response["course_id"] = courseID
	}

	utils.SuccessResponse(c, "增强PPT生成成功", response)
}

// GetSlideCountLimits 获取页数限制信息
func (h *ContentHandler) GetSlideCountLimits(c *gin.Context) {
	// 获取用户信息
	userInterface, exists := c.Get("user")
	userType := "regular" // 默认为普通用户

	if exists {
		if user, ok := userInterface.(map[string]interface{}); ok {
			if utype, ok := user["user_type"].(string); ok {
				userType = utype
			}
		}
	}

	// 从查询参数获取用户类型（可选）
	if queryUserType := c.Query("user_type"); queryUserType != "" {
		userType = queryUserType
	}

	limits := h.enhancedPPTService.GetSlideCountLimits(userType)

	response := map[string]interface{}{
		"user_type": userType,
		"limits":    limits,
		"message":   fmt.Sprintf("%s用户最多可生成%d页PPT", userType, limits["user_max"]),
	}

	utils.SuccessResponse(c, "获取页数限制成功", response)
}

// ProcessFileContent 处理文件内容
func (h *ContentHandler) ProcessFileContent(c *gin.Context) {
	filePath := c.PostForm("file_path")
	if filePath == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件路径不能为空")
		return
	}

	// 使用文件处理器处理文件
	fileProcessor := parser.NewFileProcessor()
	result, err := fileProcessor.ProcessFile(filePath)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "处理文件失败: "+err.Error())
		return
	}

	if !result.Success {
		utils.ErrorResponse(c, http.StatusInternalServerError, "处理文件失败: "+result.Error)
		return
	}

	response := map[string]interface{}{
		"success":         result.Success,
		"content":         result.Content,
		"supported_types": fileProcessor.GetSupportedTypes(),
	}

	utils.SuccessResponse(c, "文件处理成功", response)
}

// GetSupportedFileTypes 获取支持的文件类型
func (h *ContentHandler) GetSupportedFileTypes(c *gin.Context) {
	// 创建文件处理器实例获取支持的文件类型
	fileProcessor := parser.NewFileProcessor()
	supportedTypes := fileProcessor.GetSupportedTypes()

	response := map[string]interface{}{
		"supported_types": supportedTypes,
		"description":     "当前支持的文件格式列表",
	}

	utils.SuccessResponse(c, "获取支持文件类型成功", response)
}

// mapSourceType 映射源类型
func (h *ContentHandler) mapSourceType(sourceType string) models.SourceType {
	switch sourceType {
	case "file":
		return models.SourceTypeDocument
	case "url":
		return models.SourceTypeURL
	case "text":
		return models.SourceTypeText
	default:
		return models.SourceTypeDocument
	}
}

// createCourseFromPPT 从PPT结果创建课程
func (h *ContentHandler) createCourseFromPPT(pptResult *services.EnhancedGenerationResult, req GenerateEnhancedPPTRequest, userID string) (uint, error) {
	// 获取用户ID的数字形式
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("无效的用户ID: %s", userID)
	}

	// 开始事务
	tx := h.db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("开始事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建课程记录
	course := &models.Course{
		UserID:        uint(userIDUint),
		Title:         req.Title,
		Description:   fmt.Sprintf("基于%s生成的PPT课件", req.SourceType),
		Category:      models.CourseCategory(req.Category),
		SourceType:    h.mapSourceType(req.SourceType), // 使用映射后的source_type
		SourceContent: req.Content,
		Status:        models.CourseStatusCompleted, // 直接标记为完成，因为PPT已生成
		SlidesCount:   pptResult.TotalSlides,
		IsPublic:      req.IsPublic,
		VoiceType:     "zhixiaobai",
		PPTFilePath:   pptResult.PPTFilePath, // 保存PPT文件路径
		GenerationParams: models.GenerationParams{
			"template":    req.Template,
			"style":       req.Style,
			"audience":    req.Audience,
			"difficulty":  req.Difficulty,
			"slide_count": pptResult.RequestedSlides,
		},
	}

	// 保存课程
	if err := tx.Create(course).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("创建课程失败: %w", err)
	}

	fmt.Printf("course.ID after create: %d\n", course.ID)

	// 验证课程ID是否正确生成
	if course.ID == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("课程ID生成失败")
	}

	// 确保课程ID正确返回 - 强制重新查询获取最新ID
	var latestCourse models.Course
	if err := tx.Where("user_id = ?", course.UserID).Order("id DESC").First(&latestCourse).Error; err != nil {
		// 如果查询失败，仍使用原有ID
		fmt.Printf("⚠️ 查询最新课程失败，使用原ID: %d\n", course.ID)
	} else {
		// 使用数据库中的最新ID
		if latestCourse.ID > course.ID {
			course.ID = latestCourse.ID
			fmt.Printf("✅ 更新为最新课程ID: %d\n", course.ID)
		}
	}

	// 保存幻灯片
	totalDuration := 0
	for i, slideContent := range pptResult.Slides {
		// 将EnhancedSlideContent转换为Slide模型
		slide := &models.Slide{
			CourseID:     course.ID,
			SlideNumber:  i + 1,
			Title:        slideContent.Title,
			Content:      slideContent.MainContent,
			SpeakerNotes: slideContent.SpeakerNotes,
			LayoutType:   models.LayoutType(slideContent.SlideType),
		}

		// 如果有要点列表，将其转换为内容
		if len(slideContent.BulletPoints) > 0 {
			bulletContent := strings.Join(slideContent.BulletPoints, "\n• ")
			if bulletContent != "" {
				slide.Content = "• " + bulletContent
			}
		}

		// 估算音频时长（基于内容长度）
		estimatedDuration := h.estimateSlideDuration(slide.Title + " " + slide.Content + " " + slide.SpeakerNotes)
		slide.Duration = estimatedDuration
		totalDuration += estimatedDuration

		if err := tx.Create(slide).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("保存幻灯片%d失败: %w", i+1, err)
		}
	}

	// 更新课程总时长
	if err := tx.Model(course).Update("duration", totalDuration).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("更新课程时长失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("课程创建成功: ID=%d, 标题=%s, 用户ID=%s, 幻灯片数=%d\n",
		course.ID, course.Title, userID, pptResult.TotalSlides)

	return course.ID, nil
}

// 请求和响应结构体

type ExtractWebContentRequest struct {
	URL            string                   `json:"url" binding:"required"`
	CourseSettings *services.CourseSettings `json:"course_settings,omitempty"`
}

type ProcessTextContentRequest struct {
	Title          string                   `json:"title"`
	Content        string                   `json:"content" binding:"required"`
	CourseSettings *services.CourseSettings `json:"course_settings,omitempty"`
}

type ValidateURLRequest struct {
	URL string `json:"url" binding:"required"`
}

type ContentTypeInfo struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
	Supported   bool     `json:"supported"`
}

type BatchProcessRequest struct {
	Requests []*services.ContentRequest `json:"requests" binding:"required"`
}

type BatchProcessResponse struct {
	Results      []*services.ContentResult `json:"results"`
	SuccessCount int                       `json:"success_count"`
	FailCount    int                       `json:"fail_count"`
	Errors       []string                  `json:"errors,omitempty"`
}

type AIAssistantChatRequest struct {
	Message string `json:"message" binding:"required"`
	Context string `json:"context,omitempty"`
}

type AIAssistantChatResponse struct {
	Response string `json:"response"`
	Context  string `json:"context,omitempty"`
}

type AIAssistantChatHistory struct {
	ID        uint   `json:"id"`
	Message   string `json:"message"`
	Response  string `json:"response"`
	CreatedAt string `json:"created_at"`
}

type GenerateCourseRequest struct {
	Content string `json:"content" binding:"required"`
	Type    string `json:"type" binding:"required"` // text, url, file
}

type GenerateAISummaryRequest struct {
	Content        string `json:"content" binding:"required"`
	SummaryLevel   string `json:"summary_level"`   // brief, detailed, comprehensive
	TargetAudience string `json:"target_audience"` // student, professional, general
	SourceType     string `json:"source_type"`     // text, url, file
}

type GenerateEnhancedPPTRequest struct {
	// 内容来源
	SourceType string `json:"source_type" binding:"required"` // url, file, text
	Content    string `json:"content" binding:"required"`     // 内容或URL或文件路径

	// 生成参数
	Title      string `json:"title"`       // PPT标题
	SlideCount int    `json:"slide_count"` // 期望的幻灯片数量
	Template   string `json:"template"`    // 模板类型
	Style      string `json:"style"`       // 风格
	Audience   string `json:"audience"`    // 目标受众
	Difficulty string `json:"difficulty"`  // 难度等级
	Language   string `json:"language"`    // 语言

	// 选项
	IncludeImages bool `json:"include_images"` // 是否包含图片建议
	IncludeNotes  bool `json:"include_notes"`  // 是否包含备注
	AutoOptimize  bool `json:"auto_optimize"`  // 是否自动优化内容长度

	// 课程相关
	CreateCourse bool   `json:"create_course"` // 是否创建课程
	Category     string `json:"category"`      // 分类
	Tags         string `json:"tags"`          // 标签
	IsPublic     bool   `json:"is_public"`     // 是否公开

	// 音频相关
	GenerateAudio bool   `json:"generate_audio"` // 是否生成音频
	VoiceType     string `json:"voice_type"`     // 语音类型
}

type EnhancedPPTResponse struct {
	GenerationResult *services.GenerationResult `json:"generation_result"`
	Title            string                     `json:"title"`
	CourseID         *uint64                    `json:"course_id,omitempty"`
}

// createCourseFromPPTResult 从PPT结果创建课程记录
func (h *ContentHandler) createCourseFromPPTResult(userID uint, title string, summaryResult *services.SummaryResult, pptResult *services.GenerationResult) (uint64, error) {
	// 开始事务
	tx := h.db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	// 创建课程记录
	course := &models.Course{
		UserID:        userID,
		Title:         title,
		Description:   summaryResult.Summary,
		Category:      models.CategoryGeneral, // 可以根据内容类型智能选择
		SourceType:    models.SourceTypeText,
		SourceContent: summaryResult.Summary,
		Status:        models.CourseStatusCompleted, // 直接标记为完成，因为幻灯片已生成
		SlidesCount:   len(pptResult.Slides),
		IsPublic:      false, // 默认私有
		VoiceType:     "zhixiaobai",
	}

	// 保存课程
	if err := tx.Create(course).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("创建课程失败: %w", err)
	}

	// 验证课程ID是否正确生成
	if course.ID == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("课程ID生成失败")
	}

	// 保存幻灯片
	totalDuration := 0
	for i, slideContent := range pptResult.Slides {
		// 将SlideContent转换为Slide模型
		var content string
		if len(slideContent.Content) > 0 {
			content = strings.Join(slideContent.Content, "\n")
		}

		slide := &models.Slide{
			CourseID:     course.ID,
			SlideNumber:  i + 1,
			Title:        slideContent.Title,
			Content:      content,
			SpeakerNotes: slideContent.Notes,
			LayoutType:   models.LayoutType(slideContent.SlideType),
		}

		// 如果有要点列表，将其转换为内容
		if len(slideContent.BulletPoints) > 0 {
			bulletContent := strings.Join(slideContent.BulletPoints, "\n• ")
			if bulletContent != "" {
				slide.Content = "• " + bulletContent
			}
		}

		// 估算音频时长（基于内容长度）
		estimatedDuration := h.estimateSlideDuration(slide.Title + " " + slide.Content + " " + slide.SpeakerNotes)
		slide.Duration = estimatedDuration
		totalDuration += estimatedDuration

		if err := tx.Create(slide).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("保存幻灯片%d失败: %w", i+1, err)
		}
	}

	// 更新课程总时长
	if err := tx.Model(course).Update("duration", totalDuration).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("更新课程时长失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	return uint64(course.ID), nil
}

// estimateSlideDuration 估算幻灯片音频时长（基于文本长度）
func (h *ContentHandler) estimateSlideDuration(text string) int {
	// 简单估算：中文约每分钟300字，英文约每分钟150词
	textLength := len([]rune(text))
	if textLength == 0 {
		return 10 // 默认10秒
	}

	// 假设平均阅读速度为每分钟200字
	estimatedMinutes := float64(textLength) / 200.0
	estimatedSeconds := int(estimatedMinutes * 60)

	// 最少5秒，最多300秒（5分钟）
	if estimatedSeconds < 5 {
		estimatedSeconds = 5
	} else if estimatedSeconds > 300 {
		estimatedSeconds = 300
	}

	return estimatedSeconds
}

// 辅助函数

func isAllowedFileType(filename string, allowedTypes []string) bool {
	for _, ext := range allowedTypes {
		if len(filename) >= len(ext) && filename[len(filename)-len(ext):] == ext {
			return true
		}
	}
	return false
}

// GetAudioProgress 获取音频生成进度
func (h *ContentHandler) GetAudioProgress(c *gin.Context) {
	courseIDStr := c.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的课程ID",
		})
		return
	}

	// 获取音频生成进度
	progress, err := h.enhancedPPTService.GetAudioProgress(uint(courseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "未找到音频生成进度",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取音频生成进度成功",
		"data":    progress,
	})
}

// CreateCourse 创建课程
func (h *ContentHandler) CreateCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 解析请求
	var req struct {
		SourceType    string `json:"source_type" binding:"required"`
		Content       string `json:"content" binding:"required"`
		Title         string `json:"title" binding:"required"`
		SlideCount    int    `json:"slide_count"`
		Template      string `json:"template"`
		Style         string `json:"style"`
		Audience      string `json:"audience"`
		Difficulty    string `json:"difficulty"`
		Language      string `json:"language"`
		IncludeImages bool   `json:"include_images"`
		IncludeNotes  bool   `json:"include_notes"`
		AutoOptimize  bool   `json:"auto_optimize"`
		CreateCourse  bool   `json:"create_course"`
		Category      string `json:"category"`
		Tags          string `json:"tags"`
		IsPublic      bool   `json:"is_public"`
		GenerateAudio bool   `json:"generate_audio"`
		VoiceType     string `json:"voice_type"`
		URL           string `json:"url"`
		FilePath      string `json:"file_path"`
		FileSize      int64  `json:"file_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("请求参数错误: %v", err)})
		return
	}

	// 设置默认值
	if req.SlideCount == 0 {
		req.SlideCount = 8
	}
	if req.Template == "" {
		req.Template = "professional"
	}
	if req.VoiceType == "" {
		req.VoiceType = "zhixiaobai"
	}

	// 创建课程模型
	course := &models.Course{
		UserID:        userID.(uint),
		Title:         req.Title,
		Description:   req.Content[:minInt(500, len(req.Content))], // 取前500字符作为描述
		Category:      models.CourseCategory(req.Category),
		SourceType:    models.SourceType(req.SourceType),
		SourceContent: req.Content,
		SourceURL:     req.URL,
		FilePath:      req.FilePath,
		FileSize:      req.FileSize,
		IsPublic:      req.IsPublic,
		VoiceType:     req.VoiceType,
		Status:        "generating",
	}

	// 处理标签
	if req.Tags != "" {
		tagList := strings.Split(req.Tags, ",")
		var cleanTags []string
		for _, tag := range tagList {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				cleanTags = append(cleanTags, trimmed)
			}
		}
		course.Tags = cleanTags
	}

	// 保存课程到数据库
	if err := h.db.Create(course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建课程失败: %v", err)})
		return
	}

	// 异步生成PPT内容和幻灯片
	courseReq := CourseGenerationRequest{
		SourceType:    req.SourceType,
		Content:       req.Content,
		Title:         req.Title,
		SlideCount:    req.SlideCount,
		Template:      req.Template,
		Style:         req.Style,
		Audience:      req.Audience,
		Difficulty:    req.Difficulty,
		Language:      req.Language,
		IncludeImages: req.IncludeImages,
		IncludeNotes:  req.IncludeNotes,
		AutoOptimize:  req.AutoOptimize,
		CreateCourse:  req.CreateCourse,
		Category:      req.Category,
		Tags:          req.Tags,
		IsPublic:      req.IsPublic,
		GenerateAudio: req.GenerateAudio,
		VoiceType:     req.VoiceType,
		URL:           req.URL,
		FilePath:      req.FilePath,
		FileSize:      req.FileSize,
	}
	go h.generateCourseContent(course, courseReq)

	c.JSON(http.StatusCreated, gin.H{
		"message":   "课程创建成功，正在生成内容",
		"course_id": course.ID,
		"status":    course.Status,
	})
}

type CourseGenerationRequest struct {
	SourceType    string `json:"source_type"`
	Content       string `json:"content"`
	Title         string `json:"title"`
	SlideCount    int    `json:"slide_count"`
	Template      string `json:"template"`
	Style         string `json:"style"`
	Audience      string `json:"audience"`
	Difficulty    string `json:"difficulty"`
	Language      string `json:"language"`
	IncludeImages bool   `json:"include_images"`
	IncludeNotes  bool   `json:"include_notes"`
	AutoOptimize  bool   `json:"auto_optimize"`
	CreateCourse  bool   `json:"create_course"`
	Category      string `json:"category"`
	Tags          string `json:"tags"`
	IsPublic      bool   `json:"is_public"`
	GenerateAudio bool   `json:"generate_audio"`
	VoiceType     string `json:"voice_type"`
	URL           string `json:"url"`
	FilePath      string `json:"file_path"`
	FileSize      int64  `json:"file_size"`
}

// generateCourseContent 异步生成课程内容
func (h *ContentHandler) generateCourseContent(course *models.Course, req CourseGenerationRequest) {
	// 构建Enhanced PPT生成请求
	enhancedReq := &services.EnhancedGenerationRequest{
		SourceType:    req.SourceType,
		Content:       req.Content,
		Title:         req.Title,
		SlideCount:    req.SlideCount,
		Template:      req.Template,
		Style:         req.Style,
		Audience:      req.Audience,
		Difficulty:    req.Difficulty,
		Language:      req.Language,
		UserType:      "regular", // TODO: 根据用户VIP状态设置
		UserID:        fmt.Sprintf("%d", course.UserID),
		IncludeImages: req.IncludeImages,
		IncludeNotes:  req.IncludeNotes,
		AutoOptimize:  req.AutoOptimize,
		GenerateAudio: req.GenerateAudio,
		VoiceType:     req.VoiceType,
	}

	// 调用Enhanced PPT服务生成内容
	result, err := h.enhancedPPTService.GeneratePPT(enhancedReq)
	if err != nil {
		// 更新课程状态为失败
		course.Status = "failed"
		course.ErrorMessage = fmt.Sprintf("PPT生成失败: %v", err)
		h.db.Save(course)
		fmt.Printf("课程 %d PPT生成失败: %v\n", course.ID, err)
		return
	}

	if !result.Success {
		// 更新课程状态为失败
		course.Status = "failed"
		course.ErrorMessage = result.Error
		h.db.Save(course)
		fmt.Printf("课程 %d PPT生成失败: %s\n", course.ID, result.Error)
		return
	}

	// ✅ 保存幻灯片到数据库（使用新的SlideCreationService）
	slideContents := make([]services.SlideCreationContent, 0, len(result.Slides))
	for _, slideContent := range result.Slides {
		// 转换为SlideCreationService所需的格式
		content := slideContent.MainContent
		if content == "" && len(slideContent.BulletPoints) > 0 {
			content = strings.Join(slideContent.BulletPoints, "\n")
		}

		slideContents = append(slideContents, services.SlideCreationContent{
			SlideNumber:  slideContent.SlideNumber,
			Title:        slideContent.Title,
			Content:      content,
			Keywords:     slideContent.KeyConcepts, // ✅ 传递关键概念
			Notes:        slideContent.SpeakerNotes,
			SlideType:    slideContent.SlideType,
			BulletPoints: slideContent.BulletPoints,
		})
	}

	// ✅ 使用SlideCreationService批量创建幻灯片（包含关键词处理）
	creationResult, err := h.slideService.BatchCreateSlides(course.ID, slideContents)
	if err != nil {
		fmt.Printf("❌ 批量创建幻灯片失败: %v\n", err)
		course.Status = "failed"
		course.ErrorMessage = fmt.Sprintf("幻灯片创建失败: %v", err)
		h.db.Save(course)
		return
	}

	// ✅ 记录创建结果
	fmt.Printf("📊 幻灯片创建完成: 总数=%d, 成功=%d, 失败=%d\n",
		creationResult.TotalSlides, creationResult.SuccessCount, creationResult.FailedCount)

	// 如果有失败的幻灯片，记录详细错误
	if creationResult.FailedCount > 0 {
		for _, errMsg := range creationResult.Errors {
			fmt.Printf("⚠️ %s\n", errMsg)
		}
	}

	// 更新课程信息
	course.Status = "completed"
	course.SlidesCount = creationResult.SuccessCount // ✅ 使用实际成功创建的幻灯片数量
	course.Duration = result.TotalAudioDuration
	if result.PPTFilePath != "" {
		course.PPTFilePath = result.PPTFilePath
	}

	// 保存更新
	if err := h.db.Save(course).Error; err != nil {
		fmt.Printf("更新课程 %d 状态失败: %v\n", course.ID, err)
	}

	fmt.Printf("✅ 课程 %d 内容生成完成，共 %d 张幻灯片（包含关键词）\n", course.ID, creationResult.SuccessCount)
}

// minInt 返回两个整数中的较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

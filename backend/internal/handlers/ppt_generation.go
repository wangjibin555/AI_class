package handlers

import (
	"ai-classroom/internal/services"
	"ai-classroom/pkg/database/crawler"
	"ai-classroom/pkg/parser"
	"ai-classroom/pkg/utils"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ai-classroom/internal/models"
)

// PPTGenerationHandler PPT生成处理器
type PPTGenerationHandler struct {
	pptService    *services.PPTGenerationService
	fileProcessor *parser.EnhancedFileProcessor
	db            *gorm.DB
}

// NewPPTGenerationHandler 创建PPT生成处理器
func NewPPTGenerationHandler(pptService *services.PPTGenerationService, db *gorm.DB) *PPTGenerationHandler {
	return &PPTGenerationHandler{
		pptService:    pptService,
		fileProcessor: parser.NewEnhancedFileProcessor(),
		db:            db,
	}
}

// GeneratePPTRequest 生成PPT请求
type GeneratePPTRequest struct {
	Content    string                    `json:"content" binding:"required"`
	Params     services.GenerationParams `json:"params"`
	SourceType string                    `json:"source_type"` // url, file, text
	SourceURL  string                    `json:"source_url,omitempty"`
	FileName   string                    `json:"file_name,omitempty"`
}

// GeneratePPTResponse 生成PPT响应
type GeneratePPTResponse struct {
	CourseID       int64                   `json:"course_id"`
	Slides         []services.SlideContent `json:"slides"`
	TotalSlides    int                     `json:"total_slides"`
	GenerationTime string                  `json:"generation_time"`
	Template       string                  `json:"template"`
	Status         string                  `json:"status"`
	Message        string                  `json:"message"`
}

// GeneratePPT 生成PPT课件
func (h *PPTGenerationHandler) GeneratePPT(c *gin.Context) {
	var req GeneratePPTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 验证内容长度
	if len(req.Content) < 50 {
		utils.ErrorResponse(c, http.StatusBadRequest, "内容长度不足", "内容至少需要50个字符")
		return
	}

	// 设置默认参数
	if req.Params.SlideCount == 0 {
		req.Params.SlideCount = 10
	}
	if req.Params.Template == "" {
		req.Params.Template = "business"
	}
	if req.Params.Audience == "" {
		req.Params.Audience = "general"
	}
	if req.Params.Difficulty == "" {
		req.Params.Difficulty = "intermediate"
	}

	// 生成PPT
	result, err := h.pptService.GenerateSlides(req.Content, req.Params)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成PPT失败", err.Error())
		return
	}

	// 创建课程记录
	fmt.Printf("🔍 正在调用createCourseRecord方法...\n")
	courseID, err := h.createCourseRecord(c, req, result)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存课程失败", err.Error())
		return
	}

	response := GeneratePPTResponse{
		CourseID:       courseID,
		Slides:         result.Slides,
		TotalSlides:    result.TotalSlides,
		GenerationTime: result.GenerationTime.String(),
		Template:       result.Template,
		Status:         "success",
		Message:        "PPT生成成功",
	}

	utils.SuccessResponse(c, "PPT生成成功", response)
}

// GetTemplates 获取可用模板
func (h *PPTGenerationHandler) GetTemplates(c *gin.Context) {
	templates := h.pptService.GetTemplates()
	utils.SuccessResponse(c, "获取模板成功", templates)
}

// GetGenerationStatus 获取生成状态
func (h *PPTGenerationHandler) GetGenerationStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少生成ID")
		return
	}

	// 这里应该从数据库或缓存中获取生成状态
	// 暂时返回模拟状态
	status := map[string]interface{}{
		"id":       id,
		"status":   "completed",
		"progress": 100,
		"message":  "PPT生成完成",
	}

	utils.SuccessResponse(c, "获取状态成功", status)
}

// DownloadPPT 下载PPT文件
func (h *PPTGenerationHandler) DownloadPPT(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少文件名")
		return
	}

	// 构建文件路径
	filePath := filepath.Join("ppt", filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.ErrorResponse(c, http.StatusNotFound, "PPT文件不存在")
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// 发送文件
	c.File(filePath)
}

// PreviewPPT 预览PPT文件
func (h *PPTGenerationHandler) PreviewPPT(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少文件名")
		return
	}

	// 构建文件路径
	filePath := filepath.Join("ppt", filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.ErrorResponse(c, http.StatusNotFound, "PPT文件不存在")
		return
	}

	// 设置响应头用于预览
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", "inline")

	// 发送文件
	c.File(filePath)
}

// GetPPTInfo 获取PPT信息（页数等）
func (h *PPTGenerationHandler) GetPPTInfo(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少文件名")
		return
	}

	// 构建文件路径
	filePath := filepath.Join("ppt", filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.ErrorResponse(c, http.StatusNotFound, "PPT文件不存在")
		return
	}

	// 读取HTML文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取PPT文件失败")
		return
	}

	// 解析HTML获取页数信息
	totalSlides := h.extractSlidesCount(string(content))

	utils.SuccessResponse(c, "获取PPT信息成功", map[string]interface{}{
		"filename":      filename,
		"total_slides":  totalSlides,
		"current_slide": 1,
	})
}

// extractSlidesCount 从HTML内容中提取幻灯片数量
func (h *PPTGenerationHandler) extractSlidesCount(htmlContent string) int {
	// 方法1: 查找幻灯片编号来确定总页数 "第 X 页" (最精确的方法)
	re := regexp.MustCompile(`第\s*(\d+)\s*页`)
	submatchAll := re.FindAllStringSubmatch(htmlContent, -1)
	maxPage := 0
	for _, submatch := range submatchAll {
		if len(submatch) > 1 {
			if page, err := strconv.Atoi(submatch[1]); err == nil && page > maxPage {
				maxPage = page
			}
		}
	}
	if maxPage > 0 {
		return maxPage
	}

	// 方法2: 查找精确的 <div class="slide "> 或 <div class="slide title-slide"> 格式
	re = regexp.MustCompile(`<div\s+class\s*=\s*['"']slide\s+[^'"]*['"']|<div\s+class\s*=\s*['"']slide['"']`)
	matches := re.FindAllString(htmlContent, -1)
	if len(matches) > 0 {
		return len(matches)
	}

	// 方法3: 查找section标签的数量（reveal.js格式）
	re = regexp.MustCompile(`<section[^>]*>`)
	matches = re.FindAllString(htmlContent, -1)
	if len(matches) > 0 {
		return len(matches)
	}

	// 方法4: 查找data-total-slides属性
	re = regexp.MustCompile(`data-total-slides\s*=\s*['"'](\d+)['"']`)
	match := re.FindStringSubmatch(htmlContent)
	if len(match) > 1 {
		if count, err := strconv.Atoi(match[1]); err == nil {
			return count
		}
	}

	// 默认返回1页
	return 1
}

// createCourseRecord 创建课程记录
func (h *PPTGenerationHandler) createCourseRecord(c *gin.Context, req GeneratePPTRequest, result *services.GenerationResult) (int64, error) {
	fmt.Printf("🔍 [DEBUG] createCourseRecord 方法开始执行\n")

	// 获取当前用户ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		fmt.Printf("🔍 [DEBUG] 用户未登录错误\n")
		return 0, utils.NewError("用户未登录")
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		fmt.Printf("🔍 [DEBUG] 用户ID格式错误\n")
		return 0, utils.NewError("用户ID格式错误")
	}

	fmt.Printf("🔍 [DEBUG] 获取到用户ID: %d\n", userID)

	// 开始事务
	tx := h.db.Begin()
	if tx.Error != nil {
		fmt.Printf("🔍 [DEBUG] 开始事务失败: %v\n", tx.Error)
		return 0, fmt.Errorf("开始事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("🔍 [DEBUG] 发生panic，回滚事务: %v\n", r)
			tx.Rollback()
		}
	}()

	// 从result中提取标题，如果没有则使用默认标题
	title := result.Title
	if title == "" {
		title = "AI生成的PPT课件"
	}

	fmt.Printf("🔍 [DEBUG] 课程标题: %s\n", title)

	// 创建课程记录
	course := &models.Course{
		UserID:        userID,
		Title:         title,
		Description:   fmt.Sprintf("基于%s生成的PPT课件", req.SourceType),
		Category:      models.CourseCategory("general"),
		SourceType:    models.SourceType(req.SourceType),
		SourceContent: req.Content,
		Status:        models.CourseStatusCompleted, // 直接标记为完成，因为PPT已生成
		SlidesCount:   result.TotalSlides,
		IsPublic:      false, // 默认私有
		VoiceType:     "zhixiaobai",
		PPTFilePath:   result.PPTFilePath, // 保存PPT文件路径
		GenerationParams: models.GenerationParams{
			"template":    req.Params.Template,
			"style":       req.Params.Style,
			"audience":    req.Params.Audience,
			"difficulty":  req.Params.Difficulty,
			"slide_count": req.Params.SlideCount,
		},
	}

	fmt.Printf("🔍 [DEBUG] 准备创建课程，幻灯片数: %d\n", result.TotalSlides)

	// 保存课程
	if err := tx.Create(course).Error; err != nil {
		fmt.Printf("🔍 [DEBUG] 创建课程失败: %v\n", err)
		tx.Rollback()
		return 0, fmt.Errorf("创建课程失败: %w", err)
	}

	fmt.Printf("🔍 [DEBUG] 课程创建成功，ID: %d\n", course.ID)

	// 验证课程ID是否正确生成
	if course.ID == 0 {
		fmt.Printf("🔍 [DEBUG] 课程ID生成失败\n")
		tx.Rollback()
		return 0, fmt.Errorf("课程ID生成失败")
	}

	// 保存幻灯片
	totalDuration := 0
	fmt.Printf("🔍 [DEBUG] 开始保存 %d 张幻灯片\n", len(result.Slides))
	for i, slideContent := range result.Slides {
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

		// 估算音频时长（基于内容长度，每100字符约6秒）
		contentLength := len(slide.Title + " " + slide.Content + " " + slide.SpeakerNotes)
		estimatedDuration := (contentLength / 100) * 6
		if estimatedDuration < 5 {
			estimatedDuration = 5 // 最少5秒
		}
		slide.Duration = estimatedDuration
		totalDuration += estimatedDuration

		if err := tx.Create(slide).Error; err != nil {
			fmt.Printf("🔍 [DEBUG] 保存幻灯片%d失败: %v\n", i+1, err)
			tx.Rollback()
			return 0, fmt.Errorf("保存幻灯片%d失败: %w", i+1, err)
		}
	}

	// 更新课程总时长
	if err := tx.Model(course).Update("duration", totalDuration).Error; err != nil {
		fmt.Printf("🔍 [DEBUG] 更新课程时长失败: %v\n", err)
		tx.Rollback()
		return 0, fmt.Errorf("更新课程时长失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("🔍 [DEBUG] 提交事务失败: %v\n", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("🔍 [DEBUG] createCourseRecord 成功完成，返回课程ID: %d\n", course.ID)
	return int64(course.ID), nil
}

// UploadDocument 上传文档
func (h *PPTGenerationHandler) UploadDocument(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件上传失败", err.Error())
		return
	}

	// 验证文件大小
	if !h.fileProcessor.ValidateFileSize(file.Size) {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件过大", "文件大小不能超过10MB")
		return
	}

	// 验证文件类型
	if !h.fileProcessor.ValidateFileType(file.Filename) {
		supportedTypes := h.fileProcessor.GetSupportedFileTypes()
		utils.ErrorResponse(c, http.StatusBadRequest, "文件类型不支持",
			fmt.Sprintf("只支持%v格式", supportedTypes))
		return
	}

	// 保存文件到临时目录
	tempDir := filepath.Join("storage", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建临时目录失败", err.Error())
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("%s_%s", utils.GenerateRandomString(12), file.Filename)
	filePath := filepath.Join(tempDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "文件保存失败", err.Error())
		return
	}

	// 处理文件内容
	fileContent, err := h.fileProcessor.ProcessFile(filePath)
	if err != nil {
		// 删除临时文件
		os.Remove(filePath)
		utils.ErrorResponse(c, http.StatusInternalServerError, "文件处理失败", err.Error())
		return
	}

	response := map[string]interface{}{
		"file_name": file.Filename,
		"file_size": file.Size,
		"file_type": fileContent.Type,
		"file_path": filePath,
		"title":     fileContent.Title,
		"content":   fileContent.Content,
		"pages":     fileContent.Pages,
		"processed": true,
		"message":   "文件上传并处理成功",
	}

	utils.SuccessResponse(c, "文件上传成功", response)
}

// ValidateURL 验证URL
func (h *PPTGenerationHandler) ValidateURL(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 使用URL验证器进行详细验证
	validator := crawler.NewURLValidator()
	validationResult := validator.ValidateURL(req.URL)

	if !validationResult.Valid {
		utils.ErrorResponse(c, http.StatusBadRequest, "URL验证失败", validationResult.Reason)
		return
	}

	// 尝试访问URL获取基本信息
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(req.URL)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "URL访问失败", "无法访问该URL，请检查网络连接或URL是否正确")
		return
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		utils.ErrorResponse(c, http.StatusBadRequest, "URL访问失败",
			fmt.Sprintf("HTTP状态码: %d", resp.StatusCode))
		return
	}

	// 读取页面内容的前1KB来获取标题
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	content := string(body[:n])

	// 提取标题
	title := extractTitleFromHTML(content)
	if title == "" {
		title = "未知标题"
	}

	// 获取平台信息
	platformName := validationResult.Platform.String()
	if platformName == "不支持" {
		platformName = "通用网页"
	}

	response := map[string]interface{}{
		"url":          req.URL,
		"title":        title,
		"description":  fmt.Sprintf("来自%s的内容", platformName),
		"platform":     platformName,
		"valid":        true,
		"message":      "URL验证成功",
		"status_code":  resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
	}

	utils.SuccessResponse(c, "URL验证成功", response)
}

// extractTitleFromHTML 从HTML内容中提取标题
func extractTitleFromHTML(content string) string {
	// 匹配<title>标签
	titleRegex := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	matches := titleRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// 匹配h1标签
	h1Regex := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	matches = h1Regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

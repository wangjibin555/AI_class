package services

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	"ai-classroom/internal/models"
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/database/crawler"
	"ai-classroom/pkg/parser"

	"encoding/json"

	"gorm.io/gorm"
)

// ContentService 内容处理服务
type ContentService struct {
	db           *gorm.DB
	crawler      *crawler.WebCrawler   // 原有的CSS选择器爬虫
	xpathCrawler *crawler.XPathCrawler // 新的XPath爬虫
	parser       parser.DocumentParser
	aiClient     *ai.DashScopeClient
	useXPath     bool // 是否使用XPath爬虫
}

// ContentRequest 内容处理请求
type ContentRequest struct {
	// 输入类型
	Type string `json:"type" binding:"required"` // url, text, file

	// URL输入
	URL string `json:"url,omitempty"`

	// 文本输入
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`

	// 课件设置
	CourseSettings *CourseSettings `json:"course_settings,omitempty"`
}

// CourseSettings 课件设置
type CourseSettings struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Difficulty  string   `json:"difficulty"`
	Tags        []string `json:"tags"`
	IsPublic    bool     `json:"is_public"`
}

// ContentResult 内容处理结果
type ContentResult struct {
	// 基础信息
	ID        string `json:"id"`
	Type      string `json:"type"`
	SourceURL string `json:"source_url,omitempty"`

	// 内容信息
	Title       string `json:"title"`
	Content     string `json:"content"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`

	// 元数据
	WordCount     int      `json:"word_count"`
	ReadTime      int      `json:"read_time"`
	Language      string   `json:"language"`
	Keywords      []string `json:"keywords,omitempty"`
	PublishedTime string   `json:"published_time,omitempty"`

	// 媒体资源
	Images []crawler.ImageInfo `json:"images,omitempty"`
	Videos []crawler.VideoInfo `json:"videos,omitempty"`

	// 处理信息
	ProcessedAt time.Time `json:"processed_at"`
	ProcessTime int64     `json:"process_time_ms"`
	Platform    string    `json:"platform,omitempty"`

	// 状态信息
	Status   string   `json:"status"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ProcessStats 处理统计
type ProcessStats struct {
	TotalRequests   int     `json:"total_requests"`
	SuccessRequests int     `json:"success_requests"`
	FailedRequests  int     `json:"failed_requests"`
	SuccessRate     float64 `json:"success_rate"`
	AvgProcessTime  int64   `json:"avg_process_time_ms"`
}

// NewContentService 创建内容服务
func NewContentService(db *gorm.DB, aiClient *ai.DashScopeClient) *ContentService {
	// 创建原有爬虫配置
	crawlerConfig := crawler.DefaultCrawlerConfig()
	crawlerConfig.DelayMin = 1 * time.Second
	crawlerConfig.DelayMax = 2 * time.Second
	crawlerConfig.RandomDelay = true
	crawlerConfig.EnableDebug = false

	// 创建原有爬虫实例
	webCrawler := crawler.NewWebCrawler(crawlerConfig)

	// 创建XPath爬虫配置
	xpathConfig := crawler.DefaultXPathCrawlerConfig()
	xpathConfig.EnableDebug = true // 启用调试输出
	xpathConfig.RetryCount = 3
	xpathConfig.RetryDelay = 2 * time.Second

	// 创建XPath爬虫实例
	xpathCrawler := crawler.NewXPathCrawler(xpathConfig)

	// 创建文档解析器
	documentParser := parser.NewDocumentParser()

	return &ContentService{
		db:           db,
		crawler:      webCrawler,
		xpathCrawler: xpathCrawler,
		parser:       documentParser,
		aiClient:     aiClient,
		useXPath:     true, // 默认使用XPath爬虫
	}
}

// ProcessContent 处理内容
func (s *ContentService) ProcessContent(req *ContentRequest) (*ContentResult, error) {
	startTime := time.Now()

	// 验证请求
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// 根据类型处理内容
	var result *ContentResult
	var err error

	switch req.Type {
	case "url":
		result, err = s.processURLContent(req)
	case "text":
		result, err = s.processTextContent(req)
	case "file":
		return nil, errors.New("文件处理功能尚未实现，请使用文档解析服务")
	default:
		return nil, fmt.Errorf("不支持的内容类型: %s", req.Type)
	}

	if err != nil {
		return nil, err
	}

	// 计算处理时间
	processTime := time.Since(startTime).Milliseconds()
	result.ProcessTime = processTime
	result.ProcessedAt = time.Now()

	// 保存到数据库（可选）
	if err := s.saveContentResult(result); err != nil {
		// 记录日志，但不影响返回结果
		fmt.Printf("保存内容结果失败: %v\n", err)
	}

	return result, nil
}

// ProcessDocumentFile 处理文档文件
func (s *ContentService) ProcessDocumentFile(file *multipart.FileHeader) (*ContentResult, error) {
	startTime := time.Now()

	// 验证文件
	if err := s.parser.ValidateFile(file); err != nil {
		return &ContentResult{
			Type:        "file",
			Status:      "failed",
			Error:       err.Error(),
			ProcessTime: time.Since(startTime).Milliseconds(),
		}, err
	}

	// 解析文档
	parseResult, err := s.parser.ParseFile(file)
	if err != nil {
		return &ContentResult{
			Type:        "file",
			Status:      "failed",
			Error:       err.Error(),
			ProcessTime: time.Since(startTime).Milliseconds(),
		}, err
	}

	// 转换为内容结果
	result := &ContentResult{
		ID:          generateContentID(),
		Type:        "file",
		Title:       parseResult.Title,
		Content:     parseResult.Content,
		Author:      parseResult.Author,
		WordCount:   parseResult.WordCount,
		ReadTime:    calculateReadTime(parseResult.WordCount),
		Language:    parseResult.Language,
		Keywords:    parseResult.Keywords,
		Platform:    fmt.Sprintf("文档上传 (%s)", parseResult.FileType),
		Status:      parseResult.Status,
		ProcessedAt: time.Now(),
		ProcessTime: time.Since(startTime).Milliseconds(),
	}

	// 如果解析失败，设置错误信息
	if parseResult.Status == "failed" {
		result.Error = parseResult.ErrorMessage
	}

	// 保存到数据库（可选）
	if err := s.saveContentResult(result); err != nil {
		// 记录日志，但不影响返回结果
		fmt.Printf("保存内容结果失败: %v\n", err)
	}

	return result, nil
}

// processURLContent 处理URL内容
func (s *ContentService) processURLContent(req *ContentRequest) (*ContentResult, error) {
	fmt.Printf("\n🌐 内容服务处理URL调试开始\n")
	fmt.Printf("🎯 请求URL: %s\n", req.URL)
	fmt.Printf("📋 内容类型: %s\n", req.Type)
	fmt.Printf("🔧 爬虫类型: %s\n", map[bool]string{true: "XPath爬虫", false: "CSS选择器爬虫"}[s.useXPath])

	var result *ContentResult

	if s.useXPath {
		// 使用XPath爬虫抓取内容
		fmt.Printf("\n🕷️ 调用XPath爬虫服务\n")
		xpathResult, xpathErr := s.xpathCrawler.CrawlURL(req.URL)
		if xpathErr != nil {
			fmt.Printf("❌ XPath爬虫抓取失败: %v\n", xpathErr)
			return &ContentResult{
				Type:      "url",
				SourceURL: req.URL,
				Status:    "failed",
				Error:     xpathErr.Error(),
			}, xpathErr
		}

		// 检查XPath爬取是否成功
		fmt.Printf("\n📊 检查XPath爬取结果\n")
		if !xpathResult.Success {
			fmt.Printf("❌ XPath爬取失败: %s\n", xpathResult.Error)
			return &ContentResult{
				Type:      "url",
				SourceURL: req.URL,
				Status:    "failed",
				Error:     xpathResult.Error,
			}, errors.New(xpathResult.Error)
		}
		fmt.Printf("✅ XPath爬取成功\n")

		// 转换XPath结果格式
		fmt.Printf("\n🔄 转换XPath结果格式\n")
		result = &ContentResult{
			ID:            generateContentID(),
			Type:          "url",
			SourceURL:     req.URL,
			Title:         xpathResult.Title,
			Content:       xpathResult.Content,
			Author:        xpathResult.Author,
			Description:   xpathResult.Description,
			WordCount:     xpathResult.WordCount,
			ReadTime:      calculateReadTime(xpathResult.WordCount),
			Language:      xpathResult.Language,
			Keywords:      xpathResult.Keywords,
			PublishedTime: xpathResult.PublishedTime,
			Images:        xpathResult.Images,
			Videos:        xpathResult.Videos,
			Platform:      xpathResult.Platform,
			Status:        "success",
		}

	} else {
		// 使用原有CSS选择器爬虫抓取内容
		fmt.Printf("\n🕷️ 调用CSS选择器爬虫服务\n")
		crawlResult, crawlErr := s.crawler.CrawlURL(req.URL)
		if crawlErr != nil {
			fmt.Printf("❌ CSS爬虫抓取失败: %v\n", crawlErr)
			return &ContentResult{
				Type:      "url",
				SourceURL: req.URL,
				Status:    "failed",
				Error:     crawlErr.Error(),
			}, crawlErr
		}

		// 检查爬取是否成功
		fmt.Printf("\n📊 检查爬取结果\n")
		if !crawlResult.Success {
			fmt.Printf("❌ 爬取失败: %s\n", crawlResult.Error)
			return &ContentResult{
				Type:      "url",
				SourceURL: req.URL,
				Status:    "failed",
				Error:     crawlResult.Error,
			}, errors.New(crawlResult.Error)
		}
		fmt.Printf("✅ 爬取成功\n")

		// 转换为结果格式
		fmt.Printf("\n🔄 转换结果格式\n")
		result = &ContentResult{
			ID:            generateContentID(),
			Type:          "url",
			SourceURL:     req.URL,
			Title:         crawlResult.Title,
			Content:       crawlResult.Content,
			Author:        crawlResult.Author,
			Description:   crawlResult.Description,
			WordCount:     crawlResult.WordCount,
			ReadTime:      calculateReadTime(crawlResult.WordCount),
			Language:      crawlResult.Language,
			Keywords:      crawlResult.Keywords,
			PublishedTime: crawlResult.PublishedTime,
			Images:        crawlResult.Images,
			Videos:        crawlResult.Videos,
			Platform:      s.getPlatformName(req.URL),
			Status:        "success",
		}
	}

	fmt.Printf("✅ 结果格式转换完成\n")
	fmt.Printf("📊 ContentResult详情:\n")
	fmt.Printf("   ID: %s\n", result.ID)
	fmt.Printf("   标题: %s\n", result.Title)
	fmt.Printf("   作者: %s\n", result.Author)
	fmt.Printf("   字数: %d\n", result.WordCount)
	fmt.Printf("   阅读时间: %d 分钟\n", result.ReadTime)
	fmt.Printf("   关键词: %v\n", result.Keywords)
	fmt.Printf("   平台: %s\n", result.Platform)

	// 添加警告信息
	fmt.Printf("\n⚠️ 生成警告信息\n")
	result.Warnings = s.generateWarningsFromResult(result, req)
	if len(result.Warnings) > 0 {
		fmt.Printf("⚠️ 警告信息: %v\n", result.Warnings)
	} else {
		fmt.Printf("✅ 无警告信息\n")
	}

	// 应用课件设置
	if req.CourseSettings != nil {
		fmt.Printf("\n⚙️ 应用课件设置\n")
		s.applyCourseSettings(result, req.CourseSettings)
		fmt.Printf("✅ 课件设置应用完成\n")
	} else {
		fmt.Printf("\n📋 未提供课件设置，跳过\n")
	}

	fmt.Printf("\n🎉 URL内容处理完成!\n")
	fmt.Printf("═══════════════════════════════════════\n")
	return result, nil
}

// processTextContent 处理文本内容
func (s *ContentService) processTextContent(req *ContentRequest) (*ContentResult, error) {
	if req.Content == "" {
		return nil, errors.New("文本内容不能为空")
	}

	// 处理文本内容
	content := strings.TrimSpace(req.Content)
	title := req.Title
	if title == "" {
		title = extractTitleFromContent(content)
	}

	wordCount := len([]rune(content))

	result := &ContentResult{
		ID:        generateContentID(),
		Type:      "text",
		Title:     title,
		Content:   content,
		WordCount: wordCount,
		ReadTime:  calculateReadTime(wordCount),
		Language:  detectLanguage(content),
		Platform:  "手动输入",
		Status:    "success",
	}

	// 生成关键词
	result.Keywords = extractKeywords(content)

	// 生成描述
	if req.CourseSettings == nil || req.CourseSettings.Description == "" {
		result.Description = generateDescription(content)
	}

	// 应用课件设置
	if req.CourseSettings != nil {
		s.applyCourseSettings(result, req.CourseSettings)
	}

	return result, nil
}

// validateRequest 验证请求
func (s *ContentService) validateRequest(req *ContentRequest) error {
	if req == nil {
		return errors.New("请求不能为空")
	}

	switch req.Type {
	case "url":
		if req.URL == "" {
			return errors.New("URL不能为空")
		}

		// 验证URL格式
		if _, err := url.Parse(req.URL); err != nil {
			return fmt.Errorf("URL格式无效: %v", err)
		}

		// 检查URL是否支持
		validator := crawler.NewURLValidator()
		if !validator.IsValidURL(req.URL) {
			return errors.New("不支持的URL格式或网站")
		}

	case "text":
		if req.Content == "" {
			return errors.New("文本内容不能为空")
		}

		if len([]rune(req.Content)) < 10 {
			return errors.New("文本内容过短，至少需要10个字符")
		}

		if len([]rune(req.Content)) > 100000 {
			return errors.New("文本内容过长，最多支持10万字符")
		}

	case "file":
		return errors.New("文件处理功能尚未实现")

	default:
		return fmt.Errorf("不支持的内容类型: %s", req.Type)
	}

	// 验证课件设置
	if req.CourseSettings != nil {
		if err := s.validateCourseSettings(req.CourseSettings); err != nil {
			return err
		}
	}

	return nil
}

// validateCourseSettings 验证课件设置
func (s *ContentService) validateCourseSettings(settings *CourseSettings) error {
	if settings.Title != "" && len([]rune(settings.Title)) > 100 {
		return errors.New("课件标题过长，最多100个字符")
	}

	if settings.Description != "" && len([]rune(settings.Description)) > 500 {
		return errors.New("课件描述过长，最多500个字符")
	}

	// 验证难度级别
	validDifficulties := []string{"入门", "初级", "中级", "高级", "专家"}
	if settings.Difficulty != "" {
		valid := false
		for _, diff := range validDifficulties {
			if settings.Difficulty == diff {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("无效的难度级别: %s", settings.Difficulty)
		}
	}

	return nil
}

// applyCourseSettings 应用课件设置
func (s *ContentService) applyCourseSettings(result *ContentResult, settings *CourseSettings) {
	if settings.Title != "" {
		result.Title = settings.Title
	}

	if settings.Description != "" {
		result.Description = settings.Description
	}

	// 合并标签和关键词
	if len(settings.Tags) > 0 {
		// 去重合并
		keywordMap := make(map[string]bool)
		for _, keyword := range result.Keywords {
			keywordMap[keyword] = true
		}
		for _, tag := range settings.Tags {
			keywordMap[tag] = true
		}

		keywords := make([]string, 0, len(keywordMap))
		for keyword := range keywordMap {
			keywords = append(keywords, keyword)
		}
		result.Keywords = keywords
	}
}

// getPlatformName 获取平台名称
func (s *ContentService) getPlatformName(urlStr string) string {
	validator := crawler.NewURLValidator()
	platform := validator.DetectPlatform(urlStr)
	return platform.String()
}

// generateWarnings 生成警告信息
func (s *ContentService) generateWarnings(crawlResult *crawler.CrawlResult, req *ContentRequest) []string {
	var warnings []string

	// 检查内容长度
	if crawlResult.WordCount < 100 {
		warnings = append(warnings, "内容较短，可能不适合生成完整的课件")
	}

	if crawlResult.WordCount > 50000 {
		warnings = append(warnings, "内容较长，生成课件可能需要较长时间")
	}

	// 检查媒体资源
	if len(crawlResult.Images) == 0 {
		warnings = append(warnings, "未检测到图片，生成的课件可能缺少视觉元素")
	}

	// 检查平台特定警告
	validator := crawler.NewURLValidator()
	platform := validator.DetectPlatform(req.URL)
	switch platform {
	case crawler.PlatformWechat:
		warnings = append(warnings, "微信公众号文章可能包含受版权保护的内容")
	case crawler.PlatformGithub:
		warnings = append(warnings, "GitHub页面可能包含大量代码，适合技术类课件")
	}

	return warnings
}

// saveContentResult 保存内容结果
func (s *ContentService) saveContentResult(result *ContentResult) error {
	// 这里应该保存到数据库
	// 由于当前没有对应的数据模型，暂时跳过
	// TODO: 实现内容结果的数据库保存
	return nil
}

// GetSupportedPlatforms 获取支持的平台列表
func (s *ContentService) GetSupportedPlatforms() []*crawler.PlatformInfo {
	validator := crawler.NewURLValidator()
	return validator.GetSupportedPlatforms()
}

// ValidateURL 验证URL
func (s *ContentService) ValidateURL(url string) *crawler.ValidationResult {
	validator := crawler.NewURLValidator()
	return validator.ValidateURL(url)
}

// GetProcessStats 获取处理统计
func (s *ContentService) GetProcessStats() *ProcessStats {
	// TODO: 实现统计功能
	return &ProcessStats{
		TotalRequests:   0,
		SuccessRequests: 0,
		FailedRequests:  0,
		SuccessRate:     0,
		AvgProcessTime:  0,
	}
}

// GetSupportedFileFormats 获取支持的文件格式
func (s *ContentService) GetSupportedFileFormats() []string {
	return s.parser.GetSupportedFormats()
}

// AI助手对话
func (s *ContentService) AIAssistantChat(userID uint, message, context string) (string, error) {
	if s.aiClient == nil {
		// 如果没有AI客户端，使用模拟响应
		prompt := s.buildAIAssistantPrompt(message, context)
		response := s.generateAIResponse(prompt)

		// 保存对话记录
		chatRecord := &models.AIAssistantChat{
			UserID:   userID,
			Message:  message,
			Response: response,
			Context:  context,
		}

		if err := s.db.Create(chatRecord).Error; err != nil {
			return "", fmt.Errorf("保存对话记录失败: %v", err)
		}

		return response, nil
	}

	// 构建消息历史
	var messages []ai.Message

	// 添加系统消息
	messages = append(messages, ai.Message{
		Role:    "system",
		Content: "你是一个专业的AI教学助手，专门帮助用户解答学习相关的问题。请用友好、专业的语气回答问题。",
	})

	// 如果有上下文，添加到消息中
	if context != "" {
		messages = append(messages, ai.Message{
			Role:    "assistant",
			Content: context,
		})
	}

	// 添加用户消息
	messages = append(messages, ai.Message{
		Role:    "user",
		Content: message,
	})

	// 调用AI
	response, err := s.aiClient.ChatCompletion(messages)
	if err != nil {
		return "", fmt.Errorf("AI chat failed: %v", err)
	}

	// 保存聊天记录
	chatRecord := &models.AIAssistantChat{
		UserID:    userID,
		Message:   message,
		Response:  response.Output.Text,
		Context:   context,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(chatRecord).Error; err != nil {
		// 记录日志但不影响返回结果
		log.Printf("Failed to save chat record: %v", err)
	}

	return response.Output.Text, nil
}

// 获取AI助手对话历史
func (s *ContentService) GetAIAssistantHistory(userID uint, page, limit int) ([]models.AIAssistantChat, int64, error) {
	var chats []models.AIAssistantChat
	var total int64

	offset := (page - 1) * limit

	// 获取总数
	if err := s.db.Model(&models.AIAssistantChat{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取对话总数失败: %v", err)
	}

	// 获取分页数据
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&chats).Error; err != nil {
		return nil, 0, fmt.Errorf("获取对话历史失败: %v", err)
	}

	return chats, total, nil
}

// 清除AI助手对话历史
func (s *ContentService) ClearAIAssistantHistory(userID uint) error {
	if err := s.db.Where("user_id = ?", userID).Delete(&models.AIAssistantChat{}).Error; err != nil {
		return fmt.Errorf("清除对话历史失败: %v", err)
	}
	return nil
}

// 构建AI助手的prompt
func (s *ContentService) buildAIAssistantPrompt(message, context string) string {
	prompt := "你是AI课堂的智能助手，专门帮助用户解答学习相关的问题。请用友好、专业的语调回答用户的问题。\n\n"

	if context != "" {
		prompt += fmt.Sprintf("上下文信息：%s\n\n", context)
	}

	prompt += fmt.Sprintf("用户问题：%s\n\n请提供详细、准确的回答。", message)

	return prompt
}

// 生成AI响应（模拟实现）
func (s *ContentService) generateAIResponse(prompt string) string {
	// 这里是模拟的AI响应，实际应该调用真实的AI服务
	responses := []string{
		"感谢您的问题！我是AI课堂的智能助手，很高兴为您服务。",
		"这是一个很好的问题。让我为您详细解答。",
		"根据您的问题，我建议您可以从以下几个方面来理解：",
		"我理解您的疑问，这里有一些相关的学习建议：",
		"基于您的问题，我可以为您提供以下帮助：",
	}

	// 简单的随机选择（实际应该根据prompt内容生成）
	return responses[len(prompt)%len(responses)]
}

// 辅助函数

// generateContentID 生成内容ID
func generateContentID() string {
	return fmt.Sprintf("content_%d", time.Now().UnixNano())
}

// calculateReadTime 计算阅读时间（分钟）
func calculateReadTime(wordCount int) int {
	// 假设中文阅读速度是每分钟300字
	const wordsPerMinute = 300
	readTime := wordCount / wordsPerMinute
	if readTime < 1 {
		readTime = 1
	}
	return readTime
}

// extractTitleFromContent 从内容中提取标题
func extractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len([]rune(line)) <= 100 {
			return line
		}
	}

	// 如果没有找到合适的标题，取前50个字符
	runes := []rune(content)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return string(runes)
}

// detectLanguage 检测语言
func detectLanguage(content string) string {
	// 简单的语言检测
	chineseCount := 0
	totalChars := 0

	for _, r := range content {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		}
		if r > 32 { // 排除空白字符
			totalChars++
		}
	}

	if totalChars > 0 && float64(chineseCount)/float64(totalChars) > 0.3 {
		return "zh-CN"
	}

	return "en"
}

// extractKeywords 提取关键词
func extractKeywords(content string) []string {
	// 简单的关键词提取
	// TODO: 实现更智能的关键词提取算法

	words := strings.Fields(content)
	keywordMap := make(map[string]int)

	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 && len(word) < 20 {
			keywordMap[word]++
		}
	}

	// 取出现频率最高的前10个词作为关键词
	type wordCount struct {
		word  string
		count int
	}

	var wordCounts []wordCount
	for word, count := range keywordMap {
		if count > 1 { // 至少出现2次
			wordCounts = append(wordCounts, wordCount{word, count})
		}
	}

	// 简单排序（这里可以优化）
	keywords := make([]string, 0, 10)
	for len(keywords) < 10 && len(wordCounts) > 0 {
		maxIndex := 0
		for i, wc := range wordCounts {
			if wc.count > wordCounts[maxIndex].count {
				maxIndex = i
			}
		}

		keywords = append(keywords, wordCounts[maxIndex].word)

		// 移除已选择的词
		wordCounts = append(wordCounts[:maxIndex], wordCounts[maxIndex+1:]...)
	}

	return keywords
}

// generateDescription 生成描述
func generateDescription(content string) string {
	// 取内容的前200个字符作为描述
	runes := []rune(content)
	if len(runes) > 200 {
		return string(runes[:200]) + "..."
	}
	return string(runes)
}

// PPTGenerationResult PPT生成结果
type PPTGenerationResult struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	Difficulty        string `json:"difficulty"`
	EstimatedDuration int    `json:"estimated_duration"`
	Slides            []struct {
		Number       int    `json:"number"`
		Title        string `json:"title"`
		Content      string `json:"content"`
		SpeakerNotes string `json:"speaker_notes"`
		Layout       string `json:"layout"`
	} `json:"slides"`
}

// ContentAnalysis 内容分析结果
type ContentAnalysis struct {
	Topic              string   `json:"topic"`
	Field              string   `json:"field"`
	Difficulty         string   `json:"difficulty"`
	KeyPoints          []string `json:"key_points"`
	SuggestedSlides    int      `json:"suggested_slides"`
	TargetAudience     string   `json:"target_audience"`
	LearningObjectives []string `json:"learning_objectives"`
}

// GenerateCourse 生成课程
func (s *ContentService) GenerateCourse(content string, userID uint) (*models.Course, error) {
	// 1. 分析内容，确定幻灯片数量
	slideCount, err := s.analyzeContentAndSuggestSlides(content)
	if err != nil {
		return nil, fmt.Errorf("content analysis failed: %v", err)
	}

	// 2. 生成PPT内容
	pptResult, err := s.generatePPTContent(content, slideCount)
	if err != nil {
		return nil, fmt.Errorf("PPT generation failed: %v", err)
	}

	// 3. 创建课程记录
	course := &models.Course{
		UserID:        userID,
		Title:         pptResult.Title,
		Description:   pptResult.Description,
		SourceType:    "text",
		SourceContent: content,
		Status:        "generating",
		SlidesCount:   len(pptResult.Slides),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 4. 保存课程和幻灯片
	if err := s.saveCourseAndSlides(course, pptResult); err != nil {
		return nil, fmt.Errorf("save course failed: %v", err)
	}

	return course, nil
}

// analyzeContentAndSuggestSlides 分析内容并建议幻灯片数量
func (s *ContentService) analyzeContentAndSuggestSlides(content string) (int, error) {
	if s.aiClient == nil {
		// 如果没有AI客户端，使用默认算法
		wordCount := len(strings.Fields(content))
		return max(5, min(20, wordCount/100)), nil
	}

	response, err := s.aiClient.AnalyzeContent(content)
	if err != nil {
		// AI分析失败，使用默认算法
		wordCount := len(strings.Fields(content))
		return max(5, min(20, wordCount/100)), nil
	}

	// 解析分析结果
	var analysis ContentAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		// 如果JSON解析失败，使用默认算法
		wordCount := len(strings.Fields(content))
		return max(5, min(20, wordCount/100)), nil
	}

	return max(5, min(20, analysis.SuggestedSlides)), nil
}

// generatePPTContent 生成PPT内容
func (s *ContentService) generatePPTContent(content string, slideCount int) (*PPTGenerationResult, error) {
	if s.aiClient == nil {
		return nil, fmt.Errorf("AI client not available")
	}

	var result *PPTGenerationResult
	var lastError error

	// 最多重试3次，每次都可能调整prompt
	for i := 0; i < 3; i++ {
		response, err := s.aiClient.GeneratePPT(content, slideCount)
		if err != nil {
			lastError = err
			continue
		}

		// 清理响应内容，移除可能的markdown格式
		cleanResponse := s.cleanAIResponse(response)

		// 尝试解析JSON
		err = json.Unmarshal([]byte(cleanResponse), &result)
		if err == nil && s.validatePPTResult(result) {
			return result, nil
		}

		lastError = err
	}

	return nil, fmt.Errorf("failed to generate valid PPT after retries: %v", lastError)
}

// cleanAIResponse 清理AI响应
func (s *ContentService) cleanAIResponse(response string) string {
	// 移除markdown代码块标记
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}

// validatePPTResult 验证PPT结果
func (s *ContentService) validatePPTResult(result *PPTGenerationResult) bool {
	if result.Title == "" || len(result.Slides) == 0 {
		return false
	}

	for _, slide := range result.Slides {
		if slide.Title == "" || slide.Content == "" {
			return false
		}
	}

	return true
}

// saveCourseAndSlides 保存课程和幻灯片
func (s *ContentService) saveCourseAndSlides(course *models.Course, pptResult *PPTGenerationResult) error {
	// 开始事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 保存课程
	if err := tx.Create(course).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 保存幻灯片
	for _, slideData := range pptResult.Slides {
		slide := &models.Slide{
			CourseID:     course.ID,
			SlideNumber:  slideData.Number,
			Title:        slideData.Title,
			Content:      slideData.Content,
			SpeakerNotes: slideData.SpeakerNotes,
			LayoutType:   models.LayoutType(slideData.Layout),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := tx.Create(slide).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 更新课程状态为已完成
	if err := tx.Model(course).Update("status", "completed").Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// 辅助函数
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateWarningsFromResult 从ContentResult生成警告信息
func (s *ContentService) generateWarningsFromResult(result *ContentResult, req *ContentRequest) []string {
	var warnings []string

	// 检查内容完整性
	if result.Title == "" {
		warnings = append(warnings, "未能提取到页面标题")
	}
	if result.Content == "" {
		warnings = append(warnings, "未能提取到页面内容")
	}
	if result.WordCount < 100 {
		warnings = append(warnings, "提取的内容过短，可能不完整")
	}
	if len(result.Keywords) == 0 {
		warnings = append(warnings, "未能提取到关键词")
	}

	// 检查媒体资源
	if len(result.Images) == 0 {
		warnings = append(warnings, "页面中未找到图片资源")
	}

	// 检查元数据
	if result.Author == "" {
		warnings = append(warnings, "未能识别文章作者")
	}
	if result.PublishedTime == "" {
		warnings = append(warnings, "未能获取发布时间")
	}

	return warnings
}

// SetUseXPath 设置是否使用XPath爬虫
func (s *ContentService) SetUseXPath(useXPath bool) {
	s.useXPath = useXPath
}

// GetCrawlerType 获取当前使用的爬虫类型
func (s *ContentService) GetCrawlerType() string {
	if s.useXPath {
		return "XPath爬虫"
	}
	return "CSS选择器爬虫"
}

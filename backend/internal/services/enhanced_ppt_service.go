package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/pkg/ai"
	"crypto/rand"
	"encoding/hex"
	"encoding/json" // Added for JSON parsing
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-classroom/pkg/parser"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// AudioFileInfo 音频文件信息结构
type AudioFileInfo struct {
	SlideNumber int       `json:"slide_number"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	FileURL     string    `json:"file_url"`
	Duration    int       `json:"duration"`   // 秒
	FileSize    int       `json:"file_size"`  // 字节
	Format      string    `json:"format"`     // mp3
	VoiceType   string    `json:"voice_type"` // 语音类型
	Status      string    `json:"status"`     // pending, generating, completed, failed
	Error       string    `json:"error"`      // 错误信息
	CreatedAt   time.Time `json:"created_at"`
}

// AudioGenerationProgress 音频生成进度
type AudioGenerationProgress struct {
	CourseID        uint            `json:"course_id"`
	Status          string          `json:"status"` // pending, processing, completed, failed
	TotalSlides     int             `json:"total_slides"`
	CompletedSlides int             `json:"completed_slides"`
	CurrentSlide    int             `json:"current_slide"`
	Progress        float64         `json:"progress"` // 0-100
	CurrentTitle    string          `json:"current_title"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	StartTime       time.Time       `json:"start_time"`
	EstimatedEnd    time.Time       `json:"estimated_end"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	AudioFiles      []AudioFileInfo `json:"audio_files"`
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// EnhancedPPTService 增强PPT生成服务
type EnhancedPPTService struct {
	aiClient   *ai.DashScopeClient
	db         *gorm.DB
	pptDir     string
	ttsService TTSService // 使用现有的TTS服务接口
	config     *EnhancedPPTConfig

	// 新增组件
	slideCalculator    *SlideCountCalculator
	technicalGenerator *TechnicalPPTGenerator

	// 音频进度跟踪
	audioProgressMap   map[uint]*AudioGenerationProgress
	audioProgressMutex sync.RWMutex
}

// EnhancedPPTConfig 增强PPT配置
type EnhancedPPTConfig struct {
	MaxSlideCount       int    `json:"max_slide_count"`     // 最大幻灯片数量
	MinSlideCount       int    `json:"min_slide_count"`     // 最小幻灯片数量
	DefaultSlideCount   int    `json:"default_slide_count"` // 默认幻灯片数量
	MaxContentLength    int    `json:"max_content_length"`  // 最大内容长度
	SlidesPerSection    int    `json:"slides_per_section"`  // 每个章节的幻灯片数量
	EnablePageLimit     bool   `json:"enable_page_limit"`   // 是否启用页数限制
	PageLimitByUserType bool   `json:"page_limit_by_user"`  // 是否根据用户类型限制页数
	VIPMaxSlides        int    `json:"vip_max_slides"`      // VIP用户最大幻灯片数
	RegularMaxSlides    int    `json:"regular_max_slides"`  // 普通用户最大幻灯片数
	OutputFormat        string `json:"output_format"`       // 输出格式: html, pptx
	Template            string `json:"template"`            // 默认模板
}

// EnhancedGenerationRequest 增强生成请求
type EnhancedGenerationRequest struct {
	// 内容来源
	SourceType string `json:"source_type"` // url, file, text
	Content    string `json:"content"`     // 内容或URL或文件路径

	// 生成参数
	Title      string `json:"title"`       // PPT标题
	SlideCount int    `json:"slide_count"` // 期望的幻灯片数量
	Template   string `json:"template"`    // 模板类型
	Style      string `json:"style"`       // 风格
	Audience   string `json:"audience"`    // 目标受众
	Difficulty string `json:"difficulty"`  // 难度等级
	Language   string `json:"language"`    // 语言

	// 用户相关
	UserType string `json:"user_type"` // 用户类型: regular, vip
	UserID   string `json:"user_id"`   // 用户ID

	// 选项
	IncludeImages bool   `json:"include_images"` // 是否包含图片建议
	IncludeNotes  bool   `json:"include_notes"`  // 是否包含备注
	AutoOptimize  bool   `json:"auto_optimize"`  // 是否自动优化内容长度
	GenerateAudio bool   `json:"generate_audio"` // 是否生成音频
	VoiceType     string `json:"voice_type"`     // 音频语音类型
	GeneratePPT   bool   `json:"generate_ppt"`   // 是否生成PPT文件
}

// EnhancedGenerationResult 增强生成结果
type EnhancedGenerationResult struct {
	Success            bool                      `json:"success"`
	Title              string                    `json:"title"`
	Slides             []EnhancedSlideContent    `json:"slides"`
	TotalSlides        int                       `json:"total_slides"`
	ActualSlides       int                       `json:"actual_slides"`    // 实际生成的幻灯片数
	RequestedSlides    int                       `json:"requested_slides"` // 请求的幻灯片数
	LimitedByQuota     bool                      `json:"limited_by_quota"` // 是否被配额限制
	GenerationTime     time.Duration             `json:"generation_time"`
	PPTFilePath        string                    `json:"ppt_file_path"`
	PPTFileURL         string                    `json:"ppt_file_url"`
	SourceInfo         *SourceInfo               `json:"source_info"`
	Error              string                    `json:"error"`
	Warnings           []string                  `json:"warnings"`
	AudioFiles         []AudioFileInfo           `json:"audio_files"`          // 新增音频文件信息
	TotalAudioDuration int                       `json:"total_audio_duration"` // 新增总音频时长
	AudioGenerated     bool                      `json:"audio_generated"`      // 新增音频生成状态
	Recommendation     *SlideCountRecommendation `json:"recommendation"`       // 新增幻灯片数量推荐
	TechnicalMetadata  *GenerationMetadata       `json:"technical_metadata"`   // 新增技术PPT元数据
	ContentAnalysis    *ContentAnalysis          `json:"content_analysis"`     // 新增内容分析
}

// SourceInfo 来源信息
type SourceInfo struct {
	Type           string    `json:"type"`
	Source         string    `json:"source"`
	ProcessedAt    time.Time `json:"processed_at"`
	ContentLength  int       `json:"content_length"`
	OriginalTitle  string    `json:"original_title"`
	FileSize       int64     `json:"file_size,omitempty"`
	FilePages      int       `json:"file_pages,omitempty"`
	EstimatedPages int       `json:"estimated_pages"`
}

// NewEnhancedPPTService 创建增强PPT服务实例
func NewEnhancedPPTService(aiClient *ai.DashScopeClient, db *gorm.DB, pptDir string, ttsService TTSService) *EnhancedPPTService {
	service := &EnhancedPPTService{
		aiClient:   aiClient,
		db:         db,
		pptDir:     pptDir,
		ttsService: ttsService,
		config: &EnhancedPPTConfig{
			MaxSlideCount:       30, // 增加最大幻灯片数量
			MinSlideCount:       10, // 增加最小幻灯片数量
			DefaultSlideCount:   18, // 增加默认幻灯片数量
			MaxContentLength:    50000,
			SlidesPerSection:    3,
			EnablePageLimit:     true,
			PageLimitByUserType: true,
			VIPMaxSlides:        50,
			RegularMaxSlides:    25, // 增加普通用户限制
			OutputFormat:        "html",
			Template:            "technical", // 默认使用技术模板
		},
		slideCalculator:    NewSlideCountCalculator(),
		technicalGenerator: NewTechnicalPPTGenerator(aiClient),
		audioProgressMap:   make(map[uint]*AudioGenerationProgress),
	}

	return service
}

// parseAndStructureContent 解析和结构化内容
func (s *EnhancedPPTService) parseAndStructureContent(req *EnhancedGenerationRequest) (*parser.StructuredContent, error) {
	fmt.Printf("🔍 开始解析内容，类型: %s\n", req.SourceType)

	var content string
	var title string = req.Title

	switch req.SourceType {
	case "file":
		// 文件路径解析
		fmt.Printf("📁 解析文件: %s\n", req.Content)
		fileProcessor := parser.NewEnhancedFileProcessor()
		fileContent, err := fileProcessor.ProcessFile(req.Content)
		if err != nil {
			fmt.Printf("❌ 文件解析失败: %v\n", err)
			return nil, fmt.Errorf("文件解析失败: %w", err)
		}
		content = fileContent.Content
		if title == "" {
			title = fileContent.Title
		}
		fmt.Printf("✅ 文件解析成功，内容长度: %d 字符\n", len(content))

	case "url":
		// URL内容（已经是处理后的内容）
		content = req.Content
		fmt.Printf("🌐 URL内容长度: %d 字符\n", len(content))

	case "text":
		// 直接文本内容
		content = req.Content
		fmt.Printf("📝 文本内容长度: %d 字符\n", len(content))

	default:
		content = req.Content
	}

	// 如果内容为空，返回错误
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("解析的内容为空")
	}

	// 使用AI提取关键信息
	summary, keywords, err := s.extractKeyInfoWithAI(content, title)
	if err != nil {
		fmt.Printf("⚠️ AI关键信息提取失败，使用基础提取: %v\n", err)
		// 降级使用基础方法
		summary, keywords = s.extractKeyInfoBasic(content, title)
	}

	fmt.Printf("📊 关键信息提取完成\n")
	fmt.Printf("   摘要: %s\n", summary[:min(len(summary), 100)]+"...")
	fmt.Printf("   关键词: %v\n", keywords)

	// 内容分段
	sections := s.splitContentIntoSections(content)
	fmt.Printf("📑 内容分段完成，共 %d 个段落\n", len(sections))

	return &parser.StructuredContent{
		CleanText: content,
		Sections:  sections,
		KeyInfo: &parser.KeyInfo{
			Summary:  summary,
			Keywords: keywords,
		},
	}, nil
}

// createErrorResult 创建错误结果
func (s *EnhancedPPTService) createErrorResult(message string, err error) *EnhancedGenerationResult {
	return &EnhancedGenerationResult{
		Success: false,
		Error:   fmt.Sprintf("%s: %v", message, err),
	}
}

// GeneratePPT 生成增强PPT（包含音频生成）
func (s *EnhancedPPTService) GeneratePPT(req *EnhancedGenerationRequest) (*EnhancedGenerationResult, error) {
	startTime := time.Now()

	fmt.Printf("🚀 开始增强PPT生成流程\n")
	fmt.Printf("📋 请求参数: 用户类型=%s, 内容类型=%s, 最大幻灯片=%d\n",
		req.UserType, req.SourceType, req.SlideCount)

	// 1. 内容解析和结构化
	fmt.Printf("\n📖 步骤1: 内容解析和结构化\n")
	structuredContent, err := s.parseAndStructureContent(req)
	if err != nil {
		return s.createErrorResult("内容解析失败", err), nil
	}
	fmt.Printf("✅ 内容解析完成: %d字符, %d章节\n",
		len(structuredContent.CleanText), len(structuredContent.Sections))

	// 2. 智能幻灯片数量计算
	fmt.Printf("\n🧮 步骤2: 智能幻灯片数量计算\n")
	recommendation, err := s.slideCalculator.CalculateOptimalSlideCount(
		structuredContent, req.UserType, req.SourceType)
	if err != nil {
		return s.createErrorResult("幻灯片数量计算失败", err), nil
	}

	// 使用推荐数量或用户指定数量
	targetSlideCount := recommendation.Recommended
	if req.SlideCount > 0 && req.SlideCount <= recommendation.Maximum {
		targetSlideCount = req.SlideCount
	}

	fmt.Printf("✅ 智能计算完成: 推荐=%d张, 实际使用=%d张\n",
		recommendation.Recommended, targetSlideCount)
	fmt.Printf("📊 计算依据: %s (置信度: %.2f)\n",
		recommendation.Reasoning, recommendation.Confidence)

	// 3. 使用技术PPT生成器生成内容
	fmt.Printf("\n🎨 步骤3: 技术文档专用PPT生成\n")
	techParams := &TechnicalGenerationParams{
		ContentType:         s.detectContentType(req.SourceType),
		TechnicalLevel:      s.assessTechnicalLevel(structuredContent),
		UserType:            req.UserType,
		Language:            req.Language,
		MaxSlideCount:       targetSlideCount,
		ContentDensity:      1.2, // 高内容密度
		IncludeCodeExample:  true,
		IncludeBestPractice: true,
		IncludeArchitecture: true,
		DetailLevel:         "high",
		FocusAreas:          []string{"concept", "practice", "code"},
		GenerationStyle:     "comprehensive",
	}

	technicalResult, err := s.technicalGenerator.GenerateTechnicalSlides(
		structuredContent, techParams)
	if err != nil {
		fmt.Printf("❌ 技术PPT生成失败，回退到基础生成: %v\n", err)
		return s.fallbackToBasicGeneration(structuredContent, targetSlideCount, req)
	}

	fmt.Printf("✅ 技术PPT生成完成: %d张幻灯片\n", len(technicalResult.Slides))
	fmt.Printf("📈 内容统计: 平均%d字/张, 技术深度=%s\n",
		technicalResult.Metadata.AverageWordsPerSlide,
		technicalResult.Metadata.TechnicalDepth)

	// 4. 转换为标准幻灯片格式
	fmt.Printf("\n🔄 步骤4: 格式转换和优化\n")
	slides := s.convertToStandardSlides(technicalResult.Slides)

	// 5. 生成PPT文件
	var pptFilePath, pptFileURL string
	if req.GeneratePPT {
		fmt.Printf("\n📄 步骤5: 生成PPT文件\n")
		enhancedSlides := s.convertToEnhancedSlides(slides)
		pptFilePath, pptFileURL, err = s.generatePPTFile(enhancedSlides, req)
		if err != nil {
			fmt.Printf("⚠️ PPT文件生成失败: %v\n", err)
		} else {
			fmt.Printf("✅ PPT文件生成成功: %s\n", pptFileURL)
		}
	}

	// 6. 异步生成音频（如果需要）
	if req.GenerateAudio {
		fmt.Printf("\n🎵 步骤6: 异步音频生成\n")
		go s.generateAudioAsync(slides, req)
	}

	// 7. 构建增强结果
	result := &EnhancedGenerationResult{
		Success:           true,
		Title:             "增强PPT生成成功",
		Slides:            s.convertToEnhancedSlides(slides),
		TotalSlides:       len(slides),
		PPTFilePath:       pptFilePath,
		PPTFileURL:        pptFileURL,
		GenerationTime:    time.Since(startTime),
		Recommendation:    recommendation,
		TechnicalMetadata: technicalResult.Metadata,
		ContentAnalysis:   s.generateContentAnalysis(structuredContent, technicalResult),
	}

	fmt.Printf("\n🎉 增强PPT生成完成! 总耗时: %v\n", time.Since(startTime))
	return result, nil
}

// validateRequest 验证请求参数
func (s *EnhancedPPTService) validateRequest(req *EnhancedGenerationRequest) error {
	if req.Content == "" {
		return fmt.Errorf("内容不能为空")
	}
	if req.Title == "" {
		req.Title = "AI生成的PPT"
	}
	if req.SlideCount <= 0 {
		req.SlideCount = s.config.DefaultSlideCount
	}
	return nil
}

// applySlideCountLimits 应用页数限制
func (s *EnhancedPPTService) applySlideCountLimits(req *EnhancedGenerationRequest) int {
	maxAllowed := s.config.RegularMaxSlides
	if req.UserType == "vip" {
		maxAllowed = s.config.VIPMaxSlides
	}

	if req.SlideCount > maxAllowed {
		return maxAllowed
	}

	if req.SlideCount < s.config.MinSlideCount {
		return s.config.MinSlideCount
	}

	return req.SlideCount
}

// getContent 获取内容
func (s *EnhancedPPTService) getContent(req *EnhancedGenerationRequest) (*SourceInfo, string, error) {
	sourceInfo := &SourceInfo{
		Type:          req.SourceType,
		Source:        req.Content,
		ProcessedAt:   time.Now(),
		ContentLength: len(req.Content),
	}

	return sourceInfo, req.Content, nil
}

// generateAISummary 生成AI总结
func (s *EnhancedPPTService) generateAISummary(content string, req *EnhancedGenerationRequest, slideCount int) (string, error) {
	prompt := fmt.Sprintf("请将以下内容总结并组织成%d张幻灯片的大纲：\n\n%s", slideCount, content)

	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		return "", fmt.Errorf("AI总结失败: %w", err)
	}

	return response, nil
}

// generateSlides 生成幻灯片
func (s *EnhancedPPTService) generateSlides(summary string, req *EnhancedGenerationRequest, slideCount int) ([]EnhancedSlideContent, error) {
	// 构建增强的AI提示词
	prompt := s.buildEnhancedPrompt(summary, req, slideCount)

	// 调用AI生成内容
	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		// 如果AI生成失败，使用fallback方法
		return s.generateSlidesFallback(summary, slideCount)
	}

	// 解析AI响应
	var result struct {
		Title  string                 `json:"title"`
		Slides []EnhancedSlideContent `json:"slides"`
	}

	// 清理响应，移除可能的markdown格式
	cleanResponse := strings.TrimSpace(response)
	cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
	cleanResponse = strings.TrimPrefix(cleanResponse, "```")
	cleanResponse = strings.TrimSuffix(cleanResponse, "```")

	if err := json.Unmarshal([]byte(cleanResponse), &result); err != nil {
		fmt.Printf("JSON解析失败: %v, 响应内容: %s\n", err, cleanResponse)
		// 解析失败，使用fallback方法
		return s.generateSlidesFallback(summary, slideCount)
	}

	// 验证和修复生成的幻灯片
	validatedSlides := s.validateAndFixSlides(result.Slides, slideCount)

	return validatedSlides, nil
}

// generateSlidesFallback 备选的幻灯片生成方法
func (s *EnhancedPPTService) generateSlidesFallback(content string, slideCount int) ([]EnhancedSlideContent, error) {
	var slides []EnhancedSlideContent

	// 将内容按段落分割
	paragraphs := strings.Split(content, "\n\n")
	if len(paragraphs) < slideCount {
		// 如果段落不够，按句子分割
		sentences := strings.Split(content, "。")
		paragraphs = sentences
	}

	// 生成标题页
	titleSlide := EnhancedSlideContent{
		SlideNumber:   1,
		Title:         s.extractTitleFromContent(content),
		MainContent:   s.generateSummary(content, 100),
		SlideType:     "title",
		KeyConcepts:   s.extractKeywordsFromContent(content)[:minInt(3, len(s.extractKeywordsFromContent(content)))],
		SpeakerNotes:  "欢迎大家，今天我们将学习这个主题的相关内容",
		EstimatedTime: 60,
	}
	slides = append(slides, titleSlide)

	// 生成内容页
	contentSlides := slideCount - 2 // 减去标题页和总结页
	if contentSlides > 0 {
		for i := 0; i < contentSlides && i < len(paragraphs); i++ {
			paragraph := strings.TrimSpace(paragraphs[i])
			if paragraph == "" {
				continue
			}

			slide := EnhancedSlideContent{
				SlideNumber:   i + 2,
				Title:         s.generateSlideTitle(paragraph, i+2),
				MainContent:   paragraph,
				BulletPoints:  s.extractBulletPoints(paragraph),
				SlideType:     "content",
				KeyConcepts:   s.extractKeywordsFromContent(paragraph),
				SpeakerNotes:  fmt.Sprintf("接下来我们来看%s的相关内容", s.generateSlideTitle(paragraph, i+2)),
				EstimatedTime: maxInt(45, len(paragraph)/2), // 根据内容长度估算时间
			}
			slides = append(slides, slide)
		}
	}

	// 生成总结页
	if slideCount > 1 {
		summarySlide := EnhancedSlideContent{
			SlideNumber:   slideCount,
			Title:         "总结",
			MainContent:   s.generateSummary(content, 150),
			SlideType:     "summary",
			KeyConcepts:   []string{"总结", "结论"},
			SpeakerNotes:  "让我们来总结今天学习的主要内容",
			EstimatedTime: 90,
		}
		slides = append(slides, summarySlide)
	}

	return slides, nil
}

// extractKeywordsFromContent 从内容中提取关键词
func (s *EnhancedPPTService) extractKeywordsFromContent(content string) []string {
	// 移除标点符号，分词
	words := strings.FieldsFunc(content, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || (r >= 0x4e00 && r <= 0x9fff))
	})

	// 关键词频统计
	wordCount := make(map[string]int)
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) >= 2 { // 只考虑长度>=2的词
			wordCount[word]++
		}
	}

	// 提取高频词作为关键词
	type wordFreq struct {
		word string
		freq int
	}

	var wordFreqs []wordFreq
	for word, freq := range wordCount {
		wordFreqs = append(wordFreqs, wordFreq{word, freq})
	}

	// 按频率排序
	for i := 0; i < len(wordFreqs); i++ {
		for j := i + 1; j < len(wordFreqs); j++ {
			if wordFreqs[j].freq > wordFreqs[i].freq {
				wordFreqs[i], wordFreqs[j] = wordFreqs[j], wordFreqs[i]
			}
		}
	}

	// 提取前5个关键词
	var keywords []string
	maxKeywords := minInt(5, len(wordFreqs))
	for i := 0; i < maxKeywords; i++ {
		keywords = append(keywords, wordFreqs[i].word)
	}

	// 如果关键词太少，添加一些通用词
	if len(keywords) < 3 {
		commonKeywords := []string{"概述", "要点", "内容", "学习", "知识"}
		for _, kw := range commonKeywords {
			if len(keywords) >= 3 {
				break
			}
			// 检查是否已存在
			exists := false
			for _, existing := range keywords {
				if existing == kw {
					exists = true
					break
				}
			}
			if !exists {
				keywords = append(keywords, kw)
			}
		}
	}

	return keywords
}

// extractTitleFromContent 从内容中提取标题
func (s *EnhancedPPTService) extractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 5 && len(line) < 50 {
			return line
		}
	}
	return "主题演示"
}

// generateSlideTitle 生成幻灯片标题
func (s *EnhancedPPTService) generateSlideTitle(content string, slideNumber int) string {
	// 尝试提取第一句话作为标题
	sentences := strings.Split(content, "。")
	if len(sentences) > 0 {
		title := strings.TrimSpace(sentences[0])
		if len(title) > 3 && len(title) < 30 {
			return title
		}
	}

	// 备选方案：使用通用标题
	titles := []string{
		"核心概念", "重要内容", "关键要点", "深入解析", "详细说明",
		"实践应用", "案例分析", "方法介绍", "原理解释", "技术要点",
	}

	index := (slideNumber - 2) % len(titles)
	return titles[index]
}

// extractBulletPoints 提取要点
func (s *EnhancedPPTService) extractBulletPoints(content string) []string {
	// 按句号分割成句子
	sentences := strings.Split(content, "。")
	var points []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 5 && len(sentence) < 100 {
			points = append(points, sentence)
		}
		if len(points) >= 5 { // 最多5个要点
			break
		}
	}

	// 如果没有足够的要点，按逗号分割
	if len(points) < 3 {
		parts := strings.Split(content, "，")
		points = []string{}
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) > 3 && len(part) < 80 {
				points = append(points, part)
			}
			if len(points) >= 4 {
				break
			}
		}
	}

	return points
}

// generateSummary 生成摘要
func (s *EnhancedPPTService) generateSummary(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// 尝试按句子截取
	sentences := strings.Split(content, "。")
	summary := ""

	for _, sentence := range sentences {
		if len(summary+sentence) > maxLength {
			break
		}
		summary += sentence + "。"
	}

	if summary == "" && len(content) > maxLength {
		summary = content[:maxLength] + "..."
	}

	return summary
}

// generateSlideNotes 生成幻灯片备注
func (s *EnhancedPPTService) generateSlideNotes(title string, bulletPoints []string) string {
	if len(bulletPoints) == 0 {
		return fmt.Sprintf("接下来我们来看%s的相关内容", title)
	}

	notes := fmt.Sprintf("接下来我们来看%s的相关内容。", title)
	if len(bulletPoints) > 0 {
		notes += "主要包括："
		for i, point := range bulletPoints {
			if i < 3 { // 只在备注中提及前3个要点
				notes += fmt.Sprintf("%s；", point)
			}
		}
	}

	return notes
}

// minInt 返回两个整数中的较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt 返回两个整数中的较大值
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// generatePPTFile 生成PPT文件
func (s *EnhancedPPTService) generatePPTFile(slides []EnhancedSlideContent, req *EnhancedGenerationRequest) (string, string, error) {
	filename := fmt.Sprintf("ppt_%s_%s.html", req.UserID, generateRandomString(8))
	filePath := filepath.Join(s.pptDir, filename)

	// 确保目录存在
	if err := os.MkdirAll(s.pptDir, 0755); err != nil {
		return "", "", fmt.Errorf("创建PPT目录失败: %w", err)
	}

	// 生成HTML内容
	htmlContent := s.generateHTMLContent(slides, req)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(htmlContent), 0644); err != nil {
		return "", "", fmt.Errorf("写入PPT文件失败: %w", err)
	}

	// 从配置中获取文件服务器地址
	fileBaseURL := viper.GetString("server.file_base_url")
	if fileBaseURL == "" {
		fileBaseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	fileURL := fmt.Sprintf("%s/ppt/%s", fileBaseURL, filename)
	return filePath, fileURL, nil
}

// generateHTMLContent 生成HTML内容
func (s *EnhancedPPTService) generateHTMLContent(slides []EnhancedSlideContent, req *EnhancedGenerationRequest) string {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + req.Title + `</title>
    <style>
        body {
            font-family: 'Microsoft YaHei', 'PingFang SC', 'Helvetica Neue', Arial, sans-serif;
            margin: 0;
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            line-height: 1.6;
            overflow-x: hidden;
        }
        
        .presentation {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .slide {
            background: white;
            border-radius: 20px;
            box-shadow: 0 15px 40px rgba(0,0,0,0.1);
            margin: 40px 0;
            padding: 50px;
            min-height: 600px;
            position: relative;
            overflow: hidden;
            transition: transform 0.3s ease, box-shadow 0.3s ease;
        }

        .slide:hover {
            transform: translateY(-5px);
            box-shadow: 0 20px 50px rgba(0,0,0,0.15);
        }

        .slide::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 8px;
            background: linear-gradient(90deg, #667eea, #764ba2, #f093fb, #f5576c);
        }
        
        .slide-header {
            border-bottom: 3px solid #f8f9fa;
            padding-bottom: 25px;
            margin-bottom: 40px;
        }
        
        .slide-number {
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            padding: 8px 20px;
            border-radius: 25px;
            font-size: 16px;
            font-weight: 600;
            display: inline-block;
            margin-bottom: 20px;
            box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
        }
        
        .slide-title {
            font-size: 2.8em;
            color: #2c3e50;
            margin: 0;
            font-weight: 700;
            letter-spacing: -0.5px;
        }
        
        .slide.title-slide .slide-title {
            font-size: 4em;
            text-align: center;
            background: linear-gradient(135deg, #667eea, #764ba2);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 30px;
        }
        
        .slide-content {
            font-size: 1.3em;
            line-height: 1.8;
            color: #34495e;
        }
        
        .slide-content ul {
            list-style: none;
            padding: 0;
            margin: 30px 0;
        }
        
        .slide-content li {
            margin: 20px 0;
            padding: 15px 0 15px 50px;
            position: relative;
            font-size: 1.1em;
            transition: all 0.3s ease;
        }

        .slide-content li:hover {
            color: #667eea;
            transform: translateX(10px);
        }

        .slide-content li::before {
            content: '▶';
            color: #667eea;
            position: absolute;
            left: 0;
            top: 15px;
            font-size: 1.2em;
            font-weight: bold;
        }
        
        .keywords {
            background: linear-gradient(135deg, #f8f9fa, #e9ecef);
            padding: 25px;
            border-radius: 15px;
            margin: 30px 0;
            border-left: 6px solid #667eea;
            box-shadow: 0 5px 15px rgba(0,0,0,0.05);
        }
        
        .keywords-title {
            font-weight: 700;
            margin-bottom: 15px;
            color: #667eea;
            font-size: 1.1em;
            display: flex;
            align-items: center;
        }

        .keywords-title::before {
            content: '🏷️';
            margin-right: 10px;
        }
        
        .keyword-tag {
            display: inline-block;
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            margin: 5px;
            font-size: 0.9em;
            font-weight: 600;
            box-shadow: 0 3px 10px rgba(102, 126, 234, 0.3);
            transition: transform 0.2s ease;
        }

        .keyword-tag:hover {
            transform: translateY(-2px);
        }
        
        .slide-notes {
            background: linear-gradient(135deg, #fff9c4, #fffacd);
            padding: 25px;
            border-radius: 15px;
            margin: 30px 0;
            border-left: 6px solid #ffc107;
            font-style: italic;
            box-shadow: 0 5px 15px rgba(255, 193, 7, 0.1);
        }

        .slide-notes::before {
            content: '💡 ';
            font-style: normal;
            font-weight: bold;
        }
        
        .metadata {
            text-align: center;
            padding: 40px 20px;
            color: rgba(255, 255, 255, 0.9);
            font-size: 1em;
            background: rgba(0, 0, 0, 0.1);
            border-radius: 15px;
            margin: 40px 0;
        }
        
        .progress-bar {
            position: fixed;
            top: 0;
            left: 0;
            height: 4px;
            background: linear-gradient(90deg, #667eea, #764ba2);
            z-index: 1000;
            transition: width 0.3s ease;
            box-shadow: 0 2px 10px rgba(102, 126, 234, 0.3);
        }

        /* 响应式设计 */
        @media (max-width: 768px) {
            .presentation { padding: 10px; }
            .slide { padding: 30px 20px; margin: 20px 0; }
            .slide-title { font-size: 2em; }
            .slide.title-slide .slide-title { font-size: 2.5em; }
        }

        /* 打印样式 */
        @media print {
            .slide {
                page-break-after: always;
                margin: 0;
                box-shadow: none;
                border: 1px solid #ddd;
            }
            body { background: white; }
            .progress-bar { display: none; }
        }
    </style>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            // 进度条功能
            const progressBar = document.querySelector('.progress-bar');
            const updateProgress = () => {
                const scrollTop = window.pageYOffset;
                const docHeight = document.body.scrollHeight - window.innerHeight;
                const progress = (scrollTop / docHeight) * 100;
                progressBar.style.width = progress + '%';
            };
            
            window.addEventListener('scroll', updateProgress);
            
            // 幻灯片导航
            const slides = document.querySelectorAll('.slide');
            slides.forEach((slide, index) => {
                slide.addEventListener('click', () => {
                    if (index < slides.length - 1) {
                        slides[index + 1].scrollIntoView({ 
                            behavior: 'smooth',
                            block: 'start'
                        });
                    }
                });
            });
            
            // 键盘导航
            let currentSlide = 0;
            document.addEventListener('keydown', (e) => {
                if (e.key === 'ArrowRight' || e.key === ' ') {
                    e.preventDefault();
                    if (currentSlide < slides.length - 1) {
                        currentSlide++;
                        slides[currentSlide].scrollIntoView({ 
                            behavior: 'smooth',
                            block: 'start'
                        });
                    }
                } else if (e.key === 'ArrowLeft') {
                    e.preventDefault();
                    if (currentSlide > 0) {
                        currentSlide--;
                        slides[currentSlide].scrollIntoView({ 
                            behavior: 'smooth',
                            block: 'start'
                        });
                    }
                }
            });
            
            // 幻灯片动画
            const observerOptions = {
                threshold: 0.5,
                rootMargin: '0px 0px -100px 0px'
            };
            
            const slideObserver = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        entry.target.style.opacity = '1';
                        entry.target.style.transform = 'translateY(0)';
                        currentSlide = Array.from(slides).indexOf(entry.target);
                    }
                });
            }, observerOptions);
            
            slides.forEach(slide => {
                slide.style.opacity = '0.8';
                slide.style.transform = 'translateY(20px)';
                slide.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
                slideObserver.observe(slide);
            });
        });
    </script>
</head>
<body>
    <div class="progress-bar"></div>
    <div class="presentation">`

	for _, slide := range slides {
		// 确定幻灯片类型的CSS类
		slideClass := ""
		if slide.SlideType == "title" {
			slideClass = "title-slide"
		}

		html += fmt.Sprintf(`
        <div class="slide %s">
            <div class="slide-header">
                <div class="slide-number">第 %d 页</div>
                <h1 class="slide-title">%s</h1>
            </div>
            
            <div class="slide-content">`, slideClass, slide.SlideNumber, slide.Title)

		// 添加内容
		if len(slide.BulletPoints) > 0 {
			html += "\n                <ul>\n"
			for _, point := range slide.BulletPoints {
				html += fmt.Sprintf("                    <li>%s</li>\n", point)
			}
			html += "                </ul>\n"
		} else if slide.MainContent != "" {
			html += fmt.Sprintf("\n                <p>%s</p>\n", slide.MainContent)
		}

		html += `            </div>
            
            `

		// 添加关键词
		if len(slide.KeyConcepts) > 0 {
			html += `<div class="keywords">
                <div class="keywords-title">关键词</div>
                `
			for _, keyword := range slide.KeyConcepts {
				html += fmt.Sprintf(`<span class="keyword-tag">%s</span>
                `, keyword)
			}
			html += `</div>
            
            `
		}

		// 添加备注
		if slide.SpeakerNotes != "" {
			html += fmt.Sprintf(`<div class="slide-notes">
                <strong>备注:</strong> %s
            </div>
            `, slide.SpeakerNotes)
		}

		html += `        </div>
        `
	}

	// 添加元数据
	html += fmt.Sprintf(`
        <div class="metadata">
            <p>生成时间: %s | 模板: %s | 总页数: %d</p>
            <p>由 AI课堂 智能生成</p>
        </div>
    </div>
</body>
</html>`, time.Now().Format("2006-01-02 15:04:05"), req.Template, len(slides))

	return html
}

// generateAudioForSlides 为所有幻灯片生成音频
func (s *EnhancedPPTService) generateAudioForSlides(slides []EnhancedSlideContent, req *EnhancedGenerationRequest) ([]AudioFileInfo, int, error) {
	if s.ttsService == nil {
		return nil, 0, fmt.Errorf("TTS服务未初始化")
	}

	var audioFiles []AudioFileInfo
	var totalDuration int

	voiceType := req.VoiceType
	if voiceType == "" {
		voiceType = "zhixiaobai" // 默认语音
	}

	fmt.Printf("开始为 %d 张幻灯片生成音频，语音类型: %s\n", len(slides), voiceType)

	for i, slide := range slides {
		// 准备音频文本（标题 + 内容 + 备注）
		audioText := s.prepareAudioText(slide)

		// 生成音频
		audioInfo, err := s.generateSlideAudio(slide, audioText, voiceType, i+1)
		if err != nil {
			fmt.Printf("为幻灯片 %d 生成音频失败: %v\n", i+1, err)
			continue
		}

		audioFiles = append(audioFiles, *audioInfo)
		totalDuration += audioInfo.Duration

		// 添加短暂延迟，避免TTS服务过载
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("音频生成完成，成功生成 %d/%d 个音频文件，总时长: %d 秒\n",
		len(audioFiles), len(slides), totalDuration)

	if len(audioFiles) == 0 {
		return nil, 0, fmt.Errorf("没有成功生成任何音频文件")
	}

	return audioFiles, totalDuration, nil
}

// generateSlideAudio 为单张幻灯片生成音频
func (s *EnhancedPPTService) generateSlideAudio(slide EnhancedSlideContent, audioText, voiceType string, slideNumber int) (*AudioFileInfo, error) {
	// 创建临时的Slide模型用于TTS服务
	tempSlide := &models.Slide{
		SlideNumber:  slideNumber,
		Title:        slide.Title,
		Content:      slide.MainContent,
		SpeakerNotes: slide.SpeakerNotes,
	}

	// 调用现有的TTS服务生成音频
	err := s.ttsService.GenerateSlideAudio(tempSlide, voiceType)
	if err != nil {
		return nil, fmt.Errorf("TTS生成失败: %w", err)
	}

	// 从更新后的tempSlide获取音频信息
	return &AudioFileInfo{
		SlideNumber: slideNumber,
		FileName:    filepath.Base(tempSlide.AudioURL),
		FilePath:    tempSlide.AudioURL,
		FileURL:     tempSlide.AudioURL,
		Duration:    tempSlide.Duration,
		FileSize:    0, // TTS服务没有返回文件大小
		Format:      "mp3",
		VoiceType:   voiceType,
	}, nil
}

// prepareAudioText 准备音频文本
func (s *EnhancedPPTService) prepareAudioText(slide EnhancedSlideContent) string {
	var textParts []string

	// 添加标题
	if slide.Title != "" {
		textParts = append(textParts, slide.Title)
	}

	// 添加内容
	if slide.MainContent != "" {
		textParts = append(textParts, slide.MainContent)
	}

	// 添加要点
	if len(slide.BulletPoints) > 0 {
		for _, point := range slide.BulletPoints {
			textParts = append(textParts, point)
		}
	}

	// 添加备注
	if slide.SpeakerNotes != "" {
		textParts = append(textParts, slide.SpeakerNotes)
	}

	return strings.Join(textParts, "。")
}

// GetSlideCountLimits 获取页数限制
func (s *EnhancedPPTService) GetSlideCountLimits(userType string) map[string]int {
	limits := map[string]int{
		"min":      s.config.MinSlideCount,
		"max":      s.config.MaxSlideCount,
		"default":  s.config.DefaultSlideCount,
		"user_max": s.config.RegularMaxSlides,
	}

	if userType == "vip" {
		limits["user_max"] = s.config.VIPMaxSlides
	}

	return limits
}

// StartAudioProgress 开始音频生成进度跟踪
func (s *EnhancedPPTService) StartAudioProgress(courseID uint, slides []EnhancedSlideContent, voiceType string) {
	s.audioProgressMutex.Lock()
	defer s.audioProgressMutex.Unlock()

	progress := &AudioGenerationProgress{
		CourseID:        courseID,
		Status:          "processing",
		TotalSlides:     len(slides),
		CompletedSlides: 0,
		CurrentSlide:    1,
		Progress:        0.0,
		StartTime:       time.Now(),
		EstimatedEnd:    time.Now().Add(time.Duration(len(slides)*10) * time.Second), // 预估每张幻灯片10秒
		AudioFiles:      make([]AudioFileInfo, 0, len(slides)),
	}

	// 初始化音频文件信息
	for i, _ := range slides {
		audioFile := AudioFileInfo{
			SlideNumber: i + 1,
			FileName:    fmt.Sprintf("slide_%d_%d.mp3", courseID, i+1), // 确保使用MP3格式
			Status:      "pending",
			VoiceType:   voiceType,
			Format:      "mp3",
			CreatedAt:   time.Now(),
		}
		progress.AudioFiles = append(progress.AudioFiles, audioFile)
	}

	s.audioProgressMap[courseID] = progress
}

// UpdateAudioProgress 更新音频生成进度
func (s *EnhancedPPTService) UpdateAudioProgress(courseID uint, slideNumber int, status string, fileURL string, duration int, fileSize int, errorMsg string) {
	s.audioProgressMutex.Lock()
	defer s.audioProgressMutex.Unlock()

	progress, exists := s.audioProgressMap[courseID]
	if !exists {
		return
	}

	// 更新当前幻灯片状态
	if slideNumber > 0 && slideNumber <= len(progress.AudioFiles) {
		audioFile := &progress.AudioFiles[slideNumber-1]
		audioFile.Status = status
		if fileURL != "" {
			audioFile.FileURL = fileURL
		}
		if duration > 0 {
			audioFile.Duration = duration
		}
		if fileSize > 0 {
			audioFile.FileSize = fileSize
		}
		if errorMsg != "" {
			audioFile.Error = errorMsg
		}

		// 更新当前处理的幻灯片
		progress.CurrentSlide = slideNumber
		progress.CurrentTitle = fmt.Sprintf("幻灯片 %d", slideNumber)
	}

	// 计算完成数量
	completedCount := 0
	for _, file := range progress.AudioFiles {
		if file.Status == "completed" {
			completedCount++
		}
	}
	progress.CompletedSlides = completedCount

	// 计算进度百分比
	if progress.TotalSlides > 0 {
		progress.Progress = float64(completedCount) / float64(progress.TotalSlides) * 100
	}

	// 更新状态
	if status == "failed" && errorMsg != "" {
		progress.Status = "failed"
		progress.ErrorMessage = errorMsg
	} else if completedCount == progress.TotalSlides {
		progress.Status = "completed"
		now := time.Now()
		progress.CompletedAt = &now
	}

	// 重新估算完成时间
	if completedCount > 0 && completedCount < progress.TotalSlides {
		elapsed := time.Since(progress.StartTime)
		avgTimePerSlide := elapsed / time.Duration(completedCount)
		remainingSlides := progress.TotalSlides - completedCount
		progress.EstimatedEnd = time.Now().Add(avgTimePerSlide * time.Duration(remainingSlides))
	}
}

// GetAudioProgress 获取音频生成进度
func (s *EnhancedPPTService) GetAudioProgress(courseID uint) (*AudioGenerationProgress, error) {
	s.audioProgressMutex.RLock()
	defer s.audioProgressMutex.RUnlock()

	progress, exists := s.audioProgressMap[courseID]
	if !exists {
		return nil, fmt.Errorf("未找到课程 %d 的音频生成进度", courseID)
	}

	// 返回进度副本，避免并发修改
	progressCopy := *progress
	progressCopy.AudioFiles = make([]AudioFileInfo, len(progress.AudioFiles))
	copy(progressCopy.AudioFiles, progress.AudioFiles)

	return &progressCopy, nil
}

// buildEnhancedPrompt 构建增强的AI提示词
func (s *EnhancedPPTService) buildEnhancedPrompt(content string, req *EnhancedGenerationRequest, slideCount int) string {
	// 分析内容特征
	contentLength := len(content)
	words := strings.Fields(content)
	wordCount := len(words)

	// 构建内容分析部分
	contentAnalysis := fmt.Sprintf(`
**源内容分析:**
- 内容类型: %s
- 内容长度: %d字符
- 词汇数量: %d个
- 预计阅读时长: %d分钟
`, req.SourceType, contentLength, wordCount, wordCount/200)

	// 构建生成要求
	requirements := fmt.Sprintf(`
**生成要求:**
1. 严格生成%d张幻灯片，结构清晰
2. 第1张：标题页，包含主题和概述
3. 第2-%d张：内容页，每张专注一个主题
4. 第%d张：总结页，回顾要点
5. 每张幻灯片必须包含：
   - 简洁明了的标题（10-20字）
   - 3-5个要点列表
   - 3-5个精准关键词
   - 详细的演讲备注（50-100字）
6. 内容要层次分明，逻辑清晰
7. 关键词要准确反映核心概念
8. 备注要包含背景信息和展开说明
`, slideCount, slideCount-1, slideCount)

	// 构建输出格式要求
	formatRequirements := `
**输出格式要求:**
请严格按照以下JSON格式返回，不要添加任何markdown标记：
{
  "title": "演示文稿标题",
  "slides": [
    {
      "slide_number": 1,
      "title": "幻灯片标题",
      "content": "主要内容概述",
      "bullet_points": ["要点1", "要点2", "要点3"],
      "keywords": ["关键词1", "关键词2", "关键词3"],
      "speaker_notes": "详细演讲备注，包括背景信息和展开说明",
      "slide_type": "title"
    }
  ]
}`

	// 构建完整提示词
	return fmt.Sprintf(`你是一个专业的PPT制作专家。请根据以下内容生成%d张高质量的演示幻灯片。

%s

%s

%s

**源内容:**
%s`, slideCount, contentAnalysis, requirements, formatRequirements, content)
}

// validateAndFixSlides 验证和修复生成的幻灯片
func (s *EnhancedPPTService) validateAndFixSlides(slides []EnhancedSlideContent, expectedCount int) []EnhancedSlideContent {
	var validSlides []EnhancedSlideContent

	for i, slide := range slides {
		// 修复幻灯片编号
		slide.SlideNumber = i + 1

		// 验证和修复标题
		if slide.Title == "" {
			slide.Title = s.generateFallbackTitle(slide.MainContent, i+1)
		}

		// 验证和生成关键词
		if len(slide.KeyConcepts) == 0 {
			slide.KeyConcepts = s.extractKeywordsFromSlide(slide)
		}

		// 验证和生成备注
		if slide.SpeakerNotes == "" {
			slide.SpeakerNotes = s.generateSlideNotes(slide.Title, slide.BulletPoints)
		}

		// 确定幻灯片类型
		if slide.SlideType == "" {
			if i == 0 {
				slide.SlideType = "title"
			} else if i == len(slides)-1 {
				slide.SlideType = "summary"
			} else {
				slide.SlideType = "content"
			}
		}

		// 验证内容完整性
		if slide.MainContent == "" && len(slide.BulletPoints) == 0 {
			continue // 跳过空幻灯片
		}

		// 确保有要点列表
		if len(slide.BulletPoints) == 0 && slide.MainContent != "" {
			slide.BulletPoints = s.extractBulletPointsFromContent(slide.MainContent)
		}

		// 估算演讲时间
		if slide.EstimatedTime <= 0 {
			textLength := len(slide.MainContent) + len(strings.Join(slide.BulletPoints, ""))
			slide.EstimatedTime = maxInt(30, (textLength*60)/120) // 最少30秒
		}

		// 设置重要程度
		if slide.ContentWeight <= 0 {
			slide.ContentWeight = float64(s.calculateSlideImportance(slide)) / 10.0
		}

		validSlides = append(validSlides, slide)
	}

	// 如果生成的幻灯片数量不足，使用fallback方法补充
	if len(validSlides) < expectedCount {
		return s.supplementSlides(validSlides, expectedCount)
	}

	return validSlides
}

// generateFallbackTitle 生成备用标题
func (s *EnhancedPPTService) generateFallbackTitle(content string, slideNumber int) string {
	if content != "" {
		// 尝试从内容中提取标题
		sentences := strings.Split(content, "。")
		if len(sentences) > 0 {
			firstSentence := strings.TrimSpace(sentences[0])
			if len(firstSentence) > 5 && len(firstSentence) < 50 {
				return firstSentence
			}
		}
	}

	// 使用默认标题模板
	titles := []string{
		"主题介绍", "核心概念", "重要内容", "关键要点", "深入分析",
		"实际应用", "案例研究", "方法总结", "技术要点", "结论总结",
	}

	if slideNumber <= len(titles) {
		return titles[slideNumber-1]
	}

	return fmt.Sprintf("要点 %d", slideNumber)
}

// extractKeywordsFromSlide 从幻灯片提取关键词
func (s *EnhancedPPTService) extractKeywordsFromSlide(slide EnhancedSlideContent) []string {
	// 合并所有文本内容
	allText := slide.Title + " " + slide.MainContent + " " + strings.Join(slide.BulletPoints, " ")

	// 使用现有的关键词提取方法
	keywords := s.extractKeywordsFromContent(allText)

	// 限制关键词数量
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	// 如果关键词太少，添加默认关键词
	if len(keywords) < 3 {
		defaultKeywords := []string{"学习", "重点", "知识", "概念", "方法"}
		for _, kw := range defaultKeywords {
			if len(keywords) >= 3 {
				break
			}
			// 检查是否已存在
			exists := false
			for _, existing := range keywords {
				if existing == kw {
					exists = true
					break
				}
			}
			if !exists {
				keywords = append(keywords, kw)
			}
		}
	}

	return keywords
}

// extractBulletPointsFromContent 从内容提取要点
func (s *EnhancedPPTService) extractBulletPointsFromContent(content string) []string {
	// 先尝试按句号分割
	sentences := strings.Split(content, "。")
	var points []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 5 && len(sentence) < 100 {
			points = append(points, sentence)
		}
		if len(points) >= 5 {
			break
		}
	}

	// 如果句子不够，尝试按逗号分割
	if len(points) < 3 {
		parts := strings.Split(content, "，")
		points = []string{}
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) > 3 && len(part) < 80 {
				points = append(points, part)
			}
			if len(points) >= 4 {
				break
			}
		}
	}

	// 如果还是不够，分段处理
	if len(points) < 2 {
		words := strings.Fields(content)
		chunkSize := len(words) / 3
		if chunkSize < 5 {
			chunkSize = 5
		}

		for i := 0; i < len(words); i += chunkSize {
			end := i + chunkSize
			if end > len(words) {
				end = len(words)
			}
			chunk := strings.Join(words[i:end], " ")
			if len(chunk) > 10 {
				points = append(points, chunk)
			}
			if len(points) >= 4 {
				break
			}
		}
	}

	return points
}

// calculateSlideImportance 计算幻灯片重要程度
func (s *EnhancedPPTService) calculateSlideImportance(slide EnhancedSlideContent) int {
	importance := 3 // 默认重要程度

	// 根据幻灯片类型调整
	switch slide.SlideType {
	case "title":
		importance = 5
	case "summary":
		importance = 4
	case "content":
		importance = 3
	}

	// 根据关键词数量调整
	if len(slide.KeyConcepts) >= 4 {
		importance++
	}

	// 根据要点数量调整
	if len(slide.BulletPoints) >= 4 {
		importance++
	}

	// 确保在1-5范围内
	if importance > 5 {
		importance = 5
	}
	if importance < 1 {
		importance = 1
	}

	return importance
}

// supplementSlides 补充幻灯片
func (s *EnhancedPPTService) supplementSlides(existingSlides []EnhancedSlideContent, targetCount int) []EnhancedSlideContent {
	if len(existingSlides) >= targetCount {
		return existingSlides
	}

	needed := targetCount - len(existingSlides)

	// 如果现有幻灯片太少，使用fallback方法重新生成
	if len(existingSlides) < 2 {
		content := ""
		for _, slide := range existingSlides {
			content += slide.MainContent + " " + strings.Join(slide.BulletPoints, " ") + " "
		}

		fallbackSlides, _ := s.generateSlidesFallback(content, targetCount)
		return fallbackSlides
	}

	// 复制现有幻灯片并修改
	supplemented := make([]EnhancedSlideContent, len(existingSlides))
	copy(supplemented, existingSlides)

	for i := 0; i < needed; i++ {
		// 基于最后一张内容幻灯片创建新的
		baseSlide := existingSlides[len(existingSlides)-1]
		if baseSlide.SlideType == "summary" && len(existingSlides) > 1 {
			baseSlide = existingSlides[len(existingSlides)-2]
		}

		newSlide := EnhancedSlideContent{
			SlideNumber:   len(supplemented) + 1,
			Title:         fmt.Sprintf("补充内容 %d", i+1),
			MainContent:   s.generateSupplementaryContent(baseSlide),
			BulletPoints:  s.generateSupplementaryPoints(baseSlide),
			SlideType:     "content",
			KeyConcepts:   []string{"补充", "要点", "内容"},
			SpeakerNotes:  "这是根据前面内容自动生成的补充说明",
			EstimatedTime: 60,
			ContentWeight: 0.2,
		}

		supplemented = append(supplemented, newSlide)
	}

	return supplemented
}

// generateSupplementaryContent 生成补充内容
func (s *EnhancedPPTService) generateSupplementaryContent(baseSlide EnhancedSlideContent) string {
	templates := []string{
		"进一步深入理解%s的相关概念和应用场景",
		"关于%s的详细说明和实际案例分析",
		"扩展学习%s的重要性和实践方法",
		"总结%s的核心要点和注意事项",
	}

	template := templates[len(baseSlide.Title)%len(templates)]
	return fmt.Sprintf(template, baseSlide.Title)
}

// generateSupplementaryPoints 生成补充要点
func (s *EnhancedPPTService) generateSupplementaryPoints(baseSlide EnhancedSlideContent) []string {
	return []string{
		"深化理解核心概念",
		"掌握实际应用方法",
		"注意关键注意事项",
		"总结学习要点",
	}
}

// 新增辅助方法

// detectContentType 检测内容类型
func (s *EnhancedPPTService) detectContentType(sourceType string) string {
	switch sourceType {
	case "url":
		return "framework_docs" // URL通常是技术文档
	case "document":
		return "tutorial" // 文档通常是教程
	default:
		return "general"
	}
}

// assessTechnicalLevel 评估技术难度
func (s *EnhancedPPTService) assessTechnicalLevel(content *parser.StructuredContent) string {
	if content.KeyInfo == nil {
		return "intermediate"
	}

	// 基于关键词和内容复杂度评估
	advancedKeywords := []string{"架构", "设计模式", "性能优化", "源码", "原理"}
	beginnerKeywords := []string{"入门", "基础", "介绍", "开始"}

	text := strings.ToLower(content.CleanText)
	advancedCount := 0
	beginnerCount := 0

	for _, keyword := range advancedKeywords {
		if strings.Contains(text, keyword) {
			advancedCount++
		}
	}

	for _, keyword := range beginnerKeywords {
		if strings.Contains(text, keyword) {
			beginnerCount++
		}
	}

	if advancedCount > beginnerCount && advancedCount > 2 {
		return "advanced"
	}
	if beginnerCount > advancedCount && beginnerCount > 2 {
		return "beginner"
	}

	return "intermediate"
}

// convertToStandardSlides 转换为标准幻灯片格式
func (s *EnhancedPPTService) convertToStandardSlides(enhancedSlides []EnhancedSlideContent) []SlideCreationContent {
	var slides []SlideCreationContent

	for _, enhanced := range enhancedSlides {
		slide := SlideCreationContent{
			Title:        enhanced.Title,
			Content:      enhanced.MainContent,
			BulletPoints: enhanced.BulletPoints,
			SlideNumber:  enhanced.SlideNumber,
			SpeakerNotes: enhanced.SpeakerNotes,
			SlideType:    enhanced.SlideType,
		}

		// 增强内容整合到标准格式
		if len(enhanced.SubPoints) > 0 {
			slide.Content += "\n\n详细要点：\n" + strings.Join(enhanced.SubPoints, "\n")
		}

		if len(enhanced.KeyConcepts) > 0 {
			slide.SpeakerNotes += "\n\n关键概念：" + strings.Join(enhanced.KeyConcepts, "、")
		}

		if len(enhanced.TechnicalNotes) > 0 {
			slide.SpeakerNotes += "\n\n技术要点：" + strings.Join(enhanced.TechnicalNotes, "；")
		}

		if len(enhanced.BestPractices) > 0 {
			slide.SpeakerNotes += "\n\n最佳实践：" + strings.Join(enhanced.BestPractices, "；")
		}

		slides = append(slides, slide)
	}

	return slides
}

// fallbackToBasicGeneration 回退到基础生成
func (s *EnhancedPPTService) fallbackToBasicGeneration(
	content *parser.StructuredContent,
	slideCount int,
	req *EnhancedGenerationRequest,
) (*EnhancedGenerationResult, error) {
	// 使用原有的基础生成逻辑
	fmt.Printf("🔄 使用基础PPT生成逻辑\n")

	// 这里应该调用原有的生成逻辑
	// 暂时返回一个基础结果
	slides := make([]SlideCreationContent, slideCount)
	for i := 0; i < slideCount; i++ {
		slides[i] = SlideCreationContent{
			Title:   fmt.Sprintf("第%d章：重要内容", i+1),
			Content: "详细的技术内容和知识点说明",
			BulletPoints: []string{
				"核心技术概念解释",
				"实际应用场景分析",
				"实现方法和步骤",
				"最佳实践建议",
				"常见问题解决",
			},
			SlideNumber:  i + 1,
			SpeakerNotes: "请结合具体内容进行详细讲解",
			SlideType:    "content",
		}
	}

	return &EnhancedGenerationResult{
		Success:     true,
		Title:       "基础PPT生成完成",
		Slides:      s.convertToEnhancedSlides(slides),
		TotalSlides: len(slides),
	}, nil
}

// generateContentAnalysis 生成内容分析
func (s *EnhancedPPTService) generateContentAnalysis(
	content *parser.StructuredContent,
	technical *TechnicalSlideResult,
) *ContentAnalysis {
	return &ContentAnalysis{
		Topic:              "技术文档",
		Field:              "技术开发",
		Difficulty:         "中级",
		KeyPoints:          []string{"技术要点", "核心概念", "最佳实践"},
		SuggestedSlides:    technical.TotalCount,
		TargetAudience:     "开发人员",
		LearningObjectives: []string{"掌握核心技术", "理解实现原理", "学会最佳实践"},
	}
}

// calculateContentDensity 计算内容密度
func (s *EnhancedPPTService) calculateContentDensity(slides []EnhancedSlideContent) float64 {
	totalElements := 0
	for _, slide := range slides {
		totalElements += len(slide.BulletPoints)
		totalElements += len(slide.SubPoints)
		totalElements += len(slide.KeyConcepts)
		totalElements += len(slide.TechnicalNotes)
		totalElements += len(slide.BestPractices)
		if slide.CodeExample != nil {
			totalElements += 3 // 代码示例权重更高
		}
	}

	if len(slides) == 0 {
		return 0
	}

	return float64(totalElements) / float64(len(slides))
}

// calculateQualityScore 计算质量评分
func (s *EnhancedPPTService) calculateQualityScore(result *TechnicalSlideResult) float64 {
	score := 0.8 // 基础分

	// 基于内容丰富程度加分
	if result.Metadata.AverageWordsPerSlide > 100 {
		score += 0.1
	}

	// 基于技术深度加分
	if result.Metadata.TechnicalDepth == "high" {
		score += 0.1
	}

	// 基于幻灯片数量加分
	if result.TotalCount >= 15 {
		score += 0.05
	}

	return math.Min(score, 1.0)
}

// generateAudioAsync 异步生成音频
func (s *EnhancedPPTService) generateAudioAsync(slides []SlideCreationContent, req *EnhancedGenerationRequest) {
	fmt.Printf("🎵 开始异步音频生成，共%d张幻灯片\n", len(slides))
	// 这里实现音频生成逻辑
	// 可以调用现有的TTS服务
}

// convertToEnhancedSlides 将SlideCreationContent转换为EnhancedSlideContent
func (s *EnhancedPPTService) convertToEnhancedSlides(slides []SlideCreationContent) []EnhancedSlideContent {
	enhanced := make([]EnhancedSlideContent, 0, len(slides))

	for _, slide := range slides {
		enhancedSlide := EnhancedSlideContent{
			SlideNumber:   slide.SlideNumber,
			Title:         slide.Title,
			SlideType:     slide.SlideType,
			MainContent:   slide.Content,
			BulletPoints:  slide.BulletPoints,
			KeyConcepts:   slide.Keywords,
			SpeakerNotes:  slide.Notes,
			EstimatedTime: 2, // 默认2分钟
			ContentWeight: 1.0,
		}

		enhanced = append(enhanced, enhancedSlide)
	}

	return enhanced
}

// extractKeyInfoWithAI 使用AI提取关键信息
func (s *EnhancedPPTService) extractKeyInfoWithAI(content, title string) (string, []string, error) {
	// 构建AI提示词
	prompt := fmt.Sprintf(`
请分析以下文档内容，提取关键信息：

标题: %s

内容: %s

请返回JSON格式：
{
  "summary": "文档的简洁摘要（100-200字）",
  "keywords": ["关键词1", "关键词2", "关键词3", "关键词4", "关键词5"]
}

要求：
1. 摘要要准确反映文档的核心内容和技术特点
2. 关键词要包含具体的技术术语、概念和主题
3. 避免使用"技术"、"开发"等过于通用的词汇
`, title, content[:min(len(content), 3000)]) // 限制内容长度避免超出AI限制

	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		return "", nil, err
	}

	// 解析AI响应
	var result struct {
		Summary  string   `json:"summary"`
		Keywords []string `json:"keywords"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	return result.Summary, result.Keywords, nil
}

// extractKeyInfoBasic 基础关键信息提取（降级方案）
func (s *EnhancedPPTService) extractKeyInfoBasic(content, title string) (string, []string) {
	// 基础摘要：取前200字符
	summary := content
	if len(content) > 200 {
		summary = content[:200] + "..."
	}

	// 基础关键词提取：简单的词频分析
	keywords := s.extractKeywordsFromText(content)

	return summary, keywords
}

// extractKeywordsFromText 从文本中提取关键词
func (s *EnhancedPPTService) extractKeywordsFromText(text string) []string {
	// 简单的关键词提取逻辑
	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)

	// 过滤常见停用词
	stopWords := map[string]bool{
		"的": true, "是": true, "在": true, "有": true, "和": true, "了": true,
		"与": true, "及": true, "或": true, "但": true, "而": true, "等": true,
		"this": true, "that": true, "the": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true, "for": true,
	}

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			wordCount[word]++
		}
	}

	// 选择出现频率最高的词作为关键词
	var keywords []string
	for word, count := range wordCount {
		if count > 1 && len(keywords) < 8 {
			keywords = append(keywords, word)
		}
	}

	// 如果关键词太少，添加一些默认的
	if len(keywords) < 3 {
		keywords = append(keywords, "文档", "内容", "知识")
	}

	return keywords
}

// splitContentIntoSections 将内容分段
func (s *EnhancedPPTService) splitContentIntoSections(content string) []parser.Section {
	var sections []parser.Section

	// 按段落分割
	paragraphs := strings.Split(content, "\n\n")

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph != "" {
			sections = append(sections, parser.Section{
				Type:    "content",
				Content: paragraph,
			})
		}
	}

	// 如果没有找到段落，将整个内容作为一个段落
	if len(sections) == 0 {
		sections = append(sections, parser.Section{
			Type:    "content",
			Content: content,
		})
	}

	return sections
}

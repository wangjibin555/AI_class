package services

import (
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/parser"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// PPTGenerationService PPT生成服务
type PPTGenerationService struct {
	aiClient        *ai.DashScopeClient
	templates       map[string]*Template
	pptDir          string                    // PPT文件存储目录
	fileProcessor   *parser.FileProcessor     // 文件处理器
	keywordService  *KeywordExtractionService // ✅ 关键词提取服务
	currentKeywords []string                  // ✅ 当前处理的全局关键词
}

// Template PPT模板
type Template struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Style       string            `json:"style"`
	Colors      map[string]string `json:"colors"`
	Fonts       map[string]string `json:"fonts"`
	Layouts     []string          `json:"layouts"`
}

// GenerationParams 生成参数
type GenerationParams struct {
	Template       string   `json:"template"`        // 模板类型
	SlideCount     int      `json:"slide_count"`     // 幻灯片数量
	MaxSlideCount  int      `json:"max_slide_count"` // 最大幻灯片数量限制
	MinSlideCount  int      `json:"min_slide_count"` // 最小幻灯片数量限制
	VoiceType      string   `json:"voice_type"`      // 语音类型
	Language       string   `json:"language"`        // 语言
	Style          string   `json:"style"`           // 风格
	Keywords       []string `json:"keywords"`        // 关键词
	MaxTokens      int      `json:"max_tokens"`      // 最大token数
	Temperature    float64  `json:"temperature"`     // 温度参数
	Audience       string   `json:"audience"`        // 目标受众
	Difficulty     string   `json:"difficulty"`      // 难度等级
	SourceType     string   `json:"source_type"`     // 来源类型: url, file, text
	ContentLength  int      `json:"content_length"`  // 内容长度
	EstimatedPages int      `json:"estimated_pages"` // 预估页数
}

// SlideContent 幻灯片内容
type SlideContent struct {
	Title           string   `json:"title"`            // 标题
	Content         []string `json:"content"`          // 内容（改为数组）
	BulletPoints    []string `json:"bullet_points"`    // 要点列表
	SlideNumber     int      `json:"slide_number"`     // 幻灯片编号
	SlideType       string   `json:"slide_type"`       // 幻灯片类型: title, content, summary
	Keywords        []string `json:"keywords"`         // 关键词
	ImageSuggestion string   `json:"image_suggestion"` // 图片建议
	Notes           string   `json:"notes"`            // 备注

	// 新增字体自适应相关字段
	CharCount    int     `json:"char_count"`    // 字符数量
	FontSize     int     `json:"font_size"`     // 建议字体大小
	LineHeight   float64 `json:"line_height"`   // 建议行高
	NeedSplit    bool    `json:"need_split"`    // 是否需要分页
	ContentClass string  `json:"content_class"` // 内容样式类名
}

// GenerationResult 生成结果
type GenerationResult struct {
	Title          string          `json:"title"`           // 演示标题
	Slides         []SlideContent  `json:"slides"`          // 幻灯片内容
	TotalSlides    int             `json:"total_slides"`    // 总幻灯片数
	GenerationTime time.Duration   `json:"generation_time"` // 生成耗时
	Template       string          `json:"template"`        // 使用的模板
	Error          string          `json:"error"`           // 错误信息
	KeyInfo        *parser.KeyInfo `json:"key_info"`        // 关键信息
	PPTFilePath    string          `json:"ppt_file_path"`   // PPT文件路径
	PPTFileURL     string          `json:"ppt_file_url"`    // PPT文件URL
}

// NewPPTGenerationService 创建PPT生成服务实例
func NewPPTGenerationService(aiClient *ai.DashScopeClient, keywordService *KeywordExtractionService) *PPTGenerationService {
	service := &PPTGenerationService{
		aiClient:       aiClient,
		templates:      make(map[string]*Template),
		pptDir:         "ppt",          // 默认PPT存储目录
		keywordService: keywordService, // ✅ 初始化关键词服务
	}

	// 确保PPT目录存在
	if err := os.MkdirAll(service.pptDir, 0755); err != nil {
		fmt.Printf("创建PPT目录失败: %v\n", err)
	}

	// 初始化模板
	service.initTemplates()

	return service
}

// initTemplates 初始化模板
func (s *PPTGenerationService) initTemplates() {
	// 商务风格模板
	s.templates["business"] = &Template{
		Name:        "商务风格",
		Description: "专业的商务演示模板，适合商业报告和商务会议",
		Style:       "professional",
		Colors: map[string]string{
			"primary":    "#1e3a8a",
			"secondary":  "#64748b",
			"accent":     "#3b82f6",
			"background": "#ffffff",
		},
		Fonts: map[string]string{
			"title": "Microsoft YaHei",
			"body":  "SimSun",
		},
		Layouts: []string{"title", "content", "two-column", "image-text"},
	}

	// 教育风格模板
	s.templates["education"] = &Template{
		Name:        "教育风格",
		Description: "清新的教育演示模板，适合教学和培训",
		Style:       "clean",
		Colors: map[string]string{
			"primary":    "#059669",
			"secondary":  "#6b7280",
			"accent":     "#10b981",
			"background": "#f9fafb",
		},
		Fonts: map[string]string{
			"title": "Microsoft YaHei",
			"body":  "SimSun",
		},
		Layouts: []string{"title", "content", "image-text", "question"},
	}

	// 科技风格模板
	s.templates["tech"] = &Template{
		Name:        "科技风格",
		Description: "现代科技风格模板，适合技术演示和产品介绍",
		Style:       "modern",
		Colors: map[string]string{
			"primary":    "#1f2937",
			"secondary":  "#9ca3af",
			"accent":     "#6366f1",
			"background": "#111827",
		},
		Fonts: map[string]string{
			"title": "Microsoft YaHei",
			"body":  "Consolas",
		},
		Layouts: []string{"title", "content", "code", "diagram"},
	}

	// 简约风格模板
	s.templates["minimal"] = &Template{
		Name:        "简约风格",
		Description: "极简风格模板，突出内容本身",
		Style:       "minimal",
		Colors: map[string]string{
			"primary":    "#000000",
			"secondary":  "#666666",
			"accent":     "#333333",
			"background": "#ffffff",
		},
		Fonts: map[string]string{
			"title": "Microsoft YaHei",
			"body":  "SimSun",
		},
		Layouts: []string{"title", "content", "image-only"},
	}
}

// GenerateSlides 生成幻灯片
func (s *PPTGenerationService) GenerateSlides(content string, params GenerationParams) (*GenerationResult, error) {
	startTime := time.Now()

	// ✅ 0. 全局关键词提取（新增）
	fmt.Printf("\n🌍 开始全局关键词提取 (PPT生成服务)\n")
	var globalKeywords []string
	if s.keywordService != nil {
		fmt.Printf("🔧 关键词服务已初始化，准备提取全局关键词\n")
		keywordOptions := &ExtractionOptions{
			MaxKeywords: 25,
			ContentType: "academic",
			UseAI:       true,
		}
		fmt.Printf("⚙️ 全局关键词提取配置: MaxKeywords=%d, ContentType=%s, UseAI=%t\n",
			keywordOptions.MaxKeywords, keywordOptions.ContentType, keywordOptions.UseAI)

		keywordResult, err := s.keywordService.ExtractKeywords(content, keywordOptions)
		if err != nil {
			fmt.Printf("❌ 全局关键词提取失败: %v\n", err)
			globalKeywords = []string{} // 使用空关键词列表
		} else {
			globalKeywords = keywordResult.All
			fmt.Printf("✅ 全局关键词提取成功 (%d个): %v\n", len(globalKeywords), globalKeywords)

			// 显示分类关键词
			fmt.Printf("🎯 主要关键词: %v\n", keywordResult.Primary)
			fmt.Printf("🔸 次要关键词: %v\n", keywordResult.Secondary)
			fmt.Printf("🔧 技术关键词: %v\n", keywordResult.Technical)
			fmt.Printf("📚 学术关键词: %v\n", keywordResult.Academic)
			fmt.Printf("📈 置信度: %.2f\n", keywordResult.Confidence)
		}
	} else {
		fmt.Printf("⚠️ 关键词服务未初始化，跳过全局关键词提取\n")
		globalKeywords = []string{}
	}

	// 1. 内容分析和结构化
	structuredContent, err := s.analyzeContent(content)
	if err != nil {
		return nil, fmt.Errorf("内容分析失败: %w", err)
	}

	// ✅ 存储全局关键词供其他方法使用
	fmt.Printf("\n📦 存储全局关键词到服务实例\n")
	s.currentKeywords = globalKeywords
	fmt.Printf("✅ 已存储 %d 个全局关键词: %v\n", len(s.currentKeywords), s.currentKeywords)
	fmt.Printf("═══════════════════════════════════════\n")

	// 2. 生成PPT大纲（使用全局关键词）
	outline, err := s.generateOutline(structuredContent, params)
	if err != nil {
		return nil, fmt.Errorf("生成大纲失败: %w", err)
	}

	// 3. 生成每页内容（使用全局关键词）
	slides, err := s.generateSlides(outline, params)
	if err != nil {
		return nil, fmt.Errorf("生成幻灯片失败: %w", err)
	}

	// 4. 应用模板样式
	result := s.applyTemplate(slides, params.Template)

	// 5. 生成PPT文件
	pptFilePath, pptFileURL, err := s.generatePPTFile(&GenerationResult{
		Title:       "AI生成的演示文稿", // 可以从内容中提取标题
		Slides:      result,
		TotalSlides: len(result),
		Template:    params.Template,
	}, params)
	if err != nil {
		// 即使文件生成失败，也返回内容结果
		fmt.Printf("PPT文件生成失败: %v\n", err)
	}

	generationTime := time.Since(startTime)

	return &GenerationResult{
		Title:          "AI生成的演示文稿", // 可以从内容中提取标题
		Slides:         result,
		TotalSlides:    len(result),
		GenerationTime: generationTime,
		Template:       params.Template,
		KeyInfo:        structuredContent.KeyInfo,
		PPTFilePath:    pptFilePath,
		PPTFileURL:     pptFileURL,
	}, nil
}

// GenerateFromSummary 基于总结生成PPT
func (s *PPTGenerationService) GenerateFromSummary(
	summaryResult *SummaryResult,
	params GenerationParams,
) (*GenerationResult, error) {
	startTime := time.Now()

	var slides []SlideContent

	// 1. 生成标题页
	titleSlide := SlideContent{
		SlideNumber:     1,
		Title:           summaryResult.MainTopic,
		Content:         []string{summaryResult.Summary},
		SlideType:       "title",
		Notes:           fmt.Sprintf("欢迎大家学习关于%s的内容。%s", summaryResult.MainTopic, summaryResult.Summary),
		Keywords:        []string{summaryResult.MainTopic},
		ImageSuggestion: fmt.Sprintf("关于%s的主题演示封面", summaryResult.MainTopic),
	}
	slides = append(slides, titleSlide)

	// 2. 为每个章节生成幻灯片
	slideNumber := 2
	for _, section := range summaryResult.Structure {
		sectionSlides, err := s.generateSectionSlides(section, params, slideNumber)
		if err != nil {
			// 如果生成失败，创建基本幻灯片
			basicSlide := SlideContent{
				SlideNumber:     slideNumber,
				Title:           section.Title,
				BulletPoints:    section.KeyPoints,
				Content:         []string{strings.Join(section.KeyPoints, "\n")},
				SlideType:       "content",
				Notes:           s.generateSectionNotes(section),
				Keywords:        section.KeyPoints,
				ImageSuggestion: fmt.Sprintf("关于%s的内容展示", section.Title),
			}
			slides = append(slides, basicSlide)
			slideNumber++
		} else {
			slides = append(slides, sectionSlides...)
			slideNumber += len(sectionSlides)
		}
	}

	// 3. 生成总结页
	summarySlide := SlideContent{
		SlideNumber:     slideNumber,
		Title:           "总结",
		BulletPoints:    summaryResult.KeyPoints,
		Content:         []string{strings.Join(summaryResult.KeyPoints, "\n")},
		SlideType:       "summary",
		Notes:           "让我们来总结今天学习的主要内容：" + strings.Join(summaryResult.KeyPoints, "；"),
		Keywords:        summaryResult.KeyPoints,
		ImageSuggestion: "总结页面，展示主要要点和结论",
	}
	slides = append(slides, summarySlide)

	// 4. 应用模板样式
	styledSlides := s.applyAdvancedTemplate(slides, params.Template, summaryResult.Metadata)

	// 5. 生成PPT文件
	pptFilePath, pptFileURL, err := s.generatePPTFile(&GenerationResult{
		Title:       summaryResult.MainTopic, // 使用总结的主题作为标题
		Slides:      styledSlides,
		TotalSlides: len(styledSlides),
		Template:    params.Template,
	}, params)
	if err != nil {
		// 即使文件生成失败，也返回内容结果
		fmt.Printf("PPT文件生成失败: %v\n", err)
	}

	generationTime := time.Since(startTime)

	return &GenerationResult{
		Title:          summaryResult.MainTopic, // 使用总结的主题作为标题
		Slides:         styledSlides,
		TotalSlides:    len(styledSlides),
		GenerationTime: generationTime,
		Template:       params.Template,
		PPTFilePath:    pptFilePath,
		PPTFileURL:     pptFileURL,
	}, nil
}

// generateSectionSlides 生成章节幻灯片
func (s *PPTGenerationService) generateSectionSlides(
	section SectionInfo,
	params GenerationParams,
	startSlideNumber int,
) ([]SlideContent, error) {
	var slides []SlideContent

	// 根据章节重要性和内容复杂度决定幻灯片数量
	slideCount := section.SlideCount
	if slideCount <= 0 {
		slideCount = 1
	}

	// 如果只需要一张幻灯片，直接生成
	if slideCount == 1 {
		slide := SlideContent{
			SlideNumber:  startSlideNumber,
			Title:        section.Title,
			BulletPoints: section.KeyPoints,
			Content:      []string{strings.Join(section.KeyPoints, "\n")},
			SlideType:    "content",
			Notes:        s.generateSectionNotes(section),
		}
		slides = append(slides, slide)
		return slides, nil
	}

	// 如果需要多张幻灯片，将要点分配到不同幻灯片
	pointsPerSlide := len(section.KeyPoints) / slideCount
	if pointsPerSlide == 0 {
		pointsPerSlide = 1
	}

	for i := 0; i < slideCount; i++ {
		slideNumber := startSlideNumber + i

		// 计算该幻灯片包含的要点范围
		startIdx := i * pointsPerSlide
		endIdx := startIdx + pointsPerSlide
		if i == slideCount-1 { // 最后一张幻灯片包含剩余所有要点
			endIdx = len(section.KeyPoints)
		}
		if endIdx > len(section.KeyPoints) {
			endIdx = len(section.KeyPoints)
		}

		var slidePoints []string
		if startIdx < len(section.KeyPoints) {
			slidePoints = section.KeyPoints[startIdx:endIdx]
		}

		// 生成幻灯片标题
		slideTitle := section.Title
		if slideCount > 1 {
			slideTitle = fmt.Sprintf("%s（%d）", section.Title, i+1)
		}

		slide := SlideContent{
			SlideNumber:  slideNumber,
			Title:        slideTitle,
			BulletPoints: slidePoints,
			Content:      []string{strings.Join(slidePoints, "\n")},
			SlideType:    "content",
			Notes:        s.generateSlideNotes(slideTitle, slidePoints),
		}

		slides = append(slides, slide)
	}

	return slides, nil
}

// generateSectionNotes 生成章节备注
func (s *PPTGenerationService) generateSectionNotes(section SectionInfo) string {
	notes := fmt.Sprintf("在这个部分，我们将重点介绍%s的相关内容。", section.Title)

	if section.Content != "" {
		notes += section.Content
	}

	if len(section.KeyPoints) > 0 {
		notes += "主要包括：" + strings.Join(section.KeyPoints, "、") + "。"
	}

	return notes
}

// generateSlideNotes 生成幻灯片备注
func (s *PPTGenerationService) generateSlideNotes(title string, points []string) string {
	notes := fmt.Sprintf("接下来我们来学习%s。", title)

	if len(points) > 0 {
		notes += "重点内容包括：" + strings.Join(points, "、") + "。"
	}

	return notes
}

// applyTemplate 应用模板样式
func (s *PPTGenerationService) applyTemplate(slides []SlideContent, templateName string) []SlideContent {
	template, exists := s.templates[templateName]
	if !exists {
		template = s.templates["business"] // 默认使用商务模板
	}

	// 为每个幻灯片应用模板样式
	for i := range slides {
		slides[i].SlideType = s.selectLayout(template, i)
		slides[i].ImageSuggestion = s.generateImagePrompt(slides[i], template)
	}

	return slides
}

// applyAdvancedTemplate 应用高级模板样式
func (s *PPTGenerationService) applyAdvancedTemplate(slides []SlideContent, templateName string, metadata ContentMetadata) []SlideContent {
	// 获取模板
	template, exists := s.templates[templateName]
	if !exists {
		// 基于内容类型智能选择模板
		template = s.selectOptimalTemplate(metadata)
	}

	// 为每个幻灯片应用模板样式
	for i := range slides {
		slides[i].SlideType = s.selectAdvancedLayout(template, i, slides[i])
		slides[i].ImageSuggestion = s.generateAdvancedImagePrompt(slides[i], template, metadata)
	}

	return slides
}

// selectOptimalTemplate 智能选择最优模板
func (s *PPTGenerationService) selectOptimalTemplate(metadata ContentMetadata) *Template {
	// 基于内容类型选择模板
	switch metadata.ContentType {
	case "技术":
		return s.templates["tech"]
	case "商业":
		return s.templates["business"]
	case "教育":
		return s.templates["education"]
	case "科学":
		return s.templates["education"] // 科学内容使用教育模板
	default:
		// 根据难度选择
		if metadata.Difficulty == "高级" {
			return s.templates["business"] // 高级内容使用商务模板
		}
		return s.templates["minimal"] // 默认使用简约模板
	}
}

// selectAdvancedLayout 选择高级布局
func (s *PPTGenerationService) selectAdvancedLayout(template *Template, slideIndex int, slide SlideContent) string {
	// 根据幻灯片类型和内容选择布局
	if slide.SlideType != "" {
		return slide.SlideType // 如果已有布局，保持不变
	}

	if slideIndex == 0 {
		return "title" // 第一张是标题页
	}

	// 根据内容要点数量选择布局
	pointCount := len(slide.BulletPoints)
	if pointCount <= 2 {
		return "simple"
	} else if pointCount <= 4 {
		return "content"
	} else {
		return "detailed"
	}
}

// generateImagePrompt 生成图片提示
func (s *PPTGenerationService) generateImagePrompt(slide SlideContent, template *Template) string {
	// 基于幻灯片标题和模板风格生成图片提示
	style := template.Style
	title := slide.Title

	switch style {
	case "professional":
		return fmt.Sprintf("Professional business presentation slide about %s, clean design, blue theme", title)
	case "clean":
		return fmt.Sprintf("Clean educational slide about %s, green theme, modern design", title)
	case "modern":
		return fmt.Sprintf("Modern tech presentation slide about %s, dark theme, futuristic design", title)
	case "minimal":
		return fmt.Sprintf("Minimalist slide about %s, white background, simple design", title)
	default:
		return fmt.Sprintf("Presentation slide about %s", title)
	}
}

// generateAdvancedImagePrompt 生成高级图片提示
func (s *PPTGenerationService) generateAdvancedImagePrompt(slide SlideContent, template *Template, metadata ContentMetadata) string {
	style := template.Style
	title := slide.Title
	contentType := metadata.ContentType

	// 基于内容类型和模板风格生成更精确的图片提示
	basePrompt := fmt.Sprintf("%s presentation slide about '%s'", contentType, title)

	switch style {
	case "professional":
		return fmt.Sprintf("%s, professional business style, blue theme, clean modern design", basePrompt)
	case "clean":
		return fmt.Sprintf("%s, clean educational style, green theme, friendly and approachable", basePrompt)
	case "modern":
		return fmt.Sprintf("%s, modern tech style, dark theme, futuristic and innovative", basePrompt)
	case "minimal":
		return fmt.Sprintf("%s, minimalist style, white background, simple and elegant", basePrompt)
	default:
		return fmt.Sprintf("%s, presentation style", basePrompt)
	}
}

// analyzeContent 分析内容
func (s *PPTGenerationService) analyzeContent(content string) (*parser.StructuredContent, error) {
	// 使用文档解析器分析内容
	docParser := parser.NewDocumentParser()
	return docParser.StructureContent(content)
}

// generateOutline 生成PPT大纲
func (s *PPTGenerationService) generateOutline(content *parser.StructuredContent, params GenerationParams) ([]string, error) {
	// 构建大纲生成提示词
	prompt := s.buildOutlinePrompt(content, params)

	// 调用AI生成大纲
	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		return nil, err
	}

	// 解析大纲
	var outline []string
	if err := json.Unmarshal([]byte(response), &outline); err != nil {
		// 如果JSON解析失败，尝试手动解析
		outline = s.parseOutlineManually(response)
	}

	return outline, nil
}

// generateSlides 生成幻灯片内容
func (s *PPTGenerationService) generateSlides(outline []string, params GenerationParams) ([]SlideContent, error) {
	var slides []SlideContent

	for i, title := range outline {
		// 构建幻灯片生成提示词
		prompt := s.buildSlidePrompt(title, params, i+1)

		// 调用AI生成幻灯片内容
		response, err := s.aiClient.GenerateContent(prompt)
		if err != nil {
			continue
		}

		// 解析幻灯片内容
		slide, err := s.parseSlideContent(response, i+1)
		if err != nil {
			continue
		}

		slides = append(slides, *slide)
	}

	return slides, nil
}

// buildOutlinePrompt 构建大纲生成提示词
func (s *PPTGenerationService) buildOutlinePrompt(content *parser.StructuredContent, params GenerationParams) string {
	tmpl := `
你是一个专业的PPT制作专家。请严格基于以下具体文档内容生成{{.SlideCount}}张幻灯片的标题大纲：

【重要】必须基于以下实际文档内容，不要生成通用模板：

文档摘要：{{.Summary}}
核心要点：{{.MainPoints}}
关键技术词汇：{{.Keywords}}
目标受众：{{.Audience}}
技术难度：{{.Difficulty}}

【严格要求】：
1. 标题必须反映文档的具体内容，不能是通用的"课程大纲"、"学习目标"、"课程介绍"等
2. 如果是技术文档，标题应包含具体的技术概念和知识点
3. 如果是编程语言文档，标题应体现该语言的特定特性和语法
4. 避免使用"主要章节"、"学习路径"、"知识体系"、"时间安排"等通用词汇
5. 每个标题都应该是文档中实际存在的主题或概念
6. 标题应具体到技术细节，而不是抽象概念

请以JSON数组格式返回标题列表：
["具体标题1", "具体标题2", "具体标题3", ...]
`

	// ✅ 合并内容关键词和全局关键词
	allKeywords := content.KeyInfo.Keywords
	if len(s.currentKeywords) > 0 {
		// 去重合并关键词
		keywordSet := make(map[string]bool)
		for _, kw := range content.KeyInfo.Keywords {
			keywordSet[kw] = true
		}
		for _, kw := range s.currentKeywords {
			if !keywordSet[kw] {
				allKeywords = append(allKeywords, kw)
			}
		}
	}

	data := struct {
		SlideCount int
		Summary    string
		MainPoints string
		Keywords   string
		Audience   string
		Difficulty string
	}{
		SlideCount: params.SlideCount,
		Summary:    content.KeyInfo.Summary,
		MainPoints: strings.Join(content.KeyInfo.MainPoints, "；"),
		Keywords:   strings.Join(allKeywords, "、"), // ✅ 使用合并后的关键词
		Audience:   params.Audience,
		Difficulty: params.Difficulty,
	}

	var buf strings.Builder
	t := template.Must(template.New("outline").Parse(tmpl))
	t.Execute(&buf, data)

	return buf.String()
}

// buildSlidePrompt 构建幻灯片生成提示词 - 增强版
func (s *PPTGenerationService) buildSlidePrompt(title string, params GenerationParams, slideNumber int) string {
	tmpl := `
你是一个专业的技术培训师和PPT制作专家。请为第{{.SlideNumber}}张幻灯片生成简洁明了的技术内容：

【重要】请严格基于以下信息生成具体内容，避免通用模板：

幻灯片标题：{{.Title}}
模板风格：{{.Style}}
目标受众：{{.Audience}}
内容类型：技术文档
✅ 参考关键词：{{.Keywords}}

【严格要求 - PPT自适应字体优化】：
1. **简洁明了**：每张幻灯片内容控制在300字以内
2. **结构清晰**：使用标题+要点的形式
3. **要点突出**：每张幻灯片3-6个要点，每个要点15-30字
4. **逻辑分层**：相关内容归类到同一张幻灯片
5. 内容必须与标题和关键词高度相关，不能是通用的"学习要点"、"课程目标"等
6. 每个要点都应该是具体的技术概念、语法特性或实现方法
7. 避免使用"重要要点1"、"重要要点2"等占位符
8. 如果是编程语言，要点应包含具体的语法、特性、函数、方法等
9. ✅ 深度结合参考关键词，确保内容专业准确
10. 优先保证内容精炼，而非详细冗长

请以JSON格式返回简洁内容：
{
  "title": "{{.Title}}",
  "content": "简洁的主要内容描述（50-80字）",
  "bullet_points": ["要点1：核心概念（15-30字）", "要点2：关键特性（15-30字）", "要点3：应用场景（15-30字）", "要点4：注意事项（15-30字）"],
  "keywords": ["关键词1", "关键词2", "关键词3"],
  "speaker_notes": "详细的演讲者备注，包含具体的讲解要点和示例说明（100-150字）",
  "layout": "concise_content",
  "char_count": 250
}
`

	data := struct {
		SlideNumber int
		Title       string
		Style       string
		Audience    string
		Keywords    string // ✅ 添加关键词字段
	}{
		SlideNumber: slideNumber,
		Title:       title,
		Style:       params.Style,
		Audience:    params.Audience,
		Keywords:    strings.Join(s.currentKeywords, "、"), // ✅ 传递全局关键词
	}

	var buf strings.Builder
	t := template.Must(template.New("slide").Parse(tmpl))
	t.Execute(&buf, data)

	return buf.String()
}

// parseSlideContent 解析幻灯片内容
func (s *PPTGenerationService) parseSlideContent(response string, slideNumber int) (*SlideContent, error) {
	// 清理响应文本，提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("无法找到有效的JSON内容")
	}

	jsonContent := response[jsonStart : jsonEnd+1]

	var slideData struct {
		Title          string   `json:"title"`
		Content        string   `json:"content"`
		BulletPoints   []string `json:"bullet_points"`
		SubPoints      []string `json:"sub_points"`
		Keywords       []string `json:"keywords"`
		TechnicalNotes []string `json:"technical_notes"`
		BestPractices  []string `json:"best_practices"`
		CodeExample    string   `json:"code_example"`
		QuickTips      []string `json:"quick_tips"`
		SpeakerNotes   string   `json:"speaker_notes"`
		Layout         string   `json:"layout"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &slideData); err != nil {
		return nil, err
	}

	// 创建增强的幻灯片内容
	content := []string{}
	if slideData.Content != "" {
		content = strings.Split(slideData.Content, "\n")
	}

	slide := &SlideContent{
		Title:        slideData.Title,
		Content:      content,
		BulletPoints: slideData.BulletPoints,
		Keywords:     slideData.Keywords,
		Notes:        slideData.SpeakerNotes,
		SlideType:    slideData.Layout,
		SlideNumber:  slideNumber,
	}

	// 确保内容丰富性 - 如果要点不足，进行补充
	if len(slide.BulletPoints) < 5 {
		additionalPoints := []string{
			"核心技术概念和原理解释",
			"实际应用场景和案例分析",
			"实现方法和操作步骤",
			"最佳实践和优化建议",
			"常见问题和解决方案",
			"相关工具和资源推荐",
		}

		needed := 6 - len(slide.BulletPoints)
		for i := 0; i < needed && i < len(additionalPoints); i++ {
			slide.BulletPoints = append(slide.BulletPoints, additionalPoints[i])
		}
	}

	// 确保关键词数量
	if len(slide.Keywords) < 3 {
		defaultKeywords := []string{"核心技术", "实践方法", "应用场景", "最佳实践", "解决方案"}
		needed := 5 - len(slide.Keywords)
		for i := 0; i < needed && i < len(defaultKeywords); i++ {
			slide.Keywords = append(slide.Keywords, defaultKeywords[i])
		}
	}

	// 确保演讲备注详细
	if slide.Notes == "" || len(slide.Notes) < 50 {
		slide.Notes = fmt.Sprintf("详细讲解%s的核心内容，结合实际案例和代码示例进行说明。重点解释技术原理、实现方法和应用场景，确保学员能够理解并掌握相关技术要点。", slide.Title)
	}

	// 如果有技术注释，添加到备注中
	if len(slideData.TechnicalNotes) > 0 {
		slide.Notes += "\n\n技术要点：" + strings.Join(slideData.TechnicalNotes, "；")
	}

	// 如果有最佳实践，添加到备注中
	if len(slideData.BestPractices) > 0 {
		slide.Notes += "\n\n最佳实践：" + strings.Join(slideData.BestPractices, "；")
	}

	// 如果有代码示例，添加到内容中
	if slideData.CodeExample != "" {
		slide.Content = append(slide.Content, "代码示例：\n"+slideData.CodeExample)
	}

	// 如果有快速提示，添加到备注中
	if len(slideData.QuickTips) > 0 {
		slide.Notes += "\n\n快速提示：" + strings.Join(slideData.QuickTips, "；")
	}

	// 确保标题不为空
	if slide.Title == "" {
		slide.Title = fmt.Sprintf("第%d张幻灯片：技术要点详解", slideNumber)
	}

	// 确保内容不为空
	if len(slide.Content) == 0 {
		slide.Content = []string{"本幻灯片包含详细的技术内容，涵盖核心概念、实现方法、应用场景和最佳实践等多个方面。"}
	}

	return slide, nil
}

// parseOutlineManually 手动解析大纲
func (s *PPTGenerationService) parseOutlineManually(response string) []string {
	// 简单的文本解析，提取标题
	lines := strings.Split(response, "\n")
	var outline []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 移除序号和特殊字符
		line = strings.TrimPrefix(line, "1.")
		line = strings.TrimPrefix(line, "2.")
		line = strings.TrimPrefix(line, "3.")
		line = strings.TrimPrefix(line, "4.")
		line = strings.TrimPrefix(line, "5.")
		line = strings.TrimPrefix(line, "6.")
		line = strings.TrimPrefix(line, "7.")
		line = strings.TrimPrefix(line, "8.")
		line = strings.TrimPrefix(line, "9.")
		line = strings.TrimPrefix(line, "10.")
		line = strings.TrimSpace(line)

		if line != "" && len(line) > 2 {
			outline = append(outline, line)
		}
	}

	return outline
}

// selectLayout 选择布局
func (s *PPTGenerationService) selectLayout(template *Template, slideIndex int) string {
	if len(template.Layouts) == 0 {
		return "content"
	}

	// 根据幻灯片索引选择布局
	layoutIndex := slideIndex % len(template.Layouts)
	return template.Layouts[layoutIndex]
}

// EnhancedGenerationParams 增强型生成参数
type EnhancedGenerationParams struct {
	GenerationParams
	SummaryLevel   string `json:"summary_level"`   // 总结级别
	TargetAudience string `json:"target_audience"` // 目标受众
	IncludeNotes   bool   `json:"include_notes"`   // 是否包含备注
	GenerateAudio  bool   `json:"generate_audio"`  // 是否生成音频
	CustomTemplate string `json:"custom_template"` // 自定义模板
}

// GetTemplates 获取可用模板
func (s *PPTGenerationService) GetTemplates() map[string]*Template {
	return s.templates
}

// GetTemplate 获取指定模板
func (s *PPTGenerationService) GetTemplate(name string) (*Template, bool) {
	template, exists := s.templates[name]
	return template, exists
}

// parsePPTJSONResponse 解析PPT JSON响应
func parsePPTJSONResponse(response string, target interface{}) error {
	// 清理响应文本，提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		// 尝试查找数组格式
		jsonStart = strings.Index(response, "[")
		jsonEnd = strings.LastIndex(response, "]")
	}

	if jsonStart == -1 || jsonEnd == -1 {
		return fmt.Errorf("无法找到有效的JSON内容")
	}

	jsonContent := response[jsonStart : jsonEnd+1]

	// 使用标准库解析JSON
	return json.Unmarshal([]byte(jsonContent), target)
}

// generatePPTFile 生成PPT文件
func (s *PPTGenerationService) generatePPTFile(result *GenerationResult, params GenerationParams) (string, string, error) {
	// 确保PPT目录存在
	if err := os.MkdirAll(s.pptDir, 0755); err != nil {
		return "", "", fmt.Errorf("创建PPT目录失败: %w", err)
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("ppt_%s_%s.html", timestamp, params.Template)
	filepath := filepath.Join(s.pptDir, filename)

	// 生成HTML格式的PPT
	htmlContent, err := s.generateHTMLPPT(result, params)
	if err != nil {
		return "", "", fmt.Errorf("生成HTML PPT失败: %w", err)
	}

	// 保存HTML文件
	if err := os.WriteFile(filepath, []byte(htmlContent), 0644); err != nil {
		return "", "", fmt.Errorf("保存PPT文件失败: %w", err)
	}

	// 生成访问URL
	fileURL := fmt.Sprintf("/api/v1/ppt/download/%s", filename)

	fmt.Printf("PPT文件生成成功: %s\n", filepath)

	return filepath, fileURL, nil
}

// generateHTMLPPT 生成HTML格式的PPT
func (s *PPTGenerationService) generateHTMLPPT(result *GenerationResult, params GenerationParams) (string, error) {
	// HTML模板
	htmlTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: 'Microsoft YaHei', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .presentation-container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        .slide {
            background: white;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.3);
            margin-bottom: 30px;
            overflow: hidden;
            page-break-after: always;
        }
        .slide-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .slide-title {
            font-size: 2.5em;
            font-weight: bold;
            margin: 0;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        .slide-subtitle {
            font-size: 1.2em;
            margin-top: 10px;
            opacity: 0.9;
        }
        .slide-content {
            padding: 40px;
            font-size: 1.1em;
            line-height: 1.6;
        }
        .slide-content h2 {
            color: #333;
            border-bottom: 3px solid #667eea;
            padding-bottom: 10px;
            margin-bottom: 20px;
        }
        .slide-content ul {
            list-style: none;
            padding: 0;
        }
        .slide-content li {
            background: #f8f9fa;
            margin: 10px 0;
            padding: 15px 20px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            position: relative;
        }
        .slide-content li:before {
            content: "•";
            color: #667eea;
            font-weight: bold;
            position: absolute;
            left: 10px;
        }
        .slide-notes {
            background: #f8f9fa;
            border-top: 1px solid #dee2e6;
            padding: 20px 40px;
            font-style: italic;
            color: #666;
        }
        .slide-number {
            position: absolute;
            bottom: 20px;
            right: 20px;
            background: rgba(102, 126, 234, 0.8);
            color: white;
            padding: 5px 15px;
            border-radius: 20px;
            font-size: 0.9em;
        }
        .presentation-info {
            background: white;
            border-radius: 15px;
            padding: 20px;
            margin-bottom: 30px;
            text-align: center;
        }
        .info-title {
            font-size: 2em;
            color: #333;
            margin-bottom: 10px;
        }
        .info-meta {
            color: #666;
            font-size: 1.1em;
        }
        @media print {
            body { background: white; }
            .slide { box-shadow: none; margin-bottom: 20px; }
        }
    </style>
</head>
<body>
    <div class="presentation-container">
        <!-- 演示信息 -->
        <div class="presentation-info">
            <h1 class="info-title">{{.Title}}</h1>
            <div class="info-meta">
                <p>生成时间: {{.GeneratedAt}}</p>
                <p>模板: {{.Template}}</p>
                <p>总页数: {{.TotalSlides}}</p>
            </div>
        </div>

        <!-- 幻灯片内容 -->
        {{range $index, $slide := .Slides}}
        <div class="slide">
            <div class="slide-header">
                <h1 class="slide-title">{{$slide.Title}}</h1>
                {{if $slide.Content}}
                <p class="slide-subtitle">{{$slide.Content}}</p>
                {{end}}
            </div>
            
            <div class="slide-content">
                {{if $slide.Content}}
                <div>{{$slide.Content}}</div>
                {{end}}
                
                {{if $slide.BulletPoints}}
                <h2>要点</h2>
                <ul>
                    {{range $slide.BulletPoints}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
                {{end}}
                
                {{if $slide.ImageSuggestion}}
                <h2>图片提示</h2>
                <div class="image-prompt">{{$slide.ImageSuggestion}}</div>
                {{end}}
            </div>
            
            {{if $slide.Notes}}
            <div class="slide-notes">
                <strong>演讲备注:</strong> {{$slide.Notes}}
            </div>
            {{end}}
            
            <div class="slide-number">{{add $index 1}} / {{$.TotalSlides}}</div>
        </div>
        {{end}}
    </div>
</body>
</html>`

	// 准备模板数据
	data := struct {
		Title       string
		GeneratedAt string
		Template    string
		TotalSlides int
		Slides      []SlideContent
	}{
		Title:       result.Title,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Template:    params.Template,
		TotalSlides: len(result.Slides),
		Slides:      result.Slides,
	}

	// 创建模板
	tmpl, err := template.New("ppt").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("解析HTML模板失败: %w", err)
	}

	// 执行模板
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("生成HTML内容失败: %w", err)
	}

	return buf.String(), nil
}

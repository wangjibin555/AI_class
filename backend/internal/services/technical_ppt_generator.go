package services

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/parser"
)

// TechnicalPPTGenerator 技术文档专用PPT生成器
type TechnicalPPTGenerator struct {
	aiClient         *ai.DashScopeClient
	slideCalculator  *SlideCountCalculator
	templateManager  *TechnicalTemplateManager
	contentOptimizer *ContentOptimizer
}

// TechnicalGenerationParams 技术文档生成参数
type TechnicalGenerationParams struct {
	// 基础参数
	ContentType    string `json:"content_type"`    // framework_docs, api_reference, tutorial
	TechnicalLevel string `json:"technical_level"` // beginner, intermediate, advanced
	UserType       string `json:"user_type"`       // regular, premium, vip
	Language       string `json:"language"`        // zh-CN, en-US

	// 内容控制
	MaxSlideCount       int     `json:"max_slide_count"`
	ContentDensity      float64 `json:"content_density"` // 0.5-1.5 内容密度系数
	IncludeCodeExample  bool    `json:"include_code_example"`
	IncludeBestPractice bool    `json:"include_best_practice"`
	IncludeArchitecture bool    `json:"include_architecture"`

	// 生成控制
	DetailLevel     string   `json:"detail_level"`     // high, medium, low
	FocusAreas      []string `json:"focus_areas"`      // concept, practice, code, theory
	GenerationStyle string   `json:"generation_style"` // comprehensive, concise, detailed
}

// TechnicalSlideResult 技术文档幻灯片生成结果
type TechnicalSlideResult struct {
	Slides       []EnhancedSlideContent `json:"slides"`
	TotalCount   int                    `json:"total_count"`
	Distribution *SlideDistribution     `json:"distribution"`
	Metadata     *GenerationMetadata    `json:"metadata"`
}

// EnhancedSlideContent 增强的幻灯片内容
type EnhancedSlideContent struct {
	// 基础信息
	SlideNumber int    `json:"slide_number"`
	Title       string `json:"title"`
	SlideType   string `json:"slide_type"` // intro, concept, tutorial, code, practice, summary

	// 丰富内容
	MainContent  string   `json:"main_content"`  // 主要内容
	BulletPoints []string `json:"bullet_points"` // 要点列表 (3-8个)
	SubPoints    []string `json:"sub_points"`    // 子要点 (每个主要点的详细说明)
	KeyConcepts  []string `json:"key_concepts"`  // 关键概念 (2-5个)

	// 技术元素
	CodeExample    *CodeExample `json:"code_example,omitempty"` // 代码示例
	TechnicalNotes []string     `json:"technical_notes"`        // 技术注释
	BestPractices  []string     `json:"best_practices"`         // 最佳实践要点
	CommonPitfalls []string     `json:"common_pitfalls"`        // 常见陷阱

	// 学习辅助
	Prerequisites   []string `json:"prerequisites"`    // 前置知识
	LearnObjectives []string `json:"learn_objectives"` // 学习目标
	QuickTips       []string `json:"quick_tips"`       // 快速提示
	References      []string `json:"references"`       // 参考资料

	// 展示控制
	ContentWeight   float64 `json:"content_weight"`   // 内容权重 0-1
	EstimatedTime   int     `json:"estimated_time"`   // 预计讲解时间(分钟)
	DifficultyLevel string  `json:"difficulty_level"` // easy, medium, hard

	// 演讲备注
	SpeakerNotes    string   `json:"speaker_notes"`    // 详细的演讲备注
	TransitionNotes string   `json:"transition_notes"` // 过渡说明
	InteractionTips []string `json:"interaction_tips"` // 互动提示
}

// CodeExample 代码示例
type CodeExample struct {
	Language    string   `json:"language"`    // javascript, html, css, json
	Code        string   `json:"code"`        // 代码内容
	Description string   `json:"description"` // 代码说明
	KeyPoints   []string `json:"key_points"`  // 代码要点
	Output      string   `json:"output"`      // 预期输出
	Explanation []string `json:"explanation"` // 逐行解释
}

// GenerationMetadata 生成元数据
type GenerationMetadata struct {
	TotalWords           int            `json:"total_words"`
	AverageWordsPerSlide int            `json:"average_words_per_slide"`
	ConceptCoverage      map[string]int `json:"concept_coverage"` // 概念覆盖统计
	TechnicalDepth       string         `json:"technical_depth"`  // 技术深度评估
	ContentTypes         map[string]int `json:"content_types"`    // 内容类型分布
	Recommendations      []string       `json:"recommendations"`  // 改进建议
}

// TechnicalTemplateManager 技术模板管理器
type TechnicalTemplateManager struct {
	templates map[string]*TechnicalTemplate
}

// TechnicalTemplate 技术文档模板
type TechnicalTemplate struct {
	Name             string            `json:"name"`
	ConceptTemplate  string            `json:"concept_template"`
	CodeTemplate     string            `json:"code_template"`
	TutorialTemplate string            `json:"tutorial_template"`
	PracticeTemplate string            `json:"practice_template"`
	PromptModifiers  map[string]string `json:"prompt_modifiers"`
}

// NewTechnicalPPTGenerator 创建技术PPT生成器
func NewTechnicalPPTGenerator(aiClient *ai.DashScopeClient) *TechnicalPPTGenerator {
	return &TechnicalPPTGenerator{
		aiClient:         aiClient,
		slideCalculator:  NewSlideCountCalculator(),
		templateManager:  NewTechnicalTemplateManager(),
		contentOptimizer: NewContentOptimizer(),
	}
}

// GenerateTechnicalSlides 生成技术文档幻灯片
func (g *TechnicalPPTGenerator) GenerateTechnicalSlides(
	content *parser.StructuredContent,
	params *TechnicalGenerationParams,
) (*TechnicalSlideResult, error) {

	// 1. 计算最优幻灯片数量
	recommendation, err := g.slideCalculator.CalculateOptimalSlideCount(
		content, params.UserType, "url")
	if err != nil {
		return nil, fmt.Errorf("计算幻灯片数量失败: %w", err)
	}

	// 2. 使用推荐数量或用户指定数量
	targetCount := recommendation.Recommended
	if params.MaxSlideCount > 0 && params.MaxSlideCount <= recommendation.Maximum {
		targetCount = params.MaxSlideCount
	}

	// 3. 分析内容结构，生成幻灯片大纲
	outline, err := g.generateTechnicalOutline(content, targetCount, params)
	if err != nil {
		return nil, fmt.Errorf("生成技术大纲失败: %w", err)
	}

	// 4. 为每张幻灯片生成详细内容
	slides, err := g.generateSlidesFromOutline(outline, content, params)
	if err != nil {
		return nil, fmt.Errorf("生成幻灯片内容失败: %w", err)
	}

	// 5. 内容优化和质量控制
	optimizedSlides := g.optimizeSlideContent(slides, params)

	// 6. 生成元数据和统计信息
	metadata := g.generateMetadata(optimizedSlides, content)

	// 7. 在返回结果前添加内容验证
	validator := NewContentValidator()
	validationResult := validator.ValidatePPTContent(optimizedSlides)

	if !validationResult.IsValid {
		log.Printf("⚠️ PPT内容验证失败，模板内容比例: %.2f%%", validationResult.TemplateRate*100)
		for _, issue := range validationResult.Issues {
			log.Printf("❌ 验证问题: %s", issue)
		}
	} else {
		log.Printf("✅ PPT内容验证通过，原创内容比例: %.2f%%", (1-validationResult.TemplateRate)*100)
	}

	return &TechnicalSlideResult{
		Slides:       optimizedSlides,
		TotalCount:   len(optimizedSlides),
		Distribution: recommendation.Distribution,
		Metadata:     metadata,
	}, nil
}

// generateTechnicalOutline 生成技术文档大纲
func (g *TechnicalPPTGenerator) generateTechnicalOutline(
	content *parser.StructuredContent,
	targetCount int,
	params *TechnicalGenerationParams,
) ([]SlideOutline, error) {

	// 构建技术文档专用的大纲生成提示词
	prompt := g.buildTechnicalOutlinePrompt(content, targetCount, params)

	// 调用AI生成大纲
	fmt.Printf("🤖 开始调用AI生成技术大纲，提示词长度: %d 字符\n", len(prompt))
	response, err := g.aiClient.GenerateContent(prompt)
	if err != nil {
		fmt.Printf("❌ AI调用失败: %v\n", err)
		fmt.Printf("⚠️ 降级使用规则生成大纲\n")
		return g.generateOutlineByRules(content, targetCount, params), nil
	}

	fmt.Printf("✅ AI响应成功，响应长度: %d 字符\n", len(response))
	fmt.Printf("📝 AI响应内容预览: %s\n", response[:min(len(response), 500)]+"...")

	// 解析AI响应
	var outlines []SlideOutline
	if err := json.Unmarshal([]byte(response), &outlines); err != nil {
		fmt.Printf("❌ JSON解析失败: %v\n", err)
		fmt.Printf("📄 完整AI响应: %s\n", response)
		fmt.Printf("⚠️ 降级使用规则生成大纲\n")
		return g.generateOutlineByRules(content, targetCount, params), nil
	}

	fmt.Printf("✅ JSON解析成功，生成了 %d 张幻灯片大纲\n", len(outlines))

	return outlines, nil
}

// SlideOutline 幻灯片大纲
type SlideOutline struct {
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	Type         string   `json:"type"`
	MainPoints   []string `json:"main_points"`
	KeyConcepts  []string `json:"key_concepts"`
	ContentFocus string   `json:"content_focus"` // theory, practice, code, concept
	Priority     int      `json:"priority"`      // 1-5
}

// buildTechnicalOutlinePrompt 构建技术大纲生成提示词
func (g *TechnicalPPTGenerator) buildTechnicalOutlinePrompt(
	content *parser.StructuredContent,
	targetCount int,
	params *TechnicalGenerationParams,
) string {

	prompt := fmt.Sprintf(`
你是一个专业的技术文档PPT制作专家。请根据以下技术内容生成%d张幻灯片的详细大纲。

## 内容分析
内容类型: %s
技术难度: %s
内容摘要: %s
关键词: %s

## 生成要求
1. 每张幻灯片都要包含丰富的技术内容
2. 确保知识点的完整性和连贯性
3. 每张幻灯片至少包含3-5个主要技术要点
4. 突出实践性和可操作性
5. 包含代码示例、最佳实践、常见问题等
6. 适合%s难度水平的学习者

## 幻灯片类型分配
- 介绍幻灯片: 2张
- 概念解释幻灯片: %d张 
- 实践教程幻灯片: %d张
- 代码示例幻灯片: %d张
- 最佳实践幻灯片: %d张
- 总结幻灯片: 1张

## 输出格式
请严格按照以下JSON数组格式返回，必须是完整的JSON数组：
[
  {
    "number": 1,
    "title": "幻灯片标题",
    "type": "intro|concept|tutorial|code|practice|summary",
    "main_points": ["要点1", "要点2", "要点3", "要点4", "要点5"],
    "key_concepts": ["概念1", "概念2", "概念3"],
    "content_focus": "theory|practice|code|concept",
    "priority": 1-5
  }
]

【重要】：
1. 必须返回完整的JSON数组，以[开头，以]结尾
2. 不要添加任何解释文字，只返回JSON
3. 确保JSON格式正确，可以被解析
4. 总共%d张幻灯片
`,
		targetCount,
		params.ContentType,
		params.TechnicalLevel,
		g.getSummary(content),
		strings.Join(g.getKeywords(content), ", "),
		params.TechnicalLevel,
		getSlideTypeCount("concept", targetCount),
		getSlideTypeCount("tutorial", targetCount),
		getSlideTypeCount("code", targetCount),
		getSlideTypeCount("practice", targetCount),
		targetCount,
	)

	return prompt
}

// generateSlidesFromOutline 从大纲生成详细幻灯片内容
func (g *TechnicalPPTGenerator) generateSlidesFromOutline(
	outlines []SlideOutline,
	content *parser.StructuredContent,
	params *TechnicalGenerationParams,
) ([]EnhancedSlideContent, error) {

	var slides []EnhancedSlideContent

	for _, outline := range outlines {
		slide, err := g.generateDetailedSlide(outline, content, params)
		if err != nil {
			// 如果生成失败，创建基础幻灯片
			slide = g.createFallbackSlide(outline)
		}
		slides = append(slides, *slide)
	}

	return slides, nil
}

// generateDetailedSlide 生成详细的幻灯片内容
func (g *TechnicalPPTGenerator) generateDetailedSlide(
	outline SlideOutline,
	content *parser.StructuredContent,
	params *TechnicalGenerationParams,
) (*EnhancedSlideContent, error) {

	// 根据幻灯片类型选择模板
	template := g.templateManager.GetTemplate(outline.Type, params.ContentType)

	// 构建详细内容生成提示词
	prompt := g.buildSlideContentPrompt(outline, content, params, template)

	// 调用AI生成详细内容
	response, err := g.aiClient.GenerateContent(prompt)
	if err != nil {
		return nil, err
	}

	// 解析生成的内容
	slide, err := g.parseSlideResponse(response, outline)
	if err != nil {
		return nil, err
	}

	// 后处理优化
	g.postProcessSlide(slide, params)

	return slide, nil
}

// buildSlideContentPrompt 构建幻灯片内容生成提示词
func (g *TechnicalPPTGenerator) buildSlideContentPrompt(
	outline SlideOutline,
	content *parser.StructuredContent,
	params *TechnicalGenerationParams,
	template *TechnicalTemplate,
) string {

	basePrompt := fmt.Sprintf(`
你是一个专业的技术培训师。请为第%d张幻灯片生成详细且丰富的内容。

## 幻灯片信息
标题: %s
类型: %s
主要要点: %s
关键概念: %s

## 内容要求
1. 内容要详细充实，包含大量实用信息
2. 每个要点都要有具体的解释和示例
3. 包含3-8个主要要点，每个要点都有2-3个子要点
4. 添加技术注释、最佳实践、常见陷阱等
5. 确保内容的实用性和可操作性
6. 适合%s水平的学习者

## 输出格式 (JSON)
{
  "slide_number": %d,
  "title": "%s",
  "slide_type": "%s",
  "main_content": "主要内容描述",
  "bullet_points": ["要点1", "要点2", "要点3", "要点4", "要点5"],
  "sub_points": ["子要点1", "子要点2", "子要点3", "子要点4"],
  "key_concepts": ["概念1", "概念2", "概念3"],
  "technical_notes": ["技术注释1", "技术注释2"],
  "best_practices": ["最佳实践1", "最佳实践2"],
  "common_pitfalls": ["常见陷阱1", "常见陷阱2"],
  "quick_tips": ["快速提示1", "快速提示2"],
  "speaker_notes": "详细的演讲备注，包含具体的讲解要点和示例",
  "estimated_time": 3-5分钟
}
`,
		outline.Number,
		outline.Title,
		outline.Type,
		strings.Join(outline.MainPoints, ", "),
		strings.Join(outline.KeyConcepts, ", "),
		params.TechnicalLevel,
		outline.Number,
		outline.Title,
		outline.Type,
	)

	// 根据幻灯片类型添加特定要求
	switch outline.Type {
	case "code":
		basePrompt += g.addCodeSlideRequirements(content)
	case "tutorial":
		basePrompt += g.addTutorialSlideRequirements()
	case "concept":
		basePrompt += g.addConceptSlideRequirements()
	case "practice":
		basePrompt += g.addPracticeSlideRequirements()
	}

	return basePrompt
}

// addCodeSlideRequirements 添加代码幻灯片特殊要求
func (g *TechnicalPPTGenerator) addCodeSlideRequirements(content *parser.StructuredContent) string {
	return `

## 代码幻灯片特殊要求
请在JSON中额外添加:
"code_example": {
  "language": "javascript/html/css/json",
  "code": "实际的代码示例",
  "description": "代码功能说明",
  "key_points": ["代码要点1", "代码要点2"],
  "explanation": ["第1行解释", "第2行解释"]
}
确保代码示例实用、完整且有详细注释。
`
}

// addTutorialSlideRequirements 添加教程幻灯片特殊要求
func (g *TechnicalPPTGenerator) addTutorialSlideRequirements() string {
	return `

## 教程幻灯片特殊要求
- 提供详细的步骤说明（至少5-8个步骤）
- 每个步骤都要有具体的操作指导
- 包含预期结果和可能遇到的问题
- 添加实际操作的截图说明或命令示例
`
}

// addConceptSlideRequirements 添加概念幻灯片特殊要求
func (g *TechnicalPPTGenerator) addConceptSlideRequirements() string {
	return `

## 概念幻灯片特殊要求
- 深入解释技术概念的原理和机制
- 提供具体的应用场景和实例
- 比较相关概念的异同
- 包含概念之间的关联关系
- 添加记忆技巧和理解要点
`
}

// addPracticeSlideRequirements 添加实践幻灯片特殊要求
func (g *TechnicalPPTGenerator) addPracticeSlideRequirements() string {
	return `

## 最佳实践幻灯片特殊要求
- 提供具体的实践指导和建议
- 包含常见错误和解决方案
- 添加性能优化和代码质量提升技巧
- 提供实际项目中的应用示例
- 包含工具推荐和资源链接
`
}

// parseSlideResponse 解析AI生成的幻灯片内容
func (g *TechnicalPPTGenerator) parseSlideResponse(response string, outline SlideOutline) (*EnhancedSlideContent, error) {
	var slide EnhancedSlideContent

	// 首先尝试JSON解析
	if err := json.Unmarshal([]byte(response), &slide); err == nil {
		return &slide, nil
	}

	// 如果JSON解析失败，使用文本解析
	slide = EnhancedSlideContent{
		SlideNumber:  outline.Number,
		Title:        outline.Title,
		SlideType:    outline.Type,
		BulletPoints: outline.MainPoints,
		KeyConcepts:  outline.KeyConcepts,
	}

	// 从响应中提取内容
	slide.MainContent = g.extractMainContent(response)
	slide.SpeakerNotes = g.extractSpeakerNotes(response)
	slide.TechnicalNotes = g.extractTechnicalNotes(response)
	slide.BestPractices = g.extractBestPractices(response)
	slide.EstimatedTime = g.estimateSlideTime(&slide)

	return &slide, nil
}

// Helper functions for content extraction
func (g *TechnicalPPTGenerator) extractMainContent(response string) string {
	// 使用正则表达式提取主要内容
	re := regexp.MustCompile(`(?i)main[_\s]*content["\s]*:["\s]*([^"]+)`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return matches[1]
	}
	return "请根据标题和要点进行详细讲解"
}

func (g *TechnicalPPTGenerator) extractSpeakerNotes(response string) string {
	re := regexp.MustCompile(`(?i)speaker[_\s]*notes["\s]*:["\s]*([^"]+)`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return matches[1]
	}
	return "详细展开各个要点，结合实际案例进行说明"
}

func (g *TechnicalPPTGenerator) extractTechnicalNotes(response string) []string {
	re := regexp.MustCompile(`(?i)technical[_\s]*notes["\s]*:\s*\[(.*?)\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return strings.Split(matches[1], ",")
	}
	return []string{"注意技术细节和最佳实践"}
}

func (g *TechnicalPPTGenerator) extractBestPractices(response string) []string {
	re := regexp.MustCompile(`(?i)best[_\s]*practices["\s]*:\s*\[(.*?)\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return strings.Split(matches[1], ",")
	}
	return []string{"遵循业界最佳实践"}
}

func (g *TechnicalPPTGenerator) estimateSlideTime(slide *EnhancedSlideContent) int {
	wordCount := len(strings.Fields(slide.MainContent))
	pointCount := len(slide.BulletPoints)

	// 基础时间：3分钟
	baseTime := 3

	// 根据内容量调整
	if wordCount > 100 {
		baseTime += 1
	}
	if pointCount > 5 {
		baseTime += 1
	}
	if slide.CodeExample != nil {
		baseTime += 2
	}

	return baseTime
}

// optimizeSlideContent 优化幻灯片内容
func (g *TechnicalPPTGenerator) optimizeSlideContent(slides []EnhancedSlideContent, params *TechnicalGenerationParams) []EnhancedSlideContent {
	for i := range slides {
		// 确保内容密度
		g.ensureContentDensity(&slides[i], params.ContentDensity)

		// 添加学习目标
		g.addLearningObjectives(&slides[i])

		// 优化要点数量和质量
		g.optimizeBulletPoints(&slides[i])

		// 添加过渡说明
		if i > 0 {
			g.addTransitionNotes(&slides[i], &slides[i-1])
		}
	}

	return slides
}

// ensureContentDensity 确保内容密度
func (g *TechnicalPPTGenerator) ensureContentDensity(slide *EnhancedSlideContent, targetDensity float64) {
	currentDensity := g.calculateContentDensity(slide)

	if currentDensity < targetDensity {
		// 增加内容密度
		g.enhanceSlideContent(slide)
	}
}

func (g *TechnicalPPTGenerator) calculateContentDensity(slide *EnhancedSlideContent) float64 {
	score := 0.0

	// 基于各种内容元素计算密度
	score += float64(len(slide.BulletPoints)) * 0.2
	score += float64(len(slide.SubPoints)) * 0.15
	score += float64(len(slide.KeyConcepts)) * 0.2
	score += float64(len(slide.TechnicalNotes)) * 0.15
	score += float64(len(slide.BestPractices)) * 0.15
	score += float64(len(slide.QuickTips)) * 0.1

	if slide.CodeExample != nil {
		score += 0.3
	}

	return score
}

func (g *TechnicalPPTGenerator) enhanceSlideContent(slide *EnhancedSlideContent) {
	// 如果要点不足，补充要点
	if len(slide.BulletPoints) < 4 {
		slide.BulletPoints = append(slide.BulletPoints,
			"重要知识点补充",
			"实践应用场景",
			"注意事项说明",
		)
	}

	// 如果缺少子要点，添加子要点
	if len(slide.SubPoints) < 3 {
		slide.SubPoints = append(slide.SubPoints,
			"详细解释和说明",
			"具体实现方法",
			"相关技术要点",
		)
	}

	// 如果缺少技术注释，添加技术注释
	if len(slide.TechnicalNotes) == 0 {
		slide.TechnicalNotes = []string{
			"关键技术要点",
			"实现注意事项",
		}
	}
}

// addLearningObjectives 添加学习目标
func (g *TechnicalPPTGenerator) addLearningObjectives(slide *EnhancedSlideContent) {
	if len(slide.LearnObjectives) == 0 {
		switch slide.SlideType {
		case "concept":
			slide.LearnObjectives = []string{
				"理解核心概念和原理",
				"掌握相关技术要点",
				"能够应用到实际项目中",
			}
		case "tutorial":
			slide.LearnObjectives = []string{
				"掌握操作步骤和方法",
				"能够独立完成相关任务",
				"理解每个步骤的作用",
			}
		case "code":
			slide.LearnObjectives = []string{
				"理解代码结构和逻辑",
				"掌握关键API使用方法",
				"能够修改和扩展代码",
			}
		case "practice":
			slide.LearnObjectives = []string{
				"掌握最佳实践方法",
				"避免常见错误和陷阱",
				"提升代码质量和性能",
			}
		}
	}
}

// optimizeBulletPoints 优化要点列表
func (g *TechnicalPPTGenerator) optimizeBulletPoints(slide *EnhancedSlideContent) {
	// 确保要点数量在合理范围内
	if len(slide.BulletPoints) < 3 {
		// 补充要点
		slide.BulletPoints = append(slide.BulletPoints, "补充技术要点")
	}

	if len(slide.BulletPoints) > 8 {
		// 精简要点，保留最重要的8个
		slide.BulletPoints = slide.BulletPoints[:8]
	}

	// ✅ 保持原始内容，不添加硬编码文本
	for i := range slide.BulletPoints {
		// 只在内容确实为空或过短时才进行优化
		point := strings.TrimSpace(slide.BulletPoints[i])
		if point == "" {
			slide.BulletPoints[i] = "待补充要点内容"
		} else if len(point) < 5 && !strings.Contains(point, ":") {
			// 对于过短的要点，保持原样或轻微优化，但不覆盖原意
			slide.BulletPoints[i] = point
		}
		// ✅ 移除强制添加": 详细解释和实际应用"的逻辑
	}
}

// addTransitionNotes 添加过渡说明
func (g *TechnicalPPTGenerator) addTransitionNotes(current, previous *EnhancedSlideContent) {
	current.TransitionNotes = fmt.Sprintf(
		"从%s过渡到%s，注意知识点的连贯性和逻辑关系",
		previous.Title, current.Title)
}

// generateMetadata 生成元数据
func (g *TechnicalPPTGenerator) generateMetadata(slides []EnhancedSlideContent, content *parser.StructuredContent) *GenerationMetadata {
	metadata := &GenerationMetadata{
		ConceptCoverage: make(map[string]int),
		ContentTypes:    make(map[string]int),
	}

	totalWords := 0
	for _, slide := range slides {
		// 统计字数
		slideWords := len(strings.Fields(slide.MainContent)) +
			len(strings.Fields(strings.Join(slide.BulletPoints, " ")))
		totalWords += slideWords

		// 统计内容类型
		metadata.ContentTypes[slide.SlideType]++

		// 统计概念覆盖
		for _, concept := range slide.KeyConcepts {
			metadata.ConceptCoverage[concept]++
		}
	}

	metadata.TotalWords = totalWords
	metadata.AverageWordsPerSlide = totalWords / len(slides)
	metadata.TechnicalDepth = g.assessTechnicalDepth(slides)
	metadata.Recommendations = g.generateRecommendations(slides, metadata)

	return metadata
}

// Helper functions

func (g *TechnicalPPTGenerator) generateOutlineByRules(content *parser.StructuredContent, targetCount int, params *TechnicalGenerationParams) []SlideOutline {
	outlines := []SlideOutline{}

	// 1. 标题页
	outlines = append(outlines, SlideOutline{
		Number:       1,
		Title:        "课程介绍",
		Type:         "intro",
		MainPoints:   []string{"课程目标", "学习要点", "预期收获"},
		KeyConcepts:  []string{"概述"},
		ContentFocus: "concept",
		Priority:     5,
	})

	// 2. 目录页
	outlines = append(outlines, SlideOutline{
		Number:       2,
		Title:        "课程大纲",
		Type:         "outline",
		MainPoints:   []string{"主要章节", "学习路径", "知识体系"},
		KeyConcepts:  []string{"结构"},
		ContentFocus: "concept",
		Priority:     4,
	})

	// 3. 主要内容部分
	sections := g.extractMainSections(content, targetCount-3) // 减去开头和结尾
	for i, section := range sections {
		outlines = append(outlines, SlideOutline{
			Number:       i + 3,
			Title:        section.Title,
			Type:         "content",
			MainPoints:   section.KeyPoints,
			KeyConcepts:  g.generateSectionConcepts(g.getKeywords(content), i),
			ContentFocus: g.getSectionFocus("content"),
			Priority:     section.Importance,
		})
	}

	// 4. 总结页
	if len(outlines) < targetCount {
		outlines = append(outlines, SlideOutline{
			Number:       targetCount,
			Title:        "课程总结",
			Type:         "summary",
			MainPoints:   []string{"重点回顾", "核心概念", "应用展望"},
			KeyConcepts:  []string{"总结"},
			ContentFocus: "theory",
			Priority:     5,
		})
	}

	return outlines
}

func (g *TechnicalPPTGenerator) extractMainSections(content *parser.StructuredContent, count int) []SectionInfo {
	sections := []SectionInfo{}

	// 简单的基于关键词的章节划分
	keywords := g.getKeywords(content)

	sectionTypes := []string{"概念介绍", "技术实现", "实践应用"}

	for i, sectionType := range sectionTypes {
		section := SectionInfo{
			Title:      fmt.Sprintf("第%d部分：%s", i+1, g.generateSectionTitle(keywords, i, sectionType)),
			Content:    sectionType,
			KeyPoints:  g.generateSectionPoints(sectionType, i),
			Importance: 3 + (i % 3), // 3-5的重要程度
		}

		sections = append(sections, section)

		// 如果需要更多章节，继续生成
		if len(sections) >= count {
			break
		}
	}

	return sections
}

func (g *TechnicalPPTGenerator) generateSectionTitle(keywords []string, index int, sectionType string) string {
	if index < len(keywords) {
		switch sectionType {
		case "concept":
			return fmt.Sprintf("%s核心概念", keywords[index])
		case "tutorial":
			return fmt.Sprintf("%s实践教程", keywords[index])
		case "code":
			return fmt.Sprintf("%s代码示例", keywords[index])
		case "practice":
			return fmt.Sprintf("%s最佳实践", keywords[index])
		}
	}
	return fmt.Sprintf("重要知识点%d", index+1)
}

func (g *TechnicalPPTGenerator) generateSectionPoints(sectionType string, index int) []string {
	switch sectionType {
	case "concept":
		return []string{"核心概念解释", "技术原理说明", "应用场景分析", "相关技术对比", "重要特性介绍"}
	case "tutorial":
		return []string{"准备工作", "详细步骤", "关键配置", "常见问题", "验证结果"}
	case "code":
		return []string{"代码结构", "关键API", "实现逻辑", "调试技巧", "优化建议"}
	case "practice":
		return []string{"最佳实践", "性能优化", "安全考虑", "维护建议", "扩展方案"}
	default:
		return []string{"重要要点1", "重要要点2", "重要要点3", "重要要点4"}
	}
}

func (g *TechnicalPPTGenerator) generateSectionConcepts(keywords []string, index int) []string {
	concepts := []string{"核心概念", "技术要点", "实践方法"}
	if index < len(keywords) && len(keywords) > index {
		concepts[0] = keywords[index]
	}
	return concepts
}

func (g *TechnicalPPTGenerator) getSectionFocus(sectionType string) string {
	switch sectionType {
	case "concept":
		return "theory"
	case "tutorial":
		return "practice"
	case "code":
		return "code"
	case "practice":
		return "practice"
	default:
		return "theory"
	}
}

func (g *TechnicalPPTGenerator) getSummary(content *parser.StructuredContent) string {
	if content.KeyInfo != nil && content.KeyInfo.Summary != "" {
		return content.KeyInfo.Summary
	}
	// 如果没有摘要，从内容中提取前200字符作为摘要
	if len(content.CleanText) > 200 {
		return content.CleanText[:200] + "..."
	}
	return content.CleanText
}

func (g *TechnicalPPTGenerator) getKeywords(content *parser.StructuredContent) []string {
	if content.KeyInfo != nil && len(content.KeyInfo.Keywords) > 0 {
		return content.KeyInfo.Keywords
	}
	// 如果没有关键词，从内容中简单提取
	return g.extractBasicKeywords(content.CleanText)
}

// extractBasicKeywords 从文本中提取基础关键词
func (g *TechnicalPPTGenerator) extractBasicKeywords(text string) []string {
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
		if count > 1 && len(keywords) < 6 {
			keywords = append(keywords, word)
		}
	}

	// 如果关键词太少，添加一些默认的
	if len(keywords) < 3 {
		keywords = append(keywords, "文档", "内容", "知识")
	}

	return keywords
}

func (g *TechnicalPPTGenerator) createFallbackSlide(outline SlideOutline) *EnhancedSlideContent {
	return &EnhancedSlideContent{
		SlideNumber:    outline.Number,
		Title:          outline.Title,
		SlideType:      outline.Type,
		MainContent:    "详细内容请参考文档和实际演示",
		BulletPoints:   outline.MainPoints,
		KeyConcepts:    outline.KeyConcepts,
		TechnicalNotes: []string{"技术要点说明"},
		BestPractices:  []string{"相关最佳实践"},
		SpeakerNotes:   "请结合实际情况进行详细讲解",
		EstimatedTime:  4,
	}
}

func (g *TechnicalPPTGenerator) assessTechnicalDepth(slides []EnhancedSlideContent) string {
	totalComplexity := 0
	for _, slide := range slides {
		if slide.CodeExample != nil {
			totalComplexity += 3
		}
		totalComplexity += len(slide.TechnicalNotes)
		totalComplexity += len(slide.BestPractices)
	}

	avgComplexity := float64(totalComplexity) / float64(len(slides))

	if avgComplexity > 4 {
		return "high"
	} else if avgComplexity > 2 {
		return "medium"
	}
	return "low"
}

func (g *TechnicalPPTGenerator) generateRecommendations(slides []EnhancedSlideContent, metadata *GenerationMetadata) []string {
	var recommendations []string

	if metadata.AverageWordsPerSlide < 50 {
		recommendations = append(recommendations, "建议增加每张幻灯片的内容详细程度")
	}

	if metadata.ContentTypes["code"] == 0 {
		recommendations = append(recommendations, "建议添加更多代码示例幻灯片")
	}

	if metadata.TechnicalDepth == "low" {
		recommendations = append(recommendations, "可以增加技术深度和复杂度")
	}

	return recommendations
}

func getSlideTypeCount(slideType string, total int) int {
	switch slideType {
	case "concept":
		return int(float64(total-3) * 0.4) // 40%
	case "tutorial":
		return int(float64(total-3) * 0.3) // 30%
	case "code":
		return int(float64(total-3) * 0.2) // 20%
	case "practice":
		return int(float64(total-3) * 0.1) // 10%
	default:
		return 1
	}
}

// NewTechnicalTemplateManager 创建技术模板管理器
func NewTechnicalTemplateManager() *TechnicalTemplateManager {
	return &TechnicalTemplateManager{
		templates: make(map[string]*TechnicalTemplate),
	}
}

func (tm *TechnicalTemplateManager) GetTemplate(slideType, contentType string) *TechnicalTemplate {
	// 返回默认模板
	return &TechnicalTemplate{
		Name: fmt.Sprintf("%s_%s", slideType, contentType),
	}
}

// postProcessSlide 后处理幻灯片
func (g *TechnicalPPTGenerator) postProcessSlide(slide *EnhancedSlideContent, params *TechnicalGenerationParams) {
	// 确保关键概念数量合理
	if len(slide.KeyConcepts) > 5 {
		slide.KeyConcepts = slide.KeyConcepts[:5]
	}

	// 确保要点数量合理
	if len(slide.BulletPoints) > 8 {
		slide.BulletPoints = slide.BulletPoints[:8]
	}

	// 确保技术注释不为空
	if len(slide.TechnicalNotes) == 0 && slide.SlideType == "concept" {
		slide.TechnicalNotes = []string{"重要技术概念，需要重点理解"}
	}

	// 设置难度等级
	if slide.DifficultyLevel == "" {
		slide.DifficultyLevel = params.TechnicalLevel
	}
}

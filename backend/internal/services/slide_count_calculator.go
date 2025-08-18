package services

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"ai-classroom/pkg/parser"
)

// SlideCountCalculator 智能幻灯片数量计算器
type SlideCountCalculator struct {
	techTermsDB *TechnicalTermsDatabase
}

// SlideCountRecommendation 幻灯片数量推荐结果
type SlideCountRecommendation struct {
	Recommended  int                `json:"recommended"`  // 推荐数量
	Minimum      int                `json:"minimum"`      // 最少数量
	Maximum      int                `json:"maximum"`      // 最多数量
	Reasoning    string             `json:"reasoning"`    // 推荐理由
	Confidence   float64            `json:"confidence"`   // 推荐置信度 0-1
	Distribution *SlideDistribution `json:"distribution"` // 幻灯片分布建议
}

// SlideDistribution 幻灯片分布建议
type SlideDistribution struct {
	IntroSlides    int `json:"intro_slides"`    // 介绍幻灯片
	ConceptSlides  int `json:"concept_slides"`  // 概念解释幻灯片
	TutorialSlides int `json:"tutorial_slides"` // 教程步骤幻灯片
	CodeSlides     int `json:"code_slides"`     // 代码示例幻灯片
	PracticeSlides int `json:"practice_slides"` // 最佳实践幻灯片
	SummarySlides  int `json:"summary_slides"`  // 总结幻灯片
}

// ContentComplexityAnalysis 内容复杂度分析结果
type ContentComplexityAnalysis struct {
	WordCount          int     `json:"word_count"`           // 字数
	ConceptCount       int     `json:"concept_count"`        // 概念数量
	CodeBlockCount     int     `json:"code_block_count"`     // 代码块数量
	TechnicalTermCount int     `json:"technical_term_count"` // 技术术语数量
	SectionCount       int     `json:"section_count"`        // 章节数量
	ListItemCount      int     `json:"list_item_count"`      // 列表项数量
	ComplexityScore    float64 `json:"complexity_score"`     // 复杂度评分 0-1
	ContentType        string  `json:"content_type"`         // 内容类型
	TechnicalLevel     string  `json:"technical_level"`      // 技术难度
}

// TechnicalTermsDatabase 技术术语数据库
type TechnicalTermsDatabase struct {
	webDevelopment []string
	mobile         []string
	backend        []string
	framework      []string
	database       []string
	devOps         []string
}

// NewSlideCountCalculator 创建新的幻灯片数量计算器
func NewSlideCountCalculator() *SlideCountCalculator {
	return &SlideCountCalculator{
		techTermsDB: initTechnicalTermsDatabase(),
	}
}

// CalculateOptimalSlideCount 计算最优幻灯片数量
func (c *SlideCountCalculator) CalculateOptimalSlideCount(
	content *parser.StructuredContent,
	userType string,
	contentSource string, // "url", "document", "text"
) (*SlideCountRecommendation, error) {

	// 1. 分析内容复杂度
	complexity := c.analyzeContentComplexity(content)

	// 2. 计算基础幻灯片数量
	baseCount := c.calculateBaseSlideCount(complexity)

	// 3. 应用复杂度调整
	complexityMultiplier := c.getComplexityMultiplier(complexity)
	adjustedCount := int(math.Ceil(float64(baseCount) * complexityMultiplier))

	// 4. 获取用户限制
	userLimits := c.getUserLimits(userType)

	// 5. 应用用户限制
	finalCount := c.applyUserLimits(adjustedCount, userLimits)

	// 6. 生成分布建议
	distribution := c.generateSlideDistribution(finalCount, complexity)

	// 7. 计算置信度
	confidence := c.calculateConfidence(complexity, finalCount)

	// 8. 生成推荐理由
	reasoning := c.generateReasoningText(complexity, finalCount, distribution)

	return &SlideCountRecommendation{
		Recommended:  finalCount,
		Minimum:      c.getMinimumSlideCount(complexity),
		Maximum:      userLimits.MaxSlides,
		Reasoning:    reasoning,
		Confidence:   confidence,
		Distribution: distribution,
	}, nil
}

// analyzeContentComplexity 分析内容复杂度
func (c *SlideCountCalculator) analyzeContentComplexity(content *parser.StructuredContent) *ContentComplexityAnalysis {
	analysis := &ContentComplexityAnalysis{}

	// 基础统计
	analysis.WordCount = utf8.RuneCountInString(content.CleanText)
	analysis.SectionCount = len(content.Sections)

	// 代码块统计
	analysis.CodeBlockCount = c.countCodeBlocks(content.CleanText)

	// 技术术语统计
	analysis.TechnicalTermCount = c.countTechnicalTerms(content.CleanText)

	// 概念数量估算 (基于标题和关键词)
	analysis.ConceptCount = c.estimateConceptCount(content)

	// 列表项统计
	analysis.ListItemCount = c.countListItems(content.CleanText)

	// 内容类型识别
	analysis.ContentType = c.detectContentType(content)

	// 技术难度评估
	analysis.TechnicalLevel = c.assessTechnicalLevel(content)

	// 复杂度评分计算
	analysis.ComplexityScore = c.calculateComplexityScore(analysis)

	return analysis
}

// calculateBaseSlideCount 计算基础幻灯片数量
func (c *SlideCountCalculator) calculateBaseSlideCount(complexity *ContentComplexityAnalysis) int {
	baseCount := 8 // 最小基础数量

	// 根据字数增加幻灯片
	// 技术文档：平均300-500字/张幻灯片
	wordsPerSlide := 400
	if complexity.TechnicalLevel == "advanced" {
		wordsPerSlide = 300 // 高级内容每张幻灯片字数更少
	} else if complexity.TechnicalLevel == "beginner" {
		wordsPerSlide = 500 // 初级内容可以包含更多文字
	}

	wordBasedCount := complexity.WordCount / wordsPerSlide

	// 根据章节数量增加
	sectionBasedCount := complexity.SectionCount

	// 根据代码块数量增加
	codeBasedCount := complexity.CodeBlockCount

	// 根据概念数量增加
	conceptBasedCount := complexity.ConceptCount / 2 // 平均2个概念/张

	// 综合计算
	calculatedCount := wordBasedCount + sectionBasedCount + codeBasedCount + conceptBasedCount

	// 返回基础数量和计算数量的较大值
	if calculatedCount > baseCount {
		return calculatedCount
	}
	return baseCount
}

// getComplexityMultiplier 获取复杂度调整系数
func (c *SlideCountCalculator) getComplexityMultiplier(complexity *ContentComplexityAnalysis) float64 {
	multiplier := 1.0

	// 根据复杂度评分调整
	if complexity.ComplexityScore > 0.8 {
		multiplier = 1.5 // 高复杂度内容增加50%
	} else if complexity.ComplexityScore > 0.6 {
		multiplier = 1.3 // 中等复杂度增加30%
	} else if complexity.ComplexityScore > 0.4 {
		multiplier = 1.1 // 低复杂度增加10%
	}

	// 根据内容类型调整
	switch complexity.ContentType {
	case "framework_docs":
		multiplier *= 1.4 // 框架文档需要更多解释
	case "api_reference":
		multiplier *= 1.2 // API文档需要详细示例
	case "tutorial":
		multiplier *= 1.3 // 教程需要分步说明
	case "concept_explanation":
		multiplier *= 1.2 // 概念解释需要深入讲解
	}

	// 根据技术术语密度调整
	termDensity := float64(complexity.TechnicalTermCount) / float64(complexity.WordCount) * 1000
	if termDensity > 20 { // 每1000字超过20个技术术语
		multiplier *= 1.2
	}

	return multiplier
}

// getUserLimits 获取用户限制
func (c *SlideCountCalculator) getUserLimits(userType string) *UserLimits {
	switch userType {
	case "vip":
		return &UserLimits{
			MaxSlides: 50,
			MinSlides: 10,
		}
	case "premium":
		return &UserLimits{
			MaxSlides: 35,
			MinSlides: 8,
		}
	default: // regular
		return &UserLimits{
			MaxSlides: 25,
			MinSlides: 8,
		}
	}
}

// UserLimits 用户限制
type UserLimits struct {
	MaxSlides int `json:"max_slides"`
	MinSlides int `json:"min_slides"`
}

// applyUserLimits 应用用户限制
func (c *SlideCountCalculator) applyUserLimits(calculatedCount int, limits *UserLimits) int {
	if calculatedCount > limits.MaxSlides {
		return limits.MaxSlides
	}
	if calculatedCount < limits.MinSlides {
		return limits.MinSlides
	}
	return calculatedCount
}

// generateSlideDistribution 生成幻灯片分布建议
func (c *SlideCountCalculator) generateSlideDistribution(totalCount int, complexity *ContentComplexityAnalysis) *SlideDistribution {
	distribution := &SlideDistribution{}

	// 基础分配策略
	distribution.IntroSlides = 2   // 固定2张介绍
	distribution.SummarySlides = 1 // 固定1张总结

	remaining := totalCount - distribution.IntroSlides - distribution.SummarySlides

	// 根据内容类型分配剩余幻灯片
	switch complexity.ContentType {
	case "framework_docs":
		// 框架文档：重点在概念解释和实践
		distribution.ConceptSlides = int(float64(remaining) * 0.5)  // 50%
		distribution.TutorialSlides = int(float64(remaining) * 0.3) // 30%
		distribution.CodeSlides = int(float64(remaining) * 0.15)    // 15%
		distribution.PracticeSlides = remaining - distribution.ConceptSlides - distribution.TutorialSlides - distribution.CodeSlides
	case "api_reference":
		// API文档：重点在代码示例和使用方法
		distribution.CodeSlides = int(float64(remaining) * 0.4)     // 40%
		distribution.ConceptSlides = int(float64(remaining) * 0.3)  // 30%
		distribution.TutorialSlides = int(float64(remaining) * 0.2) // 20%
		distribution.PracticeSlides = remaining - distribution.CodeSlides - distribution.ConceptSlides - distribution.TutorialSlides
	case "tutorial":
		// 教程：重点在步骤说明和实践
		distribution.TutorialSlides = int(float64(remaining) * 0.5) // 50%
		distribution.CodeSlides = int(float64(remaining) * 0.25)    // 25%
		distribution.ConceptSlides = int(float64(remaining) * 0.15) // 15%
		distribution.PracticeSlides = remaining - distribution.TutorialSlides - distribution.CodeSlides - distribution.ConceptSlides
	default:
		// 默认均衡分配
		each := remaining / 4
		distribution.ConceptSlides = each
		distribution.TutorialSlides = each
		distribution.CodeSlides = each
		distribution.PracticeSlides = remaining - each*3
	}

	return distribution
}

// Helper functions

func (c *SlideCountCalculator) countCodeBlocks(text string) int {
	// 匹配代码块的正则表达式
	patterns := []string{
		"```[\\s\\S]*?```",        // Markdown代码块
		"`[^`]+`",                 // 内联代码
		"<code>[\\s\\S]*?</code>", // HTML代码标签
		"<pre>[\\s\\S]*?</pre>",   // HTML预格式化标签
	}

	count := 0
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(text, -1)
		count += len(matches)
	}

	return count
}

func (c *SlideCountCalculator) countTechnicalTerms(text string) int {
	count := 0
	lowerText := strings.ToLower(text)

	// 统计各类技术术语
	allTerms := [][]string{
		c.techTermsDB.webDevelopment,
		c.techTermsDB.mobile,
		c.techTermsDB.backend,
		c.techTermsDB.framework,
		c.techTermsDB.database,
		c.techTermsDB.devOps,
	}

	counted := make(map[string]bool) // 避免重复计算

	for _, termGroup := range allTerms {
		for _, term := range termGroup {
			if !counted[term] && strings.Contains(lowerText, strings.ToLower(term)) {
				count++
				counted[term] = true
			}
		}
	}

	return count
}

func (c *SlideCountCalculator) estimateConceptCount(content *parser.StructuredContent) int {
	count := 0

	// 基于关键词估算
	if content.KeyInfo != nil {
		count += len(content.KeyInfo.Keywords)
	}

	// 基于章节标题估算
	for _, section := range content.Sections {
		if strings.Contains(section.Type, "heading") {
			count++
		}
	}

	// 基于技术术语密度估算
	technicalTerms := c.countTechnicalTerms(content.CleanText)
	count += technicalTerms / 3 // 平均3个技术术语对应1个概念

	return count
}

func (c *SlideCountCalculator) countListItems(text string) int {
	// 匹配列表项的正则表达式
	patterns := []string{
		`^\s*[-*+]\s+.+$`, // Markdown无序列表
		`^\s*\d+\.\s+.+$`, // Markdown有序列表
		`<li>.+</li>`,     // HTML列表项
	}

	count := 0
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		for _, pattern := range patterns {
			matched, _ := regexp.MatchString(pattern, line)
			if matched {
				count++
				break
			}
		}
	}

	return count
}

func (c *SlideCountCalculator) detectContentType(content *parser.StructuredContent) string {
	text := strings.ToLower(content.CleanText)

	// 关键词检测
	if strings.Contains(text, "框架") || strings.Contains(text, "framework") {
		return "framework_docs"
	}
	if strings.Contains(text, "api") || strings.Contains(text, "接口") {
		return "api_reference"
	}
	if strings.Contains(text, "教程") || strings.Contains(text, "tutorial") || strings.Contains(text, "步骤") {
		return "tutorial"
	}
	if strings.Contains(text, "概念") || strings.Contains(text, "原理") || strings.Contains(text, "理论") {
		return "concept_explanation"
	}

	return "general"
}

func (c *SlideCountCalculator) assessTechnicalLevel(content *parser.StructuredContent) string {
	text := strings.ToLower(content.CleanText)

	// 高级技术指标
	advancedKeywords := []string{"架构", "设计模式", "性能优化", "源码", "原理", "算法", "advanced", "architecture"}
	// 初级技术指标
	beginnerKeywords := []string{"入门", "基础", "介绍", "开始", "beginner", "basic", "introduction", "getting started"}

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

func (c *SlideCountCalculator) calculateComplexityScore(analysis *ContentComplexityAnalysis) float64 {
	score := 0.0

	// 字数权重 (20%)
	wordScore := math.Min(float64(analysis.WordCount)/10000, 1.0) * 0.2

	// 技术术语密度权重 (25%)
	termDensity := float64(analysis.TechnicalTermCount) / float64(analysis.WordCount) * 1000
	termScore := math.Min(termDensity/50, 1.0) * 0.25

	// 代码块密度权重 (20%)
	codeDensity := float64(analysis.CodeBlockCount) / float64(analysis.WordCount) * 1000
	codeScore := math.Min(codeDensity/20, 1.0) * 0.2

	// 概念密度权重 (20%)
	conceptDensity := float64(analysis.ConceptCount) / float64(analysis.WordCount) * 1000
	conceptScore := math.Min(conceptDensity/30, 1.0) * 0.2

	// 结构复杂度权重 (15%)
	structureScore := math.Min(float64(analysis.SectionCount)/20, 1.0) * 0.15

	score = wordScore + termScore + codeScore + conceptScore + structureScore

	return math.Min(score, 1.0)
}

func (c *SlideCountCalculator) getMinimumSlideCount(complexity *ContentComplexityAnalysis) int {
	baseMin := 8

	if complexity.ComplexityScore > 0.7 {
		return 12 // 高复杂度内容最少12张
	}
	if complexity.ComplexityScore > 0.5 {
		return 10 // 中等复杂度最少10张
	}

	return baseMin
}

func (c *SlideCountCalculator) calculateConfidence(complexity *ContentComplexityAnalysis, finalCount int) float64 {
	confidence := 0.8 // 基础置信度

	// 如果内容复杂度很高，置信度增加
	if complexity.ComplexityScore > 0.7 {
		confidence += 0.1
	}

	// 如果有足够的结构化信息，置信度增加
	if complexity.SectionCount > 5 && complexity.ConceptCount > 10 {
		confidence += 0.05
	}

	// 如果幻灯片数量在合理范围内，置信度增加
	if finalCount >= 12 && finalCount <= 25 {
		confidence += 0.05
	}

	return math.Min(confidence, 1.0)
}

func (c *SlideCountCalculator) generateReasoningText(complexity *ContentComplexityAnalysis, finalCount int, distribution *SlideDistribution) string {
	reasoning := fmt.Sprintf("基于内容分析，推荐生成%d张幻灯片。", finalCount)

	// 添加复杂度分析
	if complexity.ComplexityScore > 0.7 {
		reasoning += "内容复杂度较高，包含大量技术概念和代码示例。"
	} else if complexity.ComplexityScore > 0.5 {
		reasoning += "内容具有中等复杂度，需要适当的详细解释。"
	} else {
		reasoning += "内容相对简单，但仍需要充分的知识点覆盖。"
	}

	// 添加分布说明
	reasoning += fmt.Sprintf("建议分配：概念解释%d张，实践教程%d张，代码示例%d张，最佳实践%d张。",
		distribution.ConceptSlides, distribution.TutorialSlides, distribution.CodeSlides, distribution.PracticeSlides)

	return reasoning
}

// initTechnicalTermsDatabase 初始化技术术语数据库
func initTechnicalTermsDatabase() *TechnicalTermsDatabase {
	return &TechnicalTermsDatabase{
		webDevelopment: []string{
			"html", "css", "javascript", "typescript", "react", "vue", "angular", "webpack", "babel",
			"scss", "sass", "less", "responsive", "bootstrap", "tailwind", "jquery", "ajax", "dom",
			"xhr", "fetch", "promise", "async", "await", "closure", "prototype", "es6", "es2015",
		},
		mobile: []string{
			"小程序", "miniprogram", "wxml", "wxss", "微信开发者工具", "app.json", "page", "component",
			"生命周期", "事件", "数据绑定", "模板", "样式", "布局", "flex", "rpx", "canvas", "api",
			"授权", "支付", "分享", "路由", "导航", "tabbar", "scroll-view", "swiper", "picker",
		},
		backend: []string{
			"服务端", "api", "restful", "graphql", "数据库", "mysql", "mongodb", "redis", "缓存",
			"session", "cookie", "jwt", "oauth", "认证", "授权", "中间件", "路由", "控制器",
			"模型", "orm", "sql", "nosql", "事务", "索引", "查询", "优化", "并发", "异步",
		},
		framework: []string{
			"框架", "架构", "mvc", "mvvm", "设计模式", "依赖注入", "控制反转", "aop", "ioc",
			"单例", "工厂", "观察者", "发布订阅", "策略", "装饰器", "适配器", "代理", "外观",
		},
		database: []string{
			"数据库", "表", "字段", "主键", "外键", "索引", "视图", "存储过程", "触发器",
			"范式", "反范式", "分区", "分片", "主从", "读写分离", "事务", "锁", "死锁",
		},
		devOps: []string{
			"部署", "运维", "docker", "kubernetes", "ci/cd", "jenkins", "git", "版本控制",
			"分支", "合并", "冲突", "代码审查", "测试", "单元测试", "集成测试", "压力测试",
			"监控", "日志", "性能", "优化", "扩容", "负载均衡", "cdn", "缓存", "备份",
		},
	}
}

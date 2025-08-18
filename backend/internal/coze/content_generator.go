package coze

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// PPTContentGenerator Coze内容转PPT结构生成器
type PPTContentGenerator struct {
	config *CozeConfig
}

// NewPPTContentGenerator 创建PPT内容生成器
func NewPPTContentGenerator(config *CozeConfig) *PPTContentGenerator {
	return &PPTContentGenerator{
		config: config,
	}
}

// PPTSlide PPT幻灯片结构
type PPTSlide struct {
	Index        int      `json:"index"`
	Title        string   `json:"title"`
	Content      []string `json:"content"`
	BulletPoints []string `json:"bullet_points"`
	SlideType    string   `json:"slide_type"` // title, content, summary
	Speaker      string   `json:"speaker_notes,omitempty"`
}

// PPTStructure PPT整体结构
type PPTStructure struct {
	Title       string     `json:"title"`
	Author      string     `json:"author"`
	CreatedAt   time.Time  `json:"created_at"`
	Description string     `json:"description"`
	Slides      []PPTSlide `json:"slides"`
	SlideCount  int        `json:"slide_count"`
	Template    string     `json:"template"`
	Language    string     `json:"language"`
}

// GenerateFromCozeResponse 从Coze API响应生成PPT结构
func (g *PPTContentGenerator) GenerateFromCozeResponse(cozeResponse string, req *GeneratePPTRequest) (*PPTStructure, error) {
	// 解析Coze响应内容
	content := g.extractMainContent(cozeResponse)

	// 构建PPT结构
	structure := &PPTStructure{
		Title:       g.extractTitle(content, req.URL),
		Author:      "Coze智能体",
		CreatedAt:   time.Now(),
		Description: g.extractDescription(content),
		Template:    req.Template,
		Language:    g.getLanguage(req.Options),
		Slides:      []PPTSlide{},
	}

	// 生成幻灯片
	slides := g.generateSlides(content, req)
	structure.Slides = slides
	structure.SlideCount = len(slides)

	return structure, nil
}

// extractMainContent 提取主要内容
func (g *PPTContentGenerator) extractMainContent(response string) string {
	// 尝试解析JSON响应
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(response), &responseData); err == nil {
		if content, exists := responseData["content"]; exists {
			if contentStr, ok := content.(string); ok {
				return contentStr
			}
		}
	}

	// 如果不是JSON，直接返回文本内容
	return response
}

// extractTitle 提取PPT标题
func (g *PPTContentGenerator) extractTitle(content, url string) string {
	// 尝试从内容中提取标题
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && len(line) < 100 {
			// 可能是标题的行
			if g.isTitleLike(line) {
				return line
			}
		}
	}

	// 从URL提取域名作为默认标题
	if strings.Contains(url, "://") {
		parts := strings.Split(url, "://")
		if len(parts) > 1 {
			domain := strings.Split(parts[1], "/")[0]
			return fmt.Sprintf("基于%s内容生成的PPT", domain)
		}
	}

	return "基于URL内容生成的PPT"
}

// isTitleLike 判断是否像标题
func (g *PPTContentGenerator) isTitleLike(line string) bool {
	// 标题特征：较短、包含关键词、没有过多标点符号
	titleKeywords := []string{"介绍", "概述", "指南", "教程", "方案", "分析", "报告"}

	for _, keyword := range titleKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}

	// 检查是否以特定格式开头
	titlePrefixes := []string{"##", "**", "###"}
	for _, prefix := range titlePrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// extractDescription 提取描述
func (g *PPTContentGenerator) extractDescription(content string) string {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 20 && len(line) < 200 && !g.isTitleLike(line) {
			return line
		}
	}

	return "由Coze智能体基于URL内容生成的专业演示文稿"
}

// getLanguage 获取语言设置
func (g *PPTContentGenerator) getLanguage(options map[string]interface{}) string {
	if lang, exists := options["language"]; exists {
		if langStr, ok := lang.(string); ok {
			return langStr
		}
	}
	return "zh-CN"
}

// generateSlides 生成幻灯片
func (g *PPTContentGenerator) generateSlides(content string, req *GeneratePPTRequest) []PPTSlide {
	var slides []PPTSlide

	// 获取目标幻灯片数量
	targetCount := g.getTargetSlideCount(req.Options)

	// 分割内容为段落
	sections := g.splitContentIntoSections(content, targetCount)

	// 生成标题页
	titleSlide := PPTSlide{
		Index:     1,
		Title:     g.extractTitle(content, req.URL),
		Content:   []string{g.extractDescription(content)},
		SlideType: "title",
	}
	slides = append(slides, titleSlide)

	// 生成内容页
	for i, section := range sections {
		slide := PPTSlide{
			Index:        i + 2,
			Title:        g.generateSectionTitle(section, i+1),
			Content:      []string{section},
			BulletPoints: g.extractBulletPoints(section),
			SlideType:    "content",
		}
		slides = append(slides, slide)
	}

	// 生成总结页（如果有足够的内容）
	if len(slides) < targetCount {
		summarySlide := PPTSlide{
			Index:     len(slides) + 1,
			Title:     "总结",
			Content:   []string{g.generateSummary(content)},
			SlideType: "summary",
		}
		slides = append(slides, summarySlide)
	}

	return slides
}

// getTargetSlideCount 获取目标幻灯片数量
func (g *PPTContentGenerator) getTargetSlideCount(options map[string]interface{}) int {
	if count, exists := options["slide_count"]; exists {
		if countInt, ok := count.(int); ok {
			return countInt
		}
		if countFloat, ok := count.(float64); ok {
			return int(countFloat)
		}
	}
	return 10 // 默认值
}

// splitContentIntoSections 将内容分割为段落
func (g *PPTContentGenerator) splitContentIntoSections(content string, targetCount int) []string {
	// 按段落分割
	paragraphs := strings.Split(content, "\n\n")

	// 过滤空段落
	var validParagraphs []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if len(p) > 20 { // 只保留有意义的段落
			validParagraphs = append(validParagraphs, p)
		}
	}

	// 如果没有有效段落，使用原始内容作为单个段落
	if len(validParagraphs) == 0 {
		if len(strings.TrimSpace(content)) > 0 {
			validParagraphs = []string{strings.TrimSpace(content)}
		} else {
			// 如果内容完全为空，返回默认内容
			validParagraphs = []string{"基于提供的URL内容生成的演示文稿"}
		}
	}

	// 计算需要的段落数（减去标题页）
	needSections := targetCount - 1
	if needSections <= 0 {
		needSections = 1
	}

	// 如果段落太多，合并相邻段落
	if len(validParagraphs) > needSections {
		var combined []string
		itemsPerSection := len(validParagraphs) / needSections
		if itemsPerSection < 1 {
			itemsPerSection = 1
		}

		for i := 0; i < len(validParagraphs); i += itemsPerSection {
			end := i + itemsPerSection
			if end > len(validParagraphs) {
				end = len(validParagraphs)
			}

			section := strings.Join(validParagraphs[i:end], "\n\n")
			combined = append(combined, section)

			if len(combined) >= needSections {
				break
			}
		}
		return combined
	}

	// 如果段落太少，分割长段落
	if len(validParagraphs) < needSections {
		var expanded []string
		for _, p := range validParagraphs {
			if len(p) > 300 { // 长段落
				sentences := g.splitIntoSentences(p)
				mid := len(sentences) / 2
				if mid > 0 {
					part1 := strings.Join(sentences[:mid], " ")
					part2 := strings.Join(sentences[mid:], " ")
					expanded = append(expanded, part1, part2)
				} else {
					expanded = append(expanded, p)
				}
			} else {
				expanded = append(expanded, p)
			}

			if len(expanded) >= needSections {
				break
			}
		}

		// 安全地返回结果，避免数组越界
		if len(expanded) >= needSections {
			return expanded[:needSections]
		}
		return expanded
	}

	return validParagraphs
}

// splitIntoSentences 将段落分割为句子
func (g *PPTContentGenerator) splitIntoSentences(paragraph string) []string {
	// 简单的句子分割
	re := regexp.MustCompile(`[。！？.!?]+`)
	sentences := re.Split(paragraph, -1)

	var validSentences []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 10 {
			validSentences = append(validSentences, s)
		}
	}

	return validSentences
}

// generateSectionTitle 生成段落标题
func (g *PPTContentGenerator) generateSectionTitle(section string, index int) string {
	// 尝试从段落第一行提取标题
	lines := strings.Split(section, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if len(firstLine) < 80 && g.isTitleLike(firstLine) {
			return firstLine
		}
	}

	// 尝试提取关键词
	keywords := g.extractKeywords(section)
	if len(keywords) > 0 {
		return fmt.Sprintf("第%d部分：%s", index, keywords[0])
	}

	return fmt.Sprintf("内容要点 %d", index)
}

// extractKeywords 提取关键词
func (g *PPTContentGenerator) extractKeywords(text string) []string {
	// 简单的关键词提取
	words := strings.Fields(text)
	var keywords []string

	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) > 2 && len(word) < 20 {
			// 简单的关键词筛选
			if g.isKeyword(word) {
				keywords = append(keywords, word)
				if len(keywords) >= 3 {
					break
				}
			}
		}
	}

	return keywords
}

// isKeyword 判断是否为关键词
func (g *PPTContentGenerator) isKeyword(word string) bool {
	// 排除常见的停用词
	stopWords := []string{"的", "了", "在", "是", "和", "与", "或", "但", "如果", "因为", "所以"}

	for _, stop := range stopWords {
		if word == stop {
			return false
		}
	}

	return true
}

// extractBulletPoints 提取要点
func (g *PPTContentGenerator) extractBulletPoints(section string) []string {
	var points []string

	lines := strings.Split(section, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检查是否已经是列表项
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "• ") {
			points = append(points, strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "), "• "))
		} else if len(line) > 10 && len(line) < 150 { // 合适长度的句子
			// 尝试分割成句子作为要点
			sentences := g.splitIntoSentences(line)
			for _, sentence := range sentences {
				if len(sentence) > 10 && len(sentence) < 100 {
					points = append(points, sentence)
				}
			}
		}

		if len(points) >= 5 { // 最多5个要点
			break
		}
	}

	return points
}

// generateSummary 生成总结
func (g *PPTContentGenerator) generateSummary(content string) string {
	lines := strings.Split(content, "\n")

	// 查找可能的总结段落
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "总结") || strings.Contains(line, "结论") || strings.Contains(line, "小结") {
			if len(line) > 20 {
				return line
			}
		}
	}

	// 如果没有找到总结，取最后一个有意义的段落
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if len(line) > 20 && len(line) < 200 {
			return line
		}
	}

	return "感谢观看本次演示，希望对您有所帮助。"
}

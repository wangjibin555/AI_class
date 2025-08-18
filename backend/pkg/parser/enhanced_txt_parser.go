package parser

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// EnhancedTXTParser 增强TXT解析器
type EnhancedTXTParser struct {
	keywordExtractor *KeywordExtractor
	structAnalyzer   *DocumentStructureAnalyzer
	nlpProcessor     *NLPProcessor
}

// NLPProcessor NLP处理器
type NLPProcessor struct {
	sentencePatterns []*regexp.Regexp
	topicPatterns    []*regexp.Regexp
	importanceRules  []ImportanceRule
}

// ImportanceRule 重要性规则
type ImportanceRule struct {
	Keywords []string
	Weight   float64
	Pattern  *regexp.Regexp
}

// NewEnhancedTXTParser 创建增强TXT解析器
func NewEnhancedTXTParser() *EnhancedTXTParser {
	return &EnhancedTXTParser{
		keywordExtractor: NewEnhancedTXTKeywordExtractor(),
		structAnalyzer:   NewEnhancedTXTDocumentStructureAnalyzer(),
		nlpProcessor:     NewEnhancedNLPProcessor(),
	}
}

// NewEnhancedTXTKeywordExtractor 创建增强TXT关键词提取器（别名）
func NewEnhancedTXTKeywordExtractor() *KeywordExtractor {
	return NewEnhancedKeywordExtractor()
}

// NewEnhancedTXTDocumentStructureAnalyzer 创建增强TXT文档结构分析器（别名）
func NewEnhancedTXTDocumentStructureAnalyzer() *DocumentStructureAnalyzer {
	return NewEnhancedDocumentStructureAnalyzer()
}

// NewEnhancedNLPProcessor 创建增强NLP处理器
func NewEnhancedNLPProcessor() *NLPProcessor {
	// 句子分割模式
	sentencePatterns := []*regexp.Regexp{
		regexp.MustCompile(`[。！？]`),
		regexp.MustCompile(`\.\s+[A-Z]`),
		regexp.MustCompile(`[!?]\s+`),
	}

	// 主题识别模式（增强版）
	topicPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[第一二三四五六七八九十0-9]+[章节部分]`),
		regexp.MustCompile(`^[0-9]+\.[0-9]*\s+`),
		regexp.MustCompile(`^[一二三四五六七八九十][、\.]\s+`),
		regexp.MustCompile(`^[①②③④⑤⑥⑦⑧⑨⑩]\s+`),
		// 技术主题模式
		regexp.MustCompile(`^[A-Z][a-zA-Z\s]+:`), // 英文主题
		regexp.MustCompile(`^[配置|实现|优化|使用]`),     // 技术操作
		regexp.MustCompile(`^[什么是|如何|怎样|为什么]`),   // 问题模式
	}

	// 增强的重要性判断规则
	importanceRules := []ImportanceRule{
		{
			Keywords: []string{"重要", "关键", "核心", "主要", "基本", "必须", "关键点", "要点"},
			Weight:   0.9,
		},
		{
			Keywords: []string{"总结", "结论", "总之", "综上", "因此", "所以", "综合"},
			Weight:   0.8,
		},
		{
			Keywords: []string{"首先", "其次", "最后", "第一", "第二", "第三", "接下来"},
			Weight:   0.7,
		},
		{
			Keywords: []string{"例如", "比如", "譬如", "举例", "示例", "案例"},
			Weight:   0.5,
		},
		{
			Keywords: []string{"定义", "概念", "原理", "特点", "特征", "性质"},
			Weight:   0.7,
		},
		// 技术相关高权重规则
		{
			Keywords: []string{"SQL注入", "预编译", "动态SQL", "连接池", "性能优化", "安全防护"},
			Weight:   0.95,
		},
		{
			Keywords: []string{"MyBatis", "Hibernate", "GORM", "ORM", "数据库", "框架"},
			Weight:   0.85,
		},
		{
			Keywords: []string{"Go", "golang", "Java", "Python", "JavaScript", "函数", "方法", "变量"},
			Weight:   0.8,
		},
		{
			Keywords: []string{"配置", "实现", "使用", "部署", "安装", "调试", "测试"},
			Weight:   0.75,
		},
		{
			Keywords: []string{"代码", "示例", "实战", "项目", "应用", "开发"},
			Weight:   0.7,
		},
	}

	// 编译重要性规则模式
	for i := range importanceRules {
		pattern := strings.Join(importanceRules[i].Keywords, "|")
		importanceRules[i].Pattern = regexp.MustCompile(pattern)
	}

	return &NLPProcessor{
		sentencePatterns: sentencePatterns,
		topicPatterns:    topicPatterns,
		importanceRules:  importanceRules,
	}
}

// ParseTXTToStructuredContent 解析TXT为结构化内容
func (p *EnhancedTXTParser) ParseTXTToStructuredContent(content string) (*StructuredContent, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("TXT内容为空")
	}

	// 1. 文本预处理
	cleanText := p.preprocessText(content)

	// 2. 语义分析
	segments := p.nlpProcessor.segmentBySemantic(cleanText)

	// 3. 提取关键信息
	keyInfo := p.extractKeyInformation(segments)

	// 4. 构建章节结构
	sections := p.buildSections(segments, keyInfo)

	// 5. 提取关键词
	keywords := p.keywordExtractor.ExtractKeywords(cleanText)

	// 6. 生成结构化内容
	result := &StructuredContent{
		OriginalText: content,
		CleanText:    cleanText,
		Sections:     sections,
		KeyInfo: &KeyInfo{
			Title:      p.extractTitle(cleanText),
			Summary:    keyInfo.Summary,
			Keywords:   keywords,
			MainPoints: keyInfo.MainPoints,
			Topics:     keyInfo.Topics,
			Difficulty: p.estimateDifficulty(cleanText),
			Duration:   p.estimateDuration(cleanText),
			Structure:  p.extractStructure(sections),
		},
	}

	return result, nil
}

// preprocessText 文本预处理
func (p *EnhancedTXTParser) preprocessText(text string) string {
	// 1. 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 2. 移除多余的空白字符
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")

	// 3. 移除多余的换行符
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// 4. 规范化标点符号
	text = strings.ReplaceAll(text, "。", "。\n")
	text = strings.ReplaceAll(text, "！", "！\n")
	text = strings.ReplaceAll(text, "？", "？\n")

	// 5. 移除首尾空白
	return strings.TrimSpace(text)
}

// segmentBySemantic 语义分割
func (nlp *NLPProcessor) segmentBySemantic(text string) []TextSegment {
	// 1. 按段落分割
	paragraphs := strings.Split(text, "\n\n")

	var segments []TextSegment
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) < 10 { // 跳过太短的段落
			continue
		}

		// 2. 提取主题
		topic := nlp.extractTopic(para)

		// 3. 计算重要性
		importance := nlp.calculateImportance(para)

		// 4. 提取关键词
		keywords := nlp.extractParagraphKeywords(para)

		// 5. 生成摘要
		summary := nlp.generateParagraphSummary(para)

		segment := TextSegment{
			Text:       para,
			Topic:      topic,
			Importance: importance,
			Keywords:   keywords,
			Summary:    summary,
		}

		segments = append(segments, segment)
	}

	// 3. 按重要性排序（用于后续处理）
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].Importance > segments[j].Importance
	})

	return segments
}

// extractTopic 提取主题
func (nlp *NLPProcessor) extractTopic(text string) string {
	// 1. 检查是否为标题格式
	for _, pattern := range nlp.topicPatterns {
		if pattern.MatchString(text) {
			// 提取标题内容
			lines := strings.Split(text, "\n")
			if len(lines) > 0 {
				return strings.TrimSpace(lines[0])
			}
		}
	}

	// 2. 提取第一句作为主题
	sentences := nlp.splitIntoSentences(text)
	if len(sentences) > 0 {
		topic := sentences[0]
		if len(topic) > 50 {
			topic = topic[:50] + "..."
		}
		return topic
	}

	return "文本内容"
}

// calculateImportance 计算重要性
func (nlp *NLPProcessor) calculateImportance(text string) float64 {
	var totalWeight float64 = 0.0
	var matchCount int = 0

	// 1. 基于关键词权重
	for _, rule := range nlp.importanceRules {
		if rule.Pattern.MatchString(text) {
			totalWeight += rule.Weight
			matchCount++
		}
	}

	// 2. 基于位置权重（开头和结尾段落更重要）
	if strings.HasPrefix(text, "总结") || strings.HasPrefix(text, "结论") {
		totalWeight += 0.8
	}

	// 3. 基于长度权重（适中长度更重要）
	length := len(text)
	if length > 50 && length < 500 {
		totalWeight += 0.3
	}

	// 4. 基于数字列表权重
	if regexp.MustCompile(`^[0-9]+[\.、]`).MatchString(text) {
		totalWeight += 0.4
	}

	// 5. 规范化权重
	if matchCount > 0 {
		totalWeight = totalWeight / float64(matchCount+1)
	}

	// 确保权重在0-1之间
	if totalWeight > 1.0 {
		totalWeight = 1.0
	}

	return totalWeight
}

// extractParagraphKeywords 提取段落关键词
func (nlp *NLPProcessor) extractParagraphKeywords(text string) []string {
	extractor := NewEnhancedKeywordExtractor()
	keywords := extractor.ExtractKeywords(text)

	// 限制关键词数量
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	return keywords
}

// generateParagraphSummary 生成段落摘要
func (nlp *NLPProcessor) generateParagraphSummary(text string) string {
	sentences := nlp.splitIntoSentences(text)

	if len(sentences) <= 1 {
		return text
	}

	// 取第一句作为摘要
	summary := sentences[0]
	if len(summary) > 100 {
		summary = summary[:100] + "..."
	}

	return summary
}

// splitIntoSentences 分句
func (nlp *NLPProcessor) splitIntoSentences(text string) []string {
	// 简单的分句逻辑
	sentences := strings.Split(text, "。")
	var result []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 5 {
			result = append(result, sentence)
		}
	}

	return result
}

// extractKeyInformation 提取关键信息
func (p *EnhancedTXTParser) extractKeyInformation(segments []TextSegment) *KeyInfo {
	var summary []string
	var mainPoints []string
	var topics []string

	// 1. 生成摘要（取重要性最高的前3个段落）
	count := 0
	for _, segment := range segments {
		if count >= 3 {
			break
		}
		if segment.Importance > 0.5 {
			summary = append(summary, segment.Summary)
			count++
		}
	}

	// 2. 提取要点
	for _, segment := range segments {
		if segment.Importance > 0.4 {
			// 查找列表项
			listItems := p.extractListItems(segment.Text)
			mainPoints = append(mainPoints, listItems...)
		}
	}

	// 3. 提取主题
	topicSet := make(map[string]bool)
	for _, segment := range segments {
		if segment.Topic != "文本内容" && !topicSet[segment.Topic] {
			topics = append(topics, segment.Topic)
			topicSet[segment.Topic] = true
		}
	}

	return &KeyInfo{
		Summary:    strings.Join(summary, " "),
		MainPoints: mainPoints,
		Topics:     topics,
	}
}

// extractListItems 提取列表项
func (p *EnhancedTXTParser) extractListItems(text string) []string {
	var items []string

	// 数字列表
	re := regexp.MustCompile(`(?m)^[0-9]+[\.、]\s*(.+)$`)
	matches := re.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			items = append(items, strings.TrimSpace(match[1]))
		}
	}

	// 中文列表
	re = regexp.MustCompile(`(?m)^[一二三四五六七八九十][、\.]\s*(.+)$`)
	matches = re.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			items = append(items, strings.TrimSpace(match[1]))
		}
	}

	// 符号列表
	re = regexp.MustCompile(`(?m)^[•\-*]\s*(.+)$`)
	matches = re.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			items = append(items, strings.TrimSpace(match[1]))
		}
	}

	return items
}

// buildSections 构建章节
func (p *EnhancedTXTParser) buildSections(segments []TextSegment, keyInfo *KeyInfo) []Section {
	var sections []Section
	var currentSection *Section

	for _, segment := range segments {
		// 判断是否为新章节
		if p.isNewSection(segment) {
			// 保存当前章节
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}

			// 创建新章节
			currentSection = &Section{
				Title:    segment.Topic,
				Content:  segment.Text,
				Type:     p.determineSectionType(segment),
				Level:    p.determineSectionLevel(segment.Topic),
				Keywords: segment.Keywords,
			}
		} else {
			// 添加到当前章节
			if currentSection != nil {
				if currentSection.Content != "" {
					currentSection.Content += "\n\n"
				}
				currentSection.Content += segment.Text

				// 合并关键词
				currentSection.Keywords = p.mergeKeywords(currentSection.Keywords, segment.Keywords)
			} else {
				// 创建默认章节
				currentSection = &Section{
					Title:    "主要内容",
					Content:  segment.Text,
					Type:     "content",
					Level:    1,
					Keywords: segment.Keywords,
				}
			}
		}
	}

	// 添加最后一个章节
	if currentSection != nil {
		sections = append(sections, *currentSection)
	}

	return sections
}

// isNewSection 判断是否为新章节
func (p *EnhancedTXTParser) isNewSection(segment TextSegment) bool {
	// 检查是否为标题格式
	titlePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[第一二三四五六七八九十0-9]+[章节部分]`),
		regexp.MustCompile(`^[0-9]+\.[0-9]*\s+`),
		regexp.MustCompile(`^[一二三四五六七八九十][、\.]\s+`),
	}

	for _, pattern := range titlePatterns {
		if pattern.MatchString(segment.Text) {
			return true
		}
	}

	// 检查重要性
	return segment.Importance > 0.7
}

// determineSectionType 确定章节类型
func (p *EnhancedTXTParser) determineSectionType(segment TextSegment) string {
	if regexp.MustCompile(`^[第一二三四五六七八九十0-9]+章`).MatchString(segment.Topic) {
		return "chapter"
	}
	if regexp.MustCompile(`^[第一二三四五六七八九十0-9]+节`).MatchString(segment.Topic) {
		return "section"
	}
	if segment.Importance > 0.6 {
		return "important"
	}
	return "content"
}

// determineSectionLevel 确定章节层级
func (p *EnhancedTXTParser) determineSectionLevel(topic string) int {
	if regexp.MustCompile(`^[第一二三四五六七八九十0-9]+章`).MatchString(topic) {
		return 1
	}
	if regexp.MustCompile(`^[第一二三四五六七八九十0-9]+节`).MatchString(topic) {
		return 2
	}
	if regexp.MustCompile(`^[0-9]+\.[0-9]+`).MatchString(topic) {
		return 3
	}
	return 4
}

// mergeKeywords 合并关键词
func (p *EnhancedTXTParser) mergeKeywords(keywords1, keywords2 []string) []string {
	keywordSet := make(map[string]bool)
	var result []string

	// 添加第一组关键词
	for _, keyword := range keywords1 {
		if !keywordSet[keyword] {
			result = append(result, keyword)
			keywordSet[keyword] = true
		}
	}

	// 添加第二组关键词
	for _, keyword := range keywords2 {
		if !keywordSet[keyword] && len(result) < 10 {
			result = append(result, keyword)
			keywordSet[keyword] = true
		}
	}

	return result
}

// extractTitle 提取标题
func (p *EnhancedTXTParser) extractTitle(text string) string {
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 5 && len(line) < 100 {
			// 检查是否为标题格式
			if !regexp.MustCompile(`^\d+$`).MatchString(line) &&
				!strings.Contains(line, "。") {
				return line
			}
		}
	}

	return "文本文档"
}

// estimateDifficulty 估算难度
func (p *EnhancedTXTParser) estimateDifficulty(text string) string {
	// 基于文本复杂性判断难度
	sentences := strings.Split(text, "。")
	avgLength := len(text) / len(sentences)

	// 计算复杂词汇比例
	words := strings.Fields(text)
	complexWords := 0
	for _, word := range words {
		if len(word) > 4 {
			complexWords++
		}
	}
	complexRatio := float64(complexWords) / float64(len(words))

	if avgLength > 50 || complexRatio > 0.3 {
		return "困难"
	} else if avgLength > 30 || complexRatio > 0.2 {
		return "中等"
	}
	return "简单"
}

// estimateDuration 估算时长
func (p *EnhancedTXTParser) estimateDuration(text string) int {
	// 按平均阅读速度估算（约200字/分钟）
	charCount := len([]rune(text))
	duration := charCount / 200

	if duration < 5 {
		return 5
	}
	if duration > 60 {
		return 60
	}
	return duration
}

// extractStructure 提取结构
func (p *EnhancedTXTParser) extractStructure(sections []Section) []string {
	var structure []string

	for _, section := range sections {
		switch section.Type {
		case "chapter":
			structure = append(structure, "章节: "+section.Title)
		case "section":
			structure = append(structure, "小节: "+section.Title)
		case "important":
			structure = append(structure, "要点: "+section.Title)
		}
	}

	if len(structure) == 0 {
		structure = []string{"文本分析", "内容提取"}
	}

	return structure
}

package services

import (
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/parser"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
)

// KeywordExtractionService 关键词提取服务
type KeywordExtractionService struct {
	parser         *parser.KeywordExtractor
	aiClient       *ai.DashScopeClient
	stopWords      map[string]bool
	techTerms      map[string]bool
	metricsService *KeywordMetricsService
}

// ExtractionOptions 关键词提取选项
type ExtractionOptions struct {
	MaxKeywords    int    `json:"max_keywords"`
	ContentType    string `json:"content_type"`
	Language       string `json:"language"`
	UseAI          bool   `json:"use_ai"`
	IncludePhrases bool   `json:"include_phrases"`
}

// KeywordResult 关键词提取结果
type KeywordResult struct {
	Primary    []string `json:"primary"`
	Secondary  []string `json:"secondary"`
	Technical  []string `json:"technical"`
	Academic   []string `json:"academic"`
	All        []string `json:"all"`
	Confidence float64  `json:"confidence"`
}

// CategorizedKeywords 分类后的关键词
type CategorizedKeywords struct {
	Primary   []string `json:"primary"`
	Secondary []string `json:"secondary"`
	Technical []string `json:"technical"`
	Academic  []string `json:"academic"`
}

// NewKeywordExtractionService 创建关键词提取服务
func NewKeywordExtractionService(aiClient *ai.DashScopeClient) *KeywordExtractionService {
	return &KeywordExtractionService{
		parser:         parser.NewEnhancedKeywordExtractor(),
		aiClient:       aiClient,
		stopWords:      loadStopWords(),
		techTerms:      loadTechnicalTerms(),
		metricsService: nil,
	}
}

// SetMetricsService 设置监控服务
func (s *KeywordExtractionService) SetMetricsService(metricsService *KeywordMetricsService) {
	s.metricsService = metricsService
}

// ExtractKeywords 提取关键词的统一接口
func (s *KeywordExtractionService) ExtractKeywords(content string, options *ExtractionOptions) (*KeywordResult, error) {
	startTime := time.Now()

	// 设置默认选项 - 技术文档需要更多关键词
	if options == nil {
		options = &ExtractionOptions{
			MaxKeywords: 50, // 增加到50个关键词
			ContentType: "technical",
			UseAI:       true,
		}
	}

	// 如果是技术内容，确保关键词数量足够
	if options.ContentType == "technical" && options.MaxKeywords < 30 {
		options.MaxKeywords = 30
	}

	// 1. 基础关键词提取
	basicKeywords := s.extractBasicKeywords(content)

	// 2. AI增强关键词提取
	var aiKeywords []string
	var aiError error

	if options.UseAI && s.aiClient != nil {
		aiKeywords, aiError = s.extractKeywordsWithAI(content, options)
		if aiError != nil {
			log.Printf("⚠️ AI关键词提取失败，使用基础提取: %v", aiError)
			aiKeywords = basicKeywords
		}
	} else {
		aiKeywords = basicKeywords
	}

	// 3. 合并和去重
	finalKeywords := s.mergeAndRankKeywords(basicKeywords, aiKeywords)

	// 4. 分类关键词
	categorized := s.categorizeKeywords(finalKeywords, options.ContentType)

	// 5. 应用数量限制
	if options.MaxKeywords > 0 && len(finalKeywords) > options.MaxKeywords {
		finalKeywords = finalKeywords[:options.MaxKeywords]
	}

	// 6. 计算置信度
	confidence := s.calculateConfidence(basicKeywords, aiKeywords, aiError == nil)

	// 7. 记录监控指标
	processTime := time.Since(startTime)
	s.logExtraction(content, finalKeywords, aiError == nil, processTime)

	// 8. 记录监控事件
	if s.metricsService != nil {
		event := &ExtractionEvent{
			ContentLength: len(content),
			KeywordCount:  len(finalKeywords),
			ProcessTime:   processTime,
			ContentType:   options.ContentType,
			UseAI:         options.UseAI,
			Success:       aiError == nil,
			ErrorMessage:  "",
			Keywords:      finalKeywords,
			QualityScore:  confidence,
			Timestamp:     time.Now(),
		}
		if aiError != nil {
			event.ErrorMessage = aiError.Error()
		}
		s.metricsService.RecordExtraction(event)
	}

	return &KeywordResult{
		Primary:    categorized.Primary,
		Secondary:  categorized.Secondary,
		Technical:  categorized.Technical,
		Academic:   categorized.Academic,
		All:        finalKeywords,
		Confidence: confidence,
	}, nil
}

// extractBasicKeywords 基础关键词提取
func (s *KeywordExtractionService) extractBasicKeywords(content string) []string {
	if s.parser != nil {
		return s.parser.ExtractKeywords(content)
	}
	return s.simpleKeywordExtraction(content)
}

// simpleKeywordExtraction 简单关键词提取
func (s *KeywordExtractionService) simpleKeywordExtraction(content string) []string {
	cleanContent := s.cleanContent(content)
	words := s.segmentWords(cleanContent)
	filteredWords := s.filterStopWords(words)

	wordFreq := make(map[string]int)
	for _, word := range filteredWords {
		if len(word) >= 2 {
			wordFreq[word]++
		}
	}

	return s.selectTopKeywords(wordFreq, 15)
}

// extractKeywordsWithAI AI增强关键词提取
func (s *KeywordExtractionService) extractKeywordsWithAI(content string, options *ExtractionOptions) ([]string, error) {
	prompt := s.buildAIPrompt(content, options)

	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI关键词提取请求失败: %w", err)
	}

	keywords, err := s.parseAIResponse(response)
	if err != nil {
		return nil, fmt.Errorf("AI响应解析失败: %w", err)
	}

	return keywords, nil
}

// buildAIPrompt 构建AI提示词
func (s *KeywordExtractionService) buildAIPrompt(content string, options *ExtractionOptions) string {
	maxKeywords := options.MaxKeywords
	if maxKeywords <= 0 {
		maxKeywords = 15
	}

	contentType := options.ContentType
	if contentType == "" {
		contentType = "academic"
	}

	prompt := fmt.Sprintf(`
请从以下内容中提取%d个最重要的关键词，要求：

1. 关键词要能代表内容的核心概念
2. 优先选择技术术语、专业概念
3. 避免过于通用的词汇（如"内容"、"方法"、"系统"等）
4. 按重要性排序
5. 适合%s类型的内容
6. 关键词长度2-8个字符
7. 以JSON数组格式返回

内容：
%s

请返回格式：["关键词1", "关键词2", "关键词3", ...]
`, maxKeywords, contentType, content)

	return prompt
}

// parseAIResponse 解析AI响应
func (s *KeywordExtractionService) parseAIResponse(response string) ([]string, error) {
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("AI响应中未找到有效的JSON数组")
	}

	jsonContent := response[jsonStart : jsonEnd+1]

	var keywords []string
	if err := json.Unmarshal([]byte(jsonContent), &keywords); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	var cleanedKeywords []string
	for _, kw := range keywords {
		cleaned := strings.TrimSpace(kw)
		if cleaned != "" && len(cleaned) >= 2 && len(cleaned) <= 8 {
			cleanedKeywords = append(cleanedKeywords, cleaned)
		}
	}

	return cleanedKeywords, nil
}

// mergeAndRankKeywords 合并和排序关键词
func (s *KeywordExtractionService) mergeAndRankKeywords(basicKeywords, aiKeywords []string) []string {
	keywordWeight := make(map[string]float64)

	// AI关键词权重更高
	for i, kw := range aiKeywords {
		weight := 1.0 - float64(i)*0.1
		if weight < 0.1 {
			weight = 0.1
		}
		keywordWeight[kw] = weight * 2.0
	}

	// 基础关键词作为补充
	for i, kw := range basicKeywords {
		if _, exists := keywordWeight[kw]; !exists {
			weight := 1.0 - float64(i)*0.05
			if weight < 0.1 {
				weight = 0.1
			}
			keywordWeight[kw] = weight
		}
	}

	type keywordItem struct {
		keyword string
		weight  float64
	}

	var sortedKeywords []keywordItem
	for kw, weight := range keywordWeight {
		sortedKeywords = append(sortedKeywords, keywordItem{
			keyword: kw,
			weight:  weight,
		})
	}

	sort.Slice(sortedKeywords, func(i, j int) bool {
		return sortedKeywords[i].weight > sortedKeywords[j].weight
	})

	var result []string
	for _, item := range sortedKeywords {
		result = append(result, item.keyword)
	}

	return result
}

// categorizeKeywords 分类关键词
func (s *KeywordExtractionService) categorizeKeywords(keywords []string, contentType string) CategorizedKeywords {
	var categorized CategorizedKeywords

	for i, kw := range keywords {
		if i < 5 {
			categorized.Primary = append(categorized.Primary, kw)
		} else if i < 15 {
			categorized.Secondary = append(categorized.Secondary, kw)
		}

		if s.isTechnicalTerm(kw) {
			categorized.Technical = append(categorized.Technical, kw)
		}

		if s.isAcademicTerm(kw, contentType) {
			categorized.Academic = append(categorized.Academic, kw)
		}
	}

	return categorized
}

// calculateConfidence 计算置信度
func (s *KeywordExtractionService) calculateConfidence(basicKeywords, aiKeywords []string, aiSuccess bool) float64 {
	if !aiSuccess {
		return 0.6
	}

	overlap := s.calculateOverlap(basicKeywords, aiKeywords)
	confidence := 0.7 + overlap*0.3
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// cleanContent 清理内容
func (s *KeywordExtractionService) cleanContent(content string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(content, "")

	re = regexp.MustCompile(`[^\p{Han}\p{L}\p{N}\s]`)
	cleaned = re.ReplaceAllString(cleaned, " ")

	re = regexp.MustCompile(`\s+`)
	cleaned = re.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}

// segmentWords 分词
func (s *KeywordExtractionService) segmentWords(content string) []string {
	words := strings.Fields(content)

	var result []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) >= 2 {
			result = append(result, word)
		}
	}

	return result
}

// filterStopWords 过滤停用词
func (s *KeywordExtractionService) filterStopWords(words []string) []string {
	var filtered []string
	for _, word := range words {
		if !s.stopWords[word] {
			filtered = append(filtered, word)
		}
	}
	return filtered
}

// selectTopKeywords 选择Top关键词
func (s *KeywordExtractionService) selectTopKeywords(wordFreq map[string]int, maxCount int) []string {
	type wordItem struct {
		word  string
		count int
	}

	var words []wordItem
	for word, count := range wordFreq {
		words = append(words, wordItem{word: word, count: count})
	}

	sort.Slice(words, func(i, j int) bool {
		return words[i].count > words[j].count
	})

	var result []string
	for i, item := range words {
		if i >= maxCount {
			break
		}
		result = append(result, item.word)
	}

	return result
}

// isTechnicalTerm 判断是否为技术术语
func (s *KeywordExtractionService) isTechnicalTerm(term string) bool {
	return s.techTerms[term] ||
		strings.Contains(term, "技术") ||
		strings.Contains(term, "算法") ||
		strings.Contains(term, "系统") ||
		strings.Contains(term, "程序") ||
		strings.Contains(term, "开发")
}

// isAcademicTerm 判断是否为学术术语
func (s *KeywordExtractionService) isAcademicTerm(term, contentType string) bool {
	switch contentType {
	case "academic":
		return strings.Contains(term, "理论") ||
			strings.Contains(term, "研究") ||
			strings.Contains(term, "分析") ||
			strings.Contains(term, "方法")
	case "technical":
		return s.isTechnicalTerm(term)
	default:
		return false
	}
}

// calculateOverlap 计算重叠度
func (s *KeywordExtractionService) calculateOverlap(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, kw := range keywords1 {
		set1[kw] = true
	}

	overlap := 0
	for _, kw := range keywords2 {
		if set1[kw] {
			overlap++
		}
	}

	return float64(overlap) / float64(len(keywords2))
}

// logExtraction 记录关键词提取日志
func (s *KeywordExtractionService) logExtraction(content string, keywords []string, aiSuccess bool, processTime time.Duration) {
	log.Printf("📊 关键词提取完成: 内容长度=%d, 提取数量=%d, AI成功=%t, 耗时=%v",
		len(content), len(keywords), aiSuccess, processTime)

	if len(keywords) == 0 {
		log.Printf("⚠️ 关键词提取为空")
	}
}

// loadStopWords 加载停用词
func loadStopWords() map[string]bool {
	stopWords := map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true,
		"有": true, "和": true, "就": true, "不": true, "人": true,
		"都": true, "一": true, "一个": true, "上": true, "也": true,
		"很": true, "到": true, "说": true, "要": true, "去": true,
		"你": true, "会": true, "着": true, "没有": true, "看": true,
		"好": true, "自己": true, "这": true, "那": true, "里": true,
		"就是": true, "还是": true, "时候": true, "可以": true, "这个": true,
		"那个": true, "什么": true, "怎么": true, "为什么": true, "如何": true,
		"内容": true, "方法": true, "方式": true, "问题": true, "情况": true,
		"the": true, "of": true, "and": true, "to": true, "a": true,
		"in": true, "is": true, "it": true, "you": true, "that": true,
		"he": true, "was": true, "for": true, "on": true, "are": true,
		"as": true, "with": true, "his": true, "they": true, "i": true,
	}
	return stopWords
}

// loadTechnicalTerms 加载技术术语
func loadTechnicalTerms() map[string]bool {
	techTerms := map[string]bool{
		"API": true, "HTTP": true, "JSON": true, "XML": true, "SQL": true,
		"数据库": true, "算法": true, "架构": true, "框架": true, "组件": true,
		"接口": true, "协议": true, "服务": true, "中间件": true, "缓存": true,
		"分布式": true, "微服务": true, "容器": true, "云计算": true, "大数据": true,
		"机器学习": true, "人工智能": true, "深度学习": true, "神经网络": true,
		"前端": true, "后端": true, "全栈": true, "移动端": true, "Web": true,
		"JavaScript": true, "Python": true, "Java": true, "Go": true, "C++": true,
		"React": true, "Vue": true, "Angular": true, "Node.js": true, "Spring": true,
		"Docker": true, "Kubernetes": true, "Redis": true, "MySQL": true, "MongoDB": true,
	}
	return techTerms
}

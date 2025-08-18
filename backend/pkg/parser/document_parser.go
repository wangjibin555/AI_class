package parser

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	// PDF parsing library
	"github.com/ledongthuc/pdf"
)

// DocumentParser 文档解析器接口
type DocumentParser interface {
	ParseFile(file *multipart.FileHeader) (*ParseResult, error)
	ParsePDF(reader io.ReadSeeker) (*ParseResult, error)
	ParseWord(reader io.ReadSeeker) (*ParseResult, error)
	ParseText(content string) (*ParseResult, error)
	ValidateFile(file *multipart.FileHeader) error
	GetSupportedFormats() []string
	ExtractKeyInfo(content string) (*KeyInfo, error)
	StructureContent(content string) (*StructuredContent, error)
	SetKeywordExtractor(extractor KeywordExtractor) // ✅ 新增方法设置关键词提取器
}

// ParseResult 解析结果
type ParseResult struct {
	Content       string            `json:"content"`
	Title         string            `json:"title"`
	Author        string            `json:"author"`
	CreatedAt     time.Time         `json:"created_at"`
	WordCount     int               `json:"word_count"`
	PageCount     int               `json:"page_count"`
	Language      string            `json:"language"`
	Keywords      []string          `json:"keywords"`
	Metadata      map[string]string `json:"metadata"`
	ExtractedText string            `json:"extracted_text"`
	FileType      string            `json:"file_type"`
	FileSize      int64             `json:"file_size"`
	ProcessTime   time.Duration     `json:"process_time"`
	Status        string            `json:"status"`
	ErrorMessage  string            `json:"error_message"`
}

// 类型定义现在在shared_types.go中

// documentParser 文档解析器实现
type documentParser struct {
	maxFileSize      int64
	supportedFormats []string
	keywordExtractor *KeywordExtractor // ✅ 修改为指针类型
}

// NewDocumentParser 创建文档解析器
func NewDocumentParser() DocumentParser {
	return &documentParser{
		maxFileSize:      10 * 1024 * 1024, // 10MB
		supportedFormats: []string{".pdf", ".doc", ".docx", ".txt", ".md"},
		keywordExtractor: nil, // 初始化为空，可通过SetKeywordExtractor设置
	}
}

// SetKeywordExtractor 设置关键词提取器
func (p *documentParser) SetKeywordExtractor(extractor KeywordExtractor) {
	p.keywordExtractor = &extractor
}

// ParseFile 解析文件
func (p *documentParser) ParseFile(file *multipart.FileHeader) (*ParseResult, error) {
	startTime := time.Now()

	// 验证文件
	if err := p.ValidateFile(file); err != nil {
		return &ParseResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
			ProcessTime:  time.Since(startTime),
		}, err
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		return &ParseResult{
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("无法打开文件: %v", err),
			ProcessTime:  time.Since(startTime),
		}, err
	}
	defer src.Close()

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))

	var result *ParseResult

	// 根据文件类型解析
	switch ext {
	case ".pdf":
		result, err = p.ParsePDF(src)
	case ".doc", ".docx":
		result, err = p.ParseWord(src)
	case ".txt", ".md":
		// 读取文本内容
		content, readErr := io.ReadAll(src)
		if readErr != nil {
			return &ParseResult{
				Status:       "failed",
				ErrorMessage: fmt.Sprintf("无法读取文件内容: %v", readErr),
				ProcessTime:  time.Since(startTime),
			}, readErr
		}
		result, err = p.ParseText(string(content))
	default:
		return &ParseResult{
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("不支持的文件格式: %s", ext),
			ProcessTime:  time.Since(startTime),
		}, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	if err != nil {
		return result, err
	}

	// 补充基本信息
	result.FileType = ext
	result.FileSize = file.Size
	result.ProcessTime = time.Since(startTime)
	result.Status = "success"

	return result, nil
}

// ParsePDF 解析PDF文件
func (p *documentParser) ParsePDF(reader io.ReadSeeker) (*ParseResult, error) {
	startTime := time.Now()
	result := &ParseResult{
		CreatedAt: time.Now(),
		Language:  "zh-CN",
		Metadata:  make(map[string]string),
	}

	// 读取PDF内容
	data, err := io.ReadAll(reader)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("读取PDF文件失败: %v", err)
		return result, err
	}

	// 使用PDF库解析
	pdfReader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		// 如果PDF解析失败，尝试基本的文本提取
		result.Status = "partial"
		result.ErrorMessage = fmt.Sprintf("PDF解析失败，使用基本文本提取: %v", err)
		result.Content = p.extractBasicTextFromPDF(data)
		result.ExtractedText = result.Content
		result.WordCount = len([]rune(result.Content))
		result.PageCount = 1
		result.Title = p.extractTitle(result.Content)
		result.Keywords = p.extractKeywords(result.Content)
		return result, nil
	}

	// 获取PDF基本信息
	result.PageCount = pdfReader.NumPage()

	// 提取文本内容
	var textContent strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// 获取页面文本 - 提供空的字体映射
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			// 如果获取页面文本失败，记录但继续处理其他页面
			continue
		}

		// 清理和格式化文本
		cleanedText := p.cleanText(pageText)
		if cleanedText != "" {
			textContent.WriteString(cleanedText)
			textContent.WriteString("\n\n")
		}
	}

	extractedText := textContent.String()

	// 如果没有提取到文本，使用基本提取方法
	if strings.TrimSpace(extractedText) == "" {
		extractedText = p.extractBasicTextFromPDF(data)
	}

	// 处理提取的文本
	result.Content = extractedText
	result.ExtractedText = extractedText
	result.WordCount = len([]rune(extractedText))
	result.Title = p.extractTitle(extractedText)
	result.Keywords = p.extractKeywords(extractedText)
	result.ProcessTime = time.Since(startTime)
	result.Status = "success"

	// 如果内容为空，标记为部分成功
	if strings.TrimSpace(extractedText) == "" {
		result.Status = "partial"
		result.ErrorMessage = "PDF文档可能包含图片或特殊格式，无法提取文本内容"
		result.Content = "PDF文档解析完成，但未能提取到文本内容。这可能是因为PDF主要包含图片、扫描件或特殊格式。建议使用文本格式或重新制作PDF文档。"
		result.ExtractedText = result.Content
		result.WordCount = len([]rune(result.Content))
		result.Title = "PDF文档"
	}

	return result, nil
}

// ParseWord 解析Word文档
func (p *documentParser) ParseWord(reader io.ReadSeeker) (*ParseResult, error) {
	startTime := time.Now()
	result := &ParseResult{
		CreatedAt: time.Now(),
		Language:  "zh-CN",
		Metadata:  make(map[string]string),
	}

	// 读取所有数据到内存
	data, err := io.ReadAll(reader)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("读取Word文档失败: %v", err)
		return result, err
	}

	if len(data) == 0 {
		result.Status = "failed"
		result.ErrorMessage = "Word文档内容为空"
		return result, fmt.Errorf("Word文档内容为空")
	}

	// 尝试基本的文本提取
	// 对于.docx文件，我们可以尝试提取一些基本信息
	var extractedText string

	// 检查文件头来确定Word文档类型
	if p.isDocxFile(data) {
		// 处理.docx文件（实际上是ZIP格式）
		extractedText = p.extractTextFromDocx(data)
	} else {
		// 处理旧版.doc文件
		extractedText = p.extractTextFromDoc(data)
	}

	// 如果没有提取到有效文本，使用默认内容
	if strings.TrimSpace(extractedText) == "" {
		result.Status = "partial"
		result.ErrorMessage = "Word文档解析完成，但未能提取到完整文本内容"
		extractedText = "Word文档已解析。由于Word文档格式的复杂性，建议使用文本格式或PDF格式以获得更好的解析效果。您也可以复制文档内容到文本输入框中。"
	}

	// 处理提取的文本
	result.Content = extractedText
	result.ExtractedText = extractedText
	result.WordCount = len([]rune(extractedText))
	result.PageCount = 1 // 简化处理，Word文档页数难以准确计算
	result.Title = p.extractTitle(extractedText)
	result.Keywords = p.extractKeywords(extractedText)
	result.ProcessTime = time.Since(startTime)
	result.Status = "success"

	return result, nil
}

// ParseText 解析纯文本
func (p *documentParser) ParseText(content string) (*ParseResult, error) {
	text := strings.TrimSpace(content)

	if len(text) == 0 {
		return &ParseResult{
			Status:       "failed",
			ErrorMessage: "文本内容为空",
		}, fmt.Errorf("文本内容为空")
	}

	result := &ParseResult{
		Content:       text,
		ExtractedText: text,
		WordCount:     len([]rune(text)),
		PageCount:     1,
		CreatedAt:     time.Now(),
		Language:      "zh-CN",
		Status:        "success",
		Metadata:      make(map[string]string),
	}

	// 提取标题
	result.Title = p.extractTitle(text)

	// 提取关键词
	result.Keywords = p.extractKeywords(text)

	return result, nil
}

// ValidateFile 验证文件
func (p *documentParser) ValidateFile(file *multipart.FileHeader) error {
	if file == nil {
		return fmt.Errorf("文件不能为空")
	}

	// 检查文件大小
	if file.Size > p.maxFileSize {
		return fmt.Errorf("文件大小超过限制，最大允许 %d MB", p.maxFileSize/(1024*1024))
	}

	// 检查文件格式
	ext := strings.ToLower(filepath.Ext(file.Filename))
	supported := false
	for _, format := range p.supportedFormats {
		if ext == format {
			supported = true
			break
		}
	}

	if !supported {
		return fmt.Errorf("不支持的文件格式: %s，支持的格式: %s", ext, strings.Join(p.supportedFormats, ", "))
	}

	return nil
}

// GetSupportedFormats 获取支持的格式
func (p *documentParser) GetSupportedFormats() []string {
	return p.supportedFormats
}

// 辅助方法

// extractTitle 提取标题
func (p *documentParser) extractTitle(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len([]rune(line)) <= 100 {
			// 移除特殊字符，保留标题的核心内容
			title := p.cleanTitle(line)
			if title != "" {
				return title
			}
		}
	}

	// 如果没有找到合适的标题，取前50个字符
	runes := []rune(text)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return string(runes)
}

// extractKeywords 提取关键词
func (p *documentParser) extractKeywords(text string) []string {
	// ✅ 使用新的KeywordExtractor（如果已设置）
	if p.keywordExtractor != nil {
		return p.keywordExtractor.ExtractKeywords(text)
	}

	// ✅ 回退到简单的关键词提取算法
	// 1. 分词
	words := p.segmentText(text)

	// 2. 统计词频
	wordCount := make(map[string]int)
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len([]rune(word)) > 1 && !p.isStopWord(word) {
			wordCount[word]++
		}
	}

	// 3. 选择高频词作为关键词
	var keywords []string
	for word, count := range wordCount {
		if count >= 2 && len(keywords) < 10 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// cleanText 清理文本内容
func (p *documentParser) cleanText(text string) string {
	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// 移除控制字符
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, text)

	// 移除多余的换行
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// cleanTitle 清理标题
func (p *documentParser) cleanTitle(title string) string {
	// 移除特殊字符和多余空格
	title = regexp.MustCompile(`[^\p{L}\p{N}\s\-_()（）【】]`).ReplaceAllString(title, "")
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	return strings.TrimSpace(title)
}

// segmentText 简单分词
func (p *documentParser) segmentText(text string) []string {
	// 简单的分词实现，按标点符号和空格分割
	words := regexp.MustCompile(`[\s\p{P}]+`).Split(text, -1)

	var result []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len([]rune(word)) > 1 {
			result = append(result, word)
		}
	}

	return result
}

// isStopWord 判断是否为停用词
func (p *documentParser) isStopWord(word string) bool {
	stopWords := []string{
		"的", "了", "在", "是", "我", "有", "和", "就", "不", "人", "都", "一", "一个", "上", "也", "很", "到", "说", "要", "去", "你", "会", "着", "没有", "看", "好", "自己", "这", "那", "它", "他", "她",
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by", "this", "that", "these", "those", "is", "are", "was", "were", "be", "been", "being", "have", "has", "had", "do", "does", "did", "will", "would", "could", "should",
	}

	for _, stopWord := range stopWords {
		if strings.EqualFold(word, stopWord) {
			return true
		}
	}

	return false
}

// extractBasicTextFromPDF 基本PDF文本提取
func (p *documentParser) extractBasicTextFromPDF(data []byte) string {
	// 简单的PDF文本提取，查找可读文本
	text := string(data)

	// 查找可能的文本内容
	textPattern := regexp.MustCompile(`\(([^)]+)\)`)
	matches := textPattern.FindAllStringSubmatch(text, -1)

	var extractedText strings.Builder
	for _, match := range matches {
		if len(match) > 1 {
			content := strings.TrimSpace(match[1])
			if len(content) > 3 && p.isReadableText(content) {
				extractedText.WriteString(content)
				extractedText.WriteString(" ")
			}
		}
	}

	result := extractedText.String()
	if strings.TrimSpace(result) == "" {
		return "PDF文档解析完成，但未能提取到文本内容。这可能是因为PDF主要包含图片、扫描件或特殊格式。"
	}

	return p.cleanText(result)
}

// isReadableText 判断是否为可读文本
func (p *documentParser) isReadableText(text string) bool {
	// 检查文本是否包含足够的可读字符
	readableChars := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			readableChars++
		}
	}

	return float64(readableChars)/float64(len(text)) > 0.7
}

// isDocxFile 判断是否为DOCX文件
func (p *documentParser) isDocxFile(data []byte) bool {
	// DOCX文件实际上是ZIP格式，检查ZIP文件头
	return len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04
}

// extractTextFromDocx 从DOCX文件提取文本
func (p *documentParser) extractTextFromDocx(data []byte) string {
	// 简化的DOCX文本提取
	// 实际项目中可以使用专门的DOCX解析库

	// 将数据转换为字符串进行简单的文本搜索
	content := string(data)

	// 查找XML文本内容
	textPattern := regexp.MustCompile(`<w:t[^>]*>([^<]+)</w:t>`)
	matches := textPattern.FindAllStringSubmatch(content, -1)

	var extractedText strings.Builder
	for _, match := range matches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			if text != "" {
				extractedText.WriteString(text)
				extractedText.WriteString(" ")
			}
		}
	}

	result := extractedText.String()
	if strings.TrimSpace(result) == "" {
		return "DOCX文档解析完成，但未能提取到完整文本内容。建议使用文本格式或PDF格式。"
	}

	return p.cleanText(result)
}

// extractTextFromDoc 从DOC文件提取文本
func (p *documentParser) extractTextFromDoc(data []byte) string {
	// 简化的DOC文本提取
	// 旧版DOC格式较为复杂，这里进行基本的文本搜索

	content := string(data)

	// 查找可能的文本内容
	var extractedText strings.Builder
	words := strings.Fields(content)

	for _, word := range words {
		if len(word) > 2 && p.isReadableText(word) {
			extractedText.WriteString(word)
			extractedText.WriteString(" ")
		}
	}

	result := extractedText.String()
	if strings.TrimSpace(result) == "" {
		return "DOC文档解析完成，但未能提取到完整文本内容。建议使用较新的DOCX格式或文本格式。"
	}

	return p.cleanText(result)
}

// ExtractKeyInfo 提取关键信息
func (p *documentParser) ExtractKeyInfo(content string) (*KeyInfo, error) {
	if content == "" {
		return nil, fmt.Errorf("内容为空")
	}

	// 清理文本
	cleanText := p.cleanText(content)

	// 提取标题
	title := p.extractTitle(cleanText)

	// 提取关键词
	keywords := p.extractKeywords(cleanText)

	// 生成摘要
	summary := p.generateSummary(cleanText)

	// 主题分类
	topics := p.classifyTopics(cleanText)

	// 难度评估
	difficulty := p.assessDifficulty(cleanText)

	// 预估学习时长
	duration := p.estimateDuration(len([]rune(cleanText)))

	// 提取主要观点
	mainPoints := p.extractMainPoints(cleanText)

	// 分析内容结构
	structure := p.analyzeStructure(cleanText)

	return &KeyInfo{
		Title:      title,
		Keywords:   keywords,
		Summary:    summary,
		Topics:     topics,
		Difficulty: difficulty,
		Duration:   duration,
		MainPoints: mainPoints,
		Structure:  structure,
	}, nil
}

// StructureContent 结构化内容
func (p *documentParser) StructureContent(content string) (*StructuredContent, error) {
	if content == "" {
		return nil, fmt.Errorf("内容为空")
	}

	// 清理文本
	cleanText := p.cleanText(content)

	// 分段处理
	sections := p.segmentContent(cleanText)

	// 提取关键信息
	keyInfo, err := p.ExtractKeyInfo(cleanText)
	if err != nil {
		return nil, err
	}

	return &StructuredContent{
		OriginalText: content,
		CleanText:    cleanText,
		Sections:     sections,
		KeyInfo:      keyInfo,
	}, nil
}

// generateSummary 生成内容摘要
func (p *documentParser) generateSummary(text string) string {
	// 简单的摘要生成算法
	sentences := p.splitSentences(text)

	if len(sentences) <= 3 {
		return text
	}

	// 选择前3个句子作为摘要
	var summary strings.Builder
	for i := 0; i < 3 && i < len(sentences); i++ {
		summary.WriteString(sentences[i])
		summary.WriteString(" ")
	}

	return strings.TrimSpace(summary.String())
}

// classifyTopics 主题分类
func (p *documentParser) classifyTopics(text string) []string {
	topics := []string{}

	// 定义主题关键词
	topicKeywords := map[string][]string{
		"技术": {"编程", "代码", "算法", "开发", "技术", "软件", "系统", "数据库", "网络", "API"},
		"商业": {"商业", "市场", "营销", "管理", "策略", "投资", "财务", "创业", "企业", "产品"},
		"教育": {"学习", "教育", "培训", "课程", "教学", "知识", "技能", "考试", "学校", "老师"},
		"科学": {"科学", "研究", "实验", "理论", "数据", "分析", "发现", "创新", "技术", "方法"},
		"文学": {"文学", "小说", "诗歌", "散文", "创作", "艺术", "文化", "历史", "传统", "经典"},
		"历史": {"历史", "古代", "现代", "事件", "人物", "时期", "文化", "传统", "发展", "演变"},
		"艺术": {"艺术", "设计", "创作", "美学", "风格", "作品", "表现", "创意", "视觉", "音乐"},
	}

	// 统计关键词出现次数
	topicScores := make(map[string]int)
	for topic, keywords := range topicKeywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				topicScores[topic]++
			}
		}
	}

	// 选择得分最高的主题
	maxScore := 0
	for _, score := range topicScores {
		if score > maxScore {
			maxScore = score
		}
	}

	// 添加得分较高的主题
	for topic, score := range topicScores {
		if score >= maxScore/2 {
			topics = append(topics, topic)
		}
	}

	// 如果没有匹配的主题，返回通用
	if len(topics) == 0 {
		topics = append(topics, "通用")
	}

	return topics
}

// assessDifficulty 评估难度
func (p *documentParser) assessDifficulty(text string) string {
	// 基于文本长度、词汇复杂度等评估难度
	wordCount := len(strings.Fields(text))
	avgWordLength := p.calculateAverageWordLength(text)

	if wordCount < 500 || avgWordLength < 4 {
		return "初级"
	} else if wordCount < 2000 || avgWordLength < 5 {
		return "中级"
	} else {
		return "高级"
	}
}

// estimateDuration 预估学习时长
func (p *documentParser) estimateDuration(charCount int) int {
	// 基于字符数预估学习时长（分钟）
	// 假设每分钟阅读300个字符
	baseTime := charCount / 300

	// 最少5分钟，最多120分钟
	if baseTime < 5 {
		return 5
	} else if baseTime > 120 {
		return 120
	}

	return baseTime
}

// extractMainPoints 提取主要观点
func (p *documentParser) extractMainPoints(text string) []string {
	points := []string{}

	// 按段落分割
	paragraphs := strings.Split(text, "\n\n")

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		// 提取段落的关键句子
		sentences := p.splitSentences(paragraph)
		if len(sentences) > 0 {
			// 选择第一个句子作为主要观点
			point := strings.TrimSpace(sentences[0])
			if len(point) > 10 && len(point) < 200 {
				points = append(points, point)
			}
		}

		// 限制主要观点数量
		if len(points) >= 5 {
			break
		}
	}

	return points
}

// analyzeStructure 分析内容结构
func (p *documentParser) analyzeStructure(text string) []string {
	structure := []string{}

	// 按段落分割
	paragraphs := strings.Split(text, "\n\n")

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		// 判断段落类型
		paragraphType := p.classifyParagraph(paragraph)
		structure = append(structure, paragraphType)
	}

	return structure
}

// segmentContent 分段处理内容
func (p *documentParser) segmentContent(text string) []Section {
	sections := []Section{}

	// 按段落分割
	paragraphs := strings.Split(text, "\n\n")

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		// 判断段落类型
		paragraphType := p.classifyParagraph(paragraph)

		// 提取段落标题
		title := p.extractParagraphTitle(paragraph)

		// 提取段落关键词
		keywords := p.extractKeywords(paragraph)

		section := Section{
			Title:    title,
			Content:  paragraph,
			Type:     paragraphType,
			Level:    p.determineLevel(paragraph),
			Keywords: keywords,
		}

		sections = append(sections, section)
	}

	return sections
}

// classifyParagraph 分类段落类型
func (p *documentParser) classifyParagraph(paragraph string) string {
	// 判断是否为标题
	if p.isTitle(paragraph) {
		return "标题"
	}

	// 判断是否为列表
	if p.isList(paragraph) {
		return "列表"
	}

	// 判断是否为引用
	if p.isQuote(paragraph) {
		return "引用"
	}

	// 默认为正文
	return "正文"
}

// isTitle 判断是否为标题
func (p *documentParser) isTitle(text string) bool {
	// 标题特征：较短、以数字开头、包含冒号等
	if len(text) < 100 && (strings.Contains(text, "：") || strings.Contains(text, ":") ||
		regexp.MustCompile(`^[0-9一二三四五六七八九十]+[、.．]`).MatchString(text)) {
		return true
	}

	return false
}

// isList 判断是否为列表
func (p *documentParser) isList(text string) bool {
	// 列表特征：以项目符号开头
	return regexp.MustCompile(`^[•·▪▫◦‣⁃➤➢➣➤➥➦➧➨➩➪➫➬➭➮➯➱➲➳➴➵➶➷➸➹➺➻➼➽➾➿]`).MatchString(text) ||
		regexp.MustCompile(`^[0-9]+[.．]`).MatchString(text) ||
		regexp.MustCompile(`^[一二三四五六七八九十]+[、.．]`).MatchString(text)
}

// isQuote 判断是否为引用
func (p *documentParser) isQuote(text string) bool {
	// 引用特征：以引号开头或缩进
	return strings.HasPrefix(text, "「") || strings.HasPrefix(text, "」") ||
		strings.HasPrefix(text, "'") || strings.HasPrefix(text, "\"")
}

// extractParagraphTitle 提取段落标题
func (p *documentParser) extractParagraphTitle(paragraph string) string {
	// 如果是标题段落，直接返回
	if p.isTitle(paragraph) {
		return paragraph
	}

	// 否则提取第一句话作为标题
	sentences := p.splitSentences(paragraph)
	if len(sentences) > 0 {
		title := strings.TrimSpace(sentences[0])
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		return title
	}

	return "无标题"
}

// determineLevel 确定层级
func (p *documentParser) determineLevel(paragraph string) int {
	// 基于段落特征确定层级
	if p.isTitle(paragraph) {
		// 检查是否包含数字编号
		if regexp.MustCompile(`^[0-9]+[.．]`).MatchString(paragraph) {
			return 1
		} else if regexp.MustCompile(`^[一二三四五六七八九十]+[、.．]`).MatchString(paragraph) {
			return 2
		} else {
			return 3
		}
	}

	return 0
}

// splitSentences 分割句子
func (p *documentParser) splitSentences(text string) []string {
	// 按句号、问号、感叹号分割
	re := regexp.MustCompile(`[。！？.!?]+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence != "" {
			result = append(result, sentence)
		}
	}

	return result
}

// calculateAverageWordLength 计算平均词长
func (p *documentParser) calculateAverageWordLength(text string) float64 {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	totalLength := 0
	for _, word := range words {
		totalLength += len([]rune(word))
	}

	return float64(totalLength) / float64(len(words))
}

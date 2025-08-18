package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// DocumentElement 文档元素
type DocumentElement struct {
	Text       string
	Type       ElementType
	StyleLevel int
	IsBold     bool
	FontSize   float64
	Position   ElementPosition
}

// ElementType 元素类型
type ElementType int

const (
	ElementTitle ElementType = iota
	ElementHeading
	ElementParagraph
	ElementList
	ElementQuote
)

// ElementPosition 元素位置信息
type ElementPosition struct {
	Page   int
	X, Y   float64
	Width  float64
	Height float64
}

// TextSegment定义在shared_types.go中

// EnhancedPDFParser 增强PDF解析器
type EnhancedPDFParser struct {
	keywordExtractor *KeywordExtractor
	structAnalyzer   *DocumentStructureAnalyzer
}

// NewEnhancedPDFParser 创建增强PDF解析器
func NewEnhancedPDFParser() *EnhancedPDFParser {
	return &EnhancedPDFParser{
		keywordExtractor: NewEnhancedKeywordExtractor(),
		structAnalyzer:   NewEnhancedDocumentStructureAnalyzer(),
	}
}

// ParsePDFToStructuredContent 解析PDF为结构化内容
func (p *EnhancedPDFParser) ParsePDFToStructuredContent(filePath string) (*StructuredContent, error) {
	// 1. 提取原始文本
	rawText, err := p.extractTextFromPDF(filePath)
	if err != nil {
		return nil, fmt.Errorf("提取PDF文本失败: %v", err)
	}

	if strings.TrimSpace(rawText) == "" {
		return nil, fmt.Errorf("PDF文件为空或无法提取文本")
	}

	// 2. 清理文本
	cleanText := p.cleanText(rawText)

	// 3. 分析文档结构
	sections, err := p.structAnalyzer.AnalyzeSections(cleanText)
	if err != nil {
		return nil, fmt.Errorf("分析文档结构失败: %v", err)
	}

	// 4. 提取关键词
	keywords := p.keywordExtractor.ExtractKeywords(cleanText)

	// 5. 生成关键信息
	keyInfo := &KeyInfo{
		Title:      p.extractTitle(cleanText),
		Summary:    p.generateSummary(cleanText),
		Keywords:   keywords,
		MainPoints: p.extractMainPoints(cleanText),
		Topics:     []string{"PDF文档"},
		Difficulty: "中等",
		Duration:   30,
		Structure:  []string{"文档解析", "内容分析"},
	}

	return &StructuredContent{
		OriginalText: rawText,
		CleanText:    cleanText,
		Sections:     sections,
		KeyInfo:      keyInfo,
	}, nil
}

// extractTextFromPDF 从PDF提取文本
func (p *EnhancedPDFParser) extractTextFromPDF(filePath string) (string, error) {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var textBuilder strings.Builder
	totalPages := reader.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		texts := page.Content().Text
		for _, text := range texts {
			if strings.TrimSpace(text.S) != "" {
				textBuilder.WriteString(text.S)
				textBuilder.WriteString(" ")
			}
		}
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// cleanText 清理文本
func (p *EnhancedPDFParser) cleanText(text string) string {
	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// 移除特殊字符但保留中文标点
	text = regexp.MustCompile(`[^\p{L}\p{N}\p{P}\p{Z}]`).ReplaceAllString(text, "")

	// 规范化标点符号
	text = strings.ReplaceAll(text, "。", "。\n")
	text = strings.ReplaceAll(text, "！", "！\n")
	text = strings.ReplaceAll(text, "？", "？\n")

	return strings.TrimSpace(text)
}

// extractTitle 提取标题
func (p *EnhancedPDFParser) extractTitle(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 5 && len(line) < 100 {
			// 检查是否包含常见的标题特征
			if !strings.Contains(line, "页码") &&
				!strings.Contains(line, "第") &&
				!regexp.MustCompile(`^\d+$`).MatchString(line) {
				return line
			}
		}
	}
	return "PDF文档"
}

// generateSummary 生成摘要
func (p *EnhancedPDFParser) generateSummary(text string) string {
	sentences := strings.Split(text, "。")
	if len(sentences) < 3 {
		return text
	}

	// 取前3个非空句子作为摘要
	var summary []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 10 && len(summary) < 3 {
			summary = append(summary, sentence+"。")
		}
	}

	return strings.Join(summary, "")
}

// extractMainPoints 提取要点
func (p *EnhancedPDFParser) extractMainPoints(text string) []string {
	var points []string

	// 查找列表模式
	listPatterns := []*regexp.Regexp{
		regexp.MustCompile(`[0-9]+[\.、]\s*([^。！？\n]+)`),
		regexp.MustCompile(`[一二三四五六七八九十][、\.]\s*([^。！？\n]+)`),
		regexp.MustCompile(`[①②③④⑤⑥⑦⑧⑨⑩]\s*([^。！？\n]+)`),
		regexp.MustCompile(`[（(][0-9一二三四五六七八九十]+[）)]\s*([^。！？\n]+)`),
	}

	for _, pattern := range listPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 && len(match[1]) > 5 {
				points = append(points, strings.TrimSpace(match[1]))
			}
		}
	}

	// 如果没有找到列表，提取重要句子
	if len(points) == 0 {
		sentences := strings.Split(text, "。")
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if len(sentence) > 15 && len(sentence) < 100 {
				// 检查是否包含重要关键词
				if strings.Contains(sentence, "重要") ||
					strings.Contains(sentence, "关键") ||
					strings.Contains(sentence, "核心") ||
					strings.Contains(sentence, "主要") {
					points = append(points, sentence)
					if len(points) >= 5 {
						break
					}
				}
			}
		}
	}

	// 限制要点数量
	if len(points) > 8 {
		points = points[:8]
	}

	return points
}

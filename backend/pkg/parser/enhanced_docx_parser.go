package parser

import (
	"archive/zip"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// EnhancedDOCXParser 增强DOCX解析器
type EnhancedDOCXParser struct {
	keywordExtractor *KeywordExtractor
	structAnalyzer   *DocumentStructureAnalyzer
}

// DocxDocument 表示DOCX文档结构
type DocxDocument struct {
	Body DocxBody `xml:"body"`
}

// DocxBody 表示文档主体
type DocxBody struct {
	Paragraphs []DocxParagraph `xml:"p"`
}

// DocxParagraph 表示段落
type DocxParagraph struct {
	Runs       []DocxRun               `xml:"r"`
	Properties DocxParagraphProperties `xml:"pPr"`
}

// DocxRun 表示文本运行
type DocxRun struct {
	Text       string            `xml:"t"`
	Properties DocxRunProperties `xml:"rPr"`
}

// DocxParagraphProperties 段落属性
type DocxParagraphProperties struct {
	Style DocxStyle `xml:"pStyle"`
}

// DocxRunProperties 文本运行属性
type DocxRunProperties struct {
	Bold   *struct{} `xml:"b"`
	Italic *struct{} `xml:"i"`
}

// DocxStyle 样式
type DocxStyle struct {
	Val string `xml:"val,attr"`
}

// NewEnhancedDOCXParser 创建增强DOCX解析器
func NewEnhancedDOCXParser() *EnhancedDOCXParser {
	return &EnhancedDOCXParser{
		keywordExtractor: NewEnhancedKeywordExtractor(),
		structAnalyzer:   NewEnhancedDocumentStructureAnalyzer(),
	}
}

// ParseDOCXToStructuredContent 解析DOCX为结构化内容
func (p *EnhancedDOCXParser) ParseDOCXToStructuredContent(filePath string) (*StructuredContent, error) {
	// 1. 读取DOCX文件
	rawText, err := p.extractTextFromDOCX(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开DOCX文件失败: %v", err)
	}

	// 2. 清理文本
	cleanText := p.cleanText(rawText)

	if strings.TrimSpace(cleanText) == "" {
		return nil, fmt.Errorf("DOCX文件为空或无法提取文本")
	}

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
		Topics:     []string{"DOCX文档"},
		Difficulty: "中等",
		Duration:   25,
		Structure:  p.extractStructureFromSections(sections),
	}

	return &StructuredContent{
		OriginalText: rawText,
		CleanText:    cleanText,
		Sections:     sections,
		KeyInfo:      keyInfo,
	}, nil
}

// extractTextFromDOCX 从DOCX文件提取文本
func (p *EnhancedDOCXParser) extractTextFromDOCX(filePath string) (string, error) {
	// 打开DOCX文件（实际上是ZIP格式）
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开DOCX文件: %v", err)
	}
	defer reader.Close()

	// 查找document.xml文件
	var documentXML []byte
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("无法打开document.xml: %v", err)
			}
			defer rc.Close()

			documentXML, err = io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("无法读取document.xml: %v", err)
			}
			break
		}
	}

	if len(documentXML) == 0 {
		return "", fmt.Errorf("在DOCX文件中未找到document.xml")
	}

	// 解析XML并提取文本
	return p.extractTextFromXML(documentXML)
}

// extractTextFromXML 从XML提取文本
func (p *EnhancedDOCXParser) extractTextFromXML(xmlData []byte) (string, error) {
	// 简化的XML文本提取
	// 移除XML标签，保留纯文本
	text := string(xmlData)

	// 使用正则表达式提取<w:t>标签中的文本
	re := regexp.MustCompile(`<w:t[^>]*>(.*?)</w:t>`)
	matches := re.FindAllStringSubmatch(text, -1)

	var textBuilder strings.Builder
	for _, match := range matches {
		if len(match) > 1 {
			// 处理XML实体
			textContent := strings.ReplaceAll(match[1], "&lt;", "<")
			textContent = strings.ReplaceAll(textContent, "&gt;", ">")
			textContent = strings.ReplaceAll(textContent, "&amp;", "&")
			textContent = strings.ReplaceAll(textContent, "&quot;", "\"")
			textContent = strings.ReplaceAll(textContent, "&apos;", "'")

			textBuilder.WriteString(textContent)
			textBuilder.WriteString(" ")
		}
	}

	// 如果没有找到<w:t>标签，尝试其他方法
	if textBuilder.Len() == 0 {
		// 移除所有XML标签
		re = regexp.MustCompile(`<[^>]*>`)
		plainText := re.ReplaceAllString(text, " ")

		// 清理多余的空格
		re = regexp.MustCompile(`\s+`)
		plainText = re.ReplaceAllString(plainText, " ")

		return strings.TrimSpace(plainText), nil
	}

	return strings.TrimSpace(textBuilder.String()), nil
}

// extractTitle 提取标题
func (p *EnhancedDOCXParser) extractTitle(text string) string {
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
	return "DOCX文档"
}

// cleanText 清理文本
func (p *EnhancedDOCXParser) cleanText(text string) string {
	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// 规范化标点符号
	text = strings.ReplaceAll(text, "。", "。\n")
	text = strings.ReplaceAll(text, "！", "！\n")
	text = strings.ReplaceAll(text, "？", "？\n")

	return strings.TrimSpace(text)
}

// generateSummary 生成摘要
func (p *EnhancedDOCXParser) generateSummary(text string) string {
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
func (p *EnhancedDOCXParser) extractMainPoints(text string) []string {
	var points []string

	// 查找列表模式
	listPatterns := []*regexp.Regexp{
		regexp.MustCompile(`[0-9]+[\.、]\s*([^。！？\n]+)`),
		regexp.MustCompile(`[一二三四五六七八九十][、\.]\s*([^。！？\n]+)`),
		regexp.MustCompile(`[①②③④⑤⑥⑦⑧⑨⑩]\s*([^。！？\n]+)`),
		regexp.MustCompile(`[-•*]\s*([^。！？\n]+)`),
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

// extractStructureFromSections 从章节提取结构信息
func (p *EnhancedDOCXParser) extractStructureFromSections(sections []Section) []string {
	var structure []string

	for _, section := range sections {
		if section.Type == "chapter" {
			structure = append(structure, "章节: "+section.Title)
		} else if section.Type == "section" {
			structure = append(structure, "小节: "+section.Title)
		}
	}

	if len(structure) == 0 {
		structure = []string{"文档解析", "内容分析"}
	}

	return structure
}

package ppt_parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SlideData 幻灯片数据结构
type SlideData struct {
	SlideNumber  int      `json:"slide_number"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Keywords     []string `json:"keywords"`
	SpeakerNotes string   `json:"speaker_notes"`
	Duration     int      `json:"duration"`
}

// PPTMetadata PPT元数据
type PPTMetadata struct {
	CourseID      int       `json:"course_id"`
	SourceType    string    `json:"source_type"`
	GeneratedAt   time.Time `json:"generated_at"`
	Template      string    `json:"template"`
	TotalSlides   int       `json:"total_slides"`
	FilePath      string    `json:"file_path"`
	OriginalTitle string    `json:"original_title"`
}

// HTMLParser HTML解析器
type HTMLParser struct{}

// NewHTMLParser 创建HTML解析器实例
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// ParsePPTFile 解析PPT HTML文件
func (p *HTMLParser) ParsePPTFile(filePath, htmlContent string) (*PPTMetadata, []SlideData, error) {
	// 1. 解析文件名获取元数据
	metadata, err := p.extractMetadataFromFileName(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("解析文件名失败: %w", err)
	}

	// 2. 解析HTML内容
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	// 3. 提取PPT标题
	metadata.OriginalTitle = doc.Find("title").Text()

	// 4. 提取幻灯片数据
	slides, err := p.extractSlides(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("提取幻灯片失败: %w", err)
	}

	// 5. 从metadata区域提取额外信息
	if err := p.extractMetadataFromHTML(doc, metadata); err != nil {
		// 非致命错误，记录日志但继续处理
		fmt.Printf("警告: 提取HTML元数据失败: %v\n", err)
	}

	metadata.TotalSlides = len(slides)
	metadata.FilePath = filePath

	return metadata, slides, nil
}

// extractMetadataFromFileName 从文件名提取元数据
func (p *HTMLParser) extractMetadataFromFileName(filePath string) (*PPTMetadata, error) {
	// 从路径中提取文件名
	parts := strings.Split(filePath, "/")
	fileName := parts[len(parts)-1]

	// 解析文件名格式: ppt_{course_id}_{source_type}_{timestamp}.html
	re := regexp.MustCompile(`ppt_(\d+)_([^_]+)_(\d{8}_\d{6})\.html`)
	matches := re.FindStringSubmatch(fileName)

	if len(matches) != 4 {
		return nil, fmt.Errorf("文件名格式不正确: %s, 应为 ppt_{course_id}_{source_type}_{timestamp}.html", fileName)
	}

	courseID, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("解析课程ID失败: %w", err)
	}

	sourceType := matches[2]
	timestamp := matches[3]

	// 解析时间戳
	generatedAt, err := time.Parse("20060102_150405", timestamp)
	if err != nil {
		// 如果时间解析失败，使用当前时间
		generatedAt = time.Now()
		fmt.Printf("警告: 时间戳解析失败，使用当前时间: %v\n", err)
	}

	return &PPTMetadata{
		CourseID:    courseID,
		SourceType:  sourceType,
		GeneratedAt: generatedAt,
	}, nil
}

// extractSlides 提取幻灯片数据
func (p *HTMLParser) extractSlides(doc *goquery.Document) ([]SlideData, error) {
	var slides []SlideData

	doc.Find(".slide").Each(func(i int, s *goquery.Selection) {
		slide := SlideData{}

		// 提取幻灯片序号
		slideNumberText := s.Find(".slide-number").Text()
		if slideNumberText != "" {
			// 从"第 X 页"中提取数字
			re := regexp.MustCompile(`第\s*(\d+)\s*页`)
			if matches := re.FindStringSubmatch(slideNumberText); len(matches) > 1 {
				if num, err := strconv.Atoi(matches[1]); err == nil {
					slide.SlideNumber = num
				}
			}
		}
		// 如果无法从文本中提取，使用索引+1
		if slide.SlideNumber == 0 {
			slide.SlideNumber = i + 1
		}

		// 提取标题
		slide.Title = strings.TrimSpace(s.Find(".slide-title").Text())

		// 提取内容
		slide.Content = p.extractSlideContent(s)

		// 提取关键词
		slide.Keywords = p.extractKeywords(s)

		// 提取演讲备注
		slide.SpeakerNotes = p.extractSpeakerNotes(s)

		// 估算时长（基于内容长度）
		slide.Duration = p.estimateDuration(slide.Title, slide.Content, slide.SpeakerNotes)

		slides = append(slides, slide)
	})

	return slides, nil
}

// extractSlideContent 提取幻灯片内容
func (p *HTMLParser) extractSlideContent(s *goquery.Selection) string {
	contentElement := s.Find(".slide-content")

	var content strings.Builder

	// 处理列表项
	contentElement.Find("li").Each(func(i int, li *goquery.Selection) {
		text := strings.TrimSpace(li.Text())
		if text != "" {
			content.WriteString("• ")
			content.WriteString(text)
			content.WriteString("\n")
		}
	})

	// 处理段落
	contentElement.Find("p").Each(func(i int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())
		if text != "" {
			content.WriteString(text)
			content.WriteString("\n")
		}
	})

	// 如果没有结构化内容，直接获取文本
	result := strings.TrimSpace(content.String())
	if result == "" {
		result = strings.TrimSpace(contentElement.Text())
	}

	return result
}

// extractKeywords 提取关键词
func (p *HTMLParser) extractKeywords(s *goquery.Selection) []string {
	var keywords []string

	s.Find(".keyword-tag").Each(func(i int, tag *goquery.Selection) {
		keyword := strings.TrimSpace(tag.Text())
		if keyword != "" && len(keyword) < 100 { // 过滤过长的关键词
			keywords = append(keywords, keyword)
		}
	})

	return keywords
}

// extractSpeakerNotes 提取演讲备注
func (p *HTMLParser) extractSpeakerNotes(s *goquery.Selection) string {
	notesElement := s.Find(".slide-notes")
	if notesElement.Length() == 0 {
		return ""
	}

	// 移除"备注:"标签，只保留内容
	notes := notesElement.Text()
	notes = strings.TrimSpace(notes)

	// 移除"备注:"前缀
	if strings.HasPrefix(notes, "备注:") {
		notes = strings.TrimSpace(notes[3:])
	}

	return notes
}

// extractMetadataFromHTML 从HTML中提取元数据
func (p *HTMLParser) extractMetadataFromHTML(doc *goquery.Document, metadata *PPTMetadata) error {
	metadataElement := doc.Find(".metadata")
	if metadataElement.Length() == 0 {
		return fmt.Errorf("未找到元数据区域")
	}

	metadataText := metadataElement.Text()

	// 提取模板信息
	templateRe := regexp.MustCompile(`模板:\s*(\w+)`)
	if matches := templateRe.FindStringSubmatch(metadataText); len(matches) > 1 {
		metadata.Template = matches[1]
	}

	// 提取生成时间（如果HTML中的时间更准确）
	timeRe := regexp.MustCompile(`生成时间:\s*([0-9-:\s]+)`)
	if matches := timeRe.FindStringSubmatch(metadataText); len(matches) > 1 {
		if t, err := time.Parse("2006-01-02 15:04:05", matches[1]); err == nil {
			metadata.GeneratedAt = t
		}
	}

	// 提取总页数（验证）
	pagesRe := regexp.MustCompile(`总页数:\s*(\d+)`)
	if matches := pagesRe.FindStringSubmatch(metadataText); len(matches) > 1 {
		if totalPages, err := strconv.Atoi(matches[1]); err == nil {
			// 可以用来验证解析的幻灯片数量是否正确
			if totalPages != metadata.TotalSlides {
				fmt.Printf("警告: HTML元数据中的总页数(%d)与实际解析的幻灯片数量(%d)不匹配\n",
					totalPages, metadata.TotalSlides)
			}
		}
	}

	return nil
}

// estimateDuration 估算幻灯片播放时长
func (p *HTMLParser) estimateDuration(title, content, notes string) int {
	// 计算总文本长度
	totalText := title + " " + content + " " + notes
	textLength := len([]rune(totalText))

	if textLength == 0 {
		return 10 // 默认10秒
	}

	// 估算规则：
	// - 中文约每分钟200字
	// - 基础时间5秒
	// - 最长不超过5分钟
	estimatedSeconds := 5 + (textLength * 60 / 200)

	// 限制范围：5秒到300秒
	if estimatedSeconds < 5 {
		estimatedSeconds = 5
	} else if estimatedSeconds > 300 {
		estimatedSeconds = 300
	}

	return estimatedSeconds
}

// ValidateSlideData 验证幻灯片数据
func (p *HTMLParser) ValidateSlideData(slides []SlideData) error {
	if len(slides) == 0 {
		return fmt.Errorf("没有找到有效的幻灯片")
	}

	for i, slide := range slides {
		if slide.Title == "" {
			return fmt.Errorf("第%d张幻灯片缺少标题", i+1)
		}
		if slide.SlideNumber <= 0 {
			return fmt.Errorf("第%d张幻灯片序号无效: %d", i+1, slide.SlideNumber)
		}
	}

	return nil
}

// GetSupportedSourceTypes 获取支持的来源类型
func (p *HTMLParser) GetSupportedSourceTypes() []string {
	return []string{"file", "url", "text", "anonymous"}
}

// ExtractMetadataFromFileName 公开方法：从文件名提取元数据
func (p *HTMLParser) ExtractMetadataFromFileName(filePath string) (*PPTMetadata, error) {
	return p.extractMetadataFromFileName(filePath)
}

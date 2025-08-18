package coze

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
)

// PPTXParser PPTX文件解析器
type PPTXParser struct {
	zipReader *zip.ReadCloser
}

// PPTXSlideContent PPTX幻灯片内容
type PPTXSlideContent struct {
	SlideNumber int      `json:"slide_number"`
	Title       string   `json:"title"`
	Content     []string `json:"content"`
	Notes       string   `json:"notes"`
}

// PPTXPresentationInfo PPTX演示文稿信息
type PPTXPresentationInfo struct {
	Title       string             `json:"title"`
	Author      string             `json:"author"`
	SlideCount  int                `json:"slide_count"`
	Slides      []PPTXSlideContent `json:"slides"`
}

// NewPPTXParser 创建PPTX解析器
func NewPPTXParser(pptxPath string) (*PPTXParser, error) {
	zipReader, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开PPTX文件: %w", err)
	}

	return &PPTXParser{
		zipReader: zipReader,
	}, nil
}

// Close 关闭解析器
func (p *PPTXParser) Close() error {
	if p.zipReader != nil {
		return p.zipReader.Close()
	}
	return nil
}

// ParsePresentation 解析演示文稿
func (p *PPTXParser) ParsePresentation() (*PPTXPresentationInfo, error) {
	info := &PPTXPresentationInfo{
		Slides: []PPTXSlideContent{},
	}

	// 1. 解析文档属性获取标题和作者
	if err := p.parseDocumentProperties(info); err != nil {
		log.Printf("解析文档属性失败: %v", err)
	}

	// 2. 获取幻灯片数量
	slideFiles := p.getSlideFiles()
	info.SlideCount = len(slideFiles)

	// 3. 解析每张幻灯片
	for i, slideFile := range slideFiles {
		slideContent, err := p.parseSlide(slideFile, i+1)
		if err != nil {
			log.Printf("解析幻灯片 %d 失败: %v", i+1, err)
			// 创建一个基础的幻灯片内容
			slideContent = &PPTXSlideContent{
				SlideNumber: i + 1,
				Title:       fmt.Sprintf("幻灯片 %d", i+1),
				Content:     []string{"内容解析失败"},
			}
		}
		info.Slides = append(info.Slides, *slideContent)
	}

	// 如果没有找到标题，使用默认标题
	if info.Title == "" {
		info.Title = "PowerPoint演示文稿"
	}

	return info, nil
}

// parseDocumentProperties 解析文档属性
func (p *PPTXParser) parseDocumentProperties(info *PPTXPresentationInfo) error {
	// 查找docProps/core.xml文件
	for _, file := range p.zipReader.File {
		if file.Name == "docProps/core.xml" {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return err
			}

			// 解析XML获取标题和作者
			var coreProps struct {
				Title   string `xml:"title"`
				Creator string `xml:"creator"`
			}

			if err := xml.Unmarshal(content, &coreProps); err == nil {
				info.Title = coreProps.Title
				info.Author = coreProps.Creator
			}
			break
		}
	}
	return nil
}

// getSlideFiles 获取所有幻灯片文件
func (p *PPTXParser) getSlideFiles() []*zip.File {
	var slideFiles []*zip.File
	slidePattern := regexp.MustCompile(`^ppt/slides/slide\d+\.xml$`)

	for _, file := range p.zipReader.File {
		if slidePattern.MatchString(file.Name) {
			slideFiles = append(slideFiles, file)
		}
	}

	return slideFiles
}

// parseSlide 解析单张幻灯片
func (p *PPTXParser) parseSlide(file *zip.File, slideNumber int) (*PPTXSlideContent, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	slideContent := &PPTXSlideContent{
		SlideNumber: slideNumber,
		Content:     []string{},
	}

	// 解析XML获取文本内容
	textContent := p.extractTextFromSlideXML(string(content))
	
	// 分离标题和内容
	if len(textContent) > 0 {
		slideContent.Title = textContent[0]
		if len(textContent) > 1 {
			slideContent.Content = textContent[1:]
		}
	}

	// 如果没有标题，设置默认标题
	if slideContent.Title == "" {
		slideContent.Title = fmt.Sprintf("幻灯片 %d", slideNumber)
	}

	return slideContent, nil
}

// extractTextFromSlideXML 从幻灯片XML中提取文本
func (p *PPTXParser) extractTextFromSlideXML(xmlContent string) []string {
	var texts []string

	// 使用正则表达式提取<a:t>标签中的文本
	textPattern := regexp.MustCompile(`<a:t[^>]*>([^<]*)</a:t>`)
	matches := textPattern.FindAllStringSubmatch(xmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			if text != "" {
				texts = append(texts, text)
			}
		}
	}

	// 如果没有找到文本，尝试提取其他文本节点
	if len(texts) == 0 {
		// 尝试提取更通用的文本内容
		generalTextPattern := regexp.MustCompile(`>([^<]{3,})<`)
		matches = generalTextPattern.FindAllStringSubmatch(xmlContent, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				text := strings.TrimSpace(match[1])
				// 过滤掉XML标签名和属性
				if text != "" && !strings.Contains(text, "xmlns") && !strings.Contains(text, "=") {
					texts = append(texts, text)
				}
			}
		}
	}

	return texts
}

// ConvertToHTMLSlides 将PPTX内容转换为HTML幻灯片
func (p *PPTXParser) ConvertToHTMLSlides() ([]HTMLPPTSlide, error) {
	info, err := p.ParsePresentation()
	if err != nil {
		return nil, err
	}

	var htmlSlides []HTMLPPTSlide

	for _, slide := range info.Slides {
		htmlSlide := HTMLPPTSlide{
			ID:         fmt.Sprintf("slide-%d", slide.SlideNumber),
			Title:      slide.Title,
			Content:    p.formatSlideContent(slide),
			Transition: "slide",
			Duration:   5000,
			Layout:     "text-content",
			Elements:   p.createSlideElements(slide),
		}

		htmlSlides = append(htmlSlides, htmlSlide)
	}

	return htmlSlides, nil
}

// formatSlideContent 格式化幻灯片内容
func (p *PPTXParser) formatSlideContent(slide PPTXSlideContent) string {
	var content strings.Builder
	
	content.WriteString(fmt.Sprintf("<h2>%s</h2>", slide.Title))
	
	if len(slide.Content) > 0 {
		content.WriteString("<div class='slide-content'>")
		for _, text := range slide.Content {
			content.WriteString(fmt.Sprintf("<p>%s</p>", text))
		}
		content.WriteString("</div>")
	}

	if slide.Notes != "" {
		content.WriteString(fmt.Sprintf("<div class='speaker-notes' style='display:none;'>%s</div>", slide.Notes))
	}

	return content.String()
}

// createSlideElements 创建幻灯片元素
func (p *PPTXParser) createSlideElements(slide PPTXSlideContent) []SlideElement {
	var elements []SlideElement

	// 标题元素
	titleElement := SlideElement{
		ID:      fmt.Sprintf("title-%d", slide.SlideNumber),
		Type:    "text",
		Content: fmt.Sprintf("<h2>%s</h2>", slide.Title),
		Position: ElementPosition{
			X:      50,
			Y:      100,
			Width:  800,
			Height: 80,
			ZIndex: 1,
		},
		Style: map[string]interface{}{
			"text-align": "center",
			"font-size":  "36px",
			"font-weight": "bold",
			"color":      "#333",
		},
	}
	elements = append(elements, titleElement)

	// 内容元素
	if len(slide.Content) > 0 {
		contentHTML := "<div class='content-list'>"
		for _, text := range slide.Content {
			contentHTML += fmt.Sprintf("<p>• %s</p>", text)
		}
		contentHTML += "</div>"

		contentElement := SlideElement{
			ID:      fmt.Sprintf("content-%d", slide.SlideNumber),
			Type:    "text",
			Content: contentHTML,
			Position: ElementPosition{
				X:      100,
				Y:      200,
				Width:  700,
				Height: 400,
				ZIndex: 1,
			},
			Style: map[string]interface{}{
				"text-align": "left",
				"font-size":  "24px",
				"line-height": "1.6",
				"color":      "#444",
			},
		}
		elements = append(elements, contentElement)
	}

	return elements
}
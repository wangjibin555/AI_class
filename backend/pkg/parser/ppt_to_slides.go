package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Slide 幻灯片结构
type Slide struct {
	ID          uint      `json:"id"`
	CourseID    uint      `json:"course_id"`
	SlideNumber int       `json:"slide_number"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Keywords    []string  `json:"keywords"`
	Notes       string    `json:"notes"`
	Duration    int       `json:"duration"`  // 预计播放时长（秒）
	AudioURL    string    `json:"audio_url"` // 音频文件URL
	ImageURL    string    `json:"image_url"` // 缩略图URL
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PPTMetadata PPT元数据
type PPTMetadata struct {
	Title       string    `json:"title"`
	TotalSlides int       `json:"total_slides"`
	GeneratedAt time.Time `json:"generated_at"`
	Template    string    `json:"template"`
	Keywords    []string  `json:"keywords"`
}

// PPTToSlidesParser PPT转幻灯片解析器
type PPTToSlidesParser struct {
	htmlContent string
	metadata    PPTMetadata
	slides      []Slide
}

// NewPPTToSlidesParser 创建新的PPT解析器
func NewPPTToSlidesParser() *PPTToSlidesParser {
	return &PPTToSlidesParser{
		slides: make([]Slide, 0),
	}
}

// ParsePPTFile 解析PPT HTML文件
func (p *PPTToSlidesParser) ParsePPTFile(filePath string) ([]Slide, PPTMetadata, error) {
	// 读取HTML文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, PPTMetadata{}, fmt.Errorf("读取PPT文件失败: %w", err)
	}

	p.htmlContent = string(content)

	// 解析元数据
	if err := p.parseMetadata(); err != nil {
		return nil, PPTMetadata{}, fmt.Errorf("解析元数据失败: %w", err)
	}

	// 解析幻灯片
	if err := p.parseSlides(); err != nil {
		return nil, PPTMetadata{}, fmt.Errorf("解析幻灯片失败: %w", err)
	}

	return p.slides, p.metadata, nil
}

// parseMetadata 解析PPT元数据
func (p *PPTToSlidesParser) parseMetadata() error {
	// 解析标题
	titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
	if matches := titleRegex.FindStringSubmatch(p.htmlContent); len(matches) > 1 {
		p.metadata.Title = strings.TrimSpace(matches[1])
	}

	// 解析生成时间
	timeRegex := regexp.MustCompile(`生成时间: (\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`)
	if matches := timeRegex.FindStringSubmatch(p.htmlContent); len(matches) > 1 {
		if t, err := time.Parse("2006-01-02 15:04:05", matches[1]); err == nil {
			p.metadata.GeneratedAt = t
		}
	}

	// 解析模板
	templateRegex := regexp.MustCompile(`模板: (\w+)`)
	if matches := templateRegex.FindStringSubmatch(p.htmlContent); len(matches) > 1 {
		p.metadata.Template = matches[1]
	}

	// 解析总页数
	pageRegex := regexp.MustCompile(`总页数: (\d+)`)
	if matches := pageRegex.FindStringSubmatch(p.htmlContent); len(matches) > 1 {
		if total, err := strconv.Atoi(matches[1]); err == nil {
			p.metadata.TotalSlides = total
		}
	}

	return nil
}

// parseSlides 解析幻灯片内容
func (p *PPTToSlidesParser) parseSlides() error {
	doc, err := html.Parse(strings.NewReader(p.htmlContent))
	if err != nil {
		return fmt.Errorf("解析HTML失败: %w", err)
	}

	// 查找所有幻灯片div
	var slides []*html.Node
	var findSlides func(*html.Node)
	findSlides = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "slide") {
					slides = append(slides, n)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSlides(c)
		}
	}
	findSlides(doc)

	// 解析每个幻灯片
	for i, slideNode := range slides {
		slide, err := p.parseSingleSlide(slideNode, i+1)
		if err != nil {
			fmt.Printf("解析第%d张幻灯片失败: %v\n", i+1, err)
			continue
		}
		p.slides = append(p.slides, slide)
	}

	return nil
}

// parseSingleSlide 解析单个幻灯片
func (p *PPTToSlidesParser) parseSingleSlide(slideNode *html.Node, slideNumber int) (Slide, error) {
	slide := Slide{
		SlideNumber: slideNumber,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 解析标题
	slide.Title = p.extractSlideTitle(slideNode)

	// 解析内容
	slide.Content = p.extractSlideContent(slideNode)

	// 解析关键词
	slide.Keywords = p.extractSlideKeywords(slideNode)

	// 解析备注
	slide.Notes = p.extractSlideNotes(slideNode)

	// 计算预计播放时长
	slide.Duration = p.calculateSlideDuration(slide.Content, slide.Notes)

	return slide, nil
}

// extractSlideTitle 提取幻灯片标题
func (p *PPTToSlidesParser) extractSlideTitle(slideNode *html.Node) string {
	var title string
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h1" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "slide-title") {
					title = p.extractText(n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(slideNode)
	return strings.TrimSpace(title)
}

// extractSlideContent 提取幻灯片内容
func (p *PPTToSlidesParser) extractSlideContent(slideNode *html.Node) string {
	var content string
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "slide-content") {
					content = p.extractText(n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findContent(c)
		}
	}
	findContent(slideNode)
	return strings.TrimSpace(content)
}

// extractSlideKeywords 提取幻灯片关键词
func (p *PPTToSlidesParser) extractSlideKeywords(slideNode *html.Node) []string {
	var keywords []string
	var findKeywords func(*html.Node)
	findKeywords = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "keyword-tag") {
					keyword := p.extractText(n)
					if keyword != "" {
						keywords = append(keywords, strings.TrimSpace(keyword))
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findKeywords(c)
		}
	}
	findKeywords(slideNode)
	return keywords
}

// extractSlideNotes 提取幻灯片备注
func (p *PPTToSlidesParser) extractSlideNotes(slideNode *html.Node) string {
	var notes string
	var findNotes func(*html.Node)
	findNotes = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "slide-notes") {
					notes = p.extractText(n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNotes(c)
		}
	}
	findNotes(slideNode)
	return strings.TrimSpace(notes)
}

// extractText 提取节点文本内容
func (p *PPTToSlidesParser) extractText(n *html.Node) string {
	var buf bytes.Buffer
	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.TextNode {
			buf.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return buf.String()
}

// calculateSlideDuration 计算幻灯片预计播放时长
func (p *PPTToSlidesParser) calculateSlideDuration(content, notes string) int {
	// 基础时长：标题 + 内容 + 备注
	baseDuration := 5 // 基础5秒

	// 根据内容长度计算额外时长
	contentLength := len(content) + len(notes)
	additionalDuration := contentLength / 50 // 每50个字符增加1秒

	// 根据关键词数量调整
	keywordCount := len(strings.Split(content, " ")) / 10 // 粗略估算关键词数量
	keywordDuration := keywordCount * 2

	totalDuration := baseDuration + additionalDuration + keywordDuration

	// 限制在合理范围内
	if totalDuration < 10 {
		totalDuration = 10
	} else if totalDuration > 120 {
		totalDuration = 120
	}

	return totalDuration
}

// GenerateSlidesFromPPT 从PPT文件生成幻灯片数据
func GenerateSlidesFromPPT(pptFilePath string, courseID uint) ([]Slide, PPTMetadata, error) {
	parser := NewPPTToSlidesParser()
	slides, metadata, err := parser.ParsePPTFile(pptFilePath)
	if err != nil {
		return nil, PPTMetadata{}, err
	}

	// 设置课程ID
	for i := range slides {
		slides[i].CourseID = courseID
	}

	return slides, metadata, nil
}

// BatchProcessPPTFiles 批量处理PPT文件
func BatchProcessPPTFiles(pptDir string) (map[string][]Slide, error) {
	results := make(map[string][]Slide)

	// 遍历PPT目录
	err := filepath.Walk(pptDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理HTML文件
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			parser := NewPPTToSlidesParser()
			slides, _, err := parser.ParsePPTFile(path)
			if err != nil {
				fmt.Printf("处理文件 %s 失败: %v\n", path, err)
				return nil // 继续处理其他文件
			}

			// 使用文件名作为key
			fileName := filepath.Base(path)
			results[fileName] = slides
		}

		return nil
	})

	return results, err
}

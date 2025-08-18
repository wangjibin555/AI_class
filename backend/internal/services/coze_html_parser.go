package services

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-classroom/internal/models"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"
)

// CozeHTMLParser Coze HTML解析器
type CozeHTMLParser struct {
	db *gorm.DB
}

// NewCozeHTMLParser 创建新的Coze HTML解析器
func NewCozeHTMLParser(db *gorm.DB) *CozeHTMLParser {
	return &CozeHTMLParser{
		db: db,
	}
}

// CozeSlideContent Coze幻灯片内容
type CozeSlideContent struct {
	SlideNumber int    `json:"slide_number"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	HTMLContent string `json:"html_content"` // 原始HTML内容
}

// ParsedCozeResult Coze解析结果
type ParsedCozeResult struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Slides      []CozeSlideContent `json:"slides"`
	SourceURL   string             `json:"source_url"`
	Theme       string             `json:"theme"`
}

// ParseHTMLToCourse 解析Coze生成的HTML文件为Course实体
func (p *CozeHTMLParser) ParseHTMLToCourse(htmlContent string, userID uint, sourceURL string) (*models.Course, error) {
	log.Printf("🔍 [Coze解析] 开始解析HTML内容，用户ID: %d", userID)

	// 1. 解析HTML内容
	parsedResult, err := p.parseHTML(htmlContent)
	if err != nil {
		log.Printf("❌ [Coze解析] HTML解析失败: %v", err)
		return nil, fmt.Errorf("HTML解析失败: %v", err)
	}

	// 2. 设置来源URL
	if sourceURL != "" {
		parsedResult.SourceURL = sourceURL
	}

	log.Printf("✅ [Coze解析] HTML解析完成: 标题=%s, 幻灯片数量=%d", parsedResult.Title, len(parsedResult.Slides))

	// 3. 创建Course实体
	course, err := p.createCourseFromParsedResult(parsedResult, userID)
	if err != nil {
		log.Printf("❌ [Coze解析] 创建Course实体失败: %v", err)
		return nil, fmt.Errorf("创建Course实体失败: %v", err)
	}

	log.Printf("✅ [Coze解析] Course创建成功，课程ID: %d", course.ID)
	return course, nil
}

// parseHTML 解析HTML内容
func (p *CozeHTMLParser) parseHTML(htmlContent string) (*ParsedCozeResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %v", err)
	}

	result := &ParsedCozeResult{
		Theme: "professional", // 默认主题
	}

	// 1. 提取标题
	result.Title = p.extractTitle(doc)

	// 2. 提取幻灯片内容
	slides, err := p.extractSlides(doc)
	if err != nil {
		return nil, fmt.Errorf("提取幻灯片失败: %v", err)
	}
	result.Slides = slides

	// 3. 生成描述
	result.Description = p.generateDescription(result.Title, len(slides), result.SourceURL)

	return result, nil
}

// extractTitle 提取PPT标题
func (p *CozeHTMLParser) extractTitle(doc *goquery.Document) string {
	// 优先从第一张幻灯片的h2标签提取
	firstSlideTitle := doc.Find("#slide-1 h2").First().Text()
	if firstSlideTitle != "" {
		log.Printf("🎯 [标题提取] 从第一张幻灯片提取标题: %s", firstSlideTitle)
		return strings.TrimSpace(firstSlideTitle)
	}

	// 备选：从任意第一个section的h2标签提取
	firstSectionTitle := doc.Find(".slides section:first-child h2").First().Text()
	if firstSectionTitle != "" {
		log.Printf("🎯 [标题提取] 从第一个section提取标题: %s", firstSectionTitle)
		return strings.TrimSpace(firstSectionTitle)
	}

	// 再备选：从所有section中查找第一个h2
	anySlideTitle := doc.Find(".slides section h2").First().Text()
	if anySlideTitle != "" {
		log.Printf("🎯 [标题提取] 从任意幻灯片提取标题: %s", anySlideTitle)
		return strings.TrimSpace(anySlideTitle)
	}

	// 最后：从title标签提取（虽然通常是"PPT演示文稿"）
	pageTitle := doc.Find("title").Text()
	if pageTitle != "" && pageTitle != "PPT演示文稿" {
		log.Printf("🎯 [标题提取] 从HTML title提取标题: %s", pageTitle)
		return strings.TrimSpace(pageTitle)
	}

	// 默认标题
	log.Printf("⚠️ [标题提取] 未找到有效标题，使用默认标题")
	return "AI生成的课程"
}

// extractSlides 提取所有幻灯片内容
func (p *CozeHTMLParser) extractSlides(doc *goquery.Document) ([]CozeSlideContent, error) {
	var slides []CozeSlideContent

	// 查找所有section元素（幻灯片）
	doc.Find("section[id^='slide-']").Each(func(i int, section *goquery.Selection) {
		slideID, exists := section.Attr("id")
		if !exists {
			return
		}

		// 提取幻灯片编号
		slideNumber := p.extractSlideNumber(slideID)
		if slideNumber == 0 {
			slideNumber = i + 1 // 备选方案
		}

		// 提取标题
		title := section.Find("h1, h2").First().Text()
		if title == "" {
			title = fmt.Sprintf("幻灯片 %d", slideNumber)
		}

		// 提取内容
		content := p.extractSlideContent(section)

		// 保存原始HTML内容
		htmlContent, _ := section.Html()

		slide := CozeSlideContent{
			SlideNumber: slideNumber,
			Title:       strings.TrimSpace(title),
			Content:     strings.TrimSpace(content),
			HTMLContent: htmlContent,
		}

		slides = append(slides, slide)
		log.Printf("🔍 [幻灯片解析] 第%d张: %s (内容长度: %d)", slideNumber, slide.Title, len(slide.Content))
	})

	if len(slides) == 0 {
		return nil, fmt.Errorf("未找到有效的幻灯片内容")
	}

	return slides, nil
}

// extractSlideNumber 从slide ID中提取编号
func (p *CozeHTMLParser) extractSlideNumber(slideID string) int {
	re := regexp.MustCompile(`slide-(\d+)`)
	matches := re.FindStringSubmatch(slideID)
	if len(matches) >= 2 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return num
		}
	}
	return 0
}

// extractSlideContent 提取幻灯片文本内容
func (p *CozeHTMLParser) extractSlideContent(section *goquery.Selection) string {
	var contentParts []string

	// 提取所有文本内容，排除标题
	section.Find("div, p, li").Each(func(i int, elem *goquery.Selection) {
		// 跳过标题元素
		if elem.Is("h1, h2, h3, h4, h5, h6") {
			return
		}

		text := strings.TrimSpace(elem.Text())
		if text != "" && !p.isContentDuplicate(text, contentParts) {
			contentParts = append(contentParts, text)
		}
	})

	// 合并内容
	content := strings.Join(contentParts, "\n")

	// 清理和格式化内容
	content = p.cleanContent(content)

	return content
}

// isContentDuplicate 检查内容是否重复
func (p *CozeHTMLParser) isContentDuplicate(text string, existingParts []string) bool {
	for _, part := range existingParts {
		if strings.Contains(part, text) || strings.Contains(text, part) {
			return true
		}
	}
	return false
}

// cleanContent 清理和格式化内容
func (p *CozeHTMLParser) cleanContent(content string) string {
	// 移除多余的换行符
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")

	// 移除行首行尾空白
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// generateDescription 生成课程描述
func (p *CozeHTMLParser) generateDescription(title string, slideCount int, sourceURL string) string {
	baseDescription := fmt.Sprintf("%s - 本课程包含%d张幻灯片，通过Coze AI智能体精心生成，涵盖核心知识点和实践要点。", title, slideCount)

	// 如果有源URL，尝试从中提取更多语义信息
	if sourceURL != "" {
		sourceInfo := p.extractSourceInfo(sourceURL)
		if sourceInfo != "" {
			baseDescription += fmt.Sprintf(" 基于%s的内容。", sourceInfo)
		}
	}

	return baseDescription
}

// extractSourceInfo 从源URL中提取语义信息
func (p *CozeHTMLParser) extractSourceInfo(sourceURL string) string {
	if sourceURL == "" {
		return ""
	}

	// 提取域名信息
	var domain string
	if strings.Contains(sourceURL, "blog.csdn.net") {
		domain = "CSDN博客"
	} else if strings.Contains(sourceURL, "github.com") {
		domain = "GitHub"
	} else if strings.Contains(sourceURL, "stackoverflow.com") {
		domain = "Stack Overflow"
	} else if strings.Contains(sourceURL, "medium.com") {
		domain = "Medium"
	} else if strings.Contains(sourceURL, "juejin.cn") {
		domain = "掘金"
	} else if strings.Contains(sourceURL, "zhihu.com") {
		domain = "知乎"
	} else if strings.Contains(sourceURL, "jianshu.com") {
		domain = "简书"
	} else {
		// 提取基本域名
		if strings.HasPrefix(sourceURL, "http") {
			parts := strings.Split(sourceURL, "/")
			if len(parts) >= 3 {
				domain = parts[2]
			}
		}
	}

	if domain != "" {
		return domain
	}

	return "网络资源"
}

// createCourseFromParsedResult 从解析结果创建Course实体
func (p *CozeHTMLParser) createCourseFromParsedResult(result *ParsedCozeResult, userID uint) (*models.Course, error) {
	// 1. 创建Course实体
	course := &models.Course{
		UserID:      userID,
		Title:       result.Title,
		Description: result.Description,
		Category:    models.CategoryGeneral, // 默认分类
		SourceType:  models.SourceTypeURL,
		SourceURL:   result.SourceURL,
		Status:      models.CourseStatusGenerating,
		SlidesCount: len(result.Slides),
		IsPublic:    false, // 默认私有
		GenerationParams: models.GenerationParams{
			"engine":    "coze",
			"template":  result.Theme,
			"source":    "coze_html_parser",
			"parsed_at": time.Now().Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 2. 保存Course到数据库
	if err := p.db.Create(course).Error; err != nil {
		return nil, fmt.Errorf("保存课程失败: %v", err)
	}

	log.Printf("✅ [Course创建] 课程创建成功，ID: %d", course.ID)

	// 3. 创建Slide实体
	for _, slideContent := range result.Slides {
		slide := &models.Slide{
			CourseID:     course.ID,
			SlideNumber:  slideContent.SlideNumber,
			Title:        slideContent.Title,
			Content:      slideContent.Content,
			Notes:        "", // Coze生成的HTML中没有备注信息
			SpeakerNotes: p.generateSpeakerNotes(slideContent),
			Duration:     p.estimateSlideDuration(slideContent.Content),
			LayoutType:   p.determineLayoutType(slideContent),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := p.db.Create(slide).Error; err != nil {
			log.Printf("❌ [Slide创建] 创建幻灯片失败: %v", err)
			return nil, fmt.Errorf("创建幻灯片失败: %v", err)
		}

		log.Printf("✅ [Slide创建] 第%d张幻灯片创建成功: %s", slide.SlideNumber, slide.Title)
	}

	// 4. 更新Course状态为完成
	course.Status = models.CourseStatusCompleted
	if err := p.db.Save(course).Error; err != nil {
		log.Printf("❌ [Course更新] 更新课程状态失败: %v", err)
		return nil, fmt.Errorf("更新课程状态失败: %v", err)
	}

	log.Printf("✅ [Course完成] 课程生成完成，ID: %d，幻灯片数量: %d", course.ID, len(result.Slides))

	// 5. 重新查询完整的Course信息（包含Slides）
	var completeCourse models.Course
	if err := p.db.Preload("Slides").First(&completeCourse, course.ID).Error; err != nil {
		log.Printf("❌ [Course查询] 查询完整课程信息失败: %v", err)
		return nil, fmt.Errorf("查询完整课程信息失败: %v", err)
	}

	return &completeCourse, nil
}

// generateSpeakerNotes 生成演讲备注
func (p *CozeHTMLParser) generateSpeakerNotes(slide CozeSlideContent) string {
	// 基于幻灯片内容生成简单的演讲备注
	lines := strings.Split(slide.Content, "\n")
	var notes []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) > 10 {
			// 为每个要点添加演讲提示
			if strings.HasPrefix(line, "•") || strings.HasPrefix(line, "-") {
				notes = append(notes, fmt.Sprintf("重点强调：%s", strings.TrimPrefix(strings.TrimPrefix(line, "•"), "-")))
			} else {
				notes = append(notes, fmt.Sprintf("详细说明：%s", line))
			}
		}
	}

	if len(notes) == 0 {
		return fmt.Sprintf("请详细讲解\"%s\"的相关内容", slide.Title)
	}

	return strings.Join(notes, "\n")
}

// estimateSlideDuration 估算幻灯片播放时长
func (p *CozeHTMLParser) estimateSlideDuration(content string) int {
	// 简单估算：基于内容长度和字数
	charCount := len(content)

	// 基本时长：30秒
	baseDuration := 30

	// 根据内容长度调整
	if charCount > 500 {
		baseDuration = 60 // 1分钟
	} else if charCount > 200 {
		baseDuration = 45 // 45秒
	}

	return baseDuration
}

// determineLayoutType 确定布局类型
func (p *CozeHTMLParser) determineLayoutType(slide CozeSlideContent) models.LayoutType {
	// 根据内容特征判断布局类型
	content := slide.Content

	// 标题页检测
	if slide.SlideNumber == 1 || strings.Contains(strings.ToLower(slide.Title), "标题") {
		return models.LayoutTypeTitleSlide
	}

	// 列表内容检测
	if strings.Contains(content, "•") || strings.Contains(content, "-") ||
		strings.Count(content, "\n") > 3 {
		return models.LayoutTypeBulletList
	}

	// 默认内容布局
	return models.LayoutTypeContent
}

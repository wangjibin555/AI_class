package services

import (
	"ai-classroom/internal/models"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SlideCreationService 幻灯片创建服务
type SlideCreationService struct {
	db             *gorm.DB
	keywordService *KeywordExtractionService
}

// SlideCreationContent 幻灯片创建内容
type SlideCreationContent struct {
	SlideNumber   int      `json:"slide_number"`   // 幻灯片编号
	Title         string   `json:"title"`          // 标题
	Content       string   `json:"content"`        // 内容
	Keywords      []string `json:"keywords"`       // 来源关键词
	Notes         string   `json:"notes"`          // 备注
	SpeakerNotes  string   `json:"speaker_notes"`  // 演讲者备注
	EstimatedTime int      `json:"estimated_time"` // 预计时长（秒）
	SlideType     string   `json:"slide_type"`     // 幻灯片类型
	BulletPoints  []string `json:"bullet_points"`  // 要点列表
}

// SlideCreationResult 幻灯片创建结果
type SlideCreationResult struct {
	TotalSlides   int             `json:"total_slides"`   // 总幻灯片数
	SuccessCount  int             `json:"success_count"`  // 成功创建数
	FailedCount   int             `json:"failed_count"`   // 失败数
	CreatedSlides []*models.Slide `json:"created_slides"` // 创建的幻灯片
	Errors        []string        `json:"errors"`         // 错误列表
	ProcessTime   time.Duration   `json:"process_time"`   // 处理时间
}

// NewSlideCreationService 创建幻灯片创建服务
func NewSlideCreationService(db *gorm.DB, keywordService *KeywordExtractionService) *SlideCreationService {
	return &SlideCreationService{
		db:             db,
		keywordService: keywordService,
	}
}

// CreateSlideWithKeywords 创建包含关键词的幻灯片
func (s *SlideCreationService) CreateSlideWithKeywords(courseID uint, slideContent *SlideCreationContent) (*models.Slide, error) {
	// ✅ 增加调试输出 - 开始处理
	fmt.Printf("\n🎯 开始创建幻灯片 (课程ID: %d, 幻灯片号: %d)\n", courseID, slideContent.SlideNumber)
	fmt.Printf("📝 幻灯片标题: %s\n", slideContent.Title)

	// 1. 提取幻灯片内容的关键词
	fullContent := slideContent.Title + " " + slideContent.Content + " " + slideContent.Notes
	fmt.Printf("🔍 准备提取关键词的完整内容长度: %d 字符\n", len(fullContent))

	var slideKeywords []string

	if s.keywordService != nil {
		fmt.Printf("🔧 使用关键词提取服务...\n")
		keywordResult, err := s.keywordService.ExtractKeywords(fullContent, &ExtractionOptions{
			MaxKeywords: 10,
			ContentType: "academic",
			UseAI:       false, // 幻灯片级别使用基础提取以提高速度
		})
		if err != nil {
			fmt.Printf("❌ 幻灯片关键词提取失败: %v\n", err)
			log.Printf("幻灯片关键词提取失败: %v", err)
			slideKeywords = []string{}
		} else {
			slideKeywords = keywordResult.Primary
			fmt.Printf("✅ 幻灯片关键词提取成功: %v\n", slideKeywords)
		}
	} else {
		fmt.Printf("⚠️ 关键词提取服务未初始化\n")
	}

	// 2. 合并来源关键词和提取的关键词
	fmt.Printf("\n🔄 步骤2: 合并关键词\n")
	fmt.Printf("📥 来源关键词: %v\n", slideContent.Keywords)
	fmt.Printf("🔧 提取关键词: %v\n", slideKeywords)

	var finalKeywords models.Keywords
	if len(slideContent.Keywords) > 0 {
		fmt.Printf("➕ 添加来源关键词...\n")
		finalKeywords.AddAll(slideContent.Keywords)
	}
	finalKeywords.AddAll(slideKeywords)

	// 3. 去重和数量限制
	fmt.Printf("\n✂️ 步骤3: 去重和限制数量\n")
	fmt.Printf("🔄 合并前关键词: %v\n", finalKeywords)
	finalKeywords.Limit(10) // 限制最多10个关键词
	fmt.Printf("✅ 最终关键词: %v\n", finalKeywords)

	// 4. 处理内容
	content := slideContent.Content
	if content == "" && len(slideContent.BulletPoints) > 0 {
		content = strings.Join(slideContent.BulletPoints, "\n• ")
		if content != "" {
			content = "• " + content
		}
	}

	// 5. 估算时长
	estimatedDuration := slideContent.EstimatedTime
	if estimatedDuration == 0 {
		estimatedDuration = s.estimateSlideDuration(slideContent.Title + " " + content + " " + slideContent.SpeakerNotes)
	}

	// 6. 创建幻灯片模型
	fmt.Printf("\n💾 步骤4: 准备保存到数据库\n")
	slide := &models.Slide{
		CourseID:     courseID,
		SlideNumber:  slideContent.SlideNumber,
		Title:        slideContent.Title,
		Content:      content,
		Keywords:     finalKeywords, // ✅ 正确赋值关键词
		Notes:        slideContent.Notes,
		SpeakerNotes: slideContent.SpeakerNotes,
		Duration:     estimatedDuration,
		LayoutType:   models.LayoutType(slideContent.SlideType),
	}

	// ✅ 显示准备保存的数据
	fmt.Printf("🗂️ 准备保存的幻灯片数据:\n")
	fmt.Printf("   - 课程ID: %d\n", slide.CourseID)
	fmt.Printf("   - 幻灯片号: %d\n", slide.SlideNumber)
	fmt.Printf("   - 标题: %s\n", slide.Title)
	fmt.Printf("   - 关键词: %v\n", slide.Keywords)
	fmt.Printf("   - 关键词字符串: %s\n", slide.Keywords.String())
	fmt.Printf("   - 关键词是否为空: %t\n", slide.Keywords.IsEmpty())

	// 7. 保存到数据库
	fmt.Printf("\n💾 正在保存到数据库...\n")
	if err := s.db.Create(slide).Error; err != nil {
		fmt.Printf("❌ 保存失败: %v\n", err)
		return nil, fmt.Errorf("创建幻灯片失败: %w", err)
	}
	fmt.Printf("✅ 保存成功，幻灯片ID: %d\n", slide.ID)

	// 8. 验证关键词存储
	fmt.Printf("\n🔍 步骤5: 验证关键词存储\n")

	// 重新从数据库读取验证
	var savedSlide models.Slide
	if err := s.db.First(&savedSlide, slide.ID).Error; err == nil {
		fmt.Printf("📊 数据库中的关键词: %v\n", savedSlide.Keywords)
		fmt.Printf("📊 数据库中关键词字符串: %s\n", savedSlide.Keywords.String())
		fmt.Printf("📊 数据库中关键词是否为空: %t\n", savedSlide.Keywords.IsEmpty())
	} else {
		fmt.Printf("⚠️ 验证读取失败: %v\n", err)
	}

	if slide.Keywords.IsEmpty() {
		fmt.Printf("⚠️ 检测到关键词为空，尝试重新提取\n")
		log.Printf("⚠️ 幻灯片 %d 关键词为空，尝试重新提取", slide.SlideNumber)
		// 尝试重新提取关键词
		if s.keywordService != nil {
			if keywords, err := s.keywordService.ExtractKeywords(slideContent.Title+" "+slideContent.Content, &ExtractionOptions{MaxKeywords: 5}); err == nil {
				slide.Keywords = models.Keywords(keywords.Primary)
				s.db.Model(slide).Update("keywords", slide.Keywords)
				fmt.Printf("🔄 重新提取成功: %v\n", slide.Keywords)
				log.Printf("✅ 幻灯片 %d 关键词重新提取成功: %v", slide.SlideNumber, slide.Keywords)
			} else {
				fmt.Printf("❌ 重新提取失败: %v\n", err)
			}
		}
	} else {
		fmt.Printf("✅ 关键词存储验证成功\n")
		log.Printf("✅ 幻灯片 %d 关键词: %s", slide.SlideNumber, slide.Keywords.String())
	}

	fmt.Printf("═══════════════════════════════════════\n\n")
	return slide, nil
}

// BatchCreateSlides 批量创建幻灯片
func (s *SlideCreationService) BatchCreateSlides(courseID uint, slideContents []SlideCreationContent) (*SlideCreationResult, error) {
	startTime := time.Now()

	result := &SlideCreationResult{
		TotalSlides:   len(slideContents),
		SuccessCount:  0,
		FailedCount:   0,
		CreatedSlides: []*models.Slide{},
		Errors:        []string{},
	}

	// 开始事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return result, fmt.Errorf("开始事务失败: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建临时服务使用事务
	tempService := &SlideCreationService{
		db:             tx,
		keywordService: s.keywordService,
	}

	// 逐个创建幻灯片
	for _, slideContent := range slideContents {
		slide, err := tempService.CreateSlideWithKeywords(courseID, &slideContent)
		if err != nil {
			result.FailedCount++
			errorMsg := fmt.Sprintf("创建幻灯片 %d 失败: %v", slideContent.SlideNumber, err)
			result.Errors = append(result.Errors, errorMsg)
			log.Printf("❌ %s", errorMsg)
			continue
		}

		result.SuccessCount++
		result.CreatedSlides = append(result.CreatedSlides, slide)
		log.Printf("✅ 幻灯片 %d 创建成功，关键词: %v", slide.SlideNumber, slide.Keywords)
	}

	// 如果全部失败，回滚事务
	if result.SuccessCount == 0 {
		tx.Rollback()
		return result, fmt.Errorf("所有幻灯片创建失败")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return result, fmt.Errorf("提交事务失败: %w", err)
	}

	result.ProcessTime = time.Since(startTime)

	// 记录总体结果
	log.Printf("📊 批量创建幻灯片完成: 总数=%d, 成功=%d, 失败=%d, 耗时=%v",
		result.TotalSlides, result.SuccessCount, result.FailedCount, result.ProcessTime)

	return result, nil
}

// UpdateSlideKeywords 更新幻灯片关键词
func (s *SlideCreationService) UpdateSlideKeywords(slideID uint, keywords []string) error {
	if s.keywordService == nil {
		return fmt.Errorf("关键词服务未初始化")
	}

	// 获取幻灯片
	var slide models.Slide
	if err := s.db.First(&slide, slideID).Error; err != nil {
		return fmt.Errorf("幻灯片不存在: %w", err)
	}

	// 验证和清理关键词
	cleanedKeywords := s.deduplicateAndSort(keywords)

	// 更新关键词
	slide.Keywords = models.Keywords(cleanedKeywords)
	if err := s.db.Model(&slide).Update("keywords", slide.Keywords).Error; err != nil {
		return fmt.Errorf("更新关键词失败: %w", err)
	}

	log.Printf("✅ 幻灯片 %d 关键词更新成功: %v", slideID, slide.Keywords)
	return nil
}

// RegenerateSlideKeywords 重新生成幻灯片关键词
func (s *SlideCreationService) RegenerateSlideKeywords(slideID uint) error {
	if s.keywordService == nil {
		return fmt.Errorf("关键词服务未初始化")
	}

	// 获取幻灯片
	var slide models.Slide
	if err := s.db.First(&slide, slideID).Error; err != nil {
		return fmt.Errorf("幻灯片不存在: %w", err)
	}

	// 重新提取关键词
	fullContent := slide.Title + " " + slide.Content + " " + slide.SpeakerNotes
	keywordResult, err := s.keywordService.ExtractKeywords(fullContent, &ExtractionOptions{
		MaxKeywords: 10,
		ContentType: "academic",
		UseAI:       true,
	})
	if err != nil {
		return fmt.Errorf("关键词提取失败: %w", err)
	}

	// 更新关键词
	slide.Keywords = models.Keywords(keywordResult.All)
	if err := s.db.Model(&slide).Update("keywords", slide.Keywords).Error; err != nil {
		return fmt.Errorf("更新关键词失败: %w", err)
	}

	log.Printf("✅ 幻灯片 %d 关键词重新生成成功: %v", slideID, slide.Keywords)
	return nil
}

// BatchRegenerateKeywords 批量重新生成关键词
func (s *SlideCreationService) BatchRegenerateKeywords(courseID uint) (*SlideCreationResult, error) {
	startTime := time.Now()

	// 获取课程所有幻灯片
	var slides []models.Slide
	if err := s.db.Where("course_id = ?", courseID).Find(&slides).Error; err != nil {
		return nil, fmt.Errorf("获取幻灯片失败: %w", err)
	}

	result := &SlideCreationResult{
		TotalSlides:  len(slides),
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       []string{},
	}

	// 逐个重新生成关键词
	for _, slide := range slides {
		if err := s.RegenerateSlideKeywords(slide.ID); err != nil {
			result.FailedCount++
			errorMsg := fmt.Sprintf("重新生成幻灯片 %d 关键词失败: %v", slide.ID, err)
			result.Errors = append(result.Errors, errorMsg)
			continue
		}
		result.SuccessCount++
	}

	result.ProcessTime = time.Since(startTime)

	log.Printf("📊 批量重新生成关键词完成: 总数=%d, 成功=%d, 失败=%d, 耗时=%v",
		result.TotalSlides, result.SuccessCount, result.FailedCount, result.ProcessTime)

	return result, nil
}

// GetSlidesByKeywords 根据关键词搜索幻灯片
func (s *SlideCreationService) GetSlidesByKeywords(keywords []string, limit int) ([]models.Slide, error) {
	if len(keywords) == 0 {
		return []models.Slide{}, nil
	}

	// 构建搜索查询
	query := s.db.Model(&models.Slide{})

	// 为每个关键词添加搜索条件
	for i, keyword := range keywords {
		if i == 0 {
			query = query.Where("JSON_CONTAINS(keywords, ?)", fmt.Sprintf(`"%s"`, keyword))
		} else {
			query = query.Or("JSON_CONTAINS(keywords, ?)", fmt.Sprintf(`"%s"`, keyword))
		}
	}

	var slides []models.Slide
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&slides).Error; err != nil {
		return nil, fmt.Errorf("搜索幻灯片失败: %w", err)
	}

	return slides, nil
}

// 辅助方法

// deduplicateAndSort 去重和排序关键词
func (s *SlideCreationService) deduplicateAndSort(keywords []string) []string {
	// 使用map去重
	keywordSet := make(map[string]bool)
	var result []string

	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" && !keywordSet[keyword] {
			keywordSet[keyword] = true
			result = append(result, keyword)
		}
	}

	return result
}

// estimateSlideDuration 估算幻灯片时长
func (s *SlideCreationService) estimateSlideDuration(text string) int {
	// 简单估算：中文约每分钟300字，英文约每分钟150词
	textLength := len([]rune(text))
	if textLength == 0 {
		return 30 // 默认30秒
	}

	// 假设平均阅读速度为每分钟200字
	estimatedMinutes := float64(textLength) / 200.0
	estimatedSeconds := int(estimatedMinutes * 60)

	// 最少15秒，最多300秒（5分钟）
	if estimatedSeconds < 15 {
		estimatedSeconds = 15
	} else if estimatedSeconds > 300 {
		estimatedSeconds = 300
	}

	return estimatedSeconds
}

// ValidateSlideContent 验证幻灯片内容
func (s *SlideCreationService) ValidateSlideContent(content *SlideCreationContent) []string {
	var errors []string

	if content.SlideNumber <= 0 {
		errors = append(errors, "幻灯片编号必须大于0")
	}

	if strings.TrimSpace(content.Title) == "" {
		errors = append(errors, "幻灯片标题不能为空")
	}

	if strings.TrimSpace(content.Content) == "" && len(content.BulletPoints) == 0 {
		errors = append(errors, "幻灯片内容不能为空")
	}

	return errors
}

// GetSlideStatistics 获取幻灯片统计信息
func (s *SlideCreationService) GetSlideStatistics(courseID uint) (*SlideStatistics, error) {
	var stats SlideStatistics

	// 基本统计
	if err := s.db.Model(&models.Slide{}).Where("course_id = ?", courseID).Count(&stats.TotalSlides).Error; err != nil {
		return nil, fmt.Errorf("获取幻灯片数量失败: %w", err)
	}

	// 有关键词的幻灯片数量
	if err := s.db.Model(&models.Slide{}).
		Where("course_id = ? AND JSON_LENGTH(keywords) > 0", courseID).
		Count(&stats.SlidesWithKeywords).Error; err != nil {
		return nil, fmt.Errorf("获取有关键词幻灯片数量失败: %w", err)
	}

	// 计算比例
	if stats.TotalSlides > 0 {
		stats.KeywordCoverage = float64(stats.SlidesWithKeywords) / float64(stats.TotalSlides) * 100
	}

	// 平均关键词数量
	var slides []models.Slide
	if err := s.db.Where("course_id = ?", courseID).Find(&slides).Error; err != nil {
		return nil, fmt.Errorf("获取幻灯片失败: %w", err)
	}

	totalKeywords := 0
	for _, slide := range slides {
		totalKeywords += len(slide.Keywords)
	}

	if len(slides) > 0 {
		stats.AverageKeywordsPerSlide = float64(totalKeywords) / float64(len(slides))
	}

	return &stats, nil
}

// SlideStatistics 幻灯片统计信息
type SlideStatistics struct {
	TotalSlides             int64   `json:"total_slides"`               // 总幻灯片数
	SlidesWithKeywords      int64   `json:"slides_with_keywords"`       // 有关键词的幻灯片数
	KeywordCoverage         float64 `json:"keyword_coverage"`           // 关键词覆盖率（百分比）
	AverageKeywordsPerSlide float64 `json:"average_keywords_per_slide"` // 平均每张幻灯片关键词数
}

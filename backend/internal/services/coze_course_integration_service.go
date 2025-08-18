package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ai-classroom/internal/models"

	"gorm.io/gorm"
)

// CozeCourseIntegrationService Coze课程集成服务
type CozeCourseIntegrationService struct {
	db                       *gorm.DB
	htmlParser               *CozeHTMLParser
	aiContentAnalysisService *AIContentAnalysisService
}

// NewCozeCourseIntegrationService 创建新的Coze课程集成服务
func NewCozeCourseIntegrationService(
	db *gorm.DB,
	aiContentAnalysisService *AIContentAnalysisService,
) *CozeCourseIntegrationService {
	return &CozeCourseIntegrationService{
		db:                       db,
		htmlParser:               NewCozeHTMLParser(db),
		aiContentAnalysisService: aiContentAnalysisService,
	}
}

// CozeIntegrationRequest Coze集成请求
type CozeIntegrationRequest struct {
	HTMLFilePath string `json:"html_file_path" binding:"required"` // HTML文件路径
	SourceURL    string `json:"source_url"`                        // 源URL
	UserID       uint   `json:"user_id" binding:"required"`        // 用户ID
	Title        string `json:"title"`                             // 自定义标题
	Description  string `json:"description"`                       // 自定义描述
}

// CozeIntegrationResponse Coze集成响应
type CozeIntegrationResponse struct {
	Success  bool           `json:"success"`
	Course   *models.Course `json:"course,omitempty"`
	Message  string         `json:"message"`
	CourseID uint           `json:"course_id,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// IntegrateCozeHTMLToCourse 将Coze生成的HTML集成到课程系统
func (s *CozeCourseIntegrationService) IntegrateCozeHTMLToCourse(req *CozeIntegrationRequest) (*CozeIntegrationResponse, error) {
	log.Printf("🚀 [Coze集成] 开始集成HTML文件到课程系统")
	log.Printf("🔍 [Coze集成] 文件路径: %s, 用户ID: %d", req.HTMLFilePath, req.UserID)

	// 1. 验证参数
	if err := s.validateRequest(req); err != nil {
		log.Printf("❌ [Coze集成] 参数验证失败: %v", err)
		return &CozeIntegrationResponse{
			Success: false,
			Error:   fmt.Sprintf("参数验证失败: %v", err),
		}, nil
	}

	// 2. 读取HTML文件内容
	htmlContent, err := s.readHTMLFile(req.HTMLFilePath)
	if err != nil {
		log.Printf("❌ [Coze集成] 读取HTML文件失败: %v", err)
		return &CozeIntegrationResponse{
			Success: false,
			Error:   fmt.Sprintf("读取HTML文件失败: %v", err),
		}, nil
	}

	log.Printf("✅ [Coze集成] HTML文件读取成功，内容长度: %d", len(htmlContent))

	// 3. 解析HTML并创建课程
	course, err := s.htmlParser.ParseHTMLToCourse(htmlContent, req.UserID, req.SourceURL)
	if err != nil {
		log.Printf("❌ [Coze集成] HTML解析失败: %v", err)
		return &CozeIntegrationResponse{
			Success: false,
			Error:   fmt.Sprintf("HTML解析失败: %v", err),
		}, nil
	}

	// 4. 应用自定义标题和描述（如果提供）
	if err := s.applyCustomizations(course, req); err != nil {
		log.Printf("❌ [Coze集成] 应用自定义设置失败: %v", err)
		return &CozeIntegrationResponse{
			Success: false,
			Error:   fmt.Sprintf("应用自定义设置失败: %v", err),
		}, nil
	}

	// 5. 更新课程的生成参数，记录来源
	s.updateGenerationParams(course, req)

	log.Printf("✅ [Coze集成] 课程集成成功，课程ID: %d", course.ID)

	return &CozeIntegrationResponse{
		Success:  true,
		Course:   course,
		CourseID: course.ID,
		Message:  "Coze HTML成功集成到课程系统",
	}, nil
}

// validateRequest 验证请求参数
func (s *CozeCourseIntegrationService) validateRequest(req *CozeIntegrationRequest) error {
	if req.HTMLFilePath == "" {
		return fmt.Errorf("HTML文件路径不能为空")
	}

	if req.UserID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}

	// 验证文件扩展名
	ext := filepath.Ext(req.HTMLFilePath)
	if ext != ".html" && ext != ".htm" {
		return fmt.Errorf("文件必须是HTML格式")
	}

	return nil
}

// readHTMLFile 读取HTML文件内容
func (s *CozeCourseIntegrationService) readHTMLFile(filePath string) (string, error) {
	// 检查是否为上传文件标记
	if strings.HasPrefix(filePath, "upload://") {
		return "", fmt.Errorf("上传文件应使用IntegrateHTMLContentToCourse方法")
	}

	// 这里可以根据实际情况实现文件读取逻辑
	// 可能是从本地文件系统、对象存储或其他位置读取
	log.Printf("🔍 [文件读取] 准备读取文件: %s", filePath)

	// 实际的文件读取逻辑
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	return string(content), nil
}

// IntegrateHTMLContentToCourse 直接从HTML内容集成到课程系统
func (s *CozeCourseIntegrationService) IntegrateHTMLContentToCourse(htmlContent string, req *CozeIntegrationRequest) (*CozeIntegrationResponse, error) {
	log.Printf("🚀 [Coze集成] 开始从HTML内容集成到课程系统")
	log.Printf("🔍 [Coze集成] 内容长度: %d, 用户ID: %d", len(htmlContent), req.UserID)

	// 1. 验证参数
	if req.UserID == 0 {
		return &CozeIntegrationResponse{
			Success: false,
			Error:   "用户ID不能为空",
		}, nil
	}

	if htmlContent == "" {
		return &CozeIntegrationResponse{
			Success: false,
			Error:   "HTML内容不能为空",
		}, nil
	}

	// 2. 解析HTML并创建课程
	course, err := s.htmlParser.ParseHTMLToCourse(htmlContent, req.UserID, req.SourceURL)
	if err != nil {
		log.Printf("❌ [Coze集成] HTML解析失败: %v", err)
		return &CozeIntegrationResponse{
			Success: false,
			Error:   fmt.Sprintf("HTML解析失败: %v", err),
		}, nil
	}

	// 3. 应用自定义标题和描述（如果提供）
	if err := s.applyCustomizations(course, req); err != nil {
		log.Printf("❌ [Coze集成] 应用自定义设置失败: %v", err)
		return &CozeIntegrationResponse{
			Success: false,
			Error:   fmt.Sprintf("应用自定义设置失败: %v", err),
		}, nil
	}

	// 4. 更新课程的生成参数，记录来源
	s.updateGenerationParams(course, req)

	log.Printf("✅ [Coze集成] 课程集成成功，课程ID: %d", course.ID)

	return &CozeIntegrationResponse{
		Success:  true,
		Course:   course,
		CourseID: course.ID,
		Message:  "Coze HTML内容成功集成到课程系统",
	}, nil
}

// applyCustomizations 应用自定义设置
func (s *CozeCourseIntegrationService) applyCustomizations(course *models.Course, req *CozeIntegrationRequest) error {
	updated := false

	// 应用自定义标题 - 优化逻辑：只有提供了有意义的自定义标题时才覆盖
	if req.Title != "" && req.Title != course.Title && s.isValidCustomTitle(req.Title) {
		log.Printf("🔧 [自定义] 检测到有效的自定义标题，从 '%s' 更新为 '%s'", course.Title, req.Title)
		course.Title = req.Title
		updated = true
	} else if req.Title != "" && !s.isValidCustomTitle(req.Title) {
		log.Printf("🚫 [自定义] 忽略通用标题 '%s'，保留从PPT解析的标题 '%s'", req.Title, course.Title)
	}

	// 应用自定义描述
	if req.Description != "" && req.Description != course.Description {
		course.Description = req.Description
		updated = true
		log.Printf("🔧 [自定义] 应用自定义描述: %s", req.Description)
	}

	// 如果有更新，保存到数据库
	if updated {
		course.UpdatedAt = time.Now()
		if err := s.db.Save(course).Error; err != nil {
			return fmt.Errorf("保存自定义设置失败: %v", err)
		}
		log.Printf("✅ [自定义] 自定义设置已保存")
	}

	return nil
}

// updateGenerationParams 更新生成参数
func (s *CozeCourseIntegrationService) updateGenerationParams(course *models.Course, req *CozeIntegrationRequest) {
	if course.GenerationParams == nil {
		course.GenerationParams = make(models.GenerationParams)
	}

	// 添加集成相关信息
	course.GenerationParams["integration_source"] = "coze_html_parser"
	course.GenerationParams["integration_at"] = time.Now().Format(time.RFC3339)
	course.GenerationParams["html_file_path"] = req.HTMLFilePath
	if req.SourceURL != "" {
		course.GenerationParams["original_source_url"] = req.SourceURL
	}

	// 保存更新
	s.db.Save(course)
	log.Printf("📝 [参数更新] 生成参数已更新")
}

// GetCourseWithSlides 获取课程及其幻灯片信息
func (s *CozeCourseIntegrationService) GetCourseWithSlides(courseID uint) (*models.Course, error) {
	var course models.Course

	if err := s.db.Preload("Slides").First(&course, courseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("课程不存在")
		}
		return nil, fmt.Errorf("查询课程失败: %v", err)
	}

	return &course, nil
}

// ListCozeIntegratedCourses 列出Coze集成的课程
func (s *CozeCourseIntegrationService) ListCozeIntegratedCourses(userID uint, page, pageSize int) ([]models.Course, int64, error) {
	var courses []models.Course
	var total int64

	// 构建查询条件：只查询通过Coze集成的课程
	query := s.db.Model(&models.Course{}).Where("user_id = ?", userID)

	// 通过GenerationParams中的标识来筛选Coze集成的课程
	query = query.Where("JSON_EXTRACT(generation_params, '$.engine') = ? OR JSON_EXTRACT(generation_params, '$.integration_source') = ?",
		"coze", "coze_html_parser")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询课程总数失败: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Slides").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&courses).Error; err != nil {
		return nil, 0, fmt.Errorf("查询课程列表失败: %v", err)
	}

	return courses, total, nil
}

// isValidCustomTitle 判断是否是有效的自定义标题
func (s *CozeCourseIntegrationService) isValidCustomTitle(title string) bool {
	// 通用标题列表 - 这些标题应该被忽略，保留从PPT解析的真实标题
	genericTitles := []string{
		"通过Coze智能体生成的课程",
		"AI生成的课程",
		"AI生成的PPT",
		"基于URL内容生成的PPT",
		"PPT演示文稿",
		"课程内容",
		"学习材料",
		"教学内容",
		"培训资料",
		"演示文稿",
		"课件内容",
		"新建课程",
		"untitled",
		"无标题",
		"",
	}

	// 标准化标题进行比较
	normalizedTitle := strings.TrimSpace(strings.ToLower(title))

	// 检查是否是通用标题
	for _, generic := range genericTitles {
		if strings.ToLower(generic) == normalizedTitle {
			return false
		}
	}

	// 检查是否包含通用词汇模式
	genericPatterns := []string{
		"基于.*生成的",
		".*生成的课程",
		".*生成的ppt",
		"通过.*生成",
		"ai.*课程",
		"智能.*生成",
	}

	for _, pattern := range genericPatterns {
		if matched, _ := regexp.MatchString(pattern, normalizedTitle); matched {
			return false
		}
	}

	// 检查标题长度和内容质量
	if len([]rune(title)) < 3 {
		return false // 太短
	}

	if len([]rune(title)) > 100 {
		return false // 太长
	}

	// 如果包含具体的技术术语或专业词汇，认为是有效标题
	technicalKeywords := []string{
		"redis", "mysql", "mongodb", "kubernetes", "docker", "react", "vue", "angular",
		"python", "java", "golang", "javascript", "typescript", "算法", "数据结构",
		"机器学习", "深度学习", "微服务", "架构", "设计模式", "数据库", "缓存",
		"消息队列", "分布式", "高并发", "性能优化", "网络编程", "操作系统",
	}

	titleLower := strings.ToLower(title)
	for _, keyword := range technicalKeywords {
		if strings.Contains(titleLower, keyword) {
			return true
		}
	}

	// 默认认为是有效的自定义标题（如果不匹配上述通用模式）
	return true
}

// DeleteCourse 删除课程（包括相关的幻灯片）
func (s *CozeCourseIntegrationService) DeleteCourse(courseID, userID uint) error {
	// 验证课程归属
	var course models.Course
	if err := s.db.Where("id = ? AND user_id = ?", courseID, userID).First(&course).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("课程不存在或无权限删除")
		}
		return fmt.Errorf("查询课程失败: %v", err)
	}

	// 使用事务删除课程和相关数据
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除幻灯片
		if err := tx.Where("course_id = ?", courseID).Delete(&models.Slide{}).Error; err != nil {
			return fmt.Errorf("删除幻灯片失败: %v", err)
		}

		// 删除课程
		if err := tx.Delete(&course).Error; err != nil {
			return fmt.Errorf("删除课程失败: %v", err)
		}

		log.Printf("✅ [课程删除] 课程及相关数据删除成功，课程ID: %d", courseID)
		return nil
	})
}

// UpdateCourseInfo 更新课程基本信息
func (s *CozeCourseIntegrationService) UpdateCourseInfo(courseID, userID uint, title, description string) error {
	// 验证课程归属
	var course models.Course
	if err := s.db.Where("id = ? AND user_id = ?", courseID, userID).First(&course).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("课程不存在或无权限修改")
		}
		return fmt.Errorf("查询课程失败: %v", err)
	}

	// 更新信息
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if title != "" {
		updates["title"] = title
	}
	if description != "" {
		updates["description"] = description
	}

	if err := s.db.Model(&course).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新课程信息失败: %v", err)
	}

	log.Printf("✅ [课程更新] 课程信息更新成功，课程ID: %d", courseID)
	return nil
}

// GetCourseStats 获取课程统计信息
func (s *CozeCourseIntegrationService) GetCourseStats(userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总课程数
	var totalCourses int64
	if err := s.db.Model(&models.Course{}).Where("user_id = ?", userID).Count(&totalCourses).Error; err != nil {
		return nil, fmt.Errorf("查询总课程数失败: %v", err)
	}

	// Coze集成课程数
	var cozeCourses int64
	if err := s.db.Model(&models.Course{}).Where("user_id = ?", userID).
		Where("JSON_EXTRACT(generation_params, '$.engine') = ? OR JSON_EXTRACT(generation_params, '$.integration_source') = ?",
			"coze", "coze_html_parser").
		Count(&cozeCourses).Error; err != nil {
		return nil, fmt.Errorf("查询Coze课程数失败: %v", err)
	}

	// 总幻灯片数
	var totalSlides int64
	if err := s.db.Model(&models.Slide{}).
		Joins("JOIN courses ON slides.course_id = courses.id").
		Where("courses.user_id = ?", userID).
		Count(&totalSlides).Error; err != nil {
		return nil, fmt.Errorf("查询总幻灯片数失败: %v", err)
	}

	stats["total_courses"] = totalCourses
	stats["coze_courses"] = cozeCourses
	stats["total_slides"] = totalSlides
	stats["coze_ratio"] = float64(cozeCourses) / float64(totalCourses) * 100

	return stats, nil
}

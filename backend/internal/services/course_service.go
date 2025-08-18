package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/internal/repositories"
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/parser"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
)

// CourseService 课件服务接口
type CourseService interface {
	CreateCourse(userID uint, req CreateCourseRequest) (*models.Course, error)
	CreateCourseFromFile(userID uint, file *multipart.FileHeader, req CreateCourseRequest) (*models.Course, error)
	GetCourseByID(id uint, userID uint) (*models.Course, error)
	GetUserCourses(req GetCoursesRequest) (*GetCoursesResponse, error)
	GetPublicCourses(req GetCoursesRequest) (*GetCoursesResponse, error)
	SearchCourses(req SearchCoursesRequest) (*GetCoursesResponse, error)
	UpdateCourse(id uint, userID uint, req UpdateCourseRequest) (*models.Course, error)
	DeleteCourse(id uint, userID uint) error
	IncrementViewCount(id uint) error
	IncrementLikeCount(id uint) error
	IncrementShareCount(id uint) error
	GetUserStats(userID uint) (*UserCourseStats, error)
	ValidateCreateRequest(req CreateCourseRequest) error
	ValidateFileUpload(file *multipart.FileHeader) error
}

// CreateCourseRequest 创建课件请求
type CreateCourseRequest struct {
	Title            string                  `json:"title" binding:"required,min=1,max=200"`
	Description      string                  `json:"description" binding:"max=1000"`
	Category         models.CourseCategory   `json:"category" binding:"required"`
	Tags             models.Tags             `json:"tags"`
	SourceType       models.SourceType       `json:"source_type" binding:"required"`
	SourceContent    string                  `json:"source_content" binding:"required"`
	SourceURL        string                  `json:"source_url"`
	IsPublic         bool                    `json:"is_public"`
	VoiceType        string                  `json:"voice_type"`
	GenerationParams models.GenerationParams `json:"generation_params"`
}

// UpdateCourseRequest 更新课件请求
type UpdateCourseRequest struct {
	Title       *string                `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string                `json:"description" binding:"omitempty,max=1000"`
	Category    *models.CourseCategory `json:"category"`
	Tags        *models.Tags           `json:"tags"`
	IsPublic    *bool                  `json:"is_public"`
	PPTFilePath *string                `json:"ppt_file_path" binding:"omitempty"`
}

// GetCoursesRequest 获取课件列表请求
type GetCoursesRequest struct {
	UserID    uint                  `json:"user_id"`
	Page      int                   `json:"page" binding:"min=1"`
	PageSize  int                   `json:"page_size" binding:"min=1,max=100"`
	Status    models.CourseStatus   `json:"status"`
	Category  models.CourseCategory `json:"category"`
	IsPublic  *bool                 `json:"is_public"`
	SortBy    string                `json:"sort_by"`
	SortOrder string                `json:"sort_order"`
}

// SearchCoursesRequest 搜索课件请求
type SearchCoursesRequest struct {
	Keyword  string `json:"keyword" binding:"required,min=1"`
	Page     int    `json:"page" binding:"min=1"`
	PageSize int    `json:"page_size" binding:"min=1,max=100"`
	UserID   uint   `json:"user_id"`
}

// GetCoursesResponse 获取课件列表响应
type GetCoursesResponse struct {
	Courses    []*models.Course `json:"courses"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// UserCourseStats 用户课件统计
type UserCourseStats struct {
	TotalCourses      int64 `json:"total_courses"`
	CompletedCourses  int64 `json:"completed_courses"`
	GeneratingCourses int64 `json:"generating_courses"`
	FailedCourses     int64 `json:"failed_courses"`
	TotalViews        int64 `json:"total_views"`
	TotalLikes        int64 `json:"total_likes"`
	TotalShares       int64 `json:"total_shares"`
}

// courseService 课件服务实现
type courseService struct {
	courseRepo     repositories.CourseRepository
	documentParser parser.DocumentParser
	pptGenerator   *PPTGenerationService
	ttsService     TTSService
}

// NewCourseService 创建课件服务实例
func NewCourseService(courseRepo repositories.CourseRepository, aiClient *ai.DashScopeClient, ttsService TTSService) CourseService {
	return &courseService{
		courseRepo:     courseRepo,
		documentParser: parser.NewDocumentParser(),
		pptGenerator:   NewPPTGenerationService(aiClient, NewKeywordExtractionService(aiClient)),
		ttsService:     ttsService,
	}
}

// CreateCourse 创建课件
func (s *courseService) CreateCourse(userID uint, req CreateCourseRequest) (*models.Course, error) {
	// 验证请求参数
	if err := s.ValidateCreateRequest(req); err != nil {
		return nil, err
	}

	// 创建课件对象
	course := &models.Course{
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		Category:         req.Category,
		Tags:             req.Tags,
		SourceType:       req.SourceType,
		SourceContent:    req.SourceContent,
		SourceURL:        req.SourceURL,
		IsPublic:         req.IsPublic,
		VoiceType:        req.VoiceType,
		GenerationParams: req.GenerationParams,
		Status:           models.CourseStatusGenerating,
	}

	// 设置默认值
	if course.VoiceType == "" {
		course.VoiceType = "zhimao"
	}

	// 保存到数据库
	if err := s.courseRepo.Create(course); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}

	// 异步生成PPT内容
	go s.generatePPTContent(course, req)

	return course, nil
}

// generatePPTContent 异步生成PPT内容
func (s *courseService) generatePPTContent(course *models.Course, req CreateCourseRequest) {
	// 设置生成参数 - 优化为技术文档
	params := GenerationParams{
		Template:      "technical",
		SlideCount:    18, // 增加默认幻灯片数量
		MaxSlideCount: 25, // 设置最大限制
		VoiceType:     course.VoiceType,
		Language:      "zh-CN",
		Style:         "technical",
		MaxTokens:     3000, // 增加token限制支持更多内容
		Temperature:   0.7,
	}

	// 如果有自定义生成参数，使用自定义参数
	if slideCount, ok := req.GenerationParams["slide_count"].(float64); ok && slideCount > 0 {
		params.SlideCount = int(slideCount)
	}

	// 生成PPT内容
	result, err := s.pptGenerator.GenerateSlides(req.SourceContent, params)
	if err != nil {
		// 更新课件状态为失败
		course.Status = models.CourseStatusFailed
		course.ErrorMessage = fmt.Sprintf("PPT生成失败: %v", err)
		s.courseRepo.Update(course)
		return
	}

	// 创建幻灯片记录
	var slides []models.Slide
	for i, slideContent := range result.Slides {
		// 将内容数组转换为字符串
		var content string
		if len(slideContent.Content) > 0 {
			content = strings.Join(slideContent.Content, "\n")
		}

		// 如果有要点列表，追加到内容
		if len(slideContent.BulletPoints) > 0 {
			bulletContent := strings.Join(slideContent.BulletPoints, "\n")
			if content != "" {
				content += "\n\n" + bulletContent
			} else {
				content = bulletContent
			}
		}

		slide := models.Slide{
			CourseID:     course.ID,
			SlideNumber:  i + 1,
			Title:        slideContent.Title,
			Content:      content,
			SpeakerNotes: slideContent.Notes,
			AudioURL:     "", // TODO: 生成音频后更新
			Duration:     0,  // TODO: 计算音频时长后更新
		}
		slides = append(slides, slide)
	}

	// 保存幻灯片到数据库
	for i := range slides {
		if err := s.courseRepo.CreateSlide(&slides[i]); err != nil {
			// 更新课件状态为失败
			course.Status = models.CourseStatusFailed
			course.ErrorMessage = fmt.Sprintf("保存幻灯片失败: %v", err)
			s.courseRepo.Update(course)
			return
		}
	}

	// 更新课件信息
	course.Status = models.CourseStatusCompleted
	course.SlidesCount = len(slides)
	s.courseRepo.Update(course)

	// 异步生成音频
	go s.generateAudioForSlides(course, slides)
}

// generateAudioForSlides 为幻灯片生成音频
func (s *courseService) generateAudioForSlides(course *models.Course, slides []models.Slide) {
	for i := range slides {
		// 为每张幻灯片生成音频
		if err := s.ttsService.GenerateSlideAudio(&slides[i], course.VoiceType); err != nil {
			// 记录错误但不中断整个流程
			fmt.Printf("为幻灯片 %d 生成音频失败: %v\n", slides[i].SlideNumber, err)
			continue
		}

		// 更新幻灯片信息
		if err := s.courseRepo.UpdateSlide(&slides[i]); err != nil {
			fmt.Printf("更新幻灯片 %d 失败: %v\n", slides[i].SlideNumber, err)
			continue
		}
	}
}

// CreateCourseFromFile 从文件创建课件
func (s *courseService) CreateCourseFromFile(userID uint, file *multipart.FileHeader, req CreateCourseRequest) (*models.Course, error) {
	// 验证文件
	if err := s.ValidateFileUpload(file); err != nil {
		return nil, err
	}

	// 使用文档解析器解析文件
	parseResult, err := s.documentParser.ParseFile(file)
	if err != nil {
		return nil, fmt.Errorf("文档解析失败: %w", err)
	}

	// 如果解析失败，返回错误
	if parseResult.Status == "failed" {
		return nil, fmt.Errorf("文档解析失败: %s", parseResult.ErrorMessage)
	}

	// 使用解析后的内容作为源内容
	sourceContent := parseResult.Content
	if sourceContent == "" {
		sourceContent = parseResult.ExtractedText
	}

	// 如果标题为空，使用解析出的标题
	if req.Title == "" {
		req.Title = parseResult.Title
	}

	// 验证请求参数
	modifiedReq := req
	modifiedReq.SourceContent = sourceContent
	if err := s.ValidateCreateRequest(modifiedReq); err != nil {
		return nil, err
	}

	// 创建课件对象
	course := &models.Course{
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		Category:         req.Category,
		Tags:             req.Tags,
		SourceType:       models.SourceTypeDocument,
		SourceContent:    sourceContent,
		FileSize:         file.Size,
		IsPublic:         req.IsPublic,
		VoiceType:        req.VoiceType,
		GenerationParams: req.GenerationParams,
		Status:           models.CourseStatusGenerating,
	}

	// 设置默认值
	if course.VoiceType == "" {
		course.VoiceType = "zhixiaobai"
	}

	// 保存到数据库
	if err := s.courseRepo.Create(course); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}

	return course, nil
}

// GetCourseByID 根据ID获取课件
func (s *courseService) GetCourseByID(id uint, userID uint) (*models.Course, error) {
	if id == 0 {
		return nil, errors.New("课件ID不能为空")
	}

	course, err := s.courseRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查权限：只有创建者或公开课件才能访问
	if course.UserID != userID && !course.IsPublic {
		return nil, errors.New("无权限访问该课件")
	}

	return course, nil
}

// GetUserCourses 获取用户课件列表
func (s *courseService) GetUserCourses(req GetCoursesRequest) (*GetCoursesResponse, error) {
	if req.UserID == 0 {
		return nil, errors.New("用户ID不能为空")
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	offset := (req.Page - 1) * req.PageSize

	// 构建过滤器
	filters := repositories.CourseFilters{
		Status:    req.Status,
		Category:  req.Category,
		IsPublic:  req.IsPublic,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// 获取课件列表
	courses, total, err := s.courseRepo.GetByUserID(req.UserID, offset, req.PageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("获取用户课件列表失败: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &GetCoursesResponse{
		Courses:    courses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetPublicCourses 获取公开课件列表
func (s *courseService) GetPublicCourses(req GetCoursesRequest) (*GetCoursesResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	offset := (req.Page - 1) * req.PageSize

	// 构建过滤器
	filters := repositories.CourseFilters{
		Status:    req.Status,
		Category:  req.Category,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// 获取公开课件列表
	courses, total, err := s.courseRepo.GetPublicCourses(offset, req.PageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("获取公开课件列表失败: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &GetCoursesResponse{
		Courses:    courses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// SearchCourses 搜索课件
func (s *courseService) SearchCourses(req SearchCoursesRequest) (*GetCoursesResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	offset := (req.Page - 1) * req.PageSize

	// 搜索课件（支持用户ID过滤）
	courses, total, err := s.courseRepo.SearchCoursesByUser(req.Keyword, req.UserID, offset, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("搜索课件失败: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &GetCoursesResponse{
		Courses:    courses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateCourse 更新课件
func (s *courseService) UpdateCourse(id uint, userID uint, req UpdateCourseRequest) (*models.Course, error) {
	if id == 0 {
		return nil, errors.New("课件ID不能为空")
	}

	// 获取课件信息
	course, err := s.courseRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查权限：只有创建者才能更新
	if course.UserID != userID {
		return nil, errors.New("无权限更新该课件")
	}

	// 检查课件状态：生成中的课件不允许更新
	if course.Status == models.CourseStatusGenerating {
		return nil, errors.New("课件生成中，无法更新")
	}

	// 更新字段
	if req.Title != nil {
		course.Title = *req.Title
	}
	if req.Description != nil {
		course.Description = *req.Description
	}
	if req.Category != nil {
		course.Category = *req.Category
	}
	if req.Tags != nil {
		course.Tags = *req.Tags
	}
	if req.IsPublic != nil {
		course.IsPublic = *req.IsPublic
	}
	if req.PPTFilePath != nil {
		course.PPTFilePath = *req.PPTFilePath
	}

	// 保存更新
	if err := s.courseRepo.Update(course); err != nil {
		return nil, fmt.Errorf("更新课件失败: %w", err)
	}

	return course, nil
}

// DeleteCourse 删除课件
func (s *courseService) DeleteCourse(id uint, userID uint) error {
	if id == 0 {
		return errors.New("课件ID不能为空")
	}

	// 获取课件信息
	course, err := s.courseRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 检查权限：只有创建者才能删除
	if course.UserID != userID {
		return errors.New("无权限删除该课件")
	}

	// 删除课件
	if err := s.courseRepo.Delete(id); err != nil {
		return fmt.Errorf("删除课件失败: %w", err)
	}

	// TODO: 删除相关文件（音频、图片等）
	// 这里需要集成存储服务

	return nil
}

// IncrementViewCount 增加观看次数
func (s *courseService) IncrementViewCount(id uint) error {
	return s.courseRepo.IncrementViewCount(id)
}

// IncrementLikeCount 增加点赞数
func (s *courseService) IncrementLikeCount(id uint) error {
	return s.courseRepo.IncrementLikeCount(id)
}

// IncrementShareCount 增加分享次数
func (s *courseService) IncrementShareCount(id uint) error {
	return s.courseRepo.IncrementShareCount(id)
}

// GetUserStats 获取用户课件统计
func (s *courseService) GetUserStats(userID uint) (*UserCourseStats, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为空")
	}

	// 获取总课件数
	totalCourses, err := s.courseRepo.GetUserCourseCount(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户课件统计失败: %w", err)
	}

	// TODO: 实现其他统计数据的计算
	// 这里可以通过多个查询或者一个复杂的SQL来获取各种统计数据

	stats := &UserCourseStats{
		TotalCourses:      totalCourses,
		CompletedCourses:  0, // TODO: 实现
		GeneratingCourses: 0, // TODO: 实现
		FailedCourses:     0, // TODO: 实现
		TotalViews:        0, // TODO: 实现
		TotalLikes:        0, // TODO: 实现
		TotalShares:       0, // TODO: 实现
	}

	return stats, nil
}

// ValidateCreateRequest 验证创建课件请求
func (s *courseService) ValidateCreateRequest(req CreateCourseRequest) error {
	// 基础验证
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("课件标题不能为空")
	}

	if len(req.Title) > 200 {
		return errors.New("课件标题不能超过200字符")
	}

	if len(req.Description) > 1000 {
		return errors.New("课件描述不能超过1000字符")
	}

	if req.SourceType == "" {
		return errors.New("来源类型不能为空")
	}

	if strings.TrimSpace(req.SourceContent) == "" {
		return errors.New("源内容不能为空")
	}

	// 验证来源类型
	validSourceTypes := map[models.SourceType]bool{
		models.SourceTypeURL:      true,
		models.SourceTypeDocument: true,
		models.SourceTypeText:     true,
	}

	if !validSourceTypes[req.SourceType] {
		return errors.New("无效的来源类型")
	}

	// 验证分类
	validCategories := map[models.CourseCategory]bool{
		models.CategoryGeneral:     true,
		models.CategoryProgramming: true,
		models.CategoryDatabase:    true,
		models.CategoryFrontend:    true,
		models.CategoryAI:          true,
		models.CategoryDevOps:      true,
	}

	if !validCategories[req.Category] {
		return errors.New("无效的课件分类")
	}

	// 如果是URL类型，验证URL格式
	if req.SourceType == models.SourceTypeURL {
		if req.SourceURL == "" {
			return errors.New("URL来源必须提供源链接")
		}
		// TODO: 添加URL格式验证
	}

	return nil
}

// ValidateFileUpload 验证文件上传
func (s *courseService) ValidateFileUpload(file *multipart.FileHeader) error {
	if file == nil {
		return errors.New("文件不能为空")
	}

	// 检查文件大小（10MB限制）
	maxFileSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxFileSize {
		return errors.New("文件大小不能超过10MB")
	}

	// 检查文件类型
	filename := strings.ToLower(file.Filename)
	validExtensions := []string{".pdf", ".doc", ".docx", ".txt", ".md"}

	validExt := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			validExt = true
			break
		}
	}

	if !validExt {
		return errors.New("不支持的文件类型，仅支持PDF、Word、TXT、Markdown文件")
	}

	return nil
}

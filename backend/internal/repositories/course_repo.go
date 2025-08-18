package repositories

import (
	"ai-classroom/internal/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// CourseRepository 课件数据访问接口
type CourseRepository interface {
	Create(course *models.Course) error
	GetByID(id uint) (*models.Course, error)
	GetByUserID(userID uint, offset, limit int, filters CourseFilters) ([]*models.Course, int64, error)
	Update(course *models.Course) error
	Delete(id uint) error
	GetPublicCourses(offset, limit int, filters CourseFilters) ([]*models.Course, int64, error)
	SearchCourses(keyword string, offset, limit int) ([]*models.Course, int64, error)
	SearchCoursesByUser(keyword string, userID uint, offset, limit int) ([]*models.Course, int64, error)
	IncrementViewCount(id uint) error
	IncrementLikeCount(id uint) error
	IncrementShareCount(id uint) error
	GetCoursesByStatus(status models.CourseStatus, limit int) ([]*models.Course, error)
	UpdateStatus(id uint, status models.CourseStatus, errorMessage string) error
	GetUserCourseCount(userID uint) (int64, error)
	CreateSlide(slide *models.Slide) error
	UpdateSlide(slide *models.Slide) error
}

// CourseFilters 课件查询过滤器
type CourseFilters struct {
	Status    models.CourseStatus   `json:"status"`
	Category  models.CourseCategory `json:"category"`
	IsPublic  *bool                 `json:"is_public"`
	SortBy    string                `json:"sort_by"`    // created_at, view_count, like_count
	SortOrder string                `json:"sort_order"` // asc, desc
}

// courseRepository 课件数据访问实现
type courseRepository struct {
	db *gorm.DB
}

// NewCourseRepository 创建课件数据访问实例
func NewCourseRepository(db *gorm.DB) CourseRepository {
	return &courseRepository{db: db}
}

// Create 创建课件
func (r *courseRepository) Create(course *models.Course) error {
	if course == nil {
		return errors.New("课件对象不能为空")
	}

	// 基础验证
	if course.UserID == 0 {
		return errors.New("用户ID不能为空")
	}
	if course.Title == "" {
		return errors.New("课件标题不能为空")
	}
	if course.SourceContent == "" {
		return errors.New("源内容不能为空")
	}

	// 使用事务确保数据一致性
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	result := tx.Create(course)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("创建课件失败: %w", result.Error)
	}

	// 验证ID是否正确生成
	if course.ID == 0 {
		tx.Rollback()
		return errors.New("课件ID生成失败")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("课件创建成功: ID=%d, UserID=%d, Title=%s\n", course.ID, course.UserID, course.Title)
	return nil
}

// GetByID 根据ID获取课件
func (r *courseRepository) GetByID(id uint) (*models.Course, error) {
	if id == 0 {
		return nil, errors.New("课件ID不能为空")
	}

	var course models.Course
	result := r.db.Preload("Slides").First(&course, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("课件不存在")
		}
		return nil, fmt.Errorf("获取课件失败: %w", result.Error)
	}

	return &course, nil
}

// GetByUserID 根据用户ID获取课件列表
func (r *courseRepository) GetByUserID(userID uint, offset, limit int, filters CourseFilters) ([]*models.Course, int64, error) {
	if userID == 0 {
		return nil, 0, errors.New("用户ID不能为空")
	}

	var courses []*models.Course
	var total int64

	// 构建查询条件
	query := r.db.Model(&models.Course{}).Where("user_id = ?", userID)

	// 应用过滤器
	query = r.applyFilters(query, filters)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取课件总数失败: %w", err)
	}

	// 应用排序
	query = r.applySorting(query, filters)

	// 分页查询
	if err := query.Offset(offset).Limit(limit).Find(&courses).Error; err != nil {
		return nil, 0, fmt.Errorf("获取用户课件列表失败: %w", err)
	}

	return courses, total, nil
}

// Update 更新课件
func (r *courseRepository) Update(course *models.Course) error {
	if course == nil {
		return errors.New("课件对象不能为空")
	}
	if course.ID == 0 {
		return errors.New("课件ID不能为空")
	}

	// 检查课件是否存在
	var existingCourse models.Course
	if err := r.db.First(&existingCourse, course.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("课件不存在")
		}
		return fmt.Errorf("检查课件存在性失败: %w", err)
	}

	result := r.db.Save(course)
	if result.Error != nil {
		return fmt.Errorf("更新课件失败: %w", result.Error)
	}

	return nil
}

// Delete 删除课件
func (r *courseRepository) Delete(id uint) error {
	if id == 0 {
		return errors.New("课件ID不能为空")
	}

	// 检查课件是否存在
	var course models.Course
	if err := r.db.First(&course, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("课件不存在")
		}
		return fmt.Errorf("检查课件存在性失败: %w", err)
	}

	result := r.db.Delete(&models.Course{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除课件失败: %w", result.Error)
	}

	return nil
}

// GetPublicCourses 获取公开课件列表
func (r *courseRepository) GetPublicCourses(offset, limit int, filters CourseFilters) ([]*models.Course, int64, error) {
	var courses []*models.Course
	var total int64

	// 构建查询条件（只查询公开课件）
	query := r.db.Model(&models.Course{}).Where("is_public = ?", true)

	// 应用过滤器
	query = r.applyFilters(query, filters)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取公开课件总数失败: %w", err)
	}

	// 应用排序
	query = r.applySorting(query, filters)

	// 分页查询
	if err := query.Offset(offset).Limit(limit).Find(&courses).Error; err != nil {
		return nil, 0, fmt.Errorf("获取公开课件列表失败: %w", err)
	}

	return courses, total, nil
}

// SearchCourses 搜索课件
func (r *courseRepository) SearchCourses(keyword string, offset, limit int) ([]*models.Course, int64, error) {
	if keyword == "" {
		return nil, 0, errors.New("搜索关键词不能为空")
	}

	var courses []*models.Course
	var total int64

	// 构建搜索查询（支持多字段搜索）
	searchPattern := "%" + keyword + "%"
	query := r.db.Model(&models.Course{}).Where(
		"(title LIKE ? OR description LIKE ? OR tags LIKE ?)",
		searchPattern, searchPattern, searchPattern,
	)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取搜索结果总数失败: %w", err)
	}

	// 按相关性排序（标题匹配优先，然后按创建时间排序）
	if err := query.Order("CASE WHEN title LIKE ? THEN 1 ELSE 2 END, created_at DESC").
		Offset(offset).Limit(limit).Find(&courses).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索课件失败: %w", err)
	}

	return courses, total, nil
}

// SearchCoursesByUser 搜索用户的课件
func (r *courseRepository) SearchCoursesByUser(keyword string, userID uint, offset, limit int) ([]*models.Course, int64, error) {
	if keyword == "" {
		return nil, 0, errors.New("搜索关键词不能为空")
	}

	var courses []*models.Course
	var total int64

	// 构建搜索查询（支持多字段搜索，并限制为用户自己的课程）
	searchPattern := "%" + keyword + "%"
	query := r.db.Model(&models.Course{}).Where(
		"(title LIKE ? OR description LIKE ? OR tags LIKE ?) AND user_id = ?",
		searchPattern, searchPattern, searchPattern, userID,
	)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取搜索结果总数失败: %w", err)
	}

	// 按相关性排序（标题匹配优先，然后按创建时间排序）
	if err := query.Order("CASE WHEN title LIKE ? THEN 1 ELSE 2 END, created_at DESC").
		Offset(offset).Limit(limit).Find(&courses).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索课件失败: %w", err)
	}

	return courses, total, nil
}

// IncrementViewCount 增加观看次数
func (r *courseRepository) IncrementViewCount(id uint) error {
	if id == 0 {
		return errors.New("课件ID不能为空")
	}

	result := r.db.Model(&models.Course{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("增加观看次数失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("课件不存在")
	}

	return nil
}

// IncrementLikeCount 增加点赞数
func (r *courseRepository) IncrementLikeCount(id uint) error {
	if id == 0 {
		return errors.New("课件ID不能为空")
	}

	result := r.db.Model(&models.Course{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("增加点赞数失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("课件不存在")
	}

	return nil
}

// IncrementShareCount 增加分享次数
func (r *courseRepository) IncrementShareCount(id uint) error {
	if id == 0 {
		return errors.New("课件ID不能为空")
	}

	result := r.db.Model(&models.Course{}).Where("id = ?", id).
		UpdateColumn("share_count", gorm.Expr("share_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("增加分享次数失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("课件不存在")
	}

	return nil
}

// GetCoursesByStatus 根据状态获取课件列表
func (r *courseRepository) GetCoursesByStatus(status models.CourseStatus, limit int) ([]*models.Course, error) {
	var courses []*models.Course

	query := r.db.Where("status = ?", status)
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("根据状态获取课件失败: %w", err)
	}

	return courses, nil
}

// UpdateStatus 更新课件状态
func (r *courseRepository) UpdateStatus(id uint, status models.CourseStatus, errorMessage string) error {
	if id == 0 {
		return errors.New("课件ID不能为空")
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	result := r.db.Model(&models.Course{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新课件状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("课件不存在")
	}

	return nil
}

// GetUserCourseCount 获取用户课件总数
func (r *courseRepository) GetUserCourseCount(userID uint) (int64, error) {
	if userID == 0 {
		return 0, errors.New("用户ID不能为空")
	}

	var count int64
	if err := r.db.Model(&models.Course{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取用户课件总数失败: %w", err)
	}

	return count, nil
}

// CreateSlide 创建幻灯片
func (r *courseRepository) CreateSlide(slide *models.Slide) error {
	if slide == nil {
		return errors.New("幻灯片对象不能为空")
	}

	if slide.CourseID == 0 {
		return errors.New("课件ID不能为空")
	}

	if slide.SlideNumber <= 0 {
		return errors.New("幻灯片序号必须大于0")
	}

	result := r.db.Create(slide)
	if result.Error != nil {
		return fmt.Errorf("创建幻灯片失败: %w", result.Error)
	}

	return nil
}

// UpdateSlide 更新幻灯片
func (r *courseRepository) UpdateSlide(slide *models.Slide) error {
	if slide == nil {
		return errors.New("幻灯片对象不能为空")
	}

	if slide.ID == 0 {
		return errors.New("幻灯片ID不能为空")
	}

	// 检查幻灯片是否存在
	var existingSlide models.Slide
	if err := r.db.First(&existingSlide, slide.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("幻灯片不存在")
		}
		return fmt.Errorf("检查幻灯片存在性失败: %w", err)
	}

	result := r.db.Save(slide)
	if result.Error != nil {
		return fmt.Errorf("更新幻灯片失败: %w", result.Error)
	}

	return nil
}

// applyFilters 应用查询过滤器
func (r *courseRepository) applyFilters(query *gorm.DB, filters CourseFilters) *gorm.DB {
	// 状态过滤
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	// 分类过滤
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}

	// 公开状态过滤
	if filters.IsPublic != nil {
		query = query.Where("is_public = ?", *filters.IsPublic)
	}

	return query
}

// applySorting 应用排序
func (r *courseRepository) applySorting(query *gorm.DB, filters CourseFilters) *gorm.DB {
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// 验证排序字段
	validSortFields := map[string]bool{
		"created_at":  true,
		"updated_at":  true,
		"view_count":  true,
		"like_count":  true,
		"share_count": true,
		"title":       true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	// 验证排序方向
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
}

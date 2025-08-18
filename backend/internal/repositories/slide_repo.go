package repositories

import (
	"errors"

	"gorm.io/gorm"
	"ai-classroom/internal/models"
)

// SlideRepository 幻灯片仓库接口
type SlideRepository interface {
	Create(slide *models.Slide) error
	GetByID(id uint) (*models.Slide, error)
	GetByCourseID(courseID uint) ([]models.Slide, error)
	GetByCourseIDOrdered(courseID uint) ([]models.Slide, error)
	Update(slide *models.Slide) error
	Delete(id uint) error
	DeleteByCourseID(courseID uint) error
	BatchCreate(slides []models.Slide) error
	GetSlideCount(courseID uint) (int64, error)
}

// slideRepository 幻灯片仓库实现
type slideRepository struct {
	db *gorm.DB
}

// NewSlideRepository 创建幻灯片仓库
func NewSlideRepository(db *gorm.DB) SlideRepository {
	return &slideRepository{db: db}
}

// Create 创建幻灯片
func (r *slideRepository) Create(slide *models.Slide) error {
	return r.db.Create(slide).Error
}

// GetByID 根据ID获取幻灯片
func (r *slideRepository) GetByID(id uint) (*models.Slide, error) {
	var slide models.Slide
	err := r.db.Preload("Course").First(&slide, id).Error
	if err != nil {
		return nil, err
	}
	return &slide, nil
}

// GetByCourseID 根据课程ID获取幻灯片
func (r *slideRepository) GetByCourseID(courseID uint) ([]models.Slide, error) {
	var slides []models.Slide
	err := r.db.Where("course_id = ?", courseID).Find(&slides).Error
	return slides, err
}

// GetByCourseIDOrdered 根据课程ID获取幻灯片（按序号排序）
func (r *slideRepository) GetByCourseIDOrdered(courseID uint) ([]models.Slide, error) {
	var slides []models.Slide
	err := r.db.Where("course_id = ?", courseID).Order("slide_number ASC").Find(&slides).Error
	return slides, err
}

// Update 更新幻灯片
func (r *slideRepository) Update(slide *models.Slide) error {
	return r.db.Save(slide).Error
}

// Delete 删除幻灯片
func (r *slideRepository) Delete(id uint) error {
	return r.db.Delete(&models.Slide{}, id).Error
}

// DeleteByCourseID 根据课程ID删除幻灯片
func (r *slideRepository) DeleteByCourseID(courseID uint) error {
	return r.db.Where("course_id = ?", courseID).Delete(&models.Slide{}).Error
}

// BatchCreate 批量创建幻灯片
func (r *slideRepository) BatchCreate(slides []models.Slide) error {
	if len(slides) == 0 {
		return errors.New("幻灯片列表为空")
	}
	return r.db.Create(&slides).Error
}

// GetSlideCount 获取课程幻灯片数量
func (r *slideRepository) GetSlideCount(courseID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Slide{}).Where("course_id = ?", courseID).Count(&count).Error
	return count, err
}
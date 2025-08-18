package services

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-classroom/internal/models"
	"ai-classroom/pkg/ppt_parser"

	"gorm.io/gorm"
)

// PPTConversionService PPT转换服务接口
type PPTConversionService interface {
	ConvertSinglePPT(filePath string) (*ConversionResult, error)
	ConvertBatchPPT(pptDir string) (*BatchConversionResult, error)
	ImportSlidesToCourse(courseID uint, filePath string) error
	GetConversionStatus(filePath string) (*ConversionStatus, error)
	ScanAndImportNewPPTs(pptDir string) (*BatchConversionResult, error)
}

// pptConversionService PPT转换服务实现
type pptConversionService struct {
	db     *gorm.DB
	parser *ppt_parser.HTMLParser
}

// ConversionResult 单个文件转换结果
type ConversionResult struct {
	FilePath    string                  `json:"file_path"`
	CourseID    int                     `json:"course_id"`
	Success     bool                    `json:"success"`
	Error       string                  `json:"error,omitempty"`
	SlidesCount int                     `json:"slides_count"`
	Metadata    *ppt_parser.PPTMetadata `json:"metadata,omitempty"`
	ProcessedAt time.Time               `json:"processed_at"`
}

// BatchConversionResult 批量转换结果
type BatchConversionResult struct {
	TotalFiles      int                 `json:"total_files"`
	SuccessfulFiles int                 `json:"successful_files"`
	FailedFiles     int                 `json:"failed_files"`
	Results         []*ConversionResult `json:"results"`
	ProcessedAt     time.Time           `json:"processed_at"`
}

// ConversionStatus 转换状态
type ConversionStatus struct {
	FilePath    string    `json:"file_path"`
	Status      string    `json:"status"` // pending, processing, completed, failed
	ProcessedAt time.Time `json:"processed_at"`
	Error       string    `json:"error,omitempty"`
	SlidesCount int       `json:"slides_count"`
}

// NewPPTConversionService 创建PPT转换服务实例
func NewPPTConversionService(db *gorm.DB) PPTConversionService {
	return &pptConversionService{
		db:     db,
		parser: ppt_parser.NewHTMLParser(),
	}
}

// ConvertSinglePPT 转换单个PPT文件
func (s *pptConversionService) ConvertSinglePPT(filePath string) (*ConversionResult, error) {
	result := &ConversionResult{
		FilePath:    filePath,
		ProcessedAt: time.Now(),
	}

	// 1. 读取HTML文件
	htmlContent, err := s.readHTMLFile(filePath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("读取文件失败: %v", err)
		return result, err
	}

	// 2. 解析HTML内容
	metadata, slides, err := s.parser.ParsePPTFile(filePath, htmlContent)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("解析HTML失败: %v", err)
		return result, err
	}

	// 3. 验证幻灯片数据
	if err := s.parser.ValidateSlideData(slides); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("幻灯片数据验证失败: %v", err)
		return result, err
	}

	// 4. 检查课程是否存在
	course, err := s.getCourse(uint(metadata.CourseID))
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("课程不存在: %v", err)
		return result, err
	}

	// 5. 保存幻灯片到数据库
	if err := s.saveSlides(course, slides, metadata); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("保存幻灯片失败: %v", err)
		return result, err
	}

	// 6. 更新课程信息
	if err := s.updateCourseInfo(course, len(slides), metadata); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("更新课程信息失败: %v", err)
		return result, err
	}

	result.Success = true
	result.CourseID = metadata.CourseID
	result.SlidesCount = len(slides)
	result.Metadata = metadata

	log.Printf("✅ PPT转换成功: %s -> 课程ID: %d, 幻灯片数: %d", filePath, metadata.CourseID, len(slides))
	return result, nil
}

// ConvertBatchPPT 批量转换PPT文件
func (s *pptConversionService) ConvertBatchPPT(pptDir string) (*BatchConversionResult, error) {
	result := &BatchConversionResult{
		ProcessedAt: time.Now(),
		Results:     make([]*ConversionResult, 0),
	}

	// 扫描PPT目录
	files, err := s.scanPPTFiles(pptDir)
	if err != nil {
		return nil, fmt.Errorf("扫描PPT目录失败: %w", err)
	}

	result.TotalFiles = len(files)

	// 逐个处理文件
	for _, file := range files {
		convertResult, err := s.ConvertSinglePPT(file)
		if err != nil {
			log.Printf("❌ 转换文件失败: %s, 错误: %v", file, err)
			result.FailedFiles++
		} else if convertResult.Success {
			result.SuccessfulFiles++
		} else {
			result.FailedFiles++
		}

		result.Results = append(result.Results, convertResult)
	}

	log.Printf("📊 批量转换完成: 总数=%d, 成功=%d, 失败=%d",
		result.TotalFiles, result.SuccessfulFiles, result.FailedFiles)

	return result, nil
}

// ImportSlidesToCourse 为指定课程导入幻灯片
func (s *pptConversionService) ImportSlidesToCourse(courseID uint, filePath string) error {
	// 读取和解析文件
	htmlContent, err := s.readHTMLFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	metadata, slides, err := s.parser.ParsePPTFile(filePath, htmlContent)
	if err != nil {
		return fmt.Errorf("解析HTML失败: %w", err)
	}

	// 验证幻灯片数据
	if err := s.parser.ValidateSlideData(slides); err != nil {
		return fmt.Errorf("幻灯片数据验证失败: %w", err)
	}

	// 获取课程
	course, err := s.getCourse(courseID)
	if err != nil {
		return fmt.Errorf("课程不存在: %w", err)
	}

	// 删除现有幻灯片（如果有）
	if err := s.deleteExistingSlides(courseID); err != nil {
		return fmt.Errorf("删除现有幻灯片失败: %w", err)
	}

	// 保存新幻灯片
	if err := s.saveSlides(course, slides, metadata); err != nil {
		return fmt.Errorf("保存幻灯片失败: %w", err)
	}

	// 更新课程信息
	if err := s.updateCourseInfo(course, len(slides), metadata); err != nil {
		return fmt.Errorf("更新课程信息失败: %w", err)
	}

	log.Printf("✅ 成功为课程 %d 导入 %d 张幻灯片", courseID, len(slides))
	return nil
}

// GetConversionStatus 获取转换状态
func (s *pptConversionService) GetConversionStatus(filePath string) (*ConversionStatus, error) {
	// 从文件名提取课程ID，只需要元数据
	metadata, err := s.parser.ExtractMetadataFromFileName(filePath)
	if err != nil {
		return nil, fmt.Errorf("解析文件名失败: %w", err)
	}

	// 检查课程中是否已有幻灯片
	var count int64
	err = s.db.Model(&models.Slide{}).Where("course_id = ?", metadata.CourseID).Count(&count).Error
	if err != nil {
		return &ConversionStatus{
			FilePath: filePath,
			Status:   "failed",
			Error:    err.Error(),
		}, nil
	}

	status := "pending"
	if count > 0 {
		status = "completed"
	}

	return &ConversionStatus{
		FilePath:    filePath,
		Status:      status,
		SlidesCount: int(count),
		ProcessedAt: time.Now(),
	}, nil
}

// ScanAndImportNewPPTs 扫描并导入新的PPT文件
func (s *pptConversionService) ScanAndImportNewPPTs(pptDir string) (*BatchConversionResult, error) {
	result := &BatchConversionResult{
		ProcessedAt: time.Now(),
		Results:     make([]*ConversionResult, 0),
	}

	// 扫描PPT目录
	files, err := s.scanPPTFiles(pptDir)
	if err != nil {
		return nil, fmt.Errorf("扫描PPT目录失败: %w", err)
	}

	// 过滤已处理的文件
	newFiles := make([]string, 0)
	for _, file := range files {
		status, err := s.GetConversionStatus(file)
		if err != nil || status.Status == "pending" {
			newFiles = append(newFiles, file)
		}
	}

	result.TotalFiles = len(newFiles)

	// 处理新文件
	for _, file := range newFiles {
		convertResult, err := s.ConvertSinglePPT(file)
		if err != nil {
			log.Printf("❌ 转换新文件失败: %s, 错误: %v", file, err)
			result.FailedFiles++
		} else if convertResult.Success {
			result.SuccessfulFiles++
		} else {
			result.FailedFiles++
		}

		result.Results = append(result.Results, convertResult)
	}

	log.Printf("📊 新文件转换完成: 新文件=%d, 成功=%d, 失败=%d",
		result.TotalFiles, result.SuccessfulFiles, result.FailedFiles)

	return result, nil
}

// readHTMLFile 读取HTML文件内容
func (s *pptConversionService) readHTMLFile(filePath string) (string, error) {
	if !strings.HasSuffix(filePath, ".html") {
		return "", fmt.Errorf("文件类型不支持: %s", filePath)
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	return string(content), nil
}

// scanPPTFiles 扫描PPT目录中的HTML文件
func (s *pptConversionService) scanPPTFiles(pptDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(pptDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理HTML文件，且文件名符合PPT命名规则
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			fileName := info.Name()
			if strings.HasPrefix(fileName, "ppt_") {
				files = append(files, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("扫描目录失败: %w", err)
	}

	return files, nil
}

// getCourse 获取课程信息
func (s *pptConversionService) getCourse(courseID uint) (*models.Course, error) {
	var course models.Course
	err := s.db.First(&course, courseID).Error
	if err != nil {
		return nil, fmt.Errorf("课程 %d 不存在: %w", courseID, err)
	}
	return &course, nil
}

// deleteExistingSlides 删除现有幻灯片
func (s *pptConversionService) deleteExistingSlides(courseID uint) error {
	return s.db.Where("course_id = ?", courseID).Delete(&models.Slide{}).Error
}

// saveSlides 保存幻灯片到数据库
func (s *pptConversionService) saveSlides(course *models.Course, slides []ppt_parser.SlideData, metadata *ppt_parser.PPTMetadata) error {
	// 开始事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 逐个保存幻灯片
	for _, slideData := range slides {
		slide := &models.Slide{
			CourseID:    course.ID,
			SlideNumber: slideData.SlideNumber,
			Title:       slideData.Title,
			Content:     slideData.Content,
			Keywords:    models.Keywords(slideData.Keywords),
			Notes:       slideData.SpeakerNotes,
			Duration:    slideData.Duration,
			CreatedAt:   metadata.GeneratedAt,
			UpdatedAt:   time.Now(),
		}

		if err := tx.Create(slide).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存幻灯片 %d 失败: %w", slideData.SlideNumber, err)
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// updateCourseInfo 更新课程信息
func (s *pptConversionService) updateCourseInfo(course *models.Course, slidesCount int, metadata *ppt_parser.PPTMetadata) error {
	updates := map[string]interface{}{
		"slides_count": slidesCount,
		"updated_at":   time.Now(),
	}

	// 如果课程还没有PPT文件路径，设置它
	if course.PPTFilePath == "" {
		updates["ppt_file_path"] = metadata.FilePath
	}

	// 计算总时长
	var totalDuration int64
	s.db.Model(&models.Slide{}).Where("course_id = ?", course.ID).Select("SUM(duration)").Scan(&totalDuration)
	updates["duration"] = int(totalDuration)

	return s.db.Model(course).Updates(updates).Error
}

// GetStatistics 获取转换统计信息
func (s *pptConversionService) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总课程数
	var totalCourses int64
	s.db.Model(&models.Course{}).Count(&totalCourses)
	stats["total_courses"] = totalCourses

	// 有幻灯片的课程数
	var coursesWithSlides int64
	s.db.Model(&models.Course{}).Where("slides_count > 0").Count(&coursesWithSlides)
	stats["courses_with_slides"] = coursesWithSlides

	// 总幻灯片数
	var totalSlides int64
	s.db.Model(&models.Slide{}).Count(&totalSlides)
	stats["total_slides"] = totalSlides

	// 转换率
	if totalCourses > 0 {
		stats["conversion_rate"] = float64(coursesWithSlides) / float64(totalCourses) * 100
	} else {
		stats["conversion_rate"] = 0.0
	}

	return stats, nil
}

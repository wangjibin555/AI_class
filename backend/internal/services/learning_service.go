package services

import (
	"encoding/json"
	"fmt"
	"time"

	"ai-classroom/internal/models"
	"ai-classroom/internal/repositories"

	"gorm.io/gorm"
)

// LearningService 学习记录服务
type LearningService struct {
	db         *gorm.DB
	courseRepo repositories.CourseRepository
}

// ProgressData 学习进度数据
type ProgressData struct {
	LearningRecordID uint                `json:"learning_record_id"`
	CourseID         uint                `json:"course_id"`
	CurrentSlide     int                 `json:"current_slide"`
	TotalTime        int                 `json:"total_time"`      // 总学习时间(秒)
	CompletionRate   float64             `json:"completion_rate"` // 完成率(0-1)
	SlidesProgress   []SlideProgressData `json:"slides_progress"`
}

// SlideProgressData 幻灯片进度数据
type SlideProgressData struct {
	SlideIndex    int       `json:"slide_index"`
	FirstAccess   time.Time `json:"first_access"`
	LastAccess    time.Time `json:"last_access"`
	TotalTime     int       `json:"total_time"`     // 该幻灯片学习时间(秒)
	AudioProgress float64   `json:"audio_progress"` // 音频播放进度(0-1)
	Completed     bool      `json:"completed"`
}

// StartLearningRequest 开始学习请求
type StartLearningRequest struct {
	CourseID     uint   `json:"course_id" binding:"required"`
	StartTime    int64  `json:"start_time" binding:"required"`
	LearningType string `json:"learning_type" binding:"required"` // audio_ppt, text_only
}

// StartLearningResponse 开始学习响应
type StartLearningResponse struct {
	LearningRecordID uint      `json:"learning_record_id"`
	CourseID         uint      `json:"course_id"`
	StartTime        time.Time `json:"start_time"`
	LearningType     string    `json:"learning_type"`
	Message          string    `json:"message"`
}

// NewLearningService 创建学习记录服务
func NewLearningService(
	db *gorm.DB,
	courseRepo repositories.CourseRepository,
) *LearningService {
	return &LearningService{
		db:         db,
		courseRepo: courseRepo,
	}
}

// StartLearning 开始学习记录
func (s *LearningService) StartLearning(userID uint, request StartLearningRequest) (*StartLearningResponse, error) {
	// 1. 验证课程是否存在
	_, err := s.courseRepo.GetByID(request.CourseID)
	if err != nil {
		return nil, fmt.Errorf("课程不存在: %w", err)
	}

	// 2. 检查是否已有未完成的学习记录
	existingRecord, err := s.getActiveRecordByCourse(userID, request.CourseID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询学习记录失败: %w", err)
	}

	var learningRecord *models.LearningRecord
	startTime := time.Unix(request.StartTime/1000, 0)

	if existingRecord != nil {
		// 更新现有记录
		existingRecord.StartTime = &startTime
		existingRecord.Status = "learning"
		existingRecord.LastStudyTime = &startTime

		if err := s.db.Save(existingRecord).Error; err != nil {
			return nil, fmt.Errorf("更新学习记录失败: %w", err)
		}
		learningRecord = existingRecord
	} else {
		// 创建新的学习记录
		learningRecord = &models.LearningRecord{
			UserID:        userID,
			CourseID:      request.CourseID,
			StartTime:     &startTime,
			CurrentSlide:  1,
			TotalSlides:   s.getTotalSlides(request.CourseID),
			Progress:      0,
			StudyDuration: 0,
			CompleteRate:  0,
			IsCompleted:   false,
			Status:        "learning",
			LastStudyTime: &startTime,
		}

		if err := s.db.Create(learningRecord).Error; err != nil {
			return nil, fmt.Errorf("创建学习记录失败: %w", err)
		}
	}

	// 3. 记录操作日志
	s.logLearningStart(userID, request.CourseID, request.LearningType)

	return &StartLearningResponse{
		LearningRecordID: learningRecord.ID,
		CourseID:         learningRecord.CourseID,
		StartTime:        *learningRecord.StartTime,
		LearningType:     request.LearningType,
		Message:          "学习记录创建成功",
	}, nil
}

// UpdateProgress 更新学习进度
func (s *LearningService) UpdateProgress(userID uint, progressData ProgressData) error {
	// 1. 验证学习记录
	var learningRecord models.LearningRecord
	if err := s.db.Where("id = ? AND user_id = ?", progressData.LearningRecordID, userID).
		First(&learningRecord).Error; err != nil {
		return fmt.Errorf("学习记录不存在或无权限: %w", err)
	}

	// 2. 更新基本进度信息
	now := time.Now()
	updates := map[string]interface{}{
		"current_slide":   progressData.CurrentSlide,
		"study_duration":  progressData.TotalTime,
		"complete_rate":   progressData.CompletionRate, // 直接存储0-1范围的小数
		"progress":        progressData.CompletionRate, // 直接存储0-1范围的小数
		"last_study_time": now,
		"updated_at":      now,
	}

	// 检查是否完成
	if progressData.CompletionRate >= 0.9 { // 90%算完成
		updates["is_completed"] = true
		updates["status"] = "completed"
		endTime := now
		updates["end_time"] = &endTime
	}

	if err := s.db.Model(&learningRecord).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新学习记录失败: %w", err)
	}

	// 3. 保存详细的幻灯片进度
	if len(progressData.SlidesProgress) > 0 {
		if err := s.updateSlideProgress(progressData.LearningRecordID, progressData.SlidesProgress); err != nil {
			// 不因为详细进度保存失败而影响主要进度更新
			fmt.Printf("保存幻灯片详细进度失败: %v\n", err)
		}
	}

	return nil
}

// GetLearningRecord 获取学习记录
func (s *LearningService) GetLearningRecord(userID, courseID uint) (*models.LearningRecord, error) {
	var record models.LearningRecord
	err := s.db.Where("user_id = ? AND course_id = ?", userID, courseID).
		Order("created_at DESC").
		First(&record).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &record, err
}

// GetUserLearningStats 获取用户学习统计
func (s *LearningService) GetUserLearningStats(userID uint) (map[string]interface{}, error) {
	var stats struct {
		TotalCourses      int64   `json:"total_courses"`
		CompletedCourses  int64   `json:"completed_courses"`
		TotalStudyTime    int64   `json:"total_study_time"`
		AvgCompletionRate float64 `json:"avg_completion_rate"`
	}

	// 总课程数
	if err := s.db.Model(&models.LearningRecord{}).
		Where("user_id = ?", userID).
		Distinct("course_id").
		Count(&stats.TotalCourses).Error; err != nil {
		return nil, err
	}

	// 完成课程数
	if err := s.db.Model(&models.LearningRecord{}).
		Where("user_id = ? AND is_completed = ?", userID, true).
		Count(&stats.CompletedCourses).Error; err != nil {
		return nil, err
	}

	// 总学习时间
	if err := s.db.Model(&models.LearningRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(study_duration), 0)").
		Scan(&stats.TotalStudyTime).Error; err != nil {
		return nil, err
	}

	// 平均完成率
	if err := s.db.Model(&models.LearningRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(AVG(complete_rate), 0)").
		Scan(&stats.AvgCompletionRate).Error; err != nil {
		return nil, err
	}

	// 智能转换平均完成率
	avgCompletionRate := stats.AvgCompletionRate
	if avgCompletionRate > 10 {
		avgCompletionRate = avgCompletionRate / 100 // 0-10000格式转换为百分比
	} else {
		avgCompletionRate = avgCompletionRate * 100 // 0-1格式转换为百分比
	}

	result := map[string]interface{}{
		"total_courses":         stats.TotalCourses,
		"completed_courses":     stats.CompletedCourses,
		"total_study_time":      stats.TotalStudyTime,
		"avg_completion_rate":   avgCompletionRate,
		"completion_percentage": float64(stats.CompletedCourses) / float64(stats.TotalCourses) * 100,
	}

	return result, nil
}

// getActiveRecordByCourse 获取课程的活跃学习记录
func (s *LearningService) getActiveRecordByCourse(userID, courseID uint) (*models.LearningRecord, error) {
	var record models.LearningRecord
	err := s.db.Where("user_id = ? AND course_id = ? AND status IN (?)",
		userID, courseID, []string{"learning", "paused"}).
		Order("created_at DESC").
		First(&record).Error

	return &record, err
}

// getTotalSlides 获取课程总幻灯片数
func (s *LearningService) getTotalSlides(courseID uint) int {
	var count int64
	s.db.Model(&models.Slide{}).Where("course_id = ?", courseID).Count(&count)
	return int(count)
}

// updateSlideProgress 更新幻灯片详细进度
func (s *LearningService) updateSlideProgress(recordID uint, slidesProgress []SlideProgressData) error {
	// 将详细进度保存为JSON字段（简化实现）
	// 在实际项目中可能需要单独的幻灯片进度表
	progressJSON, err := json.Marshal(slidesProgress)
	if err != nil {
		return fmt.Errorf("序列化幻灯片进度失败: %w", err)
	}

	// 这里可以扩展为专门的幻灯片进度表
	// 目前保存在操作日志中作为记录
	operationLog := &models.OperationLog{
		OperationType: "update_slide_progress",
		OperationDesc: fmt.Sprintf("更新学习记录 %d 的幻灯片进度", recordID),
		ResourceType:  "learning_record",
		ResourceID:    &recordID,
		RequestData:   models.RequestData{"slides_progress": string(progressJSON)},
		Status:        "success",
		CreatedAt:     time.Now(),
	}

	return s.db.Create(operationLog).Error
}

// logLearningStart 记录学习开始日志
func (s *LearningService) logLearningStart(userID, courseID uint, learningType string) {
	operationLog := &models.OperationLog{
		UserID:        &userID,
		OperationType: "start_learning",
		OperationDesc: fmt.Sprintf("用户开始学习课程 %d，学习模式: %s", courseID, learningType),
		ResourceType:  "course",
		ResourceID:    &courseID,
		RequestData: models.RequestData{
			"learning_type": learningType,
			"start_time":    time.Now().Format(time.RFC3339),
		},
		Status:    "success",
		CreatedAt: time.Now(),
	}

	s.db.Create(operationLog)
}

// GetLearningHistory 获取学习历史
func (s *LearningService) GetLearningHistory(userID uint, limit, offset int) ([]models.LearningRecord, error) {
	var records []models.LearningRecord

	query := s.db.Where("user_id = ?", userID).
		Order("updated_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&records).Error
	return records, err
}

// PauseLearning 暂停学习
func (s *LearningService) PauseLearning(userID, courseID uint) error {
	return s.db.Model(&models.LearningRecord{}).
		Where("user_id = ? AND course_id = ? AND status = ?", userID, courseID, "learning").
		Updates(map[string]interface{}{
			"status":          "paused",
			"last_study_time": time.Now(),
		}).Error
}

// ResumeLearning 恢复学习
func (s *LearningService) ResumeLearning(userID, courseID uint) error {
	return s.db.Model(&models.LearningRecord{}).
		Where("user_id = ? AND course_id = ? AND status = ?", userID, courseID, "paused").
		Updates(map[string]interface{}{
			"status":          "learning",
			"last_study_time": time.Now(),
		}).Error
}

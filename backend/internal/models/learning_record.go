package models

import (
	"fmt"
	"math"
	"time"
)

// LearningStatus 学习状态
type LearningStatus string

const (
	LearningStatusLearning  LearningStatus = "learning"
	LearningStatusCompleted LearningStatus = "completed"
	LearningStatusPaused    LearningStatus = "paused"
)

// LearningRecord 学习记录模型
type LearningRecord struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"not null;index;comment:用户ID"`
	CourseID      uint           `json:"course_id" gorm:"not null;index;comment:课件ID"`
	CurrentSlide  int            `json:"current_slide" gorm:"default:1;comment:当前学习到的幻灯片"`
	TotalSlides   int            `json:"total_slides" gorm:"default:0;comment:总幻灯片数"`
	Progress      float64        `json:"progress" gorm:"default:0.00;comment:学习进度百分比"`
	StudyDuration int            `json:"study_duration" gorm:"default:0;comment:学习时长（秒）"`
	CompleteRate  float64        `json:"complete_rate" gorm:"default:0.00;comment:完成百分比"`
	LastPosition  int            `json:"last_position" gorm:"default:0;comment:最后播放位置"`
	QuizBestScore *float64       `json:"quiz_best_score" gorm:"comment:最佳练习得分"`
	QuizAttempts  int            `json:"quiz_attempts" gorm:"default:0;comment:练习尝试次数"`
	IsCompleted   bool           `json:"is_completed" gorm:"default:false;comment:是否完成学习"`
	IsBookmarked  bool           `json:"is_bookmarked" gorm:"default:false;comment:是否收藏"`
	Status        LearningStatus `json:"status" gorm:"type:enum('learning','completed','paused');default:'learning';comment:学习状态"`
	StartTime     *time.Time     `json:"start_time" gorm:"comment:开始学习时间"`
	EndTime       *time.Time     `json:"end_time" gorm:"comment:完成学习时间"`
	LastStudyTime *time.Time     `json:"last_study_time" gorm:"comment:最后学习时间"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// TableName 指定表名
func (LearningRecord) TableName() string {
	return "learning_records"
}

// UpdateProgress 更新学习进度
func (lr *LearningRecord) UpdateProgress(currentSlide, totalSlides int) {
	lr.CurrentSlide = currentSlide
	lr.TotalSlides = totalSlides

	if totalSlides > 0 {
		progress := float64(currentSlide) / float64(totalSlides) // 计算0-1范围的小数
		// 确保进度不超过1.0
		lr.Progress = math.Min(progress, 1.0)
		lr.CompleteRate = lr.Progress
	}

	now := time.Now()
	lr.LastStudyTime = &now

	// 如果进度达到100%（1.0），标记为完成
	if lr.Progress >= 1.0 {
		lr.IsCompleted = true
		lr.Status = LearningStatusCompleted
		lr.EndTime = &now
	}
}

// AddStudyTime 增加学习时长
func (lr *LearningRecord) AddStudyTime(duration int) {
	lr.StudyDuration += duration
	now := time.Now()
	lr.LastStudyTime = &now
}

// SetBookmark 设置收藏状态
func (lr *LearningRecord) SetBookmark(bookmarked bool) {
	lr.IsBookmarked = bookmarked
}

// StartLearning 开始学习
func (lr *LearningRecord) StartLearning() {
	if lr.StartTime == nil {
		now := time.Now()
		lr.StartTime = &now
	}
	lr.Status = LearningStatusLearning
	now := time.Now()
	lr.LastStudyTime = &now
}

// PauseLearning 暂停学习
func (lr *LearningRecord) PauseLearning() {
	lr.Status = LearningStatusPaused
}

// UpdateQuizScore 更新练习成绩
func (lr *LearningRecord) UpdateQuizScore(score float64) {
	lr.QuizAttempts++
	if lr.QuizBestScore == nil || score > *lr.QuizBestScore {
		lr.QuizBestScore = &score
	}
}

// GetFormattedStudyTime 获取格式化的学习时长
func (lr *LearningRecord) GetFormattedStudyTime() string {
	hours := lr.StudyDuration / 3600
	minutes := (lr.StudyDuration % 3600) / 60
	seconds := lr.StudyDuration % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// GetProgressText 获取进度文本
func (lr *LearningRecord) GetProgressText() string {
	return fmt.Sprintf("%.1f%%", lr.Progress*100) // 转换为百分比显示
}

// GetStatusText 获取状态文本
func (lr *LearningRecord) GetStatusText() string {
	switch lr.Status {
	case LearningStatusLearning:
		return "学习中"
	case LearningStatusCompleted:
		return "已完成"
	case LearningStatusPaused:
		return "已暂停"
	default:
		return "未知"
	}
}

// IsInProgress 检查是否在学习中
func (lr *LearningRecord) IsInProgress() bool {
	return lr.Status == LearningStatusLearning
}

// IsPaused 检查是否已暂停
func (lr *LearningRecord) IsPaused() bool {
	return lr.Status == LearningStatusPaused
}

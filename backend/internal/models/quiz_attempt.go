package models

import (
	"fmt"
	"time"
)

// AttemptStatus 答题状态
type AttemptStatus string

const (
	AttemptStatusInProgress AttemptStatus = "in_progress"
	AttemptStatusCompleted  AttemptStatus = "completed"
	AttemptStatusAbandoned  AttemptStatus = "abandoned"
)

// QuizAttempt 用户答题记录模型
type QuizAttempt struct {
	ID            uint          `json:"id" gorm:"primaryKey"`
	UserID        uint          `json:"user_id" gorm:"not null;index;comment:用户ID"`
	QuizID        uint          `json:"quiz_id" gorm:"not null;index;comment:练习ID"`
	CourseID      uint          `json:"course_id" gorm:"not null;index;comment:课件ID"`
	Status        AttemptStatus `json:"status" gorm:"type:enum('in_progress','completed','abandoned');default:'in_progress';comment:答题状态"`
	StartTime     time.Time     `json:"start_time" gorm:"comment:开始时间"`
	EndTime       *time.Time    `json:"end_time" gorm:"comment:结束时间"`
	Duration      int           `json:"duration" gorm:"default:0;comment:答题时长（秒）"`
	TotalScore    int           `json:"total_score" gorm:"default:0;comment:总分"`
	UserScore     int           `json:"user_score" gorm:"default:0;comment:用户得分"`
	Percentage    float64       `json:"percentage" gorm:"default:0;comment:得分率"`
	IsPassed      bool          `json:"is_passed" gorm:"default:false;comment:是否通过"`
	IsCompleted   bool          `json:"is_completed" gorm:"default:false;comment:是否完成"`
	CorrectCount  int           `json:"correct_count" gorm:"default:0;comment:正确题数"`
	WrongCount    int           `json:"wrong_count" gorm:"default:0;comment:错误题数"`
	SkippedCount  int           `json:"skipped_count" gorm:"default:0;comment:跳过题数"`
	QuestionCount int           `json:"question_count" gorm:"default:0;comment:题目数量"`
	SubmittedAt   *time.Time    `json:"submitted_at" gorm:"comment:提交时间"`

	// 关联关系
	UserAnswers []UserAnswer `json:"user_answers" gorm:"foreignKey:AttemptID"`
}

// TableName 指定表名
func (QuizAttempt) TableName() string {
	return "quiz_attempts"
}

// CalculateStats 计算统计信息
func (qa *QuizAttempt) CalculateStats() {
	if qa.TotalScore > 0 {
		qa.Percentage = float64(qa.UserScore) / float64(qa.TotalScore) * 100
	}

	// 计算答题时长
	if qa.EndTime != nil {
		qa.Duration = int(qa.EndTime.Sub(qa.StartTime).Seconds())
	}
}

// CheckPassed 检查是否通过（需要传入及格分数）
func (qa *QuizAttempt) CheckPassed(passScore int) {
	qa.IsPassed = qa.UserScore >= passScore
}

// Complete 完成答题
func (qa *QuizAttempt) Complete() {
	now := time.Now()
	qa.EndTime = &now
	qa.SubmittedAt = &now
	qa.IsCompleted = true
	qa.CalculateStats()
}

// GetFormattedDuration 获取格式化的答题时长
func (qa *QuizAttempt) GetFormattedDuration() string {
	if qa.Duration == 0 {
		return "0:00"
	}
	minutes := qa.Duration / 60
	seconds := qa.Duration % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// GetGrade 获取等级
func (qa *QuizAttempt) GetGrade() string {
	switch {
	case qa.Percentage >= 90:
		return "优秀"
	case qa.Percentage >= 80:
		return "良好"
	case qa.Percentage >= 70:
		return "中等"
	case qa.Percentage >= 60:
		return "及格"
	default:
		return "不及格"
	}
}

// UserAnswer 用户答案模型
type UserAnswer struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null;index;comment:用户ID"`
	QuestionID uint      `json:"question_id" gorm:"not null;index;comment:题目ID"`
	AttemptID  uint      `json:"attempt_id" gorm:"not null;index;comment:答题记录ID"`
	UserAnswer string    `json:"user_answer" gorm:"type:text;comment:用户答案"`
	IsCorrect  bool      `json:"is_correct" gorm:"default:false;comment:是否正确"`
	Points     int       `json:"points" gorm:"default:0;comment:得分"`
	Duration   int       `json:"duration" gorm:"default:0;comment:答题时长（秒）"`
	AnsweredAt time.Time `json:"answered_at" gorm:"comment:回答时间"`
}

// TableName 指定表名
func (UserAnswer) TableName() string {
	return "user_answers"
}

// CheckAnswer 检查答案是否正确
func (ua *UserAnswer) CheckAnswer(correctAnswer string, points int) {
	ua.IsCorrect = ua.UserAnswer == correctAnswer
	if ua.IsCorrect {
		ua.Points = points
	} else {
		ua.Points = 0
	}
}

// GetFormattedDuration 获取格式化的答题时长
func (ua *UserAnswer) GetFormattedDuration() string {
	if ua.Duration == 0 {
		return "0:00"
	}
	minutes := ua.Duration / 60
	seconds := ua.Duration % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

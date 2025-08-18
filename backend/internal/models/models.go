package models

import (
	"time"

	"gorm.io/gorm"
)

// AllModels 返回所有模型的切片，用于数据库迁移
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Course{},
		&Slide{},
		&Quiz{},
		&Question{},
		&QuizAttempt{},
		&UserAnswer{},
		&LearningRecord{},
		&ShareRecord{},
		&ShareLike{},
		&UsageRecord{},
		&PurchaseRecord{},
		&AudioMetadata{},
		&SystemConfig{},
		&OperationLog{},
		&AIAssistantChat{},
		&AIAnalysisRecord{}, // 🆕 新增AI分析记录模型
	}
}

// AutoMigrate 执行自动迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// 常用的预加载关系
const (
	// 课件相关预加载
	PreloadCourseSlides  = "Slides"
	PreloadCourseQuizzes = "Quizzes"
	PreloadCourseUser    = "User"

	// 用户相关预加载
	PreloadUserCourses = "Courses"
	PreloadUserRecords = "LearningRecords"

	// 练习相关预加载
	PreloadQuizQuestions = "Questions"
	PreloadQuizAttempts  = "Attempts"

	// 学习记录相关预加载
	PreloadLearningCourse = "Course"
	PreloadLearningUser   = "User"
)

// 数据库表名常量
const (
	TableUsers            = "users"
	TableCourses          = "courses"
	TableSlides           = "slides"
	TableQuizzes          = "quizzes"
	TableQuestions        = "questions"
	TableQuizAttempts     = "quiz_attempts"
	TableUserAnswers      = "user_answers"
	TableLearningRecords  = "learning_records"
	TableShareRecords     = "share_records"
	TableShareLikes       = "share_likes"
	TableUsageRecords     = "usage_records"
	TablePurchaseRecords  = "purchase_records"
	TableAudioMetadata    = "audio_metadata"
	TableSystemConfigs    = "system_configs"
	TableOperationLogs    = "operation_logs"
	TableAIAssistantChats = "ai_assistant_chats"
)

// AI助手对话记录
type AIAssistantChat struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Message   string    `json:"message" gorm:"type:text;not null"`
	Response  string    `json:"response" gorm:"type:text;not null"`
	Context   string    `json:"context,omitempty" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

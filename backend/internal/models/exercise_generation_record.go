package models

import (
	"time"
)

// ExerciseGenerationRecord 练习生成记录模型
type ExerciseGenerationRecord struct {
	ID                     uint      `json:"id" gorm:"primarykey"`
	UserID                 uint      `json:"user_id" gorm:"not null;index:idx_user_id"`
	CourseID               uint      `json:"course_id" gorm:"not null;index:idx_course_id"`
	QuizID                 *uint     `json:"quiz_id,omitempty" gorm:"index:idx_quiz_id"`
	WorkflowID             string    `json:"workflow_id" gorm:"not null;index:idx_workflow_id;size:100"`
	SourceURL              string    `json:"source_url" gorm:"not null;type:text"`
	GenerationStatus       string    `json:"generation_status" gorm:"type:enum('pending','processing','completed','failed');default:pending;index:idx_generation_status"`
	RequestData            *string   `json:"request_data,omitempty" gorm:"type:json"`
	ResponseData           *string   `json:"response_data,omitempty" gorm:"type:json"`
	ErrorMessage           *string   `json:"error_message,omitempty" gorm:"type:text"`
	ProcessingDuration     int       `json:"processing_duration" gorm:"default:0"`
	TotalQuestions         int       `json:"total_questions" gorm:"default:0"`
	DifficultyDistribution *string   `json:"difficulty_distribution,omitempty" gorm:"type:json"`
	CreatedAt              time.Time `json:"created_at" gorm:"index:idx_created_at"`
	UpdatedAt              time.Time `json:"updated_at"`

	// 关联关系
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Course Course `json:"course,omitempty" gorm:"foreignKey:CourseID"`
	Quiz   *Quiz  `json:"quiz,omitempty" gorm:"foreignKey:QuizID"`
}

// TableName 指定表名
func (ExerciseGenerationRecord) TableName() string {
	return "exercise_generation_records"
}

// GenerationStatus 枚举常量
const (
	GenerationStatusPending    = "pending"
	GenerationStatusProcessing = "processing"
	GenerationStatusCompleted  = "completed"
	GenerationStatusFailed     = "failed"
)

// WorkflowRequest 工作流请求数据结构
type WorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Parameters map[string]interface{} `json:"parameters"`
}

// WorkflowResponse 工作流响应数据结构
type WorkflowResponse struct {
	ResponseType string `json:"response_type"`
	Structure    struct {
		CourseName             string `json:"course_name"`
		TotalQuestions         int    `json:"total_questions"`
		DifficultyDistribution struct {
			Easy   int `json:"easy"`
			Medium int `json:"medium"`
			Hard   int `json:"hard"`
		} `json:"difficulty_distribution"`
		Questions []WorkflowQuestion `json:"questions"`
	} `json:"structure"`
}

// WorkflowQuestion 工作流返回的题目结构
type WorkflowQuestion struct {
	ID            int                    `json:"id"`
	Question      string                 `json:"question"`
	Options       map[string]interface{} `json:"options"`
	CorrectAnswer string                 `json:"correct_answer"`
	Difficulty    string                 `json:"difficulty"`
}

// GenerationRequest API请求结构
type GenerationRequest struct {
	CourseID      uint   `json:"course_id" binding:"required"`
	QuizTitle     string `json:"quiz_title,omitempty"`
	Difficulty    string `json:"difficulty,omitempty"`
	QuestionCount int    `json:"question_count,omitempty"`
}

// GenerationResponse API响应结构
type GenerationResponse struct {
	QuizID         uint   `json:"quiz_id"`
	TotalQuestions int    `json:"total_questions"`
	GenerationID   uint   `json:"generation_id"`
	Status         string `json:"status"`
	Message        string `json:"message"`
}

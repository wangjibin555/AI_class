package models

import (
	"time"
)

// AIAnalysisRecord AI分析记录
type AIAnalysisRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	URL          string    `gorm:"type:text;not null" json:"url"`
	AnalysisType string    `gorm:"type:varchar(50);not null;default:'comprehensive'" json:"analysis_type"`
	EngineType   string    `gorm:"type:varchar(20);not null;default:'dashscope'" json:"engine_type"` // dashscope, coze, hybrid
	Result       string    `gorm:"type:longtext" json:"result"`                                      // JSON格式的分析结果
	Status       string    `gorm:"type:varchar(20);default:'completed'" json:"status"`
	ErrorMessage string    `gorm:"type:text" json:"error_message,omitempty"`
	ProcessTime  int       `gorm:"type:int;default:0" json:"process_time"` // 处理时间（毫秒）
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (AIAnalysisRecord) TableName() string {
	return "ai_analysis_records"
}

// AIAnalysisResponse AI分析响应结构
type AIAnalysisResponse struct {
	URL               string             `json:"url"`
	Title             string             `json:"title"`
	Summary           string             `json:"summary"`
	KeyPoints         []string           `json:"key_points"`
	TechnicalConcepts []TechnicalConcept `json:"technical_concepts"`
	StructuredContent *StructuredContent `json:"structured_content"`
	Metadata          *AnalysisMetadata  `json:"metadata"`
}

// TechnicalConcept 技术概念
type TechnicalConcept struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Importance  int    `json:"importance"` // 1-10
}

// StructuredContent 结构化内容
type StructuredContent struct {
	Introduction  string           `json:"introduction"`
	MainSections  []ContentSection `json:"main_sections"`
	Conclusion    string           `json:"conclusion"`
	Examples      []CodeExample    `json:"examples"`
	BestPractices []string         `json:"best_practices"`
}

// ContentSection 内容章节
type ContentSection struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	SubSections []string `json:"sub_sections"`
	KeyPoints   []string `json:"key_points"`
}

// CodeExample 代码示例
type CodeExample struct {
	Title       string `json:"title"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	Description string `json:"description"`
}

// AnalysisMetadata 分析元数据
type AnalysisMetadata struct {
	AnalysisTime    time.Time `json:"analysis_time"`
	ContentType     string    `json:"content_type"`
	DifficultyLevel string    `json:"difficulty_level"`
	EstimatedTime   int       `json:"estimated_time"` // 预估学习时间（分钟）
	Tags            []string  `json:"tags"`
	EngineUsed      string    `json:"engine_used"`
}

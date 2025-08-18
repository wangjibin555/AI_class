package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CourseStatus 课件状态
type CourseStatus string

const (
	CourseStatusGenerating CourseStatus = "generating"
	CourseStatusCompleted  CourseStatus = "completed"
	CourseStatusFailed     CourseStatus = "failed"
)

// SourceType 来源类型
type SourceType string

const (
	SourceTypeURL      SourceType = "url"
	SourceTypeDocument SourceType = "document"
	SourceTypeText     SourceType = "text"
)

// CourseCategory 课件分类
type CourseCategory string

const (
	CategoryGeneral     CourseCategory = "general"
	CategoryProgramming CourseCategory = "programming"
	CategoryDatabase    CourseCategory = "database"
	CategoryFrontend    CourseCategory = "frontend"
	CategoryAI          CourseCategory = "ai"
	CategoryDevOps      CourseCategory = "devops"
)

// Tags JSON类型字段
type Tags []string

// Scan 实现 sql.Scanner 接口
func (t *Tags) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}

	switch s := value.(type) {
	case []byte:
		return json.Unmarshal(s, t)
	case string:
		return json.Unmarshal([]byte(s), t)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (t Tags) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// GenerationParams 生成参数
type GenerationParams map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (gp *GenerationParams) Scan(value interface{}) error {
	if value == nil {
		*gp = nil
		return nil
	}

	switch s := value.(type) {
	case []byte:
		return json.Unmarshal(s, gp)
	case string:
		return json.Unmarshal([]byte(s), gp)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (gp GenerationParams) Value() (driver.Value, error) {
	if gp == nil {
		return nil, nil
	}
	return json.Marshal(gp)
}

// Course 课件模型
type Course struct {
	ID               uint             `json:"id" gorm:"primaryKey"`
	UserID           uint             `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	Title            string           `json:"title" gorm:"size:200;not null;comment:课件标题"`
	Description      string           `json:"description" gorm:"type:text;comment:课件描述"`
	Category         CourseCategory   `json:"category" gorm:"size:50;default:'general';comment:课件分类"`
	Tags             Tags             `json:"tags" gorm:"type:json;comment:课件标签"`
	SourceType       SourceType       `json:"source_type" gorm:"type:enum('url','document','text');not null;comment:来源类型"`
	SourceContent    string           `json:"source_content" gorm:"type:text;not null;comment:原始内容"`
	SourceURL        string           `json:"source_url" gorm:"size:1000;comment:原始URL"`
	FilePath         string           `json:"file_path" gorm:"size:500;comment:上传文件路径"`
	FileSize         int64            `json:"file_size" gorm:"comment:文件大小（字节）"`
	ThumbnailURL     string           `json:"thumbnail_url" gorm:"size:500;comment:缩略图URL"`
	Status           CourseStatus     `json:"status" gorm:"type:enum('generating','completed','failed');default:'generating';comment:生成状态"`
	ErrorMessage     string           `json:"error_message" gorm:"type:text;comment:错误信息"`
	SlidesCount      int              `json:"slides_count" gorm:"default:0;comment:幻灯片数量"`
	Duration         int              `json:"duration" gorm:"default:0;comment:课件总时长（秒）"`
	ViewCount        int              `json:"view_count" gorm:"default:0;comment:观看次数"`
	LikeCount        int              `json:"like_count" gorm:"default:0;comment:点赞数"`
	ShareCount       int              `json:"share_count" gorm:"default:0;comment:分享次数"`
	IsPublic         bool             `json:"is_public" gorm:"default:false;comment:是否公开"`
	VoiceType        string           `json:"voice_type" gorm:"size:50;default:'zhixiaobai';comment:语音类型"`
	PPTFilePath      string           `json:"ppt_file_path" gorm:"size:500;comment:PPT文件路径"`
	GenerationParams GenerationParams `json:"generation_params" gorm:"type:json;comment:生成参数"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`

	// 关联字段
	Slides []Slide `json:"slides" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;comment:幻灯片列表"`
}

// TableName 指定表名
func (Course) TableName() string {
	return "courses"
}

// IsCompleted 检查课件是否已完成
func (c *Course) IsCompleted() bool {
	return c.Status == CourseStatusCompleted
}

// IsFailed 检查课件是否生成失败
func (c *Course) IsFailed() bool {
	return c.Status == CourseStatusFailed
}

// IsGenerating 检查课件是否正在生成中
func (c *Course) IsGenerating() bool {
	return c.Status == CourseStatusGenerating
}

// GetFormattedDuration 获取格式化的时长
func (c *Course) GetFormattedDuration() string {
	minutes := c.Duration / 60
	seconds := c.Duration % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// IncrementViewCount 增加观看次数
func (c *Course) IncrementViewCount() {
	c.ViewCount++
}

// IncrementLikeCount 增加点赞数
func (c *Course) IncrementLikeCount() {
	c.LikeCount++
}

// IncrementShareCount 增加分享次数
func (c *Course) IncrementShareCount() {
	c.ShareCount++
}

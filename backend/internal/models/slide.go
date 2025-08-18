package models

import (
	"time"
)

// LayoutType 布局类型
type LayoutType string

const (
	LayoutTypeContent    LayoutType = "content"
	LayoutTypeTitleSlide LayoutType = "title_slide"
	LayoutTypeTwoColumn  LayoutType = "two_column"
	LayoutTypeBulletList LayoutType = "bullet_list"
	LayoutTypeImageText  LayoutType = "image_text"
)

// Slide 幻灯片模型
type Slide struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	CourseID     uint       `json:"course_id" gorm:"not null;index"`
	SlideNumber  int        `json:"slide_number" gorm:"not null"`
	Title        string     `json:"title" gorm:"type:text;not null"`
	Content      string     `json:"content" gorm:"type:longtext"`
	Keywords     Keywords   `json:"keywords" gorm:"type:json"`
	Notes        string     `json:"notes" gorm:"type:text"`
	SpeakerNotes string     `json:"speaker_notes" gorm:"type:text"` // 演讲备注
	Duration     int        `json:"duration" gorm:"default:30"`     // 预计播放时长（秒）
	AudioURL     string     `json:"audio_url" gorm:"type:varchar(500)"`
	ImageURL     string     `json:"image_url" gorm:"type:varchar(500)"`
	LayoutType   LayoutType `json:"layout_type" gorm:"type:varchar(50);default:'content'"` // 布局类型
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联关系
	Course Course `json:"course,omitempty" gorm:"foreignKey:CourseID"`
}

// TableName 指定表名
func (Slide) TableName() string {
	return "slides"
}

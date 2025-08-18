package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ShareRecord 分享记录模型
type ShareRecord struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	UserID        uint       `json:"user_id" gorm:"not null;index;comment:分享用户ID"`
	CourseID      uint       `json:"course_id" gorm:"not null;index;comment:课件ID"`
	ShareCode     string     `json:"share_code" gorm:"uniqueIndex;size:32;not null;comment:分享码"`
	Title         string     `json:"title" gorm:"size:200;comment:分享标题"`
	Description   string     `json:"description" gorm:"size:500;comment:分享描述"`
	CoverImage    string     `json:"cover_image" gorm:"size:500;comment:分享封面图"`
	ViewCount     int        `json:"view_count" gorm:"default:0;comment:查看次数"`
	LikeCount     int        `json:"like_count" gorm:"default:0;comment:点赞次数"`
	DownloadCount int        `json:"download_count" gorm:"default:0;comment:下载次数"`
	AllowDownload bool       `json:"allow_download" gorm:"default:true;comment:是否允许下载"`
	ExpiresAt     *time.Time `json:"expires_at" gorm:"comment:过期时间"`
	IsActive      bool       `json:"is_active" gorm:"default:true;comment:是否有效"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (ShareRecord) TableName() string {
	return "share_records"
}

// GenerateShareCode 生成分享码
func (sr *ShareRecord) GenerateShareCode() error {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return err
	}
	sr.ShareCode = hex.EncodeToString(bytes)
	return nil
}

// IsExpired 检查是否已过期
func (sr *ShareRecord) IsExpired() bool {
	if sr.ExpiresAt == nil {
		return false
	}
	return sr.ExpiresAt.Before(time.Now())
}

// IncrementViewCount 增加查看次数
func (sr *ShareRecord) IncrementViewCount() {
	sr.ViewCount++
}

// IncrementLikeCount 增加点赞次数
func (sr *ShareRecord) IncrementLikeCount() {
	sr.LikeCount++
}

// IncrementDownloadCount 增加下载次数
func (sr *ShareRecord) IncrementDownloadCount() {
	if sr.AllowDownload {
		sr.DownloadCount++
	}
}

// SetExpiration 设置过期时间
func (sr *ShareRecord) SetExpiration(days int) {
	if days > 0 {
		expireTime := time.Now().AddDate(0, 0, days)
		sr.ExpiresAt = &expireTime
	}
}

// Deactivate 停用分享
func (sr *ShareRecord) Deactivate() {
	sr.IsActive = false
}

// GetShareURL 获取分享链接
func (sr *ShareRecord) GetShareURL(baseURL string) string {
	return fmt.Sprintf("%s/share/%s", baseURL, sr.ShareCode)
}

// ShareLike 分享点赞模型
type ShareLike struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ShareID   uint      `json:"share_id" gorm:"not null;index;comment:分享记录ID"`
	UserID    uint      `json:"user_id" gorm:"not null;index;comment:点赞用户ID"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (ShareLike) TableName() string {
	return "share_likes"
}

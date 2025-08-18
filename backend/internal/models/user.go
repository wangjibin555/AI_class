package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID                  uint       `json:"id" gorm:"primaryKey"`
	OpenID              string     `json:"openid" gorm:"uniqueIndex;size:100;not null;comment:微信用户唯一标识"`
	Nickname            string     `json:"nickname" gorm:"size:100;comment:用户昵称"`
	AvatarURL           string     `json:"avatar_url" gorm:"size:500;comment:头像URL"`
	Phone               string     `json:"phone" gorm:"size:20;comment:手机号"`
	Email               string     `json:"email" gorm:"size:100;comment:邮箱"`
	VipLevel            int        `json:"vip_level" gorm:"default:0;comment:VIP等级：0-普通用户，1-月度会员，2-年度会员"`
	VipExpiredAt        *time.Time `json:"vip_expired_at" gorm:"comment:VIP过期时间"`
	Credits             int        `json:"credits" gorm:"default:10;comment:用户积分/次数"`
	TotalCoursesCreated int        `json:"total_courses_created" gorm:"default:0;comment:创建课件总数"`
	TotalStudyTime      int        `json:"total_study_time" gorm:"default:0;comment:总学习时长（秒）"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// IsVIP 检查是否为VIP用户
func (u *User) IsVIP() bool {
	if u.VipLevel == 0 {
		return false
	}
	if u.VipExpiredAt == nil {
		return false
	}
	return u.VipExpiredAt.After(time.Now())
}

// GetVipLevelName 获取VIP等级名称
func (u *User) GetVipLevelName() string {
	switch u.VipLevel {
	case 1:
		return "月度会员"
	case 2:
		return "年度会员"
	default:
		return "普通用户"
	}
}

// CanCreateCourse 检查用户是否可以创建课件
func (u *User) CanCreateCourse() bool {
	return u.Credits > 0 || u.IsVIP()
}

// ConsumeCredits 消耗积分
func (u *User) ConsumeCredits(amount int) bool {
	if u.Credits >= amount {
		u.Credits -= amount
		return true
	}
	return false
}

// GetDailyQuota 获取用户每日配额
func (u *User) GetDailyQuota() int {
	switch u.VipLevel {
	case 1:
		return 50 // 月度VIP
	case 2:
		return 100 // 年度VIP
	default:
		return 3 // 普通用户
	}
}

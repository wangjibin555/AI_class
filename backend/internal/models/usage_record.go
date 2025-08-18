package models

import (
	"fmt"
	"time"
)

// UsageType 使用类型
type UsageType string

const (
	UsageTypeCourseGeneration UsageType = "course_generation"
	UsageTypeTTSGeneration    UsageType = "tts_generation"
	UsageTypeQuizGeneration   UsageType = "quiz_generation"
)

// UsageRecord 使用记录模型
type UsageRecord struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"not null;index;comment:用户ID"`
	Type            UsageType `json:"type" gorm:"type:enum('course_generation','tts_generation','quiz_generation');not null;comment:使用类型"`
	ResourceID      *uint     `json:"resource_id" gorm:"comment:资源ID（课件ID等）"`
	ConsumedCredits int       `json:"consumed_credits" gorm:"default:1;comment:消耗积分"`
	IPAddress       string    `json:"ip_address" gorm:"size:45;comment:IP地址"`
	UserAgent       string    `json:"user_agent" gorm:"size:1000;comment:用户代理"`
	ConsumedAt      time.Time `json:"consumed_at" gorm:"comment:消费时间"`
}

// TableName 指定表名
func (UsageRecord) TableName() string {
	return "usage_records"
}

// GetTypeText 获取类型文本
func (ur *UsageRecord) GetTypeText() string {
	switch ur.Type {
	case UsageTypeCourseGeneration:
		return "课件生成"
	case UsageTypeTTSGeneration:
		return "语音合成"
	case UsageTypeQuizGeneration:
		return "练习生成"
	default:
		return "未知"
	}
}

// PaymentMethod 支付方式
type PaymentMethod string

const (
	PaymentMethodWechat PaymentMethod = "wechat"
	PaymentMethodAlipay PaymentMethod = "alipay"
)

// PaymentStatus 支付状态
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PurchaseRecord 购买记录模型
type PurchaseRecord struct {
	ID            uint          `json:"id" gorm:"primaryKey"`
	UserID        uint          `json:"user_id" gorm:"not null;index;comment:用户ID"`
	OrderNo       string        `json:"order_no" gorm:"uniqueIndex;size:64;not null;comment:订单号"`
	PlanID        int           `json:"plan_id" gorm:"not null;comment:套餐ID"`
	PlanName      string        `json:"plan_name" gorm:"size:100;not null;comment:套餐名称"`
	Amount        int           `json:"amount" gorm:"not null;comment:金额（分）"`
	PaymentMethod PaymentMethod `json:"payment_method" gorm:"type:enum('wechat','alipay');default:'wechat';comment:支付方式"`
	PaymentID     string        `json:"payment_id" gorm:"size:100;comment:第三方支付ID"`
	Status        PaymentStatus `json:"status" gorm:"type:enum('pending','completed','failed','refunded');default:'pending';comment:支付状态"`
	PaidAt        *time.Time    `json:"paid_at" gorm:"comment:支付时间"`
	RefundedAt    *time.Time    `json:"refunded_at" gorm:"comment:退款时间"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// TableName 指定表名
func (PurchaseRecord) TableName() string {
	return "purchase_records"
}

// GenerateOrderNo 生成订单号
func (pr *PurchaseRecord) GenerateOrderNo() {
	pr.OrderNo = fmt.Sprintf("ORDER_%d_%d", time.Now().Unix(), pr.UserID)
}

// MarkAsPaid 标记为已支付
func (pr *PurchaseRecord) MarkAsPaid(paymentID string) {
	pr.Status = PaymentStatusCompleted
	pr.PaymentID = paymentID
	now := time.Now()
	pr.PaidAt = &now
}

// MarkAsFailed 标记为失败
func (pr *PurchaseRecord) MarkAsFailed() {
	pr.Status = PaymentStatusFailed
}

// MarkAsRefunded 标记为已退款
func (pr *PurchaseRecord) MarkAsRefunded() {
	pr.Status = PaymentStatusRefunded
	now := time.Now()
	pr.RefundedAt = &now
}

// GetStatusText 获取状态文本
func (pr *PurchaseRecord) GetStatusText() string {
	switch pr.Status {
	case PaymentStatusPending:
		return "待支付"
	case PaymentStatusCompleted:
		return "已支付"
	case PaymentStatusFailed:
		return "支付失败"
	case PaymentStatusRefunded:
		return "已退款"
	default:
		return "未知"
	}
}

// GetPaymentMethodText 获取支付方式文本
func (pr *PurchaseRecord) GetPaymentMethodText() string {
	switch pr.PaymentMethod {
	case PaymentMethodWechat:
		return "微信支付"
	case PaymentMethodAlipay:
		return "支付宝"
	default:
		return "未知"
	}
}

// GetAmountYuan 获取金额（元）
func (pr *PurchaseRecord) GetAmountYuan() float64 {
	return float64(pr.Amount) / 100.0
}

// IsCompleted 检查是否已完成支付
func (pr *PurchaseRecord) IsCompleted() bool {
	return pr.Status == PaymentStatusCompleted
}

// IsPending 检查是否待支付
func (pr *PurchaseRecord) IsPending() bool {
	return pr.Status == PaymentStatusPending
}

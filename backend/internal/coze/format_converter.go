package coze

import (
	"sync"
)

// FormatConverter 格式转换器
type FormatConverter struct {
	metrics *ConversionMetrics
}

// ConversionMetrics 转换指标
type ConversionMetrics struct {
	mu                 sync.RWMutex
	TotalConversions   int64 `json:"total_conversions"`
	SuccessConversions int64 `json:"success_conversions"`
	FailedConversions  int64 `json:"failed_conversions"`
}

// NewFormatConverter 创建格式转换器
func NewFormatConverter() *FormatConverter {
	return &FormatConverter{
		metrics: &ConversionMetrics{},
	}
}

// GetMetrics 获取转换指标
func (c *FormatConverter) GetMetrics() *ConversionMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	return &ConversionMetrics{
		TotalConversions:   c.metrics.TotalConversions,
		SuccessConversions: c.metrics.SuccessConversions,
		FailedConversions:  c.metrics.FailedConversions,
	}
}

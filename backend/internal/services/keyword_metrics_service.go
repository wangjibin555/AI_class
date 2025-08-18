package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// KeywordMetricsService 关键词指标监控服务
type KeywordMetricsService struct {
	db                    *gorm.DB
	metrics               *KeywordMetricsData
	mu                    sync.RWMutex
	lastReportTime        time.Time
	reportInterval        time.Duration
	qualityThresholds     *QualityThresholds
	enableDetailedLogging bool
}

// KeywordMetricsData 关键词指标数据
type KeywordMetricsData struct {
	// 基础统计
	TotalExtractions      int64 `json:"total_extractions"`
	SuccessfulExtractions int64 `json:"successful_extractions"`
	FailedExtractions     int64 `json:"failed_extractions"`

	// AI相关统计
	AIExtractions      int64 `json:"ai_extractions"`
	AISuccessCount     int64 `json:"ai_success_count"`
	BasicFallbackCount int64 `json:"basic_fallback_count"`

	// 质量指标
	AverageKeywordCount float64 `json:"average_keyword_count"`
	EmptyKeywordCount   int64   `json:"empty_keyword_count"`
	QualityScore        float64 `json:"quality_score"`

	// 性能指标
	AverageProcessTime float64 `json:"average_process_time_ms"`
	MaxProcessTime     float64 `json:"max_process_time_ms"`
	MinProcessTime     float64 `json:"min_process_time_ms"`

	// 内容类型统计
	TechnicalContent int64 `json:"technical_content"`
	AcademicContent  int64 `json:"academic_content"`
	GeneralContent   int64 `json:"general_content"`

	// 错误统计
	TopErrors []ErrorInfo `json:"top_errors"`
	ErrorRate float64     `json:"error_rate"`

	// 时间相关
	LastUpdated     time.Time `json:"last_updated"`
	ReportingPeriod string    `json:"reporting_period"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	ErrorMessage string    `json:"error_message"`
	Count        int64     `json:"count"`
	LastOccurred time.Time `json:"last_occurred"`
}

// QualityThresholds 质量阈值配置
type QualityThresholds struct {
	MinKeywordCount  int     `json:"min_keyword_count"`   // 最少关键词数
	MaxKeywordCount  int     `json:"max_keyword_count"`   // 最多关键词数
	MinProcessTimeMs float64 `json:"min_process_time_ms"` // 最短处理时间
	MaxProcessTimeMs float64 `json:"max_process_time_ms"` // 最长处理时间
	MinQualityScore  float64 `json:"min_quality_score"`   // 最低质量分数
	MaxErrorRate     float64 `json:"max_error_rate"`      // 最大错误率
}

// ExtractionEvent 关键词提取事件
type ExtractionEvent struct {
	ContentLength int           `json:"content_length"`
	KeywordCount  int           `json:"keyword_count"`
	ProcessTime   time.Duration `json:"process_time"`
	ContentType   string        `json:"content_type"`
	UseAI         bool          `json:"use_ai"`
	Success       bool          `json:"success"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	Keywords      []string      `json:"keywords"`
	QualityScore  float64       `json:"quality_score"`
	Timestamp     time.Time     `json:"timestamp"`
}

// NewKeywordMetricsService 创建关键词指标监控服务
func NewKeywordMetricsService(db *gorm.DB) *KeywordMetricsService {
	return &KeywordMetricsService{
		db:             db,
		metrics:        &KeywordMetricsData{},
		lastReportTime: time.Now(),
		reportInterval: 5 * time.Minute, // 每5分钟报告一次
		qualityThresholds: &QualityThresholds{
			MinKeywordCount:  2,
			MaxKeywordCount:  15,
			MinProcessTimeMs: 10,
			MaxProcessTimeMs: 30000, // 30秒
			MinQualityScore:  0.6,
			MaxErrorRate:     0.1, // 10%
		},
		enableDetailedLogging: true,
	}
}

// RecordExtraction 记录关键词提取事件
func (m *KeywordMetricsService) RecordExtraction(event *ExtractionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新基础统计
	m.metrics.TotalExtractions++
	if event.Success {
		m.metrics.SuccessfulExtractions++
	} else {
		m.metrics.FailedExtractions++
		m.recordError(event.ErrorMessage)
	}

	// 更新AI统计
	if event.UseAI {
		m.metrics.AIExtractions++
		if event.Success {
			m.metrics.AISuccessCount++
		} else {
			m.metrics.BasicFallbackCount++
		}
	}

	// 更新质量指标
	if event.Success {
		m.updateQualityMetrics(event)
	}

	// 更新性能指标
	m.updatePerformanceMetrics(event)

	// 更新内容类型统计
	m.updateContentTypeStats(event.ContentType)

	// 详细日志记录
	if m.enableDetailedLogging {
		m.logExtractionEvent(event)
	}

	// 质量检查和告警
	m.checkQualityThresholds(event)

	m.metrics.LastUpdated = time.Now()

	// 定期报告
	if time.Since(m.lastReportTime) >= m.reportInterval {
		m.generateReport()
		m.lastReportTime = time.Now()
	}
}

// updateQualityMetrics 更新质量指标
func (m *KeywordMetricsService) updateQualityMetrics(event *ExtractionEvent) {
	// 更新平均关键词数量
	totalKeywords := float64(m.metrics.SuccessfulExtractions-1)*m.metrics.AverageKeywordCount + float64(event.KeywordCount)
	m.metrics.AverageKeywordCount = totalKeywords / float64(m.metrics.SuccessfulExtractions)

	// 记录空关键词
	if event.KeywordCount == 0 {
		m.metrics.EmptyKeywordCount++
	}

	// 计算质量分数
	m.metrics.QualityScore = m.calculateQualityScore()
}

// updatePerformanceMetrics 更新性能指标
func (m *KeywordMetricsService) updatePerformanceMetrics(event *ExtractionEvent) {
	processTimeMs := float64(event.ProcessTime.Nanoseconds()) / 1e6

	// 更新平均处理时间
	totalTime := float64(m.metrics.TotalExtractions-1)*m.metrics.AverageProcessTime + processTimeMs
	m.metrics.AverageProcessTime = totalTime / float64(m.metrics.TotalExtractions)

	// 更新最大最小处理时间
	if m.metrics.MaxProcessTime == 0 || processTimeMs > m.metrics.MaxProcessTime {
		m.metrics.MaxProcessTime = processTimeMs
	}
	if m.metrics.MinProcessTime == 0 || processTimeMs < m.metrics.MinProcessTime {
		m.metrics.MinProcessTime = processTimeMs
	}
}

// updateContentTypeStats 更新内容类型统计
func (m *KeywordMetricsService) updateContentTypeStats(contentType string) {
	switch contentType {
	case "technical":
		m.metrics.TechnicalContent++
	case "academic":
		m.metrics.AcademicContent++
	default:
		m.metrics.GeneralContent++
	}
}

// recordError 记录错误信息
func (m *KeywordMetricsService) recordError(errorMsg string) {
	if errorMsg == "" {
		return
	}

	// 查找现有错误
	for i := range m.metrics.TopErrors {
		if m.metrics.TopErrors[i].ErrorMessage == errorMsg {
			m.metrics.TopErrors[i].Count++
			m.metrics.TopErrors[i].LastOccurred = time.Now()
			return
		}
	}

	// 添加新错误
	errorInfo := ErrorInfo{
		ErrorMessage: errorMsg,
		Count:        1,
		LastOccurred: time.Now(),
	}

	if len(m.metrics.TopErrors) < 10 {
		m.metrics.TopErrors = append(m.metrics.TopErrors, errorInfo)
	} else {
		// 替换最少发生的错误
		minIndex := 0
		for i := range m.metrics.TopErrors {
			if m.metrics.TopErrors[i].Count < m.metrics.TopErrors[minIndex].Count {
				minIndex = i
			}
		}
		m.metrics.TopErrors[minIndex] = errorInfo
	}

	// 更新错误率
	m.metrics.ErrorRate = float64(m.metrics.FailedExtractions) / float64(m.metrics.TotalExtractions)
}

// calculateQualityScore 计算质量分数
func (m *KeywordMetricsService) calculateQualityScore() float64 {
	if m.metrics.SuccessfulExtractions == 0 {
		return 0.0
	}

	// 基础成功率权重: 50%
	successRate := float64(m.metrics.SuccessfulExtractions) / float64(m.metrics.TotalExtractions)

	// AI成功率权重: 30%
	aiSuccessRate := 1.0
	if m.metrics.AIExtractions > 0 {
		aiSuccessRate = float64(m.metrics.AISuccessCount) / float64(m.metrics.AIExtractions)
	}

	// 关键词质量权重: 20%
	keywordQuality := 1.0
	if m.metrics.SuccessfulExtractions > 0 {
		emptyRate := float64(m.metrics.EmptyKeywordCount) / float64(m.metrics.SuccessfulExtractions)
		keywordQuality = 1.0 - emptyRate
	}

	return successRate*0.5 + aiSuccessRate*0.3 + keywordQuality*0.2
}

// logExtractionEvent 记录详细的提取事件日志
func (m *KeywordMetricsService) logExtractionEvent(event *ExtractionEvent) {
	logLevel := "INFO"
	if !event.Success {
		logLevel = "ERROR"
	} else if event.KeywordCount == 0 {
		logLevel = "WARN"
	}

	log.Printf("[%s] 关键词提取 - 内容长度:%d, 关键词数:%d, 处理时间:%.2fms, 类型:%s, AI:%t, 成功:%t, 质量分:%.2f",
		logLevel,
		event.ContentLength,
		event.KeywordCount,
		float64(event.ProcessTime.Nanoseconds())/1e6,
		event.ContentType,
		event.UseAI,
		event.Success,
		event.QualityScore,
	)

	if !event.Success {
		log.Printf("[ERROR] 关键词提取失败: %s", event.ErrorMessage)
	}

	if event.KeywordCount == 0 && event.Success {
		log.Printf("[WARN] 关键词提取为空，内容长度: %d", event.ContentLength)
	}
}

// checkQualityThresholds 检查质量阈值并告警
func (m *KeywordMetricsService) checkQualityThresholds(event *ExtractionEvent) {
	warnings := []string{}

	// 检查关键词数量
	if event.Success && event.KeywordCount < m.qualityThresholds.MinKeywordCount {
		warnings = append(warnings, fmt.Sprintf("关键词数量过少: %d < %d", event.KeywordCount, m.qualityThresholds.MinKeywordCount))
	}
	if event.Success && event.KeywordCount > m.qualityThresholds.MaxKeywordCount {
		warnings = append(warnings, fmt.Sprintf("关键词数量过多: %d > %d", event.KeywordCount, m.qualityThresholds.MaxKeywordCount))
	}

	// 检查处理时间
	processTimeMs := float64(event.ProcessTime.Nanoseconds()) / 1e6
	if processTimeMs > m.qualityThresholds.MaxProcessTimeMs {
		warnings = append(warnings, fmt.Sprintf("处理时间过长: %.2fms > %.2fms", processTimeMs, m.qualityThresholds.MaxProcessTimeMs))
	}

	// 检查质量分数
	if event.QualityScore < m.qualityThresholds.MinQualityScore {
		warnings = append(warnings, fmt.Sprintf("质量分数过低: %.2f < %.2f", event.QualityScore, m.qualityThresholds.MinQualityScore))
	}

	// 检查错误率
	if m.metrics.ErrorRate > m.qualityThresholds.MaxErrorRate {
		warnings = append(warnings, fmt.Sprintf("错误率过高: %.2f%% > %.2f%%", m.metrics.ErrorRate*100, m.qualityThresholds.MaxErrorRate*100))
	}

	// 记录告警
	for _, warning := range warnings {
		log.Printf("[WARN] 关键词提取质量告警: %s", warning)
	}
}

// generateReport 生成定期报告
func (m *KeywordMetricsService) generateReport() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log.Printf("📊 关键词提取质量报告 [%s]", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("   总提取次数: %d", m.metrics.TotalExtractions)
	log.Printf("   成功次数: %d (%.2f%%)", m.metrics.SuccessfulExtractions,
		float64(m.metrics.SuccessfulExtractions)/float64(m.metrics.TotalExtractions)*100)
	log.Printf("   失败次数: %d (%.2f%%)", m.metrics.FailedExtractions,
		float64(m.metrics.FailedExtractions)/float64(m.metrics.TotalExtractions)*100)
	log.Printf("   AI提取次数: %d, 成功率: %.2f%%", m.metrics.AIExtractions,
		float64(m.metrics.AISuccessCount)/float64(m.metrics.AIExtractions)*100)
	log.Printf("   平均关键词数: %.2f", m.metrics.AverageKeywordCount)
	log.Printf("   空关键词数: %d", m.metrics.EmptyKeywordCount)
	log.Printf("   质量分数: %.2f", m.metrics.QualityScore)
	log.Printf("   平均处理时间: %.2fms", m.metrics.AverageProcessTime)
	log.Printf("   内容类型分布: 技术=%d, 学术=%d, 通用=%d",
		m.metrics.TechnicalContent, m.metrics.AcademicContent, m.metrics.GeneralContent)

	// 记录Top错误
	if len(m.metrics.TopErrors) > 0 {
		log.Printf("   常见错误:")
		for i, err := range m.metrics.TopErrors {
			if i >= 3 { // 只显示前3个
				break
			}
			log.Printf("     %d. %s (出现%d次)", i+1, err.ErrorMessage, err.Count)
		}
	}
}

// GetMetrics 获取当前指标
func (m *KeywordMetricsService) GetMetrics() *KeywordMetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 深拷贝避免并发问题
	metricsCopy := *m.metrics
	metricsCopy.TopErrors = make([]ErrorInfo, len(m.metrics.TopErrors))
	copy(metricsCopy.TopErrors, m.metrics.TopErrors)

	return &metricsCopy
}

// ResetMetrics 重置指标
func (m *KeywordMetricsService) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = &KeywordMetricsData{
		LastUpdated:     time.Now(),
		ReportingPeriod: "reset",
	}
	log.Printf("📊 关键词提取指标已重置")
}

// SaveMetricsToDatabase 将指标保存到数据库
func (m *KeywordMetricsService) SaveMetricsToDatabase() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 这里可以实现将指标保存到数据库的逻辑
	// 例如创建一个metrics表来存储历史数据

	return nil
}

// GetDatabaseKeywordStats 获取数据库中的关键词统计
func (m *KeywordMetricsService) GetDatabaseKeywordStats() (*DatabaseKeywordStats, error) {
	var stats DatabaseKeywordStats

	// 基础统计
	err := m.db.Raw(`
		SELECT 
			COUNT(*) as total_slides,
			COUNT(CASE WHEN JSON_LENGTH(keywords) > 0 THEN 1 END) as slides_with_keywords,
			COUNT(CASE WHEN JSON_LENGTH(keywords) = 0 THEN 1 END) as empty_keywords,
			COUNT(CASE WHEN keywords IS NULL THEN 1 END) as null_keywords,
			ROUND(AVG(JSON_LENGTH(keywords)), 2) as avg_keywords_per_slide
		FROM slides
	`).Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("获取数据库关键词统计失败: %w", err)
	}

	// 计算覆盖率
	if stats.TotalSlides > 0 {
		stats.KeywordCoverage = float64(stats.SlidesWithKeywords) / float64(stats.TotalSlides) * 100
	}

	return &stats, nil
}

// DatabaseKeywordStats 数据库关键词统计
type DatabaseKeywordStats struct {
	TotalSlides         int64   `gorm:"column:total_slides" json:"total_slides"`
	SlidesWithKeywords  int64   `gorm:"column:slides_with_keywords" json:"slides_with_keywords"`
	EmptyKeywords       int64   `gorm:"column:empty_keywords" json:"empty_keywords"`
	NullKeywords        int64   `gorm:"column:null_keywords" json:"null_keywords"`
	AvgKeywordsPerSlide float64 `gorm:"column:avg_keywords_per_slide" json:"avg_keywords_per_slide"`
	KeywordCoverage     float64 `json:"keyword_coverage"`
}

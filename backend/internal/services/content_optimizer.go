package services

import (
	"strings"
	"unicode/utf8"

	"ai-classroom/internal/models"
)

// ContentOptimizer 内容优化器
type ContentOptimizer struct {
	densityTarget    float64
	qualityThreshold float64
}

// FontConfig 字体配置
type FontConfig struct {
	FontSize   int     `json:"font_size"`
	LineHeight float64 `json:"line_height"`
	NeedSplit  bool    `json:"need_split"`
}

// NewContentOptimizer 创建内容优化器
func NewContentOptimizer() *ContentOptimizer {
	return &ContentOptimizer{
		densityTarget:    1.0,
		qualityThreshold: 0.8,
	}
}

// OptimizedSlideData 优化后的幻灯片数据
type OptimizedSlideData struct {
	Title        string   `json:"title"`
	Content      []string `json:"content"`
	CharCount    int      `json:"char_count"`
	FontSize     int      `json:"font_size"`
	LineHeight   float64  `json:"line_height"`
	NeedSplit    bool     `json:"need_split"`
	ContentClass string   `json:"content_class"`
}

// OptimizeSlide 优化幻灯片内容
func (co *ContentOptimizer) OptimizeSlide(slide *models.Slide) (*OptimizedSlideData, error) {
	charCount := utf8.RuneCountInString(slide.Content)
	fontConfig := co.calculateFontConfig(charCount)

	result := &OptimizedSlideData{
		Title:        slide.Title,
		Content:      co.parseContent(slide.Content),
		CharCount:    charCount,
		FontSize:     fontConfig.FontSize,
		LineHeight:   fontConfig.LineHeight,
		NeedSplit:    fontConfig.NeedSplit,
		ContentClass: co.getContentClass(charCount),
	}

	if result.NeedSplit {
		return co.splitSlide(result)
	}

	return result, nil
}

// calculateFontConfig 计算字体配置
func (co *ContentOptimizer) calculateFontConfig(charCount int) FontConfig {
	// 字符数量分级策略
	levels := []struct {
		max        int
		fontSize   int
		lineHeight float64
	}{
		{max: 100, fontSize: 24, lineHeight: 1.5},     // 简短内容
		{max: 200, fontSize: 20, lineHeight: 1.4},     // 中等内容
		{max: 350, fontSize: 18, lineHeight: 1.3},     // 较多内容
		{max: 500, fontSize: 16, lineHeight: 1.2},     // 大量内容
		{max: 800, fontSize: 14, lineHeight: 1.1},     // 超多内容
		{max: 9999999, fontSize: 12, lineHeight: 1.0}, // 极多内容（需分页）
	}

	for _, level := range levels {
		if charCount <= level.max {
			return FontConfig{
				FontSize:   level.fontSize,
				LineHeight: level.lineHeight,
				NeedSplit:  charCount > 500, // 超过500字需要考虑分页
			}
		}
	}

	// 默认返回最小字体
	return FontConfig{
		FontSize:   12,
		LineHeight: 1.0,
		NeedSplit:  true,
	}
}

// getContentClass 获取内容样式类名
func (co *ContentOptimizer) getContentClass(charCount int) string {
	switch {
	case charCount <= 100:
		return "content-short"
	case charCount <= 200:
		return "content-medium"
	case charCount <= 400:
		return "content-long"
	default:
		return "content-extra-long"
	}
}

// parseContent 解析内容为数组
func (co *ContentOptimizer) parseContent(content string) []string {
	lines := strings.Split(content, "\n")
	var items []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}

	return items
}

// splitSlide 分割幻灯片内容（超长内容处理）
func (co *ContentOptimizer) splitSlide(slide *OptimizedSlideData) (*OptimizedSlideData, error) {
	// 如果内容超过500字，建议分页，但暂时返回优化后的内容
	// 实际分页逻辑可以在后续Phase 3中实现

	// 先尝试简化内容
	optimizedContent := co.optimizeContentForDisplay(slide.Content)
	charCount := 0
	for _, item := range optimizedContent {
		charCount += utf8.RuneCountInString(item)
	}

	fontConfig := co.calculateFontConfig(charCount)

	result := &OptimizedSlideData{
		Title:        slide.Title,
		Content:      optimizedContent,
		CharCount:    charCount,
		FontSize:     fontConfig.FontSize,
		LineHeight:   fontConfig.LineHeight,
		NeedSplit:    charCount > 500,
		ContentClass: co.getContentClass(charCount),
	}

	return result, nil
}

// optimizeContentForDisplay 优化内容用于显示
func (co *ContentOptimizer) optimizeContentForDisplay(content []string) []string {
	var optimized []string

	for _, item := range content {
		// 处理过长的项目
		if utf8.RuneCountInString(item) > 100 {
			// 截取前80个字符并添加省略号
			runes := []rune(item)
			if len(runes) > 80 {
				item = string(runes[:80]) + "..."
			}
		}

		// 移除重复的分类和重要性标记（简化显示）
		item = co.cleanDisplayText(item)
		optimized = append(optimized, item)
	}

	return optimized
}

// cleanDisplayText 清理显示文本
func (co *ContentOptimizer) cleanDisplayText(text string) string {
	// 移除多余的重要性标记重复
	text = strings.ReplaceAll(text, "重要性：", "")
	text = strings.ReplaceAll(text, "分类：", "")

	// 移除多余的空格
	text = strings.TrimSpace(text)

	return text
}

// AnalyzeContentLength 分析内容长度并给出建议
func (co *ContentOptimizer) AnalyzeContentLength(content string) map[string]interface{} {
	charCount := utf8.RuneCountInString(content)
	wordCount := len(strings.Fields(content))
	fontConfig := co.calculateFontConfig(charCount)

	analysis := map[string]interface{}{
		"char_count":    charCount,
		"word_count":    wordCount,
		"font_size":     fontConfig.FontSize,
		"line_height":   fontConfig.LineHeight,
		"content_class": co.getContentClass(charCount),
		"need_split":    fontConfig.NeedSplit,
		"readability":   co.getReadabilityScore(charCount),
		"suggestions":   co.getSuggestions(charCount),
	}

	return analysis
}

// getReadabilityScore 获取可读性评分
func (co *ContentOptimizer) getReadabilityScore(charCount int) string {
	switch {
	case charCount <= 100:
		return "excellent"
	case charCount <= 200:
		return "good"
	case charCount <= 350:
		return "fair"
	case charCount <= 500:
		return "poor"
	default:
		return "very_poor"
	}
}

// getSuggestions 获取优化建议
func (co *ContentOptimizer) getSuggestions(charCount int) []string {
	var suggestions []string

	if charCount <= 100 {
		suggestions = append(suggestions, "内容长度适中，可读性优秀")
	} else if charCount <= 200 {
		suggestions = append(suggestions, "内容适中，建议保持当前长度")
	} else if charCount <= 350 {
		suggestions = append(suggestions, "内容较多，建议精简部分要点")
	} else if charCount <= 500 {
		suggestions = append(suggestions, "内容过多，强烈建议精简或分页")
	} else {
		suggestions = append(suggestions, "内容严重超标，必须分页或大幅精简")
		suggestions = append(suggestions, "考虑拆分为多个幻灯片")
		suggestions = append(suggestions, "移除非关键信息")
	}

	return suggestions
}

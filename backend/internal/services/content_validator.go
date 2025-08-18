package services

import (
	"strings"
)

// ContentValidator 内容验证器
type ContentValidator struct{}

// NewContentValidator 创建内容验证器
func NewContentValidator() *ContentValidator {
	return &ContentValidator{}
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid      bool     `json:"is_valid"`
	Issues       []string `json:"issues"`
	TemplateRate float64  `json:"template_rate"`
}

// ValidatePPTContent 验证PPT内容是否为模板内容
func (v *ContentValidator) ValidatePPTContent(slides []EnhancedSlideContent) ValidationResult {
	result := ValidationResult{
		IsValid:      true,
		Issues:       []string{},
		TemplateRate: 0.0,
	}

	totalPoints := 0
	templatePoints := 0

	for _, slide := range slides {
		for _, point := range slide.BulletPoints {
			totalPoints++
			if v.isTemplateContent(point) {
				templatePoints++
				result.Issues = append(result.Issues, "发现模板内容: "+point)
			}
		}
	}

	if totalPoints > 0 {
		result.TemplateRate = float64(templatePoints) / float64(totalPoints)
	}

	if result.TemplateRate > 0.3 {
		result.IsValid = false
		result.Issues = append(result.Issues, "模板内容比例过高，可能存在硬编码问题")
	}

	return result
}

// isTemplateContent 检测是否为模板内容
func (v *ContentValidator) isTemplateContent(content string) bool {
	templatePhrases := []string{
		"详细解释和实际应用",
		"核心技术概念和原理解释",
		"实际应用场景和案例分析",
		"实现方法和操作步骤",
		"最佳实践和优化建议",
		"常见问题和解决方案",
		"相关工具和资源推荐",
		"补充技术要点",
	}

	for _, phrase := range templatePhrases {
		if strings.Contains(content, phrase) {
			return true
		}
	}

	return false
}

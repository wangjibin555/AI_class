package coze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileValidator 文件验证器
type FileValidator struct {
	metrics *ValidationMetrics
}

// ValidationMetrics 验证指标
type ValidationMetrics struct {
	mu                 sync.RWMutex
	TotalValidations   int64 `json:"total_validations"`
	SuccessValidations int64 `json:"success_validations"`
	FailedValidations  int64 `json:"failed_validations"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	FileType string   `json:"file_type"`
	Size     int64    `json:"size"`
	Errors   []string `json:"errors,omitempty"`
}

// NewFileValidator 创建文件验证器
func NewFileValidator() *FileValidator {
	return &FileValidator{
		metrics: &ValidationMetrics{},
	}
}

// ValidatePPTFile 验证PPT文件
func (v *FileValidator) ValidatePPTFile(filePath string) error {
	v.recordValidationAttempt()

	result := v.validateFile(filePath)

	if result.IsValid {
		v.recordValidationSuccess()
		return nil
	}

	v.recordValidationFailure()
	return fmt.Errorf("文件验证失败: %v", result.Errors)
}

// ValidateGeneratedFile 验证生成的文件
func (v *FileValidator) ValidateGeneratedFile(filePath string) error {
	return v.ValidatePPTFile(filePath)
}

// validateFile 验证文件
func (v *FileValidator) validateFile(filePath string) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Errors:  make([]string, 0),
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("文件不存在: %v", err))
		return result
	}

	// 检查文件大小
	result.Size = fileInfo.Size()
	if result.Size == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "文件大小为0")
		return result
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	result.FileType = ext

	validExtensions := []string{".ppt", ".pptx", ".pdf", ".doc", ".docx", ".html", ".htm"}
	isValidExtension := false
	for _, validExt := range validExtensions {
		if ext == validExt {
			isValidExtension = true
			break
		}
	}

	// 特殊处理.tmp文件：通过文件头检测实际类型
	if ext == ".tmp" {
		actualType, err := v.detectFileTypeByHeader(filePath)
		if err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("无法检测.tmp文件的实际类型: %v", err))
			return result
		}

		// 检查检测到的类型是否有效
		for _, validExt := range validExtensions {
			if actualType == validExt {
				isValidExtension = true
				result.FileType = actualType // 更新为实际检测到的类型
				break
			}
		}

		if !isValidExtension {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("检测到的文件类型不支持: %s", actualType))
			return result
		}
	} else if !isValidExtension {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("不支持的文件类型: %s", ext))
		return result
	}

	// 基础文件头验证
	// 如果是.tmp文件，使用检测到的实际类型进行验证
	validationExt := ext
	if ext == ".tmp" && result.FileType != ".tmp" {
		validationExt = result.FileType
	}

	if err := v.validateFileHeader(filePath, validationExt); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("文件头验证失败: %v", err))
		return result
	}

	// 检查文件大小合理性
	maxSize := int64(500 * 1024 * 1024) // 500MB
	minSize := int64(100)               // 100 bytes

	if result.Size > maxSize {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("文件太大: %d bytes (最大: %d bytes)", result.Size, maxSize))
	}

	if result.Size < minSize {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("文件太小: %d bytes (最小: %d bytes)", result.Size, minSize))
	}

	return result
}

// validateFileHeader 验证文件头
func (v *FileValidator) validateFileHeader(filePath, ext string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取前16字节作为文件头
	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil {
		return fmt.Errorf("读取文件头失败: %w", err)
	}

	if n < 4 {
		return fmt.Errorf("文件头太短")
	}

	// 根据扩展名验证文件头
	switch ext {
	case ".pptx", ".docx":
		// Office Open XML格式 (ZIP文件头)
		if header[0] != 0x50 || header[1] != 0x4B || header[2] != 0x03 || header[3] != 0x04 {
			return fmt.Errorf("PPTX/DOCX文件头不正确")
		}
	case ".ppt", ".doc":
		// OLE文件头
		if header[0] != 0xD0 || header[1] != 0xCF || header[2] != 0x11 || header[3] != 0xE0 {
			return fmt.Errorf("PPT/DOC文件头不正确")
		}
	case ".pdf":
		// PDF文件头
		if string(header[:4]) != "%PDF" {
			return fmt.Errorf("PDF文件头不正确")
		}
	case ".html", ".htm":
		// HTML文件 - 检查是否包含HTML标签
		headerStr := strings.ToLower(string(header))
		if !strings.Contains(headerStr, "<html") && !strings.Contains(headerStr, "<!doctype") {
			// 可能是简单的HTML，不一定有完整的头部
			// 这里放宽验证
		}
	}

	return nil
}

// detectFileTypeByHeader 通过文件头检测文件类型
func (v *FileValidator) detectFileTypeByHeader(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取前16字节作为文件头
	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil {
		return "", fmt.Errorf("读取文件头失败: %w", err)
	}

	if n < 4 {
		return "", fmt.Errorf("文件头太短")
	}

	// 检测文件类型
	// Office Open XML格式 (PPTX/DOCX) - ZIP文件头
	if header[0] == 0x50 && header[1] == 0x4B && header[2] == 0x03 && header[3] == 0x04 {
		// 需要进一步区分是PPTX还是DOCX
		// 这里我们读取更多内容来判断
		fileContent := make([]byte, 1024)
		file.Seek(0, 0) // 重置到文件开头
		n, _ := file.Read(fileContent)
		contentStr := string(fileContent[:n])

		if strings.Contains(contentStr, "ppt/") || strings.Contains(contentStr, "presentation") {
			return ".pptx", nil
		} else if strings.Contains(contentStr, "word/") || strings.Contains(contentStr, "document") {
			return ".docx", nil
		} else {
			// 默认假设是PPTX
			return ".pptx", nil
		}
	}

	// OLE文件头 (PPT/DOC)
	if header[0] == 0xD0 && header[1] == 0xCF && header[2] == 0x11 && header[3] == 0xE0 {
		// 默认假设是PPT（可以进一步细化检测）
		return ".ppt", nil
	}

	// PDF文件头
	if string(header[:4]) == "%PDF" {
		return ".pdf", nil
	}

	// HTML文件
	headerStr := strings.ToLower(string(header))
	if strings.Contains(headerStr, "<html") || strings.Contains(headerStr, "<!doctype") || strings.Contains(headerStr, "<?xml") {
		return ".html", nil
	}

	// 如果无法识别，返回错误
	return "", fmt.Errorf("无法识别文件类型，文件头: %x", header[:8])
}

// recordValidationAttempt 记录验证尝试
func (v *FileValidator) recordValidationAttempt() {
	v.metrics.mu.Lock()
	defer v.metrics.mu.Unlock()
	v.metrics.TotalValidations++
}

// recordValidationSuccess 记录验证成功
func (v *FileValidator) recordValidationSuccess() {
	v.metrics.mu.Lock()
	defer v.metrics.mu.Unlock()
	v.metrics.SuccessValidations++
}

// recordValidationFailure 记录验证失败
func (v *FileValidator) recordValidationFailure() {
	v.metrics.mu.Lock()
	defer v.metrics.mu.Unlock()
	v.metrics.FailedValidations++
}

// GetMetrics 获取验证指标
func (v *FileValidator) GetMetrics() *ValidationMetrics {
	v.metrics.mu.RLock()
	defer v.metrics.mu.RUnlock()

	return &ValidationMetrics{
		TotalValidations:   v.metrics.TotalValidations,
		SuccessValidations: v.metrics.SuccessValidations,
		FailedValidations:  v.metrics.FailedValidations,
	}
}

// GetValidationReport 获取详细的验证报告
func (v *FileValidator) GetValidationReport(filePath string) *ValidationResult {
	v.recordValidationAttempt()
	result := v.validateFile(filePath)

	if result.IsValid {
		v.recordValidationSuccess()
	} else {
		v.recordValidationFailure()
	}

	return result
}

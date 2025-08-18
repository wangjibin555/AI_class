package crawler

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// NewEncodingDetector 创建编码检测器
func NewEncodingDetector() *EncodingDetector {
	detector := chardet.NewTextDetector()
	return &EncodingDetector{
		detector: detector,
	}
}

// DetectAndDecode 检测编码并解码
func (d *EncodingDetector) DetectAndDecode(body []byte, contentType string) (string, string) {
	// 1. 检查UTF-8 BOM
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		body = body[3:] // 移除BOM
		if utf8.Valid(body) {
			return string(body), "utf-8-bom"
		}
	}

	// 2. 检查是否已经是有效的UTF-8
	if utf8.Valid(body) {
		content := string(body)
		if d.isValidUTF8Content(content) {
			return content, "utf-8"
		}
	}

	// 3. 尝试从Content-Type获取编码
	if charset := d.extractCharsetFromContentType(contentType); charset != "" {
		if decoded, ok := d.tryDecode(body, charset); ok {
			return decoded, charset
		}
	}

	// 4. 尝试从HTML meta标签获取编码
	if charset := d.extractHTMLCharset(body); charset != "" {
		if decoded, ok := d.tryDecode(body, charset); ok {
			return decoded, charset
		}
	}

	// 5. 使用chardet自动检测
	if detector, ok := d.detector.(*chardet.Detector); ok {
		result, err := detector.DetectBest(body)
		if err == nil && result.Confidence > 60 {
			if decoded, ok := d.tryDecode(body, result.Charset); ok {
				return decoded, result.Charset
			}
		}
	}

	// 6. 尝试常见中文编码
	commonEncodings := []string{"gbk", "gb2312", "gb18030", "big5"}
	for _, encoding := range commonEncodings {
		if decoded, ok := d.tryDecode(body, encoding); ok {
			return decoded, encoding
		}
	}

	// 7. 最后的降级处理 - 直接使用原始内容，保持 HTML 结构
	content := string(body)

	// 仅进行最基本的清理，保留 HTML 结构
	// 只替换明显的无效字符，但保留换行符和制表符
	var result strings.Builder
	result.Grow(len(content))

	for _, r := range content {
		if r == utf8.RuneError {
			result.WriteRune('?') // 用 ? 替换无效字符
		} else {
			result.WriteRune(r) // 保留所有其他字符，包括换行符和制表符
		}
	}

	return result.String(), "utf-8-fallback"
}

// tryDecode 尝试使用指定编码解码
func (d *EncodingDetector) TryDecode(body []byte, charset string) (string, bool) {
	return d.tryDecode(body, charset)
}

// tryDecode 内部解码方法
func (d *EncodingDetector) tryDecode(body []byte, charset string) (string, bool) {
	charset = strings.ToLower(strings.TrimSpace(charset))

	switch charset {
	case "utf-8", "utf8":
		if utf8.Valid(body) {
			return string(body), true
		}

	case "gbk", "gb2312":
		decoder := simplifiedchinese.GBK.NewDecoder()
		decoded, _, err := transform.Bytes(decoder, body)
		if err == nil && utf8.Valid(decoded) {
			content := string(decoded)
			if d.validateDecodingQuality(content) {
				return content, true
			}
		}

	case "gb18030":
		decoder := simplifiedchinese.GB18030.NewDecoder()
		decoded, _, err := transform.Bytes(decoder, body)
		if err == nil && utf8.Valid(decoded) {
			content := string(decoded)
			if d.validateDecodingQuality(content) {
				return content, true
			}
		}

	case "iso-8859-1", "latin1":
		decoder := charmap.ISO8859_1.NewDecoder()
		decoded, _, err := transform.Bytes(decoder, body)
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), true
		}

	case "windows-1252", "cp1252":
		decoder := charmap.Windows1252.NewDecoder()
		decoded, _, err := transform.Bytes(decoder, body)
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), true
		}
	}

	return "", false
}

// extractCharsetFromContentType 从Content-Type中提取字符集
func (d *EncodingDetector) extractCharsetFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}

	// 查找charset参数
	re := regexp.MustCompile(`charset=([^;\s]+)`)
	matches := re.FindStringSubmatch(strings.ToLower(contentType))
	if len(matches) > 1 {
		return strings.Trim(matches[1], `"' `)
	}

	return ""
}

// extractHTMLCharset 从HTML meta标签中提取字符集
func (d *EncodingDetector) extractHTMLCharset(body []byte) string {
	// 只检查前8KB，提高性能
	searchContent := body
	if len(body) > 8192 {
		searchContent = body[:8192]
	}

	content := string(searchContent)
	content = strings.ToLower(content)

	// 查找各种meta charset格式
	patterns := []string{
		`<meta[^>]+charset\s*=\s*["']?([^"'\s>]+)`,
		`<meta[^>]+content\s*=\s*["'][^"']*charset=([^"'\s;]+)`,
		`<\?xml[^>]+encoding\s*=\s*["']?([^"'\s>]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			charset := strings.Trim(matches[1], `"' `)
			if charset != "" {
				return charset
			}
		}
	}

	return ""
}

// isValidUTF8Content 检查UTF-8内容的质量
func (d *EncodingDetector) isValidUTF8Content(content string) bool {
	if len(content) == 0 {
		return false
	}

	// 检查替换字符的比例
	replacementCount := strings.Count(content, "\uFFFD")
	if len(content) > 0 && float64(replacementCount)/float64(len(content)) > 0.01 {
		return false
	}

	// 检查控制字符的比例
	controlCount := 0
	for _, r := range content {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
	}

	if len(content) > 0 && float64(controlCount)/float64(len(content)) > 0.05 {
		return false
	}

	return true
}

// validateDecodingQuality 验证解码质量
func (d *EncodingDetector) validateDecodingQuality(content string) bool {
	if len(content) == 0 {
		return false
	}

	// 检查是否包含过多的替换字符
	replacementCount := strings.Count(content, "\uFFFD")
	if float64(replacementCount)/float64(len(content)) > 0.02 {
		return false
	}

	// 检查是否包含合理的可打印字符
	printableCount := 0
	for _, r := range content {
		if r >= 32 && r < 127 || r >= 0x4e00 && r <= 0x9fff { // ASCII可打印字符或中文字符
			printableCount++
		}
	}

	if float64(printableCount)/float64(len(content)) < 0.5 {
		return false
	}

	return true
}

// cleanInvalidChars 清理无效字符
func (d *EncodingDetector) cleanInvalidChars(content string) string {
	// 替换无效的UTF-8字符
	var result strings.Builder
	result.Grow(len(content))

	for _, r := range content {
		if r == utf8.RuneError {
			result.WriteRune(' ')
		} else if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			result.WriteRune(' ')
		} else {
			result.WriteRune(r)
		}
	}

	// 标准化空白字符
	cleaned := result.String()
	re := regexp.MustCompile(`\s+`)
	cleaned = re.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}

package utils

import (
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// KeyInfo 关键信息结构
type KeyInfo struct {
	Title      string   `json:"title"`       // 文档标题
	Keywords   []string `json:"keywords"`    // 关键词
	Summary    string   `json:"summary"`     // 内容摘要
	Topics     []string `json:"topics"`      // 主题分类
	Difficulty string   `json:"difficulty"`  // 难度等级
	Duration   int      `json:"duration"`    // 预估学习时长（分钟）
	MainPoints []string `json:"main_points"` // 主要观点
	Structure  []string `json:"structure"`   // 内容结构
}

// NewError 创建新的错误
func NewError(message string) error {
	return fmt.Errorf("%s", message)
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// Contains 检查切片是否包含指定元素
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// SaveUploadedFile 保存上传的文件
func SaveUploadedFile(file *multipart.FileHeader, subDir string) (string, error) {
	// 创建存储目录
	uploadDir := filepath.Join("storage", "uploads", subDir)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	return filePath, nil
}

// IsValidURL 验证URL格式
func IsValidURL(url string) bool {
	urlPattern := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlPattern.MatchString(url)
}

// NewDocumentParser 创建文档解析器
func NewDocumentParser() interface{} {
	// 这里应该返回实际的文档解析器实例
	// 暂时返回nil，实际使用时需要导入parser包
	return nil
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

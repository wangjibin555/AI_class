package coze

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// FileStorage 文件存储器
type FileStorage struct {
	config     *CozeConfig
	storageDir string
	baseURL    string
	metrics    *StorageMetrics
}

// StoredFile 存储的文件信息
type StoredFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

// StorageMetrics 存储指标
type StorageMetrics struct {
	mu            sync.RWMutex
	TotalStored   int64 `json:"total_stored"`
	SuccessStored int64 `json:"success_stored"`
	FailedStored  int64 `json:"failed_stored"`
	TotalBytes    int64 `json:"total_bytes"`
}

// NewFileStorage 创建文件存储器
func NewFileStorage(config *CozeConfig) *FileStorage {
	storageDir := "./storage/ppt"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Printf("创建存储目录失败: %v", err)
	}

	// 从配置中获取文件服务基础URL
	fileBaseURL := viper.GetString("server.file_base_url")
	if fileBaseURL == "" {
		fileBaseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}

	return &FileStorage{
		config:     config,
		storageDir: storageDir,
		baseURL:    fileBaseURL,
		metrics:    &StorageMetrics{},
	}
}

// StorePPTFile 存储PPT文件
func (s *FileStorage) StorePPTFile(sourcePath, title string) (*StoredFile, error) {
	s.recordStorageAttempt()

	// 生成唯一文件名
	fileName := s.generateFileName(title, filepath.Ext(sourcePath))
	destPath := filepath.Join(s.storageDir, fileName)

	// 复制文件
	if err := s.copyFile(sourcePath, destPath); err != nil {
		s.recordStorageFailure()
		return nil, fmt.Errorf("复制文件失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(destPath)
	if err != nil {
		s.recordStorageFailure()
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 生成下载URL
	downloadURL := fmt.Sprintf("%s/api/v1/coze/download/%s", s.baseURL, fileName)

	s.recordStorageSuccess(fileInfo.Size())

	return &StoredFile{
		Path: destPath,
		URL:  downloadURL,
		Size: fileInfo.Size(),
	}, nil
}

// generateFileName 生成文件名
func (s *FileStorage) generateFileName(title, ext string) string {
	// 清理标题作为文件名
	cleanTitle := s.cleanFileName(title)
	if cleanTitle == "" {
		cleanTitle = "ppt"
	}

	// 生成唯一ID
	id := uuid.New().String()[:8]
	timestamp := time.Now().Format("20060102")

	return fmt.Sprintf("%s_%s_%s%s", cleanTitle, timestamp, id, ext)
}

// cleanFileName 清理文件名
func (s *FileStorage) cleanFileName(name string) string {
	// 移除特殊字符
	cleaned := strings.ReplaceAll(name, " ", "_")
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "\\", "_")
	cleaned = strings.ReplaceAll(cleaned, ":", "_")
	cleaned = strings.ReplaceAll(cleaned, "*", "_")
	cleaned = strings.ReplaceAll(cleaned, "?", "_")
	cleaned = strings.ReplaceAll(cleaned, "\"", "_")
	cleaned = strings.ReplaceAll(cleaned, "<", "_")
	cleaned = strings.ReplaceAll(cleaned, ">", "_")
	cleaned = strings.ReplaceAll(cleaned, "|", "_")

	// 限制长度
	if len(cleaned) > 50 {
		cleaned = cleaned[:50]
	}

	return cleaned
}

// copyFile 复制文件
func (s *FileStorage) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	return nil
}

// recordStorageAttempt 记录存储尝试
func (s *FileStorage) recordStorageAttempt() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.TotalStored++
}

// recordStorageSuccess 记录存储成功
func (s *FileStorage) recordStorageSuccess(bytes int64) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.SuccessStored++
	s.metrics.TotalBytes += bytes
}

// recordStorageFailure 记录存储失败
func (s *FileStorage) recordStorageFailure() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.FailedStored++
}

// GetMetrics 获取存储指标
func (s *FileStorage) GetMetrics() *StorageMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return &StorageMetrics{
		TotalStored:   s.metrics.TotalStored,
		SuccessStored: s.metrics.SuccessStored,
		FailedStored:  s.metrics.FailedStored,
		TotalBytes:    s.metrics.TotalBytes,
	}
}

// CleanupOldFiles 清理旧文件
func (s *FileStorage) CleanupOldFiles(olderThan time.Duration) error {
	entries, err := os.ReadDir(s.storageDir)
	if err != nil {
		return fmt.Errorf("读取存储目录失败: %w", err)
	}

	cutoff := time.Now().Add(-olderThan)
	cleaned := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(s.storageDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				log.Printf("清理存储文件失败: %v", err)
			} else {
				cleaned++
			}
		}
	}

	log.Printf("清理了 %d 个存储文件", cleaned)
	return nil
}

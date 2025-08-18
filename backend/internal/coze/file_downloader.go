package coze

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileDownloader 文件下载器
type FileDownloader struct {
	httpClient  *http.Client
	tempDir     string
	maxFileSize int64
	metrics     *DownloadMetrics
}

// DownloadedFile 下载的文件信息
type DownloadedFile struct {
	TempPath    string    `json:"temp_path"`
	OriginalURL string    `json:"original_url"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	DownloadAt  time.Time `json:"download_at"`
}

// DownloadMetrics 下载指标
type DownloadMetrics struct {
	mu               sync.RWMutex
	TotalDownloads   int64         `json:"total_downloads"`
	SuccessDownloads int64         `json:"success_downloads"`
	FailedDownloads  int64         `json:"failed_downloads"`
	TotalBytes       int64         `json:"total_bytes"`
	AvgDownloadTime  time.Duration `json:"avg_download_time"`
	LastDownloadAt   time.Time     `json:"last_download_at"`
}

// NewFileDownloader 创建文件下载器
func NewFileDownloader(config *CozeConfig) *FileDownloader {
	tempDir := filepath.Join(os.TempDir(), "coze_downloads")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("创建临时目录失败: %v", err)
		tempDir = os.TempDir()
	}

	return &FileDownloader{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // 5分钟下载超时
		},
		tempDir:     tempDir,
		maxFileSize: 100 * 1024 * 1024, // 100MB
		metrics:     &DownloadMetrics{},
	}
}

// DownloadPPTFile 下载PPT文件
func (d *FileDownloader) DownloadPPTFile(ctx context.Context, url string) (*DownloadedFile, error) {
	startTime := time.Now()
	d.recordDownloadAttempt()

	log.Printf("开始下载文件: %s", url)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		d.recordDownloadFailure()
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "AI-Classroom-PPT-Downloader/1.0")
	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.presentationml.presentation,application/vnd.ms-powerpoint,application/pdf,*/*")

	// 执行请求
	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.recordDownloadFailure()
		return nil, fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		d.recordDownloadFailure()
		return nil, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 检查文件大小
	contentLength := resp.ContentLength
	if contentLength > d.maxFileSize {
		d.recordDownloadFailure()
		return nil, fmt.Errorf("文件太大: %d bytes (最大: %d bytes)", contentLength, d.maxFileSize)
	}

	// 生成临时文件名
	tempFileName := d.generateTempFileName(url, resp.Header.Get("Content-Type"))
	tempPath := filepath.Join(d.tempDir, tempFileName)

	// 创建临时文件
	tempFile, err := os.Create(tempPath)
	if err != nil {
		d.recordDownloadFailure()
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer tempFile.Close()

	// 限制读取大小
	limitedReader := io.LimitReader(resp.Body, d.maxFileSize)

	// 下载文件
	bytesWritten, err := io.Copy(tempFile, limitedReader)
	if err != nil {
		d.recordDownloadFailure()
		os.Remove(tempPath) // 清理失败的文件
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	// 验证文件大小
	if bytesWritten == 0 {
		d.recordDownloadFailure()
		os.Remove(tempPath)
		return nil, fmt.Errorf("下载的文件为空")
	}

	downloadTime := time.Since(startTime)
	d.recordDownloadSuccess(bytesWritten, downloadTime)

	downloadedFile := &DownloadedFile{
		TempPath:    tempPath,
		OriginalURL: url,
		Size:        bytesWritten,
		ContentType: resp.Header.Get("Content-Type"),
		DownloadAt:  time.Now(),
	}

	log.Printf("文件下载成功: %s (大小: %d bytes, 耗时: %v)", tempPath, bytesWritten, downloadTime)
	return downloadedFile, nil
}

// generateTempFileName 生成临时文件名
func (d *FileDownloader) generateTempFileName(url, contentType string) string {
	timestamp := time.Now().Format("20060102_150405")

	// 从URL或Content-Type推断扩展名
	ext := d.inferFileExtension(url, contentType)

	return fmt.Sprintf("coze_download_%s%s", timestamp, ext)
}

// inferFileExtension 推断文件扩展名
func (d *FileDownloader) inferFileExtension(url, contentType string) string {
	// 从URL推断
	if strings.Contains(url, ".pptx") {
		return ".pptx"
	}
	if strings.Contains(url, ".ppt") {
		return ".ppt"
	}
	if strings.Contains(url, ".pdf") {
		return ".pdf"
	}

	// 从Content-Type推断
	switch contentType {
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/pdf":
		return ".pdf"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/msword":
		return ".doc"
	default:
		return ".tmp"
	}
}

// recordDownloadAttempt 记录下载尝试
func (d *FileDownloader) recordDownloadAttempt() {
	d.metrics.mu.Lock()
	defer d.metrics.mu.Unlock()
	d.metrics.TotalDownloads++
	d.metrics.LastDownloadAt = time.Now()
}

// recordDownloadSuccess 记录下载成功
func (d *FileDownloader) recordDownloadSuccess(bytes int64, duration time.Duration) {
	d.metrics.mu.Lock()
	defer d.metrics.mu.Unlock()

	d.metrics.SuccessDownloads++
	d.metrics.TotalBytes += bytes

	// 更新平均下载时间
	if d.metrics.SuccessDownloads == 1 {
		d.metrics.AvgDownloadTime = duration
	} else {
		d.metrics.AvgDownloadTime = time.Duration(
			(int64(d.metrics.AvgDownloadTime)*(d.metrics.SuccessDownloads-1) + int64(duration)) / d.metrics.SuccessDownloads,
		)
	}
}

// recordDownloadFailure 记录下载失败
func (d *FileDownloader) recordDownloadFailure() {
	d.metrics.mu.Lock()
	defer d.metrics.mu.Unlock()
	d.metrics.FailedDownloads++
}

// GetMetrics 获取下载指标
func (d *FileDownloader) GetMetrics() *DownloadMetrics {
	d.metrics.mu.RLock()
	defer d.metrics.mu.RUnlock()

	// 创建副本
	return &DownloadMetrics{
		TotalDownloads:   d.metrics.TotalDownloads,
		SuccessDownloads: d.metrics.SuccessDownloads,
		FailedDownloads:  d.metrics.FailedDownloads,
		TotalBytes:       d.metrics.TotalBytes,
		AvgDownloadTime:  d.metrics.AvgDownloadTime,
		LastDownloadAt:   d.metrics.LastDownloadAt,
	}
}

// CleanupTempFiles 清理临时文件
func (d *FileDownloader) CleanupTempFiles(olderThan time.Duration) error {
	entries, err := os.ReadDir(d.tempDir)
	if err != nil {
		return fmt.Errorf("读取临时目录失败: %w", err)
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
			filePath := filepath.Join(d.tempDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				log.Printf("清理临时文件失败: %v", err)
			} else {
				cleaned++
			}
		}
	}

	log.Printf("清理了 %d 个临时文件", cleaned)
	return nil
}

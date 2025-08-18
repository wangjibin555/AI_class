package coze

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// DownloadManager 下载管理器
type DownloadManager struct {
	storageDir string
	baseURL    string
	serverPort string
}

// NewDownloadManager 创建下载管理器
func NewDownloadManager(storageDir, baseURL, serverPort string) *DownloadManager {
	return &DownloadManager{
		storageDir: storageDir,
		baseURL:    baseURL,
		serverPort: serverPort,
	}
}

// StoreFile 存储文件并生成下载链接
func (dm *DownloadManager) StoreFile(sourceFilePath, taskID string) (*StoredFileInfo, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(dm.storageDir, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 生成目标文件名
	ext := filepath.Ext(sourceFilePath)
	filename := fmt.Sprintf("coze_ppt_%s_%d%s", taskID, time.Now().Unix(), ext)
	targetPath := filepath.Join(dm.storageDir, filename)

	// 复制文件到存储目录
	if err := dm.copyFile(sourceFilePath, targetPath); err != nil {
		return nil, fmt.Errorf("复制文件失败: %w", err)
	}

	// 生成下载链接
	downloadURL := dm.generateDownloadURL(filename)

	return &StoredFileInfo{
		FilePath:    targetPath,
		FileName:    filename,
		FileSize:    fileInfo.Size(),
		DownloadURL: downloadURL,
		StoredAt:    time.Now(),
	}, nil
}

// copyFile 复制文件
func (dm *DownloadManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 复制文件内容
	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// generateDownloadURL 生成下载链接
func (dm *DownloadManager) generateDownloadURL(filename string) string {
	// 构建基础URL
	baseURL := dm.baseURL
	if baseURL == "" {
		// 从配置中获取文件服务基础URL
		baseURL = viper.GetString("server.file_base_url")
		if baseURL == "" {
			baseURL = fmt.Sprintf("http://%s:%s",
				viper.GetString("server.external_host"),
				viper.GetString("server.external_port"))
		}
	}

	// 添加端口
	if dm.serverPort != "" && !strings.Contains(baseURL, ":") {
		baseURL = baseURL + ":" + dm.serverPort
	}

	// 构建完整的下载URL
	downloadPath := "/api/v1/coze/download/" + url.QueryEscape(filename)
	return baseURL + downloadPath
}

// GetFileInfo 获取存储的文件信息
func (dm *DownloadManager) GetFileInfo(filename string) (*StoredFileInfo, error) {
	filePath := filepath.Join(dm.storageDir, filename)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件不存在: %w", err)
	}

	return &StoredFileInfo{
		FilePath:    filePath,
		FileName:    filename,
		FileSize:    fileInfo.Size(),
		DownloadURL: dm.generateDownloadURL(filename),
		StoredAt:    fileInfo.ModTime(),
	}, nil
}

// ServeFile 提供文件下载服务
func (dm *DownloadManager) ServeFile(filename string, w http.ResponseWriter, r *http.Request) error {
	filePath := filepath.Join(dm.storageDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return fmt.Errorf("文件不存在: %s", filename)
	}

	// 设置响应头
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := dm.getContentType(ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// 提供文件下载
	http.ServeFile(w, r, filePath)
	return nil
}

// getContentType 根据文件扩展名获取MIME类型
func (dm *DownloadManager) getContentType(ext string) string {
	switch ext {
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".html":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// CleanupOldFiles 清理旧文件
func (dm *DownloadManager) CleanupOldFiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(dm.storageDir)
	if err != nil {
		return fmt.Errorf("读取存储目录失败: %w", err)
	}

	now := time.Now()
	var deletedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(dm.storageDir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				deletedCount++
			}
		}
	}

	fmt.Printf("清理完成，删除了 %d 个旧文件\n", deletedCount)
	return nil
}

// ListFiles 列出存储的文件
func (dm *DownloadManager) ListFiles() ([]StoredFileInfo, error) {
	entries, err := os.ReadDir(dm.storageDir)
	if err != nil {
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	var files []StoredFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := StoredFileInfo{
			FilePath:    filepath.Join(dm.storageDir, entry.Name()),
			FileName:    entry.Name(),
			FileSize:    info.Size(),
			DownloadURL: dm.generateDownloadURL(entry.Name()),
			StoredAt:    info.ModTime(),
		}
		files = append(files, fileInfo)
	}

	return files, nil
}

// GetStorageStats 获取存储统计信息
func (dm *DownloadManager) GetStorageStats() (*StorageStats, error) {
	entries, err := os.ReadDir(dm.storageDir)
	if err != nil {
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	stats := &StorageStats{
		TotalFiles: 0,
		TotalSize:  0,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		stats.TotalFiles++
		stats.TotalSize += info.Size()
	}

	return stats, nil
}

// StoredFileInfo 存储的文件信息
type StoredFileInfo struct {
	FilePath    string    `json:"file_path"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	DownloadURL string    `json:"download_url"`
	StoredAt    time.Time `json:"stored_at"`
}

// StorageStats 存储统计信息
type StorageStats struct {
	TotalFiles int   `json:"total_files"`
	TotalSize  int64 `json:"total_size"`
}

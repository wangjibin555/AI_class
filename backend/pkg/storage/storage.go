package storage

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

// StorageService 存储服务接口
type StorageService interface {
	UploadAudio(fileName string, audioData []byte) (string, error)
	UploadFile(fileName string, fileData []byte) (string, error)
	SaveAudio(filepath string, data []byte) (string, error)
	DeleteFile(fileName string) error
	GetFileURL(fileName string) string
}

// LocalStorageService 本地存储服务
type LocalStorageService struct {
	basePath string
	baseURL  string
}

// NewLocalStorageService 创建本地存储服务
func NewLocalStorageService(basePath, baseURL string) *LocalStorageService {
	return &LocalStorageService{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// UploadAudio 上传音频文件
func (s *LocalStorageService) UploadAudio(fileName string, audioData []byte) (string, error) {
	return s.uploadFile("audio", fileName, audioData)
}

// UploadFile 上传通用文件
func (s *LocalStorageService) UploadFile(fileName string, fileData []byte) (string, error) {
	return s.uploadFile("files", fileName, fileData)
}

// SaveAudio 保存音频文件 (为了兼容TTS服务接口)
func (s *LocalStorageService) SaveAudio(filepath string, data []byte) (string, error) {
	// 直接使用提供的文件路径，确保目录存在
	fullPath := path.Join(s.basePath, filepath)
	dirPath := path.Dir(fullPath)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 返回访问URL
	fileURL := fmt.Sprintf("%s/%s", s.baseURL, filepath)
	return fileURL, nil
}

// uploadFile 内部上传方法
func (s *LocalStorageService) uploadFile(category, fileName string, fileData []byte) (string, error) {
	// 创建目录结构
	dirPath := filepath.Join(s.basePath, category)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成唯一文件名
	ext := filepath.Ext(fileName)
	baseName := fileName[:len(fileName)-len(ext)]
	uniqueFileName := fmt.Sprintf("%s_%d%s", baseName, time.Now().Unix(), ext)

	filePath := filepath.Join(dirPath, uniqueFileName)

	// 写入文件
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 返回访问URL
	fileURL := fmt.Sprintf("%s/%s/%s", s.baseURL, category, uniqueFileName)
	return fileURL, nil
}

// DeleteFile 删除文件
func (s *LocalStorageService) DeleteFile(fileName string) error {
	filePath := filepath.Join(s.basePath, fileName)
	return os.Remove(filePath)
}

// GetFileURL 获取文件URL
func (s *LocalStorageService) GetFileURL(fileName string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, fileName)
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
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

	_, err = io.Copy(destFile, sourceFile)
	return err
}

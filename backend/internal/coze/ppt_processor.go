package coze

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// PPTProcessor PPT处理器
type PPTProcessor struct {
	downloader *FileDownloader
	validator  *FileValidator
	converter  *FormatConverter
	storage    *FileStorage
	contentGen *PPTContentGenerator
	fileGen    *PPTFileGenerator
}

// PPTFile PPT文件信息
type PPTFile struct {
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	Size        int64     `json:"size"`
	Format      string    `json:"format"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	IsGenerated bool      `json:"is_generated"` // 是否为本地生成
}

// ProcessingResult PPT处理结果
type ProcessingResult struct {
	Success     bool          `json:"success"`
	File        *PPTFile      `json:"file,omitempty"`
	Error       string        `json:"error,omitempty"`
	ProcessTime time.Duration `json:"process_time"`
	Method      string        `json:"method"` // "download" 或 "generate"
}

// NewPPTProcessor 创建PPT处理器
func NewPPTProcessor(config *CozeConfig) *PPTProcessor {
	return &PPTProcessor{
		downloader: NewFileDownloader(config),
		validator:  NewFileValidator(),
		converter:  NewFormatConverter(),
		storage:    NewFileStorage(config),
		contentGen: NewPPTContentGenerator(config),
		fileGen:    NewPPTFileGenerator("./storage/ppt"),
	}
}

// ProcessCozeResponse 处理Coze响应
func (p *PPTProcessor) ProcessCozeResponse(ctx context.Context, result *PPTResult) (*ProcessingResult, error) {
	startTime := time.Now()

	log.Printf("开始处理Coze响应: PPTLinks=%d, Content长度=%d",
		len(result.PPTLinks), len(result.Content))
	log.Printf("详细信息: ChatID=%s, Title=%s, Status=%s",
		result.ChatID, result.Title, result.Status)

	// 详细调试信息
	if len(result.Content) > 0 {
		contentPreview := result.Content
		if len(contentPreview) > 300 {
			contentPreview = contentPreview[:300] + "..."
		}
		log.Printf("内容预览: %s", contentPreview)
	} else {
		log.Printf("⚠️  内容为空，这可能是问题所在")
	}

	// 方法1: 尝试下载PPT链接
	if len(result.PPTLinks) > 0 {
		for _, link := range result.PPTLinks {
			log.Printf("尝试下载PPT链接: %s", link)

			pptFile, err := p.downloadAndProcessPPT(ctx, link, result.Title)
			if err != nil {
				log.Printf("下载PPT失败: %v", err)
				continue
			}

			return &ProcessingResult{
				Success:     true,
				File:        pptFile,
				ProcessTime: time.Since(startTime),
				Method:      "download",
			}, nil
		}
	}

	// 方法2: 从文本内容生成PPT
	if len(result.Content) > 0 || result.Status == "need_local_generation" {
		log.Printf("尝试从内容生成PPT")

		pptFile, err := p.generatePPTFromContent(ctx, result)
		if err != nil {
			return &ProcessingResult{
				Success:     false,
				Error:       fmt.Sprintf("生成PPT失败: %v", err),
				ProcessTime: time.Since(startTime),
				Method:      "generate",
			}, nil
		}

		return &ProcessingResult{
			Success:     true,
			File:        pptFile,
			ProcessTime: time.Since(startTime),
			Method:      "generate",
		}, nil
	}

	// 都失败了
	log.Printf("❌ PPT处理失败 - 详细诊断:")
	log.Printf("   PPT链接数量: %d", len(result.PPTLinks))
	log.Printf("   内容长度: %d", len(result.Content))
	log.Printf("   状态: %s", result.Status)
	log.Printf("   标题: %s", result.Title)

	errorMsg := "无有效的PPT内容或链接"
	if len(result.Content) == 0 && len(result.PPTLinks) == 0 {
		errorMsg = "Coze API响应中既无内容也无PPT链接，可能是API调用或内容提取问题"
	} else if len(result.Content) > 0 {
		errorMsg = fmt.Sprintf("有内容(%d字符)但处理失败", len(result.Content))
	}

	return &ProcessingResult{
		Success:     false,
		Error:       errorMsg,
		ProcessTime: time.Since(startTime),
		Method:      "none",
	}, nil
}

// downloadAndProcessPPT 下载并处理PPT文件
func (p *PPTProcessor) downloadAndProcessPPT(ctx context.Context, pptURL, title string) (*PPTFile, error) {
	// 验证URL
	if !p.isValidPPTURL(pptURL) {
		return nil, fmt.Errorf("无效的PPT URL: %s", pptURL)
	}

	// 下载文件
	downloadedFile, err := p.downloader.DownloadPPTFile(ctx, pptURL)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}
	defer p.cleanupTempFile(downloadedFile.TempPath)

	// 验证文件
	if err := p.validator.ValidatePPTFile(downloadedFile.TempPath); err != nil {
		return nil, fmt.Errorf("文件验证失败: %w", err)
	}

	// 存储文件
	storedFile, err := p.storage.StorePPTFile(downloadedFile.TempPath, title)
	if err != nil {
		return nil, fmt.Errorf("存储文件失败: %w", err)
	}

	return &PPTFile{
		Path:        storedFile.Path,
		URL:         storedFile.URL,
		Size:        storedFile.Size,
		Format:      filepath.Ext(storedFile.Path),
		Title:       title,
		CreatedAt:   time.Now(),
		IsGenerated: false,
	}, nil
}

// generatePPTFromContent 从内容生成PPT
func (p *PPTProcessor) generatePPTFromContent(ctx context.Context, result *PPTResult) (*PPTFile, error) {
	// 使用现有的内容生成器和文件生成器
	req := &GeneratePPTRequest{
		URL:      fmt.Sprintf("content://%s", result.ChatID), // 虚拟URL标识
		Template: "professional",
		Options: map[string]interface{}{
			"slide_count": 10,
			"language":    "zh-CN",
		},
	}

	// 生成PPT结构
	structure, err := p.contentGen.GenerateFromCozeResponse(result.Content, req)
	if err != nil {
		return nil, fmt.Errorf("生成PPT结构失败: %w", err)
	}

	// 生成PPT文件
	filePath, err := p.fileGen.GeneratePPTFile(structure, result.ChatID)
	if err != nil {
		return nil, fmt.Errorf("生成PPT文件失败: %w", err)
	}

	// 验证生成的文件
	if err := p.validator.ValidateGeneratedFile(filePath); err != nil {
		return nil, fmt.Errorf("生成的文件验证失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 生成下载URL
	fileName := filepath.Base(filePath)

	// 从配置中获取文件服务基础URL
	fileBaseURL := viper.GetString("server.file_base_url")
	if fileBaseURL == "" {
		fileBaseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	downloadURL := fmt.Sprintf("%s/api/v1/coze/download/%s", fileBaseURL, fileName)

	return &PPTFile{
		Path:        filePath,
		URL:         downloadURL,
		Size:        fileInfo.Size(),
		Format:      filepath.Ext(filePath),
		Title:       result.Title,
		CreatedAt:   time.Now(),
		IsGenerated: true,
	}, nil
}

// isValidPPTURL 验证PPT URL是否有效
func (p *PPTProcessor) isValidPPTURL(pptURL string) bool {
	// 解析URL
	parsedURL, err := url.Parse(pptURL)
	if err != nil {
		return false
	}

	// 检查协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// 检查扩展名
	path := strings.ToLower(parsedURL.Path)
	validExtensions := []string{".ppt", ".pptx", ".pdf", ".doc", ".docx"}

	for _, ext := range validExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	// 检查特殊的PPT服务域名
	host := strings.ToLower(parsedURL.Host)
	pptServices := []string{
		"chat-ppt.com",
		"gamma.app",
		"tome.app",
		"beautiful.ai",
		"oceancloudapi.com",               // 添加Coze的文件存储域名
		"appstore-sign.oceancloudapi.com", // 添加具体的子域名
	}

	for _, service := range pptServices {
		if strings.Contains(host, service) {
			return true
		}
	}

	return false
}

// cleanupTempFile 清理临时文件
func (p *PPTProcessor) cleanupTempFile(tempPath string) {
	if tempPath != "" {
		if err := os.Remove(tempPath); err != nil {
			log.Printf("清理临时文件失败: %v", err)
		}
	}
}

// GetProcessorMetrics 获取处理器指标
func (p *PPTProcessor) GetProcessorMetrics() map[string]interface{} {
	return map[string]interface{}{
		"downloader_metrics": p.downloader.GetMetrics(),
		"validator_metrics":  p.validator.GetMetrics(),
		"converter_metrics":  p.converter.GetMetrics(),
		"storage_metrics":    p.storage.GetMetrics(),
	}
}

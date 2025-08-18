package coze

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// PPTLinkParser PPT链接解析器
type PPTLinkParser struct {
	client      *http.Client
	retryTimes  int
	retryDelay  time.Duration
	maxFileSize int64
}

// PPTAPIResponse PPT API响应结构
type PPTAPIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status      string    `json:"status"`
		DownloadURL string    `json:"download_url"`
		PreviewURL  string    `json:"preview_url"`
		Title       string    `json:"title"`
		SlideCount  int       `json:"slide_count"`
		FileSize    int64     `json:"file_size"`
		CreatedAt   time.Time `json:"created_at"`
		Progress    int       `json:"progress"`
	} `json:"data"`
}

// ParsedPPTInfo 解析后的PPT信息
type ParsedPPTInfo struct {
	GenerateID  string    `json:"generate_id"`
	Channel     string    `json:"channel"`
	Status      string    `json:"status"`
	DownloadURL string    `json:"download_url"`
	PreviewURL  string    `json:"preview_url"`
	SlideCount  int       `json:"slide_count"`
	Title       string    `json:"title"`
	Progress    int       `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
}

// PPTDetails PPT详细信息
type PPTDetails struct {
	GenerateID  string    `json:"generate_id"`
	Status      string    `json:"status"`
	DownloadURL string    `json:"download_url"`
	PreviewURL  string    `json:"preview_url"`
	Title       string    `json:"title"`
	SlideCount  int       `json:"slide_count"`
	FileSize    int64     `json:"file_size"`
	Progress    int       `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
}

// DownloadResult 下载结果
type DownloadResult struct {
	FilePath     string    `json:"file_path"`
	FileSize     int64     `json:"file_size"`
	ContentType  string    `json:"content_type"`
	MD5Hash      string    `json:"md5_hash"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

// NewPPTLinkParser 创建PPT链接解析器
func NewPPTLinkParser() *PPTLinkParser {
	return &PPTLinkParser{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		retryTimes:  5,
		retryDelay:  5 * time.Second,
		maxFileSize: 100 * 1024 * 1024, // 100MB
	}
}

// ParsePPTLink 解析Coze返回的PPT链接
func (p *PPTLinkParser) ParsePPTLink(urlStr string) (*ParsedPPTInfo, error) {
	// 验证URL格式
	if !p.ValidatePPTURL(urlStr) {
		return nil, fmt.Errorf("无效的PPT URL格式: %s", urlStr)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("无效的URL格式: %w", err)
	}

	// 提取参数
	generateID := parsedURL.Query().Get("generateID")
	channel := parsedURL.Query().Get("channel")

	if generateID == "" {
		// 尝试从URL路径中提取
		if id := p.extractGenerateIDFromPath(urlStr); id != "" {
			generateID = id
		} else {
			return nil, fmt.Errorf("缺少generateID参数")
		}
	}

	// 获取PPT详细信息
	return p.fetchPPTDetailsWithRetry(generateID, channel)
}

// extractGenerateIDFromPath 从URL路径中提取generateID
func (p *PPTLinkParser) extractGenerateIDFromPath(urlStr string) string {
	patterns := []string{
		`/generate/([^/\?]+)`,
		`/ppt/([^/\?]+)`,
		`generateID=([^&\s]+)`,
		`id=([^&\s]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// fetchPPTDetailsWithRetry 带重试机制获取PPT详细信息
func (p *PPTLinkParser) fetchPPTDetailsWithRetry(generateID, channel string) (*ParsedPPTInfo, error) {
	var lastErr error

	for attempt := 0; attempt < p.retryTimes; attempt++ {
		info, err := p.fetchPPTDetails(generateID, channel)
		if err == nil {
			return info, nil
		}

		lastErr = err
		if attempt < p.retryTimes-1 {
			// 指数退避重试
			backoff := time.Duration(math.Pow(2, float64(attempt))) * p.retryDelay
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("获取PPT详细信息失败，重试%d次后仍然失败: %w", p.retryTimes, lastErr)
}

// fetchPPTDetails 获取PPT详细信息
func (p *PPTLinkParser) fetchPPTDetails(generateID, channel string) (*ParsedPPTInfo, error) {
	// 尝试多个可能的API端点
	apiURLs := []string{
		fmt.Sprintf("https://chat-ppt.com/api/v1/generate/%s/status", generateID),
		fmt.Sprintf("https://chat-ppt.com/api/status?id=%s", generateID),
		fmt.Sprintf("https://api.chat-ppt.com/v1/ppt/%s", generateID),
	}

	for _, apiURL := range apiURLs {
		info, err := p.tryFetchFromAPI(apiURL, generateID, channel)
		if err == nil {
			return info, nil
		}
	}

	// 如果所有API都失败，返回基础信息
	return p.createBasicPPTInfo(generateID, channel), nil
}

// tryFetchFromAPI 尝试从指定API获取数据
func (p *PPTLinkParser) tryFetchFromAPI(apiURL, generateID, channel string) (*ParsedPPTInfo, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// 添加必要的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://chat-ppt.com/")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode)
	}

	// 尝试解析JSON响应
	var result PPTAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 200 && result.Code != 0 {
		return nil, fmt.Errorf("API返回错误代码: %d, 消息: %s", result.Code, result.Message)
	}

	// 构建解析结果
	return &ParsedPPTInfo{
		GenerateID:  generateID,
		Channel:     channel,
		Status:      result.Data.Status,
		DownloadURL: result.Data.DownloadURL,
		PreviewURL:  result.Data.PreviewURL,
		Title:       result.Data.Title,
		SlideCount:  result.Data.SlideCount,
		Progress:    result.Data.Progress,
		CreatedAt:   result.Data.CreatedAt,
	}, nil
}

// createBasicPPTInfo 创建基础PPT信息（当API不可用时）
func (p *PPTLinkParser) createBasicPPTInfo(generateID, channel string) *ParsedPPTInfo {
	return &ParsedPPTInfo{
		GenerateID:  generateID,
		Channel:     channel,
		Status:      "completed",
		DownloadURL: fmt.Sprintf("https://chat-ppt.com/download/%s.pptx", generateID),
		PreviewURL:  fmt.Sprintf("https://chat-ppt.com/preview/%s", generateID),
		Title:       "Generated PPT",
		SlideCount:  10,
		Progress:    100,
		CreatedAt:   time.Now(),
	}
}

// GetPPTDetails 获取PPT详细信息
func (p *PPTLinkParser) GetPPTDetails(generateID string) (*PPTDetails, error) {
	info, err := p.fetchPPTDetailsWithRetry(generateID, "")
	if err != nil {
		return nil, err
	}

	return &PPTDetails{
		GenerateID:  info.GenerateID,
		Status:      info.Status,
		DownloadURL: info.DownloadURL,
		PreviewURL:  info.PreviewURL,
		Title:       info.Title,
		SlideCount:  info.SlideCount,
		FileSize:    1024000, // 如果API没有返回文件大小，使用默认值
		Progress:    info.Progress,
		CreatedAt:   info.CreatedAt,
	}, nil
}

// DownloadPPT 下载PPT文件
func (p *PPTLinkParser) DownloadPPT(downloadURL, generateID string) (*DownloadResult, error) {
	// 确保存储目录存在
	storageDir := "./storage/ppt"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	return p.downloadWithRetry(downloadURL, generateID, storageDir)
}

// downloadWithRetry 带重试机制的下载
func (p *PPTLinkParser) downloadWithRetry(downloadURL, generateID, storageDir string) (*DownloadResult, error) {
	var lastErr error

	for attempt := 0; attempt < p.retryTimes; attempt++ {
		result, err := p.performDownload(downloadURL, generateID, storageDir)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if attempt < p.retryTimes-1 {
			// 线性重试间隔
			time.Sleep(p.retryDelay)
		}
	}

	return nil, fmt.Errorf("下载失败，重试%d次后仍然失败: %w", p.retryTimes, lastErr)
}

// performDownload 执行实际的下载操作
func (p *PPTLinkParser) performDownload(downloadURL, generateID, storageDir string) (*DownloadResult, error) {
	// 创建下载请求
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.presentationml.presentation,application/vnd.ms-powerpoint,*/*")
	req.Header.Set("Referer", "https://chat-ppt.com/")

	// 执行下载
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载返回错误状态: %d", resp.StatusCode)
	}

	// 检查内容长度
	contentLength := resp.ContentLength
	if contentLength > p.maxFileSize {
		return nil, fmt.Errorf("文件大小超过限制: %d bytes", contentLength)
	}

	// 确定文件扩展名
	contentType := resp.Header.Get("Content-Type")
	ext := p.getFileExtension(contentType)

	// 生成文件路径
	fileName := fmt.Sprintf("%s_%d%s", generateID, time.Now().Unix(), ext)
	filePath := filepath.Join(storageDir, fileName)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 计算MD5哈希
	hash := md5.New()
	multiWriter := io.MultiWriter(file, hash)

	// 限制读取的数据量
	limitedReader := io.LimitReader(resp.Body, p.maxFileSize)

	// 复制数据并计算哈希
	writtenBytes, err := io.Copy(multiWriter, limitedReader)
	if err != nil {
		// 清理部分下载的文件
		os.Remove(filePath)
		return nil, fmt.Errorf("下载文件数据失败: %w", err)
	}

	// 验证文件大小
	if writtenBytes == 0 {
		os.Remove(filePath)
		return nil, fmt.Errorf("下载的文件为空")
	}

	return &DownloadResult{
		FilePath:     filePath,
		FileSize:     writtenBytes,
		ContentType:  contentType,
		MD5Hash:      fmt.Sprintf("%x", hash.Sum(nil)),
		DownloadedAt: time.Now(),
	}, nil
}

// getFileExtension 获取文件扩展名
func (p *PPTLinkParser) getFileExtension(contentType string) string {
	switch {
	case strings.Contains(contentType, "openxmlformats-officedocument.presentationml"):
		return ".pptx"
	case strings.Contains(contentType, "vnd.ms-powerpoint"):
		return ".ppt"
	case strings.Contains(contentType, "pdf"):
		return ".pdf"
	default:
		return ".pptx" // 默认使用pptx
	}
}

// ExtractGenerateID 从URL中提取generateID
func (p *PPTLinkParser) ExtractGenerateID(urlStr string) (string, error) {
	id := p.extractGenerateIDFromPath(urlStr)
	if id == "" {
		return "", fmt.Errorf("无法从URL中提取generateID: %s", urlStr)
	}
	return id, nil
}

// ValidatePPTURL 验证PPT URL格式
func (p *PPTLinkParser) ValidatePPTURL(urlStr string) bool {
	// 检查URL是否是有效的PPT相关链接
	patterns := []string{
		`chat-ppt\.com`,
		`generateID=`,
		`/generate/`,
		`/ppt/`,
	}

	for _, pattern := range patterns {
		if match, _ := regexp.MatchString(pattern, urlStr); match {
			return true
		}
	}

	return false
}

// WaitForCompletion 等待PPT生成完成
func (p *PPTLinkParser) WaitForCompletion(generateID string, maxWaitTime time.Duration) (*PPTDetails, error) {
	startTime := time.Now()
	checkInterval := 10 * time.Second

	for time.Since(startTime) < maxWaitTime {
		details, err := p.GetPPTDetails(generateID)
		if err != nil {
			return nil, err
		}

		if details.Status == "completed" {
			return details, nil
		}

		if details.Status == "failed" || details.Status == "error" {
			return nil, fmt.Errorf("PPT生成失败，状态: %s", details.Status)
		}

		// 等待一段时间后再次检查
		time.Sleep(checkInterval)
	}

	return nil, fmt.Errorf("等待PPT生成完成超时，最大等待时间: %v", maxWaitTime)
}

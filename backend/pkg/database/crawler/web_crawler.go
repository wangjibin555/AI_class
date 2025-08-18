package crawler

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// WebCrawler 网页爬虫结构体
type WebCrawler struct {
	collector *colly.Collector
	extractor *ContentExtractor
	validator *URLValidator
	config    *CrawlerConfig
}

// CrawlerConfig 爬虫配置
type CrawlerConfig struct {
	// 请求相关配置
	UserAgent   string        `json:"user_agent"`
	Timeout     time.Duration `json:"timeout"`
	MaxDepth    int           `json:"max_depth"`
	EnableDebug bool          `json:"enable_debug"`

	// 反爬虫配置
	DelayMin    time.Duration `json:"delay_min"`
	DelayMax    time.Duration `json:"delay_max"`
	RandomDelay bool          `json:"random_delay"`

	// 内容过滤配置
	MaxContentLength int      `json:"max_content_length"`
	AllowedDomains   []string `json:"allowed_domains"`

	// 代理配置
	ProxyURLs []string `json:"proxy_urls"`
}

// CrawlResult 爬取结果
type CrawlResult struct {
	// 基础信息
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`

	// 元数据
	Description   string   `json:"description"`
	Keywords      []string `json:"keywords"`
	PublishedTime string   `json:"published_time"`
	WordCount     int      `json:"word_count"`

	// 媒体资源
	Images []ImageInfo `json:"images"`
	Videos []VideoInfo `json:"videos"`

	// 技术信息
	Language    string `json:"language"`
	Encoding    string `json:"encoding"`
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type"`

	// 处理信息
	CrawlTime   time.Time     `json:"crawl_time"`
	ProcessTime time.Duration `json:"process_time"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
}

// ImageInfo 图片信息
type ImageInfo struct {
	URL    string `json:"url"`
	Alt    string `json:"alt"`
	Title  string `json:"title"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// VideoInfo 视频信息
type VideoInfo struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Duration string `json:"duration"`
	Poster   string `json:"poster"`
}

// NewWebCrawler 创建新的网页爬虫实例
func NewWebCrawler(config *CrawlerConfig) *WebCrawler {
	if config == nil {
		config = DefaultCrawlerConfig()
	}

	// 创建Colly实例
	collector := colly.NewCollector(
		colly.UserAgent(config.UserAgent),
	)

	// 设置限制
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       config.DelayMin,
	})

	// 设置超时
	collector.SetRequestTimeout(config.Timeout)

	// 设置允许的域名
	if len(config.AllowedDomains) > 0 {
		collector.AllowedDomains = config.AllowedDomains
	}

	// 创建爬虫实例
	crawler := &WebCrawler{
		collector: collector,
		extractor: NewContentExtractor(),
		validator: NewURLValidator(),
		config:    config,
	}

	// 设置回调函数
	crawler.setupCallbacks()

	return crawler
}

// DefaultCrawlerConfig 默认爬虫配置
func DefaultCrawlerConfig() *CrawlerConfig {
	return &CrawlerConfig{
		UserAgent:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Timeout:          30 * time.Second,
		MaxDepth:         1,
		EnableDebug:      false,
		DelayMin:         1 * time.Second,
		DelayMax:         3 * time.Second,
		RandomDelay:      true,
		MaxContentLength: 1024 * 1024, // 1MB
		AllowedDomains:   []string{},
		ProxyURLs:        []string{},
	}
}

// setupCallbacks 设置爬虫回调函数
func (w *WebCrawler) setupCallbacks() {
	// 请求前回调
	w.collector.OnRequest(func(r *colly.Request) {
		// 设置随机延时
		if w.config.RandomDelay {
			// 这里可以添加随机延时逻辑
		}

		// 设置请求头
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("Cache-Control", "no-cache")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")

		if w.config.EnableDebug {
			fmt.Printf("Debug: Visiting %s\n", r.URL)
		}
	})

	// 错误回调
	w.collector.OnError(func(r *colly.Response, err error) {
		if w.config.EnableDebug {
			fmt.Printf("Debug: Error visiting %s: %s\n", r.Request.URL, err.Error())
		}
	})

	// 响应回调
	w.collector.OnResponse(func(r *colly.Response) {
		if w.config.EnableDebug {
			fmt.Printf("Debug: Visited %s with status %d\n", r.Request.URL, r.StatusCode)
		}
	})
}

// CrawlURL 爬取指定URL的内容
func (w *WebCrawler) CrawlURL(url string) (*CrawlResult, error) {
	startTime := time.Now()

	fmt.Printf("\n🕷️ 开始爬取URL调试:\n")
	fmt.Printf("🎯 目标URL: %s\n", url)
	fmt.Printf("⏰ 开始时间: %s\n", startTime.Format("2006-01-02 15:04:05"))

	// 验证URL
	fmt.Printf("\n📋 步骤1: URL验证\n")
	if !w.validator.IsValidURL(url) {
		fmt.Printf("❌ URL验证失败: %s\n", url)
		return nil, fmt.Errorf("invalid URL: %s", url)
	}
	fmt.Printf("✅ URL验证通过\n")

	// 检查URL是否支持
	fmt.Printf("\n🔍 步骤2: 平台检测\n")
	platform := w.validator.DetectPlatform(url)
	if platform == PlatformUnsupported {
		fmt.Printf("❌ 不支持的平台: %s\n", url)
		return nil, fmt.Errorf("unsupported platform for URL: %s", url)
	}
	fmt.Printf("✅ 检测到平台: %v\n", platform)

	// 初始化结果
	result := &CrawlResult{
		URL:       url,
		CrawlTime: startTime,
		Success:   false,
	}

	// 直接使用HTTP客户端获取内容（为了更好的控制）
	fmt.Printf("\n🌐 步骤3: 发送HTTP请求\n")
	content, statusCode, contentType, err := w.fetchContent(url)
	if err != nil {
		fmt.Printf("❌ HTTP请求失败: %v\n", err)
		result.Error = err.Error()
		result.ProcessTime = time.Since(startTime)
		return result, err
	}
	fmt.Printf("✅ HTTP请求成功\n")
	fmt.Printf("📊 状态码: %d\n", statusCode)
	fmt.Printf("📝 内容类型: %s\n", contentType)
	fmt.Printf("📏 内容长度: %d 字符\n", len(content))

	result.StatusCode = statusCode
	result.ContentType = contentType

	// 解析HTML内容
	fmt.Printf("\n📄 步骤4: HTML解析\n")
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		fmt.Printf("❌ HTML解析失败: %v\n", err)
		result.Error = err.Error()
		result.ProcessTime = time.Since(startTime)
		return result, err
	}
	fmt.Printf("✅ HTML解析成功\n")

	// 提取内容
	fmt.Printf("\n🔧 步骤5: 内容提取\n")
	extractedContent, err := w.extractor.ExtractContent(doc, platform)
	if err != nil {
		fmt.Printf("❌ 内容提取失败: %v\n", err)
		result.Error = err.Error()
		result.ProcessTime = time.Since(startTime)
		return result, err
	}
	fmt.Printf("✅ 内容提取成功\n")

	// 填充结果
	result.Title = extractedContent.Title
	result.Content = extractedContent.Content
	result.Author = extractedContent.Author
	result.Description = extractedContent.Description
	result.Keywords = extractedContent.Keywords
	result.PublishedTime = extractedContent.PublishedTime
	result.WordCount = len([]rune(extractedContent.Content))
	result.Images = extractedContent.Images
	result.Videos = extractedContent.Videos
	result.Language = extractedContent.Language
	result.Success = true
	result.ProcessTime = time.Since(startTime)

	// 输出详细的提取结果
	fmt.Printf("\n📊 步骤6: 提取结果汇总\n")
	fmt.Printf("📑 标题: %s\n", result.Title)
	fmt.Printf("👤 作者: %s\n", result.Author)
	fmt.Printf("📅 发布时间: %s\n", result.PublishedTime)
	fmt.Printf("📊 字数统计: %d 字\n", result.WordCount)
	fmt.Printf("🏷️ 关键词数量: %d 个\n", len(result.Keywords))
	fmt.Printf("🏷️ 关键词列表: %v\n", result.Keywords)
	fmt.Printf("🖼️ 图片数量: %d 张\n", len(result.Images))
	fmt.Printf("🎬 视频数量: %d 个\n", len(result.Videos))
	fmt.Printf("🌐 语言: %s\n", result.Language)
	fmt.Printf("⏱️ 处理耗时: %v\n", result.ProcessTime)

	// 显示内容预览
	if len(result.Content) > 0 {
		fmt.Printf("\n📄 内容预览 (前500字符):\n")
		preview := result.Content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("%s\n", preview)
	}

	fmt.Printf("\n🎉 URL爬取完成!\n")
	fmt.Printf("═══════════════════════════════════════\n")

	return result, nil
}

// truncateString 截断字符串到指定长度
// func truncateString(s string, maxLen int) string {
// 	if len(s) <= maxLen {
// 		return s
// 	}
// 	runes := []rune(s)
// 	if len(runes) <= maxLen {
// 		return s
// 	}
// 	return string(runes[:maxLen]) + "..."
// }

// fetchContent 获取网页内容
func (w *WebCrawler) fetchContent(url string) (string, int, string, error) {
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: w.config.Timeout,
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", 0, "", err
	}

	// 设置请求头
	req.Header.Set("User-Agent", w.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	fmt.Printf("🌐 发送HTTP请求，User-Agent: %s\n", w.config.UserAgent)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	fmt.Printf("📊 响应头信息:\n")
	fmt.Printf("   Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("   Content-Encoding: %s\n", resp.Header.Get("Content-Encoding"))
	fmt.Printf("   Content-Length: %s\n", resp.Header.Get("Content-Length"))

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, resp.Header.Get("Content-Type"),
			fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 读取响应内容（Go会自动处理gzip解压）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, resp.Header.Get("Content-Type"), err
	}

	fmt.Printf("📏 原始响应大小: %d 字节\n", len(body))

	// 检查内容长度
	if len(body) > w.config.MaxContentLength {
		return "", resp.StatusCode, resp.Header.Get("Content-Type"),
			fmt.Errorf("content too large: %d bytes", len(body))
	}

	// 字符编码处理
	content := w.decodeContent(body, resp.Header.Get("Content-Type"))

	fmt.Printf("📝 解码后内容长度: %d 字符\n", len(content))

	return content, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

// decodeContent 解码内容，处理字符编码 - 增强版
func (w *WebCrawler) decodeContent(body []byte, contentType string) string {
	fmt.Printf("\n🔤 开始字符编码处理...\n")

	// 检查是否为空内容
	if len(body) == 0 {
		fmt.Printf("⚠️ 响应内容为空\n")
		return ""
	}

	// 检查并移除UTF-8 BOM
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		fmt.Printf("🔤 检测到UTF-8 BOM，已移除\n")
		body = body[3:]
	}

	// 首先尝试直接UTF-8解码
	content := string(body)
	if w.isValidUTF8Enhanced(content) {
		fmt.Printf("✅ 内容为有效UTF-8编码\n")
		return content
	}

	fmt.Printf("⚠️ 内容不是有效UTF-8，启动智能编码检测...\n")

	// 智能编码检测和转换
	if decodedContent := w.smartEncodingDetection(body, contentType); decodedContent != "" {
		fmt.Printf("✅ 智能编码检测成功\n")
		return decodedContent
	}

	// 降级到基础编码处理
	return w.fallbackEncodingHandling(body, contentType)
}

// isValidUTF8Enhanced 增强的UTF-8有效性检查
func (w *WebCrawler) isValidUTF8Enhanced(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}

	// 检查是否包含过多的替换字符或控制字符
	invalidCount := 0
	totalRunes := 0

	for _, r := range s {
		totalRunes++
		if r == utf8.RuneError || r < 32 && r != '\n' && r != '\r' && r != '\t' {
			invalidCount++
		}

		// 如果检查了足够的字符，可以提前判断
		if totalRunes > 1000 {
			break
		}
	}

	// 如果无效字符比例超过5%，认为不是有效UTF-8
	if totalRunes > 0 && float64(invalidCount)/float64(totalRunes) > 0.05 {
		return false
	}

	return true
}

// extractEncodingFromContentType 从Content-Type头部提取编码
func (w *WebCrawler) extractEncodingFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}

	// 查找charset参数
	re := regexp.MustCompile(`charset\s*=\s*([^;\s]+)`)
	matches := re.FindStringSubmatch(strings.ToLower(contentType))
	if len(matches) > 1 {
		return strings.Trim(matches[1], `"' `)
	}

	return ""
}

// detectHTMLEncodingEnhanced 增强的HTML编码检测
func (w *WebCrawler) detectHTMLEncodingEnhanced(content string) string {
	// 只检查前8KB内容，提高性能
	checkContent := content
	if len(content) > 8192 {
		checkContent = content[:8192]
	}

	// 多种模式检测charset
	patterns := []string{
		`<meta\s+charset\s*=\s*["']?([^"'>\s]+)`,
		`<meta[^>]+charset\s*=\s*["']?([^"'>\s]+)`,
		`<meta[^>]+content\s*=\s*["'][^"'>]*charset\s*=\s*([^"'>\s;]+)`,
		`<\?xml[^>]+encoding\s*=\s*["']?([^"'>\s]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(checkContent)
		if len(matches) > 1 {
			encoding := strings.ToLower(strings.TrimSpace(matches[1]))
			if encoding != "" {
				return encoding
			}
		}
	}

	return ""
}

// convertEncoding 编码转换 - 使用专业库版本
func (w *WebCrawler) convertEncoding(body []byte, encoding string) string {
	encoding = strings.ToLower(strings.TrimSpace(encoding))

	fmt.Printf("🔄 使用专业库转换编码: %s\n", encoding)

	// 执行编码转换
	var result []byte
	var err error

	switch encoding {
	case "gbk", "gb2312":
		decoder := simplifiedchinese.GBK.NewDecoder()
		result, _, err = transform.Bytes(decoder, body)
	case "gb18030":
		decoder := simplifiedchinese.GB18030.NewDecoder()
		result, _, err = transform.Bytes(decoder, body)
	case "utf-8", "utf8":
		// 直接返回UTF-8字符串
		return string(body)
	case "latin1", "iso-8859-1":
		decoder := charmap.ISO8859_1.NewDecoder()
		result, _, err = transform.Bytes(decoder, body)
	case "windows-1252":
		decoder := charmap.Windows1252.NewDecoder()
		result, _, err = transform.Bytes(decoder, body)
	default:
		// 尝试自动检测
		return w.autoDetectAndConvert(body)
	}

	if err != nil {
		fmt.Printf("⚠️ 编码转换失败: %v\n", err)
		return w.autoDetectAndConvert(body)
	}

	decoded := string(result)
	fmt.Printf("✅ 编码转换成功，转换后长度: %d 字符\n", len(decoded))
	return decoded
}

// autoDetectAndConvert 自动检测编码并转换
func (w *WebCrawler) autoDetectAndConvert(body []byte) string {
	fmt.Printf("🔍 自动检测字符编码...\n")

	// 使用chardet库进行编码检测
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(body)
	if err != nil {
		fmt.Printf("⚠️ 编码检测失败: %v\n", err)
		return w.cleanInvalidCharsEnhanced(string(body))
	}

	fmt.Printf("🔍 检测到编码: %s (置信度: %d%%)\n", result.Charset, result.Confidence)

	// 如果置信度太低，使用字符清理
	if result.Confidence < 50 {
		fmt.Printf("⚠️ 编码检测置信度太低，使用字符清理\n")
		return w.cleanInvalidCharsEnhanced(string(body))
	}

	// 避免无限递归：直接执行转换而不再调用convertEncoding
	charset := strings.ToLower(strings.TrimSpace(result.Charset))

	switch charset {
	case "gbk", "gb2312":
		decoder := simplifiedchinese.GBK.NewDecoder()
		converted, _, err := transform.Bytes(decoder, body)
		if err == nil {
			fmt.Printf("✅ 自动检测转换成功 (GBK)\n")
			return string(converted)
		}
	case "gb18030":
		decoder := simplifiedchinese.GB18030.NewDecoder()
		converted, _, err := transform.Bytes(decoder, body)
		if err == nil {
			fmt.Printf("✅ 自动检测转换成功 (GB18030)\n")
			return string(converted)
		}
	case "utf-8", "utf8":
		return string(body)
	case "latin1", "iso-8859-1":
		decoder := charmap.ISO8859_1.NewDecoder()
		converted, _, err := transform.Bytes(decoder, body)
		if err == nil {
			fmt.Printf("✅ 自动检测转换成功 (Latin1)\n")
			return string(converted)
		}
	}

	// 如果所有转换都失败，使用字符清理
	fmt.Printf("⚠️ 自动检测编码转换失败，使用字符清理\n")
	return w.cleanInvalidCharsEnhanced(string(body))
}

// isLikelyGBKContent 检查是否可能是GBK编码内容
func (w *WebCrawler) isLikelyGBKContent(body []byte) bool {
	// 简单的启发式检查
	// 检查是否包含常见的GBK字节模式
	for i := 0; i < len(body)-1; i++ {
		b1, b2 := body[i], body[i+1]
		// GBK编码的中文字符通常在这些范围内
		if (b1 >= 0xA1 && b1 <= 0xFE) && (b2 >= 0xA1 && b2 <= 0xFE) {
			return true
		}
	}
	return false
}

// tryGBKConversion 尝试GBK到UTF-8的转换
func (w *WebCrawler) tryGBKConversion(body []byte) string {
	// 这里是简化实现，实际项目中应使用专业的编码转换库
	// 如 golang.org/x/text/encoding/simplifiedchinese

	// 暂时返回原始字符串，但标记了处理意图
	result := string(body)

	// 可以在这里添加更复杂的GBK转UTF-8逻辑
	return result
}

// cleanInvalidChars 清理无效字符
func (w *WebCrawler) cleanInvalidChars(content string) string {
	// 移除无效的UTF-8字符和控制字符
	var result strings.Builder

	for _, r := range content {
		// 保留有效的Unicode字符
		if r != utf8.RuneError && unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	cleaned := result.String()
	fmt.Printf("🧹 字符清理完成，清理前: %d 字符，清理后: %d 字符\n", len(content), len(cleaned))

	return cleaned
}

// CrawlMultipleURLs 批量爬取多个URL
func (w *WebCrawler) CrawlMultipleURLs(urls []string) ([]*CrawlResult, error) {
	results := make([]*CrawlResult, 0, len(urls))

	for _, url := range urls {
		result, err := w.CrawlURL(url)
		if err != nil {
			// 创建错误结果
			result = &CrawlResult{
				URL:       url,
				Success:   false,
				Error:     err.Error(),
				CrawlTime: time.Now(),
			}
		}
		results = append(results, result)

		// 添加延时
		if w.config.DelayMin > 0 {
			time.Sleep(w.config.DelayMin)
		}
	}

	return results, nil
}

// GetSupportedPlatforms 获取支持的平台列表
func (w *WebCrawler) GetSupportedPlatforms() []string {
	return []string{
		"知乎专栏",
		"知乎问答",
		"微信公众号",
		"CSDN博客",
		"哔哩哔哩",
		"简书",
		"掘金",
		"博客园",
		"Github",
		"通用网页",
	}
}

// SetConfig 更新爬虫配置
func (w *WebCrawler) SetConfig(config *CrawlerConfig) {
	w.config = config

	// 重新设置限制
	w.collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       config.DelayMin,
	})

	// 重新设置超时
	w.collector.SetRequestTimeout(config.Timeout)

	// 重新设置允许的域名
	if len(config.AllowedDomains) > 0 {
		w.collector.AllowedDomains = config.AllowedDomains
	}
}

// Close 关闭爬虫，清理资源
func (w *WebCrawler) Close() {
	// 清理资源
	w.collector = nil
	w.extractor = nil
	w.validator = nil
	w.config = nil
}

// smartEncodingDetection 智能编码检测
func (w *WebCrawler) smartEncodingDetection(body []byte, contentType string) string {
	fmt.Printf("🔍 执行智能编码检测...\n")

	// 1. 字节模式检测
	if encoding := w.detectEncodingByBytePattern(body); encoding != "" {
		fmt.Printf("🔤 字节模式检测到编码: %s\n", encoding)
		if converted := w.convertEncodingEnhanced(body, encoding); converted != "" {
			return converted
		}
	}

	// 2. Content-Type检测（优先级高）
	if encoding := w.extractEncodingFromContentType(contentType); encoding != "" {
		fmt.Printf("🔤 Content-Type检测到编码: %s\n", encoding)
		if converted := w.convertEncodingEnhanced(body, encoding); converted != "" {
			return converted
		}
	}

	// 3. HTML meta标签检测
	if strings.Contains(strings.ToLower(contentType), "html") {
		if encoding := w.detectHTMLEncodingEnhanced(string(body)); encoding != "" {
			fmt.Printf("🔤 HTML meta检测到编码: %s\n", encoding)
			if converted := w.convertEncodingEnhanced(body, encoding); converted != "" {
				return converted
			}
		}
	}

	// 4. 统计分析检测
	if encoding := w.detectEncodingByStatistics(body); encoding != "" {
		fmt.Printf("🔤 统计分析检测到编码: %s\n", encoding)
		if converted := w.convertEncodingEnhanced(body, encoding); converted != "" {
			return converted
		}
	}

	return ""
}

// detectEncodingByBytePattern 基于字节模式检测编码
func (w *WebCrawler) detectEncodingByBytePattern(body []byte) string {
	if len(body) < 100 {
		return ""
	}

	// 检查常见中文编码的字节模式
	sample := body
	if len(body) > 2048 {
		sample = body[:2048] // 只检查前2KB
	}

	// GBK/GB2312 特征检测
	gbkCount := 0
	for i := 0; i < len(sample)-1; i++ {
		b1, b2 := sample[i], sample[i+1]
		// GBK范围: A1A1-FEFE
		if (b1 >= 0xA1 && b1 <= 0xFE) && (b2 >= 0xA1 && b2 <= 0xFE) {
			gbkCount++
		}
	}

	if gbkCount > 10 { // 如果发现较多GBK模式
		return "gbk"
	}

	// Big5 特征检测
	big5Count := 0
	for i := 0; i < len(sample)-1; i++ {
		b1, b2 := sample[i], sample[i+1]
		// Big5范围
		if (b1 >= 0xA1 && b1 <= 0xF9) && ((b2 >= 0x40 && b2 <= 0x7E) || (b2 >= 0xA1 && b2 <= 0xFE)) {
			big5Count++
		}
	}

	if big5Count > 10 {
		return "big5"
	}

	return ""
}

// detectEncodingByStatistics 基于统计分析检测编码
func (w *WebCrawler) detectEncodingByStatistics(body []byte) string {
	if len(body) < 100 {
		return ""
	}

	// 统计高位字节的分布
	highByteCount := 0
	for _, b := range body {
		if b > 127 {
			highByteCount++
		}
	}

	highByteRatio := float64(highByteCount) / float64(len(body))

	// 如果高位字节比例较高，可能是中文编码
	if highByteRatio > 0.3 {
		// 进一步分析字节分布
		distribution := make(map[byte]int)
		for _, b := range body {
			if b > 127 {
				distribution[b]++
			}
		}

		// GBK编码的字节分布特征
		gbkCharRange := 0
		for b := range distribution {
			if b >= 0xA1 && b <= 0xFE {
				gbkCharRange++
			}
		}

		if gbkCharRange > 20 {
			return "gbk"
		}
	}

	return ""
}

// convertEncodingEnhanced 增强的编码转换
func (w *WebCrawler) convertEncodingEnhanced(body []byte, encoding string) string {
	fmt.Printf("🔄 尝试转换编码: %s\n", encoding)

	// 标准化编码名称
	encoding = w.normalizeEncodingName(encoding)

	// 执行转换
	converted := w.convertEncoding(body, encoding)
	if converted == "" {
		return ""
	}

	// 验证转换质量
	if w.validateConversionQuality(converted) {
		fmt.Printf("✅ 编码转换质量验证通过\n")
		return converted
	}

	fmt.Printf("⚠️ 编码转换质量验证失败\n")
	return ""
}

// normalizeEncodingName 标准化编码名称
func (w *WebCrawler) normalizeEncodingName(encoding string) string {
	encoding = strings.ToLower(strings.TrimSpace(encoding))

	// 编码名称映射
	encodingMap := map[string]string{
		"gb2312":     "gbk",
		"gb18030":    "gbk",
		"chinese":    "gbk",
		"utf8":       "utf-8",
		"iso-8859-1": "latin1",
	}

	if normalized, exists := encodingMap[encoding]; exists {
		return normalized
	}

	return encoding
}

// validateConversionQuality 验证转换质量
func (w *WebCrawler) validateConversionQuality(content string) bool {
	if len(content) == 0 {
		return false
	}

	// 1. UTF-8有效性检查
	if !w.isValidUTF8Enhanced(content) {
		return false
	}

	// 2. 乱码字符比例检查
	garbledCount := 0
	totalRunes := 0

	for _, r := range content {
		totalRunes++
		if r == '\uFFFD' || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			garbledCount++
		}
	}

	if totalRunes == 0 {
		return false
	}

	garbledRatio := float64(garbledCount) / float64(totalRunes)

	// 乱码比例不能超过5%
	if garbledRatio > 0.05 {
		fmt.Printf("⚠️ 乱码比例过高: %.2f%%\n", garbledRatio*100)
		return false
	}

	// 3. 内容完整性检查
	if len(content) < 100 { // 内容太短可能转换有问题
		return false
	}

	return true
}

// fallbackEncodingHandling 降级编码处理
func (w *WebCrawler) fallbackEncodingHandling(body []byte, contentType string) string {
	fmt.Printf("🔄 执行降级编码处理...\n")

	// 1. 尝试常见编码的穷举转换
	commonEncodings := []string{"gbk", "gb2312", "big5", "utf-8", "latin1"}
	for _, encoding := range commonEncodings {
		if converted := w.convertEncoding(body, encoding); converted != "" {
			if w.isValidUTF8Enhanced(converted) {
				garbledCount := w.countGarbledChars(converted)
				if garbledCount < len(converted)/20 { // 乱码字符不超过5%
					fmt.Printf("✅ 降级处理成功，使用编码: %s\n", encoding)
					return converted
				}
			}
		}
	}

	// 2. 最终字符清理
	fmt.Printf("⚠️ 所有编码转换失败，执行字符清理...\n")
	content := string(body)
	return w.cleanInvalidCharsEnhanced(content)
}

// countGarbledChars 统计乱码字符数量
func (w *WebCrawler) countGarbledChars(content string) int {
	garbledCount := 0
	for _, r := range content {
		if r == '\uFFFD' || (r < 32 && r != '\n' && r != '\r' && r != '\t' && r != ' ') {
			garbledCount++
		}
	}
	return garbledCount
}

// cleanInvalidCharsEnhanced 增强的字符清理
func (w *WebCrawler) cleanInvalidCharsEnhanced(content string) string {
	if len(content) == 0 {
		return ""
	}

	// 使用strings.Builder提高性能
	var builder strings.Builder
	builder.Grow(len(content)) // 预分配容量

	for _, r := range content {
		// 保留有效字符
		if w.isValidChar(r) {
			builder.WriteRune(r)
		} else {
			// 替换无效字符为空格（而不是直接删除）
			builder.WriteRune(' ')
		}
	}

	result := builder.String()

	// 清理多余的空格
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	fmt.Printf("🧹 字符清理完成，处理后长度: %d\n", len(result))
	return result
}

// isValidChar 判断字符是否有效
func (w *WebCrawler) isValidChar(r rune) bool {
	// 1. 基本ASCII字符
	if r <= 127 && r >= 32 {
		return true
	}

	// 2. 基本控制字符
	if r == '\n' || r == '\r' || r == '\t' {
		return true
	}

	// 3. Unicode字符范围
	if r >= 0x80 && r != '\uFFFD' {
		// 排除一些问题字符范围
		if r >= 0xD800 && r <= 0xDFFF { // UTF-16代理对
			return false
		}
		if r >= 0xFDD0 && r <= 0xFDEF { // 非字符
			return false
		}
		return true
	}

	return false
}

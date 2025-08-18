package crawler

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/publicsuffix"
)

// EnhancedXPathCrawler 增强版 XPath 爬虫，具备强大反爬能力
type EnhancedXPathCrawler struct {
	client        *http.Client
	cookieJar     *cookiejar.Jar
	extractor     *XPathExtractor
	validator     *URLValidator
	detector      *EncodingDetector
	config        *EnhancedCrawlerConfig
	userAgents    []string
	randomSeed    *rand.Rand
	requestCount  int
	sessionCookie map[string]string
}

// EnhancedCrawlerConfig 增强爬虫配置
type EnhancedCrawlerConfig struct {
	// 基础配置
	Timeout          time.Duration `json:"timeout"`
	MaxContentLength int           `json:"max_content_length"`
	EnableDebug      bool          `json:"enable_debug"`
	RetryCount       int           `json:"retry_count"`

	// 反爬配置
	MinDelay              time.Duration `json:"min_delay"`              // 最小请求间隔
	MaxDelay              time.Duration `json:"max_delay"`              // 最大请求间隔
	EnableRandomDelay     bool          `json:"enable_random_delay"`    // 启用随机延迟
	EnableUserAgentRotate bool          `json:"enable_ua_rotate"`       // 启用 UA 轮换
	EnableTLSFingerprint  bool          `json:"enable_tls_fingerprint"` // 启用 TLS 指纹伪装
	EnableSessionReuse    bool          `json:"enable_session_reuse"`   // 启用会话复用
	MaxConnectionPool     int           `json:"max_connection_pool"`    // 最大连接池
	EnableRealBehavior    bool          `json:"enable_real_behavior"`   // 启用真实行为模拟

	// 代理配置
	ProxyURLs   []string `json:"proxy_urls"`
	EnableProxy bool     `json:"enable_proxy"`
	ProxyRotate bool     `json:"proxy_rotate"`
}

// NewEnhancedXPathCrawler 创建增强版 XPath 爬虫
func NewEnhancedXPathCrawler(config *EnhancedCrawlerConfig) *EnhancedXPathCrawler {
	if config == nil {
		config = DefaultEnhancedCrawlerConfig()
	}

	// 创建 Cookie Jar
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// 创建随机种子
	randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

	crawler := &EnhancedXPathCrawler{
		cookieJar:     jar,
		extractor:     NewXPathExtractor(),
		validator:     NewURLValidator(),
		detector:      NewEncodingDetector(),
		config:        config,
		randomSeed:    randomSeed,
		sessionCookie: make(map[string]string),
		userAgents:    getRealisticUserAgents(),
	}

	// 初始化 HTTP 客户端
	crawler.initHTTPClient()

	return crawler
}

// DefaultEnhancedCrawlerConfig 默认增强爬虫配置
func DefaultEnhancedCrawlerConfig() *EnhancedCrawlerConfig {
	return &EnhancedCrawlerConfig{
		Timeout:               30 * time.Second,
		MaxContentLength:      5 * 1024 * 1024, // 5MB
		EnableDebug:           true,
		RetryCount:            3,
		MinDelay:              500 * time.Millisecond,
		MaxDelay:              2 * time.Second,
		EnableRandomDelay:     true,
		EnableUserAgentRotate: true,
		EnableTLSFingerprint:  true,
		EnableSessionReuse:    true,
		MaxConnectionPool:     20,
		EnableRealBehavior:    true,
		EnableProxy:           false,
		ProxyRotate:           false,
	}
}

// initHTTPClient 初始化 HTTP 客户端，模拟真实浏览器
func (c *EnhancedXPathCrawler) initHTTPClient() {
	// 创建自定义 Transport，模拟真实浏览器的 TLS 配置
	transport := &http.Transport{
		MaxIdleConns:        c.config.MaxConnectionPool,
		MaxIdleConnsPerHost: 6, // Chrome 默认每个域名6个连接
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false, // 启用压缩
		ForceAttemptHTTP2:   true,  // 强制尝试 HTTP/2
	}

	// 配置 TLS 以模拟真实浏览器
	if c.config.EnableTLSFingerprint {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
			InsecureSkipVerify: false,
			// 模拟 Chrome 的 TLS 配置
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
			PreferServerCipherSuites: false,
		}
	}

	// 配置代理（如果启用）
	if c.config.EnableProxy && len(c.config.ProxyURLs) > 0 {
		proxyURL, _ := url.Parse(c.config.ProxyURLs[0])
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	c.client = &http.Client{
		Transport: transport,
		Jar:       c.cookieJar,
		Timeout:   c.config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 允许重定向，但限制次数
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}

// CrawlURL 爬取指定 URL（增强版）
func (c *EnhancedXPathCrawler) CrawlURL(url string) (*XPathCrawlResult, error) {
	startTime := time.Now()
	c.requestCount++

	if c.config.EnableDebug {
		fmt.Printf("\n🚀 增强版 XPath 爬虫开始处理 [请求 #%d]\n", c.requestCount)
		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("🎯 目标URL: %s\n", url)
		fmt.Printf("⏰ 开始时间: %s\n", startTime.Format("2006-01-02 15:04:05"))
	}

	// 1. URL验证
	if !c.validator.IsValidURL(url) {
		return c.buildErrorResult(url, "invalid URL", startTime)
	}

	// 2. 平台检测
	platform := c.validator.DetectPlatform(url)
	if platform == PlatformUnsupported {
		return c.buildErrorResult(url, "unsupported platform", startTime)
	}

	if c.config.EnableDebug {
		fmt.Printf("✅ 检测到平台: %s\n", platform.String())
	}

	// 3. 实施反爬策略
	if err := c.applyAntiCrawlerStrategies(url); err != nil {
		if c.config.EnableDebug {
			fmt.Printf("⚠️ 反爬策略应用失败: %v\n", err)
		}
	}

	// 4. 执行真实浏览器模拟请求
	content, contentType, statusCode, err := c.fetchContentWithEnhancedStrategies(url)
	if err != nil {
		return c.buildErrorResult(url, err.Error(), startTime)
	}

	if c.config.EnableDebug {
		fmt.Printf("✅ 内容获取成功: %d 字节\n", len(content))
		fmt.Printf("📊 状态码: %d, 内容类型: %s\n", statusCode, contentType)
	}

	// 5. 编码检测和转换
	decodedContent, encoding := c.detector.DetectAndDecode([]byte(content), contentType)

	// 6. HTML 解析
	doc, err := htmlquery.Parse(strings.NewReader(decodedContent))
	if err != nil {
		return c.buildErrorResult(url, fmt.Sprintf("HTML parse failed: %v", err), startTime)
	}

	// 7. XPath 内容提取
	extractedContent, err := c.extractor.ExtractContent(doc, platform)
	if err != nil {
		return c.buildErrorResult(url, fmt.Sprintf("content extraction failed: %v", err), startTime)
	}

	// 8. 构建成功结果
	result := &XPathCrawlResult{
		URL:           url,
		Title:         extractedContent.Title,
		Content:       extractedContent.Content,
		Author:        extractedContent.Author,
		Description:   extractedContent.Description,
		Keywords:      extractedContent.Keywords,
		PublishedTime: extractedContent.PublishedTime,
		WordCount:     extractedContent.WordCount,
		Images:        extractedContent.Images,
		Videos:        extractedContent.Videos,
		Language:      extractedContent.Language,
		Encoding:      encoding,
		StatusCode:    statusCode,
		ContentType:   contentType,
		Platform:      platform.String(),
		CrawlTime:     startTime,
		ProcessTime:   time.Since(startTime),
		Success:       true,
		MatchedXPaths: make(map[string]string),
		ExtractionLog: []string{},
	}

	if c.config.EnableDebug {
		c.printSuccessResult(result)
	}

	return result, nil
}

// applyAntiCrawlerStrategies 应用反爬策略
func (c *EnhancedXPathCrawler) applyAntiCrawlerStrategies(targetURL string) error {
	// 1. 随机延迟
	if c.config.EnableRandomDelay && c.requestCount > 1 {
		delay := c.calculateRandomDelay()
		if c.config.EnableDebug {
			fmt.Printf("⏳ 应用随机延迟: %v\n", delay)
		}
		time.Sleep(delay)
	}

	// 2. 预热请求（模拟用户行为）
	if c.config.EnableRealBehavior {
		if err := c.performWarmupRequests(targetURL); err != nil {
			return fmt.Errorf("warmup requests failed: %w", err)
		}
	}

	// 3. 代理轮换
	if c.config.EnableProxy && c.config.ProxyRotate && len(c.config.ProxyURLs) > 1 {
		c.rotateProxy()
	}

	return nil
}

// fetchContentWithEnhancedStrategies 使用增强策略获取内容
func (c *EnhancedXPathCrawler) fetchContentWithEnhancedStrategies(url string) (string, string, int, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			if c.config.EnableDebug {
				fmt.Printf("🔄 重试第 %d 次: %s\n", attempt, url)
			}
			// 重试时应用不同的策略
			c.adjustStrategyForRetry(attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		content, contentType, statusCode, err := c.fetchContentWithRealBrowserBehavior(url)
		if err == nil && statusCode == http.StatusOK {
			return content, contentType, statusCode, nil
		}

		lastErr = err
		if c.config.EnableDebug {
			fmt.Printf("⚠️ 第 %d 次尝试失败: %v (状态码: %d)\n", attempt+1, err, statusCode)
		}
	}

	return "", "", 0, fmt.Errorf("all retries failed: %w", lastErr)
}

// fetchContentWithRealBrowserBehavior 模拟真实浏览器行为获取内容
func (c *EnhancedXPathCrawler) fetchContentWithRealBrowserBehavior(url string) (string, string, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("create request failed: %w", err)
	}

	// 设置真实浏览器请求头
	c.setRealisticHeaders(req, url)

	if c.config.EnableDebug {
		fmt.Printf("📡 发送增强版 HTTP 请求\n")
		fmt.Printf("   User-Agent: %s\n", req.Header.Get("User-Agent"))
		fmt.Printf("   Referer: %s\n", req.Header.Get("Referer"))
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if c.config.EnableDebug {
		fmt.Printf("📨 接收 HTTP 响应\n")
		fmt.Printf("   状态码: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Printf("   Content-Type: %s\n", resp.Header.Get("Content-Type"))
		fmt.Printf("   Content-Length: %s\n", resp.Header.Get("Content-Length"))
		fmt.Printf("   Server: %s\n", resp.Header.Get("Server"))
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", resp.StatusCode, fmt.Errorf("read response body failed: %w", err)
	}

	// 检查内容长度
	if len(body) > c.config.MaxContentLength {
		return "", "", resp.StatusCode, fmt.Errorf("content too large: %d bytes", len(body))
	}

	return string(body), resp.Header.Get("Content-Type"), resp.StatusCode, nil
}

// setRealisticHeaders 设置真实浏览器请求头
func (c *EnhancedXPathCrawler) setRealisticHeaders(req *http.Request, targetURL string) {
	// 1. User-Agent (轮换或固定)
	userAgent := c.getRandomUserAgent()
	req.Header.Set("User-Agent", userAgent)

	// 2. Accept 系列
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	// 3. 缓存控制
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	// 4. 连接信息
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// 5. 安全策略
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	// 6. Chrome 特有头部
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)

	// 7. 模拟 Referer
	if parsedURL, err := url.Parse(targetURL); err == nil {
		baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
		req.Header.Set("Referer", baseURL)
	}

	// 8. DNT (Do Not Track)
	req.Header.Set("DNT", "1")
}

// getRandomUserAgent 获取随机 User-Agent
func (c *EnhancedXPathCrawler) getRandomUserAgent() string {
	if !c.config.EnableUserAgentRotate || len(c.userAgents) == 0 {
		return c.userAgents[0] // 返回默认的第一个
	}

	index := c.randomSeed.Intn(len(c.userAgents))
	return c.userAgents[index]
}

// calculateRandomDelay 计算随机延迟
func (c *EnhancedXPathCrawler) calculateRandomDelay() time.Duration {
	minMs := int64(c.config.MinDelay / time.Millisecond)
	maxMs := int64(c.config.MaxDelay / time.Millisecond)

	if minMs >= maxMs {
		return c.config.MinDelay
	}

	randomMs := c.randomSeed.Int63n(maxMs-minMs) + minMs
	return time.Duration(randomMs) * time.Millisecond
}

// performWarmupRequests 执行预热请求，模拟真实用户行为
func (c *EnhancedXPathCrawler) performWarmupRequests(targetURL string) error {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// 1. 请求网站首页 (模拟用户先访问首页)
	if c.config.EnableDebug {
		fmt.Printf("🔥 执行预热请求: %s\n", baseURL)
	}

	req, _ := http.NewRequest("GET", baseURL, nil)
	c.setRealisticHeaders(req, baseURL)

	resp, err := c.client.Do(req)
	if err == nil {
		resp.Body.Close()
		// 短暂延迟，模拟用户浏览行为
		time.Sleep(time.Duration(500+c.randomSeed.Intn(1000)) * time.Millisecond)
	}

	return nil
}

// adjustStrategyForRetry 为重试调整策略
func (c *EnhancedXPathCrawler) adjustStrategyForRetry(attempt int) {
	if c.config.EnableDebug {
		fmt.Printf("🔧 调整重试策略 (尝试 #%d)\n", attempt)
	}

	// 重试时切换 User-Agent
	if c.config.EnableUserAgentRotate {
		// 强制轮换到下一个 User-Agent
	}

	// 重试时增加延迟
	extraDelay := time.Duration(attempt) * 500 * time.Millisecond
	time.Sleep(extraDelay)
}

// rotateProxy 轮换代理
func (c *EnhancedXPathCrawler) rotateProxy() {
	if len(c.config.ProxyURLs) <= 1 {
		return
	}

	index := c.randomSeed.Intn(len(c.config.ProxyURLs))
	proxyURL, err := url.Parse(c.config.ProxyURLs[index])
	if err != nil {
		return
	}

	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.Proxy = http.ProxyURL(proxyURL)
		if c.config.EnableDebug {
			fmt.Printf("🔄 切换代理: %s\n", c.config.ProxyURLs[index])
		}
	}
}

// getRealisticUserAgents 获取真实的 User-Agent 列表
func getRealisticUserAgents() []string {
	return []string{
		// Chrome (最新版本)
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",

		// Firefox (最新版本)
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/119.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/119.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0",

		// Safari
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",

		// Edge
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 Edg/119.0.0.0",
	}
}

// 辅助方法
func (c *EnhancedXPathCrawler) buildErrorResult(url, errorMsg string, startTime time.Time) (*XPathCrawlResult, error) {
	return &XPathCrawlResult{
		URL:         url,
		Success:     false,
		Error:       errorMsg,
		CrawlTime:   startTime,
		ProcessTime: time.Since(startTime),
	}, fmt.Errorf(errorMsg)
}

func (c *EnhancedXPathCrawler) printSuccessResult(result *XPathCrawlResult) {
	fmt.Printf("\n🎉 增强版爬取成功完成!\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("📄 标题: %s\n", getFieldStatus(result.Title))
	fmt.Printf("👤 作者: %s\n", getFieldStatus(result.Author))
	fmt.Printf("📝 内容长度: %d 字符\n", len(result.Content))
	fmt.Printf("🏷️ 关键词: %d 个\n", len(result.Keywords))
	fmt.Printf("🖼️ 图片: %d 张\n", len(result.Images))
	fmt.Printf("⏱️ 处理时间: %v\n", result.ProcessTime)
	fmt.Printf("🌐 平台: %s\n", result.Platform)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
}

func getFieldStatus(field string) string {
	if field != "" {
		if len(field) > 60 {
			return field[:60] + "..."
		}
		return field
	}
	return "❌ 未提取"
}

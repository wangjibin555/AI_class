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

// NewXPathCrawler 创建增强版 XPath 爬虫实例
func NewXPathCrawler(config *XPathCrawlerConfig) *XPathCrawler {
	if config == nil {
		config = DefaultXPathCrawlerConfig()
	}

	// 创建 Cookie Jar 用于会话管理
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// 创建随机种子
	randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

	crawler := &XPathCrawler{
		cookieJar:    jar,
		extractor:    NewXPathExtractor(),
		validator:    NewURLValidator(),
		detector:     NewEncodingDetector(),
		config:       config,
		randomSeed:   randomSeed,
		userAgents:   getRealisticUserAgentList(),
		requestCount: 0,
	}

	// 初始化增强版 HTTP 客户端
	crawler.initEnhancedHTTPClient()

	return crawler
}

// initEnhancedHTTPClient 初始化增强版 HTTP 客户端，模拟真实浏览器
func (c *XPathCrawler) initEnhancedHTTPClient() {
	// 创建自定义 Transport，模拟真实浏览器行为
	transport := &http.Transport{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 6, // Chrome 默认每个域名6个连接
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false, // 启用压缩
		ForceAttemptHTTP2:   true,  // 强制尝试 HTTP/2
	}

	// 配置 TLS 以模拟真实浏览器指纹
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

// CrawlURL 爬取指定URL（增强版实现）
func (c *XPathCrawler) CrawlURL(url string) (*XPathCrawlResult, error) {
	startTime := time.Now()
	c.requestCount++

	if c.config.EnableDebug {
		fmt.Printf("\n🕷️ 增强版 XPath 爬虫开始处理 [请求 #%d]\n", c.requestCount)
		fmt.Printf("═══════════════════════════════════════\n")
		fmt.Printf("🎯 目标URL: %s\n", url)
		fmt.Printf("⏰ 开始时间: %s\n", startTime.Format("2006-01-02 15:04:05"))
	}

	// 1. URL验证
	if c.config.EnableDebug {
		fmt.Printf("\n📋 步骤1: URL验证\n")
	}

	if !c.validator.IsValidURL(url) {
		if c.config.EnableDebug {
			fmt.Printf("❌ URL验证失败: %s\n", url)
		}
		return &XPathCrawlResult{
			URL:         url,
			Success:     false,
			Error:       "invalid URL",
			CrawlTime:   startTime,
			ProcessTime: time.Since(startTime),
		}, fmt.Errorf("invalid URL: %s", url)
	}

	if c.config.EnableDebug {
		fmt.Printf("✅ URL验证通过\n")
	}

	// 2. 检测平台类型
	if c.config.EnableDebug {
		fmt.Printf("\n🔍 步骤2: 平台检测\n")
	}

	platform := c.validator.DetectPlatform(url)
	if platform == PlatformUnsupported {
		if c.config.EnableDebug {
			fmt.Printf("❌ 不支持的平台: %s\n", url)
		}
		return &XPathCrawlResult{
			URL:         url,
			Success:     false,
			Error:       "unsupported platform",
			Platform:    platform.String(),
			CrawlTime:   startTime,
			ProcessTime: time.Since(startTime),
		}, fmt.Errorf("unsupported platform for URL: %s", url)
	}

	if c.config.EnableDebug {
		fmt.Printf("✅ 检测到平台: %s\n", platform.String())
	}

	// 3. 应用反爬策略
	if c.config.EnableDebug {
		fmt.Printf("\n🛡️ 步骤3: 应用反爬策略\n")
	}
	c.applyAntiCrawlerStrategies(url)

	// 4. 获取网页内容（使用增强策略）
	if c.config.EnableDebug {
		fmt.Printf("\n🌐 步骤4: 增强版内容获取\n")
	}

	content, contentType, statusCode, err := c.fetchContentWithEnhancedStrategies(url)
	if err != nil {
		if c.config.EnableDebug {
			fmt.Printf("❌ 内容获取失败: %v\n", err)
		}
		return &XPathCrawlResult{
			URL:         url,
			Success:     false,
			Error:       err.Error(),
			Platform:    platform.String(),
			StatusCode:  statusCode,
			CrawlTime:   startTime,
			ProcessTime: time.Since(startTime),
		}, err
	}

	if c.config.EnableDebug {
		fmt.Printf("✅ 内容获取成功\n")
		fmt.Printf("📊 状态码: %d\n", statusCode)
		fmt.Printf("📝 内容类型: %s\n", contentType)
		fmt.Printf("📏 原始内容长度: %d 字节\n", len(content))
	}

	// 5. 编码检测和转换
	if c.config.EnableDebug {
		fmt.Printf("\n🔤 步骤5: 编码检测和转换\n")
	}

	decodedContent, encoding := c.detector.DetectAndDecode([]byte(content), contentType)
	if c.config.EnableDebug {
		fmt.Printf("✅ 编码检测完成\n")
		fmt.Printf("📝 检测到编码: %s\n", encoding)
		fmt.Printf("📏 转换后长度: %d 字符\n", len(decodedContent))
	}

	// 6. 解析HTML文档
	if c.config.EnableDebug {
		fmt.Printf("\n📄 步骤6: HTML文档解析\n")
	}

	doc, err := htmlquery.Parse(strings.NewReader(decodedContent))
	if err != nil {
		if c.config.EnableDebug {
			fmt.Printf("❌ HTML解析失败: %v\n", err)
		}
		return &XPathCrawlResult{
			URL:         url,
			Success:     false,
			Error:       fmt.Sprintf("parse HTML failed: %v", err),
			Platform:    platform.String(),
			StatusCode:  statusCode,
			ContentType: contentType,
			Encoding:    encoding,
			CrawlTime:   startTime,
			ProcessTime: time.Since(startTime),
		}, err
	}

	if c.config.EnableDebug {
		fmt.Printf("✅ HTML解析成功\n")
	}

	// 7. XPath内容提取
	if c.config.EnableDebug {
		fmt.Printf("\n🎯 步骤7: XPath内容提取\n")
	}

	extractedContent, err := c.extractor.ExtractContent(doc, platform)
	if err != nil {
		if c.config.EnableDebug {
			fmt.Printf("❌ 内容提取失败: %v\n", err)
		}
		return &XPathCrawlResult{
			URL:         url,
			Success:     false,
			Error:       fmt.Sprintf("extract content failed: %v", err),
			Platform:    platform.String(),
			StatusCode:  statusCode,
			ContentType: contentType,
			Encoding:    encoding,
			CrawlTime:   startTime,
			ProcessTime: time.Since(startTime),
		}, err
	}

	// 8. 构建最终结果
	if c.config.EnableDebug {
		fmt.Printf("\n📋 步骤8: 结果构建\n")
	}

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
		fmt.Printf("✅ 结果构建完成\n")
		fmt.Printf("\n📊 增强版 XPath 爬取结果汇总:\n")
		fmt.Printf("═══════════════════════════════════════\n")
		fmt.Printf("🎯 目标URL: %s\n", url)
		fmt.Printf("🏷️ 平台类型: %s\n", platform.String())
		fmt.Printf("📝 标题: %s\n", getXPathFieldStatus(result.Title))
		fmt.Printf("👤 作者: %s\n", getXPathFieldStatus(result.Author))
		fmt.Printf("📄 内容长度: %d 字符\n", result.WordCount)
		fmt.Printf("🏷️ 关键词数量: %d 个\n", len(result.Keywords))
		if len(result.Keywords) > 0 {
			fmt.Printf("   关键词: %v\n", result.Keywords)
		}
		fmt.Printf("📅 发布时间: %s\n", getXPathFieldStatus(result.PublishedTime))
		fmt.Printf("🖼️ 图片数量: %d 张\n", len(result.Images))
		fmt.Printf("📺 视频数量: %d 个\n", len(result.Videos))
		fmt.Printf("🔤 字符编码: %s\n", encoding)
		fmt.Printf("🌐 HTTP状态: %d\n", statusCode)
		fmt.Printf("⏱️ 处理耗时: %v\n", result.ProcessTime)
		fmt.Printf("✅ 爬取状态: 成功\n")
		fmt.Printf("═══════════════════════════════════════\n")
	}

	return result, nil
}

// applyAntiCrawlerStrategies 应用反爬策略
func (c *XPathCrawler) applyAntiCrawlerStrategies(targetURL string) {
	// 1. 随机延迟（模拟人类行为）
	if c.requestCount > 1 {
		delay := c.calculateRandomDelay()
		if c.config.EnableDebug {
			fmt.Printf("⏳ 应用随机延迟: %v\n", delay)
		}
		time.Sleep(delay)
	}

	// 2. 预热请求（模拟用户先访问首页的行为）
	if c.config.EnableDebug {
		fmt.Printf("🔥 执行预热请求\n")
	}
	c.performWarmupRequest(targetURL)
}

// calculateRandomDelay 计算随机延迟
func (c *XPathCrawler) calculateRandomDelay() time.Duration {
	minMs := int64(500)  // 最小 500ms
	maxMs := int64(2000) // 最大 2s

	randomMs := c.randomSeed.Int63n(maxMs-minMs) + minMs
	return time.Duration(randomMs) * time.Millisecond
}

// performWarmupRequest 执行预热请求
func (c *XPathCrawler) performWarmupRequest(targetURL string) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	req, _ := http.NewRequest("GET", baseURL, nil)
	c.setEnhancedHeaders(req, baseURL)

	resp, err := c.client.Do(req)
	if err == nil {
		resp.Body.Close()
		// 短暂延迟，模拟用户浏览行为
		time.Sleep(time.Duration(500+c.randomSeed.Intn(1000)) * time.Millisecond)
	}
}

// fetchContentWithEnhancedStrategies 使用增强策略获取内容
func (c *XPathCrawler) fetchContentWithEnhancedStrategies(url string) (string, string, int, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			if c.config.EnableDebug {
				fmt.Printf("🔄 重试第 %d 次: %s\n", attempt, url)
			}
			// 重试时增加延迟
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
func (c *XPathCrawler) fetchContentWithRealBrowserBehavior(url string) (string, string, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("create request failed: %w", err)
	}

	// 设置增强版浏览器请求头
	c.setEnhancedHeaders(req, url)

	if c.config.EnableDebug {
		fmt.Printf("📡 发送增强版 HTTP 请求\n")
		fmt.Printf("   User-Agent: %s\n", truncateString(req.Header.Get("User-Agent"), 60))
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

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", resp.Header.Get("Content-Type"), resp.StatusCode,
			fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.Header.Get("Content-Type"), resp.StatusCode,
			fmt.Errorf("read response body failed: %w", err)
	}

	// 检查内容长度限制
	if len(body) > c.config.MaxContentLength {
		return "", resp.Header.Get("Content-Type"), resp.StatusCode,
			fmt.Errorf("content too large: %d bytes", len(body))
	}

	return string(body), resp.Header.Get("Content-Type"), resp.StatusCode, nil
}

// setEnhancedHeaders 设置增强版浏览器请求头
func (c *XPathCrawler) setEnhancedHeaders(req *http.Request, targetURL string) {
	// 1. 轮换 User-Agent
	userAgent := c.getRotatingUserAgent()
	req.Header.Set("User-Agent", userAgent)

	// 2. 完整的 Accept 系列
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	// 3. 缓存控制
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	// 4. 连接信息
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// 5. 现代浏览器安全策略
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	// 6. Chrome 特有头部
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)

	// 7. 模拟真实 Referer
	if parsedURL, err := url.Parse(targetURL); err == nil {
		baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
		req.Header.Set("Referer", baseURL)
	}

	// 8. DNT (Do Not Track)
	req.Header.Set("DNT", "1")
}

// getRotatingUserAgent 获取轮换的 User-Agent
func (c *XPathCrawler) getRotatingUserAgent() string {
	if len(c.userAgents) == 0 {
		return c.config.UserAgent
	}

	index := c.randomSeed.Intn(len(c.userAgents))
	return c.userAgents[index]
}

// getRealisticUserAgentList 获取真实的 User-Agent 列表
func getRealisticUserAgentList() []string {
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

// 保持向后兼容的原有方法
func (c *XPathCrawler) fetchContentWithRetry(url string) (string, string, int, error) {
	return c.fetchContentWithEnhancedStrategies(url)
}

func (c *XPathCrawler) fetchContent(url string) (string, string, int, error) {
	return c.fetchContentWithRealBrowserBehavior(url)
}

// CrawlMultipleURLs 批量爬取URL
func (c *XPathCrawler) CrawlMultipleURLs(urls []string) ([]*XPathCrawlResult, error) {
	if c.config.EnableDebug {
		fmt.Printf("\n🕷️ 开始批量增强版 XPath 爬取\n")
		fmt.Printf("📊 URL数量: %d\n", len(urls))
	}

	results := make([]*XPathCrawlResult, len(urls))

	if c.config.EnableConcurrent && len(urls) > 1 {
		// 并发处理
		return c.crawlConcurrent(urls)
	} else {
		// 顺序处理
		for i, url := range urls {
			result, err := c.CrawlURL(url)
			if err != nil {
				result = &XPathCrawlResult{
					URL:         url,
					Success:     false,
					Error:       err.Error(),
					CrawlTime:   time.Now(),
					ProcessTime: 0,
				}
			}
			results[i] = result

			if c.config.EnableDebug {
				fmt.Printf("📋 进度: %d/%d - %s\n", i+1, len(urls),
					map[bool]string{true: "✅", false: "❌"}[result.Success])
			}

			// 批量处理时添加延迟
			if i < len(urls)-1 {
				delay := c.calculateRandomDelay()
				time.Sleep(delay)
			}
		}
	}

	return results, nil
}

// crawlConcurrent 并发爬取
func (c *XPathCrawler) crawlConcurrent(urls []string) ([]*XPathCrawlResult, error) {
	// 限制并发数
	semaphore := make(chan struct{}, c.config.MaxConcurrency)
	results := make([]*XPathCrawlResult, len(urls))

	// 使用goroutine池
	for i, url := range urls {
		go func(index int, targetURL string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result, err := c.CrawlURL(targetURL)
			if err != nil {
				result = &XPathCrawlResult{
					URL:         targetURL,
					Success:     false,
					Error:       err.Error(),
					CrawlTime:   time.Now(),
					ProcessTime: 0,
				}
			}
			results[index] = result
		}(i, url)
	}

	// 等待所有goroutine完成
	for i := 0; i < c.config.MaxConcurrency; i++ {
		semaphore <- struct{}{}
	}

	return results, nil
}

// SetConfig 更新爬虫配置
func (c *XPathCrawler) SetConfig(config *XPathCrawlerConfig) {
	c.config = config
}

// GetSupportedPlatforms 获取支持的平台列表
func (c *XPathCrawler) GetSupportedPlatforms() []string {
	return []string{
		"CSDN博客",
		"知乎专栏",
		"知乎问答",
		"微信公众号",
		"哔哩哔哩",
		"简书",
		"掘金",
		"博客园",
		"GitHub",
		"通用网页",
	}
}

// GetStats 获取爬虫统计信息
func (c *XPathCrawler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"supported_platforms": c.GetSupportedPlatforms(),
		"config": map[string]interface{}{
			"timeout":           c.config.Timeout.String(),
			"max_content_size":  c.config.MaxContentLength,
			"retry_count":       c.config.RetryCount,
			"enable_cache":      c.config.EnableCache,
			"enable_concurrent": c.config.EnableConcurrent,
			"max_concurrency":   c.config.MaxConcurrency,
		},
		"anti_crawler_features": map[string]interface{}{
			"user_agent_rotation": len(c.userAgents),
			"tls_fingerprinting":  true,
			"session_management":  true,
			"random_delays":       true,
			"warmup_requests":     true,
		},
	}
}

// 工具函数
func xpathTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

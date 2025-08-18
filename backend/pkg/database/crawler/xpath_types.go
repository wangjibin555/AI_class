package crawler

import (
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"time"

	"golang.org/x/net/html"
)

// XPathCrawler 增强版 XPath 爬虫（集成反爬功能）
type XPathCrawler struct {
	client       *http.Client
	cookieJar    *cookiejar.Jar
	extractor    *XPathExtractor
	validator    *URLValidator
	detector     *EncodingDetector
	config       *XPathCrawlerConfig
	userAgents   []string
	randomSeed   *rand.Rand
	requestCount int
}

// XPathCrawlerConfig XPath爬虫配置
type XPathCrawlerConfig struct {
	UserAgent        string        `json:"user_agent"`
	Timeout          time.Duration `json:"timeout"`
	MaxContentLength int           `json:"max_content_length"`
	EnableDebug      bool          `json:"enable_debug"`
	RetryCount       int           `json:"retry_count"`
	RetryDelay       time.Duration `json:"retry_delay"`

	// 性能配置
	EnableCache bool          `json:"enable_cache"`
	CacheSize   int           `json:"cache_size"`
	CacheTTL    time.Duration `json:"cache_ttl"`

	// 并发配置
	EnableConcurrent bool `json:"enable_concurrent"`
	MaxConcurrency   int  `json:"max_concurrency"`
}

// XPathExtractor XPath内容提取器
type XPathExtractor struct {
	platformRules map[Platform]*XPathRule
	fallbackRule  *XPathRule
	cache         *XPathResultCache
}

// XPathRule XPath规则配置
type XPathRule struct {
	Platform      Platform          `json:"platform"`
	TitleXPaths   []XPathExpression `json:"title_xpaths"`
	ContentXPaths []XPathExpression `json:"content_xpaths"`
	AuthorXPaths  []XPathExpression `json:"author_xpaths"`
	DateXPaths    []XPathExpression `json:"date_xpaths"`
	KeywordXPaths []XPathExpression `json:"keyword_xpaths"`
	RemoveXPaths  []XPathExpression `json:"remove_xpaths"`
	ImageXPaths   []XPathExpression `json:"image_xpaths"`
	VideoXPaths   []XPathExpression `json:"video_xpaths"`
}

// XPathExpression XPath表达式
type XPathExpression struct {
	Expression  string         `json:"expression"`
	Description string         `json:"description"`
	Priority    int            `json:"priority"`
	Fallback    string         `json:"fallback,omitempty"`
	Transform   *TextTransform `json:"transform,omitempty"`
}

// TextTransform 文本转换规则
type TextTransform struct {
	Trim      bool     `json:"trim"`
	Normalize bool     `json:"normalize"`
	Regex     string   `json:"regex,omitempty"`
	Replace   []string `json:"replace,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
}

// XPathExtractedContent XPath提取的内容
type XPathExtractedContent struct {
	// 基础内容
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`

	// 元数据
	Description   string   `json:"description"`
	Keywords      []string `json:"keywords"`
	PublishedTime string   `json:"published_time"`

	// 媒体资源
	Images []ImageInfo `json:"images"`
	Videos []VideoInfo `json:"videos"`

	// 其他信息
	Language string   `json:"language"`
	Platform Platform `json:"platform"`

	// 内容统计
	WordCount int `json:"word_count"`
	ReadTime  int `json:"read_time"` // 预计阅读时间（分钟）
}

// XPathCrawlResult XPath爬取结果
type XPathCrawlResult struct {
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
	Platform    string `json:"platform"`

	// 处理信息
	CrawlTime   time.Time     `json:"crawl_time"`
	ProcessTime time.Duration `json:"process_time"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`

	// XPath特有信息
	MatchedXPaths map[string]string `json:"matched_xpaths,omitempty"` // 记录匹配的XPath表达式
	ExtractionLog []string          `json:"extraction_log,omitempty"` // 提取过程日志
}

// EncodingDetector 编码检测器
type EncodingDetector struct {
	detector interface{} // chardet.Detector
}

// XPathResultCache XPath结果缓存
type XPathResultCache struct {
	cache   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Result    interface{}
	Timestamp time.Time
	HitCount  int
}

// XPathExtractorInterface XPath提取器接口
type XPathExtractorInterface interface {
	ExtractContent(doc *html.Node, platform Platform) (*XPathExtractedContent, error)
	ExtractByXPaths(doc *html.Node, xpaths []XPathExpression, fieldType string) string
	ExtractContentByXPaths(doc *html.Node, xpaths []XPathExpression) string
}

// XPathCrawlerInterface XPath爬虫接口
type XPathCrawlerInterface interface {
	CrawlURL(url string) (*XPathCrawlResult, error)
	CrawlMultipleURLs(urls []string) ([]*XPathCrawlResult, error)
	SetConfig(config *XPathCrawlerConfig)
	GetSupportedPlatforms() []string
}

// EncodingDetectorInterface 编码检测器接口
type EncodingDetectorInterface interface {
	DetectAndDecode(body []byte, contentType string) (string, string)
	TryDecode(body []byte, charset string) (string, bool)
}

// 默认配置创建函数
func DefaultXPathCrawlerConfig() *XPathCrawlerConfig {
	return &XPathCrawlerConfig{
		UserAgent:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Timeout:          30 * time.Second,
		MaxContentLength: 1024 * 1024, // 1MB
		EnableDebug:      true,
		RetryCount:       3,
		RetryDelay:       2 * time.Second,
		EnableCache:      true,
		CacheSize:        1000,
		CacheTTL:         30 * time.Minute,
		EnableConcurrent: true,
		MaxConcurrency:   4,
	}
}

package crawler

import (
	"net/url"
	"regexp"
	"strings"
)

// Platform 平台类型
type Platform int

const (
	PlatformUnsupported Platform = iota
	PlatformGeneric
	PlatformZhihuColumn
	PlatformZhihuAnswer
	PlatformWechat
	PlatformCSDN
	PlatformBilibili
	PlatformJianshu
	PlatformJuejin
	PlatformCnblogs
	PlatformGithub
	PlatformXiaohongshu
	PlatformDouyin
)

// String 返回平台名称
func (p Platform) String() string {
	switch p {
	case PlatformGeneric:
		return "通用网页"
	case PlatformZhihuColumn:
		return "知乎专栏"
	case PlatformZhihuAnswer:
		return "知乎问答"
	case PlatformWechat:
		return "微信公众号"
	case PlatformCSDN:
		return "CSDN博客"
	case PlatformBilibili:
		return "哔哩哔哩"
	case PlatformJianshu:
		return "简书"
	case PlatformJuejin:
		return "掘金"
	case PlatformCnblogs:
		return "博客园"
	case PlatformGithub:
		return "GitHub"
	case PlatformXiaohongshu:
		return "小红书"
	case PlatformDouyin:
		return "抖音"
	default:
		return "不支持"
	}
}

// URLValidator URL验证器
type URLValidator struct {
	// 支持的域名模式
	domainPatterns map[Platform][]string

	// URL路径模式
	pathPatterns map[Platform][]string

	// 不支持的域名
	blockedDomains []string
}

// PlatformInfo 平台信息
type PlatformInfo struct {
	Platform    Platform `json:"platform"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
	Supported   bool     `json:"supported"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Platform Platform `json:"platform"`
	Reason   string   `json:"reason,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// NewURLValidator 创建URL验证器
func NewURLValidator() *URLValidator {
	validator := &URLValidator{
		domainPatterns: make(map[Platform][]string),
		pathPatterns:   make(map[Platform][]string),
		blockedDomains: []string{
			"localhost",
			"127.0.0.1",
			"0.0.0.0",
			"192.168.",
			"10.",
			"172.16.",
			"172.17.",
			"172.18.",
			"172.19.",
			"172.20.",
			"172.21.",
			"172.22.",
			"172.23.",
			"172.24.",
			"172.25.",
			"172.26.",
			"172.27.",
			"172.28.",
			"172.29.",
			"172.30.",
			"172.31.",
		},
	}

	validator.initPlatformPatterns()
	return validator
}

// initPlatformPatterns 初始化平台匹配模式
func (v *URLValidator) initPlatformPatterns() {
	// 知乎专栏
	v.domainPatterns[PlatformZhihuColumn] = []string{
		"zhuanlan.zhihu.com",
	}
	v.pathPatterns[PlatformZhihuColumn] = []string{
		`^/p/\d+`,
		`^/c_\d+`,
	}

	// 知乎问答
	v.domainPatterns[PlatformZhihuAnswer] = []string{
		"www.zhihu.com",
		"zhihu.com",
	}
	v.pathPatterns[PlatformZhihuAnswer] = []string{
		`^/question/\d+`,
		`^/question/\d+/answer/\d+`,
	}

	// 微信公众号
	v.domainPatterns[PlatformWechat] = []string{
		"mp.weixin.qq.com",
	}
	v.pathPatterns[PlatformWechat] = []string{
		`^/s/.*`,
		`^/s\?.*`,
	}

	// CSDN
	v.domainPatterns[PlatformCSDN] = []string{
		"blog.csdn.net",
		"www.csdn.net",
	}
	v.pathPatterns[PlatformCSDN] = []string{
		`^/\w+/article/details/\d+`,
		`^/article/details/\d+`,
	}

	// 哔哩哔哩
	v.domainPatterns[PlatformBilibili] = []string{
		"www.bilibili.com",
		"bilibili.com",
		"b23.tv",
	}
	v.pathPatterns[PlatformBilibili] = []string{
		`^/video/[A-Z0-9]+`,
		`^/read/cv\d+`,
		`^/article/\d+`,
		`^/bangumi/play/ss\d+`,
		`^/bangumi/play/ep\d+`,
	}

	// 简书
	v.domainPatterns[PlatformJianshu] = []string{
		"www.jianshu.com",
		"jianshu.com",
	}
	v.pathPatterns[PlatformJianshu] = []string{
		`^/p/[a-f0-9]+`,
		`^/writer#/notebooks/\d+/notes/\d+`,
	}

	// 掘金
	v.domainPatterns[PlatformJuejin] = []string{
		"juejin.cn",
		"juejin.im",
	}
	v.pathPatterns[PlatformJuejin] = []string{
		`^/post/\d+`,
		`^/entry/[a-f0-9]+`,
	}

	// 博客园
	v.domainPatterns[PlatformCnblogs] = []string{
		"www.cnblogs.com",
		"cnblogs.com",
	}
	v.pathPatterns[PlatformCnblogs] = []string{
		`^/\w+/p/\d+\.html`,
		`^/\w+/archive/\d+/\d+/\d+/\d+\.html`,
	}

	// GitHub
	v.domainPatterns[PlatformGithub] = []string{
		"github.com",
		"www.github.com",
	}
	v.pathPatterns[PlatformGithub] = []string{
		`^/[\w-]+/[\w-]+`,
		`^/[\w-]+/[\w-]+/blob/.*`,
		`^/[\w-]+/[\w-]+/tree/.*`,
	}

	// 小红书
	v.domainPatterns[PlatformXiaohongshu] = []string{
		"www.xiaohongshu.com",
		"xiaohongshu.com",
		"xhslink.com",
	}
	v.pathPatterns[PlatformXiaohongshu] = []string{
		`^/discovery/item/[a-f0-9]+`,
		`^/explore/[a-f0-9]+`,
		`^/item/[a-f0-9]+`,
		`^/[a-f0-9]+`,
	}

	// 抖音
	v.domainPatterns[PlatformDouyin] = []string{
		"www.douyin.com",
		"douyin.com",
	}
	v.pathPatterns[PlatformDouyin] = []string{
		`^/video/\d+`,
		`^/user/[^/]+`,
		`^/note/\d+`,
		`^/share/video/\d+`,
	}
}

// IsValidURL 验证URL是否有效
func (v *URLValidator) IsValidURL(rawURL string) bool {
	result := v.ValidateURL(rawURL)
	return result.Valid
}

// ValidateURL 详细验证URL
func (v *URLValidator) ValidateURL(rawURL string) *ValidationResult {
	result := &ValidationResult{
		Valid:    false,
		Platform: PlatformUnsupported,
	}

	// 基础URL格式验证
	if rawURL == "" {
		result.Reason = "URL不能为空"
		return result
	}

	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		result.Reason = "URL格式无效: " + err.Error()
		return result
	}

	// 检查协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		result.Reason = "只支持HTTP和HTTPS协议"
		return result
	}

	// 检查主机名
	if parsedURL.Host == "" {
		result.Reason = "URL缺少主机名"
		return result
	}

	// 检查是否在黑名单中
	if v.isBlockedDomain(parsedURL.Host) {
		result.Reason = "该域名被禁止访问"
		return result
	}

	// 检测平台
	platform := v.detectPlatformFromURL(parsedURL)
	result.Platform = platform

	// 验证平台特定规则
	if platform != PlatformUnsupported {
		if v.validatePlatformURL(parsedURL, platform) {
			result.Valid = true

			// 添加警告信息
			result.Warnings = v.getURLWarnings(parsedURL, platform)
		} else {
			result.Reason = "URL不符合" + platform.String() + "的格式要求"
		}
	} else {
		// 通用网页验证
		if v.validateGenericURL(parsedURL) {
			result.Valid = true
			result.Platform = PlatformGeneric
			result.Warnings = []string{"该网站可能需要特殊处理，抓取效果可能不理想"}
		} else {
			result.Reason = "不支持的网站类型"
		}
	}

	return result
}

// DetectPlatform 检测平台类型
func (v *URLValidator) DetectPlatform(rawURL string) Platform {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return PlatformUnsupported
	}

	return v.detectPlatformFromURL(parsedURL)
}

// detectPlatformFromURL 从URL检测平台
func (v *URLValidator) detectPlatformFromURL(parsedURL *url.URL) Platform {
	host := strings.ToLower(parsedURL.Host)
	path := parsedURL.Path

	// 遍历所有平台
	for platform, domainPatterns := range v.domainPatterns {
		// 检查域名匹配
		for _, domainPattern := range domainPatterns {
			if v.matchDomain(host, domainPattern) {
				// 检查路径匹配
				pathPatterns := v.pathPatterns[platform]
				if len(pathPatterns) == 0 {
					return platform // 如果没有路径要求，直接返回
				}

				for _, pathPattern := range pathPatterns {
					if matched, _ := regexp.MatchString(pathPattern, path); matched {
						return platform
					}
				}
			}
		}
	}

	return PlatformUnsupported
}

// matchDomain 匹配域名
func (v *URLValidator) matchDomain(host, pattern string) bool {
	// 精确匹配
	if host == pattern {
		return true
	}

	// 子域名匹配
	if strings.HasSuffix(host, "."+pattern) {
		return true
	}

	return false
}

// validatePlatformURL 验证平台特定URL
func (v *URLValidator) validatePlatformURL(parsedURL *url.URL, platform Platform) bool {
	pathPatterns := v.pathPatterns[platform]
	if len(pathPatterns) == 0 {
		return true // 没有路径要求
	}

	path := parsedURL.Path
	for _, pattern := range pathPatterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}

	return false
}

// validateGenericURL 验证通用URL
func (v *URLValidator) validateGenericURL(parsedURL *url.URL) bool {
	// 基本的通用网页验证
	// 检查是否有合理的路径
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		return false // 主页可能不适合作为文章内容
	}

	// 检查文件扩展名，排除一些不合适的类型
	excludeExtensions := []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".rar", ".tar", ".gz",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wmv",
		".exe", ".dmg", ".pkg",
	}

	path := strings.ToLower(parsedURL.Path)
	for _, ext := range excludeExtensions {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}

	return true
}

// isBlockedDomain 检查是否为被禁止的域名
func (v *URLValidator) isBlockedDomain(host string) bool {
	host = strings.ToLower(host)

	for _, blocked := range v.blockedDomains {
		if strings.Contains(host, blocked) {
			return true
		}
	}

	return false
}

// getURLWarnings 获取URL警告信息
func (v *URLValidator) getURLWarnings(parsedURL *url.URL, platform Platform) []string {
	var warnings []string

	switch platform {
	case PlatformWechat:
		warnings = append(warnings, "微信公众号文章可能有访问限制")
	case PlatformZhihuAnswer:
		warnings = append(warnings, "知乎问答页面可能包含多个回答，将提取最佳回答")
	case PlatformGithub:
		warnings = append(warnings, "GitHub页面可能是代码仓库，内容提取效果可能有限")
	case PlatformXiaohongshu:
		warnings = append(warnings, "小红书内容可能需要登录才能完整访问")
	case PlatformDouyin:
		warnings = append(warnings, "抖音内容主要为视频，文本提取效果可能有限")
	}

	// 检查URL参数
	if len(parsedURL.RawQuery) > 100 {
		warnings = append(warnings, "URL包含复杂参数，可能影响访问稳定性")
	}

	// 检查路径深度
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) > 5 {
		warnings = append(warnings, "URL路径较深，可能不是文章页面")
	}

	return warnings
}

// GetSupportedPlatforms 获取支持的平台列表
func (v *URLValidator) GetSupportedPlatforms() []*PlatformInfo {
	platforms := []*PlatformInfo{
		{
			Platform:    PlatformZhihuColumn,
			Name:        "知乎专栏",
			Description: "知乎专栏文章",
			Examples:    []string{"https://zhuanlan.zhihu.com/p/123456789"},
			Supported:   true,
		},
		{
			Platform:    PlatformZhihuAnswer,
			Name:        "知乎问答",
			Description: "知乎问题和回答",
			Examples:    []string{"https://www.zhihu.com/question/123456789/answer/987654321"},
			Supported:   true,
		},
		{
			Platform:    PlatformWechat,
			Name:        "微信公众号",
			Description: "微信公众号文章",
			Examples:    []string{"https://mp.weixin.qq.com/s/abc123xyz"},
			Supported:   true,
		},
		{
			Platform:    PlatformCSDN,
			Name:        "CSDN博客",
			Description: "CSDN技术博客文章",
			Examples:    []string{"https://blog.csdn.net/username/article/details/123456789"},
			Supported:   true,
		},
		{
			Platform:    PlatformBilibili,
			Name:        "哔哩哔哩",
			Description: "哔哩哔哩视频和文章",
			Examples:    []string{"https://www.bilibili.com/video/BV1234567890", "https://www.bilibili.com/read/cv1234567890"},
			Supported:   true,
		},
		{
			Platform:    PlatformJianshu,
			Name:        "简书",
			Description: "简书文章",
			Examples:    []string{"https://www.jianshu.com/p/abc123def456"},
			Supported:   true,
		},
		{
			Platform:    PlatformJuejin,
			Name:        "掘金",
			Description: "掘金技术文章",
			Examples:    []string{"https://juejin.cn/post/1234567890"},
			Supported:   true,
		},
		{
			Platform:    PlatformCnblogs,
			Name:        "博客园",
			Description: "博客园技术博客",
			Examples:    []string{"https://www.cnblogs.com/username/p/123456.html"},
			Supported:   true,
		},
		{
			Platform:    PlatformGithub,
			Name:        "GitHub",
			Description: "GitHub仓库和文档",
			Examples:    []string{"https://github.com/user/repo"},
			Supported:   true,
		},
		{
			Platform:    PlatformXiaohongshu,
			Name:        "小红书",
			Description: "小红书笔记和内容",
			Examples:    []string{"https://www.xiaohongshu.com/discovery/item/abc123", "http://xhslink.com/abc123"},
			Supported:   true,
		},
		{
			Platform:    PlatformDouyin,
			Name:        "抖音",
			Description: "抖音视频和内容",
			Examples:    []string{"https://www.douyin.com/video/123456789"},
			Supported:   true,
		},
		{
			Platform:    PlatformGeneric,
			Name:        "通用网页",
			Description: "其他网站的文章页面",
			Examples:    []string{"https://example.com/article/123"},
			Supported:   true,
		},
	}

	return platforms
}

// NormalizeURL 标准化URL
func (v *URLValidator) NormalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 确保使用HTTPS（如果支持）
	if parsedURL.Scheme == "http" {
		// 对于支持HTTPS的网站，自动转换
		secureHosts := []string{
			"zhihu.com",
			"zhuanlan.zhihu.com",
			"mp.weixin.qq.com",
			"blog.csdn.net",
			"www.jianshu.com",
			"juejin.cn",
			"www.cnblogs.com",
			"github.com",
			"xiaohongshu.com",
			"douyin.com",
		}

		for _, host := range secureHosts {
			if strings.Contains(parsedURL.Host, host) {
				parsedURL.Scheme = "https"
				break
			}
		}
	}

	// 移除无用的查询参数
	v.cleanupQueryParams(parsedURL)

	return parsedURL.String(), nil
}

// cleanupQueryParams 清理查询参数
func (v *URLValidator) cleanupQueryParams(parsedURL *url.URL) {
	// 保留有用的参数，移除跟踪参数
	query := parsedURL.Query()

	// 需要移除的跟踪参数
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"spm", "from", "ref", "referrer",
		"share_token", "timestamp", "nonce",
		"_t", "_from", "source",
	}

	for _, param := range trackingParams {
		query.Del(param)
	}

	parsedURL.RawQuery = query.Encode()
}

// IsArticleURL 判断是否为文章URL
func (v *URLValidator) IsArticleURL(rawURL string) bool {
	platform := v.DetectPlatform(rawURL)

	// 已知平台的文章URL
	if platform != PlatformUnsupported && platform != PlatformGeneric {
		return true
	}

	// 通用判断
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	path := strings.ToLower(parsedURL.Path)

	// 包含文章关键词的路径
	articleKeywords := []string{
		"article", "post", "blog", "news", "story",
		"detail", "view", "read", "content",
		"文章", "博客", "新闻", "详情",
	}

	for _, keyword := range articleKeywords {
		if strings.Contains(path, keyword) {
			return true
		}
	}

	return false
}

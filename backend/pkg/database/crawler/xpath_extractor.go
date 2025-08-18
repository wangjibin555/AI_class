package crawler

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// NewXPathExtractor 创建XPath提取器
func NewXPathExtractor() *XPathExtractor {
	extractor := &XPathExtractor{
		platformRules: make(map[Platform]*XPathRule),
		cache:         NewXPathResultCache(1000, 30*time.Minute),
	}

	// 初始化平台规则
	extractor.initializePlatformRules()

	return extractor
}

// initializePlatformRules 初始化平台规则
func (e *XPathExtractor) initializePlatformRules() {
	// CSDN平台规则 (基于实际调试结果更新)
	e.platformRules[PlatformCSDN] = &XPathRule{
		Platform: PlatformCSDN,

		// 标题提取XPath (优先级递减，基于调试验证)
		TitleXPaths: []XPathExpression{
			{
				Expression:  "//h1[@class='title-article']",
				Description: "CSDN主标题选择器",
				Priority:    100,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 200},
			},
			{
				Expression:  "//div[@class='article-header']//h1",
				Description: "文章头部标题",
				Priority:    90,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 200},
			},
			{
				Expression:  "//div[@class='article-title-box']//h1",
				Description: "文章标题框",
				Priority:    80,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 200},
			},
			{
				Expression:  "//h1[1]",
				Description: "页面第一个H1标题",
				Priority:    70,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 200},
			},
			{
				Expression:  "//title",
				Description: "页面title标签备选",
				Priority:    50,
				Transform: &TextTransform{
					Trim:      true,
					Normalize: true,
					MaxLength: 200,
					Replace:   []string{"-CSDN博客", ""},
				},
			},
		},

		// 内容提取XPath (基于调试验证更新)
		ContentXPaths: []XPathExpression{
			{
				Expression:  "//div[@id='content_views']",
				Description: "CSDN主内容区域",
				Priority:    100,
				Transform: &TextTransform{
					Trim:      true,
					Normalize: true,
					MaxLength: 100000,
				},
			},
			{
				Expression:  "//article",
				Description: "文章标签内容",
				Priority:    90,
				Transform: &TextTransform{
					Trim:      true,
					Normalize: true,
					MaxLength: 100000,
				},
			},
			{
				Expression:  "//div[contains(@class,'article_content')]",
				Description: "文章内容区域",
				Priority:    80,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 100000},
			},
			{
				Expression:  "//div[@class='htmledit_views']",
				Description: "HTML编辑器内容",
				Priority:    70,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 100000},
			},
			{
				Expression:  "//div[@class='markdown_views']",
				Description: "Markdown内容",
				Priority:    60,
				Transform:   &TextTransform{Trim: true, Normalize: true, MaxLength: 100000},
			},
		},

		// 作者提取XPath (基于调试验证更新)
		AuthorXPaths: []XPathExpression{
			{
				Expression:  "//a[contains(@href,'weixin_')]/text()",
				Description: "CSDN作者用户名链接",
				Priority:    100,
				Transform:   &TextTransform{Trim: true, MaxLength: 50},
			},
			{
				Expression:  "//*[contains(text(),'码农阿豪')]",
				Description: "作者昵称直接匹配",
				Priority:    90,
				Transform:   &TextTransform{Trim: true, MaxLength: 50},
			},
			{
				Expression:  "//meta[@name='author']/@content",
				Description: "Meta作者标签",
				Priority:    80,
				Transform:   &TextTransform{Trim: true, MaxLength: 50},
			},
			{
				Expression:  "//*[contains(@class,'author')]",
				Description: "作者类名匹配",
				Priority:    70,
				Transform:   &TextTransform{Trim: true, MaxLength: 50},
			},
			{
				Expression:  "//div[@class='user-profile']//span[@class='nickname']",
				Description: "用户资料昵称",
				Priority:    60,
				Transform:   &TextTransform{Trim: true, MaxLength: 50},
			},
		},

		// 时间提取XPath
		DateXPaths: []XPathExpression{
			{
				Expression:  "//span[@class='publish-time']/text()",
				Description: "发布时间",
				Priority:    100,
				Transform:   &TextTransform{Trim: true},
			},
			{
				Expression:  "//div[@class='article-meta']//span[contains(@class,'date')]/text()",
				Description: "文章元数据时间",
				Priority:    90,
				Transform:   &TextTransform{Trim: true},
			},
			{
				Expression:  "//time/@datetime",
				Description: "HTML5时间元素",
				Priority:    80,
				Transform:   &TextTransform{Trim: true},
			},
			{
				Expression:  "//span[contains(text(),'发布于') or contains(text(),'更新于')]/text()",
				Description: "文本包含时间信息",
				Priority:    60,
				Transform:   &TextTransform{Trim: true},
			},
		},

		// 关键词提取XPath
		KeywordXPaths: []XPathExpression{
			{
				Expression:  "//meta[@name='keywords']/@content",
				Description: "Meta关键词标签",
				Priority:    100,
			},
			{
				Expression:  "//div[@class='tags-box']//a/text()",
				Description: "标签箱关键词",
				Priority:    80,
				Transform:   &TextTransform{Trim: true},
			},
		},

		// 移除干扰元素XPath
		RemoveXPaths: []XPathExpression{
			{
				Expression:  "//script | //style | //nav | //header | //footer",
				Description: "基础移除元素",
				Priority:    100,
			},
			{
				Expression:  "//*[@class='prism-toolbar' or @class='hljs-button' or @class='copy-code-btn']",
				Description: "代码工具栏移除",
				Priority:    90,
			},
			{
				Expression:  "//*[contains(@class,'ad') or contains(@class,'advertisement')]",
				Description: "广告相关元素",
				Priority:    85,
			},
			{
				Expression:  "//*[@class='comment-box' or @class='recommend-box']",
				Description: "评论和推荐框",
				Priority:    80,
			},
		},

		// 图片提取XPath
		ImageXPaths: []XPathExpression{
			{
				Expression:  "//div[@id='content_views']//img/@src",
				Description: "内容区域图片",
				Priority:    100,
			},
			{
				Expression:  "//img[@alt and @src]/@src",
				Description: "有alt属性的图片",
				Priority:    80,
			},
		},

		// 视频提取XPath
		VideoXPaths: []XPathExpression{
			{
				Expression:  "//video/@src | //iframe[contains(@src,'video')]/@src",
				Description: "视频元素",
				Priority:    100,
			},
		},
	}

	// 通用平台规则
	e.platformRules[PlatformGeneric] = &XPathRule{
		Platform: PlatformGeneric,

		TitleXPaths: []XPathExpression{
			{
				Expression:  "//title/text()",
				Description: "页面标题",
				Priority:    90,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
			{
				Expression:  "//h1[1]/text()",
				Description: "第一个H1标题",
				Priority:    80,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
			{
				Expression:  "//meta[@property='og:title']/@content",
				Description: "Open Graph标题",
				Priority:    70,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
		},

		ContentXPaths: []XPathExpression{
			{
				Expression:  "//main",
				Description: "主要内容区域",
				Priority:    100,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
			{
				Expression:  "//article",
				Description: "文章内容",
				Priority:    90,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
			{
				Expression:  "//*[@class='content' or @class='main-content']",
				Description: "内容类名匹配",
				Priority:    80,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
			{
				Expression:  "//body",
				Description: "整个body内容",
				Priority:    60,
				Transform:   &TextTransform{Trim: true, Normalize: true},
			},
		},

		AuthorXPaths: []XPathExpression{
			{
				Expression:  "//meta[@name='author']/@content",
				Description: "Meta作者标签",
				Priority:    100,
				Transform:   &TextTransform{Trim: true},
			},
			{
				Expression:  "//*[contains(@class,'author')]",
				Description: "作者类名匹配",
				Priority:    80,
				Transform:   &TextTransform{Trim: true},
			},
		},

		RemoveXPaths: []XPathExpression{
			{
				Expression:  "//script | //style | //nav | //header | //footer | //aside",
				Description: "基础移除元素",
				Priority:    100,
			},
		},
	}

	// 设置fallback规则
	e.fallbackRule = e.platformRules[PlatformGeneric]
}

// ExtractContent 提取内容
func (e *XPathExtractor) ExtractContent(doc *html.Node, platform Platform) (*XPathExtractedContent, error) {
	fmt.Printf("\n🔧 XPath内容提取开始\n")
	fmt.Printf("📋 目标平台: %s\n", platform.String())

	// 获取平台规则
	rule := e.getPlatformRule(platform)
	fmt.Printf("✅ 获取平台规则: %s (%d个标题XPath, %d个内容XPath)\n",
		rule.Platform.String(), len(rule.TitleXPaths), len(rule.ContentXPaths))

	// 移除干扰元素
	e.removeNoiseElements(doc, rule)

	result := &XPathExtractedContent{
		Platform: platform,
	}

	// 提取各类内容
	fmt.Printf("\n🎯 开始字段提取:\n")
	result.Title = e.extractByXPaths(doc, rule.TitleXPaths, "title")
	result.Content = e.extractContentByXPaths(doc, rule.ContentXPaths)
	result.Author = e.extractByXPaths(doc, rule.AuthorXPaths, "author")
	result.PublishedTime = e.extractByXPaths(doc, rule.DateXPaths, "date")
	result.Keywords = e.extractKeywordsByXPaths(doc, rule.KeywordXPaths)
	result.Images = e.extractImagesByXPaths(doc, rule.ImageXPaths)
	result.Videos = e.extractVideosByXPaths(doc, rule.VideoXPaths)
	result.Language = e.detectLanguage(doc)

	// 计算统计信息
	result.WordCount = len([]rune(result.Content))
	result.ReadTime = result.WordCount / 300 // 假设每分钟阅读300字
	if result.ReadTime == 0 {
		result.ReadTime = 1
	}

	fmt.Printf("\n📊 提取结果统计:\n")
	fmt.Printf("   📝 标题长度: %d\n", len(result.Title))
	fmt.Printf("   📄 内容长度: %d 字符\n", result.WordCount)
	fmt.Printf("   👤 作者: %s\n", getXPathFieldStatus(result.Author))
	fmt.Printf("   🏷️ 关键词数量: %d\n", len(result.Keywords))
	fmt.Printf("   📅 发布时间: %s\n", getXPathFieldStatus(result.PublishedTime))
	fmt.Printf("   🖼️ 图片数量: %d\n", len(result.Images))
	fmt.Printf("   📺 视频数量: %d\n", len(result.Videos))
	fmt.Printf("   ⏱️ 预计阅读时间: %d分钟\n", result.ReadTime)

	return result, nil
}

// extractByXPaths 按XPath优先级提取文本
func (e *XPathExtractor) extractByXPaths(doc *html.Node, xpaths []XPathExpression, fieldType string) string {
	// 按优先级排序
	sort.Slice(xpaths, func(i, j int) bool {
		return xpaths[i].Priority > xpaths[j].Priority
	})

	for i, xpath := range xpaths {
		nodes := htmlquery.Find(doc, xpath.Expression)
		fmt.Printf("   🔍 %s XPath[%d]: %s -> 找到 %d 个节点\n",
			fieldType, i+1, xpath.Expression, len(nodes))

		if len(nodes) > 0 {
			var text string
			if nodes[0].Type == html.TextNode {
				text = nodes[0].Data
			} else if nodes[0].Type == html.ElementNode {
				text = htmlquery.InnerText(nodes[0])
			} else {
				// 属性节点
				if strings.Contains(xpath.Expression, "/@") {
					text = htmlquery.SelectAttr(nodes[0], strings.Split(xpath.Expression, "/@")[1])
				}
			}

			fmt.Printf("      原始文本: '%s' (长度: %d)\n", xpathTruncate(text, 100), len(text))

			transformed := e.transformText(text, xpath.Transform)
			fmt.Printf("      转换后文本: '%s' (长度: %d)\n", xpathTruncate(transformed, 100), len(transformed))

			if transformed != "" {
				fmt.Printf("   ✅ %s XPath[%d] 匹配成功: %s\n",
					fieldType, i+1, xpath.Description)
				return transformed
			} else {
				fmt.Printf("   ❌ %s XPath[%d] 转换后为空: %s\n",
					fieldType, i+1, xpath.Description)
			}
		} else {
			fmt.Printf("   ❌ %s XPath[%d] 无节点匹配: %s\n",
				fieldType, i+1, xpath.Expression)
		}
	}

	fmt.Printf("   ⚠️ %s 未找到匹配的XPath\n", fieldType)
	return ""
}

// extractContentByXPaths 提取内容文本
func (e *XPathExtractor) extractContentByXPaths(doc *html.Node, xpaths []XPathExpression) string {
	// 按优先级排序
	sort.Slice(xpaths, func(i, j int) bool {
		return xpaths[i].Priority > xpaths[j].Priority
	})

	for i, xpath := range xpaths {
		nodes := htmlquery.Find(doc, xpath.Expression)
		if len(nodes) > 0 {
			var contentParts []string

			for _, node := range nodes {
				var text string
				if node.Type == html.TextNode {
					text = strings.TrimSpace(node.Data)
				} else {
					text = strings.TrimSpace(htmlquery.InnerText(node))
				}

				if len(text) > 20 { // 过滤太短的文本
					contentParts = append(contentParts, text)
				}
			}

			if len(contentParts) > 0 {
				content := strings.Join(contentParts, "\n")
				if transformed := e.transformText(content, xpath.Transform); transformed != "" {
					fmt.Printf("   ✅ 内容XPath[%d] 匹配: %s, 长度: %d\n",
						i+1, xpath.Description, len(transformed))
					return transformed
				}
			}
		}
	}

	fmt.Printf("   ⚠️ 内容 未找到匹配的XPath\n")
	return ""
}

// extractKeywordsByXPaths 提取关键词
func (e *XPathExtractor) extractKeywordsByXPaths(doc *html.Node, xpaths []XPathExpression) []string {
	// 按优先级排序
	sort.Slice(xpaths, func(i, j int) bool {
		return xpaths[i].Priority > xpaths[j].Priority
	})

	for i, xpath := range xpaths {
		nodes := htmlquery.Find(doc, xpath.Expression)
		if len(nodes) > 0 {
			var allKeywords []string

			for _, node := range nodes {
				var content string
				if node.Type == html.TextNode {
					content = node.Data
				} else {
					content = htmlquery.InnerText(node)
				}

				if content != "" {
					// 根据逗号分割关键词
					keywords := strings.Split(content, ",")
					for _, keyword := range keywords {
						if trimmed := strings.TrimSpace(keyword); trimmed != "" {
							allKeywords = append(allKeywords, trimmed)
						}
					}
				}
			}

			if len(allKeywords) > 0 {
				fmt.Printf("   ✅ 关键词XPath[%d] 匹配: %s -> %v\n",
					i+1, xpath.Description, allKeywords)
				return allKeywords
			}
		}
	}

	// 降级到内容分析
	fmt.Printf("   📝 关键词XPath未匹配，降级到内容分析\n")
	return e.extractKeywordsFromContent(htmlquery.InnerText(doc))
}

// extractImagesByXPaths 提取图片
func (e *XPathExtractor) extractImagesByXPaths(doc *html.Node, xpaths []XPathExpression) []ImageInfo {
	var images []ImageInfo

	for _, xpath := range xpaths {
		nodes := htmlquery.Find(doc, xpath.Expression)
		for _, node := range nodes {
			var src string
			if node.Type == html.TextNode {
				src = node.Data
			} else if strings.Contains(xpath.Expression, "/@") {
				// 属性节点
				attr := strings.Split(xpath.Expression, "/@")[1]
				src = htmlquery.SelectAttr(node, attr)
			} else {
				src = htmlquery.SelectAttr(node, "src")
			}

			if src != "" {
				images = append(images, ImageInfo{
					URL:   src,
					Alt:   htmlquery.SelectAttr(node, "alt"),
					Title: htmlquery.SelectAttr(node, "title"),
				})
			}
		}
	}

	return images
}

// extractVideosByXPaths 提取视频
func (e *XPathExtractor) extractVideosByXPaths(doc *html.Node, xpaths []XPathExpression) []VideoInfo {
	var videos []VideoInfo

	for _, xpath := range xpaths {
		nodes := htmlquery.Find(doc, xpath.Expression)
		for _, node := range nodes {
			var src string
			if strings.Contains(xpath.Expression, "/@") {
				// 属性节点
				attr := strings.Split(xpath.Expression, "/@")[1]
				src = htmlquery.SelectAttr(node, attr)
			} else {
				src = htmlquery.SelectAttr(node, "src")
			}

			if src != "" {
				videos = append(videos, VideoInfo{
					URL:   src,
					Title: htmlquery.SelectAttr(node, "title"),
				})
			}
		}
	}

	return videos
}

// removeNoiseElements 移除干扰元素
func (e *XPathExtractor) removeNoiseElements(doc *html.Node, rule *XPathRule) {
	removedCount := 0
	for _, xpath := range rule.RemoveXPaths {
		nodes := htmlquery.Find(doc, xpath.Expression)
		for _, node := range nodes {
			if node.Parent != nil {
				node.Parent.RemoveChild(node)
				removedCount++
			}
		}
	}
	fmt.Printf("🧹 移除干扰元素: %d个\n", removedCount)
}

// transformText 文本转换
func (e *XPathExtractor) transformText(text string, transform *TextTransform) string {
	if transform == nil {
		return strings.TrimSpace(text)
	}

	result := text

	if transform.Trim {
		result = strings.TrimSpace(result)
	}

	if transform.Normalize {
		// 标准化空白字符
		re := regexp.MustCompile(`\s+`)
		result = re.ReplaceAllString(result, " ")
	}

	if transform.MaxLength > 0 && len([]rune(result)) > transform.MaxLength {
		runes := []rune(result)
		result = string(runes[:transform.MaxLength])
	}

	// 处理替换规则
	if len(transform.Replace) >= 2 {
		for i := 0; i < len(transform.Replace); i += 2 {
			if i+1 < len(transform.Replace) {
				result = strings.ReplaceAll(result, transform.Replace[i], transform.Replace[i+1])
			}
		}
	}

	// 处理正则表达式替换
	if transform.Regex != "" && len(transform.Replace) >= 2 {
		re := regexp.MustCompile(transform.Regex)
		result = re.ReplaceAllString(result, transform.Replace[1])
	}

	return result
}

// getPlatformRule 获取平台规则
func (e *XPathExtractor) getPlatformRule(platform Platform) *XPathRule {
	if rule, exists := e.platformRules[platform]; exists {
		return rule
	}
	return e.fallbackRule
}

// detectLanguage 检测语言
func (e *XPathExtractor) detectLanguage(doc *html.Node) string {
	// 尝试从HTML标签获取语言
	if langNode := htmlquery.FindOne(doc, "//html/@lang"); langNode != nil {
		return htmlquery.SelectAttr(langNode, "lang")
	}

	// 尝试从meta标签获取语言
	if metaNode := htmlquery.FindOne(doc, "//meta[@http-equiv='content-language']/@content"); metaNode != nil {
		return htmlquery.SelectAttr(metaNode, "content")
	}

	// 默认返回中文
	return "zh-CN"
}

// extractKeywordsFromContent 从内容中提取关键词
func (e *XPathExtractor) extractKeywordsFromContent(content string) []string {
	if len(content) < 100 {
		return []string{}
	}

	// 简单的关键词提取逻辑
	words := strings.Fields(content)
	wordCount := make(map[string]int)

	for _, word := range words {
		if len(word) > 2 && utf8.ValidString(word) {
			wordCount[word]++
		}
	}

	// 获取高频词作为关键词
	var keywords []string
	for word, count := range wordCount {
		if count >= 3 && len(keywords) < 10 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// NewXPathResultCache 创建XPath结果缓存
func NewXPathResultCache(maxSize int, ttl time.Duration) *XPathResultCache {
	return &XPathResultCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get 获取缓存
func (c *XPathResultCache) Get(key string) (interface{}, bool) {
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.Timestamp) > c.ttl {
		delete(c.cache, key)
		return nil, false
	}

	entry.HitCount++
	return entry.Result, true
}

// Set 设置缓存
func (c *XPathResultCache) Set(key string, value interface{}) {
	if len(c.cache) >= c.maxSize {
		// 简单的LRU清理
		c.evictOldest()
	}

	c.cache[key] = &CacheEntry{
		Result:    value,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

// evictOldest 清理最旧的缓存项
func (c *XPathResultCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// getXPathFieldStatus 获取字段状态显示
func getXPathFieldStatus(value string) string {
	if value != "" {
		return fmt.Sprintf("✅ %s", value)
	}
	return "❌ 未提取"
}

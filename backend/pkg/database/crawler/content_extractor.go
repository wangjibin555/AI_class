package crawler

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ContentExtractor 内容提取器
type ContentExtractor struct {
	// 通用规则
	titleSelectors   []string
	contentSelectors []string
	authorSelectors  []string

	// 平台特定规则
	platformRules map[Platform]*PlatformRule
}

// PlatformRule 平台特定的提取规则
type PlatformRule struct {
	Platform         Platform `json:"platform"`
	TitleSelectors   []string `json:"title_selectors"`
	ContentSelectors []string `json:"content_selectors"`
	AuthorSelectors  []string `json:"author_selectors"`
	DateSelectors    []string `json:"date_selectors"`
	ImageSelectors   []string `json:"image_selectors"`
	VideoSelectors   []string `json:"video_selectors"`

	// 内容清理规则
	RemoveSelectors []string      `json:"remove_selectors"`
	CleanupRules    []CleanupRule `json:"cleanup_rules"`
}

// CleanupRule 内容清理规则
type CleanupRule struct {
	Type    string `json:"type"`    // regex, replace, remove
	Pattern string `json:"pattern"` // 匹配模式
	Replace string `json:"replace"` // 替换内容
}

// ExtractedContent 提取的内容
type ExtractedContent struct {
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

// NewContentExtractor 创建内容提取器
func NewContentExtractor() *ContentExtractor {
	extractor := &ContentExtractor{
		// 通用选择器
		titleSelectors: []string{
			"h1",
			"title",
			".title",
			".article-title",
			".post-title",
			"[data-title]",
		},
		contentSelectors: []string{
			".content",
			".article-content",
			".post-content",
			".entry-content",
			"article",
			".markdown-body",
		},
		authorSelectors: []string{
			".author",
			".author-name",
			".byline",
			"[rel='author']",
			".post-author",
		},
		platformRules: make(map[Platform]*PlatformRule),
	}

	// 初始化平台规则
	extractor.initPlatformRules()

	return extractor
}

// initPlatformRules 初始化平台特定规则
func (e *ContentExtractor) initPlatformRules() {
	// 知乎专栏规则
	e.platformRules[PlatformZhihuColumn] = &PlatformRule{
		Platform: PlatformZhihuColumn,
		TitleSelectors: []string{
			".Post-Title",
			".ContentItem-title",
			"h1.ContentTitle",
		},
		ContentSelectors: []string{
			".Post-RichTextContainer",
			".RichContent-inner",
			".Post-RichText",
		},
		AuthorSelectors: []string{
			".AuthorInfo-name",
			".UserLink-link",
			".author-link-line",
		},
		DateSelectors: []string{
			".ContentItem-time",
			".Post-Header time",
		},
		RemoveSelectors: []string{
			".RichContent-actions",
			".Post-Sub",
			".Post-NormalMain footer",
		},
	}

	// 知乎问答规则
	e.platformRules[PlatformZhihuAnswer] = &PlatformRule{
		Platform: PlatformZhihuAnswer,
		TitleSelectors: []string{
			".QuestionPage-title",
			"h1[data-za-element='QuestionTitle']",
		},
		ContentSelectors: []string{
			".RichContent-inner",
			".AnswerItem .RichContent",
		},
		AuthorSelectors: []string{
			".AuthorInfo-name",
			".UserLink-link",
		},
		RemoveSelectors: []string{
			".RichContent-actions",
			".Answer-Sub",
		},
	}

	// 微信公众号规则
	e.platformRules[PlatformWechat] = &PlatformRule{
		Platform: PlatformWechat,
		TitleSelectors: []string{
			"#activity-name",
			".rich_media_title",
			"h1.title",
		},
		ContentSelectors: []string{
			"#js_content",
			".rich_media_content",
			".article-content",
		},
		AuthorSelectors: []string{
			".rich_media_meta_text",
			".author",
			"#js_name",
		},
		DateSelectors: []string{
			"#publish_time",
			".rich_media_meta_text",
		},
		RemoveSelectors: []string{
			".rich_media_tool",
			".share_box",
			".reward_box",
		},
	}

	// CSDN规则 - 增强版本（基于最新页面结构分析）
	e.platformRules[PlatformCSDN] = &PlatformRule{
		Platform: PlatformCSDN,
		TitleSelectors: []string{
			// 最新CSDN页面结构选择器（优先级从高到低）
			".title-article",        // 主要标题选择器
			"h1[data-v-*]",          // Vue组件标题
			".article-header h1",    // 文章头部标题
			".main-content h1",      // 主内容区标题
			"#articleContentId h1",  // 内容ID下的标题
			".blog-content-box h1",  // 博客内容框标题
			"h1.title-article",      // 类名标题
			".article-title-box h1", // 文章标题框
			".main_father .article-title-box h1",
			"[data-v-*] h1",        // Vue数据属性标题
			".article-info-box h1", // 文章信息框标题
			"h1.title",             // 标题类
			".article-title",       // 文章标题
			"#articleContentId",    // 文章内容ID
			".title",               // 通用标题
			"h1",                   // 通用h1备选
		},
		ContentSelectors: []string{
			// 最新内容选择器（按优先级排序）
			"#content_views",              // 主要内容视图（最高优先级）
			".htmledit_views",             // HTML编辑视图
			".markdown_views",             // Markdown视图
			"[data-v-*] .article_content", // Vue组件内容
			".article_content",            // 文章内容
			".blog_content_box",           // 博客内容框
			"[id*='content']",             // 包含content的ID
			".article-content",            // 文章内容
			".blog-content-box",           // 博客内容框
			".article-content-box",        // 文章内容框
			".main-content",               // 主内容
			".content",                    // 通用内容
		},
		AuthorSelectors: []string{
			// 最新作者选择器（基于现代CSDN结构）
			".user-profile .nickname",       // 用户资料昵称
			".article-header .author-name",  // 文章头部作者名
			".blog-author .nickname",        // 博客作者昵称
			".follow-box .name",             // 关注框名称
			"[data-v-*] .username",          // Vue组件用户名
			".user-info .nickname",          // 用户信息昵称
			".blog-content-box .username",   // 博客内容框用户名
			"[data-report-click*='author']", // 作者点击追踪
			".article-bar-top .author",      // 文章顶部栏作者
			".article-info .author",         // 文章信息作者
			".follow-nickName",              // 关注昵称
			".username",                     // 用户名
			".author-name",                  // 作者名
			".name",                         // 名称
			".author",                       // 作者
		},
		DateSelectors: []string{
			// 最新日期选择器（基于现代CSDN结构）
			".publish-time",               // 发布时间
			".article-meta .publish-date", // 文章元数据发布日期
			".article-header .date",       // 文章头部日期
			"[data-v-*] .time",            // Vue组件时间
			".article-bar-top .time",      // 文章顶部栏时间
			".article-info-box .time",     // 文章信息框时间
			"[data-report-click*='time']", // 时间点击追踪
			".article-info .time",         // 文章信息时间
			".blog-content-box .time",     // 博客内容框时间
			"time",                        // 时间元素
			".post-time",                  // 发布时间
			".publish-date",               // 发布日期
			".time",                       // 时间
			".article-time",               // 文章时间
			".publish-time",               // 发布时间
			".date",                       // 日期
		},
		RemoveSelectors: []string{
			// 增强的移除选择器（去除干扰内容）
			"script", "style", "nav", "header", "footer", "aside",
			".hide-article-box",         // 隐藏文章框
			".csdn-tracking-statistics", // CSDN统计追踪
			".recommend-box",            // 推荐框
			".article-footer",           // 文章页脚
			".article-footer-box",       // 文章页脚框
			".signin",                   // 登录框
			".tool-box",                 // 工具箱
			".recommend-item",           // 推荐项
			".advertisement",            // 广告
			".ad-box",                   // 广告框
			".recommend-list",           // 推荐列表
			".comment-box",              // 评论框
			".toolbar",                  // 工具栏
			".share-box",                // 分享框
			".reward-box",               // 打赏框
			".prism-toolbar",            // 代码高亮工具栏
			".hljs-button",              // 代码高亮按钮
			".copy-code-btn",            // 复制代码按钮
			".article-bottom",           // 文章底部
			".side-bar",                 // 侧边栏
			".tool-box",                 // 工具箱
			".comment-box",              // 评论框
			".feed-Sign",                // 订阅签名
			".social-share",             // 社交分享
			".related-articles",         // 相关文章
			".more-articles",            // 更多文章
			"#toolBarBox",               // 工具栏框
			".csdn-common-width",        // CSDN通用宽度（可能包含广告）
		},
	}

	// 哔哩哔哩规则
	e.platformRules[PlatformBilibili] = &PlatformRule{
		Platform: PlatformBilibili,
		TitleSelectors: []string{
			".video-title",
			".article-title",
			"h1.title",
			".title",
		},
		ContentSelectors: []string{
			".article-content",
			".content",
			".desc-info",
			".video-desc",
		},
		AuthorSelectors: []string{
			".up-name",
			".author",
			".username",
		},
		DateSelectors: []string{
			".time",
			".publish-time",
			".create-time",
		},
		RemoveSelectors: []string{
			".video-info",
			".video-data",
			".video-toolbar",
			".video-actions",
		},
	}

	// 简书规则
	e.platformRules[PlatformJianshu] = &PlatformRule{
		Platform: PlatformJianshu,
		TitleSelectors: []string{
			".article h1",
			"._1RuRku",
		},
		ContentSelectors: []string{
			".show-content",
			"._2rhmJa",
		},
		AuthorSelectors: []string{
			".author .name",
			"._1OhGeD",
		},
		RemoveSelectors: []string{
			".follow-detail",
			".support-author",
		},
	}

	// 掘金规则
	e.platformRules[PlatformJuejin] = &PlatformRule{
		Platform: PlatformJuejin,
		TitleSelectors: []string{
			".article-title",
			"h1.article-title",
		},
		ContentSelectors: []string{
			".markdown-body",
			".article-content",
		},
		AuthorSelectors: []string{
			".author-info-box .username",
			".author-name",
		},
		RemoveSelectors: []string{
			".action-box",
			".article-suspended-panel",
		},
	}

	// 博客园规则
	e.platformRules[PlatformCnblogs] = &PlatformRule{
		Platform: PlatformCnblogs,
		TitleSelectors: []string{
			"#cb_post_title_url",
			".postTitle",
		},
		ContentSelectors: []string{
			"#cnblogs_post_body",
			".post-body",
		},
		AuthorSelectors: []string{
			".author",
			"#Header1_HeaderTitle",
		},
		RemoveSelectors: []string{
			".digg",
			".feedback_area_title",
		},
	}

	// GitHub规则
	e.platformRules[PlatformGithub] = &PlatformRule{
		Platform: PlatformGithub,
		TitleSelectors: []string{
			".repository-content h1",
			".js-repo-nav-name",
		},
		ContentSelectors: []string{
			".markdown-body",
			"#readme .Box-body",
		},
		AuthorSelectors: []string{
			".author",
			".commit-author",
		},
		RemoveSelectors: []string{
			".pagehead-actions",
			".file-navigation",
		},
	}
}

// ExtractContent 提取网页内容
func (e *ContentExtractor) ExtractContent(doc *goquery.Document, platform Platform) (*ExtractedContent, error) {
	fmt.Printf("\n🎯 内容提取器调试开始\n")
	fmt.Printf("📊 平台类型: %v\n", platform)

	content := &ExtractedContent{
		Platform: platform,
	}

	// 获取平台规则
	fmt.Printf("\n🔧 获取平台规则\n")
	rule := e.platformRules[platform]
	if rule == nil {
		fmt.Printf("⚠️ 未找到平台特定规则，使用通用规则\n")
		rule = e.getGenericRule()
	} else {
		fmt.Printf("✅ 找到平台特定规则\n")
		fmt.Printf("   标题选择器: %v\n", rule.TitleSelectors)
		fmt.Printf("   内容选择器: %v\n", rule.ContentSelectors)
		fmt.Printf("   作者选择器: %v\n", rule.AuthorSelectors)
	}

	// 提取标题
	fmt.Printf("\n📑 提取标题\n")
	content.Title = e.extractTitle(doc, rule)
	fmt.Printf("✅ 标题: %s\n", content.Title)

	// 提取正文内容
	fmt.Printf("\n📝 提取正文内容\n")
	content.Content = e.extractMainContent(doc, rule)
	fmt.Printf("✅ 内容长度: %d 字符\n", len(content.Content))
	if len(content.Content) > 200 {
		fmt.Printf("📄 内容预览: %s...\n", content.Content[:200])
	} else {
		fmt.Printf("📄 完整内容: %s\n", content.Content)
	}

	// 提取作者
	fmt.Printf("\n👤 提取作者\n")
	content.Author = e.extractAuthor(doc, rule)
	fmt.Printf("✅ 作者: %s\n", content.Author)

	// 提取发布时间
	fmt.Printf("\n📅 提取发布时间\n")
	content.PublishedTime = e.extractPublishedTime(doc, rule)
	fmt.Printf("✅ 发布时间: %s\n", content.PublishedTime)

	// 提取描述
	fmt.Printf("\n📋 提取描述\n")
	content.Description = e.extractDescription(doc)
	fmt.Printf("✅ 描述: %s\n", truncateText(content.Description, 100))

	// 提取关键词
	fmt.Printf("\n🏷️ 提取关键词\n")
	content.Keywords = e.extractKeywords(doc)
	fmt.Printf("✅ 关键词(%d个): %v\n", len(content.Keywords), content.Keywords)

	// 提取图片
	fmt.Printf("\n🖼️ 提取图片\n")
	content.Images = e.extractImages(doc, rule)
	fmt.Printf("✅ 图片数量: %d 张\n", len(content.Images))
	for i, img := range content.Images {
		if i < 3 { // 只显示前3张图片信息
			fmt.Printf("   图片%d: URL=%s, Alt=%s\n", i+1, img.URL, img.Alt)
		}
	}

	// 提取视频
	fmt.Printf("\n🎬 提取视频\n")
	content.Videos = e.extractVideos(doc, rule)
	fmt.Printf("✅ 视频数量: %d 个\n", len(content.Videos))

	// 提取语言
	fmt.Printf("\n🌐 提取语言\n")
	content.Language = e.extractLanguage(doc)
	fmt.Printf("✅ 语言: %s\n", content.Language)

	// 计算字数和阅读时间
	fmt.Printf("\n📊 计算统计信息\n")
	content.WordCount = len([]rune(content.Content))
	content.ReadTime = e.calculateReadTime(content.WordCount)
	fmt.Printf("✅ 字数: %d 字\n", content.WordCount)
	fmt.Printf("✅ 预计阅读时间: %d 分钟\n", content.ReadTime)

	// 清理内容
	fmt.Printf("\n🧹 清理内容\n")
	content = e.cleanupContent(content, rule)
	fmt.Printf("✅ 内容清理完成\n")

	fmt.Printf("\n🎉 内容提取器完成!\n")
	return content, nil
}

// extractTitle 提取标题
func (e *ContentExtractor) extractTitle(doc *goquery.Document, rule *PlatformRule) string {
	selectors := rule.TitleSelectors
	if len(selectors) == 0 {
		selectors = e.titleSelectors
	}

	for _, selector := range selectors {
		if title := doc.Find(selector).First().Text(); title != "" {
			return strings.TrimSpace(title)
		}
	}

	// 如果都没找到，尝试从title标签提取
	if title := doc.Find("title").Text(); title != "" {
		return strings.TrimSpace(title)
	}

	return ""
}

// extractMainContent 提取正文内容
func (e *ContentExtractor) extractMainContent(doc *goquery.Document, rule *PlatformRule) string {
	// 先移除不需要的元素
	for _, selector := range rule.RemoveSelectors {
		doc.Find(selector).Remove()
	}

	selectors := rule.ContentSelectors
	if len(selectors) == 0 {
		selectors = e.contentSelectors
	}

	var content strings.Builder

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			if text != "" {
				content.WriteString(strings.TrimSpace(text))
				content.WriteString("\n\n")
			}
		})

		if content.Len() > 0 {
			break
		}
	}

	// 如果还是没有内容，尝试提取body中的文本
	if content.Len() == 0 {
		bodyText := doc.Find("body").Text()
		content.WriteString(strings.TrimSpace(bodyText))
	}

	return strings.TrimSpace(content.String())
}

// extractAuthor 提取作者
func (e *ContentExtractor) extractAuthor(doc *goquery.Document, rule *PlatformRule) string {
	selectors := rule.AuthorSelectors
	if len(selectors) == 0 {
		selectors = e.authorSelectors
	}

	for _, selector := range selectors {
		if author := doc.Find(selector).First().Text(); author != "" {
			return strings.TrimSpace(author)
		}
	}

	// 尝试从meta标签提取
	if author := doc.Find("meta[name='author']").AttrOr("content", ""); author != "" {
		return strings.TrimSpace(author)
	}

	return ""
}

// extractPublishedTime 提取发布时间
func (e *ContentExtractor) extractPublishedTime(doc *goquery.Document, rule *PlatformRule) string {
	for _, selector := range rule.DateSelectors {
		if dateText := doc.Find(selector).First().Text(); dateText != "" {
			// 清理和格式化时间
			dateText = strings.TrimSpace(dateText)
			if parsedTime := e.parseTime(dateText); parsedTime != "" {
				return parsedTime
			}
		}
	}

	// 尝试从meta标签提取
	metaSelectors := []string{
		"meta[property='article:published_time']",
		"meta[name='publishdate']",
		"meta[name='date']",
		"time[datetime]",
	}

	for _, selector := range metaSelectors {
		if datetime := doc.Find(selector).AttrOr("content", ""); datetime != "" {
			if parsedTime := e.parseTime(datetime); parsedTime != "" {
				return parsedTime
			}
		}
		if datetime := doc.Find(selector).AttrOr("datetime", ""); datetime != "" {
			if parsedTime := e.parseTime(datetime); parsedTime != "" {
				return parsedTime
			}
		}
	}

	return ""
}

// extractDescription 提取描述
func (e *ContentExtractor) extractDescription(doc *goquery.Document) string {
	// 从meta标签提取
	metaSelectors := []string{
		"meta[name='description']",
		"meta[property='og:description']",
		"meta[name='twitter:description']",
	}

	for _, selector := range metaSelectors {
		if desc := doc.Find(selector).AttrOr("content", ""); desc != "" {
			return strings.TrimSpace(desc)
		}
	}

	return ""
}

// extractKeywords 提取关键词
func (e *ContentExtractor) extractKeywords(doc *goquery.Document) []string {
	fmt.Printf("🏷️ 尝试从meta标签提取关键词...\n")

	// 1. 优先从meta标签提取
	keywords := doc.Find("meta[name='keywords']").AttrOr("content", "")
	if keywords != "" {
		// 分割关键词
		keywordList := strings.Split(keywords, ",")
		result := make([]string, 0, len(keywordList))

		for _, keyword := range keywordList {
			keyword = strings.TrimSpace(keyword)
			if keyword != "" {
				result = append(result, keyword)
			}
		}

		if len(result) > 0 {
			fmt.Printf("✅ 从meta标签提取到 %d 个关键词: %v\n", len(result), result)
			return result
		}
	}

	fmt.Printf("📄 meta标签无关键词，尝试从内容提取...\n")

	// 2. 从页面内容提取
	contentText := doc.Find("body").Text()
	if contentText == "" {
		fmt.Printf("⚠️ 无法获取页面内容进行关键词提取\n")
		return []string{}
	}

	// 3. 使用基础词频分析提取关键词
	keywordsList := e.extractKeywordsFromContent(contentText)

	if len(keywordsList) > 0 {
		fmt.Printf("✅ 从内容提取到 %d 个关键词: %v\n", len(keywordsList), keywordsList)
	} else {
		fmt.Printf("⚠️ 内容关键词提取失败\n")
	}

	return keywordsList
}

// extractKeywordsFromContent 从内容中提取关键词 - 增强版
func (e *ContentExtractor) extractKeywordsFromContent(content string) []string {
	fmt.Printf("🔍 开始从内容提取关键词...\n")

	if len(content) == 0 {
		fmt.Printf("⚠️ 内容为空，无法提取关键词\n")
		return []string{}
	}

	// 预处理内容
	cleanContent := e.preprocessContentForKeywords(content)
	if len(cleanContent) < 10 {
		fmt.Printf("⚠️ 清理后内容太短，无法提取关键词\n")
		return []string{}
	}

	// 分词和统计
	wordFreq := e.analyzeWordFrequency(cleanContent)

	// 过滤和排序
	candidates := e.filterAndRankKeywords(wordFreq)

	// 质量控制
	finalKeywords := e.qualityControlKeywords(candidates)

	fmt.Printf("✅ 从内容提取完成，获得 %d 个高质量关键词\n", len(finalKeywords))
	return finalKeywords
}

// preprocessContentForKeywords 预处理内容用于关键词提取
func (e *ContentExtractor) preprocessContentForKeywords(content string) string {
	// 移除HTML标签（如果有残留）
	content = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, " ")

	// 移除特殊字符但保留中英文和数字
	content = regexp.MustCompile(`[^\p{L}\p{N}\s]`).ReplaceAllString(content, " ")

	// 标准化空白字符
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	// 转换为小写（保留原始大小写信息用于后续处理）
	return content
}

// analyzeWordFrequency 分析词频
func (e *ContentExtractor) analyzeWordFrequency(content string) map[string]int {
	wordMap := make(map[string]int)

	// 扩展的停用词列表
	stopWords := e.getEnhancedStopWords()

	// 分词处理
	words := strings.Fields(strings.ToLower(content))

	for _, word := range words {
		word = strings.TrimSpace(word)

		// 基础过滤条件
		if !e.isValidKeywordCandidate(word, stopWords) {
			continue
		}

		// 检查是否为乱码
		if e.containsGarbledChars(word) {
			continue
		}

		wordMap[word]++
	}

	return wordMap
}

// getEnhancedStopWords 获取增强的停用词列表
func (e *ContentExtractor) getEnhancedStopWords() map[string]bool {
	return map[string]bool{
		// 中文停用词
		"的": true, "了": true, "在": true, "是": true, "我": true, "有": true, "和": true, "就": true,
		"不": true, "人": true, "都": true, "一": true, "一个": true, "上": true, "也": true, "很": true,
		"到": true, "说": true, "要": true, "去": true, "你": true, "会": true, "着": true, "没有": true,
		"看": true, "好": true, "自己": true, "这": true, "那": true, "而": true, "但": true, "如果": true,
		"可以": true, "这个": true, "那个": true, "什么": true, "怎么": true, "为什么": true, "因为": true,
		"所以": true, "然后": true, "但是": true, "或者": true, "还是": true, "只是": true, "已经": true,
		"一些": true, "这些": true, "那些": true, "还有": true, "比如": true, "例如": true, "通过": true,

		// 英文停用词
		"the": true, "and": true, "or": true, "but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"will": true, "would": true, "could": true, "should": true, "may": true, "might": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "a": true, "an": true, "as": true,
		"from": true, "up": true, "out": true, "down": true, "off": true, "over": true, "under": true,
		"again": true, "further": true, "then": true, "once": true, "here": true, "there": true, "when": true,
		"where": true, "why": true, "how": true, "all": true, "any": true, "both": true, "each": true,
		"few": true, "more": true, "most": true, "other": true, "some": true, "such": true, "only": true,

		// 通用无意义词
		"nbsp": true, "amp": true, "quot": true, "lt": true, "gt": true, "copy": true, "reg": true,
		"trade": true, "hellip": true, "mdash": true, "ndash": true, "lsquo": true, "rsquo": true,

		// 数字和符号
		"0": true, "1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true, "9": true,
	}
}

// isValidKeywordCandidate 检查是否为有效的关键词候选
func (e *ContentExtractor) isValidKeywordCandidate(word string, stopWords map[string]bool) bool {
	// 长度检查
	if len(word) < 2 || len(word) > 30 {
		return false
	}

	// 停用词检查
	if stopWords[word] {
		return false
	}

	// 纯数字检查
	if regexp.MustCompile(`^\d+$`).MatchString(word) {
		return false
	}

	// 纯符号检查
	if regexp.MustCompile(`^[^\p{L}\p{N}]+$`).MatchString(word) {
		return false
	}

	// 检查是否包含有意义的字符
	if !regexp.MustCompile(`[\p{L}\p{N}]`).MatchString(word) {
		return false
	}

	return true
}

// containsGarbledChars 检查是否包含乱码字符
func (e *ContentExtractor) containsGarbledChars(word string) bool {
	garbledCount := 0
	totalRunes := 0

	for _, r := range word {
		totalRunes++
		// 检查替换字符、无效字符和异常控制字符
		if r == '\uFFFD' || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			garbledCount++
		}

		// 检查是否为疑似乱码的字符组合
		if e.isSuspiciousChar(r) {
			garbledCount++
		}
	}

	// 如果乱码字符比例超过50%，认为是乱码
	if totalRunes > 0 && float64(garbledCount)/float64(totalRunes) > 0.5 {
		return true
	}

	return false
}

// isSuspiciousChar 检查是否为可疑字符
func (e *ContentExtractor) isSuspiciousChar(r rune) bool {
	// 一些常见的乱码字符模式
	suspiciousRanges := [][]rune{
		{0x00, 0x1F},     // 控制字符
		{0x7F, 0x9F},     // 扩展控制字符
		{0xFFF0, 0xFFFF}, // 特殊字符
	}

	for _, rang := range suspiciousRanges {
		if r >= rang[0] && r <= rang[1] {
			return true
		}
	}

	return false
}

// filterAndRankKeywords 过滤和排序关键词
func (e *ContentExtractor) filterAndRankKeywords(wordFreq map[string]int) []KeywordCandidate {
	var candidates []KeywordCandidate

	for word, count := range wordFreq {
		// 频率阈值：至少出现2次
		if count < 2 {
			continue
		}

		// 计算综合评分
		score := e.calculateKeywordScore(word, count)

		candidates = append(candidates, KeywordCandidate{
			Word:  word,
			Count: count,
			Score: score,
		})
	}

	// 按评分排序
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].Score < candidates[j].Score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	return candidates
}

// calculateKeywordScore 计算关键词评分
func (e *ContentExtractor) calculateKeywordScore(word string, frequency int) float64 {
	score := float64(frequency)

	// 长度加分：适中长度的词得分更高
	wordLen := len([]rune(word))
	if wordLen >= 3 && wordLen <= 10 {
		score *= 1.2
	} else if wordLen > 10 {
		score *= 0.8
	}

	// 内容质量加分：技术术语、专业词汇
	if e.isTechnicalTerm(word) {
		score *= 1.5
	}

	// 英文单词完整性加分
	if e.isCompleteEnglishWord(word) {
		score *= 1.3
	}

	// 中文词汇完整性加分
	if e.isCompleteChinese(word) {
		score *= 1.2
	}

	return score
}

// isTechnicalTerm 检查是否为技术术语
func (e *ContentExtractor) isTechnicalTerm(word string) bool {
	technicalTerms := map[string]bool{
		"mysql": true, "database": true, "sql": true, "table": true, "index": true,
		"query": true, "select": true, "insert": true, "update": true, "delete": true,
		"varchar": true, "int": true, "char": true, "text": true, "blob": true,
		"primary": true, "foreign": true, "key": true, "constraint": true, "join": true,
		"数据库": true, "数据": true, "表": true, "字段": true, "索引": true,
		"查询": true, "插入": true, "更新": true, "删除": true, "主键": true,
		"外键": true, "约束": true, "连接": true, "事务": true, "存储": true,
		// 可以根据具体领域扩展
	}

	return technicalTerms[strings.ToLower(word)]
}

// isCompleteEnglishWord 检查是否为完整的英文单词
func (e *ContentExtractor) isCompleteEnglishWord(word string) bool {
	// 基本检查：只包含英文字母
	if !regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(word) {
		return false
	}

	// 长度检查
	if len(word) < 3 || len(word) > 20 {
		return false
	}

	// 检查是否有合理的元音分布
	vowels := "aeiouAEIOU"
	hasVowel := false
	for _, char := range word {
		if strings.ContainsRune(vowels, char) {
			hasVowel = true
			break
		}
	}

	return hasVowel
}

// isCompleteChinese 检查是否为完整的中文词汇
func (e *ContentExtractor) isCompleteChinese(word string) bool {
	// 检查是否只包含中文字符
	for _, r := range word {
		if r < 0x4e00 || r > 0x9fff {
			return false
		}
	}

	// 中文词汇长度通常在1-8个字符之间
	runeCount := len([]rune(word))
	return runeCount >= 1 && runeCount <= 8
}

// qualityControlKeywords 关键词质量控制
func (e *ContentExtractor) qualityControlKeywords(candidates []KeywordCandidate) []string {
	maxKeywords := 15
	var result []string

	addedWords := make(map[string]bool)

	for _, candidate := range candidates {
		if len(result) >= maxKeywords {
			break
		}

		word := candidate.Word

		// 避免重复
		if addedWords[word] {
			continue
		}

		// 最终质量检查
		if !e.finalQualityCheck(word) {
			continue
		}

		result = append(result, word)
		addedWords[word] = true
	}

	fmt.Printf("📊 关键词质量控制完成：候选 %d 个，通过 %d 个\n", len(candidates), len(result))
	return result
}

// finalQualityCheck 最终质量检查
func (e *ContentExtractor) finalQualityCheck(word string) bool {
	// 再次检查乱码
	if e.containsGarbledChars(word) {
		return false
	}

	// 检查是否包含无意义的字符序列
	if e.containsMeaninglessPattern(word) {
		return false
	}

	// 检查长度合理性
	runeCount := len([]rune(word))
	if runeCount < 2 || runeCount > 20 {
		return false
	}

	return true
}

// containsMeaninglessPattern 检查是否包含无意义的模式
func (e *ContentExtractor) containsMeaninglessPattern(word string) bool {
	// 检查重复字符（如：aaaa, 1111）
	// Go正则表达式不支持\1语法，使用简单的字符重复检测
	if len(word) >= 4 {
		for i := 0; i <= len(word)-4; i++ {
			if word[i] == word[i+1] && word[i+1] == word[i+2] && word[i+2] == word[i+3] {
				return true
			}
		}
	}

	// 检查无意义的字符组合
	meaninglessPatterns := []string{
		"xxx", "yyy", "zzz", "000", "111", "aaa", "bbb",
		"test", "demo", "example", "sample",
	}

	lowerWord := strings.ToLower(word)
	for _, pattern := range meaninglessPatterns {
		if strings.Contains(lowerWord, pattern) {
			return true
		}
	}

	return false
}

type KeywordCandidate struct {
	Word  string
	Count int
	Score float64
}

// extractImages 提取图片
func (e *ContentExtractor) extractImages(doc *goquery.Document, rule *PlatformRule) []ImageInfo {
	var images []ImageInfo

	selectors := rule.ImageSelectors
	if len(selectors) == 0 {
		selectors = []string{"img"}
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			src := s.AttrOr("src", "")
			if src == "" {
				src = s.AttrOr("data-src", "") // 懒加载图片
			}

			if src != "" {
				image := ImageInfo{
					URL:   src,
					Alt:   s.AttrOr("alt", ""),
					Title: s.AttrOr("title", ""),
				}

				// 尝试获取尺寸
				if width := s.AttrOr("width", ""); width != "" {
					// 可以解析width
				}
				if height := s.AttrOr("height", ""); height != "" {
					// 可以解析height
				}

				images = append(images, image)
			}
		})
	}

	return images
}

// extractVideos 提取视频
func (e *ContentExtractor) extractVideos(doc *goquery.Document, rule *PlatformRule) []VideoInfo {
	var videos []VideoInfo

	selectors := rule.VideoSelectors
	if len(selectors) == 0 {
		selectors = []string{"video", "iframe[src*='video']", "iframe[src*='bilibili']", "iframe[src*='youtube']"}
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			src := s.AttrOr("src", "")
			if src != "" {
				video := VideoInfo{
					URL:      src,
					Title:    s.AttrOr("title", ""),
					Poster:   s.AttrOr("poster", ""),
					Duration: s.AttrOr("duration", ""),
				}
				videos = append(videos, video)
			}
		})
	}

	return videos
}

// extractLanguage 提取语言
func (e *ContentExtractor) extractLanguage(doc *goquery.Document) string {
	// 从html标签提取
	if lang := doc.Find("html").AttrOr("lang", ""); lang != "" {
		return lang
	}

	// 从meta标签提取
	if lang := doc.Find("meta[http-equiv='content-language']").AttrOr("content", ""); lang != "" {
		return lang
	}

	return "zh-CN" // 默认中文
}

// calculateReadTime 计算阅读时间
func (e *ContentExtractor) calculateReadTime(wordCount int) int {
	// 假设中文阅读速度是每分钟300字
	const wordsPerMinute = 300
	readTime := wordCount / wordsPerMinute
	if readTime < 1 {
		readTime = 1
	}
	return readTime
}

// parseTime 解析时间字符串
func (e *ContentExtractor) parseTime(timeStr string) string {
	// 常见的时间格式
	timeFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
		"01-02 15:04",
	}

	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	// 如果都解析不了，返回原字符串
	return timeStr
}

// cleanupContent 清理内容
func (e *ContentExtractor) cleanupContent(content *ExtractedContent, rule *PlatformRule) *ExtractedContent {
	// 清理标题
	content.Title = e.cleanupText(content.Title)

	// 清理正文
	content.Content = e.cleanupText(content.Content)

	// 清理作者
	content.Author = e.cleanupText(content.Author)

	// 应用平台特定的清理规则
	for _, cleanupRule := range rule.CleanupRules {
		content.Content = e.applyCleanupRule(content.Content, cleanupRule)
	}

	return content
}

// cleanupText 清理文本
func (e *ContentExtractor) cleanupText(text string) string {
	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// 移除首尾空白
	text = strings.TrimSpace(text)

	// 移除HTML标签
	text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")

	return text
}

// applyCleanupRule 应用清理规则
func (e *ContentExtractor) applyCleanupRule(text string, rule CleanupRule) string {
	switch rule.Type {
	case "regex":
		re := regexp.MustCompile(rule.Pattern)
		return re.ReplaceAllString(text, rule.Replace)
	case "replace":
		return strings.ReplaceAll(text, rule.Pattern, rule.Replace)
	case "remove":
		return strings.ReplaceAll(text, rule.Pattern, "")
	}
	return text
}

// getGenericRule 获取通用规则
func (e *ContentExtractor) getGenericRule() *PlatformRule {
	return &PlatformRule{
		Platform:         PlatformGeneric,
		TitleSelectors:   e.titleSelectors,
		ContentSelectors: e.contentSelectors,
		AuthorSelectors:  e.authorSelectors,
		RemoveSelectors: []string{
			"script",
			"style",
			"nav",
			"footer",
			"header",
			".advertisement",
			".ads",
			".sidebar",
		},
	}
}

// truncateText 截断文本到指定长度
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}

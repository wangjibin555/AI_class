package parser

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// KeywordExtractor 关键词提取器
type KeywordExtractor struct {
	stopWords        map[string]bool
	minWordLen       int
	technicalTerms   *TechnicalTerms
	codePatterns     []*regexp.Regexp
	importantPhrases map[string]float64
}

// TechnicalTerms 技术术语词典
type TechnicalTerms struct {
	ProgrammingLanguages map[string]float64
	DatabaseTerms        map[string]float64
	WebTechnologies      map[string]float64
	Frameworks           map[string]float64
	ToolsAndSoftware     map[string]float64
	ConceptualTerms      map[string]float64
	// 新增学科专业术语
	MathTerms     map[string]float64 // 数学术语
	ChineseTerms  map[string]float64 // 语文术语
	EnglishTerms  map[string]float64 // 英语术语
	AcademicTerms map[string]float64 // 学术术语
}

// DocumentStructureAnalyzer 文档结构分析器
type DocumentStructureAnalyzer struct {
	titlePatterns   []*regexp.Regexp
	headingPatterns []*regexp.Regexp
}

// Section 章节结构
type Section struct {
	Title    string
	Content  string
	Type     string
	Level    int
	Keywords []string
}

// KeyInfo 关键信息
type KeyInfo struct {
	Title      string
	Summary    string
	Keywords   []string
	MainPoints []string
	Topics     []string
	Difficulty string
	Duration   int
	Structure  []string
}

// StructuredContent 结构化内容
type StructuredContent struct {
	OriginalText string
	CleanText    string
	Sections     []Section
	KeyInfo      *KeyInfo
}

// TextSegment 文本片段
type TextSegment struct {
	Text       string
	Topic      string
	Importance float64
	Keywords   []string
	Summary    string
}

// NewEnhancedKeywordExtractor 创建增强关键词提取器
func NewEnhancedKeywordExtractor() *KeywordExtractor {
	stopWords := map[string]bool{
		"的": true, "是": true, "在": true, "有": true, "和": true, "了": true,
		"与": true, "及": true, "等": true, "但": true, "或": true, "如": true,
		"则": true, "将": true, "要": true, "可": true, "能": true, "会": true,
		"很": true, "更": true, "最": true, "非": true, "不": true, "无": true,
		"这": true, "那": true, "其": true, "此": true, "之": true, "以": true,
		"为": true, "从": true, "向": true, "到": true, "由": true, "于": true,
		"进行": true, "包含": true, "具有": true, "属于": true, "通过": true,
		"我们": true, "可以": true, "应该": true, "需要": true, "因为": true,
	}

	// 初始化代码检测模式
	codePatterns := []*regexp.Regexp{
		regexp.MustCompile(`func\s+\w+\(`),                // Go函数
		regexp.MustCompile(`package\s+\w+`),               // Go包声明
		regexp.MustCompile(`import\s+["\(]`),              // 导入语句
		regexp.MustCompile(`var\s+\w+\s*=`),               // 变量声明
		regexp.MustCompile(`<[a-zA-Z]+[^>]*>`),            // XML/HTML标签
		regexp.MustCompile(`\w+\s*:\s*\w+`),               // 配置项
		regexp.MustCompile(`SELECT|INSERT|UPDATE|DELETE`), // SQL语句
		regexp.MustCompile(`\$\{[^}]+\}`),                 // ${} 占位符
		regexp.MustCompile(`#\{[^}]+\}`),                  // #{} 占位符
		// 数学公式模式
		regexp.MustCompile(`[∫∑∏∂∇∆∞±×÷≠≤≥√∈∉∪∩⊆⊇]`), // 数学符号
		regexp.MustCompile(`f\([x-z]\)`), // 函数表示
		regexp.MustCompile(`\d+\^\d+`),   // 幂次表示
	}

	// 重要短语权重
	importantPhrases := map[string]float64{
		"重要提示": 1.8, "关键概念": 1.8, "核心原理": 1.8, "主要特点": 1.6,
		"最佳实践": 1.7, "性能优化": 1.7, "安全防护": 1.7, "注意事项": 1.6,
		"实战案例": 1.5, "代码示例": 1.5, "配置说明": 1.4, "使用方法": 1.4,
		"技术要点": 1.6, "解决方案": 1.5, "常见问题": 1.4, "错误处理": 1.5,
		"工作原理": 1.7, "实现机制": 1.6, "设计模式": 1.6, "架构设计": 1.7,
		// 学科相关短语
		"重点知识": 1.8, "基础概念": 1.7, "核心内容": 1.8, "学习要点": 1.6,
		"解题方法": 1.7, "答题技巧": 1.6, "知识点": 1.6, "考试重点": 1.7,
		"语法规则": 1.7, "修辞手法": 1.6, "文学常识": 1.6, "写作技巧": 1.6,
		"单词记忆": 1.6, "语法结构": 1.7, "听力技巧": 1.5, "口语表达": 1.5,
	}

	extractor := &KeywordExtractor{
		stopWords:        stopWords,
		minWordLen:       2,
		codePatterns:     codePatterns,
		importantPhrases: importantPhrases,
	}

	// 初始化技术术语词典
	extractor.initEnhancedTechnicalTerms()

	return extractor
}

// NewEnhancedDocumentStructureAnalyzer 创建增强文档结构分析器
func NewEnhancedDocumentStructureAnalyzer() *DocumentStructureAnalyzer {
	titlePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[第一二三四五六七八九十0-9]+[章节部分课时]\s*[：:]\s*(.+)`),
		regexp.MustCompile(`^[0-9]+[\.、]\s*(.+)`),
		regexp.MustCompile(`^[一二三四五六七八九十][、\.]\s*(.+)`),
		// 技术文档特有模式
		regexp.MustCompile(`^[0-9]+\.[0-9]+\s*(.+)`),      // 1.1 格式
		regexp.MustCompile(`^[#]+\s*(.+)`),                // Markdown标题
		regexp.MustCompile(`^[A-Z][a-zA-Z\s]+[：:]`),       // 英文标题
		regexp.MustCompile(`^[配置|实现|优化|使用|安装|部署]\s*(.+)`), // 技术操作标题
		regexp.MustCompile(`^[问题|解决|方案|总结|结论]\s*(.+)`),    // 问题解决模式
		// 学科教学模式
		regexp.MustCompile(`^[知识点|要点|重点|难点|考点]\s*[：:]\s*(.+)`), // 教学要点
		regexp.MustCompile(`^[例题|练习|习题|作业]\s*[：:0-9]*\s*(.+)`), // 题目模式
		regexp.MustCompile(`^[定理|公式|法则|规律]\s*[：:0-9]*\s*(.+)`), // 数学定理
	}

	headingPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[0-9]+\.[0-9]+\s*(.+)`),
		regexp.MustCompile(`^[（(][0-9一二三四五六七八九十]+[）)]\s*(.+)`),
		regexp.MustCompile(`^[①②③④⑤⑥⑦⑧⑨⑩]\s*(.+)`),
		// 技术文档小标题
		regexp.MustCompile(`^[步骤|方法|技巧|要点][0-9]*[：:]\s*(.+)`),
		regexp.MustCompile(`^[注意|提示|警告|重要][：:]\s*(.+)`),
		// 学科小标题
		regexp.MustCompile(`^[基础|进阶|高级|入门]\s*(.+)`),
		regexp.MustCompile(`^[语法|词汇|阅读|写作|听力|口语]\s*(.+)`),
	}

	return &DocumentStructureAnalyzer{
		titlePatterns:   titlePatterns,
		headingPatterns: headingPatterns,
	}
}

// initEnhancedTechnicalTerms 初始化增强技术术语词典
func (k *KeywordExtractor) initEnhancedTechnicalTerms() {
	k.technicalTerms = &TechnicalTerms{
		ProgrammingLanguages: map[string]float64{
			"Go": 2.2, "golang": 2.2, "Java": 2.0, "Python": 2.0, "JavaScript": 2.0,
			"C++": 2.0, "C#": 2.0, "PHP": 2.0, "Ruby": 2.0, "Kotlin": 2.0,
			"TypeScript": 2.0, "Rust": 2.0, "Swift": 2.0, "Scala": 2.0,
			"编程语言": 1.8, "程序设计": 1.8, "编程": 1.7, "代码": 1.6,
		},
		DatabaseTerms: map[string]float64{
			"MySQL": 2.0, "PostgreSQL": 1.9, "MongoDB": 1.8, "Redis": 1.9,
			"Oracle": 1.8, "SQLServer": 1.8, "SQLite": 1.8, "Elasticsearch": 1.8,
			"数据库": 1.8, "索引": 1.7, "查询": 1.6, "事务": 1.8, "连接池": 1.9,
			"SQL": 2.0, "NoSQL": 1.8, "ACID": 1.8, "ORM": 1.7, "MyBatis": 2.1,
			"Hibernate": 1.7, "JPA": 1.7, "GORM": 1.8, "SQL注入": 2.2, "预编译": 2.0,
			"动态SQL": 2.1, "模糊查询": 1.8, "索引失效": 1.9, "连接池配置": 2.0,
			"PreparedStatement": 1.9, "占位符": 1.8, "参数化查询": 1.9,
		},
		WebTechnologies: map[string]float64{
			"HTTP": 1.8, "HTTPS": 1.8, "REST": 1.8, "GraphQL": 1.7, "gRPC": 1.7,
			"WebSocket": 1.7, "JSON": 1.7, "XML": 1.7, "HTML": 1.6, "CSS": 1.6,
			"API": 1.9, "接口": 1.7, "微服务": 1.9, "分布式": 1.9, "负载均衡": 1.8,
			"缓存": 1.7, "中间件": 1.7, "网关": 1.7, "认证": 1.7, "授权": 1.7,
		},
		Frameworks: map[string]float64{
			"Spring": 1.9, "SpringBoot": 1.9, "Gin": 1.9, "Echo": 1.8, "Fiber": 1.8,
			"Express": 1.7, "Django": 1.7, "Flask": 1.7, "React": 1.7, "Vue": 1.7,
			"Angular": 1.7, "Laravel": 1.7, "Rails": 1.7, "框架": 1.6,
		},
		ToolsAndSoftware: map[string]float64{
			"Docker": 1.9, "Kubernetes": 1.9, "Git": 1.8, "Maven": 1.7, "Gradle": 1.7,
			"Jenkins": 1.7, "CI/CD": 1.8, "部署": 1.6, "测试": 1.5, "调试": 1.5,
			"监控": 1.6, "日志": 1.5, "配置": 1.5, "环境": 1.4,
			"HikariCP": 1.9, "Druid": 1.9, "连接池配置": 1.9, "数据源": 1.7,
		},
		ConceptualTerms: map[string]float64{
			"算法": 1.7, "数据结构": 1.7, "设计模式": 1.7, "架构": 1.7, "并发": 1.8,
			"线程": 1.7, "进程": 1.7, "协程": 1.8, "goroutine": 2.0, "channel": 2.0,
			"内存": 1.6, "性能": 1.7, "优化": 1.6, "安全": 1.7, "加密": 1.7,
			"注入": 1.9, "防护": 1.7, "漏洞": 1.8, "预编译": 1.9, "动态SQL": 2.0,
			"模糊查询": 1.7, "索引失效": 1.8, "连接池配置": 1.9, "高并发": 1.9,
			"实战": 1.5, "教程": 1.4, "指南": 1.4, "入门": 1.4, "进阶": 1.5,
			"函数": 1.6, "方法": 1.6, "变量": 1.5, "类型": 1.5, "接口": 1.6,
			"包": 1.5, "模块": 1.5, "导入": 1.5, "声明": 1.5, "定义": 1.6,
			"工作原理": 1.8, "实现机制": 1.7, "核心概念": 1.8, "最佳实践": 1.8,
		},
		// 数学术语词典
		MathTerms: map[string]float64{
			// 基础数学概念
			"数学": 2.0, "高等数学": 2.2, "微积分": 2.1, "线性代数": 2.0, "概率论": 2.0,
			"数理统计": 1.9, "离散数学": 1.9, "数值分析": 1.8, "运筹学": 1.8,
			// 函数与极限
			"函数": 1.9, "极限": 1.9, "连续": 1.8, "间断": 1.7, "单调性": 1.7,
			"有界性": 1.7, "奇偶性": 1.7, "周期性": 1.7, "反函数": 1.8,
			"复合函数": 1.8, "初等函数": 1.7, "特殊函数": 1.7,
			// 导数与微分
			"导数": 1.9, "微分": 1.9, "可导": 1.8, "导函数": 1.8, "求导": 1.7,
			"链式法则": 1.8, "隐函数": 1.8, "参数方程": 1.8, "切线": 1.7,
			"法线": 1.7, "中值定理": 1.9, "罗尔定理": 1.8, "拉格朗日": 1.8,
			// 积分
			"积分": 1.9, "不定积分": 1.8, "定积分": 1.8, "换元": 1.7, "分部积分": 1.8,
			"牛顿莱布尼茨": 1.8, "面积": 1.6, "体积": 1.6, "弧长": 1.7,
			// 级数
			"级数": 1.8, "收敛": 1.8, "发散": 1.8, "幂级数": 1.8, "泰勒级数": 1.9,
			"麦克劳林": 1.8, "傅里叶级数": 1.9,
			// 多元函数
			"多元函数": 1.8, "偏导数": 1.8, "全微分": 1.8, "梯度": 1.8, "方向导数": 1.8,
			"二重积分": 1.8, "三重积分": 1.8, "曲线积分": 1.8, "曲面积分": 1.8,
			// 微分方程
			"微分方程": 1.9, "常微分方程": 1.8, "偏微分方程": 1.8, "齐次": 1.7,
			"非齐次": 1.7, "通解": 1.7, "特解": 1.7, "初值问题": 1.7,
			// 数学符号和术语
			"无穷小": 1.8, "无穷大": 1.8, "等价无穷小": 1.8, "夹逼定理": 1.8,
			"单调有界": 1.8, "洛必达": 1.8, "泰勒展开": 1.8,
		},
		// 语文术语词典
		ChineseTerms: map[string]float64{
			// 语文基础
			"语文": 2.0, "汉语": 1.9, "中文": 1.8, "国语": 1.8, "母语": 1.7,
			"语言文字": 1.8, "汉字": 1.7, "拼音": 1.6, "笔画": 1.5, "偏旁": 1.5,
			// 语法知识
			"语法": 1.8, "词性": 1.7, "名词": 1.6, "动词": 1.6, "形容词": 1.6,
			"副词": 1.6, "介词": 1.6, "连词": 1.6, "助词": 1.6, "叹词": 1.6,
			"主语": 1.7, "谓语": 1.7, "宾语": 1.7, "定语": 1.7, "状语": 1.7,
			"补语": 1.7, "句子成分": 1.7, "句型": 1.7, "语序": 1.6,
			// 修辞手法
			"修辞": 1.8, "比喻": 1.7, "拟人": 1.7, "排比": 1.7, "对偶": 1.7,
			"夸张": 1.7, "反复": 1.7, "设问": 1.7, "反问": 1.7, "借代": 1.7,
			"对比": 1.7, "衬托": 1.7, "象征": 1.7, "讽刺": 1.7,
			// 文学体裁
			"文学": 1.8, "诗歌": 1.8, "散文": 1.8, "小说": 1.8, "戏剧": 1.8,
			"记叙文": 1.7, "议论文": 1.7, "说明文": 1.7, "应用文": 1.6,
			"古诗": 1.8, "词": 1.7, "曲": 1.7, "赋": 1.7, "骈文": 1.7,
			// 写作技巧
			"写作": 1.8, "作文": 1.7, "立意": 1.7, "构思": 1.7, "选材": 1.7,
			"布局": 1.7, "开头": 1.6, "结尾": 1.6, "过渡": 1.6, "照应": 1.6,
			"详略": 1.6, "线索": 1.7, "主题": 1.7, "中心思想": 1.7,
			// 阅读理解
			"阅读": 1.7, "理解": 1.6, "分析": 1.6, "概括": 1.6, "归纳": 1.6,
			"总结": 1.6, "品味": 1.6, "鉴赏": 1.7, "文意": 1.6, "语境": 1.7,
			"语感": 1.6, "情感": 1.6, "意境": 1.7,
		},
		// 英语术语词典
		EnglishTerms: map[string]float64{
			// 英语基础
			"英语": 2.0, "English": 2.0, "外语": 1.8, "第二语言": 1.7,
			"语言学习": 1.7, "英文": 1.8, "美式英语": 1.6, "英式英语": 1.6,
			// 语法结构
			"语法": 1.8, "grammar": 1.8, "句法": 1.7, "语音": 1.7, "音标": 1.6,
			"发音": 1.6, "重音": 1.6, "语调": 1.6, "连读": 1.6, "弱读": 1.6,
			"词性": 1.7, "名词": 1.6, "动词": 1.6, "形容词": 1.6, "副词": 1.6,
			"介词": 1.6, "连词": 1.6, "冠词": 1.6, "代词": 1.6, "数词": 1.6,
			// 时态语态
			"时态": 1.8, "语态": 1.7, "现在时": 1.7, "过去时": 1.7, "将来时": 1.7,
			"完成时": 1.7, "进行时": 1.7, "被动语态": 1.7, "主动语态": 1.7,
			"虚拟语气": 1.8, "情态动词": 1.7, "非谓语": 1.7, "分词": 1.7,
			// 词汇学习
			"词汇": 1.8, "单词": 1.7, "vocabulary": 1.8, "短语": 1.7, "习语": 1.7,
			"俚语": 1.6, "同义词": 1.6, "反义词": 1.6, "词根": 1.7, "词缀": 1.7,
			"构词法": 1.7, "词汇量": 1.7, "记忆法": 1.6, "联想记忆": 1.6,
			// 四项技能
			"听力": 1.8, "口语": 1.8, "阅读": 1.8, "写作": 1.8,
			"listening": 1.8, "speaking": 1.8, "reading": 1.8, "writing": 1.8,
			"理解": 1.6, "表达": 1.6, "交流": 1.6, "沟通": 1.6,
			// 考试相关
			"考试": 1.6, "测试": 1.6, "雅思": 1.7, "托福": 1.7,
			"四级": 1.7, "六级": 1.7, "考研英语": 1.7, "高考英语": 1.7,
			"完形填空": 1.6, "阅读理解": 1.7, "翻译": 1.7, "作文": 1.7,
		},
		// 学术术语词典
		AcademicTerms: map[string]float64{
			// 学习方法
			"学习": 1.7, "学习方法": 1.8, "学习技巧": 1.7, "学习策略": 1.7,
			"记忆": 1.6, "理解": 1.6, "应用": 1.6, "分析": 1.6, "综合": 1.6,
			"创新": 1.7, "思维": 1.7, "逻辑": 1.7, "推理": 1.7,
			// 教学术语
			"教学": 1.6, "教育": 1.6, "培养": 1.6, "训练": 1.6, "指导": 1.6,
			"辅导": 1.6, "讲解": 1.6, "演示": 1.6, "示范": 1.6, "练习": 1.6,
			"复习": 1.6, "预习": 1.6, "总结": 1.6, "归纳": 1.6, "整理": 1.6,
			// 考试评价
			"考试": 1.6, "测验": 1.6, "检测": 1.6, "考核": 1.6,
			"成绩": 1.5, "分数": 1.5, "等级": 1.5, "反馈": 1.6,
			// 学科通用
			"知识": 1.6, "知识点": 1.7, "概念": 1.7, "原理": 1.7, "规律": 1.7,
			"方法": 1.6, "技能": 1.6, "能力": 1.6, "素质": 1.6, "素养": 1.6,
			"基础": 1.6, "进阶": 1.6, "高级": 1.6, "专业": 1.6, "学科": 1.6,
		},
	}
}

// ExtractKeywords 提取关键词
func (k *KeywordExtractor) ExtractKeywords(text string) []string {
	// 分词
	words := k.tokenize(text)

	// 计算词频和权重
	wordScore := make(map[string]float64)

	// 基础词频
	for _, word := range words {
		if k.isValidKeyword(word) {
			wordScore[word] += 1.0
		}
	}

	// 技术术语加权
	k.applyTechnicalTermWeights(wordScore)

	// 短语加权
	k.applyPhraseWeights(text, wordScore)

	// 代码相关内容加权
	k.applyCodeContextWeights(text, wordScore)

	// 按权重排序
	type wordWeight struct {
		word   string
		weight float64
	}

	var wordWeights []wordWeight
	for word, weight := range wordScore {
		wordWeights = append(wordWeights, wordWeight{word, weight})
	}

	sort.Slice(wordWeights, func(i, j int) bool {
		return wordWeights[i].weight > wordWeights[j].weight
	})

	// 提取前25个关键词（增加数量）
	maxKeywords := 25
	if len(wordWeights) < maxKeywords {
		maxKeywords = len(wordWeights)
	}

	keywords := make([]string, maxKeywords)
	for i := 0; i < maxKeywords; i++ {
		keywords[i] = wordWeights[i].word
	}

	return keywords
}

// applyTechnicalTermWeights 应用技术术语权重
func (k *KeywordExtractor) applyTechnicalTermWeights(wordScore map[string]float64) {
	if k.technicalTerms == nil {
		return
	}

	termMaps := []map[string]float64{
		k.technicalTerms.ProgrammingLanguages,
		k.technicalTerms.DatabaseTerms,
		k.technicalTerms.WebTechnologies,
		k.technicalTerms.Frameworks,
		k.technicalTerms.ToolsAndSoftware,
		k.technicalTerms.ConceptualTerms,
		k.technicalTerms.MathTerms,
		k.technicalTerms.ChineseTerms,
		k.technicalTerms.EnglishTerms,
		k.technicalTerms.AcademicTerms,
	}

	for _, termMap := range termMaps {
		for term, weight := range termMap {
			if score, exists := wordScore[term]; exists {
				wordScore[term] = score * weight
			}
		}
	}
}

// applyPhraseWeights 应用短语权重
func (k *KeywordExtractor) applyPhraseWeights(text string, wordScore map[string]float64) {
	for phrase, weight := range k.importantPhrases {
		if strings.Contains(text, phrase) {
			// 为短语中的每个词增加权重
			words := strings.Fields(phrase)
			for _, word := range words {
				if score, exists := wordScore[word]; exists {
					wordScore[word] = score * weight
				}
			}
		}
	}
}

// applyCodeContextWeights 应用代码上下文权重
func (k *KeywordExtractor) applyCodeContextWeights(text string, wordScore map[string]float64) {
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		// 检测代码行
		isCodeLine := false
		for _, pattern := range k.codePatterns {
			if pattern.MatchString(line) {
				isCodeLine = true
				break
			}
		}

		if isCodeLine {
			// 提取代码行中的关键词并加权
			words := k.tokenize(line)
			for _, word := range words {
				if score, exists := wordScore[word]; exists {
					wordScore[word] = score * 1.5 // 代码相关词汇权重提升
				}
			}
		}
	}
}

// tokenize 分词
func (k *KeywordExtractor) tokenize(text string) []string {
	var words []string
	var currentWord strings.Builder

	for _, char := range text {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			currentWord.WriteRune(char)
		} else {
			if currentWord.Len() > 0 {
				word := currentWord.String()
				if len(word) >= k.minWordLen {
					words = append(words, word)
				}
				currentWord.Reset()
			}
		}
	}

	// 添加最后一个词
	if currentWord.Len() > 0 {
		word := currentWord.String()
		if len(word) >= k.minWordLen {
			words = append(words, word)
		}
	}

	return words
}

// isValidKeyword 检查是否为有效关键词
func (k *KeywordExtractor) isValidKeyword(word string) bool {
	// 检查停用词
	if k.stopWords[word] {
		return false
	}

	// 检查长度
	if len(word) < k.minWordLen || len(word) > 20 {
		return false
	}

	// 检查是否全是数字
	if regexp.MustCompile(`^\d+$`).MatchString(word) {
		return false
	}

	return true
}

// AnalyzeSections 分析文档章节
func (s *DocumentStructureAnalyzer) AnalyzeSections(text string) ([]Section, error) {
	lines := strings.Split(text, "\n")
	var sections []Section
	var currentSection *Section

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测章节标题
		isChapterTitle := false
		for _, pattern := range s.titlePatterns {
			if pattern.MatchString(line) {
				if currentSection != nil {
					sections = append(sections, *currentSection)
				}
				currentSection = &Section{
					Title:   line,
					Content: "",
					Type:    "chapter",
					Level:   1,
				}
				isChapterTitle = true
				break
			}
		}

		if isChapterTitle {
			continue
		}

		// 检测小节标题
		isSubTitle := false
		for _, pattern := range s.headingPatterns {
			if pattern.MatchString(line) {
				if currentSection != nil {
					sections = append(sections, *currentSection)
				}
				currentSection = &Section{
					Title:   line,
					Content: "",
					Type:    "section",
					Level:   2,
				}
				isSubTitle = true
				break
			}
		}

		if isSubTitle {
			continue
		}

		// 添加到当前节的内容
		if currentSection != nil {
			if currentSection.Content != "" {
				currentSection.Content += "\n"
			}
			currentSection.Content += line
		} else {
			// 如果没有当前节，创建一个默认节
			currentSection = &Section{
				Title:   "主要内容",
				Content: line,
				Type:    "content",
				Level:   1,
			}
		}
	}

	// 添加最后一个节
	if currentSection != nil {
		sections = append(sections, *currentSection)
	}

	// 为每个章节提取关键词
	for i := range sections {
		sections[i].Keywords = NewEnhancedKeywordExtractor().ExtractKeywords(sections[i].Content)
	}

	return sections, nil
}

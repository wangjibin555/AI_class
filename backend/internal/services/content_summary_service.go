package services

import (
	"ai-classroom/pkg/ai"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"gorm.io/gorm"
)

// ContentSummaryService AI内容总结服务
type ContentSummaryService struct {
	aiClient *ai.DashScopeClient
	db       *gorm.DB
}

// SummaryRequest 总结请求
type SummaryRequest struct {
	Content        string `json:"content" binding:"required"`
	SummaryLevel   string `json:"summary_level"`   // brief, detailed, comprehensive
	TargetAudience string `json:"target_audience"` // student, professional, general
	SourceType     string `json:"source_type"`     // text, url, file
}

// SummaryResult 总结结果
type SummaryResult struct {
	MainTopic       string          `json:"main_topic"`
	KeyPoints       []string        `json:"key_points"`
	Summary         string          `json:"summary"`
	Structure       []SectionInfo   `json:"structure"`
	SuggestedSlides int             `json:"suggested_slides"`
	Metadata        ContentMetadata `json:"metadata"`
}

// SectionInfo 章节信息
type SectionInfo struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	KeyPoints  []string `json:"key_points"`
	Importance int      `json:"importance"` // 1-10
	SlideCount int      `json:"slide_count"`
}

// ContentMetadata 内容元数据
type ContentMetadata struct {
	WordCount   int      `json:"word_count"`
	ReadingTime int      `json:"reading_time"` // 分钟
	Difficulty  string   `json:"difficulty"`
	Topics      []string `json:"topics"`
	Language    string   `json:"language"`
	ContentType string   `json:"content_type"`
}

// NewContentSummaryService 创建内容总结服务
func NewContentSummaryService(aiClient *ai.DashScopeClient, db *gorm.DB) *ContentSummaryService {
	return &ContentSummaryService{
		aiClient: aiClient,
		db:       db,
	}
}

// GenerateSummary 生成内容总结
func (s *ContentSummaryService) GenerateSummary(req *SummaryRequest) (*SummaryResult, error) {
	// 设置默认值
	if req.SummaryLevel == "" {
		req.SummaryLevel = "detailed"
	}
	if req.TargetAudience == "" {
		req.TargetAudience = "general"
	}

	// 构建AI提示词
	prompt := s.buildSummaryPrompt(req)

	// 调用AI生成总结
	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI生成总结失败: %w", err)
	}

	// 解析AI响应
	result, err := s.parseSummaryResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	// 生成元数据
	result.Metadata = s.generateMetadata(req.Content, result)

	// 保存总结记录（可选）
	go s.saveSummaryRecord(req, result)

	return result, nil
}

// buildSummaryPrompt 构建总结提示词
func (s *ContentSummaryService) buildSummaryPrompt(req *SummaryRequest) string {
	// 根据总结级别和目标受众选择提示词模板
	var promptTemplate string

	switch req.SummaryLevel {
	case "brief":
		promptTemplate = s.getBriefSummaryPrompt()
	case "comprehensive":
		promptTemplate = s.getComprehensiveSummaryPrompt()
	default: // detailed
		promptTemplate = s.getDetailedSummaryPrompt()
	}

	// 根据目标受众调整提示词
	audienceContext := s.getAudienceContext(req.TargetAudience)

	data := struct {
		Content         string
		SummaryLevel    string
		TargetAudience  string
		AudienceContext string
	}{
		Content:         req.Content,
		SummaryLevel:    req.SummaryLevel,
		TargetAudience:  req.TargetAudience,
		AudienceContext: audienceContext,
	}

	var buf strings.Builder
	t := template.Must(template.New("summary").Parse(promptTemplate))
	t.Execute(&buf, data)

	return buf.String()
}

// getDetailedSummaryPrompt 获取详细总结提示词
func (s *ContentSummaryService) getDetailedSummaryPrompt() string {
	return `你是一个专业的内容分析师和课件制作专家。请对以下内容进行深度分析和总结：

内容：
{{.Content}}

分析要求：
1. 总结等级：{{.SummaryLevel}}
2. 目标受众：{{.TargetAudience}}
3. 受众特点：{{.AudienceContext}}

请按以下JSON格式输出分析结果：
{
  "main_topic": "内容的主要主题（简洁明了）",
  "key_points": ["核心要点1", "核心要点2", "核心要点3", "核心要点4", "核心要点5"],
  "summary": "内容总结（150-200字，概括核心思想）",
  "structure": [
    {
      "title": "章节1标题",
      "content": "章节内容概要",
      "key_points": ["章节要点1", "章节要点2", "章节要点3"],
      "importance": 8,
      "slide_count": 2
    },
    {
      "title": "章节2标题", 
      "content": "章节内容概要",
      "key_points": ["章节要点1", "章节要点2"],
      "importance": 6,
      "slide_count": 1
    }
  ],
  "suggested_slides": 建议幻灯片总数（5-20张）
}

分析要求：
1. 确保分析准确性和逻辑性
2. 章节划分要合理，重点突出
3. 幻灯片数量建议要实用
4. 考虑{{.TargetAudience}}的理解水平
5. 严格按照JSON格式输出，不要包含其他内容`
}

// getBriefSummaryPrompt 获取简要总结提示词
func (s *ContentSummaryService) getBriefSummaryPrompt() string {
	return `你是一个专业的内容分析师。请对以下内容进行简要分析：

内容：
{{.Content}}

目标受众：{{.TargetAudience}}

请按以下JSON格式输出分析结果：
{
  "main_topic": "主要主题",
  "key_points": ["要点1", "要点2", "要点3"],
  "summary": "简要总结（100字以内）",
  "structure": [
    {
      "title": "主要内容",
      "content": "内容概要", 
      "key_points": ["要点1", "要点2"],
      "importance": 8,
      "slide_count": 3
    }
  ],
  "suggested_slides": 建议幻灯片数量（3-8张）
}

要求：简洁明了，突出重点，适合快速理解。`
}

// getComprehensiveSummaryPrompt 获取全面总结提示词
func (s *ContentSummaryService) getComprehensiveSummaryPrompt() string {
	return `你是一个资深的教育专家和内容分析师。请对以下内容进行全面深入的分析：

内容：
{{.Content}}

分析要求：
1. 目标受众：{{.TargetAudience}}
2. 受众特点：{{.AudienceContext}}
3. 需要全面分析，包含所有重要知识点

请按以下JSON格式输出详细分析结果：
{
  "main_topic": "主要主题（详细描述）",
  "key_points": ["核心要点1", "核心要点2", "核心要点3", "核心要点4", "核心要点5", "核心要点6", "核心要点7"],
  "summary": "全面总结（200-300字，深入分析核心思想和价值）",
  "structure": [
    {
      "title": "引言/背景",
      "content": "详细的背景介绍",
      "key_points": ["背景要点1", "背景要点2"],
      "importance": 7,
      "slide_count": 2
    },
    {
      "title": "核心概念",
      "content": "核心概念详解",
      "key_points": ["概念1", "概念2", "概念3"],
      "importance": 10,
      "slide_count": 3
    },
    {
      "title": "应用实例",
      "content": "实际应用案例",
      "key_points": ["应用1", "应用2"],
      "importance": 8,
      "slide_count": 2
    },
    {
      "title": "总结展望",
      "content": "总结和未来展望",
      "key_points": ["总结要点", "发展趋势"],
      "importance": 6,
      "slide_count": 1
    }
  ],
  "suggested_slides": 建议幻灯片总数（8-25张）
}

要求：
1. 全面深入，不遗漏重要信息
2. 逻辑清晰，层次分明
3. 适合{{.TargetAudience}}的知识水平
4. 包含实例和应用场景
5. 严格JSON格式输出`
}

// getAudienceContext 获取受众上下文
func (s *ContentSummaryService) getAudienceContext(audience string) string {
	switch audience {
	case "student":
		return "学生群体，注重基础概念理解，需要循序渐进，配合实例说明"
	case "professional":
		return "专业人士，具备一定基础知识，关注实用性和深度，重视应用场景"
	case "general":
		return "普通用户，知识背景多样，需要通俗易懂，平衡深度和广度"
	default:
		return "一般受众，需要清晰易懂的表达方式"
	}
}

// parseSummaryResponse 解析总结响应
func (s *ContentSummaryService) parseSummaryResponse(response string) (*SummaryResult, error) {
	// 清理响应文本，提取JSON部分
	jsonContent := s.extractJSON(response)
	if jsonContent == "" {
		return nil, fmt.Errorf("无法找到有效的JSON内容")
	}

	var result SummaryResult
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 验证结果完整性
	if err := s.validateSummaryResult(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// extractJSON 提取JSON内容
func (s *ContentSummaryService) extractJSON(response string) string {
	// 查找JSON开始和结束位置
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return ""
	}

	return response[jsonStart : jsonEnd+1]
}

// validateSummaryResult 验证总结结果
func (s *ContentSummaryService) validateSummaryResult(result *SummaryResult) error {
	if result.MainTopic == "" {
		return fmt.Errorf("主题不能为空")
	}

	if len(result.KeyPoints) == 0 {
		return fmt.Errorf("关键要点不能为空")
	}

	if result.Summary == "" {
		return fmt.Errorf("总结不能为空")
	}

	if len(result.Structure) == 0 {
		return fmt.Errorf("结构信息不能为空")
	}

	if result.SuggestedSlides <= 0 {
		return fmt.Errorf("建议幻灯片数量必须大于0")
	}

	// 验证结构信息
	for i, section := range result.Structure {
		if section.Title == "" {
			return fmt.Errorf("第%d个章节标题不能为空", i+1)
		}
		if section.SlideCount <= 0 {
			return fmt.Errorf("第%d个章节幻灯片数量必须大于0", i+1)
		}
	}

	return nil
}

// generateMetadata 生成元数据
func (s *ContentSummaryService) generateMetadata(content string, result *SummaryResult) ContentMetadata {
	wordCount := len([]rune(content))

	// 计算预计阅读时间（假设中文阅读速度每分钟300字）
	readingTime := wordCount / 300
	if readingTime < 1 {
		readingTime = 1
	}

	// 根据词数和结构复杂度判断难度
	difficulty := s.estimateDifficulty(content, result)

	// 提取主题标签
	topics := s.extractTopics(result)

	return ContentMetadata{
		WordCount:   wordCount,
		ReadingTime: readingTime,
		Difficulty:  difficulty,
		Topics:      topics,
		Language:    s.detectLanguage(content),
		ContentType: s.classifyContentType(result),
	}
}

// estimateDifficulty 估算难度
func (s *ContentSummaryService) estimateDifficulty(content string, result *SummaryResult) string {
	wordCount := len([]rune(content))
	structureComplexity := len(result.Structure)

	// 简单的难度判断逻辑
	if wordCount < 500 || structureComplexity <= 2 {
		return "入门"
	} else if wordCount < 2000 || structureComplexity <= 4 {
		return "进阶"
	} else {
		return "高级"
	}
}

// extractTopics 提取主题
func (s *ContentSummaryService) extractTopics(result *SummaryResult) []string {
	topics := []string{result.MainTopic}

	// 从章节标题中提取更多主题
	for _, section := range result.Structure {
		if len(topics) < 5 { // 最多5个主题
			topics = append(topics, section.Title)
		}
	}

	return topics
}

// detectLanguage 检测语言
func (s *ContentSummaryService) detectLanguage(content string) string {
	chineseCount := 0
	totalChars := 0

	for _, r := range content {
		if r >= 0x4e00 && r <= 0x9fff { // 中文Unicode范围
			chineseCount++
		}
		if r > 32 { // 排除空白字符
			totalChars++
		}
	}

	if totalChars > 0 && float64(chineseCount)/float64(totalChars) > 0.3 {
		return "zh-CN"
	}

	return "en"
}

// classifyContentType 分类内容类型
func (s *ContentSummaryService) classifyContentType(result *SummaryResult) string {
	topic := strings.ToLower(result.MainTopic)

	// 简单的关键词匹配
	if strings.Contains(topic, "技术") || strings.Contains(topic, "编程") ||
		strings.Contains(topic, "算法") || strings.Contains(topic, "development") {
		return "技术"
	} else if strings.Contains(topic, "商业") || strings.Contains(topic, "管理") ||
		strings.Contains(topic, "business") || strings.Contains(topic, "marketing") {
		return "商业"
	} else if strings.Contains(topic, "教育") || strings.Contains(topic, "学习") ||
		strings.Contains(topic, "教学") || strings.Contains(topic, "education") {
		return "教育"
	} else if strings.Contains(topic, "科学") || strings.Contains(topic, "研究") ||
		strings.Contains(topic, "science") || strings.Contains(topic, "research") {
		return "科学"
	} else {
		return "综合"
	}
}

// saveSummaryRecord 保存总结记录
func (s *ContentSummaryService) saveSummaryRecord(req *SummaryRequest, result *SummaryResult) {
	// 这里可以将总结记录保存到数据库，用于历史查询和分析
	// 由于是异步执行，错误只记录日志，不影响主流程

	// 示例实现，实际需要定义对应的数据模型
	/*
		record := &models.AISummaryRecord{
			Content:        req.Content,
			SummaryLevel:   req.SummaryLevel,
			TargetAudience: req.TargetAudience,
			SummaryResult:  result, // 需要序列化为JSON
			ProcessTime:    time.Since(startTime).Milliseconds(),
			CreatedAt:      time.Now(),
		}

		if err := s.db.Create(record).Error; err != nil {
			log.Printf("保存总结记录失败: %v", err)
		}
	*/
}

// GetCachedSummary 获取缓存的总结（如果存在）
func (s *ContentSummaryService) GetCachedSummary(contentHash string) (*SummaryResult, error) {
	// 这里可以实现基于内容哈希的缓存查询
	// 减少重复内容的AI调用，提高性能

	// 示例实现
	/*
		var record models.AISummaryRecord
		err := s.db.Where("content_hash = ?", contentHash).First(&record).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil // 缓存未命中
			}
			return nil, err
		}

		// 反序列化结果
		var result SummaryResult
		if err := json.Unmarshal(record.SummaryResult, &result); err != nil {
			return nil, err
		}

		return &result, nil
	*/

	return nil, nil
}

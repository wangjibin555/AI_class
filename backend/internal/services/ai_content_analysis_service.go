package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-classroom/internal/models"
	"ai-classroom/pkg/ai"

	"gorm.io/gorm"
)

// AIEngineType AI引擎类型
type AIEngineType string

const (
	EngineDashScope AIEngineType = "dashscope"
	EngineCoze      AIEngineType = "coze"   // 预留Coze引擎
	EngineHybrid    AIEngineType = "hybrid" // 预留混合引擎
)

// AIContentAnalysisService AI内容分析服务
type AIContentAnalysisService struct {
	dashScopeClient *ai.DashScopeClient
	// cozeClient      *ai.CozeClient  // 预留Coze客户端
	db             *gorm.DB
	contentService *ContentService // 复用现有内容服务
}

func NewAIContentAnalysisService(
	dashScopeClient *ai.DashScopeClient,
	db *gorm.DB,
	contentService *ContentService,
) *AIContentAnalysisService {
	return &AIContentAnalysisService{
		dashScopeClient: dashScopeClient,
		db:              db,
		contentService:  contentService,
	}
}

// AnalysisRequest 分析请求
type AnalysisRequest struct {
	URL          string       `json:"url"`
	AnalysisType string       `json:"analysis_type"` // comprehensive, summary, technical
	EngineType   AIEngineType `json:"engine_type"`   // dashscope, coze, hybrid
	Language     string       `json:"language"`      // zh-CN, en-US
}

// AnalysisOptions 分析选项
type AnalysisOptions struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

// AnalyzeURL 分析URL内容 - 主入口方法
func (s *AIContentAnalysisService) AnalyzeURL(ctx context.Context, req *AnalysisRequest) (*models.AIAnalysisResponse, error) {
	startTime := time.Now()

	log.Printf("🔍 [AI分析] 开始分析URL: %s", req.URL)
	log.Printf("🔍 [AI分析] 分析类型: %s, 引擎类型: %s, 语言: %s", req.AnalysisType, req.EngineType, req.Language)

	// 设置默认值
	if req.AnalysisType == "" {
		req.AnalysisType = "comprehensive"
	}
	if req.EngineType == "" {
		req.EngineType = EngineDashScope
	}
	if req.Language == "" {
		req.Language = "zh-CN"
	}

	log.Printf("🔍 [AI分析] 使用默认值 - 分析类型: %s, 引擎类型: %s, 语言: %s", req.AnalysisType, req.EngineType, req.Language)

	// 根据引擎类型调用不同的分析方法
	var result *models.AIAnalysisResponse
	var err error

	switch req.EngineType {
	case EngineDashScope:
		log.Printf("🔍 [AI分析] 使用DashScope引擎进行分析")
		result, err = s.analyzeWithDashScope(ctx, req)
	case EngineCoze:
		log.Printf("🔍 [AI分析] 使用Coze引擎进行分析（预留）")
		// 预留Coze引擎接口
		result, err = s.analyzeWithCoze(ctx, req)
	case EngineHybrid:
		log.Printf("🔍 [AI分析] 使用混合引擎进行分析（预留）")
		// 预留混合引擎接口
		result, err = s.analyzeWithHybrid(ctx, req)
	default:
		log.Printf("❌ [AI分析] 不支持的引擎类型: %s", req.EngineType)
		return nil, fmt.Errorf("不支持的引擎类型: %s", req.EngineType)
	}

	if err != nil {
		log.Printf("❌ [AI分析] 分析失败: %v", err)
		return nil, err
	}

	log.Printf("✅ [AI分析] 分析成功，结果标题: %s", result.Title)
	log.Printf("🔍 [AI分析] 分析耗时: %v", time.Since(startTime))

	// 设置元数据
	if result.Metadata == nil {
		result.Metadata = &models.AnalysisMetadata{}
	}
	result.Metadata.AnalysisTime = startTime
	result.Metadata.EngineUsed = string(req.EngineType)
	result.URL = req.URL

	// 处理时间（毫秒）
	processTime := int(time.Since(startTime).Milliseconds())
	result.Metadata.EstimatedTime = s.calculateEstimatedTime(result)

	// 记录分析历史（异步）
	go s.recordAnalysisHistory(req, result, processTime, "completed", "")

	return result, nil
}

// analyzeWithDashScope 使用DashScope分析
func (s *AIContentAnalysisService) analyzeWithDashScope(ctx context.Context, req *AnalysisRequest) (*models.AIAnalysisResponse, error) {
	log.Printf("🔍 开始使用DashScope分析URL: %s", req.URL)

	// 首先尝试爬取URL内容
	var content string

	// 创建内容请求
	contentReq := &ContentRequest{
		Type: "url",
		URL:  req.URL,
	}

	// 使用内容服务处理URL
	contentResult, err := s.contentService.ProcessContent(contentReq)
	if err != nil {
		log.Printf("⚠️ URL内容爬取失败: %v，使用基础分析", err)
		// 如果爬取失败，仍然可以基于URL进行基础分析
		content = fmt.Sprintf("URL: %s\n注意：由于网络或权限问题，无法直接获取页面内容，但可以基于URL特征进行分析。", req.URL)
	} else {
		log.Printf("✅ 成功爬取内容，长度: %d 字符", len(contentResult.Content))
		content = fmt.Sprintf("标题: %s\n内容: %s", contentResult.Title, contentResult.Content)

		// 限制内容长度避免token超限
		if len(content) > 8000 {
			content = content[:8000] + "...\n[内容过长已截断]"
			log.Printf("📝 内容已截断到8000字符")
		}
	}

	// 构建分析提示词
	prompt := s.buildDashScopePromptWithContent(req.URL, content, req.AnalysisType, req.Language)

	// 调用DashScope API
	log.Printf("🤖 调用DashScope API进行内容分析")
	response, err := s.dashScopeClient.GenerateContent(prompt)
	if err != nil {
		log.Printf("❌ DashScope分析失败: %v，使用回退方案", err)
		return s.generateFallbackResponse(req.URL), nil
	}

	// 解析AI响应
	result, err := s.parseDashScopeResponse(response, req.URL)
	if err != nil {
		log.Printf("❌ 解析DashScope响应失败: %v，使用回退方案", err)
		return s.generateFallbackResponse(req.URL), nil
	}

	log.Printf("✅ DashScope分析完成")
	return result, nil
}

// analyzeWithCoze 使用Coze分析（预留接口）
func (s *AIContentAnalysisService) analyzeWithCoze(ctx context.Context, req *AnalysisRequest) (*models.AIAnalysisResponse, error) {
	// TODO: 实现Coze引擎分析
	return nil, fmt.Errorf("Coze引擎暂未实现，敬请期待")
}

// analyzeWithHybrid 使用混合引擎分析（预留接口）
func (s *AIContentAnalysisService) analyzeWithHybrid(ctx context.Context, req *AnalysisRequest) (*models.AIAnalysisResponse, error) {
	// TODO: 实现混合引擎分析
	return nil, fmt.Errorf("混合引擎暂未实现，敬请期待")
}

// buildDashScopePrompt 构建DashScope提示词
func (s *AIContentAnalysisService) buildDashScopePrompt(url, analysisType, language string) string {
	basePrompt := fmt.Sprintf(`
请分析以下URL的内容并生成详细的技术总结：%s

请注意：
1. 我需要你基于对该URL内容的理解来生成总结，而不是访问该URL
2. 如果你熟悉该URL的内容，请提供详细的技术分析
3. 如果不熟悉，请基于URL结构和你的知识来推断可能的内容

分析要求：
- 分析类型：%s
- 输出语言：%s
- 输出格式：结构化JSON

请按以下JSON格式输出：
{
    "title": "内容标题",
    "summary": "内容摘要（200-300字）",
    "key_points": ["关键点1", "关键点2", "关键点3"],
    "technical_concepts": [
        {
            "name": "概念名称",
            "description": "概念描述",
            "category": "概念分类",
            "importance": 8
        }
    ],
    "structured_content": {
        "introduction": "内容介绍",
        "main_sections": [
            {
                "title": "章节标题",
                "content": "章节内容",
                "sub_sections": ["子章节1", "子章节2"],
                "key_points": ["要点1", "要点2"]
            }
        ],
        "conclusion": "总结",
        "examples": [
            {
                "title": "示例标题",
                "code": "代码示例",
                "language": "编程语言",
                "description": "示例说明"
            }
        ],
        "best_practices": ["最佳实践1", "最佳实践2"]
    },
    "metadata": {
        "content_type": "技术文档",
        "difficulty_level": "中级",
        "estimated_time": 30,
        "tags": ["标签1", "标签2"]
    }
}`, url, analysisType, language)

	// 根据URL类型添加特定提示
	if strings.Contains(url, "developers.weixin.qq.com") {
		basePrompt += `

特别说明：这是微信小程序开发者文档，请重点关注：
- 小程序开发相关的API和功能
- 具体的代码示例和使用方法
- 开发过程中的注意事项
- 最佳实践和常见问题`
	} else if strings.Contains(url, "github.com") {
		basePrompt += `

特别说明：这是GitHub项目页面，请重点关注：
- 项目功能特性和用途
- 安装和使用方法
- API文档和示例代码
- 项目架构和技术栈`
	} else if strings.Contains(url, "scene") {
		basePrompt += `

特别说明：这可能与场景值相关，请重点关注：
- 场景值的定义和作用
- 具体的场景值列表和含义
- 如何获取和使用场景值
- 相关的API调用方法`
	}

	return basePrompt
}

// buildDashScopePromptWithContent 构建包含内容的DashScope提示词
func (s *AIContentAnalysisService) buildDashScopePromptWithContent(url, content, analysisType, language string) string {
	basePrompt := fmt.Sprintf(`
请分析以下URL的内容并生成详细的技术总结：

URL: %s

内容：
%s

分析类型：%s
输出语言：%s

请严格按照以下JSON格式输出，不要添加任何其他内容：

{
    "url": "%s",
    "title": "内容标题",
    "summary": "详细内容摘要（至少100字）",
    "key_points": ["关键点1", "关键点2", "关键点3", "关键点4", "关键点5"],
    "technical_concepts": [
        {
            "name": "概念名称",
            "description": "概念描述",
            "category": "技术分类",
            "importance": 8
        }
    ],
    "structured_content": {
        "introduction": "内容介绍",
        "main_sections": [
            {
                "title": "章节标题",
                "content": "章节内容",
                "sub_sections": ["子章节1", "子章节2"],
                "key_points": ["要点1", "要点2"]
            }
        ],
        "conclusion": "总结",
        "examples": [
            {
                "title": "示例标题",
                "code": "代码示例",
                "language": "编程语言",
                "description": "示例说明"
            }
        ],
        "best_practices": ["最佳实践1", "最佳实践2"]
    },
    "metadata": {
        "content_type": "技术文档",
        "difficulty_level": "中级",
        "estimated_time": 30,
        "tags": ["标签1", "标签2"]
    }
}`, url, content, analysisType, language, url)

	// 根据URL类型添加特定提示
	if strings.Contains(url, "developers.weixin.qq.com") {
		basePrompt += `

特别说明：这是微信小程序开发者文档，请重点关注：
- 小程序开发相关的API和功能
- 具体的代码示例和使用方法
- 开发过程中的注意事项
- 最佳实践和常见问题`
	} else if strings.Contains(url, "github.com") {
		basePrompt += `

特别说明：这是GitHub项目页面，请重点关注：
- 项目功能特性和用途
- 安装和使用方法
- API文档和示例代码
- 项目架构和技术栈`
	} else if strings.Contains(url, "scene") {
		basePrompt += `

特别说明：这可能与场景值相关，请重点关注：
- 场景值的定义和作用
- 具体的场景值列表和含义
- 如何获取和使用场景值
- 相关的API调用方法`
	}

	basePrompt += `

注意：请确保输出的JSON格式完全正确，所有字符串都要用双引号包围，不要有语法错误。`

	return basePrompt
}

// parseDashScopeResponse 解析DashScope响应
func (s *AIContentAnalysisService) parseDashScopeResponse(response, url string) (*models.AIAnalysisResponse, error) {
	// 记录原始响应用于调试
	log.Printf("🔍 DashScope原始响应: %s", response)

	// 检查响应是否为空
	if strings.TrimSpace(response) == "" {
		log.Printf("❌ DashScope返回空响应")
		return s.generateFallbackResponse(url), nil
	}

	// 提取JSON部分
	jsonStr := s.extractJSON(response)
	log.Printf("🔧 提取的JSON: %s", jsonStr)

	// 检查提取的JSON是否有效
	if jsonStr == "{}" || strings.TrimSpace(jsonStr) == "" {
		log.Printf("❌ 无法从响应中提取有效JSON，使用回退方案")
		return s.generateFallbackResponse(url), nil
	}

	var result models.AIAnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Printf("❌ JSON解析失败: %v，响应内容: %s", err, jsonStr)
		// 使用回退方案而不是返回错误
		return s.generateFallbackResponse(url), nil
	}

	// 验证结果完整性
	if err := s.validateAnalysisResult(&result); err != nil {
		log.Printf("⚠️ 分析结果验证失败: %v，使用回退方案", err)
		return s.generateFallbackResponse(url), nil
	}

	log.Printf("✅ DashScope响应解析成功")
	return &result, nil
}

// generateFallbackResponse 生成回退响应
func (s *AIContentAnalysisService) generateFallbackResponse(url string) *models.AIAnalysisResponse {
	// 基于URL生成基础的分析结果
	title := "内容分析"
	summary := "由于AI分析服务暂时不可用，已为您生成基础的内容分析结果。请稍后重试以获得更详细的分析。"

	// 根据URL特征调整内容
	if strings.Contains(url, "developers.weixin.qq.com") {
		title = "微信小程序开发文档分析"
		summary = "这是微信小程序开发相关的技术文档。文档包含了小程序开发的核心概念、API使用方法和最佳实践。建议详细阅读以掌握小程序开发技能。"
	} else if strings.Contains(url, "github.com") {
		title = "GitHub项目分析"
		summary = "这是一个GitHub开源项目。项目包含了完整的代码实现、文档说明和使用示例。建议查看README文件和代码结构以了解项目功能。"
	} else if strings.Contains(url, "scene") {
		title = "场景值相关文档分析"
		summary = "这是关于场景值的技术文档。场景值用于标识用户进入小程序的路径，对于数据统计和用户行为分析非常重要。"
	}

	return &models.AIAnalysisResponse{
		URL:     url,
		Title:   title,
		Summary: summary,
		KeyPoints: []string{
			"内容分析功能暂时不可用",
			"建议稍后重试获得详细分析",
			"可以继续使用传统模式生成PPT",
		},
		TechnicalConcepts: []models.TechnicalConcept{
			{
				Name:        "基础概念",
				Description: "该URL内容的基本技术概念",
				Category:    "技术文档",
				Importance:  5,
			},
		},
		StructuredContent: &models.StructuredContent{
			Introduction: "内容简介",
			MainSections: []models.ContentSection{
				{
					Title:   "主要内容",
					Content: "该URL的主要技术内容和要点",
					KeyPoints: []string{
						"核心概念说明",
						"技术实现方式",
						"使用方法介绍",
					},
				},
			},
			Conclusion: "内容总结和要点回顾",
			Examples: []models.CodeExample{
				{
					Title:       "示例代码",
					Code:        "// 示例代码\nconsole.log('Hello World');",
					Language:    "javascript",
					Description: "基础示例代码",
				},
			},
			BestPractices: []string{
				"遵循最佳实践",
				"注意性能优化",
				"保持代码整洁",
			},
		},
		Metadata: &models.AnalysisMetadata{
			AnalysisTime:    time.Now(),
			ContentType:     "技术文档",
			DifficultyLevel: "中级",
			EstimatedTime:   15,
			Tags:            []string{"技术文档", "学习资料"},
			EngineUsed:      "fallback",
		},
	}
}

// extractJSON 提取JSON字符串
func (s *AIContentAnalysisService) extractJSON(text string) string {
	// 查找JSON开始和结束
	start := strings.Index(text, "{")
	if start == -1 {
		return "{}"
	}

	end := strings.LastIndex(text, "}")
	if end == -1 || end <= start {
		return "{}"
	}

	return text[start : end+1]
}

// validateAnalysisResult 验证分析结果
func (s *AIContentAnalysisService) validateAnalysisResult(result *models.AIAnalysisResponse) error {
	if result.Title == "" {
		return fmt.Errorf("标题不能为空")
	}

	if len(result.Summary) < 50 {
		return fmt.Errorf("摘要内容过短")
	}

	if len(result.KeyPoints) == 0 {
		return fmt.Errorf("缺少关键要点")
	}

	return nil
}

// calculateEstimatedTime 计算预估学习时间
func (s *AIContentAnalysisService) calculateEstimatedTime(result *models.AIAnalysisResponse) int {
	baseTime := 15 // 基础15分钟

	// 根据内容复杂度调整时间
	if result.StructuredContent != nil {
		baseTime += len(result.StructuredContent.MainSections) * 5
		baseTime += len(result.StructuredContent.Examples) * 3
	}

	baseTime += len(result.TechnicalConcepts) * 2

	return baseTime
}

// recordAnalysisHistory 记录分析历史
func (s *AIContentAnalysisService) recordAnalysisHistory(
	req *AnalysisRequest,
	result *models.AIAnalysisResponse,
	processTime int,
	status string,
	errorMessage string,
) {
	// 序列化结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("序列化分析结果失败: %v", err)
		return
	}

	record := &models.AIAnalysisRecord{
		UserID:       0, // 将在handler中设置
		URL:          req.URL,
		AnalysisType: req.AnalysisType,
		EngineType:   string(req.EngineType),
		Result:       string(resultJSON),
		Status:       status,
		ErrorMessage: errorMessage,
		ProcessTime:  processTime,
		CreatedAt:    time.Now(),
	}

	if err := s.db.Create(record).Error; err != nil {
		log.Printf("保存分析记录失败: %v", err)
	}
}

// GenerateCourseFromAnalysis 从AI分析结果生成课程
func (s *AIContentAnalysisService) GenerateCourseFromAnalysis(
	userID uint,
	analysis *models.AIAnalysisResponse,
	params *PPTGenerationParams,
) (*models.Course, error) {
	log.Printf("🔍 [PPT生成] 开始从AI分析结果生成课程")
	log.Printf("🔍 [PPT生成] 用户ID: %d, 分析标题: %s", userID, analysis.Title)
	log.Printf("🔍 [PPT生成] 分析URL: %s", analysis.URL)

	// 创建课程
	course := &models.Course{
		UserID:      userID,
		Title:       analysis.Title,
		Description: analysis.Summary,
		SourceType:  "ai_analysis",
		SourceURL:   analysis.URL,
		Status:      "generating",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	log.Printf("🔍 [PPT生成] 创建课程记录: 标题=%s, 描述长度=%d", course.Title, len(course.Description))

	if err := s.db.Create(course).Error; err != nil {
		log.Printf("❌ [PPT生成] 创建课程失败: %v", err)
		return nil, fmt.Errorf("创建课程失败: %v", err)
	}

	log.Printf("✅ [PPT生成] 课程创建成功，课程ID: %d", course.ID)

	// 生成幻灯片
	log.Printf("🔍 [PPT生成] 开始生成幻灯片，参数: 数量=%d, 样式=%s", params.SlideCount, params.Style)
	slides, err := s.generateSlidesFromAnalysis(analysis, params)
	if err != nil {
		log.Printf("❌ [PPT生成] 生成幻灯片失败: %v", err)
		return nil, fmt.Errorf("生成幻灯片失败: %v", err)
	}

	log.Printf("✅ [PPT生成] 幻灯片生成成功，共生成 %d 张幻灯片", len(slides))

	// 保存幻灯片
	log.Printf("🔍 [PPT生成] 开始保存幻灯片到数据库")
	for i, slideContent := range slides {
		slide := &models.Slide{
			CourseID:    course.ID,
			SlideNumber: i + 1,
			Title:       slideContent.Title,
			Content:     slideContent.Content,
			CreatedAt:   time.Now(),
		}

		log.Printf("🔍 [PPT生成] 保存第 %d 张幻灯片: 标题=%s, 内容长度=%d", i+1, slide.Title, len(slide.Content))

		if err := s.db.Create(slide).Error; err != nil {
			log.Printf("❌ [PPT生成] 保存幻灯片失败: %v", err)
			return nil, fmt.Errorf("保存幻灯片失败: %v", err)
		}
	}

	log.Printf("✅ [PPT生成] 所有幻灯片保存成功")

	// 更新课程状态和幻灯片数量
	course.SlidesCount = len(slides)
	course.Status = "completed"
	log.Printf("🔍 [PPT生成] 更新课程状态: 幻灯片数量=%d, 状态=completed", course.SlidesCount)

	if err := s.db.Save(course).Error; err != nil {
		log.Printf("❌ [PPT生成] 更新课程失败: %v", err)
		return nil, fmt.Errorf("更新课程失败: %v", err)
	}

	log.Printf("✅ [PPT生成] 课程生成完成，课程ID: %d, 幻灯片数量: %d", course.ID, course.SlidesCount)
	return course, nil
}

// AISlideContent AI幻灯片内容（重命名避免冲突）
type AISlideContent struct {
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Type     string                 `json:"type"` // title, content, code, summary
	Metadata map[string]interface{} `json:"metadata"`
}

// PPTGenerationParams PPT生成参数
type PPTGenerationParams struct {
	SlideCount           int    `json:"slide_count"`
	Style                string `json:"style"` // professional, casual, academic
	IncludeCodeExamples  bool   `json:"include_code_examples"`
	IncludeBestPractices bool   `json:"include_best_practices"`
	Template             string `json:"template"`
	Language             string `json:"language"`
}

// generateSlidesFromAnalysis 从AI分析结果生成幻灯片
func (s *AIContentAnalysisService) generateSlidesFromAnalysis(
	analysis *models.AIAnalysisResponse,
	params *PPTGenerationParams,
) ([]AISlideContent, error) {
	log.Printf("🔍 [幻灯片生成] 开始从AI分析结果生成幻灯片")
	log.Printf("🔍 [幻灯片生成] 分析标题: %s", analysis.Title)
	log.Printf("🔍 [幻灯片生成] 生成参数: 数量=%d, 样式=%s, 包含代码示例=%v", params.SlideCount, params.Style, params.IncludeCodeExamples)

	var slides []AISlideContent

	// 1. 标题页
	log.Printf("🔍 [幻灯片生成] 生成标题页")
	titleSlide := AISlideContent{
		Title:   analysis.Title,
		Content: fmt.Sprintf("基于：%s\n\n%s", analysis.URL, analysis.Summary),
		Type:    "title",
		Metadata: map[string]interface{}{
			"source_url":   analysis.URL,
			"generated_at": time.Now(),
		},
	}
	slides = append(slides, titleSlide)
	log.Printf("✅ [幻灯片生成] 标题页生成完成: 标题=%s, 内容长度=%d", titleSlide.Title, len(titleSlide.Content))

	// 2. 目录页
	if analysis.StructuredContent != nil && len(analysis.StructuredContent.MainSections) > 0 {
		log.Printf("🔍 [幻灯片生成] 生成目录页，章节数量: %d", len(analysis.StructuredContent.MainSections))
		var tocItems []string
		for i, section := range analysis.StructuredContent.MainSections {
			tocItems = append(tocItems, fmt.Sprintf("%d. %s", i+1, section.Title))
		}

		tocSlide := AISlideContent{
			Title:   "课程目录",
			Content: strings.Join(tocItems, "\n"),
			Type:    "content",
			Metadata: map[string]interface{}{
				"slide_type": "table_of_contents",
			},
		}
		slides = append(slides, tocSlide)
		log.Printf("✅ [幻灯片生成] 目录页生成完成，包含 %d 个章节", len(tocItems))
	} else {
		log.Printf("⚠️ [幻灯片生成] 跳过目录页生成：StructuredContent为空或章节数量为0")
	}

	// 3. 关键概念页
	if len(analysis.TechnicalConcepts) > 0 {
		log.Printf("🔍 [幻灯片生成] 生成关键概念页，概念数量: %d", len(analysis.TechnicalConcepts))
		conceptsSlide := AISlideContent{
			Title:   "核心概念",
			Content: s.formatTechnicalConcepts(analysis.TechnicalConcepts),
			Type:    "content",
			Metadata: map[string]interface{}{
				"slide_type": "concepts",
			},
		}
		slides = append(slides, conceptsSlide)
		log.Printf("✅ [幻灯片生成] 关键概念页生成完成: 标题=%s, 内容长度=%d", conceptsSlide.Title, len(conceptsSlide.Content))
	} else {
		log.Printf("⚠️ [幻灯片生成] 跳过关键概念页生成：TechnicalConcepts为空")
	}

	// 4. 主要内容章节
	if analysis.StructuredContent != nil {
		log.Printf("🔍 [幻灯片生成] 生成主要内容章节，章节数量: %d", len(analysis.StructuredContent.MainSections))
		for i, section := range analysis.StructuredContent.MainSections {
			log.Printf("🔍 [幻灯片生成] 生成第 %d 个章节: %s", i+1, section.Title)
			sectionSlide := AISlideContent{
				Title:   section.Title,
				Content: s.formatSectionContent(section),
				Type:    "content",
				Metadata: map[string]interface{}{
					"slide_type": "section",
					"key_points": section.KeyPoints,
				},
			}
			slides = append(slides, sectionSlide)
			log.Printf("✅ [幻灯片生成] 章节生成完成: 标题=%s, 内容长度=%d, 关键点数量=%d",
				sectionSlide.Title, len(sectionSlide.Content), len(section.KeyPoints))
		}
	} else {
		log.Printf("⚠️ [幻灯片生成] 跳过主要内容章节生成：StructuredContent为空")
	}

	// 5. 代码示例页
	if params.IncludeCodeExamples && analysis.StructuredContent != nil && len(analysis.StructuredContent.Examples) > 0 {
		log.Printf("🔍 [幻灯片生成] 生成代码示例页，示例数量: %d", len(analysis.StructuredContent.Examples))
		for i, example := range analysis.StructuredContent.Examples {
			log.Printf("🔍 [幻灯片生成] 生成第 %d 个代码示例: %s", i+1, example.Title)
			exampleSlide := AISlideContent{
				Title: fmt.Sprintf("代码示例：%s", example.Title),
				Content: fmt.Sprintf("```%s\n%s\n```\n\n%s",
					example.Language, example.Code, example.Description),
				Type: "code",
				Metadata: map[string]interface{}{
					"slide_type": "code_example",
					"language":   example.Language,
				},
			}
			slides = append(slides, exampleSlide)
			log.Printf("✅ [幻灯片生成] 代码示例生成完成: 标题=%s, 语言=%s, 代码长度=%d",
				exampleSlide.Title, example.Language, len(example.Code))
		}
	} else {
		log.Printf("⚠️ [幻灯片生成] 跳过代码示例页生成：IncludeCodeExamples=%v, Examples数量=%d",
			params.IncludeCodeExamples, len(analysis.StructuredContent.Examples))
	}

	// 6. 最佳实践页
	if params.IncludeBestPractices && analysis.StructuredContent != nil && len(analysis.StructuredContent.BestPractices) > 0 {
		log.Printf("🔍 [幻灯片生成] 生成最佳实践页，实践数量: %d", len(analysis.StructuredContent.BestPractices))
		practicesSlide := AISlideContent{
			Title:   "最佳实践",
			Content: s.formatBestPractices(analysis.StructuredContent.BestPractices),
			Type:    "content",
			Metadata: map[string]interface{}{
				"slide_type": "best_practices",
			},
		}
		slides = append(slides, practicesSlide)
		log.Printf("✅ [幻灯片生成] 最佳实践页生成完成: 标题=%s, 内容长度=%d",
			practicesSlide.Title, len(practicesSlide.Content))
	} else {
		log.Printf("⚠️ [幻灯片生成] 跳过最佳实践页生成：IncludeBestPractices=%v, BestPractices数量=%d",
			params.IncludeBestPractices, len(analysis.StructuredContent.BestPractices))
	}

	// 7. 总结页
	if analysis.StructuredContent != nil && analysis.StructuredContent.Conclusion != "" {
		log.Printf("🔍 [幻灯片生成] 生成总结页")
		conclusionSlide := AISlideContent{
			Title:   "课程总结",
			Content: analysis.StructuredContent.Conclusion + "\n\n" + s.formatKeyPoints(analysis.KeyPoints),
			Type:    "summary",
			Metadata: map[string]interface{}{
				"slide_type": "conclusion",
			},
		}
		slides = append(slides, conclusionSlide)
		log.Printf("✅ [幻灯片生成] 总结页生成完成: 标题=%s, 内容长度=%d, 关键点数量=%d",
			conclusionSlide.Title, len(conclusionSlide.Content), len(analysis.KeyPoints))
	} else {
		log.Printf("⚠️ [幻灯片生成] 跳过总结页生成：Conclusion为空")
	}

	log.Printf("✅ [幻灯片生成] 所有幻灯片生成完成，总计 %d 张幻灯片", len(slides))
	return slides, nil
}

// 格式化技术概念
func (s *AIContentAnalysisService) formatTechnicalConcepts(concepts []models.TechnicalConcept) string {
	var result []string
	for _, concept := range concepts {
		result = append(result, fmt.Sprintf("**%s**\n%s\n分类：%s | 重要性：%d/10",
			concept.Name, concept.Description, concept.Category, concept.Importance))
	}
	return strings.Join(result, "\n\n")
}

// 格式化章节内容
func (s *AIContentAnalysisService) formatSectionContent(section models.ContentSection) string {
	content := section.Content

	if len(section.KeyPoints) > 0 {
		content += "\n\n**关键要点：**\n"
		for _, point := range section.KeyPoints {
			content += fmt.Sprintf("• %s\n", point)
		}
	}

	if len(section.SubSections) > 0 {
		content += "\n**子章节：**\n"
		for _, subSection := range section.SubSections {
			content += fmt.Sprintf("- %s\n", subSection)
		}
	}

	return content
}

// 格式化最佳实践
func (s *AIContentAnalysisService) formatBestPractices(practices []string) string {
	var result []string
	for i, practice := range practices {
		result = append(result, fmt.Sprintf("%d. %s", i+1, practice))
	}
	return strings.Join(result, "\n\n")
}

// 格式化关键点
func (s *AIContentAnalysisService) formatKeyPoints(points []string) string {
	var result []string
	for _, point := range points {
		result = append(result, fmt.Sprintf("• %s", point))
	}
	return "**关键要点：**\n" + strings.Join(result, "\n")
}

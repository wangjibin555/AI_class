package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DashScopeClient 通义千问客户端
type DashScopeClient struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	timeout     time.Duration
	client      *http.Client
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model      string     `json:"model"`
	Input      ChatInput  `json:"input"`
	Parameters Parameters `json:"parameters"`
}

// ChatInput 聊天输入
type ChatInput struct {
	Messages []Message `json:"messages"`
}

// Parameters 参数
type Parameters struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Output struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"output"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewDashScopeClient 创建通义千问客户端
func NewDashScopeClient(apiKey, baseURL, model string, maxTokens int, temperature float64, timeout time.Duration) *DashScopeClient {
	return &DashScopeClient{
		apiKey:      apiKey,
		baseURL:     baseURL,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		timeout:     timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// ChatCompletion 聊天完成
func (c *DashScopeClient) ChatCompletion(messages []Message) (*ChatResponse, error) {
	request := ChatRequest{
		Model: c.model,
		Input: ChatInput{
			Messages: messages,
		},
		Parameters: Parameters{
			MaxTokens:   c.maxTokens,
			Temperature: c.temperature,
			TopP:        0.8,
		},
	}

	return c.sendRequest(request)
}

// SimpleChat 简单聊天
func (c *DashScopeClient) SimpleChat(prompt string) (string, error) {
	messages := []Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := c.ChatCompletion(messages)
	if err != nil {
		return "", err
	}

	return response.Output.Text, nil
}

// ChatWithHistory 带历史记录的聊天
func (c *DashScopeClient) ChatWithHistory(messages []Message, newMessage string) (*ChatResponse, error) {
	// 添加新消息
	allMessages := append(messages, Message{
		Role:    "user",
		Content: newMessage,
	})

	return c.ChatCompletion(allMessages)
}

// sendRequest 发送请求
func (c *DashScopeClient) sendRequest(request ChatRequest) (*ChatResponse, error) {
	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", c.baseURL+"/services/aigc/text-generation/generation", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-DashScope-SSE", "disable")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("HTTP错误 %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API错误 %s: %s", errorResp.Code, errorResp.Message)
	}

	// 解析响应
	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &response, nil
}

// GenerateContent 生成内容（用于PPT生成）
func (c *DashScopeClient) GenerateContent(prompt string) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个专业的教育内容生成助手，擅长将各种内容转换为结构化的PPT课件。请严格按照要求的JSON格式输出，不要添加任何其他文本。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := c.ChatCompletion(messages)
	if err != nil {
		return "", err
	}

	return response.Output.Text, nil
}

// AnalyzeContent 分析内容
func (c *DashScopeClient) AnalyzeContent(content string) (string, error) {
	prompt := fmt.Sprintf(`请分析以下内容，提取关键信息：

内容：
%s

请分析并输出JSON格式：
{
    "topic": "内容主题",
    "field": "所属领域（技术/商业/教育/科学等）",
    "difficulty": "难度等级（入门/进阶/高级）",
    "key_points": ["关键点1", "关键点2", "关键点3"],
    "suggested_slides": 建议幻灯片数量,
    "target_audience": "目标受众",
    "learning_objectives": ["学习目标1", "学习目标2"]
}`, content)

	return c.GenerateContent(prompt)
}

// GeneratePPT 生成PPT内容
func (c *DashScopeClient) GeneratePPT(content string, slideCount int) (string, error) {
	// 检查是否为模拟模式
	if c.apiKey == "mock_key" {
		return c.generateMockPPT(content, slideCount)
	}

	prompt := fmt.Sprintf(`请基于以下内容生成一个完整的PPT课件，要求：

1. 分析内容主题，提取核心知识点
2. 合理组织结构，确保逻辑清晰
3. 生成%d张幻灯片
4. 每张幻灯片包含：标题、要点内容、演讲备注
5. 严格按照JSON格式输出

内容：
%s

输出格式（必须严格遵循）：
{
    "title": "课件总标题",
    "description": "课件简要描述",
    "difficulty": "入门/进阶/高级",
    "estimated_duration": 预计学习时长（分钟）,
    "slides": [
        {
            "number": 1,
            "title": "幻灯片标题",
            "content": "主要内容要点，用\\n分隔多个要点",
            "speaker_notes": "详细的演讲备注，用于语音合成",
            "layout": "title/content/summary"
        }
    ]
}

请确保：
- 第一张是标题页，最后一张是总结页
- 每张幻灯片内容简洁明了，要点不超过5个
- 演讲备注详细完整，适合语音朗读
- 严格遵循JSON格式，不要添加其他内容`, slideCount, content)

	return c.GenerateContent(prompt)
}

// generateMockPPT 生成模拟PPT内容
func (c *DashScopeClient) generateMockPPT(content string, slideCount int) (string, error) {
	// 简单的模拟PPT生成逻辑
	slides := []map[string]interface{}{
		{
			"number":        1,
			"title":         "人工智能概述",
			"content":       "人工智能的定义\n计算机科学分支\n智能机器开发",
			"speaker_notes": "人工智能是计算机科学的一个重要分支，致力于开发能够模拟人类智能的机器系统。",
			"layout":        "title",
		},
		{
			"number":        2,
			"title":         "AI主要应用领域",
			"content":       "机器学习\n自然语言处理\n计算机视觉\n专家系统\n机器人技术",
			"speaker_notes": "人工智能在各个领域都有广泛应用，从基础的机器学习到复杂的机器人系统。",
			"layout":        "content",
		},
		{
			"number":        3,
			"title":         "AI发展历程",
			"content":       "1950年代：图灵测试\n1960年代：专家系统\n1980年代：机器学习\n2010年代：深度学习\n2020年代：大语言模型",
			"speaker_notes": "人工智能的发展经历了多个重要阶段，每个阶段都有标志性的技术突破。",
			"layout":        "content",
		},
		{
			"number":        4,
			"title":         "AI技术核心",
			"content":       "算法优化\n数据处理\n模型训练\n智能决策\n自动化执行",
			"speaker_notes": "人工智能的核心技术包括算法设计、数据处理、模型训练等多个方面。",
			"layout":        "content",
		},
		{
			"number":        5,
			"title":         "AI未来展望",
			"content":       "医疗健康\n教育培训\n智能交通\n金融服务\n智能制造",
			"speaker_notes": "人工智能将在未来各个领域发挥重要作用，推动社会进步和经济发展。",
			"layout":        "content",
		},
		{
			"number":        6,
			"title":         "总结与展望",
			"content":       "AI技术持续发展\n应用领域不断扩大\n需要关注伦理问题\n未来充满机遇",
			"speaker_notes": "人工智能技术正在快速发展，我们需要在享受技术便利的同时，也要关注相关的伦理和社会问题。",
			"layout":        "summary",
		},
	}

	// 根据请求的幻灯片数量调整
	if slideCount < len(slides) {
		slides = slides[:slideCount]
	}

	result := map[string]interface{}{
		"title":              "人工智能技术概述",
		"description":        "全面介绍人工智能的基本概念、应用领域和发展历程",
		"difficulty":         "入门",
		"estimated_duration": 30,
		"slides":             slides,
	}

	// 转换为JSON字符串
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

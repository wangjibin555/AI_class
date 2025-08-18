package coze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CozeClient Coze API客户端
type CozeClient struct {
	config      *CozeConfig
	httpClient  *http.Client
	apiKey      string
	rateLimiter *RateLimiter
	sessionMgr  *SessionManager // 🆕 会话管理器
	metrics     *ClientMetrics  // 🆕 客户端指标
}

// ClientMetrics 客户端指标
type ClientMetrics struct {
	mu              sync.RWMutex
	TotalRequests   int64            `json:"total_requests"`
	SuccessRequests int64            `json:"success_requests"`
	FailedRequests  int64            `json:"failed_requests"`
	AvgResponseTime time.Duration    `json:"avg_response_time"`
	SlowQueries     int64            `json:"slow_queries"`
	RateLimitHits   int64            `json:"rate_limit_hits"`
	LastRequestTime time.Time        `json:"last_request_time"`
	HealthStatus    string           `json:"health_status"`
	ErrorCounts     map[string]int64 `json:"error_counts"`
}

// NewClientMetrics 创建客户端指标
func NewClientMetrics() *ClientMetrics {
	return &ClientMetrics{
		HealthStatus: "unknown",
		ErrorCounts:  make(map[string]int64),
	}
}

// RecordRequest 记录请求指标
func (m *ClientMetrics) RecordRequest(duration time.Duration, success bool, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.LastRequestTime = time.Now()

	if success {
		m.SuccessRequests++
		m.HealthStatus = "healthy"
	} else {
		m.FailedRequests++
		if errorType != "" {
			m.ErrorCounts[errorType]++
		}
		// 如果连续失败，更新健康状态
		if m.FailedRequests > m.SuccessRequests && m.TotalRequests > 10 {
			m.HealthStatus = "unhealthy"
		}
	}

	// 计算平均响应时间
	if m.TotalRequests > 0 {
		// 简化的移动平均计算
		if m.AvgResponseTime == 0 {
			m.AvgResponseTime = duration
		} else {
			m.AvgResponseTime = (m.AvgResponseTime + duration) / 2
		}
	}
}

// RecordSlowQuery 记录慢查询
func (m *ClientMetrics) RecordSlowQuery() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SlowQueries++
}

// RecordRateLimitHit 记录限频命中
func (m *ClientMetrics) RecordRateLimitHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RateLimitHits++
}

// GetMetrics 获取指标快照
func (m *ClientMetrics) GetMetrics() ClientMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 深拷贝错误计数
	errorCounts := make(map[string]int64)
	for k, v := range m.ErrorCounts {
		errorCounts[k] = v
	}

	return ClientMetrics{
		TotalRequests:   m.TotalRequests,
		SuccessRequests: m.SuccessRequests,
		FailedRequests:  m.FailedRequests,
		AvgResponseTime: m.AvgResponseTime,
		SlowQueries:     m.SlowQueries,
		RateLimitHits:   m.RateLimitHits,
		LastRequestTime: m.LastRequestTime,
		HealthStatus:    m.HealthStatus,
		ErrorCounts:     errorCounts,
	}
}

// SessionManager 会话管理器
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionConfig
}

// NewSessionManager 创建会话管理器
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionConfig),
	}
}

// GetOrCreateSession 获取或创建会话
func (sm *SessionManager) GetOrCreateSession(userID string) *SessionConfig {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		return session
	}

	session := &SessionConfig{
		SessionID:      uuid.New().String(),
		ConversationID: uuid.New().String(),
		UserID:         userID,
	}
	sm.sessions[userID] = session
	return session
}

// RateLimiter 请求限频器
type RateLimiter struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int
	window   time.Duration
}

// CozeAPIRequest Coze API请求结构体（基于真实API格式）
type CozeAPIRequest struct {
	BotID    string            `json:"bot_id"`
	MetaData map[string]string `json:"meta_data"`
	Messages []CozeMessage     `json:"messages"`
}

// CozeMessage Coze消息结构体
type CozeMessage struct {
	Content string `json:"content"`
	Type    string `json:"type"`
	Role    string `json:"role"`
}

// CozeAPIResponse Coze API响应结构体
type CozeAPIResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Type    string `json:"type,omitempty"`
}

// CreateConversationRequest 创建对话请求
type CreateConversationRequest struct {
	BotID string `json:"bot_id"`
}

// Conversation 对话信息
type Conversation struct {
	ID        string    `json:"id"`
	BotID     string    `json:"bot_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Message        string    `json:"message"`
	Response       string    `json:"response"`
	PPTLink        string    `json:"ppt_link,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// GeneratePPTRequest PPT生成请求
type GeneratePPTRequest struct {
	URL      string                 `json:"url"`
	Template string                 `json:"template"`
	Options  map[string]interface{} `json:"options"`
}

// PPTGenerationResponse PPT生成响应
type PPTGenerationResponse struct {
	TaskID        string    `json:"task_id"`
	Status        string    `json:"status"`
	EstimatedTime int       `json:"estimated_time"`
	CreatedAt     time.Time `json:"created_at"`
	PPTLink       string    `json:"ppt_link,omitempty"`
}

// NewRateLimiter 创建新的限频器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 清理过期的请求记录
	validRequests := make([]time.Time, 0)
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// 检查是否超过限制
	if len(rl.requests) >= rl.limit {
		return false
	}

	// 添加当前请求
	rl.requests = append(rl.requests, now)
	return true
}

// NewCozeClient 创建Coze客户端
func NewCozeClient(config *CozeConfig, apiKey string) *CozeClient {
	rateLimiter := NewRateLimiter(config.RateLimit, time.Minute)
	sessionMgr := NewSessionManager()
	metrics := NewClientMetrics()

	client := &CozeClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		apiKey:      apiKey,
		rateLimiter: rateLimiter,
		sessionMgr:  sessionMgr,
		metrics:     metrics,
	}

	// 启动监控routine
	if config.Monitoring != nil && config.Monitoring.Enabled {
		go client.startMonitoring()
	}

	return client
}

// startMonitoring 启动监控
func (c *CozeClient) startMonitoring() {
	ticker := time.NewTicker(c.config.Monitoring.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		metrics := c.metrics.GetMetrics()
		if c.config.Logging.EnableAPILog {
			c.logInfo("Client metrics", map[string]interface{}{
				"total_requests":    metrics.TotalRequests,
				"success_requests":  metrics.SuccessRequests,
				"failed_requests":   metrics.FailedRequests,
				"avg_response_time": metrics.AvgResponseTime.Milliseconds(),
				"slow_queries":      metrics.SlowQueries,
				"rate_limit_hits":   metrics.RateLimitHits,
				"health_status":     metrics.HealthStatus,
				"error_counts":      metrics.ErrorCounts,
			})
		}
	}
}

// logInfo 记录信息日志
func (c *CozeClient) logInfo(message string, fields map[string]interface{}) {
	if c.config.Logging.Level == "debug" || c.config.Logging.Level == "info" {
		if c.config.Logging.Format == "json" {
			data := map[string]interface{}{
				"level":     "info",
				"message":   message,
				"timestamp": time.Now().UTC(),
				"fields":    fields,
			}
			if jsonData, err := json.Marshal(data); err == nil {
				log.Printf("%s", string(jsonData))
			}
		} else {
			log.Printf("[INFO] %s: %+v", message, fields)
		}
	}
}

// logError 记录错误日志
func (c *CozeClient) logError(message string, err error, fields map[string]interface{}) {
	if c.config.Logging.EnableErrorLog {
		if fields == nil {
			fields = make(map[string]interface{})
		}
		if err != nil {
			fields["error"] = err.Error()
		}

		if c.config.Logging.Format == "json" {
			data := map[string]interface{}{
				"level":     "error",
				"message":   message,
				"timestamp": time.Now().UTC(),
				"fields":    fields,
			}
			if jsonData, err := json.Marshal(data); err == nil {
				log.Printf("%s", string(jsonData))
			}
		} else {
			log.Printf("[ERROR] %s: %+v", message, fields)
		}
	}
}

// GetHealthStatus 获取健康状态
func (c *CozeClient) GetHealthStatus() string {
	return c.metrics.GetMetrics().HealthStatus
}

// GetMetrics 获取客户端指标
func (c *CozeClient) GetMetrics() ClientMetrics {
	return c.metrics.GetMetrics()
}

// doRequestWithRetry 带重试机制的请求执行
func (c *CozeClient) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	requestID := uuid.New().String()
	startTime := time.Now()

	// 记录请求开始
	if c.config.Logging.EnableAPILog {
		c.logInfo("API request started", map[string]interface{}{
			"request_id": requestID,
			"method":     req.Method,
			"url":        req.URL.String(),
			"headers":    req.Header,
		})
	}

	defer func() {
		duration := time.Since(startTime)
		success := lastErr == nil

		// 检查是否为慢查询
		if c.config.Monitoring.SlowQueryThreshold > 0 && duration > c.config.Monitoring.SlowQueryThreshold {
			c.metrics.RecordSlowQuery()
			c.logError("Slow query detected", nil, map[string]interface{}{
				"request_id": requestID,
				"duration":   duration.Milliseconds(),
				"threshold":  c.config.Monitoring.SlowQueryThreshold.Milliseconds(),
			})
		}

		// 记录指标
		errorType := ""
		if lastErr != nil {
			errorType = "api_error"
		}
		c.metrics.RecordRequest(duration, success, errorType)

		// 记录请求结束
		if c.config.Logging.EnableAPILog {
			c.logInfo("API request completed", map[string]interface{}{
				"request_id": requestID,
				"duration":   duration.Milliseconds(),
				"success":    success,
				"error":      lastErr,
			})
		}
	}()

	for attempt := 0; attempt < c.config.RetryTimes; attempt++ {
		// 检查限频
		if !c.rateLimiter.Allow() {
			c.metrics.RecordRateLimitHit()
			c.logError("Rate limit exceeded", nil, map[string]interface{}{
				"request_id": requestID,
				"attempt":    attempt + 1,
			})
			return nil, fmt.Errorf("请求频率超过限制，请稍后重试")
		}

		// 复制请求体（如果需要重试）
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// 记录API载荷（如果启用）
		if c.config.Logging.LogAPIPayload && bodyBytes != nil {
			c.logInfo("API request payload", map[string]interface{}{
				"request_id": requestID,
				"payload":    string(bodyBytes),
			})
		}

		resp, err := c.httpClient.Do(req.WithContext(ctx))
		if err == nil && resp.StatusCode < 500 {
			// 记录成功响应
			if c.config.Logging.LogAPIResponse {
				c.logInfo("API response received", map[string]interface{}{
					"request_id":  requestID,
					"status_code": resp.StatusCode,
					"headers":     resp.Header,
				})
			}
			return resp, nil
		}

		// 记录重试
		if c.config.Logging.EnableErrorLog {
			c.logError("API request failed, retrying", err, map[string]interface{}{
				"request_id":  requestID,
				"attempt":     attempt + 1,
				"max_retries": c.config.RetryTimes,
				"status_code": func() int {
					if resp != nil {
						return resp.StatusCode
					}
					return 0
				}(),
			})
		}

		if resp != nil {
			resp.Body.Close()
		}

		lastErr = err
		if attempt < c.config.RetryTimes-1 {
			// 指数退避重试
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)

			// 重新设置请求体
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
	}

	c.logError("API request failed after all retries", lastErr, map[string]interface{}{
		"request_id":  requestID,
		"max_retries": c.config.RetryTimes,
	})

	return nil, fmt.Errorf("请求失败，重试%d次后仍然失败: %w", c.config.RetryTimes, lastErr)
}

// SendToCozeAPI 发送请求到Coze API（真实API调用）
func (c *CozeClient) SendToCozeAPI(ctx context.Context, message string, chatHistory []ChatMessage) (*ChatResponse, error) {
	// 生成消息ID
	messageID := uuid.New().String()

	// 获取或创建会话（使用默认用户ID）
	session := c.sessionMgr.GetOrCreateSession("default_user")

	// 构建真实的Coze API请求
	apiReq := &CozeAPIRequest{
		BotID: c.config.Bot.CozeBotID,
		MetaData: map[string]string{
			"uuid": messageID,
		},
		Messages: []CozeMessage{
			{
				Content: message,
				Type:    "question",
				Role:    "user",
			},
		},
	}

	requestBody, err := json.Marshal(apiReq)
	if err != nil {
		c.logError("Failed to marshal request", err, map[string]interface{}{
			"message_id": messageID,
			"session_id": session.SessionID,
		})
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求 - 使用真实的Coze API端点
	endpoint := c.config.APIBase + "/v1/conversation/create"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		c.logError("Failed to create HTTP request", err, map[string]interface{}{
			"endpoint":   endpoint,
			"message_id": messageID,
		})
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "AI-Classroom-Coze-Client/1.0")

	// 如果有会话ID，添加到请求头
	if session.SessionID != "" {
		req.Header.Set("X-Session-ID", session.SessionID)
	}

	// 执行请求
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logError("API request failed", err, map[string]interface{}{
			"endpoint":   endpoint,
			"message_id": messageID,
			"session_id": session.SessionID,
		})
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logError("Failed to read response body", err, map[string]interface{}{
			"message_id":  messageID,
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logError("API returned error status", nil, map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_body": string(body),
			"message_id":    messageID,
		})
		return nil, fmt.Errorf("API返回错误状态: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp CozeAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.logError("Failed to parse response", err, map[string]interface{}{
			"message_id":    messageID,
			"response_body": string(body),
		})
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API响应状态
	if apiResp.Code != 0 {
		c.logError("API returned error code", nil, map[string]interface{}{
			"api_code":    apiResp.Code,
			"api_message": apiResp.Message,
			"message_id":  messageID,
		})
		return nil, fmt.Errorf("API返回错误: %s", apiResp.Message)
	}

	// 构建聊天响应
	chatResp := &ChatResponse{
		ID:             messageID,
		ConversationID: session.ConversationID,
		Message:        message,
		Response:       fmt.Sprintf("%v", apiResp.Data),
		CreatedAt:      time.Now(),
	}

	// 尝试从响应中提取内容和PPT链接
	if data, ok := apiResp.Data["content"].(string); ok {
		chatResp.Response = data
	}

	if pptLink := c.extractPPTURL(chatResp.Response); pptLink != "" {
		chatResp.PPTLink = pptLink
		c.logInfo("PPT link extracted", map[string]interface{}{
			"message_id": messageID,
			"ppt_link":   pptLink,
		})
	}

	c.logInfo("Chat response generated successfully", map[string]interface{}{
		"message_id":   messageID,
		"session_id":   session.SessionID,
		"has_ppt_link": chatResp.PPTLink != "",
	})

	return chatResp, nil
}

// CreateConversation 创建对话（兼容性方法）
func (c *CozeClient) CreateConversation(ctx context.Context, req *CreateConversationRequest) (*Conversation, error) {
	// 使用会话管理器创建会话
	session := c.sessionMgr.GetOrCreateSession("default_user")

	c.logInfo("Conversation created", map[string]interface{}{
		"conversation_id": session.ConversationID,
		"bot_id":          req.BotID,
	})

	return &Conversation{
		ID:        session.ConversationID,
		BotID:     req.BotID,
		CreatedAt: time.Now(),
	}, nil
}

// SendMessage 发送消息（兼容性方法）
func (c *CozeClient) SendMessage(ctx context.Context, conversationID string, message string) (*ChatResponse, error) {
	// 调用新的SendToCozeAPI方法
	return c.SendToCozeAPI(ctx, message, []ChatMessage{})
}

// GeneratePPT 生成PPT
func (c *CozeClient) GeneratePPT(ctx context.Context, req *GeneratePPTRequest) (*PPTGenerationResponse, error) {
	c.logInfo("PPT generation started", map[string]interface{}{
		"url":      req.URL,
		"template": req.Template,
		"options":  req.Options,
	})

	// 构建PPT生成消息
	message := c.buildPPTGenerationMessage(req)

	// 发送消息到Coze API
	chatResp, err := c.SendToCozeAPI(ctx, message, []ChatMessage{})
	if err != nil {
		c.logError("PPT generation failed", err, map[string]interface{}{
			"url":      req.URL,
			"template": req.Template,
		})
		return nil, fmt.Errorf("发送PPT生成请求失败: %w", err)
	}

	// 构建PPT生成响应
	response := &PPTGenerationResponse{
		TaskID:        chatResp.ConversationID,
		Status:        "processing",
		EstimatedTime: 180,
		CreatedAt:     time.Now(),
	}

	// 如果响应中包含PPT链接，立即标记为完成
	if chatResp.PPTLink != "" {
		response.PPTLink = chatResp.PPTLink
		response.Status = "completed"
		c.logInfo("PPT generation completed immediately", map[string]interface{}{
			"task_id":  response.TaskID,
			"ppt_link": response.PPTLink,
		})
	} else {
		c.logInfo("PPT generation started async", map[string]interface{}{
			"task_id":        response.TaskID,
			"estimated_time": response.EstimatedTime,
		})
	}

	return response, nil
}

// buildPPTGenerationMessage 构建PPT生成消息
func (c *CozeClient) buildPPTGenerationMessage(req *GeneratePPTRequest) string {
	var message strings.Builder

	message.WriteString(c.config.Bot.PromptTemplate)
	message.WriteString("\n\n")
	message.WriteString(fmt.Sprintf("请分析以下URL内容并生成PPT：%s\n\n", req.URL))
	message.WriteString(fmt.Sprintf("模板类型：%s\n", req.Template))

	if slideCount, ok := req.Options["slide_count"]; ok {
		message.WriteString(fmt.Sprintf("幻灯片数量：%v\n", slideCount))
	}

	if language, ok := req.Options["language"]; ok {
		message.WriteString(fmt.Sprintf("语言：%v\n", language))
	}

	if style, ok := req.Options["style"]; ok {
		message.WriteString(fmt.Sprintf("风格：%v\n", style))
	}

	return message.String()
}

// extractPPTURL 从响应中提取PPT链接
func (c *CozeClient) extractPPTURL(response string) string {
	// 使用正则表达式匹配PPT链接
	patterns := []string{
		`https://chat-ppt\.com/generateResults\?generateID=([^&\s]+)&channel=([^&\s]+)`,
		`https://chat-ppt\.com/[^?\s]+\?[^?\s]*generateID=([^&\s]+)`,
		`(?i)ppt.*?https?://[^\s]+`,
		`https?://[^\s]*chat-?ppt[^\s]*`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(response)
		if len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

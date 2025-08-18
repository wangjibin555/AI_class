package coze

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PPTResult PPT生成结果
type PPTResult struct {
	ChatID         string   `json:"chat_id"`
	ConversationID string   `json:"conversation_id"`
	Content        string   `json:"content"`
	PPTLinks       []string `json:"ppt_links"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
	TokenUsage     int      `json:"token_usage"`
}

// CozeClientV3 Coze V3 API客户端
type CozeClientV3 struct {
	config      *CozeConfig
	httpClient  *http.Client
	apiKey      string
	rateLimiter *RateLimiter
	chatManager *ChatManager
	metrics     *ClientMetrics
}

// ChatManager 对话管理器
type ChatManager struct {
	mu           sync.RWMutex
	activeChats  map[string]*ChatSession
	pollInterval time.Duration
	maxPollTime  time.Duration
}

// ChatSession 对话会话
type ChatSession struct {
	ChatID         string
	ConversationID string
	BotID          string
	UserID         string
	Status         string
	CreatedAt      time.Time
	LastPolledAt   time.Time
	Messages       []ChatMessage
	PPTLinks       []string
}

// ChatV3Request V3对话请求
type ChatV3Request struct {
	BotID              string                 `json:"bot_id"`
	UserID             string                 `json:"user_id"`
	AdditionalMessages []ChatMessage          `json:"additional_messages,omitempty"`
	AutoSaveHistory    bool                   `json:"auto_save_history"`
	MetaData           map[string]interface{} `json:"meta_data,omitempty"`
}

// ChatV3Response V3对话响应
type ChatV3Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		BotID          string `json:"bot_id"`
		Status         string `json:"status"`
	} `json:"data"`
}

// NewCozeClientV3 创建Coze V3客户端
func NewCozeClientV3(config *CozeConfig) *CozeClientV3 {
	return &CozeClientV3{
		config:      config,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		apiKey:      config.APIKey,
		rateLimiter: NewRateLimiter(10, time.Minute),
		chatManager: &ChatManager{
			activeChats:  make(map[string]*ChatSession),
			pollInterval: 5 * time.Second,
			maxPollTime:  5 * time.Minute,
		},
		metrics: NewClientMetrics(),
	}
}

// CreateChat 创建对话
func (c *CozeClientV3) CreateChat(ctx context.Context, request *ChatV3Request) (*ChatV3Response, error) {
	// 模拟实现 - 在实际环境中需要调用真实的Coze API
	log.Printf("创建对话: BotID=%s, UserID=%s", request.BotID, request.UserID)

	// 生成模拟的对话ID
	chatID := fmt.Sprintf("chat_%d", time.Now().Unix())
	conversationID := fmt.Sprintf("conv_%d", time.Now().Unix())

	return &ChatV3Response{
		Code: 0,
		Msg:  "success",
		Data: struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversation_id"`
			BotID          string `json:"bot_id"`
			Status         string `json:"status"`
		}{
			ID:             chatID,
			ConversationID: conversationID,
			BotID:          request.BotID,
			Status:         "created",
		},
	}, nil
}

// RetrieveChat 查询对话
func (c *CozeClientV3) RetrieveChat(ctx context.Context, chatID, conversationID string) (*RetrieveChatResponse, error) {
	// 模拟实现
	log.Printf("查询对话: ChatID=%s, ConversationID=%s", chatID, conversationID)

	return &RetrieveChatResponse{
		Code: 0,
		Msg:  "success",
		Data: struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversation_id"`
			BotID          string `json:"bot_id"`
			Status         string `json:"status"`
			Messages       []struct {
				ID      string `json:"id"`
				Role    string `json:"role"`
				Type    string `json:"type"`
				Content string `json:"content"`
			} `json:"messages"`
			Usage struct {
				TokenCount       int `json:"token_count"`
				OutputTokenCount int `json:"output_token_count"`
				InputTokenCount  int `json:"input_token_count"`
			} `json:"usage"`
		}{
			ID:             chatID,
			ConversationID: conversationID,
			BotID:          c.config.Bot.CozeBotID,
			Status:         "completed",
			Messages: []struct {
				ID      string `json:"id"`
				Role    string `json:"role"`
				Type    string `json:"type"`
				Content string `json:"content"`
			}{
				{
					ID:      "msg_1",
					Role:    "user",
					Type:    "text",
					Content: "请根据URL生成PPT",
				},
				{
					ID:      "msg_2",
					Role:    "assistant",
					Type:    "text",
					Content: "已为您生成PPT，包含专业的内容分析和结构化的演示文稿。",
				},
			},
			Usage: struct {
				TokenCount       int `json:"token_count"`
				OutputTokenCount int `json:"output_token_count"`
				InputTokenCount  int `json:"input_token_count"`
			}{
				TokenCount:       150,
				OutputTokenCount: 100,
				InputTokenCount:  50,
			},
		},
	}, nil
}

// GeneratePPTWithPolling 生成PPT并轮询结果
func (c *CozeClientV3) GeneratePPTWithPolling(ctx context.Context, url, userID string) (*PPTResult, error) {
	log.Printf("开始生成PPT: URL=%s, UserID=%s", url, userID)

	// 创建对话请求
	chatReq := &ChatV3Request{
		BotID:  c.config.Bot.CozeBotID,
		UserID: userID,
		AdditionalMessages: []ChatMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("请根据以下URL生成PPT: %s", url),
				Type:    "text",
			},
		},
		AutoSaveHistory: true,
		MetaData: map[string]interface{}{
			"source":  "wangjibin",
			"version": "2.0",
			"url":     url,
		},
	}

	// 创建对话
	chatResp, err := c.CreateChat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("创建对话失败: %w", err)
	}

	chatID := chatResp.Data.ID
	conversationID := chatResp.Data.ConversationID

	log.Printf("对话创建成功: ChatID=%s, ConversationID=%s", chatID, conversationID)

	// 轮询对话结果
	ticker := time.NewTicker(c.chatManager.pollInterval)
	defer ticker.Stop()

	timeout := time.After(c.chatManager.maxPollTime)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("上下文取消")
		case <-timeout:
			return nil, fmt.Errorf("轮询超时")
		case <-ticker.C:
			// 查询对话状态
			retrieveResp, err := c.RetrieveChat(ctx, chatID, conversationID)
			if err != nil {
				log.Printf("查询对话失败: %v", err)
				continue
			}

			if retrieveResp.Data.Status == "completed" {
				// 提取PPT内容
				content := c.extractContentFromMessages(retrieveResp.Data.Messages)
				pptLinks := c.extractPPTLinks(content)

				// 生成标题
				title := c.generateTitle(url, content)

				return &PPTResult{
					ChatID:         chatID,
					ConversationID: conversationID,
					Content:        content,
					PPTLinks:       pptLinks,
					Title:          title,
					Status:         "completed",
					TokenUsage:     retrieveResp.Data.Usage.TokenCount,
				}, nil
			}
		}
	}
}

// extractContentFromMessages 从消息中提取内容
func (c *CozeClientV3) extractContentFromMessages(messages []struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Type    string `json:"type"`
	Content string `json:"content"`
}) string {
	var content strings.Builder

	for _, msg := range messages {
		if msg.Role == "assistant" && msg.Type == "text" {
			content.WriteString(msg.Content)
			content.WriteString("\n\n")
		}
	}

	return content.String()
}

// extractPPTLinks 从内容中提取PPT链接
func (c *CozeClientV3) extractPPTLinks(content string) []string {
	var links []string

	// 简单的链接提取逻辑
	if strings.Contains(content, "PPT") || strings.Contains(content, "演示") {
		// 模拟PPT链接
		links = append(links, "https://example.com/ppt/generated.pptx")
	}

	return links
}

// generateTitle 生成标题
func (c *CozeClientV3) generateTitle(url, content string) string {
	if strings.Contains(url, "mysql") || strings.Contains(url, "数据库") {
		return "MySQL数据库优化与性能调优指南"
	} else if strings.Contains(url, "blog") || strings.Contains(url, "csdn") {
		return "技术博客内容分析与总结"
	} else {
		return "基于网络内容的专业分析报告"
	}
}

// RetrieveChatResponse 查询对话响应
type RetrieveChatResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		BotID          string `json:"bot_id"`
		Status         string `json:"status"`
		Messages       []struct {
			ID      string `json:"id"`
			Role    string `json:"role"`
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"messages"`
		Usage struct {
			TokenCount       int `json:"token_count"`
			OutputTokenCount int `json:"output_token_count"`
			InputTokenCount  int `json:"input_token_count"`
		} `json:"usage"`
	} `json:"data"`
}

// AddSession 添加会话
func (cm *ChatManager) AddSession(session *ChatSession) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.activeChats[session.ChatID] = session
}

// GetSession 获取会话
func (cm *ChatManager) GetSession(chatID string) (*ChatSession, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	session, exists := cm.activeChats[chatID]
	return session, exists
}

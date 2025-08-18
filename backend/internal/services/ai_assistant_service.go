package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/pkg/ai"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AIAssistantService AI助手服务
type AIAssistantService struct {
	db       *gorm.DB
	aiClient *ai.DashScopeClient
}

// NewAIAssistantService 创建AI助手服务
func NewAIAssistantService(db *gorm.DB, aiClient *ai.DashScopeClient) *AIAssistantService {
	return &AIAssistantService{
		db:       db,
		aiClient: aiClient,
	}
}

// ChatRequest AI对话请求
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
	Context string `json:"context,omitempty"`
}

// ChatResponse AI对话响应
type ChatResponse struct {
	Response string `json:"response"`
	Context  string `json:"context,omitempty"`
}

// ChatHistoryResponse 对话历史响应
type ChatHistoryResponse struct {
	ID        uint      `json:"id"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	CreatedAt time.Time `json:"created_at"`
}

// GetChatHistoryRequest 获取对话历史请求
type GetChatHistoryRequest struct {
	Page  int `json:"page" form:"page"`
	Limit int `json:"limit" form:"limit"`
}

// GetChatHistoryResponse 获取对话历史响应
type GetChatHistoryResponse struct {
	History     []ChatHistoryResponse `json:"history"`
	Total       int64                 `json:"total"`
	Page        int                   `json:"page"`
	Limit       int                   `json:"limit"`
	TotalPages  int                   `json:"total_pages"`
	HasNext     bool                  `json:"has_next"`
	HasPrevious bool                  `json:"has_previous"`
}

// Chat 与AI助手对话
func (s *AIAssistantService) Chat(userID uint, req *ChatRequest) (*ChatResponse, error) {
	// 构建AI对话提示
	prompt := s.buildChatPrompt(req.Message, req.Context)

	// 调用AI服务
	aiResponse, err := s.aiClient.SimpleChat(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI服务调用失败: %v", err)
	}

	// 解析AI响应
	response := &ChatResponse{
		Response: aiResponse,
		Context:  req.Context, // 保持原有上下文
	}

	// 保存对话记录到数据库
	chatRecord := &models.AIAssistantChat{
		UserID:    userID,
		Message:   req.Message,
		Response:  aiResponse,
		Context:   req.Context,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(chatRecord).Error; err != nil {
		// 记录保存失败不影响对话功能，只记录日志
		fmt.Printf("保存对话记录失败: %v\n", err)
	}

	return response, nil
}

// GetChatHistory 获取对话历史
func (s *AIAssistantService) GetChatHistory(userID uint, req *GetChatHistoryRequest) (*GetChatHistoryResponse, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.Limit

	// 查询总数
	var total int64
	if err := s.db.Model(&models.AIAssistantChat{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询总数失败: %v", err)
	}

	// 查询对话历史
	var chats []models.AIAssistantChat
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("查询对话历史失败: %v", err)
	}

	// 转换为响应格式
	history := make([]ChatHistoryResponse, len(chats))
	for i, chat := range chats {
		history[i] = ChatHistoryResponse{
			ID:        chat.ID,
			Message:   chat.Message,
			Response:  chat.Response,
			CreatedAt: chat.CreatedAt,
		}
	}

	// 计算分页信息
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
	hasNext := req.Page < totalPages
	hasPrevious := req.Page > 1

	return &GetChatHistoryResponse{
		History:     history,
		Total:       total,
		Page:        req.Page,
		Limit:       req.Limit,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}, nil
}

// ClearChatHistory 清空对话历史
func (s *AIAssistantService) ClearChatHistory(userID uint) error {
	if err := s.db.Where("user_id = ?", userID).Delete(&models.AIAssistantChat{}).Error; err != nil {
		return fmt.Errorf("清空对话历史失败: %v", err)
	}
	return nil
}

// GetChatStats 获取对话统计
func (s *AIAssistantService) GetChatStats(userID uint) (map[string]interface{}, error) {
	var stats struct {
		TotalChats     int64 `json:"total_chats"`
		TodayChats     int64 `json:"today_chats"`
		ThisWeekChats  int64 `json:"this_week_chats"`
		ThisMonthChats int64 `json:"this_month_chats"`
	}

	// 总对话数
	if err := s.db.Model(&models.AIAssistantChat{}).Where("user_id = ?", userID).Count(&stats.TotalChats).Error; err != nil {
		return nil, fmt.Errorf("查询总对话数失败: %v", err)
	}

	// 今日对话数
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.AIAssistantChat{}).
		Where("user_id = ? AND created_at >= ?", userID, today).
		Count(&stats.TodayChats).Error; err != nil {
		return nil, fmt.Errorf("查询今日对话数失败: %v", err)
	}

	// 本周对话数
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	weekStart = weekStart.Truncate(24 * time.Hour)
	if err := s.db.Model(&models.AIAssistantChat{}).
		Where("user_id = ? AND created_at >= ?", userID, weekStart).
		Count(&stats.ThisWeekChats).Error; err != nil {
		return nil, fmt.Errorf("查询本周对话数失败: %v", err)
	}

	// 本月对话数
	monthStart := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	monthStart = monthStart.Truncate(24 * time.Hour)
	if err := s.db.Model(&models.AIAssistantChat{}).
		Where("user_id = ? AND created_at >= ?", userID, monthStart).
		Count(&stats.ThisMonthChats).Error; err != nil {
		return nil, fmt.Errorf("查询本月对话数失败: %v", err)
	}

	return map[string]interface{}{
		"total_chats":      stats.TotalChats,
		"today_chats":      stats.TodayChats,
		"this_week_chats":  stats.ThisWeekChats,
		"this_month_chats": stats.ThisMonthChats,
	}, nil
}

// buildChatPrompt 构建AI对话提示
func (s *AIAssistantService) buildChatPrompt(message, context string) string {
	// 基础系统提示
	systemPrompt := `你是一个智能AI助手，专门帮助用户学习和解答问题。请用友好、专业的态度回答用户的问题。

你的特点：
1. 知识渊博，能够解答各种学科问题
2. 语言简洁明了，易于理解
3. 鼓励用户思考和探索
4. 提供实用的建议和指导
5. 保持耐心和友好的态度

请根据用户的问题提供准确、有用的回答。`

	// 如果有上下文，添加到提示中
	if context != "" {
		return fmt.Sprintf("%s\n\n对话上下文：%s\n\n用户问题：%s\n\n请回答：", systemPrompt, context, message)
	}

	return fmt.Sprintf("%s\n\n用户问题：%s\n\n请回答：", systemPrompt, message)
}

// GetPopularQuestions 获取热门问题
func (s *AIAssistantService) GetPopularQuestions() []string {
	return []string{
		"如何提高学习效率？",
		"如何制作一份好的PPT？",
		"如何记忆知识点？",
		"如何准备考试？",
		"如何培养学习习惯？",
		"如何提高专注力？",
		"如何制定学习计划？",
		"如何克服学习困难？",
	}
}

// GetSuggestedTopics 获取建议话题
func (s *AIAssistantService) GetSuggestedTopics() []map[string]string {
	return []map[string]string{
		{"title": "学习方法", "description": "探讨高效的学习方法和技巧"},
		{"title": "时间管理", "description": "如何合理安排学习时间"},
		{"title": "记忆技巧", "description": "科学的记忆方法和技巧"},
		{"title": "考试准备", "description": "如何有效准备各种考试"},
		{"title": "知识整理", "description": "如何整理和归纳知识点"},
		{"title": "学习工具", "description": "推荐实用的学习工具和软件"},
	}
}

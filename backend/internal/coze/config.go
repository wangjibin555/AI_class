package coze

import (
	"time"
)

// CozeConfig Coze智能体配置
type CozeConfig struct {
	APIBase    string        `yaml:"api_base"`
	APIKey     string        `yaml:"api_key"` // 🆕 Coze API密钥
	Token      string        `yaml:"token"`   // 🆕 个人访问令牌 (工作流专用)
	Timeout    time.Duration `yaml:"timeout"`
	RetryTimes int           `yaml:"retry_times"`
	RateLimit  int           `yaml:"rate_limit"`

	// 🆕 工作流配置
	Workflow *WorkflowConfig `yaml:"workflow"`

	// 保留现有配置以支持向后兼容
	Bot        *BotConfig        `yaml:"bot"`
	Chat       *ChatConfig       `yaml:"chat"`       // 🆕 Chat V3 API配置
	Monitoring *MonitoringConfig `yaml:"monitoring"` // 🆕 监控配置
	Logging    *LoggingConfig    `yaml:"logging"`    // 🆕 日志配置
}

// BotConfig 智能体配置
type BotConfig struct {
	ID                 string   `yaml:"id" json:"id"`
	Name               string   `yaml:"name" json:"name"`
	Description        string   `yaml:"description" json:"description"`
	CozeBotID          string   `yaml:"coze_bot_id" json:"coze_bot_id"`
	BotVersion         string   `yaml:"bot_version" json:"bot_version"` // 🆕 Bot版本
	Scene              int      `yaml:"scene" json:"scene"`             // 🆕 场景类型
	TemplateType       string   `yaml:"template_type" json:"template_type"`
	MaxSlides          int      `yaml:"max_slides" json:"max_slides"`
	SupportedLanguages []string `yaml:"supported_languages" json:"supported_languages"`
	Capabilities       []string `yaml:"capabilities" json:"capabilities"`
	PromptTemplate     string   `yaml:"prompt_template" json:"prompt_template"`
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	// 🆕 支持多个工作流
	PPTGeneration      *SingleWorkflowConfig `yaml:"ppt_generation" json:"ppt_generation"`
	ExerciseGeneration *SingleWorkflowConfig `yaml:"exercise_generation" json:"exercise_generation"`

	// 🆕 向后兼容字段（废弃）
	WorkflowID     string        `yaml:"workflow_id,omitempty" json:"workflow_id,omitempty"`
	PollInterval   time.Duration `yaml:"poll_interval,omitempty" json:"poll_interval,omitempty"`
	MaxPollTime    time.Duration `yaml:"max_poll_time,omitempty" json:"max_poll_time,omitempty"`
	MaxRetries     int           `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	RetryDelay     time.Duration `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
	TimeoutSeconds int           `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
}

// SingleWorkflowConfig 单个工作流配置
type SingleWorkflowConfig struct {
	WorkflowID     string        `yaml:"workflow_id" json:"workflow_id"`
	PollInterval   time.Duration `yaml:"poll_interval" json:"poll_interval"`
	MaxPollTime    time.Duration `yaml:"max_poll_time" json:"max_poll_time"`
	MaxRetries     int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`
	TimeoutSeconds int           `yaml:"timeout_seconds" json:"timeout_seconds"`
}

// ChatConfig Chat V3 API配置
type ChatConfig struct {
	PollInterval    time.Duration `yaml:"poll_interval"`     // 轮询间隔
	MaxPollTime     time.Duration `yaml:"max_poll_time"`     // 最大轮询时间
	AutoSaveHistory bool          `yaml:"auto_save_history"` // 自动保存历史
	MaxRetries      int           `yaml:"max_retries"`       // 最大重试次数
	RetryDelay      time.Duration `yaml:"retry_delay"`       // 重试延迟
}

// SessionConfig 会话配置
type SessionConfig struct {
	SessionID      string `yaml:"session_id" json:"session_id"`           // 🆕 会话ID
	ConversationID string `yaml:"conversation_id" json:"conversation_id"` // 🆕 对话ID
	UserID         string `yaml:"user_id" json:"user_id"`                 // 🆕 用户ID
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled            bool          `yaml:"enabled"`              // 启用监控
	MetricsInterval    time.Duration `yaml:"metrics_interval"`     // 指标收集间隔
	HealthCheck        bool          `yaml:"health_check"`         // 启用健康检查
	PerformanceTrack   bool          `yaml:"performance_track"`    // 性能追踪
	ErrorTracking      bool          `yaml:"error_tracking"`       // 错误追踪
	SlowQueryThreshold time.Duration `yaml:"slow_query_threshold"` // 慢查询阈值
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level          string `yaml:"level"`            // 日志级别 (debug, info, warn, error)
	Format         string `yaml:"format"`           // 日志格式 (json, text)
	EnableAPILog   bool   `yaml:"enable_api_log"`   // 启用API请求日志
	EnableErrorLog bool   `yaml:"enable_error_log"` // 启用错误日志
	LogAPIPayload  bool   `yaml:"log_api_payload"`  // 记录API请求载荷
	LogAPIResponse bool   `yaml:"log_api_response"` // 记录API响应
}

// RateLimitConfig 限频配置
type RateLimitConfig struct {
	RequestsPerMinute int           `yaml:"requests_per_minute"`
	BurstSize         int           `yaml:"burst_size"`
	WindowSize        time.Duration `yaml:"window_size"`
}

// DefaultCozeConfig 默认配置
func DefaultCozeConfig() *CozeConfig {
	return &CozeConfig{
		APIBase:    "https://api.coze.cn",
		APIKey:     "", // 需要从环境变量或配置文件中设置
		Token:      "", // 需要从环境变量或配置文件中设置
		Timeout:    300 * time.Second,
		RetryTimes: 3,
		RateLimit:  100,

		// 🆕 工作流默认配置
		Workflow: &WorkflowConfig{
			WorkflowID:     "", // 需要从配置文件中设置
			PollInterval:   2 * time.Second,
			MaxPollTime:    300 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
			TimeoutSeconds: 300,
		},
		Bot: &BotConfig{
			ID:                 "7532702288876896298",
			Name:               "Coze PPT生成器",
			Description:        "专业的PPT生成智能体",
			CozeBotID:          "7532702288876896298",
			BotVersion:         "1",
			Scene:              2, // 默认场景
			TemplateType:       "professional",
			MaxSlides:          25,
			SupportedLanguages: []string{"zh-CN", "en-US"},
			Capabilities:       []string{"deep_analysis", "professional_ppt", "template_customization"},
			PromptTemplate:     "请根据以下URL内容生成专业的PPT演示文稿，包含清晰的标题、要点和结构化的内容布局。",
		},
		Chat: &ChatConfig{
			PollInterval:    2 * time.Second,
			MaxPollTime:     300 * time.Second,
			AutoSaveHistory: true,
			MaxRetries:      3,
			RetryDelay:      1 * time.Second,
		},
		Monitoring: &MonitoringConfig{
			Enabled:            true,
			MetricsInterval:    30 * time.Second,
			HealthCheck:        true,
			PerformanceTrack:   true,
			ErrorTracking:      true,
			SlowQueryThreshold: 5 * time.Second,
		},
		Logging: &LoggingConfig{
			Level:          "info",
			Format:         "json",
			EnableAPILog:   true,
			EnableErrorLog: true,
			LogAPIPayload:  false, // 生产环境建议关闭
			LogAPIResponse: false, // 生产环境建议关闭
		},
	}
}

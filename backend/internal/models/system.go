package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// AudioStatus 音频处理状态
type AudioStatus string

const (
	AudioStatusProcessing AudioStatus = "processing"
	AudioStatusCompleted  AudioStatus = "completed"
	AudioStatusFailed     AudioStatus = "failed"
)

// AudioMetadata 音频元数据模型
type AudioMetadata struct {
	ID         uint        `json:"id" gorm:"primaryKey"`
	CourseID   uint        `json:"course_id" gorm:"not null;index;comment:课件ID"`
	SlideID    *uint       `json:"slide_id" gorm:"index;comment:幻灯片ID"`
	FileName   string      `json:"file_name" gorm:"size:255;not null;comment:文件名"`
	FilePath   string      `json:"file_path" gorm:"size:500;not null;comment:文件路径"`
	FileSize   int64       `json:"file_size" gorm:"not null;comment:文件大小（字节）"`
	Duration   int         `json:"duration" gorm:"not null;comment:时长（秒）"`
	Format     string      `json:"format" gorm:"size:20;default:'mp3';comment:音频格式"`
	SampleRate int         `json:"sample_rate" gorm:"default:16000;comment:采样率"`
	BitRate    int         `json:"bit_rate" gorm:"default:128;comment:比特率"`
	URL        string      `json:"url" gorm:"size:500;comment:访问URL"`
	CDNURL     string      `json:"cdn_url" gorm:"size:500;comment:CDN URL"`
	Status     AudioStatus `json:"status" gorm:"type:enum('processing','completed','failed');default:'processing';comment:处理状态"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// TableName 指定表名
func (AudioMetadata) TableName() string {
	return "audio_metadata"
}

// GetFormattedDuration 获取格式化的时长
func (am *AudioMetadata) GetFormattedDuration() string {
	if am.Duration == 0 {
		return "0:00"
	}
	minutes := am.Duration / 60
	seconds := am.Duration % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// GetFormattedFileSize 获取格式化的文件大小
func (am *AudioMetadata) GetFormattedFileSize() string {
	size := float64(am.FileSize)
	units := []string{"B", "KB", "MB", "GB"}

	for i, unit := range units {
		if size < 1024 || i == len(units)-1 {
			if i == 0 {
				return fmt.Sprintf("%.0f %s", size, unit)
			}
			return fmt.Sprintf("%.2f %s", size, unit)
		}
		size /= 1024
	}
	return fmt.Sprintf("%.2f GB", size)
}

// IsCompleted 检查是否处理完成
func (am *AudioMetadata) IsCompleted() bool {
	return am.Status == AudioStatusCompleted
}

// MarkAsCompleted 标记为完成
func (am *AudioMetadata) MarkAsCompleted() {
	am.Status = AudioStatusCompleted
}

// MarkAsFailed 标记为失败
func (am *AudioMetadata) MarkAsFailed() {
	am.Status = AudioStatusFailed
}

// ConfigType 配置类型
type ConfigType string

const (
	ConfigTypeString  ConfigType = "string"
	ConfigTypeNumber  ConfigType = "number"
	ConfigTypeBoolean ConfigType = "boolean"
	ConfigTypeJSON    ConfigType = "json"
)

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	ConfigKey   string     `json:"config_key" gorm:"uniqueIndex;size:100;not null;comment:配置键"`
	ConfigValue string     `json:"config_value" gorm:"type:text;comment:配置值"`
	ConfigType  ConfigType `json:"config_type" gorm:"type:enum('string','number','boolean','json');default:'string';comment:配置类型"`
	Description string     `json:"description" gorm:"size:500;comment:配置描述"`
	IsActive    bool       `json:"is_active" gorm:"default:true;comment:是否启用"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// GetStringValue 获取字符串值
func (sc *SystemConfig) GetStringValue() string {
	return sc.ConfigValue
}

// GetIntValue 获取整数值
func (sc *SystemConfig) GetIntValue() int {
	if sc.ConfigType != ConfigTypeNumber {
		return 0
	}
	var value int
	fmt.Sscanf(sc.ConfigValue, "%d", &value)
	return value
}

// GetBoolValue 获取布尔值
func (sc *SystemConfig) GetBoolValue() bool {
	if sc.ConfigType != ConfigTypeBoolean {
		return false
	}
	return sc.ConfigValue == "true"
}

// GetJSONValue 获取JSON值
func (sc *SystemConfig) GetJSONValue(v interface{}) error {
	if sc.ConfigType != ConfigTypeJSON {
		return fmt.Errorf("config type is not json")
	}
	return json.Unmarshal([]byte(sc.ConfigValue), v)
}

// SetValue 设置值
func (sc *SystemConfig) SetValue(value interface{}) error {
	switch sc.ConfigType {
	case ConfigTypeString:
		sc.ConfigValue = fmt.Sprintf("%v", value)
	case ConfigTypeNumber:
		sc.ConfigValue = fmt.Sprintf("%v", value)
	case ConfigTypeBoolean:
		sc.ConfigValue = fmt.Sprintf("%v", value)
	case ConfigTypeJSON:
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		sc.ConfigValue = string(data)
	}
	return nil
}

// OperationStatus 操作状态
type OperationStatus string

const (
	OperationStatusSuccess OperationStatus = "success"
	OperationStatusFailed  OperationStatus = "failed"
)

// RequestData 请求数据
type RequestData map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (rd *RequestData) Scan(value interface{}) error {
	if value == nil {
		*rd = nil
		return nil
	}

	switch s := value.(type) {
	case []byte:
		return json.Unmarshal(s, rd)
	case string:
		return json.Unmarshal([]byte(s), rd)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (rd RequestData) Value() (driver.Value, error) {
	if rd == nil {
		return nil, nil
	}
	return json.Marshal(rd)
}

// ResponseData 响应数据
type ResponseData map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (rd *ResponseData) Scan(value interface{}) error {
	if value == nil {
		*rd = nil
		return nil
	}

	switch s := value.(type) {
	case []byte:
		return json.Unmarshal(s, rd)
	case string:
		return json.Unmarshal([]byte(s), rd)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (rd ResponseData) Value() (driver.Value, error) {
	if rd == nil {
		return nil, nil
	}
	return json.Marshal(rd)
}

// OperationLog 操作日志模型
type OperationLog struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	UserID        *uint           `json:"user_id" gorm:"index;comment:用户ID"`
	OperationType string          `json:"operation_type" gorm:"size:50;not null;comment:操作类型"`
	OperationDesc string          `json:"operation_desc" gorm:"size:500;comment:操作描述"`
	ResourceType  string          `json:"resource_type" gorm:"size:50;comment:资源类型"`
	ResourceID    *uint           `json:"resource_id" gorm:"comment:资源ID"`
	RequestData   RequestData     `json:"request_data" gorm:"type:json;comment:请求数据"`
	ResponseData  ResponseData    `json:"response_data" gorm:"type:json;comment:响应数据"`
	IPAddress     string          `json:"ip_address" gorm:"size:45;comment:IP地址"`
	UserAgent     string          `json:"user_agent" gorm:"size:1000;comment:用户代理"`
	ExecutionTime int             `json:"execution_time" gorm:"comment:执行时间（毫秒）"`
	Status        OperationStatus `json:"status" gorm:"type:enum('success','failed');default:'success';comment:执行状态"`
	ErrorMessage  string          `json:"error_message" gorm:"type:text;comment:错误信息"`
	CreatedAt     time.Time       `json:"created_at"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}

// IsSuccess 检查操作是否成功
func (ol *OperationLog) IsSuccess() bool {
	return ol.Status == OperationStatusSuccess
}

// GetFormattedExecutionTime 获取格式化的执行时间
func (ol *OperationLog) GetFormattedExecutionTime() string {
	if ol.ExecutionTime < 1000 {
		return fmt.Sprintf("%dms", ol.ExecutionTime)
	}
	return fmt.Sprintf("%.2fs", float64(ol.ExecutionTime)/1000.0)
}

// GetStatusText 获取状态文本
func (ol *OperationLog) GetStatusText() string {
	switch ol.Status {
	case OperationStatusSuccess:
		return "成功"
	case OperationStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

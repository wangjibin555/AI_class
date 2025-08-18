package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QuizStatus 练习状态
type QuizStatus string

const (
	QuizStatusActive   QuizStatus = "active"
	QuizStatusInactive QuizStatus = "inactive"
	QuizStatusArchived QuizStatus = "archived"
)

// QuestionType 题目类型
type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "single_choice"
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeTrueFalse      QuestionType = "true_false"
	QuestionTypeFillBlank      QuestionType = "fill_blank"
)

// Difficulty 难度等级
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyNormal Difficulty = "normal"
	DifficultyHard   Difficulty = "hard"
)

// QuestionOptions 题目选项
type QuestionOptions []string

// Scan 实现 sql.Scanner 接口
func (qo *QuestionOptions) Scan(value interface{}) error {
	if value == nil {
		*qo = nil
		return nil
	}

	switch s := value.(type) {
	case []byte:
		return json.Unmarshal(s, qo)
	case string:
		return json.Unmarshal([]byte(s), qo)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (qo QuestionOptions) Value() (driver.Value, error) {
	if qo == nil {
		return nil, nil
	}
	return json.Marshal(qo)
}

// Keywords 关键词
type Keywords []string

// Scan 实现 sql.Scanner 接口（增强版）
func (k *Keywords) Scan(value interface{}) error {
	if value == nil {
		*k = make([]string, 0) // 确保不是nil
		return nil
	}

	switch s := value.(type) {
	case []byte:
		if len(s) == 0 {
			*k = make([]string, 0)
			return nil
		}
		return json.Unmarshal(s, k)
	case string:
		if s == "" || s == "null" {
			*k = make([]string, 0)
			return nil
		}
		return json.Unmarshal([]byte(s), k)
	}
	return fmt.Errorf("无法将 %T 转换为 Keywords", value)
}

// Value 实现 driver.Valuer 接口（增强版）
func (k Keywords) Value() (driver.Value, error) {
	if k == nil || len(k) == 0 {
		return "[]", nil // 返回空数组而不是null
	}
	return json.Marshal(k)
}

// IsEmpty 检查是否为空
func (k Keywords) IsEmpty() bool {
	return len(k) == 0
}

// String 转换为字符串
func (k Keywords) String() string {
	return strings.Join(k, ", ")
}

// Contains 检查是否包含指定关键词
func (k Keywords) Contains(keyword string) bool {
	for _, kw := range k {
		if kw == keyword {
			return true
		}
	}
	return false
}

// Add 添加关键词（去重）
func (k *Keywords) Add(keyword string) {
	if keyword != "" && !k.Contains(keyword) {
		*k = append(*k, keyword)
	}
}

// AddAll 批量添加关键词（去重）
func (k *Keywords) AddAll(keywords []string) {
	for _, keyword := range keywords {
		k.Add(keyword)
	}
}

// Limit 限制关键词数量
func (k *Keywords) Limit(maxCount int) {
	if len(*k) > maxCount {
		*k = (*k)[:maxCount]
	}
}

// Quiz 练习/测验模型
type Quiz struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	CourseID       uint       `json:"course_id" gorm:"not null;index;comment:课件ID"`
	UserID         uint       `json:"user_id" gorm:"not null;index;comment:创建者ID"`
	Title          string     `json:"title" gorm:"size:200;default:'课后练习';comment:练习标题"`
	Description    string     `json:"description" gorm:"type:text;comment:练习描述"`
	TotalQuestions int        `json:"total_questions" gorm:"default:0;comment:题目总数"`
	QuestionCount  int        `json:"question_count" gorm:"default:0;comment:题目数量"`
	TotalPoints    int        `json:"total_points" gorm:"default:0;comment:总分"`
	TimeLimit      int        `json:"time_limit" gorm:"default:0;comment:时间限制（分钟），0表示无限制"`
	PassScore      int        `json:"pass_score" gorm:"default:60;comment:及格分数"`
	AttemptLimit   int        `json:"attempt_limit" gorm:"default:3;comment:答题次数限制"`
	Difficulty     Difficulty `json:"difficulty" gorm:"type:enum('easy','normal','hard');default:'normal';comment:难度等级"`
	Status         QuizStatus `json:"status" gorm:"type:enum('active','inactive','archived');default:'active';comment:状态"`
	AttemptCount   int        `json:"attempt_count" gorm:"default:0;comment:答题总次数"`
	PassCount      int        `json:"pass_count" gorm:"default:0;comment:通过人数"`
	AverageScore   float64    `json:"average_score" gorm:"default:0;comment:平均分"`

	// 🆕 练习生成相关字段
	GenerationSource string  `json:"generation_source" gorm:"size:50;default:'manual';index;comment:生成来源: manual/ai_workflow"`
	WorkflowID       *string `json:"workflow_id,omitempty" gorm:"size:100;index;comment:AI工作流ID"`
	GenerationParams *string `json:"generation_params,omitempty" gorm:"type:json;comment:生成参数"`
	SourceURL        *string `json:"source_url,omitempty" gorm:"type:text;comment:生成时使用的源URL"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	Questions []Question `json:"questions" gorm:"foreignKey:QuizID"`
}

// TableName 指定表名
func (Quiz) TableName() string {
	return "quizzes"
}

// IsActive 检查练习是否激活
func (q *Quiz) IsActive() bool {
	return q.Status == QuizStatusActive
}

// HasTimeLimit 检查是否有时间限制
func (q *Quiz) HasTimeLimit() bool {
	return q.TimeLimit > 0
}

// IsAIGenerated 检查是否为AI生成的练习
func (q *Quiz) IsAIGenerated() bool {
	return q.GenerationSource == "ai_workflow"
}

// GetWorkflowID 获取工作流ID
func (q *Quiz) GetWorkflowID() string {
	if q.WorkflowID != nil {
		return *q.WorkflowID
	}
	return ""
}

// GetSourceURL 获取源URL
func (q *Quiz) GetSourceURL() string {
	if q.SourceURL != nil {
		return *q.SourceURL
	}
	return ""
}

// SetGenerationInfo 设置生成信息
func (q *Quiz) SetGenerationInfo(workflowID, sourceURL string, params interface{}) error {
	q.GenerationSource = "ai_workflow"
	q.WorkflowID = &workflowID
	q.SourceURL = &sourceURL

	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return err
		}
		paramsStr := string(paramsJSON)
		q.GenerationParams = &paramsStr
	}

	return nil
}

// UpdateStats 更新统计信息
func (q *Quiz) UpdateStats() {
	if q.AttemptCount > 0 {
		// 这里需要从数据库查询实际的通过率和平均分
		// 实际实现时会通过service层来处理
	}
}

// Question 题目模型
type Question struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	QuizID         uint            `json:"quiz_id" gorm:"not null;index;comment:练习ID"`
	CourseID       uint            `json:"course_id" gorm:"not null;index;comment:课件ID"`
	SlideID        *uint           `json:"slide_id" gorm:"index;comment:关联幻灯片ID"`
	Type           QuestionType    `json:"type" gorm:"type:enum('single_choice','multiple_choice','true_false','fill_blank');not null;comment:题目类型"`
	Question       string          `json:"question" gorm:"type:text;not null;comment:题目内容"`
	QuestionNumber int             `json:"question_number" gorm:"default:0;comment:题目序号"`
	Options        QuestionOptions `json:"options" gorm:"type:json;comment:选择题选项"`
	CorrectAnswer  string          `json:"correct_answer" gorm:"type:text;not null;comment:正确答案"`
	Explanation    string          `json:"explanation" gorm:"type:text;comment:答案解释"`
	Difficulty     Difficulty      `json:"difficulty" gorm:"type:enum('easy','normal','hard');default:'normal';comment:难度等级"`
	Points         int             `json:"points" gorm:"default:1;comment:分值"`
	OrderNum       int             `json:"order_num" gorm:"default:0;comment:题目顺序"`
	Keywords       Keywords        `json:"keywords" gorm:"type:json;comment:关键词"`

	// 🆕 AI生成相关字段
	SourceQuestionID *int `json:"source_question_id,omitempty" gorm:"index;comment:来源题目ID(工作流返回的ID)"`

	AnswerCount  int       `json:"answer_count" gorm:"default:0;comment:回答次数"`
	CorrectCount int       `json:"correct_count" gorm:"default:0;comment:正确次数"`
	CorrectRate  float64   `json:"correct_rate" gorm:"default:0;comment:正确率"`
	AvgDuration  float64   `json:"avg_duration" gorm:"default:0;comment:平均答题时长"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Question) TableName() string {
	return "questions"
}

// IsSingleChoice 检查是否为单选题
func (q *Question) IsSingleChoice() bool {
	return q.Type == QuestionTypeSingleChoice
}

// IsMultipleChoice 检查是否为多选题
func (q *Question) IsMultipleChoice() bool {
	return q.Type == QuestionTypeMultipleChoice
}

// IsTrueFalse 检查是否为判断题
func (q *Question) IsTrueFalse() bool {
	return q.Type == QuestionTypeTrueFalse
}

// IsFillBlank 检查是否为填空题
func (q *Question) IsFillBlank() bool {
	return q.Type == QuestionTypeFillBlank
}

// GetDifficultyText 获取难度文本
func (q *Question) GetDifficultyText() string {
	switch q.Difficulty {
	case DifficultyEasy:
		return "简单"
	case DifficultyNormal:
		return "普通"
	case DifficultyHard:
		return "困难"
	default:
		return "普通"
	}
}

// UpdateStats 更新统计信息
func (q *Question) UpdateStats() {
	if q.AnswerCount > 0 {
		q.CorrectRate = float64(q.CorrectCount) / float64(q.AnswerCount) * 100
	}
}

package coze

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ExerciseWorkflowClient 练习生成工作流客户端
type ExerciseWorkflowClient struct {
	workflowClient *WorkflowClient
	config         *SingleWorkflowConfig
}

// NewExerciseWorkflowClient 创建练习生成工作流客户端
func NewExerciseWorkflowClient(cozeConfig *CozeConfig) *ExerciseWorkflowClient {
	workflowClient := NewWorkflowClient(cozeConfig)

	var exerciseConfig *SingleWorkflowConfig
	if cozeConfig.Workflow != nil && cozeConfig.Workflow.ExerciseGeneration != nil {
		exerciseConfig = cozeConfig.Workflow.ExerciseGeneration
	} else {
		// 使用默认配置
		exerciseConfig = &SingleWorkflowConfig{
			WorkflowID:     "7533813173200338996",
			PollInterval:   2 * time.Second,
			MaxPollTime:    300 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
			TimeoutSeconds: 300,
		}
	}

	return &ExerciseWorkflowClient{
		workflowClient: workflowClient,
		config:         exerciseConfig,
	}
}

// ExerciseWorkflowRequest 练习生成工作流请求
type ExerciseWorkflowRequest struct {
	SourceURL     string `json:"url"` // 课程源URL
	UserID        string `json:"user_id,omitempty"`
	CourseID      string `json:"course_id,omitempty"`
	QuestionCount int    `json:"question_count,omitempty"`
	Difficulty    string `json:"difficulty,omitempty"`
}

// ExerciseWorkflowResult 练习生成工作流结果
type ExerciseWorkflowResult struct {
	ExecuteID              string             `json:"execute_id"`
	Status                 string             `json:"status"`
	CourseName             string             `json:"course_name"`
	TotalQuestions         int                `json:"total_questions"`
	Questions              []ExerciseQuestion `json:"questions"`
	DifficultyDistribution DifficultyDist     `json:"difficulty_distribution"`
	ProcessingTime         int                `json:"processing_time"`
	ErrorMessage           string             `json:"error_message,omitempty"`
}

// ExerciseQuestion 练习题目结构
type ExerciseQuestion struct {
	ID            int                    `json:"id"`
	Question      string                 `json:"question"`
	Options       map[string]interface{} `json:"options"`
	CorrectAnswer string                 `json:"correct_answer"`
	Difficulty    string                 `json:"difficulty"`
	Explanation   string                 `json:"explanation,omitempty"`
}

// DifficultyDist 难度分布
type DifficultyDist struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

// GenerateExercise 生成练习
func (c *ExerciseWorkflowClient) GenerateExercise(ctx context.Context, req *ExerciseWorkflowRequest) (*ExerciseWorkflowResult, error) {
	log.Printf("开始练习生成工作流，URL: %s", req.SourceURL)

	// 准备工作流参数 - 包含用户选择的参数
	parameters := map[string]interface{}{
		"keyword":        req.SourceURL,     // 主要参数：课程源URL
		"question_count": req.QuestionCount, // 用户选择的题目数量
		"difficulty":     req.Difficulty,    // 用户选择的难度
	}

	// 设置默认值（如果参数为空）
	if parameters["question_count"] == 0 {
		parameters["question_count"] = 8
	}
	if parameters["difficulty"] == "" {
		parameters["difficulty"] = "normal"
	}

	log.Printf("工作流参数: %+v", parameters)

	startTime := time.Now()

	// 直接调用练习生成工作流API
	response, err := c.runExerciseWorkflow(ctx, parameters)
	if err != nil {
		return nil, fmt.Errorf("工作流执行失败: %v", err)
	}

	log.Printf("工作流执行响应: Code=%d, Msg=%s", response.Code, response.Msg)

	// 检查响应状态
	if response.Code != 0 {
		return nil, fmt.Errorf("工作流返回错误: %s (code: %d)", response.Msg, response.Code)
	}

	// 检查返回数据类型：执行ID还是直接结果
	log.Printf("🔍 检查工作流响应数据类型: %T", response.Data)
	switch data := response.Data.(type) {
	case string:
		log.Printf("🔍 收到字符串类型响应，长度: %d", len(data))
		// 安全地截取前200字符进行调试
		debugData := data
		if len(data) > 200 {
			debugData = data[:200] + "..."
		}
		log.Printf("🔍 响应前200字符: %s", debugData)

		// 检查字符串内容：是executeID还是JSON结果
		if strings.HasPrefix(data, "{") && strings.Contains(data, "response_type") {
			// 这是JSON格式的完整结果，直接解析
			log.Printf("工作流直接返回JSON结果，开始解析...")

			result := &ExerciseWorkflowResult{
				ExecuteID: "direct_result",
				Status:    "success",
			}

			// 解析JSON字符串
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
				log.Printf("❌ JSON解析失败，数据: %s", data)
				return nil, fmt.Errorf("解析工作流JSON结果失败: %v", err)
			}

			// 直接解析输出数据
			if err := c.parseWorkflowOutput(jsonData, result); err != nil {
				return nil, fmt.Errorf("解析工作流输出失败: %v", err)
			}

			result.ProcessingTime = int(time.Since(startTime).Seconds())
			log.Printf("练习生成完成，耗时: %d秒，题目数: %d", result.ProcessingTime, result.TotalQuestions)
			return result, nil
		} else {
			// 这是executeID，需要轮询状态
			log.Printf("获得工作流执行ID: %s", data)

			// 轮询等待结果
			result, err := c.pollWorkflowResult(ctx, data)
			if err != nil {
				return nil, err
			}

			result.ProcessingTime = int(time.Since(startTime).Seconds())
			log.Printf("练习生成完成，耗时: %d秒", result.ProcessingTime)
			return result, nil
		}

	case map[string]interface{}:
		// 返回的是直接结果，无需轮询
		log.Printf("工作流直接返回结果，开始解析...")

		result := &ExerciseWorkflowResult{
			ExecuteID: "direct_result",
			Status:    "success",
		}

		// 直接解析输出数据
		if err := c.parseWorkflowOutput(data, result); err != nil {
			return nil, fmt.Errorf("解析工作流输出失败: %v", err)
		}

		result.ProcessingTime = int(time.Since(startTime).Seconds())
		log.Printf("练习生成完成，耗时: %d秒，题目数: %d", result.ProcessingTime, result.TotalQuestions)
		return result, nil

	default:
		// 未知的数据类型
		log.Printf("未知的工作流响应数据类型: %T, 内容: %+v", response.Data, response.Data)
		return nil, fmt.Errorf("无法解析工作流响应数据: %v (类型: %T)", response.Data, response.Data)
	}
}

// pollWorkflowResult 轮询工作流结果
func (c *ExerciseWorkflowClient) pollWorkflowResult(ctx context.Context, executeID string) (*ExerciseWorkflowResult, error) {
	log.Printf("开始轮询工作流结果，ExecuteID: %s", executeID)

	deadline := time.Now().Add(c.config.MaxPollTime)
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("上下文取消: %v", ctx.Err())
		case <-time.After(time.Until(deadline)):
			return nil, fmt.Errorf("轮询超时，超过最大等待时间 %v", c.config.MaxPollTime)
		case <-ticker.C:
			result, err := c.checkWorkflowStatus(executeID)
			if err != nil {
				log.Printf("检查工作流状态失败: %v", err)
				continue
			}

			log.Printf("工作流状态: %s", result.Status)

			switch result.Status {
			case "success":
				return result, nil
			case "fail":
				errorMsg := "工作流执行失败"
				if result.ErrorMessage != "" {
					errorMsg = result.ErrorMessage
				}
				return nil, fmt.Errorf(errorMsg)
			case "created", "running":
				// 继续等待
				continue
			default:
				log.Printf("未知的工作流状态: %s", result.Status)
				continue
			}
		}
	}
}

// checkWorkflowStatus 检查工作流状态
func (c *ExerciseWorkflowClient) checkWorkflowStatus(executeID string) (*ExerciseWorkflowResult, error) {
	ctx := context.Background() // 使用默认上下文
	response, err := c.workflowClient.RetrieveWorkflow(ctx, executeID)
	if err != nil {
		return nil, fmt.Errorf("查询工作流状态失败: %v", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("查询工作流返回错误: %s (code: %d)", response.Msg, response.Code)
	}

	result := &ExerciseWorkflowResult{
		ExecuteID: executeID,
		Status:    response.Data.Status,
	}

	// 如果失败，记录错误信息
	if response.Data.Status == "fail" {
		if response.Data.Error.Msg != "" {
			result.ErrorMessage = response.Data.Error.Msg
		} else {
			result.ErrorMessage = "工作流执行失败，无详细错误信息"
		}
		return result, nil
	}

	// 如果成功，解析输出数据
	if response.Data.Status == "success" && response.Data.Output != nil {
		if err := c.parseWorkflowOutput(response.Data.Output, result); err != nil {
			return nil, fmt.Errorf("解析工作流输出失败: %v", err)
		}
	}

	return result, nil
}

// 帮助函数
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getSafeSubstring(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseWorkflowOutput 解析工作流输出
func (c *ExerciseWorkflowClient) parseWorkflowOutput(output map[string]interface{}, result *ExerciseWorkflowResult) error {
	log.Printf("🔍 开始解析工作流输出，输出结构: %+v", output)

	// 检查输出格式
	dataField, exists := output["data"]
	if !exists {
		log.Printf("❌ 工作流输出缺少data字段，可用字段: %v", getMapKeys(output))
		return fmt.Errorf("工作流输出缺少data字段")
	}

	log.Printf("🔍 data字段类型: %T, 内容: %+v", dataField, dataField)

	// 转换为JSON字符串再解析（处理复杂的嵌套结构）
	dataBytes, err := json.Marshal(dataField)
	if err != nil {
		return fmt.Errorf("序列化输出数据失败: %v", err)
	}

	log.Printf("🔍 序列化后的数据长度: %d", len(dataBytes))
	log.Printf("🔍 序列化后的数据前200字符: %s", getSafeSubstring(string(dataBytes), 200))

	// 尝试直接解析为字符串（有些工作流返回JSON字符串）
	var dataStr string
	if err := json.Unmarshal(dataBytes, &dataStr); err == nil {
		log.Printf("🔍 数据是字符串格式，长度: %d", len(dataStr))
		log.Printf("🔍 字符串内容前200字符: %s", getSafeSubstring(dataStr, 200))
		// 如果是字符串，再次解析
		dataBytes = []byte(dataStr)
	} else {
		log.Printf("🔍 数据不是字符串格式，直接使用: %v", err)
	}

	// 解析为工作流响应结构
	var workflowResp struct {
		ResponseType string `json:"response_type"`
		Structure    struct {
			CourseName             string `json:"course_name"`
			TotalQuestions         int    `json:"total_questions"`
			DifficultyDistribution struct {
				Easy   int `json:"easy"`
				Medium int `json:"medium"`
				Hard   int `json:"hard"`
			} `json:"difficulty_distribution"`
			Questions []struct {
				ID            int                    `json:"id"`
				Question      string                 `json:"question"`
				Options       map[string]interface{} `json:"options"`
				CorrectAnswer string                 `json:"correct_answer"`
				Difficulty    string                 `json:"difficulty"`
			} `json:"questions"`
		} `json:"structure"`
	}

	// 清理可能的SSE格式前缀（如 "data: "）
	cleanData := cleanSSEFormat(dataBytes)
	log.Printf("🔍 清理SSE格式后的数据长度: %d", len(cleanData))
	log.Printf("🔍 清理后的数据前200字符: %s", getSafeSubstring(string(cleanData), 200))

	log.Printf("🔍 尝试解析最终数据为工作流响应结构...")
	if err := json.Unmarshal(cleanData, &workflowResp); err != nil {
		// 记录原始数据以便调试
		log.Printf("❌ JSON解析失败！")
		log.Printf("❌ 失败数据长度: %d", len(dataBytes))
		log.Printf("❌ 失败数据内容: %s", string(dataBytes))
		hexLen := 50
		if len(dataBytes) < hexLen {
			hexLen = len(dataBytes)
		}
		log.Printf("❌ 失败数据十六进制: %x", dataBytes[:hexLen])
		log.Printf("❌ 错误详情: %v", err)
		return fmt.Errorf("解析工作流响应结构失败: %v", err)
	}

	// 填充结果
	result.CourseName = workflowResp.Structure.CourseName
	result.TotalQuestions = workflowResp.Structure.TotalQuestions
	result.DifficultyDistribution = DifficultyDist{
		Easy:   workflowResp.Structure.DifficultyDistribution.Easy,
		Medium: workflowResp.Structure.DifficultyDistribution.Medium,
		Hard:   workflowResp.Structure.DifficultyDistribution.Hard,
	}

	// 转换题目
	result.Questions = make([]ExerciseQuestion, len(workflowResp.Structure.Questions))
	for i, q := range workflowResp.Structure.Questions {
		result.Questions[i] = ExerciseQuestion{
			ID:            q.ID,
			Question:      q.Question,
			Options:       q.Options,
			CorrectAnswer: q.CorrectAnswer,
			Difficulty:    q.Difficulty,
		}
	}

	log.Printf("成功解析练习数据: 课程=%s, 题目数=%d", result.CourseName, result.TotalQuestions)
	return nil
}

// GetConfig 获取配置
func (c *ExerciseWorkflowClient) GetConfig() *SingleWorkflowConfig {
	return c.config
}

// GetWorkflowID 获取工作流ID
func (c *ExerciseWorkflowClient) GetWorkflowID() string {
	return c.config.WorkflowID
}

// runExerciseWorkflow 直接调用练习生成工作流API
func (c *ExerciseWorkflowClient) runExerciseWorkflow(ctx context.Context, parameters map[string]interface{}) (*WorkflowRunResponse, error) {
	url := fmt.Sprintf("%s/v1/workflow/run", c.workflowClient.config.APIBase)

	// 构建请求体，确保包含workflow_id
	reqBody := map[string]interface{}{
		"workflow_id": c.config.WorkflowID, // 使用练习生成工作流的ID
		"parameters":  parameters,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.workflowClient.config.Token)

	log.Printf("工作流请求: URL=%s, WorkflowID=%s", url, c.config.WorkflowID)

	// 使用更长的超时时间并实现重试机制
	var resp *http.Response
	var duration time.Duration

	maxRetries := c.config.MaxRetries
	retryDelay := c.config.RetryDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 根据配置设置合理的超时时间 (至少60秒，最多使用配置的TimeoutSeconds)
		timeoutDuration := time.Duration(c.config.TimeoutSeconds) * time.Second
		if timeoutDuration < 60*time.Second {
			timeoutDuration = 60 * time.Second // 最少60秒
		}

		client := &http.Client{Timeout: timeoutDuration}
		startTime := time.Now()

		log.Printf("工作流请求尝试 %d/%d: URL=%s, WorkflowID=%s, Timeout=%v",
			attempt+1, maxRetries+1, url, c.config.WorkflowID, timeoutDuration)

		resp, err = client.Do(httpReq)
		duration = time.Since(startTime)

		if err == nil {
			// 请求成功
			log.Printf("工作流执行请求成功: %s, 状态: %d, 响应时间: %v", url, resp.StatusCode, duration)
			break
		}

		// 请求失败，记录错误并决定是否重试
		log.Printf("工作流请求失败 (尝试 %d/%d): %v, 响应时间: %v", attempt+1, maxRetries+1, err, duration)

		// 如果是最后一次尝试，直接返回错误
		if attempt == maxRetries {
			return nil, fmt.Errorf("工作流请求经过 %d 次重试后仍然失败: %w", maxRetries+1, err)
		}

		// 等待后重试
		log.Printf("等待 %v 后进行重试...", retryDelay)
		time.Sleep(retryDelay)

		// 指数退避：每次重试增加等待时间
		retryDelay = retryDelay * 2
		if retryDelay > 30*time.Second {
			retryDelay = 30 * time.Second // 最大等待30秒
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("工作流请求失败，未获得响应")
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var response WorkflowRunResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		log.Printf("响应解析失败，原始响应: %s", string(respBody))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
}

// cleanSSEFormat 清理Server-Sent Events格式的数据前缀
func cleanSSEFormat(data []byte) []byte {
	dataStr := string(data)

	// 检查是否有 "data: " 前缀
	if strings.HasPrefix(dataStr, "data: ") {
		log.Printf("🔧 检测到SSE格式前缀，正在清理...")
		// 去除 "data: " 前缀
		cleanStr := strings.TrimPrefix(dataStr, "data: ")
		// 去除可能的前后空白字符
		cleanStr = strings.TrimSpace(cleanStr)
		return []byte(cleanStr)
	}

	// 也检查是否有 "data:" 前缀（无空格）
	if strings.HasPrefix(dataStr, "data:") {
		log.Printf("🔧 检测到SSE格式前缀（无空格），正在清理...")
		// 去除 "data:" 前缀
		cleanStr := strings.TrimPrefix(dataStr, "data:")
		// 去除可能的前后空白字符
		cleanStr = strings.TrimSpace(cleanStr)
		return []byte(cleanStr)
	}

	log.Printf("🔧 未检测到SSE前缀，使用原始数据")
	return data
}

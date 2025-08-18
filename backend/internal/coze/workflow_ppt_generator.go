package coze

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// WorkflowPPTGenerator 基于工作流的PPT生成器
type WorkflowPPTGenerator struct {
	client    *WorkflowClient
	processor *PPTProcessor
}

// WorkflowPPTResult 工作流PPT生成结果
type WorkflowPPTResult struct {
	ExecuteID   string                 `json:"execute_id"`
	Status      string                 `json:"status"`
	PPTContent  string                 `json:"ppt_content"`
	PPTLinks    []string               `json:"ppt_links"`
	Title       string                 `json:"title"`
	Output      map[string]interface{} `json:"output"`
	ProcessTime time.Duration          `json:"process_time"`
}

// NewWorkflowPPTGenerator 创建工作流PPT生成器
func NewWorkflowPPTGenerator(config *CozeConfig) *WorkflowPPTGenerator {
	return &WorkflowPPTGenerator{
		client:    NewWorkflowClient(config),
		processor: NewPPTProcessor(config),
	}
}

// GeneratePPTWithWorkflow 使用工作流生成PPT
func (g *WorkflowPPTGenerator) GeneratePPTWithWorkflow(ctx context.Context, url, userID string) (*WorkflowPPTResult, error) {
	startTime := time.Now()

	log.Printf("开始工作流PPT生成: URL=%s, UserID=%s", url, userID)

	// 1. 准备工作流参数
	parameters := map[string]interface{}{
		"keyword": url, // 根据工作流图表，参数名为"keyword"
	}

	// 2. 执行工作流
	runResp, err := g.client.RunWorkflow(ctx, parameters)
	if err != nil {
		return nil, fmt.Errorf("执行工作流失败: %w", err)
	}

	log.Printf("工作流执行结果: %+v", runResp.Data)
	log.Printf("工作流执行结果类型: %T", runResp.Data)

	// 检查响应格式
	var result *WorkflowPPTResult
	switch data := runResp.Data.(type) {
	case string:
		// 检查字符串是否是JSON格式的结果
		if strings.HasPrefix(data, "{") && strings.HasSuffix(data, "}") {
			// 尝试解析为JSON对象
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
				log.Printf("工作流返回JSON字符串结果，解析为对象")
				result, err = g.parseDirectResult(jsonData)
				if err != nil {
					return nil, fmt.Errorf("解析工作流JSON字符串结果失败: %w", err)
				}
			} else {
				// 如果解析失败，当作execute_id处理
				log.Printf("工作流执行ID: %s", data)
				result, err = g.pollWorkflowResult(ctx, data)
				if err != nil {
					return nil, fmt.Errorf("轮询工作流结果失败: %w", err)
				}
			}
		} else {
			// 普通字符串，当作execute_id处理
			log.Printf("工作流执行ID: %s", data)
			result, err = g.pollWorkflowResult(ctx, data)
			if err != nil {
				return nil, fmt.Errorf("轮询工作流结果失败: %w", err)
			}
		}
	case map[string]interface{}:
		// 如果是对象，说明是直接结果，解析它
		log.Printf("工作流直接返回结果对象")
		result, err = g.parseDirectResult(data)
		if err != nil {
			return nil, fmt.Errorf("解析工作流直接结果失败: %w", err)
		}
	default:
		log.Printf("未知的工作流响应格式: %T, 值: %+v", data, data)
		return nil, fmt.Errorf("未知的工作流响应格式: %T", data)
	}

	result.ProcessTime = time.Since(startTime)
	log.Printf("工作流PPT生成完成: 耗时=%v", result.ProcessTime)

	return result, nil
}

// generateUniqueID 生成唯一ID
func generateUniqueID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("workflow_%s_%d", hex.EncodeToString(bytes), time.Now().Unix())
}

// parseDirectResult 解析工作流直接返回的结果
func (g *WorkflowPPTGenerator) parseDirectResult(data map[string]interface{}) (*WorkflowPPTResult, error) {
	// 生成唯一的ExecuteID
	uniqueID := generateUniqueID()

	result := &WorkflowPPTResult{
		ExecuteID:  uniqueID,
		Status:     "success",
		PPTContent: "",
		PPTLinks:   make([]string, 0),
		Title:      "AI生成的PPT", // 使用更通用的默认标题
		Output:     data,
	}

	// 提取PPT链接
	if pptURL, ok := data["ppt_url"].(string); ok && pptURL != "" {
		result.PPTLinks = append(result.PPTLinks, pptURL)
		log.Printf("提取到PPT链接: %s", pptURL)
	}

	// 提取标题
	if title, ok := data["title"].(string); ok && title != "" {
		result.Title = title
	}

	// 提取内容
	if content, ok := data["content"].(string); ok && content != "" {
		result.PPTContent = content
	}

	log.Printf("解析直接结果完成: ExecuteID=%s, 标题=%s, PPT链接数量=%d", result.ExecuteID, result.Title, len(result.PPTLinks))
	return result, nil
}

// pollWorkflowResult 轮询工作流执行结果
func (g *WorkflowPPTGenerator) pollWorkflowResult(ctx context.Context, executeID string) (*WorkflowPPTResult, error) {
	ticker := time.NewTicker(g.client.config.Workflow.PollInterval)
	defer ticker.Stop()

	timeout := time.After(g.client.config.Workflow.MaxPollTime)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("工作流执行超时")
		case <-ticker.C:
			retrieveResp, err := g.client.RetrieveWorkflow(ctx, executeID)
			if err != nil {
				log.Printf("轮询工作流失败: %v", err)
				continue
			}

			log.Printf("工作流状态: %s", retrieveResp.Data.Status)

			switch retrieveResp.Data.Status {
			case "success":
				return g.parseWorkflowOutput(executeID, retrieveResp.Data.Output)
			case "fail":
				return nil, fmt.Errorf("工作流执行失败: %s", retrieveResp.Data.Error.Msg)
			case "created", "running":
				// 继续轮询
				continue
			default:
				log.Printf("未知工作流状态: %s", retrieveResp.Data.Status)
				continue
			}
		}
	}
}

// parseWorkflowOutput 解析工作流输出
func (g *WorkflowPPTGenerator) parseWorkflowOutput(executeID string, output map[string]interface{}) (*WorkflowPPTResult, error) {
	result := &WorkflowPPTResult{
		ExecuteID: executeID,
		Status:    "completed",
		Output:    output,
		PPTLinks:  make([]string, 0),
	}

	// 解析输出内容
	if content, ok := output["content"].(string); ok {
		result.PPTContent = content
	}

	if title, ok := output["title"].(string); ok {
		result.Title = title
	}

	// 解析PPT链接
	if links, ok := output["ppt_links"].([]interface{}); ok {
		for _, link := range links {
			if linkStr, ok := link.(string); ok {
				result.PPTLinks = append(result.PPTLinks, linkStr)
			}
		}
	}

	// 尝试从其他可能的字段名提取内容
	if result.PPTContent == "" {
		if text, ok := output["text"].(string); ok {
			result.PPTContent = text
		} else if response, ok := output["response"].(string); ok {
			result.PPTContent = response
		} else if message, ok := output["message"].(string); ok {
			result.PPTContent = message
		}
	}

	// 如果没有标题，使用默认标题
	if result.Title == "" {
		result.Title = "AI生成的PPT"
	}

	log.Printf("工作流输出解析结果:")
	log.Printf("  - 标题: %s", result.Title)
	log.Printf("  - PPT链接数量: %d", len(result.PPTLinks))
	log.Printf("  - 内容长度: %d字符", len(result.PPTContent))

	// 详细输出调试
	log.Printf("  - 原始输出字段: %+v", output)

	return result, nil
}

// ProcessWorkflowResult 处理工作流结果（将其转换为PPTResult格式以便复用现有逻辑）
func (g *WorkflowPPTGenerator) ProcessWorkflowResult(ctx context.Context, workflowResult *WorkflowPPTResult) (*ProcessingResult, error) {
	// 转换为PPTResult格式以复用现有处理逻辑
	pptResult := &PPTResult{
		ChatID:         workflowResult.ExecuteID,
		ConversationID: workflowResult.ExecuteID,
		Content:        workflowResult.PPTContent,
		PPTLinks:       workflowResult.PPTLinks,
		Title:          workflowResult.Title,
		Status:         workflowResult.Status,
		TokenUsage:     0, // 工作流模式下可能不提供token使用信息
	}

	// 使用现有的PPT处理器
	return g.processor.ProcessCozeResponse(ctx, pptResult)
}

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

// WorkflowClient Coze工作流客户端
type WorkflowClient struct {
	config     *CozeConfig
	httpClient *http.Client
}

// WorkflowRunRequest 工作流运行请求
type WorkflowRunRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Parameters map[string]interface{} `json:"parameters"`
}

// WorkflowRunResponse 工作流运行响应
type WorkflowRunResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"` // 可能是execute_id字符串或结果对象
}

// WorkflowRetrieveResponse 工作流查询响应
type WorkflowRetrieveResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ExecuteID string                 `json:"execute_id"`
		Status    string                 `json:"status"` // created, running, success, fail
		Output    map[string]interface{} `json:"output"`
		Error     struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		} `json:"error"`
	} `json:"data"`
}

// NewWorkflowClient 创建工作流客户端
func NewWorkflowClient(config *CozeConfig) *WorkflowClient {
	return &WorkflowClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// RunWorkflow 执行工作流
func (c *WorkflowClient) RunWorkflow(ctx context.Context, parameters map[string]interface{}) (*WorkflowRunResponse, error) {
	url := fmt.Sprintf("%s/v1/workflow/run", c.config.APIBase)

	// 兼容新旧配置结构获取WorkflowID
	var workflowID string
	if c.config.Workflow != nil {
		// 优先使用新的PPT生成配置
		if c.config.Workflow.PPTGeneration != nil && c.config.Workflow.PPTGeneration.WorkflowID != "" {
			workflowID = c.config.Workflow.PPTGeneration.WorkflowID
		} else if c.config.Workflow.WorkflowID != "" {
			// 使用向后兼容的配置
			workflowID = c.config.Workflow.WorkflowID
		}
	}

	req := &WorkflowRunRequest{
		WorkflowID: workflowID,
		Parameters: parameters,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 调试token信息
	tokenLen := len(c.config.Token)
	var tokenPrefix string
	if tokenLen == 0 {
		tokenPrefix = "<EMPTY>"
	} else if tokenLen < 10 {
		tokenPrefix = c.config.Token + "<TOO_SHORT>"
	} else {
		tokenPrefix = c.config.Token[:10] + "..."
	}
	log.Printf("工作流Token调试: Token长度=%d, Token前缀=%s", tokenLen, tokenPrefix)

	httpReq.Header.Set("Authorization", "Bearer "+c.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("工作流执行请求: %s, 状态: %d, 响应时间: %v", url, resp.StatusCode, responseTime)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var runResp WorkflowRunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if runResp.Code != 0 {
		return nil, fmt.Errorf("工作流执行失败: %s", runResp.Msg)
	}

	return &runResp, nil
}

// RetrieveWorkflow 查询工作流执行结果
func (c *WorkflowClient) RetrieveWorkflow(ctx context.Context, executeID string) (*WorkflowRetrieveResponse, error) {
	url := fmt.Sprintf("%s/v1/workflow/retrieve?execute_id=%s", c.config.APIBase, executeID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var retrieveResp WorkflowRetrieveResponse
	if err := json.Unmarshal(body, &retrieveResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &retrieveResp, nil
}

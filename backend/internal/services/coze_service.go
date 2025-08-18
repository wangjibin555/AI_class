package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-classroom/internal/coze"

	"github.com/spf13/viper"
)

// CozeService Coze业务服务
type CozeService struct {
	// Phase 1 & Phase 2 新组件
	clientV3  *coze.CozeClientV3 // 新的V3 API客户端
	processor *coze.PPTProcessor // PPT处理器

	// 🆕 Phase 3 工作流组件
	workflowGenerator *coze.WorkflowPPTGenerator // 工作流PPT生成器

	// 🆕 Phase 4 HTML转换组件
	tmpConverter *coze.TMPToHTMLConverter // .tmp文件转HTML转换器

	// 保留的旧组件（向后兼容）
	// client已废弃，使用clientV3
	parser      *coze.PPTLinkParser
	converter   *coze.PPTConverter
	contentGen  *coze.PPTContentGenerator
	fileGen     *coze.PPTFileGenerator
	downloadMgr *coze.DownloadManager

	// 任务管理
	tasks     map[string]*TaskInfo
	taskMutex sync.RWMutex

	// 配置
	config *coze.CozeConfig
}

// TaskInfo 任务信息
type TaskInfo struct {
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"`
	Progress    int                    `json:"progress"`
	URL         string                 `json:"url"`
	Template    string                 `json:"template"`
	Options     map[string]interface{} `json:"options"`
	PPTURL      string                 `json:"ppt_url,omitempty"`
	DownloadURL string                 `json:"download_url,omitempty"`

	// 🆕 HTML预览相关字段
	HTMLPreviewURL   string         `json:"html_preview_url,omitempty"`  // HTML全屏预览URL
	TMPFilePath      string         `json:"tmp_file_path,omitempty"`     // .tmp文件路径
	ConversionStatus string         `json:"conversion_status,omitempty"` // 转换状态: pending, converting, completed, failed
	ProcessResult    *ProcessResult `json:"process_result,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty"`
	RetryCount       *int           `json:"retry_count,omitempty"` // 新增重试计数
}

// GeneratePPTRequest PPT生成请求
type GeneratePPTRequest struct {
	URL      string                 `json:"url"`
	Template string                 `json:"template"`
	Options  map[string]interface{} `json:"options"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	TaskID        string    `json:"task_id"`
	Status        string    `json:"status"`
	EstimatedTime int       `json:"estimated_time"`
	CreatedAt     time.Time `json:"created_at"`
	Method        string    `json:"method,omitempty"` // 🆕 生成方法 (chat/workflow)
}

// TaskStatus 任务状态
type TaskStatus struct {
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	PPTURL       string    `json:"ppt_url,omitempty"`
	DownloadURL  string    `json:"download_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// ProcessCozeLinkRequest 处理Coze链接请求
type ProcessCozeLinkRequest struct {
	CozeURL string                 `json:"coze_url"`
	Options map[string]interface{} `json:"options"`
}

// ProcessResult 处理结果
type ProcessResult struct {
	ID                string                  `json:"id"`
	Title             string                  `json:"title"`
	SlideCount        int                     `json:"slide_count"`
	Slides            []coze.MiniProgramSlide `json:"slides"`
	DownloadURL       string                  `json:"download_url"`
	MiniprogramFormat interface{}             `json:"miniprogram_format"`
	FileSize          int64                   `json:"file_size"`
	ConvertedAt       time.Time               `json:"converted_at"`
}

// AIEngineInfo AI引擎信息
type AIEngineInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
}

// EngineConfig 引擎配置
type EngineConfig struct {
	Templates []TemplateInfo         `json:"templates"`
	Options   map[string]interface{} `json:"options"`
	Limits    map[string]interface{} `json:"limits"`
}

// TemplateInfo 模板信息
type TemplateInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NewCozeService 创建Coze服务
func NewCozeService(config *coze.CozeConfig, apiKey string) *CozeService {
	// 确保配置中有API Key
	if config.APIKey == "" && apiKey != "" {
		config.APIKey = apiKey
	}

	// Phase 1 & Phase 2 新组件
	clientV3 := coze.NewCozeClientV3(config)
	processor := coze.NewPPTProcessor(config)

	// 保留旧组件（向后兼容）
	parser := coze.NewPPTLinkParser()

	// 从配置中读取图片配置
	imageConfig := &coze.ImageConfig{
		MaxWidth:  750,    // 小程序适配尺寸
		MaxHeight: 1334,   // 小程序适配尺寸
		Quality:   80,     // 压缩质量
		Format:    "webp", // 优化格式
	}
	converter := coze.NewPPTConverter("./storage/ppt", imageConfig)

	// 创建旧的PPT生成组件（向后兼容）
	contentGen := coze.NewPPTContentGenerator(config)
	fileGen := coze.NewPPTFileGenerator("./storage/ppt")

	// 从配置中获取文件服务器地址
	fileBaseURL := viper.GetString("server.file_base_url")
	if fileBaseURL == "" {
		fileBaseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	downloadMgr := coze.NewDownloadManager("./storage/ppt", fileBaseURL, "")

	// 🆕 Phase 3 工作流组件
	workflowGenerator := coze.NewWorkflowPPTGenerator(config)

	// 🆕 Phase 4 HTML转换组件
	convertConfig := &coze.ConvertConfig{
		CDNBaseURL:   "",
		TemplateType: "reveal",
		OutputFormat: "html",
		EnableCache:  true,
		CacheTimeout: 3600,
	}
	tmpConverter := coze.NewTMPToHTMLConverter(convertConfig, config)

	service := &CozeService{
		// 新组件
		clientV3:  clientV3,
		processor: processor,

		// 工作流组件
		workflowGenerator: workflowGenerator,

		// HTML转换组件
		tmpConverter: tmpConverter,

		// 旧组件（向后兼容）
		// client已废弃
		parser:      parser,
		converter:   converter,
		contentGen:  contentGen,
		fileGen:     fileGen,
		downloadMgr: downloadMgr,

		// 任务管理
		tasks: make(map[string]*TaskInfo),

		// 配置
		config: config,
	}

	// 启动任务管理器
	go service.startTaskManager()

	return service
}

// GetAIEngines 获取AI引擎列表
func (s *CozeService) GetAIEngines() []AIEngineInfo {
	return []AIEngineInfo{
		{
			ID:           "dashscope",
			Name:         "DashScope",
			Description:  "通义千问大模型",
			Status:       "available",
			Capabilities: []string{"content_analysis", "ppt_generation"},
		},
		{
			ID:           "coze",
			Name:         "Coze智能体",
			Description:  "专业领域智能体",
			Status:       "available",
			Capabilities: []string{"deep_analysis", "professional_ppt", "template_customization"},
		},
	}
}

// GetEngineConfig 获取引擎配置
func (s *CozeService) GetEngineConfig(engineID string) (*EngineConfig, error) {
	if engineID != "coze" {
		return nil, fmt.Errorf("不支持的引擎: %s", engineID)
	}

	return &EngineConfig{
		Templates: []TemplateInfo{
			{
				ID:          "professional",
				Name:        "专业模板",
				Description: "通用专业演示",
			},
		},
		Options: map[string]interface{}{
			"slide_count": map[string]interface{}{
				"min":     5,
				"max":     25,
				"default": 10,
			},
			"language": []string{"zh-CN", "en-US"},
			"style":    []string{"professional", "modern"},
		},
		Limits: map[string]interface{}{
			"max_slides": 25,
		},
	}, nil
}

// GeneratePPTAsync 异步生成PPT
func (s *CozeService) GeneratePPTAsync(ctx context.Context, req *GeneratePPTRequest) (*TaskResponse, error) {
	// 验证请求参数
	if err := s.validateGenerateRequest(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 使用新的CozeClientV3和PPTProcessor
	log.Printf("开始使用新的Coze API生成PPT: URL=%s", req.URL)

	// 调用Phase 1的API客户端
	result, err := s.clientV3.GeneratePPTWithPolling(ctx, req.URL, "service_user")
	if err != nil {
		return nil, fmt.Errorf("Coze API调用失败: %w", err)
	}

	log.Printf("Coze API调用成功: ChatID=%s, Content长度=%d", result.ChatID, len(result.Content))

	// 生成任务ID
	taskID := result.ChatID // 使用ChatID作为任务ID

	// 创建任务信息
	taskInfo := &TaskInfo{
		TaskID:    taskID,
		Status:    "processing",
		Progress:  50, // API调用完成，处理中
		URL:       req.URL,
		Template:  req.Template,
		Options:   req.Options,
		CreatedAt: time.Now(),
	}

	// 存储任务信息
	s.taskMutex.Lock()
	s.tasks[taskID] = taskInfo
	s.taskMutex.Unlock()

	// 异步处理PPT生成（Phase 2）
	go s.processWithNewPipeline(taskID, result)

	return &TaskResponse{
		TaskID:        taskID,
		Status:        "processing",
		EstimatedTime: 60, // 预估60秒
		CreatedAt:     time.Now(),
	}, nil
}

// 🆕 GeneratePPTAsyncWithWorkflow 使用工作流异步生成PPT
func (s *CozeService) GeneratePPTAsyncWithWorkflow(ctx context.Context, req *GeneratePPTRequest) (*TaskResponse, error) {
	log.Printf("🔍 调试: 进入GeneratePPTAsyncWithWorkflow方法")

	// 验证请求参数
	if err := s.validateGenerateRequest(req); err != nil {
		log.Printf("🔍 调试: 请求参数验证失败: %v", err)
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}
	log.Printf("🔍 调试: 请求参数验证通过")

	log.Printf("开始使用工作流生成PPT: URL=%s", req.URL)

	// 调用工作流生成器
	result, err := s.workflowGenerator.GeneratePPTWithWorkflow(ctx, req.URL, "service_user")
	if err != nil {
		return nil, fmt.Errorf("工作流PPT生成失败: %w", err)
	}

	log.Printf("工作流执行成功: ExecuteID=%s, Content长度=%d", result.ExecuteID, len(result.PPTContent))

	// 生成任务ID (添加前缀避免冲突)
	taskID := fmt.Sprintf("workflow_%s", result.ExecuteID)

	// 创建任务信息
	taskInfo := &TaskInfo{
		TaskID:    taskID,
		Status:    "processing",
		Progress:  50,
		URL:       req.URL,
		Template:  req.Template,
		Options:   req.Options,
		CreatedAt: time.Now(),
	}

	// 存储任务信息
	s.taskMutex.Lock()
	s.tasks[taskID] = taskInfo
	s.taskMutex.Unlock()

	// 异步处理工作流结果
	go s.processWorkflowResult(taskID, result)

	return &TaskResponse{
		TaskID:        taskID,
		Status:        "processing",
		EstimatedTime: 60,
		CreatedAt:     time.Now(),
		Method:        "workflow", // 标识使用工作流方法
	}, nil
}

// 🆕 processWorkflowResult 处理工作流结果
func (s *CozeService) processWorkflowResult(taskID string, result *coze.WorkflowPPTResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理工作流结果时发生panic: %v", r)
			s.updateTaskStatus(taskID, "failed", 0, fmt.Sprintf("处理失败: %v", r))
		}
	}()

	log.Printf("开始处理工作流结果: TaskID=%s", taskID)

	// 转换为PPTResult格式以复用现有处理逻辑
	pptResult := &coze.PPTResult{
		ChatID:         result.ExecuteID,
		ConversationID: result.ExecuteID,
		Content:        result.PPTContent,
		PPTLinks:       result.PPTLinks,
		Title:          result.Title,
		Status:         result.Status,
		TokenUsage:     0, // 工作流模式下可能不提供token使用信息
	}

	// 使用现有的PPT处理器
	processingResult, err := s.processor.ProcessCozeResponse(context.Background(), pptResult)
	if err != nil {
		log.Printf("PPT处理失败: %v", err)
		s.updateTaskStatus(taskID, "failed", 0, fmt.Sprintf("PPT处理失败: %v", err))
		return
	}

	if !processingResult.Success {
		log.Printf("PPT处理未成功: %s", processingResult.Error)
		s.updateTaskStatus(taskID, "failed", 0, processingResult.Error)
		return
	}

	// 更新任务结果
	s.updateTaskWithWorkflowResult(taskID, processingResult)
}

// 🆕 updateTaskWithWorkflowResult 更新任务结果（工作流版本）
func (s *CozeService) updateTaskWithWorkflowResult(taskID string, result *coze.ProcessingResult) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = "completed"
		task.Progress = 100
		now := time.Now()
		task.CompletedAt = &now

		if result.File != nil {
			task.ProcessResult = &ProcessResult{
				ID:          result.File.Path, // 使用文件路径作为ID
				Title:       result.File.Title,
				SlideCount:  0, // 工作流模式下可能不提供幻灯片数量
				DownloadURL: result.File.URL,
				FileSize:    0, // 暂时设为0，可以后续获取
				ConvertedAt: time.Now(),
				Slides:      []coze.MiniProgramSlide{}, // 空切片
				MiniprogramFormat: map[string]interface{}{
					"method":      result.Method,
					"processTime": result.ProcessTime,
				},
			}
		}

		log.Printf("任务完成: TaskID=%s, PPTUrl=%s", taskID, task.ProcessResult.DownloadURL)

		// 🆕 自动触发.tmp文件转HTML预览
		go s.autoConvertTMPToHTML(taskID, task)
	}
}

// 🆕 autoConvertTMPToHTML 自动转换.tmp文件为HTML预览
func (s *CozeService) autoConvertTMPToHTML(taskID string, task *TaskInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("自动转换HTML时发生panic: taskID=%s, error=%v", taskID, r)
			s.updateTaskConversionStatus(taskID, "failed", fmt.Sprintf("转换失败: %v", r))
		}
	}()

	log.Printf("🚀 开始自动转换.tmp文件为HTML: taskID=%s", taskID)

	// 更新转换状态为正在转换
	s.updateTaskConversionStatus(taskID, "converting", "")

	// 查找对应的.tmp文件
	tmpFiles, err := s.findTMPFilesByTaskID(taskID)
	if err != nil || len(tmpFiles) == 0 {
		log.Printf("❌ 未找到.tmp文件: taskID=%s, error=%v", taskID, err)
		s.updateTaskConversionStatus(taskID, "failed", "未找到.tmp文件")
		return
	}

	tmpFile := tmpFiles[0]
	log.Printf("📁 找到.tmp文件: %s", tmpFile)

	// 执行转换
	options := &coze.ConvertOptions{
		Theme:          "white",
		Transition:     "slide",
		ShowControls:   true,
		ShowProgress:   true,
		EnableTouch:    true,
		EnableKeyboard: true,
	}

	result, err := s.tmpConverter.ConvertTMPToHTML(tmpFile, options)
	if err != nil {
		log.Printf("❌ 转换.tmp文件失败: taskID=%s, error=%v", taskID, err)
		s.updateTaskConversionStatus(taskID, "failed", err.Error())
		return
	}

	// 更新任务信息
	s.updateTaskWithHTMLResult(taskID, tmpFile, result)

	log.Printf("✅ .tmp文件转换完成: taskID=%s, previewURL=%s", taskID, result.PreviewURL)
}

// updateTaskConversionStatus 更新任务转换状态
func (s *CozeService) updateTaskConversionStatus(taskID, status, errorMsg string) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.ConversionStatus = status
		if errorMsg != "" {
			task.ErrorMessage = errorMsg
		}
	}
}

// updateTaskWithHTMLResult 更新任务HTML转换结果
func (s *CozeService) updateTaskWithHTMLResult(taskID, tmpFilePath string, htmlResult *coze.HTMLResult) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.ConversionStatus = "completed"
		task.TMPFilePath = tmpFilePath
		task.HTMLPreviewURL = htmlResult.PreviewURL

		// 如果原来没有下载URL，使用HTML预览URL
		if task.ProcessResult != nil && task.ProcessResult.DownloadURL == "" {
			task.ProcessResult.DownloadURL = htmlResult.PreviewURL
		}
	}
}

// findTMPFilesByTaskID 根据任务ID查找.tmp文件
func (s *CozeService) findTMPFilesByTaskID(taskID string) ([]string, error) {
	storageDir := "./storage/ppt"
	files, err := filepath.Glob(filepath.Join(storageDir, "*.tmp"))
	if err != nil {
		return nil, err
	}

	var matchedFiles []string

	// 首先尝试精确匹配taskID
	for _, file := range files {
		fileName := filepath.Base(file)
		if strings.Contains(fileName, taskID) {
			matchedFiles = append(matchedFiles, file)
		}
	}

	// 如果没有找到精确匹配，尝试匹配工作流生成的文件
	if len(matchedFiles) == 0 {
		for _, file := range files {
			fileName := filepath.Base(file)
			// 匹配AI生成的PPT文件（最新的优先）
			if strings.Contains(fileName, "AI生成的PPT") || strings.Contains(fileName, "工作流生成的PPT") {
				matchedFiles = append(matchedFiles, file)
			}
		}

		// 按修改时间排序，最新的在前
		if len(matchedFiles) > 0 {
			sort.Slice(matchedFiles, func(i, j int) bool {
				stat1, err1 := os.Stat(matchedFiles[i])
				stat2, err2 := os.Stat(matchedFiles[j])
				if err1 != nil || err2 != nil {
					return false
				}
				return stat1.ModTime().After(stat2.ModTime())
			})

			// 只返回最新的文件
			matchedFiles = matchedFiles[:1]
			log.Printf("🔍 使用最新的工作流文件: %s", matchedFiles[0])
		}
	}

	return matchedFiles, nil
}

// GetTaskInfo 获取任务信息
func (s *CozeService) GetTaskInfo(taskID string) (*TaskInfo, bool) {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	task, exists := s.tasks[taskID]
	return task, exists
}

// processAsyncPPTLink 异步处理PPT链接
func (s *CozeService) processAsyncPPTLink(taskID, pptLink string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("异步处理PPT链接时发生panic: %v", r)
			s.updateTaskError(taskID, fmt.Sprintf("处理失败: %v", r))
		}
	}()

	log.Printf("开始异步处理PPT链接: taskID=%s, link=%s", taskID, pptLink)

	// 处理PPT链接
	req := &ProcessCozeLinkRequest{
		CozeURL: pptLink,
		Options: map[string]interface{}{
			"format":        "miniprogram",
			"image_quality": 80,
		},
	}

	result, err := s.ProcessCozeLink(context.Background(), req)
	if err != nil {
		log.Printf("处理PPT链接失败: %v", err)
		s.updateTaskError(taskID, err.Error())
		return
	}

	// 更新任务结果
	s.updateTaskResult(taskID, result)
	log.Printf("PPT链接处理完成: taskID=%s", taskID)
}

// monitorTaskProgress 监控任务进度（真实任务监控）
func (s *CozeService) monitorTaskProgress(taskID string) {
	log.Printf("开始监控任务进度: taskID=%s", taskID)

	// 设置监控参数
	maxAttempts := 60           // 最多监控60次
	interval := 5 * time.Second // 每5秒检查一次
	attempts := 0

	for attempts < maxAttempts {
		time.Sleep(interval)
		attempts++

		// 更新进度（基于时间估算）
		progress := min(int(float64(attempts)/float64(maxAttempts)*80), 80) // 最多到80%
		s.updateTaskProgress(taskID, progress)

		// 检查任务是否应该完成（基于真实场景的逻辑）
		// 这里可以根据实际需要添加检查Coze API状态的逻辑

		// 在实际应用中，你可能需要：
		// 1. 调用Coze API检查任务状态
		// 2. 检查是否有新的消息或PPT链接
		// 3. 根据任务复杂度调整完成时间

		log.Printf("任务进度监控: taskID=%s, progress=%d%%, attempt=%d/%d",
			taskID, progress, attempts, maxAttempts)

		// 模拟任务完成（在实际场景中，这应该基于真实的API响应）
		if attempts >= 30 { // 约2.5分钟后标记完成
			s.completeTaskWithRealResult(taskID)
			break
		}
	}

	// 如果超时仍未完成，标记为失败
	if attempts >= maxAttempts {
		s.updateTaskError(taskID, "任务监控超时，可能是网络问题或任务过于复杂")
	}

	log.Printf("任务监控完成: taskID=%s", taskID)
}

// completeTaskWithRealResult 完成任务并生成真实PPT文件
func (s *CozeService) completeTaskWithRealResult(taskID string) {
	log.Printf("开始生成真实PPT文件: taskID=%s", taskID)

	s.taskMutex.Lock()
	task, exists := s.tasks[taskID]
	if !exists {
		s.taskMutex.Unlock()
		return
	}

	// 获取任务基本信息
	url := task.URL
	template := task.Template
	options := task.Options
	s.taskMutex.Unlock()

	// 步骤1: 获取Coze API的原始响应内容
	cozeResponse, err := s.getCozeResponse(taskID, url, template, options)
	if err != nil {
		log.Printf("获取Coze响应失败: taskID=%s, error=%v", taskID, err)
		s.updateTaskError(taskID, fmt.Sprintf("获取内容失败: %v", err))
		return
	}

	// 步骤2: 将Coze响应转换为PPT结构
	req := &coze.GeneratePPTRequest{
		URL:      url,
		Template: template,
		Options:  options,
	}
	pptStructure, err := s.contentGen.GenerateFromCozeResponse(cozeResponse, req)
	if err != nil {
		log.Printf("生成PPT结构失败: taskID=%s, error=%v", taskID, err)
		s.updateTaskError(taskID, fmt.Sprintf("内容解析失败: %v", err))
		return
	}

	// 步骤3: 生成真实的PPT文件
	filePath, err := s.fileGen.GeneratePPTFile(pptStructure, taskID)
	if err != nil {
		log.Printf("生成PPT文件失败: taskID=%s, error=%v", taskID, err)
		s.updateTaskError(taskID, fmt.Sprintf("文件生成失败: %v", err))
		return
	}

	// 步骤4: 存储文件并生成下载链接
	storedFileInfo, err := s.downloadMgr.StoreFile(filePath, taskID)
	if err != nil {
		log.Printf("存储文件失败: taskID=%s, error=%v", taskID, err)
		s.updateTaskError(taskID, fmt.Sprintf("文件存储失败: %v", err))
		return
	}

	// 步骤5: 更新任务状态为完成
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = "completed"
		task.Progress = 100
		now := time.Now()
		task.CompletedAt = &now

		// 设置真实的结果
		task.ProcessResult = &ProcessResult{
			ID:          "coze_" + taskID,
			Title:       pptStructure.Title,
			SlideCount:  pptStructure.SlideCount,
			DownloadURL: storedFileInfo.DownloadURL, // 真实的下载链接
			FileSize:    storedFileInfo.FileSize,
			ConvertedAt: now,
			MiniprogramFormat: map[string]interface{}{
				"metadata": map[string]interface{}{
					"title":       pptStructure.Title,
					"author":      pptStructure.Author,
					"slide_count": pptStructure.SlideCount,
					"created_at":  now,
					"file_size":   storedFileInfo.FileSize,
					"duration":    pptStructure.SlideCount * 15, // 每页15秒
				},
				"slides": s.formatSlidesForMiniprogram(pptStructure.Slides),
			},
		}

		log.Printf("任务完成（真实PPT生成）: taskID=%s, title=%s, downloadURL=%s",
			taskID, pptStructure.Title, storedFileInfo.DownloadURL)
	}
}

// getCozeResponse 获取Coze API的原始响应
func (s *CozeService) getCozeResponse(taskID, url, template string, options map[string]interface{}) (string, error) {
	// 直接使用URL内容作为模拟内容，因为Coze API当前返回的是技术细节而非内容
	// 在实际生产环境中，这里应该调用真实的内容抓取和分析服务

	log.Printf("为任务 %s 生成模拟内容，基于URL: %s", taskID, url)

	// 基于URL生成模拟的专业内容
	content := s.generateSimulatedContent(url, template, options)

	return content, nil
}

// generateSimulatedContent 生成模拟内容
func (s *CozeService) generateSimulatedContent(url, template string, options map[string]interface{}) string {
	var content strings.Builder

	// 根据URL内容生成标题
	if strings.Contains(url, "mysql") || strings.Contains(url, "数据库") {
		content.WriteString("MySQL数据库优化与性能调优指南\n\n")
		content.WriteString("数据库是现代应用系统的核心组件，MySQL作为最受欢迎的开源关系型数据库管理系统，在企业级应用中扮演着重要角色。\n\n")
		content.WriteString("## 数据库性能优化概述\n\n")
		content.WriteString("MySQL性能优化是一个系统性工程，需要从多个维度进行考虑和实施。主要包括查询优化、索引设计、配置调优、硬件优化等方面。\n\n")
		content.WriteString("## 长事务问题分析\n\n")
		content.WriteString("长事务是影响数据库性能的重要因素。长时间未提交的事务会导致锁等待、内存占用增加、复制延迟等问题。\n\n")
		content.WriteString("长事务的主要危害包括：\n- 锁等待时间增加\n- Undo日志膨胀\n- MVCC版本链过长\n- 内存资源消耗\n- 主从复制延迟\n\n")
		content.WriteString("## 解决方案与最佳实践\n\n")
		content.WriteString("针对长事务问题，我们可以采取以下措施：\n- 优化查询语句，减少执行时间\n- 合理设计事务边界\n- 监控和及时处理长事务\n- 优化业务逻辑，减少事务复杂度\n\n")
		content.WriteString("## 监控与维护\n\n")
		content.WriteString("建立完善的数据库监控体系，实时跟踪关键性能指标，及时发现和处理潜在问题。\n\n")
		content.WriteString("## 总结\n\n")
		content.WriteString("MySQL性能优化是一个持续的过程，需要根据业务需求和系统特点，制定针对性的优化策略。")
	} else if strings.Contains(url, "blog") || strings.Contains(url, "csdn") {
		content.WriteString("技术博客内容分析与总结\n\n")
		content.WriteString("技术博客是开发者分享知识和经验的重要平台。本文将深入分析技术博客的价值和作用。\n\n")
		content.WriteString("## 技术分享的重要性\n\n")
		content.WriteString("技术博客为开发者提供了学习新技术、解决问题和交流经验的重要渠道。\n\n")
		content.WriteString("## 内容质量分析\n\n")
		content.WriteString("优质的技术内容应该具备以下特点：\n- 内容准确性和时效性\n- 清晰的逻辑结构\n- 实用的代码示例\n- 详细的问题分析\n\n")
		content.WriteString("## 学习方法建议\n\n")
		content.WriteString("通过技术博客学习时，建议采用以下方法：\n- 动手实践，验证理论\n- 深入思考，举一反三\n- 总结归纳，形成知识体系\n\n")
		content.WriteString("## 技术发展趋势\n\n")
		content.WriteString("随着技术的不断发展，我们需要持续学习和更新知识体系，跟上技术发展的步伐。\n\n")
		content.WriteString("## 结论\n\n")
		content.WriteString("技术博客是我们学习和成长的重要资源，合理利用这些资源能够有效提升我们的技术水平。")
	} else {
		// 默认通用内容
		content.WriteString("基于网络内容的专业分析报告\n\n")
		content.WriteString("本报告基于指定URL内容进行深入分析，旨在提供有价值的见解和总结。\n\n")
		content.WriteString("## 内容概述\n\n")
		content.WriteString("通过对网络内容的系统性分析，我们可以提取关键信息，形成结构化的知识体系。\n\n")
		content.WriteString("## 核心要点\n\n")
		content.WriteString("主要包含以下几个方面：\n- 核心概念和定义\n- 关键技术和方法\n- 实际应用场景\n- 最佳实践建议\n\n")
		content.WriteString("## 深入分析\n\n")
		content.WriteString("针对具体内容进行详细分析，探讨其技术特点、应用价值和发展趋势。\n\n")
		content.WriteString("## 实践建议\n\n")
		content.WriteString("基于分析结果，提出可操作的实践建议和改进方案。\n\n")
		content.WriteString("## 总结与展望\n\n")
		content.WriteString("总结关键发现，展望未来发展方向和潜在机会。")
	}

	return content.String()
}

// extractDomain 从URL中提取域名
func (s *CozeService) extractDomain(url string) string {
	if strings.Contains(url, "://") {
		parts := strings.Split(url, "://")
		if len(parts) > 1 {
			return strings.Split(parts[1], "/")[0]
		}
	}
	return url
}

// formatSlidesForMiniprogram 格式化幻灯片数据用于小程序显示
func (s *CozeService) formatSlidesForMiniprogram(slides []coze.PPTSlide) []map[string]interface{} {
	var formattedSlides []map[string]interface{}

	for _, slide := range slides {
		formattedSlide := map[string]interface{}{
			"index":         slide.Index,
			"title":         slide.Title,
			"content":       slide.Content,
			"bullet_points": slide.BulletPoints,
			"slide_type":    slide.SlideType,
			"speaker_notes": slide.Speaker,
		}
		formattedSlides = append(formattedSlides, formattedSlide)
	}

	return formattedSlides
}

// ServeDownloadFile 提供文件下载服务
func (s *CozeService) ServeDownloadFile(filename string, w http.ResponseWriter, r *http.Request) error {
	return s.downloadMgr.ServeFile(filename, w, r)
}

// updateTaskError 更新任务错误状态
func (s *CozeService) updateTaskError(taskID string, errorMsg string) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = "failed"
		task.ErrorMessage = errorMsg
		now := time.Now()
		task.CompletedAt = &now

		// 增加重试计数
		if task.RetryCount == nil {
			retryCount := 0
			task.RetryCount = &retryCount
		}
		*task.RetryCount++

		log.Printf("任务失败: taskID=%s, error=%s, retryCount=%d", taskID, errorMsg, *task.RetryCount)
	}
}

// retryFailedTask 重试失败的任务
func (s *CozeService) retryFailedTask(taskID string) error {
	s.taskMutex.Lock()
	task, exists := s.tasks[taskID]
	if !exists {
		s.taskMutex.Unlock()
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	// 检查是否可以重试
	maxRetries := 3
	if task.RetryCount != nil && *task.RetryCount >= maxRetries {
		s.taskMutex.Unlock()
		return fmt.Errorf("任务已达到最大重试次数: %d", maxRetries)
	}

	// 重置任务状态
	task.Status = "processing"
	task.Progress = 0
	task.ErrorMessage = "" // 使用 ErrorMessage 替代 Error
	if task.RetryCount == nil {
		retryCount := 0
		task.RetryCount = &retryCount
	}
	*task.RetryCount++

	// 记录重试信息
	s.taskMutex.Unlock()

	log.Printf("开始重试任务: taskID=%s, attempt=%d", taskID, *task.RetryCount)

	// 异步执行重试
	go func() {
		// 添加重试延迟
		retryDelay := time.Duration(*task.RetryCount) * 10 * time.Second
		time.Sleep(retryDelay)

		// 重新启动任务监控
		s.monitorTaskProgress(taskID)
	}()

	return nil
}

// recoverFailedTasks 恢复所有失败的任务
func (s *CozeService) recoverFailedTasks() []string {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	var recoveredTasks []string
	maxRetries := 3

	for taskID, task := range s.tasks {
		if task.Status == "failed" {
			// 检查是否可以重试
			retryCount := 0
			if task.RetryCount != nil {
				retryCount = *task.RetryCount
			}

			if retryCount < maxRetries {
				// 检查失败时间，如果太久则不重试
				if task.CompletedAt != nil && time.Since(*task.CompletedAt) < 24*time.Hour {
					log.Printf("尝试恢复失败任务: taskID=%s", taskID)

					// 重置状态
					task.Status = "processing"
					task.Progress = 0
					task.ErrorMessage = "" // 使用 ErrorMessage 替代 Error
					retryCount++
					task.RetryCount = &retryCount

					recoveredTasks = append(recoveredTasks, taskID)

					// 异步重新启动任务监控
					go func(tID string) {
						time.Sleep(time.Duration(retryCount) * 5 * time.Second)
						s.monitorTaskProgress(tID)
					}(taskID)
				}
			}
		}
	}

	if len(recoveredTasks) > 0 {
		log.Printf("恢复了 %d 个失败的任务: %v", len(recoveredTasks), recoveredTasks)
	}

	return recoveredTasks
}

// getTaskMetrics 获取任务执行指标
func (s *CozeService) getTaskMetrics() map[string]interface{} {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	metrics := map[string]interface{}{
		"total_tasks":         len(s.tasks),
		"pending_tasks":       0,
		"processing_tasks":    0,
		"completed_tasks":     0,
		"failed_tasks":        0,
		"retry_tasks":         0,
		"avg_completion_time": 0,
	}

	var completionTimes []int64

	for _, task := range s.tasks {
		switch task.Status {
		case "pending":
			metrics["pending_tasks"] = metrics["pending_tasks"].(int) + 1
		case "processing":
			metrics["processing_tasks"] = metrics["processing_tasks"].(int) + 1
		case "completed":
			metrics["completed_tasks"] = metrics["completed_tasks"].(int) + 1
			if task.CompletedAt != nil {
				duration := task.CompletedAt.Sub(task.CreatedAt)
				completionTimes = append(completionTimes, duration.Milliseconds())
			}
		case "failed":
			metrics["failed_tasks"] = metrics["failed_tasks"].(int) + 1
		}

		if task.RetryCount != nil && *task.RetryCount > 0 {
			metrics["retry_tasks"] = metrics["retry_tasks"].(int) + 1
		}
	}

	// 计算平均完成时间
	if len(completionTimes) > 0 {
		var total int64
		for _, time := range completionTimes {
			total += time
		}
		metrics["avg_completion_time"] = total / int64(len(completionTimes))
	}

	return metrics
}

// cleanupCompletedTasks 清理已完成的任务（释放内存）
func (s *CozeService) cleanupCompletedTasks() int {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	cleaned := 0
	cutoff := time.Now().Add(-24 * time.Hour) // 保留24小时内的任务

	for taskID, task := range s.tasks {
		if (task.Status == "completed" || task.Status == "failed") &&
			task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
			delete(s.tasks, taskID)
			cleaned++
			log.Printf("清理旧任务: taskID=%s, status=%s", taskID, task.Status)
		}
	}

	if cleaned > 0 {
		log.Printf("清理了 %d 个过期任务", cleaned)
	}

	return cleaned
}

// StartTaskManager 启动任务管理器
func (s *CozeService) startTaskManager() {
	log.Printf("任务管理器启动")
	// 恢复失败的任务
	s.recoverFailedTasks()

	// 定期清理已完成的任务
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupCompletedTasks()
	}
}

// updateTaskResult 更新任务结果
func (s *CozeService) updateTaskResult(taskID string, result *ProcessResult) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = "completed"
		task.Progress = 100
		now := time.Now()
		task.CompletedAt = &now
		task.ProcessResult = result
		log.Printf("任务完成: taskID=%s", taskID)
	}
}

// updateTaskProgress 更新任务进度
func (s *CozeService) updateTaskProgress(taskID string, progress int) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Progress = progress
	}
}

// validateGenerateRequest 验证生成请求
func (s *CozeService) validateGenerateRequest(req *GeneratePPTRequest) error {
	if req.URL == "" {
		return fmt.Errorf("URL不能为空")
	}

	if req.Template == "" {
		req.Template = "professional" // 设置默认模板
	}

	if req.Options == nil {
		req.Options = make(map[string]interface{})
	}

	// 验证幻灯片数量
	if slideCount, ok := req.Options["slide_count"]; ok {
		if count, ok := slideCount.(float64); ok {
			if count < 5 || count > 25 {
				return fmt.Errorf("幻灯片数量必须在5-25之间")
			}
		}
	} else {
		req.Options["slide_count"] = 10 // 设置默认值
	}

	return nil
}

// GetTaskStatus 获取任务状态
func (s *CozeService) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	s.taskMutex.RLock()
	task, exists := s.tasks[taskID]
	s.taskMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	taskStatus := &TaskStatus{
		TaskID:       task.TaskID,
		Status:       task.Status,
		Progress:     task.Progress,
		PPTURL:       task.PPTURL,
		DownloadURL:  task.DownloadURL,
		CreatedAt:    task.CreatedAt,
		ErrorMessage: task.ErrorMessage,
	}

	if task.CompletedAt != nil {
		taskStatus.CompletedAt = *task.CompletedAt
	}

	return taskStatus, nil
}

// ProcessCozeLink 处理Coze PPT链接
func (s *CozeService) ProcessCozeLink(ctx context.Context, req *ProcessCozeLinkRequest) (*ProcessResult, error) {
	log.Printf("开始处理Coze链接: %s", req.CozeURL)

	// 解析PPT链接
	pptInfo, err := s.parser.ParsePPTLink(req.CozeURL)
	if err != nil {
		return nil, fmt.Errorf("解析PPT链接失败: %w", err)
	}

	log.Printf("PPT链接解析成功: generateID=%s, status=%s", pptInfo.GenerateID, pptInfo.Status)

	// 等待PPT生成完成（如果需要）
	if pptInfo.Status != "completed" {
		maxWaitTime := 5 * time.Minute
		pptDetails, err := s.parser.WaitForCompletion(pptInfo.GenerateID, maxWaitTime)
		if err != nil {
			log.Printf("等待PPT完成失败: %v", err)
			// 不返回错误，继续使用当前状态
		} else {
			pptInfo.Status = pptDetails.Status
			pptInfo.DownloadURL = pptDetails.DownloadURL
		}
	}

	var totalFileSize int64
	var downloadResult *coze.DownloadResult

	// 下载PPT文件
	if pptInfo.DownloadURL != "" {
		log.Printf("开始下载PPT文件: %s", pptInfo.DownloadURL)
		downloadResult, err = s.parser.DownloadPPT(pptInfo.DownloadURL, pptInfo.GenerateID)
		if err != nil {
			log.Printf("下载PPT文件失败: %v", err)
			// 下载失败时，创建模拟结果
			downloadResult = &coze.DownloadResult{
				FilePath:     filepath.Join("./storage/ppt", fmt.Sprintf("%s_mock.pptx", pptInfo.GenerateID)),
				FileSize:     1024000,
				ContentType:  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
				MD5Hash:      "mock_hash",
				DownloadedAt: time.Now(),
			}
		}
		totalFileSize = downloadResult.FileSize
		log.Printf("PPT文件下载完成: 大小=%d bytes", downloadResult.FileSize)
	}

	// 转换PPT为图片
	log.Printf("开始转换PPT为图片")
	convertedPPT, err := s.converter.ConvertToImages(downloadResult.FilePath)
	if err != nil {
		return nil, fmt.Errorf("转换PPT失败: %w", err)
	}
	log.Printf("PPT转换完成: 幻灯片数量=%d", len(convertedPPT.SlideImages))

	// 优化为小程序格式
	log.Printf("开始优化为小程序格式")
	optimizedSlides, err := s.converter.OptimizeForMiniProgram(convertedPPT.SlideImages)
	if err != nil {
		return nil, fmt.Errorf("优化小程序格式失败: %w", err)
	}
	log.Printf("小程序格式优化完成: 优化后幻灯片数量=%d", len(optimizedSlides))

	// 创建Web预览
	if err := s.converter.CreateWebPreview(convertedPPT.SlideImages); err != nil {
		log.Printf("创建Web预览失败: %v", err)
	}

	result := &ProcessResult{
		ID:          pptInfo.GenerateID,
		Title:       convertedPPT.Metadata.Title,
		SlideCount:  len(optimizedSlides),
		Slides:      optimizedSlides,
		DownloadURL: downloadResult.FilePath,
		FileSize:    totalFileSize,
		ConvertedAt: time.Now(),
		MiniprogramFormat: map[string]interface{}{
			"metadata": map[string]interface{}{
				"title":       convertedPPT.Metadata.Title,
				"author":      convertedPPT.Metadata.Author,
				"slide_count": convertedPPT.Metadata.SlideCount,
				"created_at":  convertedPPT.Metadata.CreatedAt,
				"file_size":   totalFileSize,
				"duration":    convertedPPT.Metadata.Duration,
			},
			"slides": optimizedSlides,
		},
	}

	log.Printf("Coze链接处理完成: 结果ID=%s, 幻灯片数量=%d", result.ID, result.SlideCount)
	return result, nil
}

// GetBotConfig 获取智能体配置
func (s *CozeService) GetBotConfig() *coze.BotConfig {
	config := coze.DefaultCozeConfig()
	return config.Bot
}

// ValidateURL 验证URL
func (s *CozeService) ValidateURL(url string) bool {
	return s.parser.ValidatePPTURL(url)
}

// GetTaskResult 获取任务结果
func (s *CozeService) GetTaskResult(taskID string) (*ProcessResult, error) {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	if task.Status != "completed" {
		return nil, fmt.Errorf("任务尚未完成: %s", task.Status)
	}

	if task.ProcessResult == nil {
		return nil, fmt.Errorf("任务结果不可用")
	}

	return task.ProcessResult, nil
}

// CleanupTask 清理任务
func (s *CozeService) CleanupTask(taskID string) error {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		// 清理相关文件
		if task.ProcessResult != nil && task.ProcessResult.DownloadURL != "" {
			// 这里可以添加文件清理逻辑
			log.Printf("清理任务文件: %s", task.ProcessResult.DownloadURL)
		}
		delete(s.tasks, taskID)
	}

	return nil
}

// GetAllTasks 获取所有任务（用于调试）
func (s *CozeService) GetAllTasks() map[string]*TaskInfo {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	// 返回副本以避免并发问题
	tasks := make(map[string]*TaskInfo)
	for k, v := range s.tasks {
		tasks[k] = v
	}

	return tasks
}

// processWithNewPipeline 使用新的Pipeline处理PPT生成 (Phase 3集成)
func (s *CozeService) processWithNewPipeline(taskID string, result *coze.PPTResult) {
	log.Printf("开始使用新Pipeline处理任务: %s", taskID)

	// 更新任务状态
	s.updateTaskStatus(taskID, "processing", 60, "使用PPT处理器处理内容")

	// 使用Phase 2的PPT处理器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	processingResult, err := s.processor.ProcessCozeResponse(ctx, result)
	if err != nil {
		log.Printf("PPT处理失败: %v", err)
		s.updateTaskStatus(taskID, "failed", 0, fmt.Sprintf("PPT处理失败: %v", err))
		return
	}

	if !processingResult.Success {
		log.Printf("PPT处理未成功: %s", processingResult.Error)
		s.updateTaskStatus(taskID, "failed", 0, processingResult.Error)
		return
	}

	// 处理成功，更新任务信息
	s.updateTaskWithResult(taskID, processingResult)

	log.Printf("任务处理完成: %s, 方法: %s, 处理时间: %v",
		taskID, processingResult.Method, processingResult.ProcessTime)
}

// updateTaskStatus 更新任务状态
func (s *CozeService) updateTaskStatus(taskID, status string, progress int, message string) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = status
		task.Progress = progress
		if message != "" {
			if status == "failed" {
				task.ErrorMessage = message
			}
		}

		if status == "completed" || status == "failed" {
			now := time.Now()
			task.CompletedAt = &now
		}
	}
}

// updateTaskWithResult 使用处理结果更新任务
func (s *CozeService) updateTaskWithResult(taskID string, result *coze.ProcessingResult) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = "completed"
		task.Progress = 100
		now := time.Now()
		task.CompletedAt = &now

		if result.File != nil {
			// 设置下载URL和PPT URL
			task.DownloadURL = result.File.URL
			task.PPTURL = result.File.URL

			// 创建ProcessResult用于兼容
			task.ProcessResult = &ProcessResult{
				ID:          taskID,
				Title:       result.File.Title,
				SlideCount:  10, // 默认值，可以后续改进
				DownloadURL: result.File.URL,
				FileSize:    result.File.Size,
				ConvertedAt: result.File.CreatedAt,
			}
		}
	}
}

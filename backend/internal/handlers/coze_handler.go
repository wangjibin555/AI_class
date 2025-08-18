package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"ai-classroom/internal/services"

	"github.com/gin-gonic/gin"
)

// CozeHandler Coze处理器
type CozeHandler struct {
	cozeService *services.CozeService
}

// NewCozeHandler 创建Coze处理器
func NewCozeHandler(cozeService *services.CozeService) *CozeHandler {
	return &CozeHandler{
		cozeService: cozeService,
	}
}

// GetAIEngines 获取AI引擎列表
func (h *CozeHandler) GetAIEngines(c *gin.Context) {
	engines := h.cozeService.GetAIEngines()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": engines,
	})
}

// GetEngineConfig 获取引擎配置
func (h *CozeHandler) GetEngineConfig(c *gin.Context) {
	engineID := c.Param("engine_id")

	config, err := h.cozeService.GetEngineConfig(engineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "引擎不存在",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": config,
	})
}

// GeneratePPT 生成PPT
func (h *CozeHandler) GeneratePPT(c *gin.Context) {
	var req services.GeneratePPTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("收到PPT生成请求: URL=%s, Template=%s", req.URL, req.Template)

	// 调用服务生成PPT
	taskResp, err := h.cozeService.GeneratePPTAsync(c.Request.Context(), &req)
	if err != nil {
		log.Printf("生成PPT失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成PPT失败",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("PPT生成任务创建成功: TaskID=%s", taskResp.TaskID)
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "任务创建成功",
		"data":    taskResp,
	})
}

// 🆕 GeneratePPTWithWorkflow 使用工作流生成PPT
func (h *CozeHandler) GeneratePPTWithWorkflow(c *gin.Context) {
	var req services.GeneratePPTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("收到工作流PPT生成请求: URL=%s, Template=%s", req.URL, req.Template)
	log.Printf("🔍 调试: 即将调用GeneratePPTAsyncWithWorkflow...")

	// 调用工作流服务生成PPT
	taskResp, err := h.cozeService.GeneratePPTAsyncWithWorkflow(c.Request.Context(), &req)
	log.Printf("🔍 调试: GeneratePPTAsyncWithWorkflow调用完成, error=%v", err)
	if err != nil {
		log.Printf("工作流生成PPT失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "工作流生成PPT失败",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("工作流PPT生成任务创建成功: TaskID=%s", taskResp.TaskID)
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "工作流任务创建成功",
		"data":    taskResp,
	})
}

// GetTaskStatus 获取任务状态
func (h *CozeHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "任务ID不能为空",
		})
		return
	}

	status, err := h.cozeService.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		log.Printf("获取任务状态失败: TaskID=%s, Error=%v", taskID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "任务不存在或获取状态失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": status,
	})
}

// GetTaskResult 获取任务结果
func (h *CozeHandler) GetTaskResult(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "任务ID不能为空",
		})
		return
	}

	result, err := h.cozeService.GetTaskResult(taskID)
	if err != nil {
		log.Printf("获取任务结果失败: TaskID=%s, Error=%v", taskID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "任务结果不存在或未完成",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": result,
	})
}

// ProcessCozeLink 处理Coze链接
func (h *CozeHandler) ProcessCozeLink(c *gin.Context) {
	var req services.ProcessCozeLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 验证Coze URL
	if req.CozeURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Coze URL不能为空",
		})
		return
	}

	if !h.cozeService.ValidateURL(req.CozeURL) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的Coze URL格式",
		})
		return
	}

	log.Printf("开始处理Coze链接: %s", req.CozeURL)

	// 处理链接
	result, err := h.cozeService.ProcessCozeLink(c.Request.Context(), &req)
	if err != nil {
		log.Printf("处理Coze链接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "处理Coze链接失败",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("Coze链接处理成功: ID=%s, SlideCount=%d", result.ID, result.SlideCount)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": result,
	})
}

// GetAllTasks 获取所有任务（调试接口）
func (h *CozeHandler) GetAllTasks(c *gin.Context) {
	tasks := h.cozeService.GetAllTasks()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"tasks": tasks,
			"count": len(tasks),
		},
	})
}

// CleanupTask 清理任务
func (h *CozeHandler) CleanupTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "任务ID不能为空",
		})
		return
	}

	err := h.cozeService.CleanupTask(taskID)
	if err != nil {
		log.Printf("清理任务失败: TaskID=%s, Error=%v", taskID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "清理任务失败",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("任务清理成功: TaskID=%s", taskID)
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "任务清理成功",
	})
}

// ValidateURL 验证URL
func (h *CozeHandler) ValidateURL(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "URL参数不能为空",
		})
		return
	}

	isValid := h.cozeService.ValidateURL(url)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"url":   url,
			"valid": isValid,
			"message": func() string {
				if isValid {
					return "URL格式有效"
				}
				return "URL格式无效"
			}(),
		},
	})
}

// GetBotConfig 获取智能体配置
func (h *CozeHandler) GetBotConfig(c *gin.Context) {
	config := h.cozeService.GetBotConfig()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": config,
	})
}

// HealthCheck 健康检查
func (h *CozeHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Coze服务运行正常",
		"data": gin.H{
			"service":   "coze",
			"status":    "healthy",
			"timestamp": c.GetHeader("X-Request-Time"),
		},
	})
}

// DownloadFile 下载生成的PPT文件
func (h *CozeHandler) DownloadFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "文件名不能为空",
		})
		return
	}

	// 通过服务获取下载管理器并提供文件下载
	if err := h.cozeService.ServeDownloadFile(filename, c.Writer, c.Request); err != nil {
		log.Printf("文件下载失败: filename=%s, error=%v", filename, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "文件不存在或无法访问",
			"error":   err.Error(),
		})
		return
	}
}

// 🆕 GetTaskHTMLPreview 获取任务的HTML预览信息
func (h *CozeHandler) GetTaskHTMLPreview(c *gin.Context) {
	taskID := c.Param("task_id")

	taskInfo, exists := h.cozeService.GetTaskInfo(taskID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "任务不存在",
		})
		return
	}

	// 构建预览信息
	previewInfo := gin.H{
		"task_id":           taskID,
		"status":            taskInfo.Status,
		"conversion_status": taskInfo.ConversionStatus,
		"html_preview_url":  taskInfo.HTMLPreviewURL,
		"tmp_file_path":     taskInfo.TMPFilePath,
		"download_url":      taskInfo.DownloadURL, // 添加下载链接
		"ppt_url":           taskInfo.PPTURL,      // 添加PPT URL
	}

	// 如果有下载链接，构建完整的下载URL
	if taskInfo.DownloadURL != "" {
		previewInfo["pptx_download_url"] = taskInfo.DownloadURL
	} else if taskInfo.TMPFilePath != "" {
		// 如果没有直接下载链接，但有tmp文件，尝试构建下载链接
		// 从tmp文件路径提取文件名
		if strings.Contains(taskInfo.TMPFilePath, "/") {
			parts := strings.Split(taskInfo.TMPFilePath, "/")
			filename := parts[len(parts)-1]
			// 构建coze下载链接
			previewInfo["pptx_download_url"] = fmt.Sprintf("/api/v1/coze/download/%s", strings.Replace(filename, ".tmp", ".pptx", 1))
		}
	}

	// 如果转换完成，添加更多信息
	if taskInfo.ConversionStatus == "completed" && taskInfo.HTMLPreviewURL != "" {
		previewInfo["ready_for_preview"] = true
		previewInfo["preview_type"] = "html_fullscreen"
		previewInfo["has_download"] = previewInfo["pptx_download_url"] != nil // 标识是否可下载
	} else {
		previewInfo["ready_for_preview"] = false
		previewInfo["has_download"] = false
		if taskInfo.ConversionStatus == "converting" {
			previewInfo["message"] = "正在转换为HTML预览格式，请稍候..."
		} else if taskInfo.ConversionStatus == "failed" {
			previewInfo["message"] = "转换失败: " + taskInfo.ErrorMessage
		} else {
			previewInfo["message"] = "等待转换..."
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": previewInfo,
	})
}

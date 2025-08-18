package handlers

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ai-classroom/internal/coze"

	"github.com/gin-gonic/gin"
)

// TMPHTMLHandler .tmp文件HTML转换处理器
type TMPHTMLHandler struct {
	tmpConverter      *coze.TMPToHTMLConverter
	conversionService *ConversionService
	fileStorage       *coze.FileStorage
}

// ConversionService 转换服务（简化版本）
type ConversionService struct {
	results map[string]*coze.HTMLResult
}

// NewConversionService 创建转换服务
func NewConversionService() *ConversionService {
	return &ConversionService{
		results: make(map[string]*coze.HTMLResult),
	}
}

// SaveConversionResult 保存转换结果
func (s *ConversionService) SaveConversionResult(taskID string, result *coze.HTMLResult) error {
	s.results[taskID] = result
	return nil
}

// GetConversionResult 获取转换结果
func (s *ConversionService) GetConversionResult(taskID string) (*coze.HTMLResult, error) {
	if result, exists := s.results[taskID]; exists {
		return result, nil
	}
	return nil, errors.New("conversion result not found")
}

// NewTMPHTMLHandler 创建.tmp文件HTML处理器
func NewTMPHTMLHandler(tmpConverter *coze.TMPToHTMLConverter, fileStorage *coze.FileStorage) *TMPHTMLHandler {
	return &TMPHTMLHandler{
		tmpConverter:      tmpConverter,
		conversionService: NewConversionService(),
		fileStorage:       fileStorage,
	}
}

// ConvertTMPToHTMLRequest 转换请求
type ConvertTMPToHTMLRequest struct {
	TaskID      string               `json:"task_id" binding:"required"`
	TMPFilePath string               `json:"tmp_file_path" binding:"required"`
	Options     *coze.ConvertOptions `json:"options"`
}

// ConvertTMPToHTMLResponse 转换响应
type ConvertTMPToHTMLResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *coze.HTMLResult `json:"data"`
}

// GetTMPHTMLResponse 获取HTML响应
type GetTMPHTMLResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *coze.HTMLResult `json:"data"`
}

// ConvertTMPToHTML 转换.tmp文件为HTML
func (h *TMPHTMLHandler) ConvertTMPToHTML(c *gin.Context) {
	var req ConvertTMPToHTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 验证.tmp文件存在
	tmpFilePath := filepath.Join("./storage/ppt", req.TMPFilePath)
	if _, err := os.Stat(tmpFilePath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"code": 404, "message": ".tmp文件不存在"})
		return
	}

	// 设置默认选项
	if req.Options == nil {
		req.Options = &coze.ConvertOptions{
			Theme:          "white",
			Transition:     "slide",
			ShowControls:   true,
			ShowProgress:   true,
			EnableTouch:    true,
			EnableKeyboard: true,
		}
	}

	// 执行转换
	result, err := h.tmpConverter.ConvertTMPToHTML(tmpFilePath, req.Options)
	if err != nil {
		log.Printf("转换.tmp文件失败: %v", err)
		c.JSON(500, gin.H{"code": 500, "message": "转换失败: " + err.Error()})
		return
	}

	// 保存转换结果
	if err := h.conversionService.SaveConversionResult(req.TaskID, result); err != nil {
		log.Printf("保存转换结果失败: %v", err)
	}

	c.JSON(200, ConvertTMPToHTMLResponse{
		Code:    200,
		Message: "转换成功",
		Data:    result,
	})
}

// GetTMPHTMLPresentation 获取.tmp文件HTML演示
func (h *TMPHTMLHandler) GetTMPHTMLPresentation(c *gin.Context) {
	taskID := c.Param("taskId")

	// 从存储中获取转换结果
	result, err := h.conversionService.GetConversionResult(taskID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "演示文稿不存在"})
		return
	}

	c.JSON(200, GetTMPHTMLResponse{
		Code:    200,
		Message: "获取成功",
		Data:    result,
	})
}

// PreviewTMPHTMLPresentation 预览.tmp文件HTML演示
func (h *TMPHTMLHandler) PreviewTMPHTMLPresentation(c *gin.Context) {
	taskID := c.Param("taskId")

	// 从存储中获取转换结果
	result, err := h.conversionService.GetConversionResult(taskID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "演示文稿不存在"})
		return
	}

	// 重定向到预览URL
	if result.PreviewURL != "" {
		c.Redirect(302, result.PreviewURL)
		return
	}

	// 如果没有预览URL，尝试读取本地文件
	if result.LocalPath != "" {
		content, err := os.ReadFile(result.LocalPath)
		if err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "读取文件失败"})
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(content))
		return
	}

	c.JSON(404, gin.H{"code": 404, "message": "演示文稿文件不存在"})
}

// ListTMPFiles 列出.tmp文件
func (h *TMPHTMLHandler) ListTMPFiles(c *gin.Context) {
	pptDir := "./storage/ppt"
	files, err := os.ReadDir(pptDir)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取文件列表失败: " + err.Error()})
		return
	}

	var tmpFiles []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".tmp" {
			info, err := file.Info()
			if err != nil {
				continue
			}

			tmpFiles = append(tmpFiles, map[string]interface{}{
				"name":          file.Name(),
				"relative_path": file.Name(),
				"absolute_path": filepath.Join(pptDir, file.Name()),
				"size":          info.Size(),
				"modified_at":   info.ModTime(),
			})
		}
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    tmpFiles,
	})
}

// GetTMPFileInfo 获取.tmp文件信息
func (h *TMPHTMLHandler) GetTMPFileInfo(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join("./storage/ppt", filename)

	info, err := os.Stat(filePath)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"name":        filename,
			"path":        filePath,
			"size":        info.Size(),
			"modified_at": info.ModTime(),
			"is_dir":      info.IsDir(),
		},
	})
}

// DeleteTMPFile 删除.tmp文件
func (h *TMPHTMLHandler) DeleteTMPFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join("./storage/ppt", filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除文件失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// GetConversionStatus 获取转换状态
func (h *TMPHTMLHandler) GetConversionStatus(c *gin.Context) {
	taskID := c.Param("taskId")

	result, err := h.conversionService.GetConversionResult(taskID)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "获取成功",
			"data": map[string]interface{}{
				"task_id": taskID,
				"status":  "not_found",
				"message": "转换记录不存在",
			},
		})
		return
	}

	status := "completed"
	message := "转换完成"

	if result.ConversionInfo != nil {
		if len(result.ConversionInfo.Errors) > 0 {
			status = "completed_with_errors"
			message = "转换完成但有错误"
		}
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"task_id":         taskID,
			"status":          status,
			"message":         message,
			"total_slides":    result.TotalSlides,
			"file_size":       result.FileSize,
			"generated_at":    result.GeneratedAt,
			"preview_url":     result.PreviewURL,
			"conversion_info": result.ConversionInfo,
		},
	})
}

// GetConversionProgress 获取转换进度
func (h *TMPHTMLHandler) GetConversionProgress(c *gin.Context) {
	taskID := c.Param("taskId")

	result, err := h.conversionService.GetConversionResult(taskID)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "获取成功",
			"data": map[string]interface{}{
				"task_id":  taskID,
				"progress": 0,
				"status":   "not_found",
				"message":  "转换记录不存在",
			},
		})
		return
	}

	progress := 100
	status := "completed"
	message := "转换完成"

	if result.ConversionInfo != nil && len(result.ConversionInfo.Errors) > 0 {
		status = "completed_with_errors"
		message = "转换完成但有错误"
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"task_id":         taskID,
			"progress":        progress,
			"status":          status,
			"message":         message,
			"processing_time": result.ConversionInfo.ProcessingTime,
			"total_slides":    result.TotalSlides,
		},
	})
}

// ServeTMPResource 提供.tmp转换的资源文件
func (h *TMPHTMLHandler) ServeTMPResource(c *gin.Context) {
	taskID := c.Param("taskId")
	filename := c.Param("filename")

	result, err := h.conversionService.GetConversionResult(taskID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "转换记录不存在"})
		return
	}

	// 查找资源文件
	var resourcePath string
	for _, resource := range result.Resources {
		if filepath.Base(resource.LocalPath) == filename {
			resourcePath = resource.LocalPath
			break
		}
	}

	if resourcePath == "" {
		c.JSON(404, gin.H{"code": 404, "message": "资源文件不存在"})
		return
	}

	// 设置适当的Content-Type
	ext := filepath.Ext(filename)
	switch ext {
	case ".png":
		c.Header("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		c.Header("Content-Type", "image/jpeg")
	case ".gif":
		c.Header("Content-Type", "image/gif")
	case ".css":
		c.Header("Content-Type", "text/css")
	case ".js":
		c.Header("Content-Type", "application/javascript")
	default:
		c.Header("Content-Type", "application/octet-stream")
	}

	// 提供文件
	c.File(resourcePath)
}

// ProcessTMPFile 处理.tmp文件（集成到现有Coze服务）
func (h *TMPHTMLHandler) ProcessTMPFile(c *gin.Context) {
	taskID := c.Param("taskId")

	// 获取任务关联的.tmp文件
	tmpFiles, err := h.findTMPFilesByTaskID(taskID)
	if err != nil || len(tmpFiles) == 0 {
		c.JSON(404, gin.H{"code": 404, "message": "未找到相关的.tmp文件"})
		return
	}

	// 自动转换第一个.tmp文件
	tmpFile := tmpFiles[0]
	convertReq := ConvertTMPToHTMLRequest{
		TaskID:      taskID,
		TMPFilePath: tmpFile["relative_path"].(string),
		Options: &coze.ConvertOptions{
			Theme:          "white",
			Transition:     "slide",
			ShowControls:   true,
			ShowProgress:   true,
			EnableTouch:    true,
			EnableKeyboard: true,
		},
	}

	// 调用转换方法
	tmpFilePath := filepath.Join("./storage/ppt", convertReq.TMPFilePath)
	result, err := h.tmpConverter.ConvertTMPToHTML(tmpFilePath, convertReq.Options)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "转换失败: " + err.Error()})
		return
	}

	// 保存结果
	h.conversionService.SaveConversionResult(taskID, result)

	c.JSON(200, gin.H{
		"code":    200,
		"message": "处理成功",
		"data":    result,
	})
}

// findTMPFilesByTaskID 根据任务ID查找.tmp文件
func (h *TMPHTMLHandler) findTMPFilesByTaskID(taskID string) ([]map[string]interface{}, error) {
	pptDir := "./storage/ppt"
	files, err := os.ReadDir(pptDir)
	if err != nil {
		return nil, err
	}

	var tmpFiles []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".tmp" {
			// 简单的匹配逻辑，可以根据实际需求优化
			if contains := filepath.Base(file.Name()); contains != "" {
				info, err := file.Info()
				if err != nil {
					continue
				}

				tmpFiles = append(tmpFiles, map[string]interface{}{
					"name":          file.Name(),
					"relative_path": file.Name(),
					"absolute_path": filepath.Join(pptDir, file.Name()),
					"size":          info.Size(),
					"modified_at":   info.ModTime(),
				})
			}
		}
	}

	return tmpFiles, nil
}

// ServeHTMLFile 直接提供HTML文件服务（支持GET和HEAD请求）
func (h *TMPHTMLHandler) ServeHTMLFile(c *gin.Context) {
	filePath := c.Param("filepath")
	method := c.Request.Method

	log.Printf("🔍 收到%s请求: %s", method, filePath)

	if filePath == "" {
		log.Printf("❌ 文件路径为空")
		c.JSON(400, gin.H{"code": 400, "message": "文件路径不能为空"})
		return
	}

	// 移除开头的斜杠
	if filePath[0] == '/' {
		filePath = filePath[1:]
	}

	// 🔍 获取当前工作目录，便于调试
	workingDir, _ := os.Getwd()
	log.Printf("🔍 当前工作目录: %s", workingDir)

	// 🔧 修复路径问题：尝试多个可能的路径位置
	possiblePaths := []string{
		filepath.Join("output/html", filePath),                           // 相对于当前工作目录
		filepath.Join("./output/html", filePath),                         // 明确的相对路径
		filepath.Join(workingDir, "output/html", filePath),               // 基于当前工作目录
		filepath.Join("/opt/ai-classroom/backend/output/html", filePath), // 可能的绝对路径
		filepath.Join("backend/output/html", filePath),                   // 相对于项目根目录
		filepath.Join("wangjibin/backend/output/html", filePath),         // 相对于dev-2目录
	}

	var fullPath string
	var fileExists bool
	var fileInfo os.FileInfo

	// 尝试每个可能的路径
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil {
			fullPath = path
			fileExists = true
			fileInfo = info
			log.Printf("✅ 找到HTML文件: %s (大小: %d bytes)", fullPath, info.Size())
			break
		}
	}

	log.Printf("🔍 %s请求处理HTML文件: %s (尝试了%d个路径)", method, filePath, len(possiblePaths))

	if !fileExists {
		// 记录所有尝试的路径，便于调试
		log.Printf("❌ HTML文件不存在，尝试过的路径:")
		for i, path := range possiblePaths {
			log.Printf("  %d. %s", i+1, path)
		}

		// 对于HEAD请求，返回简单的404状态
		if method == "HEAD" {
			c.Status(404)
			return
		}

		c.JSON(404, gin.H{"code": 404, "message": "HTML文件不存在"})
		return
	}

	// 设置适当的响应头
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Last-Modified", fileInfo.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	// 对于HEAD请求，只返回头部信息，不返回文件内容
	if method == "HEAD" {
		log.Printf("✅ 成功响应HEAD请求: %s (大小: %d bytes)", fullPath, fileInfo.Size())
		c.Status(200)
		return
	}

	// 对于GET请求，返回完整文件
	log.Printf("✅ 成功提供HTML文件: %s (大小: %d bytes)", fullPath, fileInfo.Size())
	c.File(fullPath)
}

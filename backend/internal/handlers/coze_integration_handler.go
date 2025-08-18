package handlers

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"

	"ai-classroom/internal/middleware"
	"ai-classroom/internal/services"
	"ai-classroom/pkg/utils"

	"github.com/gin-gonic/gin"
)

// CozeIntegrationHandler Coze集成处理器
type CozeIntegrationHandler struct {
	integrationService *services.CozeCourseIntegrationService
}

// NewCozeIntegrationHandler 创建新的Coze集成处理器
func NewCozeIntegrationHandler(integrationService *services.CozeCourseIntegrationService) *CozeIntegrationHandler {
	return &CozeIntegrationHandler{
		integrationService: integrationService,
	}
}

// IntegrateHTMLRequest HTML集成请求
type IntegrateHTMLRequest struct {
	HTMLFilePath string `json:"html_file_path" binding:"required"`
	SourceURL    string `json:"source_url"`
	Title        string `json:"title"`
	Description  string `json:"description"`
}

// IntegrateHTMLContentRequest HTML内容集成请求
type IntegrateHTMLContentRequest struct {
	HTMLContent string `json:"html_content" binding:"required"`
	SourceURL   string `json:"source_url"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// IntegrateHTMLFromFile 从HTML文件集成到课程系统
// @Summary 从HTML文件集成到课程系统
// @Description 将Coze生成的HTML文件解析并集成到课程系统中
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param request body IntegrateHTMLRequest true "集成请求"
// @Success 200 {object} services.CozeIntegrationResponse "集成成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/coze/integrate/html [post]
func (h *CozeIntegrationHandler) IntegrateHTMLFromFile(c *gin.Context) {
	var req IntegrateHTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 构建集成请求
	integrationReq := &services.CozeIntegrationRequest{
		HTMLFilePath: req.HTMLFilePath,
		SourceURL:    req.SourceURL,
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
	}

	// 执行集成
	result, err := h.integrationService.IntegrateCozeHTMLToCourse(integrationReq)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "集成失败", err.Error())
		return
	}

	if !result.Success {
		utils.ErrorResponse(c, http.StatusBadRequest, "集成失败", result.Error)
		return
	}

	utils.SuccessResponse(c, result.Message, result)
}

// IntegrateHTMLFromUpload 从上传的HTML文件集成到课程系统
// @Summary 从上传的HTML文件集成到课程系统
// @Description 上传HTML文件并将其解析集成到课程系统中
// @Tags Coze Integration
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "HTML文件"
// @Param source_url formData string false "源URL"
// @Param title formData string false "自定义标题"
// @Param description formData string false "自定义描述"
// @Success 200 {object} services.CozeIntegrationResponse "集成成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/coze/integrate/upload [post]
func (h *CozeIntegrationHandler) IntegrateHTMLFromUpload(c *gin.Context) {
	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "获取上传文件失败", err.Error())
		return
	}
	defer file.Close()

	// 验证文件类型
	if ext := filepath.Ext(header.Filename); ext != ".html" && ext != ".htm" {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件类型错误", "只支持HTML格式文件")
		return
	}

	// 读取文件内容
	htmlContent, err := ioutil.ReadAll(file)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取文件内容失败", err.Error())
		return
	}

	// 获取表单参数
	sourceURL := c.PostForm("source_url")
	title := c.PostForm("title")
	description := c.PostForm("description")

	// 直接调用HTML解析器处理内容
	result, err := h.integrateHTMLContent(string(htmlContent), userID, sourceURL, title, description)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "集成失败", err.Error())
		return
	}

	utils.SuccessResponse(c, result.Message, result)
}

// IntegrateHTMLContent 处理直接传入HTML内容的Coze集成请求
// @Summary 从HTML内容集成到课程系统
// @Description 将Coze生成的HTML内容解析并集成到课程系统中
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param request body IntegrateHTMLContentRequest true "HTML内容集成请求"
// @Success 200 {object} services.CozeIntegrationResponse "集成成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/coze-integration/html-content [post]
func (h *CozeIntegrationHandler) IntegrateHTMLContent(c *gin.Context) {
	var req IntegrateHTMLContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	if req.HTMLContent == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "HTML内容不能为空")
		return
	}

	resp, err := h.integrationService.IntegrateHTMLContentToCourse(req.HTMLContent, &services.CozeIntegrationRequest{
		UserID:      userID,
		SourceURL:   req.SourceURL,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "集成Coze HTML内容失败", err.Error())
		return
	}

	if !resp.Success {
		utils.ErrorResponse(c, http.StatusInternalServerError, "集成Coze HTML内容失败", resp.Error)
		return
	}

	utils.SuccessResponse(c, "Coze HTML内容集成成功", resp)
}

// IntegrateHTMLFile 处理从文件上传的Coze HTML集成请求
// @Summary 从文件路径集成HTML到课程系统
// @Description 将指定路径的Coze生成HTML文件解析并集成到课程系统中
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param request body IntegrateHTMLRequest true "HTML文件集成请求"
// @Success 200 {object} services.CozeIntegrationResponse "集成成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/coze-integration/html-file [post]
func (h *CozeIntegrationHandler) IntegrateHTMLFile(c *gin.Context) {
	var req IntegrateHTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	if req.HTMLFilePath == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "HTML文件路径不能为空")
		return
	}

	resp, err := h.integrationService.IntegrateCozeHTMLToCourse(&services.CozeIntegrationRequest{
		UserID:       userID,
		HTMLFilePath: req.HTMLFilePath,
		SourceURL:    req.SourceURL,
		Title:        req.Title,
		Description:  req.Description,
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "集成Coze HTML文件失败", err.Error())
		return
	}

	if !resp.Success {
		utils.ErrorResponse(c, http.StatusInternalServerError, "集成Coze HTML文件失败", resp.Error)
		return
	}

	utils.SuccessResponse(c, "Coze HTML文件集成成功", resp)
}

// integrateHTMLContent 集成HTML内容（内部方法）
func (h *CozeIntegrationHandler) integrateHTMLContent(htmlContent string, userID uint, sourceURL, title, description string) (*services.CozeIntegrationResponse, error) {
	// 创建一个临时的集成请求
	integrationReq := &services.CozeIntegrationRequest{
		HTMLFilePath: "upload://" + title, // 标记为上传文件
		SourceURL:    sourceURL,
		UserID:       userID,
		Title:        title,
		Description:  description,
	}

	// 由于我们有HTML内容，需要直接调用HTML解析器
	// 这里需要修改集成服务以支持直接传入HTML内容
	return h.integrationService.IntegrateHTMLContentToCourse(htmlContent, integrationReq)
}

// GetCourse 获取课程详情
// @Summary 获取课程详情
// @Description 获取指定课程的详细信息，包括幻灯片
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param id path int true "课程ID"
// @Success 200 {object} models.Course "课程详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "课程不存在"
// @Router /api/v1/coze/courses/{id} [get]
func (h *CozeIntegrationHandler) GetCourse(c *gin.Context) {
	// 获取课程ID
	courseIDStr := c.Param("id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "课程ID格式错误", err.Error())
		return
	}

	// 获取课程信息
	course, err := h.integrationService.GetCourseWithSlides(uint(courseID))
	if err != nil {
		if err.Error() == "课程不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, "课程不存在")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取课程失败", err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "获取课程成功", course)
}

// ListCourses 获取课程列表
// @Summary 获取Coze集成的课程列表
// @Description 分页获取当前用户通过Coze集成的课程列表
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "课程列表"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/coze/courses [get]
func (h *CozeIntegrationHandler) ListCourses(c *gin.Context) {
	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 查询课程列表
	courses, total, err := h.integrationService.ListCozeIntegratedCourses(userID, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取课程列表失败", err.Error())
		return
	}

	// 构建响应数据
	result := map[string]interface{}{
		"courses":     courses,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}

	utils.SuccessResponse(c, "获取课程列表成功", result)
}

// UpdateCourse 更新课程信息
// @Summary 更新课程信息
// @Description 更新指定课程的基本信息
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param id path int true "课程ID"
// @Param request body map[string]string true "更新信息"
// @Success 200 {object} map[string]string "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "课程不存在"
// @Router /api/v1/coze/courses/{id} [put]
func (h *CozeIntegrationHandler) UpdateCourse(c *gin.Context) {
	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取课程ID
	courseIDStr := c.Param("id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "课程ID格式错误", err.Error())
		return
	}

	// 获取更新参数
	var updateReq map[string]string
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 更新课程信息
	err = h.integrationService.UpdateCourseInfo(uint(courseID), userID, updateReq["title"], updateReq["description"])
	if err != nil {
		if err.Error() == "课程不存在或无权限修改" {
			utils.ErrorResponse(c, http.StatusNotFound, "课程不存在或无权限修改")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新课程失败", err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "课程更新成功", nil)
}

// DeleteCourse 删除课程
// @Summary 删除课程
// @Description 删除指定的课程及其相关数据
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Param id path int true "课程ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "课程不存在"
// @Router /api/v1/coze/courses/{id} [delete]
func (h *CozeIntegrationHandler) DeleteCourse(c *gin.Context) {
	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取课程ID
	courseIDStr := c.Param("id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "课程ID格式错误", err.Error())
		return
	}

	// 删除课程
	err = h.integrationService.DeleteCourse(uint(courseID), userID)
	if err != nil {
		if err.Error() == "课程不存在或无权限删除" {
			utils.ErrorResponse(c, http.StatusNotFound, "课程不存在或无权限删除")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "删除课程失败", err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "课程删除成功", nil)
}

// GetCourseStats 获取课程统计信息
// @Summary 获取课程统计信息
// @Description 获取当前用户的课程统计信息
// @Tags Coze Integration
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "统计信息"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/coze/stats [get]
func (h *CozeIntegrationHandler) GetCourseStats(c *gin.Context) {
	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取统计信息
	stats, err := h.integrationService.GetCourseStats(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取统计信息失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "获取统计信息成功", stats)
}

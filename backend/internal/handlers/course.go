package handlers

import (
	"ai-classroom/internal/models"
	"ai-classroom/internal/services"
	"ai-classroom/internal/websocket"
	"ai-classroom/pkg/utils"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CourseHandler struct {
	db            *gorm.DB
	rdb           *redis.Client
	hub           *websocket.Hub
	courseService services.CourseService
}

func NewCourseHandler(db *gorm.DB, rdb *redis.Client, hub *websocket.Hub, courseService services.CourseService) *CourseHandler {
	return &CourseHandler{
		db:            db,
		rdb:           rdb,
		hub:           hub,
		courseService: courseService,
	}
}

// GetCourses 获取课件列表
// @Summary 获取课件列表
// @Description 获取用户的课件列表，支持分页和筛选
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Param status query string false "状态筛选" Enums(generating,completed,failed)
// @Param category query string false "分类筛选"
// @Param is_public query bool false "是否公开"
// @Param sort_by query string false "排序字段" default(created_at)
// @Param sort_order query string false "排序方向" default(desc)
// @Success 200 {object} utils.Response{data=services.GetCoursesResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses [get]
func (h *CourseHandler) GetCourses(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	status := c.Query("status")
	category := c.Query("category")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// 解析is_public参数
	var isPublic *bool
	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		if val, err := strconv.ParseBool(isPublicStr); err == nil {
			isPublic = &val
		}
	}

	// 构建请求
	req := services.GetCoursesRequest{
		UserID:    userID.(uint),
		Page:      page,
		PageSize:  pageSize,
		Status:    models.CourseStatus(status),
		Category:  models.CourseCategory(category),
		IsPublic:  isPublic,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	// 调用服务
	response, err := h.courseService.GetUserCourses(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取课件列表失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "获取课件列表成功", response)
}

// CreateCourse 创建课件
// @Summary 创建课件
// @Description 从URL、文本或文档创建课件
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body services.CreateCourseRequest true "课件创建请求"
// @Success 200 {object} utils.Response{data=models.Course}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses [post]
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析请求参数
	var req services.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 设置用户ID - 修正：CreateCourseRequest结构中没有UserID字段，需要在服务层处理
	// req.UserID = userID.(uint)

	// 调用服务创建课件 - 传递userID作为参数
	course, err := h.courseService.CreateCourse(userID.(uint), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "创建课件失败: "+err.Error())
		return
	}

	// 确保返回完整的课程信息，包括ID
	if course.ID == 0 {
		utils.ErrorResponse(c, http.StatusInternalServerError, "课程ID生成失败")
		return
	}

	// 添加调试日志
	log.Printf("课程创建成功，ID: %d, Title: %s, UserID: %d", course.ID, course.Title, course.UserID)

	// TODO: 触发AI生成任务（异步）
	// 这里可以发送WebSocket消息或者放入队列

	utils.SuccessResponse(c, "课件创建成功", course)
}

// GetGenerationStatus 获取课件生成状态
// @Summary 获取课件生成状态
// @Description 获取指定课件的生成状态和进度
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Success 200 {object} utils.Response{data=models.Course}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/{id}/generation-status [get]
func (h *CourseHandler) GetGenerationStatus(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	courseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 获取课件详情
	course, err := h.courseService.GetCourseByID(uint(courseID), userID.(uint))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "课件不存在")
		return
	}

	utils.SuccessResponse(c, "获取生成状态成功", course)
}

// UploadCourse 上传文档创建课件
// @Summary 上传文档创建课件
// @Description 上传PDF、Word等文档创建课件
// @Tags 课件管理
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param file formData file true "文档文件"
// @Param title formData string true "课件标题"
// @Param description formData string false "课件描述"
// @Param category formData string true "课件分类"
// @Param tags formData string false "课件标签(JSON数组)"
// @Param is_public formData bool false "是否公开"
// @Param voice_type formData string false "语音类型"
// @Success 200 {object} utils.Response{data=models.Course}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/upload [post]
func (h *CourseHandler) UploadCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件上传失败: "+err.Error())
		return
	}

	// 解析表单参数
	title := c.PostForm("title")
	description := c.PostForm("description")
	category := c.PostForm("category")
	isPublicStr := c.DefaultPostForm("is_public", "false")
	voiceType := c.DefaultPostForm("voice_type", "zhimao")

	// 解析布尔值
	isPublic, _ := strconv.ParseBool(isPublicStr)

	// TODO: 解析tags JSON数组
	var tags models.Tags

	// 构建请求
	req := services.CreateCourseRequest{
		Title:       title,
		Description: description,
		Category:    models.CourseCategory(category),
		Tags:        tags,
		SourceType:  models.SourceTypeDocument,
		IsPublic:    isPublic,
		VoiceType:   voiceType,
	}

	// 调用服务创建课件
	course, err := h.courseService.CreateCourseFromFile(userID.(uint), file, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "创建课件失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "课件创建成功", course)
}

// GetLatestCourse 获取用户最新课程
// @Summary 获取用户最新课程
// @Description 获取用户最新创建的课程
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utils.Response{data=models.Course}
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/latest [get]
func (h *CourseHandler) GetLatestCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取用户最新课程
	req := services.GetCoursesRequest{
		UserID:    userID.(uint),
		Page:      1,
		PageSize:  1,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	response, err := h.courseService.GetUserCourses(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取最新课程失败: "+err.Error())
		return
	}

	if len(response.Courses) == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "未找到课程")
		return
	}

	utils.SuccessResponse(c, "获取最新课程成功", response.Courses[0])
}

// GetCourse 获取课件详情
// @Summary 获取课件详情
// @Description 获取指定课件的详细信息
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Success 200 {object} utils.Response{data=models.Course}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/{id} [get]
func (h *CourseHandler) GetCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 调用服务获取课件
	course, err := h.courseService.GetCourseByID(uint(id), userID.(uint))
	if err != nil {
		if err.Error() == "课件不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else if err.Error() == "无权限访问该课件" {
			utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取课件失败: "+err.Error())
		}
		return
	}

	// 增加观看次数（异步操作，不影响响应）
	go func() {
		_ = h.courseService.IncrementViewCount(uint(id))
	}()

	utils.SuccessResponse(c, "获取课件成功", course)
}

// UpdateCourse 更新课件信息
// @Summary 更新课件信息
// @Description 更新课件的基本信息
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Param request body services.UpdateCourseRequest true "课件更新请求"
// @Success 200 {object} utils.Response{data=models.Course}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/{id} [put]
func (h *CourseHandler) UpdateCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 解析请求参数
	var req services.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 调用服务更新课件
	course, err := h.courseService.UpdateCourse(uint(id), userID.(uint), req)
	if err != nil {
		if err.Error() == "课件不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else if err.Error() == "无权限更新该课件" {
			utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "更新课件失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "更新课件成功", course)
}

// DeleteCourse 删除课件
// @Summary 删除课件
// @Description 删除指定课件
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/{id} [delete]
func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 调用服务删除课件
	err = h.courseService.DeleteCourse(uint(id), userID.(uint))
	if err != nil {
		if err.Error() == "课件不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else if err.Error() == "无权限删除该课件" {
			utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "删除课件失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "删除课件成功", nil)
}

// GetProgress 获取课件生成进度
// @Summary 获取课件生成进度
// @Description 获取指定课件的生成进度
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/{id}/progress [get]
func (h *CourseHandler) GetProgress(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 获取课件信息
	course, err := h.courseService.GetCourseByID(uint(id), userID.(uint))
	if err != nil {
		if err.Error() == "课件不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取课件失败: "+err.Error())
		}
		return
	}

	// TODO: 从Redis或其他地方获取实际的生成进度
	progress := map[string]interface{}{
		"course_id":     course.ID,
		"status":        course.Status,
		"progress":      100,         // TODO: 实际进度计算
		"current_step":  "completed", // TODO: 当前步骤
		"error_message": course.ErrorMessage,
		"slides_count":  course.SlidesCount,
	}

	utils.SuccessResponse(c, "获取生成进度成功", progress)
}

// CancelGeneration 取消课件生成
// @Summary 取消课件生成
// @Description 取消正在生成中的课件
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/{id}/cancel [post]
func (h *CourseHandler) CancelGeneration(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 获取课件信息
	course, err := h.courseService.GetCourseByID(uint(id), userID.(uint))
	if err != nil {
		if err.Error() == "课件不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取课件失败: "+err.Error())
		}
		return
	}

	// 检查课件状态
	if course.Status != models.CourseStatusGenerating {
		utils.ErrorResponse(c, http.StatusBadRequest, "课件未在生成中，无法取消")
		return
	}

	// TODO: 实现取消生成逻辑
	// 1. 停止AI生成任务
	// 2. 更新课件状态为failed
	// 3. 清理临时文件

	utils.SuccessResponse(c, "取消生成成功", nil)
}

// RegenerateCourse 重新生成课件
// @Summary 重新生成课件
// @Description 重新生成失败的课件
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/{id}/regenerate [post]
func (h *CourseHandler) RegenerateCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课件ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课件ID")
		return
	}

	// 获取课件信息
	course, err := h.courseService.GetCourseByID(uint(id), userID.(uint))
	if err != nil {
		if err.Error() == "课件不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取课件失败: "+err.Error())
		}
		return
	}

	// 检查课件状态
	if course.Status == models.CourseStatusGenerating {
		utils.ErrorResponse(c, http.StatusBadRequest, "课件正在生成中，无法重新生成")
		return
	}

	// TODO: 实现重新生成逻辑
	// 1. 重置课件状态为generating
	// 2. 清理之前的生成结果
	// 3. 重新触发AI生成任务

	utils.SuccessResponse(c, "重新生成任务已启动", nil)
}

// GetPublicCourses 获取公开课件列表
// @Summary 获取公开课件列表
// @Description 获取所有公开的课件列表
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Param status query string false "状态筛选" Enums(generating,completed,failed)
// @Param category query string false "分类筛选"
// @Param sort_by query string false "排序字段" default(created_at)
// @Param sort_order query string false "排序方向" default(desc)
// @Success 200 {object} utils.Response{data=services.GetCoursesResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/public [get]
func (h *CourseHandler) GetPublicCourses(c *gin.Context) {
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	status := c.Query("status")
	category := c.Query("category")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// 构建请求
	req := services.GetCoursesRequest{
		Page:      page,
		PageSize:  pageSize,
		Status:    models.CourseStatus(status),
		Category:  models.CourseCategory(category),
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	// 调用服务
	response, err := h.courseService.GetPublicCourses(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取公开课件列表失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "获取公开课件列表成功", response)
}

// SearchCourses 搜索课件
// @Summary 搜索课件
// @Description 根据关键词搜索用户的课件
// @Tags 课件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param keyword query string true "搜索关键词"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Success 200 {object} utils.Response{data=services.GetCoursesResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/search [get]
func (h *CourseHandler) SearchCourses(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析查询参数
	keyword := c.Query("keyword")
	if keyword == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	// 构建请求
	req := services.SearchCoursesRequest{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
		UserID:   userID.(uint), // 添加用户ID
	}

	// 调用服务
	response, err := h.courseService.SearchCourses(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "搜索课件失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "搜索课件成功", response)
}

// GetLearningRecords 获取学习记录
// @Summary 获取学习记录
// @Description 获取用户的学习记录列表
// @Tags 学习记录
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Param status query string false "状态筛选" Enums(in_progress,completed,expired)
// @Param category query string false "分类筛选"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/learning-records [get]
func (h *CourseHandler) GetLearningRecords(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	status := c.Query("status")
	category := c.Query("category")

	// 构建查询条件
	query := h.db.Model(&models.LearningRecord{}).
		Joins("JOIN courses ON learning_records.course_id = courses.id").
		Where("learning_records.user_id = ?", userID)

	// 添加状态筛选
	if status != "" {
		query = query.Where("learning_records.status = ?", status)
	}

	// 添加分类筛选
	if category != "" {
		query = query.Where("courses.category = ?", category)
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 分页查询
	var records []map[string]interface{}
	err := query.
		Select("learning_records.*, courses.title as course_title, courses.category as course_category").
		Order("learning_records.last_study_time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&records).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取学习记录失败: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"records":   records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	utils.SuccessResponse(c, "获取学习记录成功", response)
}

// GetLearningStats 获取学习统计
// @Summary 获取学习统计
// @Description 获取用户的学习统计信息
// @Tags 学习记录
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/courses/learning-records/stats [get]
func (h *CourseHandler) GetLearningStats(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 查询统计信息
	var stats struct {
		TotalCourses      int64 `json:"total_courses"`
		CompletedCourses  int64 `json:"completed_courses"`
		InProgressCourses int64 `json:"in_progress_courses"`
		TotalStudyTime    int64 `json:"total_study_time"`
		TotalQuizScore    int64 `json:"total_quiz_score"`
	}

	// 总课程数
	h.db.Model(&models.LearningRecord{}).
		Where("user_id = ?", userID).
		Count(&stats.TotalCourses)

	// 已完成课程数
	h.db.Model(&models.LearningRecord{}).
		Where("user_id = ? AND status = ?", userID, models.LearningStatusCompleted).
		Count(&stats.CompletedCourses)

	// 学习中课程数
	h.db.Model(&models.LearningRecord{}).
		Where("user_id = ? AND status = ?", userID, models.LearningStatusLearning).
		Count(&stats.InProgressCourses)

	// 总学习时长
	h.db.Model(&models.LearningRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(study_duration), 0)").
		Scan(&stats.TotalStudyTime)

	// 总测验分数
	h.db.Model(&models.LearningRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(highest_score), 0)").
		Scan(&stats.TotalQuizScore)

	utils.SuccessResponse(c, "获取学习统计成功", stats)
}

// ResetLearningProgress 重置学习进度
// @Summary 重置学习进度
// @Description 重置指定课程的学习进度
// @Tags 学习记录
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课程ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/learning-records/{id}/reset [post]
func (h *CourseHandler) ResetLearningProgress(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课程ID
	courseIDStr := c.Param("id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课程ID")
		return
	}

	// 重置学习进度
	err = h.db.Model(&models.LearningRecord{}).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		Updates(map[string]interface{}{
			"status":          models.LearningStatusLearning,
			"progress":        0,
			"study_duration":  0,
			"quiz_attempts":   0,
			"last_study_time": time.Now(),
		}).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重置学习进度失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "重置学习进度成功", nil)
}

// DeleteLearningRecord 删除学习记录
// @Summary 删除学习记录
// @Description 删除指定的学习记录
// @Tags 学习记录
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "记录ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/learning-records/{id} [delete]
func (h *CourseHandler) DeleteLearningRecord(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析记录ID
	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的记录ID")
		return
	}

	// 删除学习记录
	result := h.db.Where("id = ? AND user_id = ?", recordID, userID).Delete(&models.LearningRecord{})
	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除学习记录失败: "+result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "学习记录不存在")
		return
	}

	utils.SuccessResponse(c, "删除学习记录成功", nil)
}

// UpdateLearningProgress 更新学习进度
// @Summary 更新学习进度
// @Description 更新用户对指定课程的学习进度
// @Tags 学习记录
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课程ID"
// @Param progress body map[string]interface{} true "进度数据"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/{id}/progress [put]
func (h *CourseHandler) UpdateLearningProgress(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课程ID
	courseIDStr := c.Param("id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课程ID")
		return
	}

	// 解析进度数据
	var progressData map[string]interface{}
	if err := c.ShouldBindJSON(&progressData); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err.Error())
		return
	}

	// 更新或创建学习记录
	var record models.LearningRecord
	result := h.db.Where("user_id = ? AND course_id = ?", userID, courseID).First(&record)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 创建新的学习记录
			record = models.LearningRecord{
				UserID:   userID.(uint),
				CourseID: uint(courseID),
				Status:   models.LearningStatusLearning,
			}
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询学习记录失败", result.Error.Error())
			return
		}
	}

	// 更新进度信息
	if currentSlide, ok := progressData["current_slide"].(float64); ok {
		record.CurrentSlide = int(currentSlide)
	}
	if progress, ok := progressData["progress"].(float64); ok {
		record.Progress = progress
	}
	if studyDuration, ok := progressData["study_duration"].(float64); ok {
		record.StudyDuration = int(studyDuration)
	}

	// 更新最后学习时间
	now := time.Now()
	record.LastStudyTime = &now

	// 如果进度达到100%，标记为完成
	if record.Progress >= 100 {
		record.IsCompleted = true
		record.Status = models.LearningStatusCompleted
		record.EndTime = &now
	}

	// 保存记录
	if record.ID == 0 {
		err = h.db.Create(&record).Error
	} else {
		err = h.db.Save(&record).Error
	}

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存学习进度失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "更新学习进度成功", record)
}

// CompleteCourse 完成课程
// @Summary 完成课程
// @Description 标记课程为已完成状态
// @Tags 学习记录
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "课程ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/courses/{id}/complete [post]
func (h *CourseHandler) CompleteCourse(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析课程ID
	courseIDStr := c.Param("id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课程ID")
		return
	}

	// 查找学习记录
	var record models.LearningRecord
	result := h.db.Where("user_id = ? AND course_id = ?", userID, courseID).First(&record)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "学习记录不存在")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询学习记录失败", result.Error.Error())
		}
		return
	}

	// 标记为完成
	now := time.Now()
	record.IsCompleted = true
	record.Status = models.LearningStatusCompleted
	record.Progress = 1.0     // 数据库中存储1.0表示100%
	record.CompleteRate = 1.0 // 数据库中存储1.0表示100%
	record.EndTime = &now
	record.LastStudyTime = &now

	// 保存记录
	if err := h.db.Save(&record).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "完成课程失败", err.Error())
		return
	}

	utils.SuccessResponse(c, "课程完成成功", record)
}

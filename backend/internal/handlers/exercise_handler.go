package handlers

import (
	"net/http"
	"strconv"

	"ai-classroom/internal/models"
	"ai-classroom/internal/services"
	"ai-classroom/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ExerciseHandler 练习生成处理器
type ExerciseHandler struct {
	exerciseService *services.ExerciseGenerationService
}

// NewExerciseHandler 创建练习生成处理器
func NewExerciseHandler(exerciseService *services.ExerciseGenerationService) *ExerciseHandler {
	return &ExerciseHandler{
		exerciseService: exerciseService,
	}
}

// CheckCourseQuiz 检查课程是否已有练习
// @Summary 检查课程是否已有练习
// @Description 检查指定课程是否已存在AI生成的练习
// @Tags Exercise
// @Accept json
// @Produce json
// @Param courseId path int true "课程ID"
// @Success 200 {object} map[string]interface{} "检查结果"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "课程不存在"
// @Router /api/v1/exercises/check/{courseId} [get]
func (h *ExerciseHandler) CheckCourseQuiz(c *gin.Context) {
	// 获取课程ID
	courseIDStr := c.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的课程ID", err.Error())
		return
	}

	// 从认证中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未授权访问", "")
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户ID格式错误", "")
		return
	}

	// 检查练习是否存在
	existingQuiz, err := h.exerciseService.CheckExistingAIQuiz(uint(courseID), userIDUint)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "检查练习失败", err.Error())
		return
	}

	// 返回检查结果
	result := map[string]interface{}{
		"course_id": courseID,
		"exists":    existingQuiz != nil,
		"quiz_id":   nil,
		"status":    "none",
	}

	if existingQuiz != nil {
		result["quiz_id"] = existingQuiz.ID

		// 检查是否为正在生成中的特殊状态
		if existingQuiz.ID == 0 && existingQuiz.Status == "generating" {
			result["status"] = "generating"
			result["title"] = existingQuiz.Title
			result["question_count"] = existingQuiz.QuestionCount
		} else {
			result["status"] = "completed"
		}
	}

	utils.SuccessResponse(c, "检查完成", result)
}

// GenerateExercise 生成练习
// @Summary 为课程生成练习
// @Description 基于课程内容使用AI工作流生成练习题
// @Tags Exercise
// @Accept json
// @Produce json
// @Param request body models.GenerationRequest true "生成请求"
// @Success 200 {object} models.GenerationResponse "生成成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "课程不存在"
// @Failure 409 {object} map[string]string "已存在练习"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/exercises/generate [post]
func (h *ExerciseHandler) GenerateExercise(c *gin.Context) {
	var req models.GenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 从认证中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未授权访问",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户ID格式错误",
		})
		return
	}

	// 设置默认值
	if req.QuizTitle == "" {
		req.QuizTitle = "智能生成练习"
	}
	if req.Difficulty == "" {
		req.Difficulty = "normal"
	}
	if req.QuestionCount == 0 {
		req.QuestionCount = 8 // 默认8道题
	}

	// 调用服务生成练习
	result, err := h.exerciseService.GenerateExerciseForCourse(c.Request.Context(), userIDUint, req.CourseID, req)
	if err != nil {
		if err.Error() == "课程不存在或无权访问" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		if err.Error() == "课程缺少源URL，无法生成练习" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成练习失败: " + err.Error(),
		})
		return
	}

	// 如果已存在练习
	if result.Status == "already_exists" {
		utils.ErrorResponseWithData(c, http.StatusConflict, "该课程已存在AI生成的练习", "", gin.H{
			"quiz_id": result.QuizID,
		})
		return
	}

	// 返回成功结果
	response := models.GenerationResponse{
		QuizID:         result.QuizID,
		TotalQuestions: result.TotalQuestions,
		GenerationID:   result.GenerationID,
		Status:         result.Status,
		Message:        "练习生成成功",
	}

	utils.SuccessResponse(c, "练习生成成功", response)
}

// GetGenerationRecord 获取生成记录
// @Summary 获取练习生成记录
// @Description 获取指定的练习生成记录详情
// @Tags Exercise
// @Produce json
// @Param id path int true "生成记录ID"
// @Success 200 {object} models.ExerciseGenerationRecord "生成记录详情"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "记录不存在"
// @Router /api/v1/exercises/generation/{id} [get]
func (h *ExerciseHandler) GetGenerationRecord(c *gin.Context) {
	// 获取记录ID
	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "记录ID格式错误",
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未授权访问",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户ID格式错误",
		})
		return
	}

	// 查询生成记录
	record, err := h.exerciseService.GetGenerationRecord(uint(recordID), userIDUint)
	if err != nil {
		if err.Error() == "生成记录不存在" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询生成记录失败: " + err.Error(),
		})
		return
	}

	utils.SuccessResponse(c, "获取生成记录成功", record)
}

// GetUserGenerationRecords 获取用户的生成记录列表
// @Summary 获取用户练习生成记录列表
// @Description 获取当前用户的所有练习生成记录
// @Tags Exercise
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Success 200 {array} models.ExerciseGenerationRecord "生成记录列表"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/exercises/generation [get]
func (h *ExerciseHandler) GetUserGenerationRecords(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未授权访问",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户ID格式错误",
		})
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 50

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// 查询生成记录列表
	records, err := h.exerciseService.GetUserGenerationRecords(userIDUint, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询生成记录失败: " + err.Error(),
		})
		return
	}

	recordsData := gin.H{
		"records":   records,
		"page":      page,
		"page_size": pageSize,
		"total":     len(records),
	}
	utils.SuccessResponse(c, "获取生成记录列表成功", recordsData)
}

// GetGenerationStatus 获取生成状态
// @Summary 获取练习生成状态
// @Description 获取指定练习的生成状态和进度
// @Tags Exercise
// @Produce json
// @Param id path int true "生成记录ID"
// @Success 200 {object} map[string]interface{} "生成状态"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "记录不存在"
// @Router /api/v1/exercises/generation/{id}/status [get]
func (h *ExerciseHandler) GetGenerationStatus(c *gin.Context) {
	// 获取记录ID
	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "记录ID格式错误",
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未授权访问",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户ID格式错误",
		})
		return
	}

	// 查询生成记录
	record, err := h.exerciseService.GetGenerationRecord(uint(recordID), userIDUint)
	if err != nil {
		if err.Error() == "生成记录不存在" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询生成记录失败: " + err.Error(),
		})
		return
	}

	// 计算进度百分比
	progress := 0
	switch record.GenerationStatus {
	case models.GenerationStatusPending:
		progress = 0
	case models.GenerationStatusProcessing:
		progress = 50
	case models.GenerationStatusCompleted:
		progress = 100
	case models.GenerationStatusFailed:
		progress = 0
	}

	statusData := gin.H{
		"status":              record.GenerationStatus,
		"progress":            progress,
		"total_questions":     record.TotalQuestions,
		"processing_duration": record.ProcessingDuration,
		"error_message":       record.ErrorMessage,
		"quiz_id":             record.QuizID,
		"created_at":          record.CreatedAt,
		"updated_at":          record.UpdatedAt,
	}
	utils.SuccessResponse(c, "获取生成状态成功", statusData)
}

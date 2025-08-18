package handlers

import (
	"ai-classroom/internal/services"
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/utils"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type QuizHandler struct {
	db          *gorm.DB
	rdb         *redis.Client
	quizService services.QuizService
	aiClient    *ai.DashScopeClient
}

func NewQuizHandler(db *gorm.DB, rdb *redis.Client, aiClient *ai.DashScopeClient) *QuizHandler {
	return &QuizHandler{
		db:          db,
		rdb:         rdb,
		quizService: services.NewQuizService(db, aiClient),
		aiClient:    aiClient,
	}
}

// GenerateQuiz 根据课程生成练习
// @Summary 根据课程生成练习
// @Description 基于课程内容自动生成练习题
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param courseId path int true "课程ID"
// @Param request body services.GenerateQuizRequest true "生成练习请求"
// @Success 200 {object} utils.Response{data=models.Quiz}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/quiz/generate/{courseId} [post]
func (h *QuizHandler) GenerateQuiz(c *gin.Context) {
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

	// 解析请求参数
	var req services.GenerateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 生成练习
	quiz, err := h.quizService.GenerateQuiz(uint(courseID), userID.(uint), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "生成练习失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "练习生成成功", quiz)
}

// GetQuiz 获取练习详情
// @Summary 获取练习详情
// @Description 获取指定练习的详细信息
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param quizId path int true "练习ID"
// @Success 200 {object} utils.Response{data=models.Quiz}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/quiz/{quizId} [get]
func (h *QuizHandler) GetQuiz(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析练习ID
	quizIDStr := c.Param("id")
	quizID, err := strconv.ParseUint(quizIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的练习ID")
		return
	}

	// 获取练习
	quiz, err := h.quizService.GetQuizByID(uint(quizID), userID.(uint))
	if err != nil {
		if err.Error() == "练习不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取练习失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "获取练习成功", quiz)
}

// StartQuiz 开始练习
// @Summary 开始练习
// @Description 开始一个练习，创建答题记录
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param quizId path int true "练习ID"
// @Success 200 {object} utils.Response{data=models.QuizAttempt}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/quiz/{quizId}/start [post]
func (h *QuizHandler) StartQuiz(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析练习ID
	quizIDStr := c.Param("id")
	quizID, err := strconv.ParseUint(quizIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的练习ID")
		return
	}

	// 开始练习
	attempt, err := h.quizService.StartQuiz(uint(quizID), userID.(uint))
	if err != nil {
		if err.Error() == "练习不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "开始练习失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "开始练习成功", attempt)
}

// SubmitAnswer 提交答案
// @Summary 提交答案
// @Description 提交单个题目的答案
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param attemptId path int true "答题记录ID"
// @Param request body services.SubmitAnswerRequest true "提交答案请求"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/quiz/attempts/{attemptId}/answer [post]
func (h *QuizHandler) SubmitAnswer(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析答题记录ID
	attemptIDStr := c.Param("id")
	attemptID, err := strconv.ParseUint(attemptIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的答题记录ID")
		return
	}

	// 先读取原始请求体用于调试
	body, _ := io.ReadAll(c.Request.Body)
	log.Printf("🔍 SubmitAnswer 原始请求体: %s", string(body))

	// 重新设置请求体供后续解析
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	// 解析请求参数
	var req services.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSON解析失败: %v", err)
		log.Printf("❌ 请求体内容: %s", string(body))
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	log.Printf("✅ 解析成功的请求: %+v", req)

	// 提交答案
	result, err := h.quizService.SubmitAnswer(uint(attemptID), userID.(uint), req)
	if err != nil {
		if err.Error() == "答题记录不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "提交答案失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "提交答案成功", result)
}

// SubmitQuiz 提交练习
// @Summary 提交练习
// @Description 完成练习并计算得分
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param attemptId path int true "答题记录ID"
// @Success 200 {object} utils.Response{data=services.QuizResult}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/quiz/attempts/{attemptId}/submit [post]
func (h *QuizHandler) SubmitQuiz(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析答题记录ID
	attemptIDStr := c.Param("id")
	attemptID, err := strconv.ParseUint(attemptIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的答题记录ID")
		return
	}

	// 提交练习
	result, err := h.quizService.SubmitQuiz(uint(attemptID), userID.(uint))
	if err != nil {
		if err.Error() == "答题记录不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "提交练习失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "提交练习成功", result)
}

// GetAttempt 获取答题记录
// @Summary 获取答题记录
// @Description 获取指定的答题记录详情
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param attemptId path int true "答题记录ID"
// @Success 200 {object} utils.Response{data=models.QuizAttempt}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/quiz/attempts/{attemptId} [get]
func (h *QuizHandler) GetAttempt(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析答题记录ID
	attemptIDStr := c.Param("id")
	attemptID, err := strconv.ParseUint(attemptIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的答题记录ID")
		return
	}

	// 获取答题记录
	attempt, err := h.quizService.GetAttemptByID(uint(attemptID), userID.(uint))
	if err != nil {
		if err.Error() == "答题记录不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取答题记录失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "获取答题记录成功", attempt)
}

// GetQuizStats 获取练习统计
// @Summary 获取练习统计
// @Description 获取练习的统计信息
// @Tags 练习系统
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param quizId path int true "练习ID"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/quiz/{quizId}/stats [get]
func (h *QuizHandler) GetQuizStats(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析练习ID
	quizIDStr := c.Param("id")
	quizID, err := strconv.ParseUint(quizIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的练习ID")
		return
	}

	// 获取练习统计
	stats, err := h.quizService.GetQuizStats(uint(quizID), userID.(uint))
	if err != nil {
		if err.Error() == "练习不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取练习统计失败: "+err.Error())
		}
		return
	}

	utils.SuccessResponse(c, "获取练习统计成功", stats)
}

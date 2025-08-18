package handlers

import (
	"ai-classroom/internal/services"
	"ai-classroom/pkg/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserHandler struct {
	db          *gorm.DB
	rdb         *redis.Client
	userService services.UserService
}

func NewUserHandler(db *gorm.DB, rdb *redis.Client, userService services.UserService) *UserHandler {
	return &UserHandler{
		db:          db,
		rdb:         rdb,
		userService: userService,
	}
}

// GetProfile 获取用户资料
// @Summary 获取用户资料
// @Description 获取当前用户的详细资料信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utils.Response{data=models.User}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取用户资料
	user, err := h.userService.GetUserProfile(userID.(uint))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户资料失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "获取用户资料成功", user)
}

// UpdateProfile 更新用户资料
// @Summary 更新用户资料
// @Description 更新当前用户的资料信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param profile body services.UpdateUserProfileRequest true "用户资料信息"
// @Success 200 {object} utils.Response{data=models.User}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析请求参数
	var req services.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 更新用户资料
	user, err := h.userService.UpdateUserProfile(userID.(uint), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "更新用户资料失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "更新用户资料成功", user)
}

// GetStats 获取用户统计信息
// @Summary 获取用户统计信息
// @Description 获取当前用户的统计信息，包括课程数、学习时长等
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utils.Response{data=services.UserStatsResponse}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/stats [get]
func (h *UserHandler) GetStats(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 获取用户统计信息
	stats, err := h.userService.GetUserStats(userID.(uint))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户统计信息失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "获取用户统计信息成功", stats)
}

// GetUsage 获取用户使用记录
// @Summary 获取用户使用记录
// @Description 获取当前用户的使用记录，包括课程生成、积分消费等
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param type query string false "使用类型" Enums(course_generation,tts_generation,quiz_generation)
// @Param start_date query string false "开始日期" format(date)
// @Param end_date query string false "结束日期" format(date)
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(50)
// @Success 200 {object} utils.Response{data=services.GetUserUsageResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/usage [get]
func (h *UserHandler) GetUsage(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析查询参数
	req := services.GetUserUsageRequest{
		Type:  c.Query("type"),
		Page:  1,
		Limit: 20,
	}

	// 解析页码
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	// 解析每页数量
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	// 解析开始日期
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			req.StartDate = &startDate
		}
	}

	// 解析结束日期
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			req.EndDate = &endDate
		}
	}

	// 获取用户使用记录
	usage, err := h.userService.GetUserUsage(userID.(uint), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户使用记录失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "获取用户使用记录成功", usage)
}

// ConsumeCredits 消费积分
// @Summary 消费积分
// @Description 消费用户积分
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param credits body object{amount=int} true "积分数量"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/credits/consume [post]
func (h *UserHandler) ConsumeCredits(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析请求参数
	var req struct {
		Amount int `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 消费积分
	if err := h.userService.ConsumeCredits(userID.(uint), req.Amount); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "消费积分失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "消费积分成功", nil)
}

// AddCredits 增加积分
// @Summary 增加积分
// @Description 增加用户积分（管理员接口）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param credits body object{amount=int} true "积分数量"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/credits/add [post]
func (h *UserHandler) AddCredits(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未认证")
		return
	}

	// 解析请求参数
	var req struct {
		Amount int `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 增加积分
	if err := h.userService.AddCredits(userID.(uint), req.Amount); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "增加积分失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "增加积分成功", nil)
}

// GetUserInfo 获取指定用户信息（管理员接口）
// @Summary 获取指定用户信息
// @Description 管理员获取指定用户的详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "用户ID"
// @Success 200 {object} utils.Response{data=models.User}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	// 获取用户ID参数
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户ID无效")
		return
	}

	// 获取用户信息
	user, err := h.userService.GetUserByID(uint(userID))
	if err != nil {
		if err.Error() == "用户不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, "用户不存在")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "获取用户信息成功", user)
}

// UpdateUserVIP 更新用户VIP状态（管理员接口）
// @Summary 更新用户VIP状态
// @Description 管理员更新用户的VIP状态
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "用户ID"
// @Param vip body object{vip_level=int,expired_at=string} true "VIP信息"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id}/vip [put]
func (h *UserHandler) UpdateUserVIP(c *gin.Context) {
	// 获取用户ID参数
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户ID无效")
		return
	}

	// 解析请求参数
	var req struct {
		VipLevel  int    `json:"vip_level" binding:"required,gte=0,lte=2"`
		ExpiredAt string `json:"expired_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	// 解析过期时间
	var expiredAt *time.Time
	if req.ExpiredAt != "" {
		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", req.ExpiredAt); err == nil {
			expiredAt = &parsedTime
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "过期时间格式无效")
			return
		}
	}

	// 更新用户VIP状态
	if err := h.userService.UpdateUserVIP(uint(userID), req.VipLevel, expiredAt); err != nil {
		if err.Error() == "用户不存在" {
			utils.ErrorResponse(c, http.StatusNotFound, "用户不存在")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新用户VIP状态失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "更新用户VIP状态成功", nil)
}

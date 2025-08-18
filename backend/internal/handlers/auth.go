package handlers

import (
	"ai-classroom/internal/middleware"
	"ai-classroom/internal/services"
	"ai-classroom/pkg/auth"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db          *gorm.DB
	rdb         *redis.Client
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *gorm.DB, rdb *redis.Client) *AuthHandler {
	// 从配置文件读取微信小程序配置
	appID := viper.GetString("wechat.app_id")
	appSecret := viper.GetString("wechat.app_secret")
	appMode := viper.GetString("app.mode")

	var wechatClient auth.WechatClientInterface

	// 环境判断：开发环境可选择使用Mock，生产环境必须使用真实客户端
	if appMode == "development" && (appID == "" || appSecret == "") {
		// 开发环境且未配置微信参数时使用Mock（仅用于功能测试）
		log.Printf("⚠️  开发环境使用Mock微信客户端，不同code将生成不同测试用户")
		log.Printf("🔍 微信配置状态: AppID=%s, AppSecret=%s", appID, appSecret)
		wechatClient = auth.NewMockWechatClient()
	} else if appID != "" && appSecret != "" {
		// 配置完整时使用真实微信客户端
		log.Printf("✅ 使用真实微信客户端，AppID: %s", appID)
		wechatClient = auth.NewWechatClient(appID, appSecret)
	} else {
		// 配置不完整时报错
		log.Fatalf("❌ 微信小程序配置不完整，无法初始化认证处理器")
		log.Fatalf("🔍 当前配置: AppID=%s, AppSecret=%s, Mode=%s", appID, appSecret, appMode)
		log.Fatalf("💡 请检查配置文件 configs/config.yaml 中的 wechat.app_id 和 wechat.app_secret")
	}

	authService := services.NewAuthService(db, rdb, wechatClient)

	return &AuthHandler{
		db:          db,
		rdb:         rdb,
		authService: authService,
	}
}

// WechatLogin 微信登录
func (h *AuthHandler) WechatLogin(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 调用认证服务进行登录
	response, err := h.authService.WechatLogin(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "登录失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data":    response,
	})
}

// RefreshToken 刷新token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req services.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 调用认证服务刷新token
	tokenPair, err := h.authService.RefreshToken(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Token刷新失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Token刷新成功",
		"data": gin.H{
			"token": tokenPair,
		},
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从请求中提取token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "登出成功",
			"data":    nil,
		})
		return
	}

	token, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "登出成功",
			"data":    nil,
		})
		return
	}

	// 调用认证服务进行登出
	if err := h.authService.Logout(token); err != nil {
		// 即使登出失败也返回成功，因为客户端已经丢弃token
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "登出成功",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登出成功",
		"data":    nil,
	})
}

// GetProfile 获取用户信息（需要认证）
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未找到用户信息",
			"data":    nil,
		})
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    user,
	})
}

// UpdateProfile 更新用户资料（需要认证）
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未找到用户信息",
			"data":    nil,
		})
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 调用认证服务更新用户资料
	if err := h.authService.UpdateUserProfile(userID, req.Nickname, req.Phone, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    nil,
	})
}

// GetVIPStatus 获取VIP状态（需要认证）
func (h *AuthHandler) GetVIPStatus(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未找到用户信息",
			"data":    nil,
		})
		return
	}

	isVIP, expiredAt, err := h.authService.CheckVIPStatus(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"is_vip":     isVIP,
			"expired_at": expiredAt,
		},
	})
}

// GetCredits 获取用户积分（需要认证）
func (h *AuthHandler) GetCredits(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未找到用户信息",
			"data":    nil,
		})
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"credits": user.Credits,
		},
	})
}

// ConsumeCredits 消费积分（需要认证）
func (h *AuthHandler) ConsumeCredits(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未找到用户信息",
			"data":    nil,
		})
		return
	}

	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.authService.ConsumeCredits(userID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "积分消费失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "积分消费成功",
		"data":    nil,
	})
}

// ValidateToken 验证token（工具接口）
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			if t, err := auth.ExtractTokenFromHeader(authHeader); err == nil {
				token = t
			}
		}
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少token参数",
			"data":    nil,
		})
		return
	}

	user, err := h.authService.ValidateTokenAndGetUser(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Token验证失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Token验证成功",
		"data": gin.H{
			"valid": true,
			"user":  user,
		},
	})
}

// GetUserByID 根据ID获取用户信息（管理员接口）
func (h *AuthHandler) GetUserByID(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的用户ID",
			"data":    nil,
		})
		return
	}

	user, err := h.authService.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"user": user,
		},
	})
}

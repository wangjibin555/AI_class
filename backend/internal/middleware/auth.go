package middleware

import (
	"ai-classroom/pkg/auth"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AuthRequired 必需认证中间件 - 所有用户必须登录
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := validateAndExtractClaims(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证失败：" + err.Error(),
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 将用户信息注入上下文
		injectUserInfo(c, claims)
		c.Next()
	}
}

// AuthOptional 可选认证中间件 - 用户可登录也可不登录
func AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := validateAndExtractClaims(c)
		if err == nil {
			// 如果有有效token，则注入用户信息
			injectUserInfo(c, claims)
		}
		// 无论是否有token都继续处理
		c.Next()
	}
}

// VIPRequired VIP认证中间件 - 必须是VIP用户
func VIPRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := validateAndExtractClaims(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证失败：" + err.Error(),
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查VIP等级
		if claims.VipLevel == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "需要VIP会员权限",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 将用户信息注入上下文
		injectUserInfo(c, claims)
		c.Next()
	}
}

// validateAndExtractClaims 验证并提取JWT claims
func validateAndExtractClaims(c *gin.Context) (*auth.Claims, error) {
	// 尝试从多个来源提取token
	token := extractToken(c)
	if token == "" {
		return nil, fmt.Errorf("未找到认证token")
	}

	// 验证token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// extractToken 从请求中提取token
func extractToken(c *gin.Context) string {
	// 1. 尝试从Authorization头中提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if token, err := auth.ExtractTokenFromHeader(authHeader); err == nil {
			return token
		}
	}

	// 2. 尝试从查询参数中提取
	if token := c.Query("token"); token != "" {
		return token
	}

	// 3. 尝试从Cookie中提取
	if token, err := c.Cookie("access_token"); err == nil && token != "" {
		return token
	}

	return ""
}

// injectUserInfo 将用户信息注入到上下文中
func injectUserInfo(c *gin.Context, claims *auth.Claims) {
	c.Set("user_id", claims.UserID)
	c.Set("openid", claims.OpenID)
	c.Set("nickname", claims.Nickname)
	c.Set("vip_level", claims.VipLevel)
	c.Set("is_authenticated", true)
}

// GetCurrentUserID 获取当前用户ID
func GetCurrentUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// GetCurrentUserIDString 获取当前用户ID的字符串形式
func GetCurrentUserIDString(c *gin.Context) string {
	userID := GetCurrentUserID(c)
	if userID == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(userID), 10)
}

// GetCurrentOpenID 获取当前用户OpenID
func GetCurrentOpenID(c *gin.Context) string {
	if openID, exists := c.Get("openid"); exists {
		if id, ok := openID.(string); ok {
			return id
		}
	}
	return ""
}

// GetCurrentNickname 获取当前用户昵称
func GetCurrentNickname(c *gin.Context) string {
	if nickname, exists := c.Get("nickname"); exists {
		if name, ok := nickname.(string); ok {
			return name
		}
	}
	return ""
}

// GetCurrentVipLevel 获取当前用户VIP等级
func GetCurrentVipLevel(c *gin.Context) int {
	if vipLevel, exists := c.Get("vip_level"); exists {
		if level, ok := vipLevel.(int); ok {
			return level
		}
	}
	return 0
}

// IsAuthenticated 检查是否已认证
func IsAuthenticated(c *gin.Context) bool {
	if authenticated, exists := c.Get("is_authenticated"); exists {
		if auth, ok := authenticated.(bool); ok {
			return auth
		}
	}
	return false
}

// IsVIP 检查是否为VIP用户
func IsVIP(c *gin.Context) bool {
	return GetCurrentVipLevel(c) > 0
}

// RequireUserID 确保获取到用户ID，否则返回错误
func RequireUserID(c *gin.Context) (uint, bool) {
	userID := GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未找到用户信息",
			"data":    nil,
		})
		return 0, false
	}
	return userID, true
}

// ValidateUserAccess 验证用户是否有权限访问指定资源
func ValidateUserAccess(c *gin.Context, resourceUserID uint) bool {
	currentUserID := GetCurrentUserID(c)
	if currentUserID == 0 {
		return false
	}

	// 用户只能访问自己的资源
	return currentUserID == resourceUserID
}

// RateLimiter 简单的限流中间件
func RateLimiter() gin.HandlerFunc {
	// TODO: 实现Redis基础的限流逻辑
	return func(c *gin.Context) {
		// 暂时不实现限流，直接通过
		c.Next()
	}
}

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

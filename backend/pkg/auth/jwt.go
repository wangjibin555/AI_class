package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT密钥，生产环境应从配置文件读取
var jwtSecret = []byte("ai-classroom-super-secret-key-change-in-production-2024")

// Claims JWT载荷结构
type Claims struct {
	UserID   uint   `json:"user_id"`
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	VipLevel int    `json:"vip_level"`
	jwt.RegisteredClaims
}

// TokenPair token对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // 访问token过期时间（秒）
	RefreshIn    int64  `json:"refresh_in"` // 刷新token过期时间（秒）
}

// SetSecret 设置JWT密钥
func SetSecret(secret string) {
	jwtSecret = []byte(secret)
}

// GenerateToken 生成访问token
func GenerateToken(userID uint, openID, nickname string, vipLevel int) (string, error) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // 24小时过期

	claims := Claims{
		UserID:   userID,
		OpenID:   openID,
		Nickname: nickname,
		VipLevel: vipLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ai-classroom",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateRefreshToken 生成刷新token
func GenerateRefreshToken(userID uint, openID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7天过期

	claims := Claims{
		UserID: userID,
		OpenID: openID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ai-classroom",
			Subject:   fmt.Sprintf("refresh:%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateTokenPair 生成token对
func GenerateTokenPair(userID uint, openID, nickname string, vipLevel int) (*TokenPair, error) {
	accessToken, err := GenerateToken(userID, openID, nickname, vipLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := GenerateRefreshToken(userID, openID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 60 * 60,     // 24小时（秒）
		RefreshIn:    7 * 24 * 60 * 60, // 7天（秒）
	}, nil
}

// ValidateToken 验证token
func ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token has expired")
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, errors.New("token not valid yet")
		}
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// RefreshToken 刷新token
func RefreshToken(refreshTokenString string, nickname string, vipLevel int) (*TokenPair, error) {
	// 验证刷新token
	claims, err := ValidateToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// 检查是否为刷新token
	if claims.Subject != fmt.Sprintf("refresh:%d", claims.UserID) {
		return nil, errors.New("not a refresh token")
	}

	// 生成新的token对
	return GenerateTokenPair(claims.UserID, claims.OpenID, nickname, vipLevel)
}

// ExtractTokenFromHeader 从Authorization头中提取token
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is empty")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", errors.New("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("token is empty")
	}

	return token, nil
}

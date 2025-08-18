package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/pkg/auth"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db           *gorm.DB
	rdb          *redis.Client
	wechatClient WechatClientInterface
	jwtConfig    JWTConfig
}

// WechatClientInterface 微信客户端接口
type WechatClientInterface interface {
	GetSession(code string) (*auth.WechatSession, error)
	ValidateUserInfo(encryptedData, iv, sessionKey string) (*auth.WechatUserInfo, error)
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret       string
	ExpiresHours int
}

// LoginRequest 登录请求
type LoginRequest struct {
	Code          string `json:"code" binding:"required"`
	EncryptedData string `json:"encrypted_data,omitempty"`
	IV            string `json:"iv,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token        *auth.TokenPair `json:"token"`
	User         *models.User    `json:"user"`
	IsNewUser    bool            `json:"is_new_user"`
	NeedsProfile bool            `json:"needs_profile"`
}

// RefreshTokenRequest 刷新token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// NewAuthService 创建认证服务
func NewAuthService(db *gorm.DB, rdb *redis.Client, wechatClient WechatClientInterface) *AuthService {
	// 设置JWT密钥
	auth.SetSecret("ai-classroom-super-secret-key-change-in-production-2024")

	return &AuthService{
		db:           db,
		rdb:          rdb,
		wechatClient: wechatClient,
		jwtConfig: JWTConfig{
			Secret:       "ai-classroom-super-secret-key-change-in-production-2024",
			ExpiresHours: 24,
		},
	}
}

// WechatLogin 微信登录
func (s *AuthService) WechatLogin(req *LoginRequest) (*LoginResponse, error) {
	// 1. 通过code获取session
	session, err := s.wechatClient.GetSession(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get wechat session: %w", err)
	}

	// 2. 查找或创建用户
	user, isNewUser, err := s.findOrCreateUser(session, req)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// 3. 生成JWT token
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.OpenID, user.Nickname, user.VipLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 4. 更新最后登录时间
	if err := s.updateLastLogin(user.ID); err != nil {
		// 记录错误但不影响登录
		fmt.Printf("Failed to update last login time: %v\n", err)
	}

	return &LoginResponse{
		Token:        tokenPair,
		User:         user,
		IsNewUser:    isNewUser,
		NeedsProfile: user.Nickname == "",
	}, nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(req *RefreshTokenRequest) (*auth.TokenPair, error) {
	// 验证refresh token并获取用户信息
	claims, err := auth.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// 从数据库获取最新用户信息
	var user models.User
	if err := s.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 生成新的token对
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.OpenID, user.Nickname, user.VipLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	return tokenPair, nil
}

// findOrCreateUser 查找或创建用户
func (s *AuthService) findOrCreateUser(session *auth.WechatSession, req *LoginRequest) (*models.User, bool, error) {
	var user models.User

	log.Printf("🔍 查找用户，OpenID: %s", session.OpenID)

	// 首先尝试查找现有用户
	err := s.db.Where("open_id = ?", session.OpenID).First(&user).Error
	if err == nil {
		// 用户已存在，更新用户信息（如果提供了加密数据）
		log.Printf("✅ 找到现有用户，ID: %d, 昵称: %s, OpenID: %s", user.ID, user.Nickname, user.OpenID)

		if req.EncryptedData != "" && req.IV != "" {
			if err := s.updateUserInfoFromWechat(&user, session.SessionKey, req); err != nil {
				// 记录错误但不影响登录
				log.Printf("⚠️ 更新用户信息失败: %v", err)
			} else {
				log.Printf("✅ 用户信息更新成功，昵称: %s", user.Nickname)
			}
		}
		return &user, false, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ 数据库查询错误: %v", err)
		return nil, false, fmt.Errorf("database error: %w", err)
	}

	log.Printf("🆕 OpenID不存在，开始创建新用户: %s", session.OpenID)

	// 用户不存在，创建新用户
	user = models.User{
		OpenID:              session.OpenID,
		VipLevel:            0,
		Credits:             100, // 新用户赠送100积分
		TotalCoursesCreated: 0,
		TotalStudyTime:      0,
	}

	// 如果提供了加密的用户信息，则解密并填充
	if req.EncryptedData != "" && req.IV != "" {
		if err := s.updateUserInfoFromWechat(&user, session.SessionKey, req); err != nil {
			// 记录错误但不影响用户创建
			log.Printf("⚠️ 解密用户信息失败: %v", err)
		} else {
			log.Printf("✅ 成功解密用户信息，昵称: %s", user.Nickname)
		}
	}

	// 保存用户到数据库
	if err := s.db.Create(&user).Error; err != nil {
		log.Printf("❌ 创建用户失败: %v", err)
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("🎉 用户创建成功！ID: %d, OpenID: %s, 昵称: %s, 积分: %d",
		user.ID, user.OpenID, user.Nickname, user.Credits)

	return &user, true, nil
}

// updateUserInfoFromWechat 从微信更新用户信息
func (s *AuthService) updateUserInfoFromWechat(user *models.User, sessionKey string, req *LoginRequest) error {
	userInfo, err := s.wechatClient.ValidateUserInfo(req.EncryptedData, req.IV, sessionKey)
	if err != nil {
		return err
	}

	// 更新用户信息
	user.Nickname = userInfo.NickName
	user.AvatarURL = userInfo.AvatarURL

	// 保存更新
	if user.ID != 0 {
		return s.db.Save(user).Error
	}

	return nil
}

// updateLastLogin 更新最后登录时间
func (s *AuthService) updateLastLogin(userID uint) error {
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("updated_at", time.Now()).Error
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// GetUserByOpenID 根据OpenID获取用户
func (s *AuthService) GetUserByOpenID(openID string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("open_id = ?", openID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// UpdateUserProfile 更新用户资料
func (s *AuthService) UpdateUserProfile(userID uint, nickname, phone, email string) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if nickname != "" {
		updates["nickname"] = nickname
	}
	if phone != "" {
		updates["phone"] = phone
	}
	if email != "" {
		updates["email"] = email
	}

	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

// ValidateTokenAndGetUser 验证token并获取用户信息
func (s *AuthService) ValidateTokenAndGetUser(tokenString string) (*models.User, error) {
	// 验证token
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// 获取用户信息
	user, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// Logout 登出（可选：将token加入黑名单）
func (s *AuthService) Logout(tokenString string) error {
	// 这里可以将token加入Redis黑名单
	// 暂时简单实现，实际项目中应该实现token黑名单机制
	return nil
}

// CheckVIPStatus 检查VIP状态
func (s *AuthService) CheckVIPStatus(userID uint) (bool, *time.Time, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false, nil, err
	}

	return user.IsVIP(), user.VipExpiredAt, nil
}

// ConsumeCredits 消费积分
func (s *AuthService) ConsumeCredits(userID uint, amount int) error {
	result := s.db.Model(&models.User{}).
		Where("id = ? AND credits >= ?", userID, amount).
		Update("credits", gorm.Expr("credits - ?", amount))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient credits or user not found")
	}

	return nil
}

// AddCredits 增加积分
func (s *AuthService) AddCredits(userID uint, amount int) error {
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("credits", gorm.Expr("credits + ?", amount)).Error
}

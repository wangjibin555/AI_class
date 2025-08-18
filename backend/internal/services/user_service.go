package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/internal/repositories"
	"fmt"
	"strings"
	"time"
)

// UserService 用户服务接口
type UserService interface {
	GetUserProfile(userID uint) (*models.User, error)
	UpdateUserProfile(userID uint, req UpdateUserProfileRequest) (*models.User, error)
	GetUserStats(userID uint) (*UserStatsResponse, error)
	GetUserUsage(userID uint, req GetUserUsageRequest) (*GetUserUsageResponse, error)
	ConsumeCredits(userID uint, amount int) error
	DeductCredits(userID uint, amount int) error
	AddCredits(userID uint, amount int) error
	GetUserByID(userID uint) (*models.User, error)
	UpdateUserVIP(userID uint, vipLevel int, expiredAt *time.Time) error
}

// userService 用户服务实现
type userService struct {
	userRepo repositories.UserRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

// UpdateUserProfileRequest 更新用户资料请求
type UpdateUserProfileRequest struct {
	Nickname  *string `json:"nickname"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	CoursesCreated    int        `json:"courses_created"`
	CoursesCompleted  int        `json:"courses_completed"`
	TotalStudyTime    int        `json:"total_study_time"`
	QuizAverageScore  float64    `json:"quiz_average_score"`
	QuizTotalAttempts int        `json:"quiz_total_attempts"`
	ShareCount        int        `json:"share_count"`
	BookmarksCount    int        `json:"bookmarks_count"`
	RankPercentile    float64    `json:"rank_percentile"`
	CurrentVipLevel   int        `json:"current_vip_level"`
	VipLevelName      string     `json:"vip_level_name"`
	VipExpiredAt      *time.Time `json:"vip_expired_at"`
	Credits           int        `json:"credits"`
	DailyQuota        int        `json:"daily_quota"`
}

// GetUserUsageRequest 获取用户使用记录请求
type GetUserUsageRequest struct {
	Type      string     `json:"type"` // course_generation, tts_generation, quiz_generation
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
}

// GetUserUsageResponse 获取用户使用记录响应
type GetUserUsageResponse struct {
	Total   int64                 `json:"total"`
	Page    int                   `json:"page"`
	Limit   int                   `json:"limit"`
	Records []UsageRecordResponse `json:"records"`
}

// UsageRecordResponse 使用记录响应
type UsageRecordResponse struct {
	ID              uint      `json:"id"`
	Type            string    `json:"type"`
	ResourceID      uint      `json:"resource_id"`
	ResourceTitle   string    `json:"resource_title"`
	ConsumedCredits int       `json:"consumed_credits"`
	ConsumedAt      time.Time `json:"consumed_at"`
	Status          string    `json:"status"`
	ErrorMessage    string    `json:"error_message"`
}

// GetUserProfile 获取用户资料
func (s *userService) GetUserProfile(userID uint) (*models.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户资料失败: %w", err)
	}

	return user, nil
}

// UpdateUserProfile 更新用户资料
func (s *userService) UpdateUserProfile(userID uint, req UpdateUserProfileRequest) (*models.User, error) {
	// 验证请求参数
	if err := s.validateUpdateProfileRequest(req); err != nil {
		return nil, err
	}

	// 获取现有用户信息
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 更新用户信息
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}

	// 保存更新
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("更新用户资料失败: %w", err)
	}

	return user, nil
}

// GetUserStats 获取用户统计信息
func (s *userService) GetUserStats(userID uint) (*UserStatsResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取统计数据
	stats, err := s.userRepo.GetUserStats(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户统计失败: %w", err)
	}

	response := &UserStatsResponse{
		CoursesCreated:    stats.CoursesCreated,
		CoursesCompleted:  stats.CoursesCompleted,
		TotalStudyTime:    user.TotalStudyTime,
		QuizAverageScore:  stats.QuizAverageScore,
		QuizTotalAttempts: stats.QuizTotalAttempts,
		ShareCount:        stats.ShareCount,
		BookmarksCount:    stats.BookmarksCount,
		RankPercentile:    stats.RankPercentile,
		CurrentVipLevel:   user.VipLevel,
		VipLevelName:      user.GetVipLevelName(),
		VipExpiredAt:      user.VipExpiredAt,
		Credits:           user.Credits,
		DailyQuota:        user.GetDailyQuota(),
	}

	return response, nil
}

// GetUserUsage 获取用户使用记录
func (s *userService) GetUserUsage(userID uint, req GetUserUsageRequest) (*GetUserUsageResponse, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	offset := (req.Page - 1) * req.Limit

	// 获取使用记录
	records, total, err := s.userRepo.GetUserUsage(userID, req.Type, req.StartDate, req.EndDate, offset, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("获取用户使用记录失败: %w", err)
	}

	// 转换为响应格式
	recordResponses := make([]UsageRecordResponse, len(records))
	for i, record := range records {
		recordResponses[i] = UsageRecordResponse{
			ID:              record.ID,
			Type:            record.Type,
			ResourceID:      record.ResourceID,
			ResourceTitle:   record.ResourceTitle,
			ConsumedCredits: record.ConsumedCredits,
			ConsumedAt:      record.ConsumedAt,
			Status:          record.Status,
			ErrorMessage:    record.ErrorMessage,
		}
	}

	return &GetUserUsageResponse{
		Total:   total,
		Page:    req.Page,
		Limit:   req.Limit,
		Records: recordResponses,
	}, nil
}

// ConsumeCredits 消耗积分
func (s *userService) ConsumeCredits(userID uint, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("消耗积分数量必须大于0")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 检查积分是否足够
	if user.Credits < amount {
		return fmt.Errorf("积分不足，当前积分：%d，需要积分：%d", user.Credits, amount)
	}

	// 扣除积分
	user.Credits -= amount

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("更新用户积分失败: %w", err)
	}

	return nil
}

// DeductCredits 扣除积分（ConsumeCredits的别名）
func (s *userService) DeductCredits(userID uint, amount int) error {
	return s.ConsumeCredits(userID, amount)
}

// AddCredits 增加积分
func (s *userService) AddCredits(userID uint, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("增加积分数量必须大于0")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 增加积分
	user.Credits += amount

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("更新用户积分失败: %w", err)
	}

	return nil
}

// GetUserByID 根据ID获取用户
func (s *userService) GetUserByID(userID uint) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}

// UpdateUserVIP 更新用户VIP状态
func (s *userService) UpdateUserVIP(userID uint, vipLevel int, expiredAt *time.Time) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	user.VipLevel = vipLevel
	user.VipExpiredAt = expiredAt

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("更新用户VIP状态失败: %w", err)
	}

	return nil
}

// validateUpdateProfileRequest 验证更新用户资料请求
func (s *userService) validateUpdateProfileRequest(req UpdateUserProfileRequest) error {
	if req.Nickname != nil && strings.TrimSpace(*req.Nickname) == "" {
		return fmt.Errorf("昵称不能为空")
	}

	if req.Nickname != nil && len(*req.Nickname) > 50 {
		return fmt.Errorf("昵称长度不能超过50个字符")
	}

	if req.Phone != nil && *req.Phone != "" {
		phone := strings.TrimSpace(*req.Phone)
		if len(phone) != 11 {
			return fmt.Errorf("手机号格式不正确")
		}
	}

	if req.Email != nil && *req.Email != "" {
		email := strings.TrimSpace(*req.Email)
		if !strings.Contains(email, "@") {
			return fmt.Errorf("邮箱格式不正确")
		}
	}

	return nil
}

package repositories

import (
	"ai-classroom/internal/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UserRepository 用户数据访问接口
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByOpenID(openID string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	GetUserStats(userID uint) (*UserStats, error)
	GetUserUsage(userID uint, usageType string, startDate, endDate *time.Time, offset, limit int) ([]UsageRecord, int64, error)
	GetUserCount() (int64, error)
	GetActiveUsers(days int) (int64, error)
	GetVIPUsers() (int64, error)
}

// UserStats 用户统计信息
type UserStats struct {
	CoursesCreated    int     `json:"courses_created"`
	CoursesCompleted  int     `json:"courses_completed"`
	QuizAverageScore  float64 `json:"quiz_average_score"`
	QuizTotalAttempts int     `json:"quiz_total_attempts"`
	ShareCount        int     `json:"share_count"`
	BookmarksCount    int     `json:"bookmarks_count"`
	RankPercentile    float64 `json:"rank_percentile"`
}

// UsageRecord 使用记录
type UsageRecord struct {
	ID              uint      `json:"id"`
	UserID          uint      `json:"user_id"`
	Type            string    `json:"type"`
	ResourceID      uint      `json:"resource_id"`
	ResourceTitle   string    `json:"resource_title"`
	ConsumedCredits int       `json:"consumed_credits"`
	ConsumedAt      time.Time `json:"consumed_at"`
	Status          string    `json:"status"`
	ErrorMessage    string    `json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定使用记录表名
func (UsageRecord) TableName() string {
	return "usage_records"
}

// userRepository 用户数据访问实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(user *models.User) error {
	if user == nil {
		return errors.New("用户对象不能为空")
	}

	// 基础验证
	if user.OpenID == "" {
		return errors.New("OpenID不能为空")
	}

	// 检查是否已存在相同OpenID的用户
	var existingUser models.User
	if err := r.db.Where("open_id = ?", user.OpenID).First(&existingUser).Error; err == nil {
		return errors.New("用户已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("检查用户存在性失败: %w", err)
	}

	// 设置默认值
	if user.VipLevel == 0 {
		user.VipLevel = 0
	}
	if user.Credits == 0 {
		user.Credits = 100 // 新用户默认100积分
	}

	result := r.db.Create(user)
	if result.Error != nil {
		return fmt.Errorf("创建用户失败: %w", result.Error)
	}

	return nil
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(id uint) (*models.User, error) {
	if id == 0 {
		return nil, errors.New("用户ID不能为空")
	}

	var user models.User
	result := r.db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("获取用户失败: %w", result.Error)
	}

	return &user, nil
}

// GetByOpenID 根据OpenID获取用户
func (r *userRepository) GetByOpenID(openID string) (*models.User, error) {
	if openID == "" {
		return nil, errors.New("OpenID不能为空")
	}

	var user models.User
	result := r.db.Where("open_id = ?", openID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("获取用户失败: %w", result.Error)
	}

	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(user *models.User) error {
	if user == nil {
		return errors.New("用户对象不能为空")
	}
	if user.ID == 0 {
		return errors.New("用户ID不能为空")
	}

	// 检查用户是否存在
	var existingUser models.User
	if err := r.db.First(&existingUser, user.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return fmt.Errorf("检查用户存在性失败: %w", err)
	}

	result := r.db.Save(user)
	if result.Error != nil {
		return fmt.Errorf("更新用户失败: %w", result.Error)
	}

	return nil
}

// Delete 删除用户
func (r *userRepository) Delete(id uint) error {
	if id == 0 {
		return errors.New("用户ID不能为空")
	}

	// 检查用户是否存在
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return fmt.Errorf("检查用户存在性失败: %w", err)
	}

	result := r.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除用户失败: %w", result.Error)
	}

	return nil
}

// GetUserStats 获取用户统计信息
func (r *userRepository) GetUserStats(userID uint) (*UserStats, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为空")
	}

	stats := &UserStats{}

	// 获取课程创建数
	var coursesCreated int64
	if err := r.db.Model(&models.Course{}).Where("user_id = ?", userID).Count(&coursesCreated).Error; err != nil {
		return nil, fmt.Errorf("获取课程创建数失败: %w", err)
	}
	stats.CoursesCreated = int(coursesCreated)

	// 获取课程完成数
	var coursesCompleted int64
	if err := r.db.Model(&models.Course{}).Where("user_id = ? AND status = ?", userID, "completed").Count(&coursesCompleted).Error; err != nil {
		return nil, fmt.Errorf("获取课程完成数失败: %w", err)
	}
	stats.CoursesCompleted = int(coursesCompleted)

	// 获取练习统计 (这里需要根据实际的练习模型调整)
	var quizAttempts int64
	var avgScore float64

	// 假设有quiz_attempts表
	if err := r.db.Table("quiz_attempts").Where("user_id = ?", userID).Count(&quizAttempts).Error; err != nil {
		// 如果表不存在，设置默认值
		quizAttempts = 0
	}
	stats.QuizTotalAttempts = int(quizAttempts)

	// 计算平均分
	if quizAttempts > 0 {
		if err := r.db.Table("quiz_attempts").Where("user_id = ?", userID).Select("AVG(score)").Scan(&avgScore).Error; err != nil {
			avgScore = 0
		}
	}
	stats.QuizAverageScore = avgScore

	// 获取分享数量 (假设有share_records表)
	var shareCount int64
	if err := r.db.Table("share_records").Where("user_id = ?", userID).Count(&shareCount).Error; err != nil {
		shareCount = 0
	}
	stats.ShareCount = int(shareCount)

	// 获取收藏数量 (假设有learning_records表)
	var bookmarksCount int64
	if err := r.db.Table("learning_records").Where("user_id = ? AND is_bookmarked = ?", userID, true).Count(&bookmarksCount).Error; err != nil {
		bookmarksCount = 0
	}
	stats.BookmarksCount = int(bookmarksCount)

	// 计算排名百分比 (简化实现)
	stats.RankPercentile = 75.0 // 默认75百分位

	return stats, nil
}

// GetUserUsage 获取用户使用记录
func (r *userRepository) GetUserUsage(userID uint, usageType string, startDate, endDate *time.Time, offset, limit int) ([]UsageRecord, int64, error) {
	if userID == 0 {
		return nil, 0, errors.New("用户ID不能为空")
	}

	var records []UsageRecord
	var total int64

	// 构建查询条件
	query := r.db.Model(&UsageRecord{}).Where("user_id = ?", userID)

	// 类型过滤
	if usageType != "" {
		query = query.Where("type = ?", usageType)
	}

	// 时间过滤
	if startDate != nil {
		query = query.Where("consumed_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("consumed_at <= ?", endDate)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取使用记录总数失败: %w", err)
	}

	// 分页查询
	if err := query.Order("consumed_at DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("获取用户使用记录失败: %w", err)
	}

	return records, total, nil
}

// GetUserCount 获取用户总数
func (r *userRepository) GetUserCount() (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取用户总数失败: %w", err)
	}
	return count, nil
}

// GetActiveUsers 获取活跃用户数
func (r *userRepository) GetActiveUsers(days int) (int64, error) {
	var count int64
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 这里需要根据实际的活跃判断逻辑调整
	// 例如：根据最后登录时间、最后创建课程时间等
	if err := r.db.Model(&models.User{}).Where("updated_at >= ?", cutoffTime).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取活跃用户数失败: %w", err)
	}

	return count, nil
}

// GetVIPUsers 获取VIP用户数
func (r *userRepository) GetVIPUsers() (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Where("vip_level > 0 AND (vip_expired_at IS NULL OR vip_expired_at > ?)", time.Now()).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取VIP用户数失败: %w", err)
	}
	return count, nil
}

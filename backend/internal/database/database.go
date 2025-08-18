package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"ai-classroom/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	Charset  string
}

// NewConfig 创建数据库配置
func NewConfig() *Config {
	return &Config{
		Host:     getEnv("DB_HOST", "pc-bp13v2788pv3fa00d.rwlb.rds.aliyuncs.com"),
		Port:     getEnv("DB_PORT", "3306"),
		User:     getEnv("DB_USER", "wangjibin"),
		Password: getEnv("DB_PASSWORD", "nuGM8Vb9QBRhtH#zmM"),
		DBName:   getEnv("DB_NAME", "wangjibin"),
		Charset:  "utf8mb4",
	}
}

// Connect 连接数据库
func Connect(config *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
		config.Charset,
	)

	// 配置GORM日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) error {
	log.Println("开始数据库迁移...")

	// 迁移所有模型
	err := db.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Slide{},
		&models.LearningRecord{},
		&models.AIAssistantChat{},
	)

	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

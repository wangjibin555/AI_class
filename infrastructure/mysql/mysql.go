package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"time"
)

// 相关参数
type MysqlConfig struct {
	//基础配置
	Host     string
	Port     int
	UserName string
	Password string
	Database string
	Charset  string

	//连接池配置
	MaxIdleconns    int           //最大空闲空闲连接数
	MaxOpenConns    int           //最大连接数
	ConnMaxLifetime time.Duration //最大连接生命周期
	ConnMaxIdletime time.Duration //最大空闲连接生命周期

	//超时配置
	ConnTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	//其他配置
	LogLevel      string        //日志级别
	SlowThreshold time.Duration //慢查询阈值
}

// mysql客户端定义
type MysqlClient struct {
	db     *gorm.DB
	ctx    context.Context
	config *MysqlConfig
}

// 创建连接池
func NewMysqlClient(config *MysqlConfig) (*MysqlClient, error) {
	// 设置默认值
	if config.Charset == "" {
		config.Charset = "utf8mb4"
	}
	if config.MaxIdleconns == 0 {
		config.MaxIdleconns = 10
	}
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 100
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = time.Hour
	}
	if config.ConnMaxIdletime == 0 {
		config.ConnMaxIdletime = 10 * time.Minute
	}
	if config.ConnTimeout == 0 {
		config.ConnTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&timeout=%s&readTimeout=%s&writeTimeout=%s",
		config.UserName,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Charset,
		config.ConnTimeout.String(),
		config.ReadTimeout.String(),
		config.WriteTimeout.String(),
	)

	// 添加：日志级别配置
	logLevel := logger.Silent
	switch config.LogLevel {
	case "info":
		logLevel = logger.Info
	case "warn":
		logLevel = logger.Warn
	case "error":
		logLevel = logger.Error
	case "silent":
		logLevel = logger.Silent
	}
	// 修改：使用自定义日志级别
	logger.Default.LogMode(logLevel)

	//打开连接
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, err
	}

	//获取底层sql.DB，配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	//连接池配置
	sqlDB.SetMaxIdleConns(config.MaxIdleconns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdletime)

	// 健康检查
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return &MysqlClient{
		db:     db,
		ctx:    context.Background(),
		config: config,
	}, nil
}

// 连接池健康检查
// Ping 检查数据库连接是否正常
func (c *MysqlClient) Ping() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Stats 获取连接池统计信息
func (c *MysqlClient) Stats() sql.DBStats {
	sqlDB, _ := c.db.DB()
	return sqlDB.Stats()
}

// 关闭连接
// Close 关闭数据库连接
func (c *MysqlClient) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// 辅助方法
// GetDB 获取原始GORM DB对象（用于复杂查询）
func (c *MysqlClient) GetDB() *gorm.DB {
	return c.db
}

// WithContext 使用特定上下文
func (c *MysqlClient) WithContext(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx)
}

// AutoMigrate 自动迁移表结构
func (c *MysqlClient) AutoMigrate(models ...interface{}) error {
	return c.db.AutoMigrate(models...)
}

package mysql

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
	_ "gorm.io/gorm/logger"
)

// User 示例模型
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100"`
	Email     string `gorm:"size:100;uniqueIndex"`
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func TestMySQLClient(t *testing.T) {
	// 创建配置
	config := &MysqlConfig{
		Host:            "localhost",
		Port:            3306,
		UserName:        "root",
		Password:        "xxxxxx",
		Database:        "test_db",
		Charset:         "utf8mb4",
		MaxIdleconns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		ConnTimeout:     5 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		LogLevel:        "info",
	}

	// 创建客户端
	client, err := NewMysqlClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// 自动迁移
	client.AutoMigrate(&User{})

	// 创建记录
	user := &User{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   25,
	}
	err = client.Create(user)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Created user: %+v\n", user)

	// 查询记录
	var foundUser User
	err = client.First(&foundUser, "email = ?", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Found user: %+v\n", foundUser)

	// 更新记录
	err = client.Updates(&User{ID: foundUser.ID}, map[string]interface{}{"age": 26})
	if err != nil {
		t.Fatal(err)
	}

	// 事务示例
	err = client.Transaction(func(tx *gorm.DB) error {
		// 在事务中创建多个记录
		if err := tx.Create(&User{Name: "Bob", Email: "bob@example.com", Age: 30}).Error; err != nil {
			return err
		}
		if err := tx.Create(&User{Name: "Charlie", Email: "charlie@example.com", Age: 35}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 统计数量
	var count int64
	client.Count(&User{}, &count)
	fmt.Printf("Total users: %d\n", count)

	// 分页查询
	var users []User
	client.GetDB().Scopes(client.Paginate(1, 10)).Find(&users)
	fmt.Printf("Users (page 1): %+v\n", users)

	// 检查连接池状态
	stats := client.Stats()
	fmt.Printf("DB Stats: %+v\n", stats)
}

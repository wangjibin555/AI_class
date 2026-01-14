package mysql

import (
	"context"

	"gorm.io/gorm"
)

// TxFunc 事务函数类型
type TxFunc func(*gorm.DB) error

// Transaction 执行事务
func (c *MysqlClient) Transaction(fn TxFunc) error {
	return c.db.Transaction(fn)
}

// TransactionWithContext 带上下文的事务
func (c *MysqlClient) TransactionWithContext(ctx context.Context, fn TxFunc) error {
	return c.db.WithContext(ctx).Transaction(fn)
}

// BeginTx 手动开启事务
func (c *MysqlClient) BeginTx(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx).Begin()
}

// Commit 提交事务
func Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

// Rollback 回滚事务
func Rollback(tx *gorm.DB) error {
	return tx.Rollback().Error
}

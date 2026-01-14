package mysql

import "gorm.io/gorm"

// Create 创建记录
func (c *MysqlClient) Create(value interface{}) error {
	return c.db.Create(value).Error
}

// Save 保存记录（更新所有字段）
func (c *MysqlClient) Save(value interface{}) error {
	return c.db.Save(value).Error
}

// Update 更新记录（只更新非零字段）
func (c *MysqlClient) Update(model interface{}, column string, value interface{}) error {
	return c.db.Model(model).Update(column, value).Error
}

// Updates 批量更新（只更新非零字段）
func (c *MysqlClient) Updates(model interface{}, updates interface{}) error {
	return c.db.Model(model).Updates(updates).Error
}

// Delete 删除记录
func (c *MysqlClient) Delete(model interface{}, conditions ...interface{}) error {
	return c.db.Delete(model, conditions...).Error
}

// First 查询第一条记录
func (c *MysqlClient) First(dest interface{}, conditions ...interface{}) error {
	return c.db.First(dest, conditions...).Error
}

// Find 查询多条记录
func (c *MysqlClient) Find(dest interface{}, conditions ...interface{}) error {
	return c.db.Find(dest, conditions...).Error
}

// Where 条件查询
func (c *MysqlClient) Where(query interface{}, args ...interface{}) *gorm.DB {
	return c.db.Where(query, args...)
}

// Count 统计记录数
func (c *MysqlClient) Count(model interface{}, count *int64) error {
	return c.db.Model(model).Count(count).Error
}

// Paginate 分页查询
func (c *MysqlClient) Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// Raw 执行原生SQL查询
func (c *MysqlClient) Raw(sql string, values ...interface{}) *gorm.DB {
	return c.db.Raw(sql, values...)
}

// Exec 执行原生SQL（增删改）
func (c *MysqlClient) Exec(sql string, values ...interface{}) error {
	return c.db.Exec(sql, values...).Error
}

// FindOne 查询单条记录
func (c *MysqlClient) FindOne(dest interface{}, conditions ...interface{}) error {
	return c.db.First(dest, conditions...).Error
}

// FindAll 查询多条记录
func (c *MysqlClient) FindAll(dest interface{}, conditions ...interface{}) error {
	return c.db.Find(dest, conditions...).Error
}

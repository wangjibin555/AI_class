package redis

import "github.com/gomodule/redigo/redis"

// ==================== Hash 操作 ====================

// HSet 设置哈希表字段值
func (c *RedisClient) HSet(key, field string, value interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HSET", key, field, value)
	return err
}

// HGet 获取哈希表字段值
func (c *RedisClient) HGet(key, field string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("HGET", key, field))
}

// HGetAll 获取哈希表所有字段和值
func (c *RedisClient) HGetAll(key string) (map[string]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.StringMap(conn.Do("HGETALL", key))
}

// HMSet 批量设置哈希表字段
func (c *RedisClient) HMSet(key string, fields map[string]interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, 0, len(fields)*2+1)
	args = append(args, key)
	for field, value := range fields {
		args = append(args, field, value)
	}

	_, err := conn.Do("HMSET", args...)
	return err
}

// HMGet 批量获取哈希表字段值
func (c *RedisClient) HMGet(key string, fields ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(fields)+1)
	args[0] = key
	for i, field := range fields {
		args[i+1] = field
	}

	return redis.Strings(conn.Do("HMGET", args...))
}

// HDel 删除哈希表字段
func (c *RedisClient) HDel(key string, fields ...string) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(fields)+1)
	args[0] = key
	for i, field := range fields {
		args[i+1] = field
	}

	return redis.Int(conn.Do("HDEL", args...))
}

// HExists 检查哈希表字段是否存在
func (c *RedisClient) HExists(key, field string) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()

	exists, err := redis.Int(conn.Do("HEXISTS", key, field))
	return exists == 1, err
}

// HLen 获取哈希表字段数量
func (c *RedisClient) HLen(key string) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("HLEN", key))
}

// HKeys 获取哈希表所有字段名
func (c *RedisClient) HKeys(key string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("HKEYS", key))
}

// HVals 获取哈希表所有值
func (c *RedisClient) HVals(key string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("HVALS", key))
}

// HIncrBy 哈希表字段值增加指定整数
func (c *RedisClient) HIncrBy(key, field string, increment int64) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("HINCRBY", key, field, increment))
}

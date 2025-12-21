package redis

import "github.com/gomodule/redigo/redis"

// ==================== List 操作 ====================

// LPush 将一个或多个值插入列表头部
func (c *RedisClient) LPush(key string, values ...interface{}) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(values)+1)
	args[0] = key
	copy(args[1:], values)

	return redis.Int(conn.Do("LPUSH", args...))
}

// RPush 将一个或多个值插入列表尾部
func (c *RedisClient) RPush(key string, values ...interface{}) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(values)+1)
	args[0] = key
	copy(args[1:], values)

	return redis.Int(conn.Do("RPUSH", args...))
}

// LPop 移除并返回列表头部元素
func (c *RedisClient) LPop(key string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("LPOP", key))
}

// RPop 移除并返回列表尾部元素
func (c *RedisClient) RPop(key string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("RPOP", key))
}

// LRange 获取列表指定范围内的元素
func (c *RedisClient) LRange(key string, start, stop int) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("LRANGE", key, start, stop))
}

// LLen 获取列表长度
func (c *RedisClient) LLen(key string) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("LLEN", key))
}

// LIndex 获取列表指定索引的元素
func (c *RedisClient) LIndex(key string, index int) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("LINDEX", key, index))
}

// LSet 设置列表指定索引的元素值
func (c *RedisClient) LSet(key string, index int, value interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("LSET", key, index, value)
	return err
}

// LRem 删除列表中与value相等的元素
// count > 0: 从头到尾删除count个
// count < 0: 从尾到头删除|count|个
// count = 0: 删除所有
func (c *RedisClient) LRem(key string, count int, value interface{}) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("LREM", key, count, value))
}

// LTrim 修剪列表，只保留指定区间的元素
func (c *RedisClient) LTrim(key string, start, stop int) error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("LTRIM", key, start, stop)
	return err
}

// BLPop 阻塞式弹出列表头部元素（timeout秒，0表示永久阻塞）
func (c *RedisClient) BLPop(timeout int, keys ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys)+1)
	for i, key := range keys {
		args[i] = key
	}
	args[len(keys)] = timeout

	return redis.Strings(conn.Do("BLPOP", args...))
}

// BRPop 阻塞式弹出列表尾部元素
func (c *RedisClient) BRPop(timeout int, keys ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys)+1)
	for i, key := range keys {
		args[i] = key
	}
	args[len(keys)] = timeout

	return redis.Strings(conn.Do("BRPOP", args...))
}

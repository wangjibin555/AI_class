package redis

import "github.com/gomodule/redigo/redis"

// GetBytes 获取字节数组值
func (c *RedisClient) GetBytes(key string) ([]byte, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Bytes(conn.Do("GET", key))
}

// SetNX 仅当key不存在时设置值（返回是否设置成功）
func (c *RedisClient) SetNxEx(key string, value interface{}, expiration int) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()

	if expiration > 0 {
		// SET key value NX EX seconds
		reply, err := conn.Do("SET", key, value, "NX", "EX", expiration)
		if err != nil {
			return false, err
		}
		// reply为"OK"表示成功，nil表示key已存在
		if reply == nil {
			return false, nil
		}
		return true, nil
	}

	// SETNX key value (返回1表示成功，0表示key已存在)
	result, err := redis.Int(conn.Do("SETNX", key, value))
	return result == 1, err
}

// MGet 批量获取多个key的值
func (c *RedisClient) MGet(keys ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	return redis.Strings(conn.Do("MGET", args...))
}

// MSet 批量设置多个键值对
func (c *RedisClient) MSet(pairs map[string]interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, 0, len(pairs)*2)
	for k, v := range pairs {
		args = append(args, k, v)
	}

	_, err := conn.Do("MSET", args...)
	return err
}

// Incr 将key的值加1
func (c *RedisClient) Incr(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("INCR", key))
}

// IncrBy 将key的值增加指定数量
func (c *RedisClient) IncrBy(key string, increment int64) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("INCRBY", key, increment))
}

// Decr 将key的值减1
func (c *RedisClient) Decr(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("DECR", key))
}

// DecrBy 将key的值减少指定数量
func (c *RedisClient) DecrBy(key string, decrement int64) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("DECRBY", key, decrement))
}

// Exists 检查key是否存在
func (c *RedisClient) Exists(key string) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()

	exists, err := redis.Int(conn.Do("EXISTS", key))
	return exists == 1, err
}

// Expire 设置key的过期时间（秒）
func (c *RedisClient) Expire(key string, seconds int) error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("EXPIRE", key, seconds)
	return err
}

// TTL 获取key的剩余生存时间（秒）
func (c *RedisClient) TTL(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("TTL", key))
}

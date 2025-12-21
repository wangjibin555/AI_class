package redis

import "github.com/gomodule/redigo/redis"

// ==================== Set 操作 ====================

// SAdd 向集合添加一个或多个成员
func (c *RedisClient) SAdd(key string, members ...interface{}) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(members)+1)
	args[0] = key
	copy(args[1:], members)

	return redis.Int(conn.Do("SADD", args...))
}

// SMembers 获取集合的所有成员
func (c *RedisClient) SMembers(key string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", key))
}

// SIsMember 判断元素是否是集合成员
func (c *RedisClient) SIsMember(key string, member interface{}) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()

	isMember, err := redis.Int(conn.Do("SISMEMBER", key, member))
	return isMember == 1, err
}

// SCard 获取集合的成员数
func (c *RedisClient) SCard(key string) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("SCARD", key))
}

// SRem 删除集合中一个或多个成员
func (c *RedisClient) SRem(key string, members ...interface{}) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(members)+1)
	args[0] = key
	copy(args[1:], members)

	return redis.Int(conn.Do("SREM", args...))
}

// SPop 随机移除并返回集合中的一个元素
func (c *RedisClient) SPop(key string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("SPOP", key))
}

// SRandMember 随机返回集合中的一个或多个元素（不移除）
func (c *RedisClient) SRandMember(key string, count int) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	if count == 1 {
		member, err := redis.String(conn.Do("SRANDMEMBER", key))
		if err != nil {
			return nil, err
		}
		return []string{member}, nil
	}

	return redis.Strings(conn.Do("SRANDMEMBER", key, count))
}

// SUnion 返回多个集合的并集
func (c *RedisClient) SUnion(keys ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	return redis.Strings(conn.Do("SUNION", args...))
}

// SInter 返回多个集合的交集
func (c *RedisClient) SInter(keys ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	return redis.Strings(conn.Do("SINTER", args...))
}

// SDiff 返回多个集合的差集
func (c *RedisClient) SDiff(keys ...string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	return redis.Strings(conn.Do("SDIFF", args...))
}

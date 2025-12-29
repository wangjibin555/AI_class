package config

import (
	"AI_class/pkg/logger"
	"context"
	"sync"
	"time"
)

type MemoryCache struct {
	//数据存储
	mu   sync.RWMutex
	data map[string]*CacheEntry

	//配置选项
	opts *CacheOptions

	//统计信息
	statsMu sync.RWMutex
	stats   *CacheStats

	//生命周期
	stopChan chan struct{}
	wg       sync.WaitGroup
	closed   bool
}

func NewMemoryCache(opts ...CacheOption) *MemoryCache {
	options := DefaultCacheOptions()
	for _, opt := range opts {
		opt(options)
	}
	cache := &MemoryCache{
		data: make(map[string]*CacheEntry),
		opts: options,
		stats: &CacheStats{
			MaxSize:       options.MaxSize,
			NamespaceSize: make(map[string]int),
		},
		stopChan: make(chan struct{}),
	}

	//启动定时清理
	cache.startCleanup()
	return cache
}

// 获取内存缓存条目
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := VaildateCacheKey(key); err != nil {
		c.mu.RLock()
		entry, exists := c.data[key]
		c.mu.RUnlock()

		//不存在对应key缓存
		if !exists {
			c.recordMiss()
			return nil, ErrCacheNotFound
		}

		//检查过期，惰性删除过期条目
		if entry.IsExpired() {
			c.Delete(ctx, key)
			c.recordMiss()
			c.recordEviction()
			return nil, ErrCacheExpired
		}

		//更新访问信息
		c.mu.Lock()
		entry.UpdateAccessAt()
		c.mu.Unlock()

		//返回数据
		c.recordHit()
		return entry.Value, nil
	}
	return nil, ErrCacheNotFound
}

// 设置缓存条目
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := VaildateCacheKey(key); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Lock()

	//检查容量是否足够
	if len(c.data) >= c.opts.MaxSize && c.data[key] != nil {
		//淘汰
		if err := c.evictOne(); err != nil {
			return err
		}
	}
	if err := VaildateTTL(ttl); err != nil {
		return err
	}
	expireAt := time.Now().Add(ttl)

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		ExpireAt:    expireAt,
		CreateAt:    time.Now(),
		AccessAt:    time.Now(),
		Size:        int64(len(value)),
		AccessCount: 0,
		TTL:         ttl,
	}

	c.data[key] = entry
	c.recordSet()
	return nil
}

// 删除缓存条目
func (c *MemoryCache) Delete(ctx context.Context, key string) bool {
	if err := VaildateCacheKey(key); err != nil {
		logger.Error("Invalid cache key: %d", key)
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	//校验键是否存在
	_, exists := c.data[key]
	if !exists {
		logger.Error("Cache key not found: %d", key)
		return false
	}

	delete(c.data, key)
	c.recordDelete()
	return true
}

// 定期清理过期条目

// 清理过期条目
func (c *MemoryCache) startCleanup() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.opts.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
				c.recordLastCleanup(time.Now())
			case <-c.stopChan:
				return
			}
		}
	}()
}

// 清理过期条目
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	//获取当前时间
	now := time.Now()

	expiredKeys := make([]string, 0)
	for key, entry := range c.data {
		if entry.IsExpired() && now.After(entry.ExpireAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.data, key)
		c.recordExpired()
		c.recordLastCleanup(time.Now())
	}
	c.stats.LastCleanup = now
}

// =========淘汰策略=========
// 淘汰一个条目（LRU，最久未访问）
func (c *MemoryCache) evictOne() error {
	if len(c.data) == 0 {
		logger.Warn("No entries to evict")
		return nil
	}

	var evictKey string
	var evictTime time.Time

	for key, entry := range c.data {
		if evictKey == "" || entry.AccessAt.Before(evictTime) {
			evictKey = key
			evictTime = entry.AccessAt
		}
	}
	if evictKey != "" {
		delete(c.data, evictKey)
		c.recordEviction()

		//调用当前淘汰缓存条目的回调函数
		if c.opts.OnEvicted != nil {
			c.opts.OnEvicted(evictKey, c.data[evictKey].Value)
		}
	}
	return nil
}

// 淘汰一个条目（LFU，最少使用）
func (c *MemoryCache) evictLFU() error {
	if len(c.data) == 0 {
		logger.Warn("No entries to evict")
		return nil
	}

	var evictKey string
	var evictCount int64

	for key, entry := range c.data {
		if evictKey == "" || entry.AccessCount < evictCount {
			evictKey = key
			evictCount = entry.AccessCount
		}
	}

	if evictKey != "" {
		delete(c.data, evictKey)
		c.recordEviction()
		if c.opts.OnEvicted != nil {
			c.opts.OnEvicted(evictKey, c.data[evictKey].Value)
		}
	}
	return nil
}

// 获取统计信息
func (c *MemoryCache) Stats(ctx context.Context) *CacheStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()

	c.mu.RLock()
	c.stats.Size = len(c.data)
	defer c.mu.RUnlock()
	c.stats.HitRate = CalculateHitRate(c.stats.Hits, c.stats.Misses)
	return c.stats
}

// 关闭缓存
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}

	c.closed = true
	c.mu.Unlock()

	close(c.stopChan)
	c.wg.Wait()
	return nil
}

// 记录命中
func (c *MemoryCache) recordHit() {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.Hits++
		c.statsMu.Unlock()
	}
}

// 记录未命中
func (c *MemoryCache) recordMiss() {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.Misses++
		c.statsMu.Unlock()
	}
}

// 记录过期
func (c *MemoryCache) recordEviction() {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.Evictions++
		c.statsMu.Unlock()
	}
}

// 记录过期次数
func (c *MemoryCache) recordExpired() {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.Expired++
		c.statsMu.Unlock()
	}
}

// 删除次数
func (c *MemoryCache) recordDelete() {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.Deletes++
		c.statsMu.Unlock()
	}
}

// 设置次数
func (c *MemoryCache) recordSet() {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.Sets++
		c.statsMu.Unlock()
	}
}

// 上次清理时间
func (c *MemoryCache) recordLastCleanup(date time.Time) {
	if c.opts.EnableStats {
		c.statsMu.Lock()
		c.stats.LastCleanup = date
		c.statsMu.Unlock()
	}
}

func (c *MemoryCache) evictOldest(size int64) {
	if c.opts.EnableStats {
		c.statsMu.Lock()

	}
}

package config

import (
	"AI_class/pkg/logger"
	"context"
	"errors"
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

// 检查是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) bool {
	if err := VaildateCacheKey(key); err != nil {
		logger.Warn("Invalid cache key: %d", key)
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.data[key]
	return exists
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

func (c *MemoryCache) MGet(cxt context.Context, keys ...string) (map[string][]byte, error) {
	entries := make(map[string][]byte)
	for _, key := range keys {
		value, err := c.Get(cxt, key)
		if err != nil {
			return nil, err
		}
		entries[key] = value
	}
	return entries, nil
}

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

func (c *MemoryCache) MSet(ctx context.Context, values map[string][]byte, ttl time.Duration) error {
	for key, value := range values {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			logger.Warn("MSet options chancel err")
			return err
		}
	}
	return nil
}

// 删除缓存条目
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	if err := VaildateCacheKey(key); err != nil {
		logger.Warn("Invalid cache key: %d", key)
		return ErrInvalidKey
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	//校验键是否存在
	entry, exists := c.data[key]
	if !exists {
		logger.Warn("Current Key does not exist: %s", key)
		return ErrInvalidKey
	}

	//执行回调
	if c.opts.OnEvicted != nil {
		c.opts.OnEvicted(key, c.data[key].Value)
	}
	//删除条目
	delete(c.data, key)

	//更新命名空间统计
	c.statsMu.Lock()
	if count, ok := c.stats.NamespaceSize[entry.Namespace]; ok && count > 0 {
		c.stats.NamespaceSize[entry.Namespace]--
	}
	c.statsMu.Unlock()

	c.recordDelete()
	return nil
}

func (c *MemoryCache) MDelete(ctx context.Context, keys ...string) error {
	falseKey := make([]string, 0)
	for _, key := range keys {
		if err := c.Delete(ctx, key); err != nil {
			falseKey = append(falseKey, key)
			logger.Warn("Current MDelete key: %s err")
			return ErrInvalidKey
		}
	}

	if len(keys) > len(falseKey) {
		return errors.New("Cache: MDelete some key error")
	}

	return nil
}

// 获取指定命名空间下的指定key，注意传入的key是短key，注意需要解析CacheEntry内部key才能进行匹配
func (c *MemoryCache) GetWithNamespace(ctx context.Context, namespace string, keys ...string) (map[string][]byte, error) {
	// 找到对应MemoryCache下的CacheEntry的对应key看是否匹配keys，然后匹配的进行加入一个数组内，并且进行返回
	entries := make(map[string][]byte, 0)
	if namespace == "" {
		logger.Warn("Memory_cache GetWithNamespace input namespace err")
		return nil, ErrInvalidNamespace
	}
	for fullKey, entry := range c.data {
		if entry.Namespace != namespace {
			continue
		}
		_, _, shortKey := ParseCacheKey(fullKey)

		// 从传入key中进行查找对应匹配key
		if len(keys) > 0 {
			found := false
			for _, k := range keys {
				if k == shortKey {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		entries[shortKey] = entry.Value
	}
	return entries, nil
}

// 设置缓存条目内对应data map[string]interface{}，注意当前缓存层内不进行设置或者关心数据来源，只在调用方处进行负责校验
func (c *MemoryCache) SetWithNamespace(ctx context.Context, namespace string, keys map[string][]byte, ttl time.Duration) error {
	if namespace == "" {
		logger.Warn("Memory_cache SetWithNamespace input namespace err")
		return ErrInvalidNamespace
	}
	if err := VaildateTTL(ttl); err != nil {
		return err
	}

	for key, value := range keys {
		fullKey := BuildConfigKey(namespace, key)
		if err := c.Set(ctx, fullKey, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// 删除命名空间内的对应缓存条目
func (c *MemoryCache) DeleteWithNamespace(ctx context.Context, namespace string, keys ...string) error {
	if namespace == "" {
		logger.Warn("Memory_cache DeleteWithNamespace input namespace err")
		return ErrInvalidNamespace
	}
	//构建fullKey，后在进行查找map删除
	for _, key := range keys {
		fullKey := BuildConfigKey(namespace, key)
		if err := c.Delete(ctx, fullKey); err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryCache) GetNamespaces(ctx context.Context, namespace string) ([]string, error) {
	if namespace == "" {
		logger.Warn("Memory_cache GetNamespace input namespace err")
		return nil, ErrInvalidNamespace
	}
	keys := make([]string, 0)
	for fullKey, entry := range c.data {
		if entry.Namespace == namespace {
			keys = append(keys, fullKey)
		}
	}
	return keys, nil
}

func (c *MemoryCache) GetAllNamespaces(ctx context.Context) ([]string, error) {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	namespaces := make([]string, 0)
	for namespace := range c.stats.NamespaceSize {
		namespaces = append(namespaces, namespace)
	}
	return namespaces, nil
}

// 注意，检查命名空间是否存在可以快捷在统计信息中进行判断
func (c *MemoryCache) CreateNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		logger.Warn("Memory_cache CreateNamespace input namespace err")
		return ErrInvalidNamespace
	}

	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	//检查命名空间是否已存在
	if _, exists := c.stats.NamespaceSize[namespace]; exists {
		logger.Warn("Memory_cache CreateNamespace namespace already exists: %s", namespace)
		return ErrNamespaceExists
	}
	//在统计信息中注册命名空间
	c.stats.NamespaceSize[namespace] = 0

	return nil
}

func (c *MemoryCache) DeleteNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		logger.Warn("Memory_cache DeleteNamespace input namespace error")
		return ErrInvalidNamespace
	}

	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	//先进行遍历删除data内缓存
	keysToDelete := make([]string, 0)
	for fullKey, entry := range c.data {
		if entry.Namespace == namespace {
			keysToDelete = append(keysToDelete, fullKey)
		}
	}

	for _, key := range keysToDelete {
		if c.opts.OnEvicted != nil {
			c.opts.OnEvicted(key, c.data[key].Value)
		}
		delete(c.data, key)
		c.recordDelete()
	}

	//然后进行删除统计信息
	c.statsMu.Lock()
	delete(c.stats.NamespaceSize, namespace)
	c.statsMu.Unlock()

	return nil
}

func (c *MemoryCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return 0, ErrCacheNotFound
	}

	//检查是否过期
	if entry.IsExpired() {
		return 0, ErrCacheExpired
	}

	deadline := time.Until(entry.ExpireAt)
	if deadline < 0 {
		return 0, ErrCacheNotFound
	}

	return deadline, nil
}

func (c *MemoryCache) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	if err := VaildateTTL(ttl); err != nil {
		return err
	}
	if err := VaildateCacheKey(key); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return ErrCacheNotFound
	}

	entry.ExpireAt = time.Now().Add(ttl)
	entry.TTL = ttl
	return nil
}

func (c *MemoryCache) DeleteTTL(ctx context.Context, key string) error {
	if err := VaildateCacheKey(key); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return ErrCacheNotFound
	}
	entry.ExpireAt = time.Time{}
	entry.TTL = 0
	return nil
}

func (c *MemoryCache) Size(ctx context.Context) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.data)
}

func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*CacheEntry)
	c.stats.Size = 0
	return nil
}

// 定期清理过期条目
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

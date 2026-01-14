package ratelimiter

import (
	"context"
	"sync"
)

const (
	AllLimiter     = -1
	NoLimiterQuota = 0
	limiterKey     = "limiter:frequency"
)

type LimitManager struct {
	limiter sync.Map
	newLock sync.Mutex
	store   IStore
	options *Options
}

type Options struct {
	keyPrefix string
	tl        TokenLimiterQuotaFn
}

type TokenLimiterQuotaFn func(key string) (rate int, burst int)

var lm *LimitManager

type Option func(option *Options)

func WithKeyPrefix(prefix string) Option {
	return func(option *Options) {
		option.keyPrefix = prefix
	}
}

func WithTokenLimiterQuotaFn(fn TokenLimiterQuotaFn) Option {
	return func(option *Options) {
		option.tl = fn
	}
}

var defaultOptions = Options{
	keyPrefix: limiterKey,
	tl: func(s string) (int, int) {
		return NoLimiterQuota, NoLimiterQuota // 默认不限流
	},
}

func InitLimitManager(store IStore, opts ...Option) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	lm = &LimitManager{
		store:   store,
		options: &options,
	}
}

func getLimitManager(uniKey string) Limit {
	limiter, ok := lm.limiter.Load(uniKey)
	if !ok {
		limiter = createLimitManger(uniKey)
	}
	return limiter.(Limit)
}

func createLimitManger(uniKey string) *Limit {
	lm.newLock.Lock()
	defer lm.newLock.Unlock()
	// 双重检测
	limiter, ok := lm.limiter.Load(uniKey)
	if ok {
		return limiter.(*Limit)
	}
	limiter = &LimitManager{
		store:   lm.store,
		options: lm.options,
	}
	lm.limiter.Store(uniKey, limiter)
	return limiter.(*Limit)
}

func (lm *LimitManager) newTokenLimiter(uniKey string) Limit {
	var li Limit
	rate, burst := lm.options.tl(uniKey)
	if rate == NoLimiterQuota || burst == NoLimiterQuota {
		li = &noLimit{}
	} else if rate == AllLimiter || burst == AllLimiter {
		li = &allLimit{}
	} else {
		li = NewTokenLimiter(rate, burst, lm.store, lm.options.keyPrefix)
	}
	lm.limiter.Store(uniKey, li)
	return li
}

func Allow(uniKey string) bool {
	return AllowCtx(context.Background(), uniKey)
}

func AllowCtx(ctx context.Context, uniKey string) bool {
	limiter := getLimitManager(uniKey)

	return limiter.AllowCtx(ctx)
}

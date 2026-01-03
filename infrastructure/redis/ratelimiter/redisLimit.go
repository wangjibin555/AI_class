package ratelimiter

import (
	"AI_class/pkg/logger"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gomodule/redigo/redis"
	timeRate "golang.org/x/time/rate"
)

//go:embed tokenscript.lua
var script string

const (
	tokenFormat     = "{%s}:tokens"
	timestampFormat = "{%s}:timestamp"
	pingInterval    = time.Millisecond * 1000
	defaultRate     = 100
	defaultCapacity = 2000
)

type IStore interface {
	GetConn() redis.Conn
}

type TokenLimiter struct {
	rate         int
	capacity     int
	store        IStore
	tokenKey     string
	timestampKey string
	rescueLock   sync.Mutex
	redisAlive   uint32
	moitorStart  bool
	scriptSha    string
	memLim       *timeRate.Limiter
}

func NewTokenLimiter(rate int, capacity int, store IStore, key string) *TokenLimiter {
	if rate <= 0 || capacity <= 0 {
		rate = defaultRate
		capacity = defaultCapacity
	}

	now := time.Now().Unix()
	timestampKey := fmt.Sprintf(timestampFormat, now)
	tokenKey := fmt.Sprintf(tokenFormat, key)

	tokenLimiter := &TokenLimiter{
		rate:         rate,
		capacity:     capacity,
		store:        store,
		tokenKey:     tokenKey,
		timestampKey: timestampKey,
		redisAlive:   0,
		scriptSha:    script,
		memLim:       timeRate.NewLimiter(timeRate.Every(time.Second/time.Duration(rate)), capacity),
	}

	return tokenLimiter
}

func (tl *TokenLimiter) Allow() bool {
	return tl.AllowN(time.Now(), 1)
}

func (tl *TokenLimiter) AllowN(date time.Time, n int) bool {
	return tl.recvN(context.Background(), date, n)
}

func (tl *TokenLimiter) AllowCtx(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false //上下文已取消，拒绝请求
	default:
	}

	return tl.AllowN(time.Now(), 1)
}

func (tl *TokenLimiter) recvN(ctx context.Context, date time.Time, n int) bool {
	// 如果当前redis不存活
	if atomic.LoadUint32(&tl.redisAlive) == 0 {
		return tl.memLim.AllowN(date, n)
	}

	conn := tl.store.GetConn()
	defer conn.Close()

	// evalSha <script> 2 <tokenKey><timeStampKey> <rate><burst><timeStamp><requested>
	//参数追加
	args := []interface{}{
		tl.scriptSha,
		2,
		tl.tokenKey,
		tl.timestampKey,
		strconv.Itoa(tl.rate),
		strconv.Itoa(tl.capacity),
		strconv.FormatInt(date.Unix(), 10),
		strconv.Itoa(n)}

	//执行脚本
	reply, err := redis.Int64(conn.Do("EvalSha", args...))
	if err != nil {
		//启动监控，并且返回限流器
		if errors.Is(err, redis.ErrNil) {
			return false
		}
		tl.startMonitor()
		return tl.memLim.AllowN(date, n)
	}

	return reply == 1
}

// 降级策略，启动内衬监听器
func (tl *TokenLimiter) startMonitor() {
	//上锁，无法启动两次
	tl.rescueLock.Lock()
	defer tl.rescueLock.Unlock()

	if tl.moitorStart {
		return
	}

	tl.moitorStart = true
	atomic.StoreUint32(&tl.redisAlive, 0)

	//进行监听
	go tl.waitRedis()
}

// 进行监听
func (tl *TokenLimiter) waitRedis() {
	//设置定时器
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		tl.rescueLock.Lock()
		tl.moitorStart = false
		tl.rescueLock.Unlock()
	}()

	for range ticker.C {
		if tl.testConn() {
			return
		}
	}
}

func (tl *TokenLimiter) testConn() bool {
	conn := tl.store.GetConn()
	defer conn.Close()
	_, err := conn.Do("ping")
	if err != nil {
		return false
	}
	tl.scriptSha, err = redis.String(conn.Do("script", "load", script))
	if err != nil {
		logger.Error("load script error")
		return false
	}
	atomic.StoreUint32(&tl.redisAlive, 1)
	return true
}

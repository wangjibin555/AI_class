package redislock

import (
	"AI_class/def"
	"AI_class/infrastructure/redis"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 锁对象
type LockResult struct {
	Success     bool
	LockKey     string
	RedisKey    string
	redisClient *redis.RedisClient
}

// 读写锁主要针对
type ReadWriteLockResult struct {
	LockResult  *LockResult
	IsReadLock  bool
	IsWriteLock bool
}

// 解锁脚本，上锁如果存在就失败即可
const UnlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`

var script = ""
var unlockOnce = &sync.Once{}

// 上锁
func (l *LockResult) TryLockOnce(redisClient *redis.RedisClient, redisKey string, expiration int) *LockResult {
	lockKey := uuid.New().String()
	//设置锁对象
	lockResult := &LockResult{
		Success:     true,
		LockKey:     lockKey,
		RedisKey:    redisKey,
		redisClient: redisClient,
	}
	//上锁
	success, err := lockResult.redisClient.SetNxEx(redisKey, lockKey, expiration)
	if err != nil {
		lockResult.Success = success
	}
	return lockResult
}

// 读写锁
func (rwl *ReadWriteLockResult) ReadLock(redisClient *redis.RedisClient, redisKey string, expiration, way int) *ReadWriteLockResult {
	switch way {
	case 1:
		rwl.IsReadLock = true
		rwl.IsWriteLock = false
		break
	case 2:
		rwl.IsReadLock = false
		rwl.IsWriteLock = true
		break
	default:
		return nil
	}

	lockResult := &LockResult{}

	readWriteLockResult := lockResult.TryLockOnce(redisClient, redisKey, expiration)
	rwl.LockResult = readWriteLockResult
	if readWriteLockResult.Success == false && rwl.IsReadLock == true {
		rwl.LockResult.Success = true
	}
	return rwl
}

// 放锁
func (l *LockResult) Unlock(redisClient *redis.RedisClient, redisKey string) {

	var err error
	unlockOnce.Do(func() {
		//加载脚本
		script, err = redisClient.ScriptLoad(UnlockScript)
		if err != nil {
			unlockOnce = &sync.Once{} //表示加载失败，需要在下次调用时候再次加载，相当于一个false。
		}
	})

	//加载脚本失败
	if script == "" {
		_, err = redisClient.Eval(UnlockScript, []string{redisKey}, []interface{}{l.LockKey})
	} else {
		//加载脚本成功
		_, err := redisClient.EvalSha(script, []string{redisKey}, []interface{}{l.LockKey})
		if err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT ") {
			unlockOnce = &sync.Once{}
			_, _ = l.redisClient.Eval(UnlockScript, []string{l.RedisKey}, []interface{}{l.LockKey})
		}
	}
}

// 重试锁，无时间
func (l *LockResult) RetryLock(redisClient *redis.RedisClient, redisKey string, expiredSecond int) *LockResult {
	startTime := time.Now().Unix()
	for {
		lockResult := l.TryLockOnce(redisClient, redisKey, def.RetryLockDefTime)
		if lockResult.Success {
			return lockResult
		}
		if time.Now().Unix()-startTime > int64(expiredSecond) {
			return lockResult
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// 重试锁，自定义时间
func (l *LockResult) RetryLockWithTime(redisClient *redis.RedisClient, redisKey string, expiredSecon, intervalTime int) *LockResult {
	startTime := time.Now().Unix()
	for {
		lockResult := l.TryLockOnce(redisClient, redisKey, def.RetryLockDefTime)
		if lockResult.Success {
			return lockResult
		}
		if time.Now().Unix()-startTime > int64(expiredSecon) {
			return lockResult
		}
		time.Sleep(time.Duration(intervalTime) * time.Millisecond)
	}
}

// 重试锁，退避指数
func (l *LockResult) RetryLockWithExponentBackOff(redisClient *redis.RedisClient, redisKey string, expiredSecon int) *LockResult {
	startTime := time.Now().Unix()
	retryTime := def.RetryLockDefTime
	for {
		lockResult := l.TryLockOnce(redisClient, redisKey, def.RetryLockDefTime)
		if lockResult.Success {
			return lockResult
		}
		if time.Now().Unix()-startTime > int64(expiredSecon) {
			return lockResult
		}
		time.Sleep(time.Duration(retryTime) * time.Millisecond)
		retryTime = retryTime * 2
	}
}

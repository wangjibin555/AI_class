package graceful

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	pprofBuffSize = 1024 * 1024 * 10 // pprof 10M缓存
)

type HandlerFunc func()

var (
	handleLock sync.Mutex
	handlers   []HandlerFunc
)

// 统一注册
func RegisterAllFunc(registerfunc ...HandlerFunc) {
	if len(registerfunc) <= 0 {
		return
	}

	handleLock.Lock()
	defer handleLock.Unlock()
	handlers = append(handlers, registerfunc...)
}

// 统一触发清理执行（pprof转存）
func ClearAllFunc(waitTimeout time.Duration, opts ...Opt) {
	option := appleOpt(opts)
	waitGroup := sync.WaitGroup{}
	for i, handler := range handlers {
		tmpHandler := handler
		waitGroup.Add(1)
		go func() {
			defer func() {
				logrus.Infoln("ClearAllFunc done:", i)
			}()
			tmpHandler()
		}()
	}
	if waitTimeout == 0 {
		waitGroup.Wait()
		return
	}
	done := make(chan bool, 1)
	go func() {
		waitGroup.Wait()
		done <- true
	}()
	select {
	case <-done:
		logrus.Infoln("ClearAllFunc done")
	case <-time.After(waitTimeout):
		if option.warnLog {
			logrus.Warnln("ClearAllFunc timeout:", waitTimeout)
		} else {
			logrus.Errorln("ClearAllFunc timeout:", waitTimeout)
		}
		// pprof转存
		buf := make([]byte, pprofBuffSize)
		n := runtime.Stack(buf, true)
		buf = buf[:n]
		name := fmt.Sprintf("/opt/systemd_monitor/logs/waittimeout_%s.pprof", time.Now().Format("20060102150405"))
		err := os.WriteFile(name, buf, 0666)
		if err != nil {
			logrus.WithError(err).WithField("stack", string(buf)).Error("write stack error")
		}
	}
}

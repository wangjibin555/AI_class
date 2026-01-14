package graceful

// 配置模式实现日志等级变更
type Opt func(*opt)

type opt struct {
	warnLog bool
}

func WithWarnLog(warnLog bool) Opt {
	return func(o *opt) {
		o.warnLog = warnLog
	}
}

func appleOpt(opts []Opt) *opt {
	o := &opt{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

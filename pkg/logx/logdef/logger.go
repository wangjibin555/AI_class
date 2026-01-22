package logdef

type Fields map[string]interface{}

type ILogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})

	WithField(key string, value interface{}) ILogger
	WithFields(field Fields) ILogger
	WithSkip(skip int) ILogger
	WithName(name string) ILogger
	WithLevel(level Level) ILogger
}

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelPanic Level = "panic"
	LevelFatal Level = "fatal"
)

func (l Level) IntValue() int {
	switch l {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	case LevelPanic:
		return 4
	case LevelFatal:
		return 5
	default:
		return 0
	}
}

const (
	FormatLn  = "formatLn"
	FormatF   = "formatF"
	FormatNil = "formatNil"
)

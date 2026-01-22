package stdlog

import (
	"AI_class/pkg/logx/logdef"
	"AI_class/pkg/poolx"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

var builderPool = poolx.NewObjectPool[*strings.Builder](func() *strings.Builder { return &strings.Builder{} })

type Field struct {
	K string
	V interface{}
}

type Hook func(level logdef.Level, fields []Field, msg string)

type Logger struct {
	skip   int
	fields []Field
	level  logdef.Level
	logger *log.Logger
	name   string
	Hook   Hook
}

func NewDefaultLog() logdef.ILogger {
	lg := log.New(os.Stdout, "", log.Lshortfile[log.LstdFlags[log.Lmicroseconds]])
	l := &Logger{
		logger: lg,
		skip:   3,
	}
	l.level = logdef.LevelInfo
	l.updatePrefix()
	return l
}

func NewLog(lg *log.Logger) logdef.ILogger {
	return &Logger{
		logger: lg,
	}
}

func (l Logger) Debug(args ...interface{}) {
	if logdef.LevelDebug.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelDebug, fmt.Sprint(args...))
}

func (l Logger) Debugf(format string, args ...interface{}) {
	if logdef.LevelDebug.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelDebug, fmt.Sprintf(format, args...))
}

func (l Logger) Info(args ...interface{}) {
	if logdef.LevelInfo.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelInfo, fmt.Sprint(args...))
}

func (l Logger) Infof(format string, args ...interface{}) {
	if logdef.LevelInfo.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelInfo, fmt.Sprintf(format, args...))
}

func (l Logger) Error(args ...interface{}) {
	if logdef.LevelError.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelError, fmt.Sprint(args...))
}

func (l Logger) Errorf(format string, args ...interface{}) {
	if logdef.LevelError.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelError, fmt.Sprintf(format, args...))
}

func (l Logger) Warn(args ...interface{}) {
	if logdef.LevelWarn.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelWarn, fmt.Sprint(args...))
}

func (l Logger) Warnf(format string, args ...interface{}) {
	if logdef.LevelWarn.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelWarn, fmt.Sprintf(format, args...))
}

func (l Logger) Fatal(args ...interface{}) {
	if logdef.LevelFatal.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelFatal, fmt.Sprint(args...))
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	if logdef.LevelFatal.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelFatal, fmt.Sprintf(format, args...))
}

func (l Logger) Panic(v ...interface{}) {
	if logdef.LevelPanic.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelPanic, fmt.Sprint(v...))
}

func (l Logger) Panicf(format string, args ...interface{}) {
	if logdef.LevelPanic.IntValue() < l.level.IntValue() {
		return
	}
	l.print(logdef.LevelPanic, fmt.Sprintf(format, args...))
}

func (l Logger) WithFields(fields logdef.Fields) logdef.ILogger {
	c := l.fastClone()
	for key, value := range fields {
		c.fields = slices.DeleteFunc(c.fields, func(f Field) bool {
			return f.K == key
		})
		c.fields = append(c.fields, Field{
			K: key,
			V: value,
		})
	}
	return c
}

func (l Logger) WithField(key string, value interface{}) logdef.ILogger {
	c := l.fastClone()
	c.fields = slices.DeleteFunc(c.fields, func(f Field) bool {
		return f.K == key
	})
	c.fields = append(c.fields, Field{
		K: key,
		V: value,
	})
	return c
}

func (l Logger) WithName(name string) logdef.ILogger {
	c := l.fastClone()
	c.name = l.name + "." + name
	c.updatePrefix()
	return c
}

func (l Logger) WithSkip(skip int) logdef.ILogger {
	c := l.fastClone()
	c.skip += skip
	return c
}

func (l Logger) WithLevel(level logdef.Level) logdef.ILogger {
	c := l.fastClone()
	c.level = level
	return c
}

func (l Logger) fastClone() Logger {
	c := Logger{
		skip:   l.skip,
		level:  l.level,
		logger: l.logger,
		name:   l.name,
		Hook:   l.Hook,
	}
	c.fields = make([]Field, len(l.fields))
	copy(c.fields, l.fields)
	return c
}

func (l Logger) deepClone() Logger {
	c := Logger{
		skip:   l.skip,
		level:  l.level,
		logger: log.New(l.logger.Writer(), l.logger.Prefix(), l.logger.Flags()), //新logger
		name:   l.name,
		Hook:   l.Hook,
	}
	for _, fd := range l.fields {
		c.fields = append(c.fields, Field{K: fd.K, V: fd.V})
	}
	return c
}

func (l Logger) updatePrefix() {
	l.logger.SetPrefix(fmt.Sprintf("%s %s ", l.level, l.name))
}

func (l Logger) print(level logdef.Level, msg string) {
	builderPoolObj := builderPool.Get()
	builder := builderPoolObj.GetData()
	defer func() {
		builderPoolObj.GetData().Reset()
		builderPool.Put(builderPoolObj)
	}()

	// 预估容量：每个字段约 20 字符 + 消息长度
	builder.Grow(len(l.fields)*20 + len(msg) + 10)

	for _, f := range l.fields {
		builder.WriteString(" ")
		builder.WriteString(f.K)
		builder.WriteString(" ")
		fmt.Fprint(builder, f.V)
	}
	builder.WriteString(" ")
	builder.WriteString(msg)

	result := builder.String()
	if l.Hook != nil {
		l.Hook(level, l.fields, result)
	}
	l.logger.Output(l.skip, result)
}

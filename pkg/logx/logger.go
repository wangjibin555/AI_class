package logx

import (
	"AI_class/pkg/logx/logdef"
	"AI_class/pkg/logx/stdlog"
)

var _globalLogger logdef.ILogger = nil
var _log logdef.ILogger = nil

func init() {
	SetLogger(stdlog.NewDefaultLog())
}

func SetLogger(logger logdef.ILogger) {
	_log = logger
	_globalLogger = _log.WithSkip(1)
}

func Debug(args ...interface{}) {
	_globalLogger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	_globalLogger.Debugf(format, args...)
}

func Info(args ...interface{}) {
	_globalLogger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	_globalLogger.Infof(format, args...)
}

func Warn(args ...interface{}) {
	_globalLogger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	_globalLogger.Warnf(format, args...)
}

func Error(args ...interface{}) {
	_globalLogger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	_globalLogger.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	_globalLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	_globalLogger.Fatalf(format, args...)
}

func Panic(args ...interface{}) {
	_globalLogger.Panic(args...)
}

func Panicf(format string, args ...interface{}) {
	_globalLogger.Panicf(format, args...)
}

func WithFields(fields logdef.Fields) logdef.ILogger {
	return _globalLogger.WithFields(fields)
}

func WithField(key string, value interface{}) logdef.ILogger {
	return _globalLogger.WithField(key, value)
}

func WithSkip(skip int) logdef.ILogger {
	return _globalLogger.WithSkip(skip)
}

func WithName(name string) logdef.ILogger {
	return _globalLogger.WithName(name)
}

func WithLevel(level logdef.Level) logdef.ILogger {
	return _globalLogger.WithLevel(level)
}

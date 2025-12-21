package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

type LoggerLevel int32

const (
	DEBUG LoggerLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel int32          = int32(INFO) // 使用 int32 支持原子操作
	logger       unsafe.Pointer               // 原子指针，避免锁
	loggerOnce   sync.Once
	initMu       sync.Mutex // 只在 Init 时使用，非常少
)

// 初始化默认 logger（如果未初始化）
func ensureLogger() {
	loggerOnce.Do(func() {
		if atomic.LoadPointer(&logger) == nil {
			defaultLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
			atomic.StorePointer(&logger, unsafe.Pointer(defaultLogger))
		}
	})
}

// 生命周期
// 初始化日志
func Init(level LoggerLevel, logFile string) error {
	initMu.Lock()
	defer initMu.Unlock()

	atomic.StoreInt32(&currentLevel, int32(level))

	//文件输出
	var output *os.File
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	} else {
		output = os.Stdout
	}

	//初始化logger（原子存储）
	newLogger := log.New(output, "", log.Ldate|log.Ltime)
	atomic.StorePointer(&logger, unsafe.Pointer(newLogger))
	return nil
}

// 原子读取 logger（无锁）
func getLogger() *log.Logger {
	ensureLogger()
	return (*log.Logger)(atomic.LoadPointer(&logger))
}

// DEBUG调试日志（无锁读取）
func Debug(format string, v ...interface{}) {
	level := LoggerLevel(atomic.LoadInt32(&currentLevel))
	if level <= DEBUG {
		log := getLogger()
		if log != nil {
			log.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
		}
	}
}

// Info 信息日志（无锁读取）
func Info(format string, v ...interface{}) {
	level := LoggerLevel(atomic.LoadInt32(&currentLevel))
	if level <= INFO {
		log := getLogger()
		if log != nil {
			log.Output(2, fmt.Sprintf("[INFO] "+format, v...))
		}
	}
}

// Warn 警告日志（无锁读取）
func Warn(format string, v ...interface{}) {
	level := LoggerLevel(atomic.LoadInt32(&currentLevel))
	if level <= WARN {
		log := getLogger()
		if log != nil {
			log.Output(2, fmt.Sprintf("[WARN] "+format, v...))
		}
	}
}

// Error 错误日志（无锁读取）
func Error(format string, v ...interface{}) {
	level := LoggerLevel(atomic.LoadInt32(&currentLevel))
	if level <= ERROR {
		log := getLogger()
		if log != nil {
			log.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
		}
	}
}

// 带上下文日志结构
type ContextLogger struct {
	ctx    interface{}
	logger *log.Logger // 每个实例持有自己的引用，无需锁
	level  LoggerLevel
}

// 带上下文日志
func WithContext(ctx interface{}) *ContextLogger {
	ensureLogger()
	log := getLogger()
	level := LoggerLevel(atomic.LoadInt32(&currentLevel))

	return &ContextLogger{
		ctx:    ctx,
		logger: log,
		level:  level,
	}
}

// v是不定参数
func (l *ContextLogger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG && l.logger != nil {
		l.logger.Output(2, fmt.Sprintf("[DEBUG] [%v] "+format, append([]interface{}{l.ctx}, v...)...))
	}
}

func (l *ContextLogger) Info(format string, v ...interface{}) {
	if l.level <= INFO && l.logger != nil {
		l.logger.Output(2, fmt.Sprintf("[INFO] [%v] "+format, append([]interface{}{l.ctx}, v...)...))
	}
}

func (l *ContextLogger) Warn(format string, v ...interface{}) {
	if l.level <= WARN && l.logger != nil {
		l.logger.Output(2, fmt.Sprintf("[WARN] [%v] "+format, append([]interface{}{l.ctx}, v...)...))
	}
}

func (l *ContextLogger) Error(format string, v ...interface{}) {
	if l.level <= ERROR && l.logger != nil {
		l.logger.Output(2, fmt.Sprintf("[ERROR] [%v] "+format, append([]interface{}{l.ctx}, v...)...))
	}
}

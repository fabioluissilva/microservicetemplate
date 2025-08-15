package commonlogger

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
)

var (
	logLevel *slog.LevelVar
	logger   *slog.Logger
	once     sync.Once
)

func GetLogger() *slog.Logger {
	once.Do(initializeLogger)
	return logger
}

func GetLogLevel() *slog.LevelVar {
	return logLevel
}

func logWithLevel(level func(string, ...interface{}), msg string, args ...interface{}) {
	pc, _, line, _ := runtime.Caller(2)
	packageParts := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndex(packageParts, "/")
	if lastSlash != -1 {
		packageParts = packageParts[lastSlash+1:]
	}
	level(fmt.Sprintf("[%s:%d] %s", packageParts, line, msg), args...)
}

func Debug(msg string, args ...interface{}) {
	logWithLevel(GetLogger().Debug, msg, args...)
}

func Info(msg string, args ...interface{}) {
	logWithLevel(GetLogger().Info, msg, args...)
}
func Warn(msg string, args ...interface{}) {
	logWithLevel(GetLogger().Warn, msg, args...)
}
func Error(msg string, args ...interface{}) {
	logWithLevel(GetLogger().Error, msg, args...)
}

func SetLogLevel(level string) {
	initializeLogger()
	switch level {
	case "DEBUG":
		logLevel.Set(slog.LevelDebug)
	case "INFO":
		logLevel.Set(slog.LevelInfo)
	case "WARN", "WARNING":
		logLevel.Set(slog.LevelWarn)
	case "ERROR":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
}

func initializeLogger() {
	// By Default the log level is set to Debug
	once.Do(func() {
		logLevel = new(slog.LevelVar)
		logLevel.Set(slog.LevelDebug)
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
		slog.SetDefault(logger)
		logger.Debug("Logger initialized")
	})
}

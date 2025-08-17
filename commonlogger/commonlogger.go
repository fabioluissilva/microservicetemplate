package commonlogger

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/fabioluissilva/microservicetemplate/utilities"
)

var (
	logLevel    *slog.LevelVar
	logger      *slog.Logger
	once        sync.Once
	serviceName string
)

func GetLogger() *slog.Logger {
	once.Do(initializeLogger)
	return logger
}

func GetLogLevel() *slog.LevelVar {
	return logLevel
}

func logWithLevel(level func(string, ...interface{}), msg string, args ...interface{}) {
	pkg, label, line := utilities.CallerLabel(3)
	if GetLogLevel().Level() < slog.LevelInfo {
		level(fmt.Sprintf("[%s:%d] %s", label, line, msg), args...)
	} else {
		level(fmt.Sprintf("[%s] %s", pkg, msg), args...)
	}
}

func appendServiceName(args ...interface{}) []interface{} {
	if serviceName != "" {
		return append([]interface{}{"service", serviceName}, args...)
	}
	return args
}

func Debug(msg string, args ...interface{}) {
	args = appendServiceName(args...)
	logWithLevel(GetLogger().Debug, msg, args...)
}

func Info(msg string, args ...interface{}) {
	args = appendServiceName(args...)
	logWithLevel(GetLogger().Info, msg, args...)
}
func Warn(msg string, args ...interface{}) {
	args = appendServiceName(args...)
	logWithLevel(GetLogger().Warn, msg, args...)
}
func Error(msg string, args ...interface{}) {
	args = appendServiceName(args...)
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
func SetServiceName(name string) {
	serviceName = name
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

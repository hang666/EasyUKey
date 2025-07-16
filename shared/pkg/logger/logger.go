package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// LogConfig 日志配置结构
type LogConfig struct {
	Level   string `yaml:"level" mapstructure:"level"`
	Format  string `yaml:"format" mapstructure:"format"`
	Output  string `yaml:"output" mapstructure:"output"`
	Console string `yaml:"console" mapstructure:"console"`
}

// Logger 全局logger实例
var Logger *slog.Logger

// InitLogger 初始化服务端日志系统
func InitLogger(cfg *LogConfig) (*slog.Logger, error) {
	var writer io.Writer
	var handler slog.Handler

	// 确定输出目标
	switch strings.ToLower(cfg.Output) {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// 输出到文件
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return nil, err
		}
		writer = file
	}

	// 确定日志级别
	level := parseLogLevel(cfg.Level)

	// 创建handler选项
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Debug级别时显示源码位置
	}

	// 根据格式创建不同的handler
	switch strings.ToLower(cfg.Format) {
	case "json", "":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	logger := slog.New(handler)

	// 设置为默认日志
	slog.SetDefault(logger)
	Logger = logger

	return logger, nil
}

// InitClientLogger 初始化客户端日志系统
func InitClientLogger(logLevel, logFile, logConsole string) (*slog.Logger, error) {
	var writer io.Writer
	var handler slog.Handler

	// 设置默认值
	if logLevel == "" {
		logLevel = "info"
	}
	if logFile == "" {
		logFile = "logs/client.log"
	}
	if logConsole == "" {
		logConsole = "true"
	}

	// 创建输出目标
	var writers []io.Writer

	// 文件输出
	logDir := filepath.Dir(logFile)
	if logDir != "" && logDir != "." {
		_ = os.MkdirAll(logDir, 0o755)
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	writers = append(writers, file)

	// 控制台输出
	if strings.ToLower(logConsole) == "true" {
		writers = append(writers, os.Stdout)
	}

	// 创建多重输出
	if len(writers) == 1 {
		writer = writers[0]
	} else if len(writers) > 1 {
		writer = io.MultiWriter(writers...)
	} else {
		writer = os.Stdout // 默认输出到控制台
	}

	// 确定日志级别
	level := parseLogLevel(logLevel)

	// 创建handler选项
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Debug级别时显示源码位置
	}

	// 使用文本格式
	handler = slog.NewTextHandler(writer, opts)

	logger := slog.New(handler)

	// 设置为默认日志
	slog.SetDefault(logger)
	Logger = logger

	return logger, nil
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

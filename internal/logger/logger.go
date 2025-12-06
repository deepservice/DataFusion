package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 结构化日志器
type Logger struct {
	*zap.Logger
}

// Config 日志配置
type Config struct {
	Level      string `json:"level"`       // debug, info, warn, error
	Format     string `json:"format"`      // json, console
	OutputPath string `json:"output_path"` // stdout, stderr, or file path
}

// contextKey 用于在 context 中存储请求 ID
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	taskIDKey    contextKey = "task_id"
)

var globalLogger *Logger

// NewLogger 创建日志器
func NewLogger(config *Config) (*Logger, error) {
	// 设置日志级别
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 设置编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 设置编码器
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 设置输出
	var output zapcore.WriteSyncer
	if config.OutputPath == "" || config.OutputPath == "stdout" {
		output = zapcore.AddSync(os.Stdout)
	} else if config.OutputPath == "stderr" {
		output = zapcore.AddSync(os.Stderr)
	} else {
		file, err := os.OpenFile(config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	// 创建 core
	core := zapcore.NewCore(encoder, output, level)

	// 创建 logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	logger := &Logger{Logger: zapLogger}
	globalLogger = logger

	return logger, nil
}

// GetLogger 获取全局日志器
func GetLogger() *Logger {
	if globalLogger == nil {
		// 如果没有初始化，创建默认日志器
		config := &Config{
			Level:  "info",
			Format: "console",
		}
		logger, _ := NewLogger(config)
		return logger
	}
	return globalLogger
}

// WithRequestID 添加请求 ID 到 context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithTaskID 添加任务 ID 到 context
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, taskIDKey, taskID)
}

// FromContext 从 context 创建带上下文信息的日志器
func (l *Logger) FromContext(ctx context.Context) *Logger {
	fields := []zap.Field{}

	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if taskID, ok := ctx.Value(taskIDKey).(string); ok {
		fields = append(fields, zap.String("task_id", taskID))
	}

	if len(fields) > 0 {
		return &Logger{Logger: l.With(fields...)}
	}

	return l
}

// WithFields 添加字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{Logger: l.With(zapFields...)}
}

// WithError 添加错误字段
func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.With(zap.Error(err))}
}

// WithComponent 添加组件字段
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.With(zap.String("component", component))}
}

// WithTask 添加任务字段
func (l *Logger) WithTask(taskName, taskType string) *Logger {
	return &Logger{Logger: l.With(
		zap.String("task_name", taskName),
		zap.String("task_type", taskType),
	)}
}

// Sync 刷新日志缓冲区
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Close 关闭日志器
func (l *Logger) Close() error {
	return l.Sync()
}

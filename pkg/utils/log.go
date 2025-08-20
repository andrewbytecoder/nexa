package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Option 是函数式选项类型
type Option func(*loggerConfig)

// loggerConfig 日志配置项
type loggerConfig struct {
	filename      string // 日志名
	maxSize       int    // MB
	maxBackups    int
	maxAge        int  // days 日志停留最大天数
	compress      bool //是否启用日志压缩
	localTime     bool
	level         zapcore.Level
	encodeAsJSON  bool
	enableConsole bool
	consoleLevel  zapcore.Level
	outputPaths   []string // 自定义输出路径（文件+控制台）
	errorPaths    []string // 错误日志输出路径
}

// 默认配置
func defaultConfig() *loggerConfig {
	return &loggerConfig{
		filename:      "logs/app.log",
		maxSize:       100,  // 100 MB
		maxBackups:    7,    // 7 个备份
		maxAge:        30,   // 30 天
		compress:      true, // 压缩旧日志
		localTime:     true,
		level:         zapcore.InfoLevel,
		encodeAsJSON:  false,
		enableConsole: true,
		consoleLevel:  zapcore.WarnLevel,
		outputPaths:   []string{"stdout"},
		errorPaths:    []string{"stderr"},
	}
}

// WithFilename 选项函数：设置日志文件路径
func WithFilename(filename string) Option {
	return func(c *loggerConfig) {
		if filename != "" {
			c.filename = filename
		}
	}
}

// WithMaxSize 选项函数：设置最大文件大小（MB）
func WithMaxSize(size int) Option {
	return func(c *loggerConfig) {
		if size > 0 {
			c.maxSize = size
		}
	}
}

// WithMaxBackups 选项函数：设置最大备份文件数
func WithMaxBackups(backups int) Option {
	return func(c *loggerConfig) {
		if backups >= 0 {
			c.maxBackups = backups
		}
	}
}

// WithMaxAge 选项函数：设置最大保留天数
func WithMaxAge(days int) Option {
	return func(c *loggerConfig) {
		if days > 0 {
			c.maxAge = days
		}
	}
}

// WithCompress 选项函数：是否压缩旧日志
func WithCompress(compress bool) Option {
	return func(c *loggerConfig) {
		c.compress = compress
	}
}

// WithLocalTime 选项函数：是否使用本地时间（否则用 UTC）
func WithLocalTime(local bool) Option {
	return func(c *loggerConfig) {
		c.localTime = local
	}
}

// WithLevel 选项函数：设置日志级别
func WithLevel(level zapcore.Level) Option {
	return func(c *loggerConfig) {
		c.level = level
	}
}

// WithConsoleLevel 选项函数：设置控制台日志级别
func WithConsoleLevel(level zapcore.Level) Option {
	return func(c *loggerConfig) {
		c.consoleLevel = level
	}
}

// WithJSONFormat 选项函数：是否以 JSON 格式输出
func WithJSONFormat() Option {
	return func(c *loggerConfig) {
		c.encodeAsJSON = true
	}
}

// WithoutConsole 选项函数：禁用控制台输出
func WithoutConsole() Option {
	return func(c *loggerConfig) {
		c.enableConsole = false
	}
}

// WithOutputPaths 选项函数：自定义输出路径（可覆盖文件和控制台）
func WithOutputPaths(paths ...string) Option {
	return func(c *loggerConfig) {
		c.outputPaths = paths
	}
}

// WithErrorPaths 选项函数：自定义错误日志路径
func WithErrorPaths(paths ...string) Option {
	return func(c *loggerConfig) {
		c.errorPaths = paths
	}
}

// NewLogger 创建一个新的日志实例
func NewLogger(opts ...Option) (*zap.Logger, error) {
	cfg := defaultConfig()

	// 应用所有选项
	for _, opt := range opts {
		opt(cfg)
	}

	// 创建 lumberjack writer
	lumberjackLogger := &lumberjack.Logger{
		Filename:   cfg.filename,
		MaxSize:    cfg.maxSize,
		MaxBackups: cfg.maxBackups,
		MaxAge:     cfg.maxAge,
		Compress:   cfg.compress,
		LocalTime:  cfg.localTime,
	}

	// 构建 zap 的核心 encoder config
	var encoder zapcore.Encoder
	if cfg.encodeAsJSON {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	} else {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 彩色输出
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 构建写入目标
	var cores []zapcore.Core

	// 1. 文件写入（所有级别）
	fileCore := zapcore.NewCore(encoder, zapcore.AddSync(lumberjackLogger), cfg.level)
	cores = append(cores, fileCore)

	// 2. 控制台输出（可选，且可设不同级别）
	if cfg.enableConsole {
		consoleWriter := zapcore.AddSync(os.Stdout)
		consoleCore := zapcore.NewCore(encoder, consoleWriter, cfg.consoleLevel)
		cores = append(cores, consoleCore)
	}

	// 合并所有 core
	multiCore := zapcore.NewTee(cores...)

	// 构建 zap.Logger
	zapLogger := zap.New(multiCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return zapLogger, nil
}

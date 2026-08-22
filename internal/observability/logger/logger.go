package logger

import (
	"os"
	"path/filepath"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	level    zapcore.Level
)

type LoggerConfig struct {
	Level      string `mapstructure:"LOG_LEVEL"`
	Encoding   string `mapstructure:"LOG_ENCODING"`
	OutputPath string `mapstructure:"LOG_OUTPUT_PATH"`
	LogFile    string `mapstructure:"LOG_FILE"`
}

func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "info",
		Encoding:   "json",
		OutputPath: "stdout",
		LogFile:    "logs/mcp-server.log",
	}
}

func NewLogger(cfg *config.Config) (*zap.Logger, func(), error) {
	logCfg := LoggerConfig{
		Level:      cfg.LogLevel,
		Encoding:   cfg.LogEncoding,
		OutputPath: "stdout",
		LogFile:    cfg.LogFile,
	}
	if logCfg.LogFile == "" {
		logCfg.LogFile = "logs/mcp-server.log"
	}

	lvl, err := zapcore.ParseLevel(logCfg.Level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}
	level = lvl

	// ===== ENCODER CONFIG - BONITO EM DEV =====
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	isDev := !cfg.IsProduction()

	// COR no console
	if isDev || logCfg.Encoding == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		encoderConfig.ConsoleSeparator = " | "
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	var encoder zapcore.Encoder
	if isDev {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else if logCfg.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// ===== SAÍDA DUPLA: TERMINAL + ARQUIVO =====
	var writers []zapcore.WriteSyncer
	writers = append(writers, zapcore.AddSync(os.Stdout))

	if cfg.LogFile != "" && cfg.LogFile != "stdout" {
		_ = os.MkdirAll(filepath.Dir(cfg.LogFile), 0755)
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			writers = append(writers, zapcore.AddSync(file))
		}
	}

	writeSyncer := zapcore.NewMultiWriteSyncer(writers...)
	core := zapcore.NewCore(encoder, writeSyncer, level)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	cleanup := func() { _ = logger.Sync() }
	instance = logger
	return logger, cleanup, nil
}

func GetLogger() *zap.Logger { return instance }

func LogInfo(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Sync() error {
	if instance != nil {
		return instance.Sync()
	}
	return nil
}

func SetLevel(lvl zapcore.Level) { level = lvl }
func GetLevel() zapcore.Level    { return level }

// internal/observability/logger/logger.go
package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	once     sync.Once
	level    zapcore.Level
)

// Config contém as configurações do logger
type Config struct {
	Level      string `mapstructure:"LOG_LEVEL"`
	Encoding   string `mapstructure:"LOG_ENCODING"`
	OutputPath string `mapstructure:"LOG_OUTPUT_PATH"`
	LogFile    string `mapstructure:"LOG_FILE"` // NOVO: caminho do arquivo de log
}

// DefaultConfig retorna a configuração padrão
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Encoding:   "json",
		OutputPath: "stdout",
		LogFile:    "logs/mcp-server.log",
	}
}

// Init inicializa o logger
func Init(cfg Config) error {
	var err error
	once.Do(func() {
		// Define o nível
		lvl, parseErr := zapcore.ParseLevel(cfg.Level)
		if parseErr != nil {
			lvl = zapcore.InfoLevel
		}
		level = lvl

		// Encoder config
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		encoder := zapcore.NewJSONEncoder(encoderConfig)
		if cfg.Encoding == "console" {
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		// ============================================
		// ✅ SAÍDA DUPLA: ARQUIVO + TERMINAL
		// ============================================
		var writers []zapcore.WriteSyncer

		// 1. Sempre escreve no terminal (stdout)
		writers = append(writers, zapcore.AddSync(os.Stdout))

		// 2. Se LOG_FILE estiver configurado, escreve no arquivo
		if cfg.LogFile != "" && cfg.LogFile != "stdout" {
			// Cria diretório se não existir
			// ... (ver código abaixo)

			file, openErr := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if openErr == nil {
				writers = append(writers, zapcore.AddSync(file))
			}
		}

		// 3. Se LOG_FILE estiver vazio ou "stdout", usa apenas stdout
		writeSyncer := zapcore.NewMultiWriteSyncer(writers...)

		core := zapcore.NewCore(encoder, writeSyncer, level)
		instance = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	})

	return err
}

// GetLogger retorna a instância do logger
func GetLogger() *zap.Logger {
	if instance == nil {
		_ = Init(DefaultConfig())
	}
	return instance
}

// Sync sincroniza os logs
func Sync() error {
	if instance != nil {
		return instance.Sync()
	}
	return nil
}

func SetLevel(lvl zapcore.Level) {
	level = lvl
}

func GetLevel() zapcore.Level {
	return level
}

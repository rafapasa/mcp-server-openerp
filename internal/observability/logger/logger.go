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
	Encoding   string `mapstructure:"LOG_ENCODING"` // json ou console
	OutputPath string `mapstructure:"LOG_OUTPUT_PATH"`
}

// DefaultConfig retorna a configuração padrão
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Encoding:   "json",
		OutputPath: "stdout",
	}
}

// Init inicializa o logger com as configurações fornecidas
func Init(cfg Config) error {
	var err error
	once.Do(func() {
		// Define o nível
		lvl, parseErr := zapcore.ParseLevel(cfg.Level)
		if parseErr != nil {
			lvl = zapcore.InfoLevel
		}
		level = lvl

		// Define o encoder
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

		// Define o output (stdout ou arquivo)
		var writeSyncer zapcore.WriteSyncer
		if cfg.OutputPath == "" || cfg.OutputPath == "stdout" {
			writeSyncer = zapcore.AddSync(os.Stdout)
		} else {
			writeSyncer = zapcore.AddSync(&lumberjackLogger{
				Filename:   cfg.OutputPath,
				MaxSize:    100, // MB
				MaxBackups: 3,
				MaxAge:     28, // days
			})
			if err != nil {
				return
			}
		}

		core := zapcore.NewCore(encoder, writeSyncer, level)
		instance = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	})

	return err
}

// lumberjackLogger implementa zapcore.WriteSyncer para rotação de logs
type lumberjackLogger struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

func (l *lumberjackLogger) Write(p []byte) (n int, err error) {
	// Implementação simplificada - em produção usar gopkg.in/natefinch/lumberjack.v2
	return os.Stdout.Write(p)
}

func (l *lumberjackLogger) Sync() error {
	return nil
}

// GetLogger retorna a instância do logger
func GetLogger() *zap.Logger {
	if instance == nil {
		// Inicializa com configurações padrão se não foi iniciado
		_ = Init(DefaultConfig())
	}
	return instance
}

// SetLevel altera o nível de log em runtime
func SetLevel(lvl zapcore.Level) {
	level = lvl
	// Recria o core com o novo nível
	if instance != nil {
		// O zap não permite alterar o nível diretamente, recriamos o logger
		cfg := DefaultConfig()
		cfg.Level = lvl.String()
		_ = Init(cfg)
	}
}

// GetLevel retorna o nível atual
func GetLevel() zapcore.Level {
	return level
}

// Sync sincroniza os logs
func Sync() error {
	if instance != nil {
		return instance.Sync()
	}
	return nil
}

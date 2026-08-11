// internal/observability/logger/levels.go
package logger

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

var (
	levelMutex sync.RWMutex
)

// SetLevelFromString define o nível a partir de uma string
func SetLevelFromString(levelStr string) error {
	lvl, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		return err
	}
	SetLevel(lvl)
	return nil
}

// GetLevelString retorna o nível como string
func GetLevelString() string {
	return GetLevel().String()
}

// IsDebugEnabled verifica se o nível DEBUG está habilitado
func IsDebugEnabled() bool {
	return GetLevel() <= zapcore.DebugLevel
}

// IsInfoEnabled verifica se o nível INFO está habilitado
func IsInfoEnabled() bool {
	return GetLevel() <= zapcore.InfoLevel
}

// IsWarnEnabled verifica se o nível WARN está habilitado
func IsWarnEnabled() bool {
	return GetLevel() <= zapcore.WarnLevel
}

// IsErrorEnabled verifica se o nível ERROR está habilitado
func IsErrorEnabled() bool {
	return GetLevel() <= zapcore.ErrorLevel
}

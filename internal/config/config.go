package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config armazena todas as configurações da aplicação
type Config struct {
	// Logger
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	LogEncoding string `mapstructure:"LOG_ENCODING"`
	LogOutput   string `mapstructure:"LOG_OUTPUT"`

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// API
	APIPort string
	APIEnv  string

	// JWT
	JWTSecret           string
	JWTExpiresIn        time.Duration
	JWTRefreshExpiresIn time.Duration

	// Redis (opcional)
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// CORS
	CORSAllowedOrigins string
	CORSAllowedMethods string
	CORSAllowedHeaders string

	// Timezone
	TimeZone string

	TracingEnabled      bool    `mapstructure:"TRACING_ENABLED"`
	TracingEndpoint     string  `mapstructure:"TRACING_ENDPOINT"`
	TracingServiceName  string  `mapstructure:"TRACING_SERVICE_NAME"`
	TracingSamplingRate float64 `mapstructure:"TRACING_SAMPLING_RATE"`

	JaegerEndpoint     string `mapstructure:"JAEGER_ENDPOINT"`
	RateLimitEnabled   bool   `mapstructure:"RATE_LIMIT_ENABLED"`
	RateLimitPerSecond int    `mapstructure:"RATE_LIMIT_PER_SECOND"`

	LlmProvider string `mapstructure:"PROVIDER"`
	LlmModel    string `mapstructure:"MODEL"`
	LlmBaseURL  string `mapstructure:"BASE_URL"`
	LlmAPIKey   string `mapstructure:"API_KEY"`

	// WhatsApp
	WebhookPort         string `mapstructure:"WEBHOOK_PORT"`
	WhatsAppAPIURL      string `mapstructure:"WHATSAPP_API_URL"`
	WhatsAppAccessToken string `mapstructure:"WHATSAPP_ACCESS_TOKEN"`
	WhatsAppPhoneNumber string `mapstructure:"WHATSAPP_PHONE_NUMBER"`
	WhatsAppVerifyToken string `mapstructure:"WHATSAPP_VERIFY_TOKEN"`
}

// LoadConfig carrega as configurações do arquivo .env e variáveis de ambiente
func LoadConfig() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	// Carregar .env se existir
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// JWT Expires In
	// jwtExpiresIn := getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour)
	// jwtRefreshExpiresIn := getEnvAsDuration("JWT_REFRESH_EXPIRES_IN", 168*time.Hour) // 7 dias

	// Redis DB
	redisDB := getEnvAsInt("REDIS_DB", 0)

	return &Config{
		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "etools_openerp"),

		// API
		APIPort: getEnv("API_PORT", "8080"),
		APIEnv:  getEnv("API_ENV", "development"),

		// JWT
		JWTSecret:           getEnv("JWT_SECRET", "your-super-secret-key-change-this"),
		JWTExpiresIn:        getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
		JWTRefreshExpiresIn: getEnvAsDuration("JWT_REFRESH_EXPIRES_IN", 168*time.Hour),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,

		// CORS
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods: getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSAllowedHeaders: getEnv("CORS_ALLOWED_HEADERS", "*"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "debug"),
		LogOutput: getEnv("LOG_OUTPUT", "stdout"),

		// Timezone
		TimeZone: getEnv("TIME_ZONE", "America/Sao_Paulo"),

		// Observability (com valores padrão)
		TracingEnabled:      getEnvAsBool("TRACING_ENABLED", false),
		TracingEndpoint:     getEnv("TRACING_ENDPOINT", "localhost:4317"),
		TracingServiceName:  getEnv("TRACING_SERVICE_NAME", "mcp-server"),
		TracingSamplingRate: float64(getEnvAsInt("TRACING_SAMPLING_RATE", 1) / 10),

		// LLM getEnvAsBool("RATE_LIMIT_ENABLED", false),

		JaegerEndpoint:     getEnv("JAEGER_ENDPOINT", "localhost:4317"),
		RateLimitEnabled:   getEnvAsBool("RATE_LIMIT_ENABLED", false),
		RateLimitPerSecond: getEnvAsInt("RATE_LIMIT_PER_SECOND", 100),

		LlmProvider:         getEnv("LLM_PROVIDER", ""),
		LlmModel:            getEnv("LLM_MODEL", ""),
		LlmBaseURL:          getEnv("LLM_BASE_URL", ""),
		LlmAPIKey:           getEnv("LLM_API_KEY", ""),
		WebhookPort:         getEnv("WEBHOOK_PORT", "8080"),
		WhatsAppAPIURL:      getEnv("WHATSAPP_API_URL", ""),
		WhatsAppAccessToken: getEnv("WHATSAPP_ACCESS_TOKEN", ""),
		WhatsAppPhoneNumber: getEnv("WHATSAPP_PHONE_NUMBER", ""),
		WhatsAppVerifyToken: getEnv("WHATSAPP_VERIFY_TOKEN", ""),
	}
}

func getEnvAsBool(s string, defoult bool) bool {
	if value := os.Getenv(s); value != "" {
		if value == "true" || value == "1" {
			return true
		}
	}
	return defoult
}

// GetDSN retorna a string de conexão com o MySQL
func (c *Config) GetDSN() string {
	return c.DBUser + ":" + c.DBPassword +
		"@tcp(" + c.DBHost + ":" + c.DBPort + ")" +
		"/" + c.DBName +
		"?charset=utf8mb4&parseTime=True&loc=Local"
}

// Helper functions para ler variáveis de ambiente
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// LoadConfigOrDefault carrega a configuração ou usa valores padrão
func LoadConfigOrDefault() *Config {
	cfg := LoadConfig()

	return cfg
}

// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config armazena todas as configurações da aplicação
type Config struct {
	// ============================================
	// AMBIENTE
	// ============================================
	Environment string `mapstructure:"ENVIRONMENT"`
	TimeZone    string `mapstructure:"TIME_ZONE"`

	// ============================================
	// SERVIDORES
	// ============================================
	APIPort     string `mapstructure:"API_PORT"`
	WebhookPort string `mapstructure:"WEBHOOK_PORT"`

	// ============================================
	// DATABASE (MySQL)
	// ============================================
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`

	// ============================================
	// REDIS
	// ============================================
	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     string `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	// ============================================
	// LOGGING
	// ============================================
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	LogEncoding string `mapstructure:"LOG_ENCODING"`
	LogOutput   string `mapstructure:"LOG_OUTPUT"`
	LogFile     string `mapstructure:"LOG_FILE"`

	// ============================================
	// JWT (API Authentication)
	// ============================================
	JWTSecret           string        `mapstructure:"JWT_SECRET"`
	JWTExpiresIn        time.Duration `mapstructure:"JWT_EXPIRES_IN"`
	JWTRefreshExpiresIn time.Duration `mapstructure:"JWT_REFRESH_EXPIRES_IN"`

	// ============================================
	// CORS
	// ============================================
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	CORSAllowedMethods string `mapstructure:"CORS_ALLOWED_METHODS"`
	CORSAllowedHeaders string `mapstructure:"CORS_ALLOWED_HEADERS"`

	// ============================================
	// LLM
	// ============================================
	LlmText   string `mapstructure:"LLM_TEXT"`
	LlmAudio  string `mapstructure:"LLM_AUDIO"`
	LlmVision string `mapstructure:"LLM_VISION"`

	LlmDeepSeekApiKey   string `mapstructure:"DEEPSEEK_API_KEY"`
	LlmDeepSeekModel    string `mapstructure:"DEEPSEEK_MODEL"`
	LlmGeminiApiKey     string `mapstructure:"GEMINI_API_KEY"`
	LlmGeminiModel      string `mapstructure:"GEMINI_MODEL"`
	LlmGroqApiKey       string `mapstructure:"GROQ_API_KEY"`
	LlmGroqModel        string `mapstructure:"GROQ_MODEL"`
	LlmGroqWhisperModel string `mapstructure:"GROQ_WHISPER_MODEL"`
	LlmOpenAiApiKey     string `mapstructure:"OPENAI_API_KEY"`
	LlmOpenAiModel      string `mapstructure:"OPENAI_MODEL"`

	// ============================================
	// WHATSAPP WEBHOOK
	// ============================================
	WhatsAppAPIURL      string `mapstructure:"WHATSAPP_API_URL"`
	WhatsAppAccessToken string `mapstructure:"WHATSAPP_ACCESS_TOKEN"`
	WhatsAppPhoneNumber string `mapstructure:"WHATSAPP_PHONE_NUMBER"`
	WhatsAppVerifyToken string `mapstructure:"WHATSAPP_VERIFY_TOKEN"`
	WhatsAppAppSecret   string `mapstructure:"WHATSAPP_APP_SECRET"`

	// ============================================
	// RATE LIMITING
	// ============================================
	RateLimitRequests  int    `mapstructure:"RATE_LIMIT_REQUESTS"`
	RateLimitWindow    string `mapstructure:"RATE_LIMIT_WINDOW"`
	RateLimitOverrides string `mapstructure:"RATE_LIMIT_OVERRIDES"`

	// ============================================
	// TRACING (OpenTelemetry)
	// ============================================
	TracingEnabled      bool    `mapstructure:"TRACING_ENABLED"`
	TracingEndpoint     string  `mapstructure:"TRACING_ENDPOINT"`
	TracingServiceName  string  `mapstructure:"TRACING_SERVICE_NAME"`
	TracingSamplingRate float64 `mapstructure:"TRACING_SAMPLING_RATE"`

	// ============================================
	// SSL/TLS
	// ============================================
	SSLEnabled  bool   `mapstructure:"SSL_ENABLED"`
	SSLCertPath string `mapstructure:"SSL_CERT_PATH"`
	SSLKeyPath  string `mapstructure:"SSL_KEY_PATH"`

	// ============================================
	// HSTS
	// ============================================
	HSTSEnabled           bool `mapstructure:"HSTS_ENABLED"`
	HSTSMaxAge            int  `mapstructure:"HSTS_MAX_AGE"`
	HSTSIncludeSubdomains bool `mapstructure:"HSTS_INCLUDE_SUBDOMAINS"`
	HSTSPreload           bool `mapstructure:"HSTS_PRELOAD"`
}

// LoadConfig carrega as configurações do arquivo .env e variáveis de ambiente
func LoadConfig() (*Config, error) {
	// Tenta carregar .env se existir localmente, ignora erro no Docker
	_ = godotenv.Load()

	// Viper só pra ler ENV, sem arquivo
	viper.AutomaticEnv()

	// Configuração com valores padrão
	cfg := &Config{
		// Ambiente
		Environment: getEnv("ENVIRONMENT", "development"),
		TimeZone:    getEnv("TIME_ZONE", "America/Sao_Paulo"),

		// Servidores
		APIPort:     getEnv("API_PORT", "8081"),
		WebhookPort: getEnv("WEBHOOK_PORT", "8080"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "mcp_server_openerp"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// Logging
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		LogEncoding: getEnv("LOG_ENCODING", "json"),
		LogOutput:   getEnv("LOG_OUTPUT", "stdout"),
		LogFile:     getEnv("LOG_FILE", "logs/mcp-server.log"),

		// JWT
		JWTSecret:           getEnv("JWT_SECRET", "your-super-secret-key-change-this-in-production"),
		JWTExpiresIn:        getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
		JWTRefreshExpiresIn: getEnvAsDuration("JWT_REFRESH_EXPIRES_IN", 168*time.Hour),

		// CORS
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods: getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSAllowedHeaders: getEnv("CORS_ALLOWED_HEADERS", "*"),

		// LLM
		LlmText:   getEnv("LLM_TEXT", ""),
		LlmAudio:  getEnv("LLM_AUDIO", ""),
		LlmVision: getEnv("LLM_VISION", ""),

		// DeepSeek
		LlmDeepSeekApiKey: getEnv("DEEPSEEK_API_KEY", ""),
		LlmDeepSeekModel:  getEnv("DEEPSEEK_MODEL", "deepseek-chat"),

		// Gemini
		LlmGeminiApiKey: getEnv("GEMINI_API_KEY", ""),
		LlmGeminiModel:  getEnv("GEMINI_MODEL", "gemini-2.0-flash"),

		// Groq - 3 usos: chat + whisper + vision
		LlmGroqApiKey:       getEnv("GROQ_API_KEY", ""),
		LlmGroqModel:        getEnv("GROQ_MODEL", "llama-3.3-70b-versatile"),
		LlmGroqWhisperModel: getEnv("GROQ_WHISPER_MODEL", "whisper-large-v3"),

		// OpenAI
		LlmOpenAiApiKey: getEnv("OPENAI_API_KEY", ""),
		LlmOpenAiModel:  getEnv("OPENAI_MODEL", "gpt-4o-mini"),

		// WhatsApp
		WhatsAppAPIURL:      getEnv("WHATSAPP_API_URL", "https://graph.facebook.com/v18.0"),
		WhatsAppAccessToken: getEnv("WHATSAPP_ACCESS_TOKEN", ""),
		WhatsAppPhoneNumber: getEnv("WHATSAPP_PHONE_NUMBER", ""),
		WhatsAppVerifyToken: getEnv("WHATSAPP_VERIFY_TOKEN", ""),
		WhatsAppAppSecret:   getEnv("WHATSAPP_APP_SECRET", ""),

		// Rate Limiting
		RateLimitRequests:  getEnvAsInt("RATE_LIMIT_REQUESTS", 30),
		RateLimitWindow:    getEnv("RATE_LIMIT_WINDOW", "1m"),
		RateLimitOverrides: getEnv("RATE_LIMIT_OVERRIDES", ""),

		// Tracing
		TracingEnabled:      getEnvAsBool("TRACING_ENABLED", false),
		TracingEndpoint:     getEnv("TRACING_ENDPOINT", "localhost:4317"),
		TracingServiceName:  getEnv("TRACING_SERVICE_NAME", "mcp-server"),
		TracingSamplingRate: getEnvAsFloat64("TRACING_SAMPLING_RATE", 0.1),

		// SSL/TLS
		SSLEnabled:  getEnvAsBool("SSL_ENABLED", false),
		SSLCertPath: getEnv("SSL_CERT_PATH", ""),
		SSLKeyPath:  getEnv("SSL_KEY_PATH", ""),

		// HSTS
		HSTSEnabled:           getEnvAsBool("HSTS_ENABLED", true),
		HSTSMaxAge:            getEnvAsInt("HSTS_MAX_AGE", 31536000),
		HSTSIncludeSubdomains: getEnvAsBool("HSTS_INCLUDE_SUBDOMAINS", true),
		HSTSPreload:           getEnvAsBool("HSTS_PRELOAD", false),
	}
	cfg.ValidateLLM()
	return cfg, nil
}

// LoadConfigOrDefault carrega a configuração ou usa valores padrão
func LoadConfigOrDefault() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		// Retorna configuração padrão se houver erro
		return &Config{
			Environment: "development",
			LogLevel:    "info",
			LogEncoding: "json",
			APIPort:     "8081",
			WebhookPort: "8080",
			DBHost:      "localhost",
			DBPort:      "3306",
			DBUser:      "root",
			DBPassword:  "",
			DBName:      "mcp_server_openerp",
			RedisHost:   "localhost",
			RedisPort:   "6379",
			JWTSecret:   "default-secret-change-me",
		}
	}
	return cfg
}

// GetDSN retorna a string de conexão com o MySQL
func (c *Config) GetDSN() string {
	return c.DBUser + ":" + c.DBPassword +
		"@tcp(" + c.DBHost + ":" + c.DBPort + ")" +
		"/" + c.DBName +
		"?charset=utf8mb4&parseTime=True&loc=Local"
}

// ============================================
// FUNÇÕES AUXILIARES
// ============================================

// getEnv retorna o valor da variável de ambiente ou o valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retorna o valor da variável de ambiente como int ou o valor padrão
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool retorna o valor da variável de ambiente como bool ou o valor padrão
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if value == "true" || value == "1" || value == "on" || value == "yes" {
			return true
		}
		if value == "false" || value == "0" || value == "off" || value == "no" {
			return false
		}
	}
	return defaultValue
}

// getEnvAsFloat64 retorna o valor da variável de ambiente como float64 ou o valor padrão
func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvAsDuration retorna o valor da variável de ambiente como time.Duration ou o valor padrão
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func (c *Config) IsProduction() bool {
	env := strings.ToLower(strings.TrimSpace(c.Environment))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	}
	return env == "production" || env == "prod" || env == "prd"
}

func IsProduction() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	}
	return env == "production" || env == "prod" || env == "prd"
}

func (c *Config) ValidateLLM() {
	type fail struct{ Env, Val, Err string }
	var fails []fail
	trim := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	isValid := map[string]bool{"openai": true, "groq": true, "gemini": true, "deepseek": true}

	check := func(env, val, sug string) {
		if strings.TrimSpace(val) == "" {
			fails = append(fails, fail{env, "(vazio)", fmt.Sprintf("obrigatório (Sugestão \"%s\")", sug)})
			return
		}
		if !isValid[trim(val)] {
			fails = append(fails, fail{env, val, "inválido"})
		}
	}
	check("LLM_TEXT", c.LlmText, "gemini")
	check("LLM_AUDIO", c.LlmAudio, "groq")
	check("LLM_VISION", c.LlmVision, "gemini")

	used := map[string]bool{trim(c.LlmText): true, trim(c.LlmAudio): true, trim(c.LlmVision): true}
	if used["openai"] && strings.TrimSpace(c.LlmOpenAiApiKey) == "" {
		fails = append(fails, fail{"OPENAI_API_KEY", "vazia", "precisa p/ openai"})
	}
	if used["groq"] && strings.TrimSpace(c.LlmGroqApiKey) == "" {
		fails = append(fails, fail{"GROQ_API_KEY", "vazia", "precisa p/ groq"})
	}
	if used["gemini"] && strings.TrimSpace(c.LlmGeminiApiKey) == "" {
		fails = append(fails, fail{"GEMINI_API_KEY", "vazia", "precisa p/ gemini"})
	}
	if used["deepseek"] && strings.TrimSpace(c.LlmDeepSeekApiKey) == "" {
		fails = append(fails, fail{"DEEPSEEK_API_KEY", "vazia", "precisa p/ deepseek"})
	}

	if len(fails) == 0 {
		return
	}

	red := "\x1b[31m"
	reset := "\x1b[0m"
	bold := "\x1b[1m"

	fmt.Println()
	fmt.Printf("%s─────────────────────────────────────────────────────────────────────%s", red, reset)
	fmt.Println()
	fmt.Printf("%s%s%s\n", bold, "                         OpenERP MCP-Server", reset)
	fmt.Printf("%s%s%s\n", bold, "          Falha na Validação das Configurações de LLM", reset)
	fmt.Printf("%s─────────────────────────────────────────────────────────────────────%s", red, reset)
	fmt.Println()
	for _, f := range fails {
		// X vermelho
		fmt.Printf(" %s✖%s %-12s = %-10s -> %s\n", red, reset, f.Env, f.Val, f.Err)
	}
	fmt.Println()
	fmt.Printf("%s─────────────────────────────────────────────────────────────────────%s", red, reset)
	fmt.Println()
	fmt.Println(" LLM suportadas: openai, groq, gemini, deepseek")
	fmt.Println(" Defina as variáveis no .env e  reinicie")
	fmt.Printf("%s─────────────────────────────────────────────────────────────────────%s", red, reset)
	fmt.Println()
	fmt.Println()

	os.Exit(1)

}

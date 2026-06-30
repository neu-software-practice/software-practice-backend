package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration values for the application.
type Config struct {
	ServerAddr         string
	ServerMode         string // debug, test, release
	DatabaseDSN        string
	JWTSecret          string
	CORSAllowedOrigins string
	MedAgentMode       string // http, embedded
	MedAgentBaseURL    string
	MedAgentAPIKey     string
	MedAgentProvider   string
	MedAgentModel      string
	LogLevel           string // debug, info, warn, error
}

// JWTSecretMinLen is the minimum required length for JWT_SECRET in bytes.
const JWTSecretMinLen = 32

// Weak JWT secret blacklist patterns.
var weakJWTSecrets = []string{
	"12345678901234567890123456789012",
	"changeme-changeme-changeme-change",
	"secret-secret-secret-secret-sec",
}

// Load reads .env and .env.local files and returns a validated Config.
func Load() (*Config, error) {
	// Try loading .env (ignore file not found)
	_ = godotenv.Load(".env")

	// Try loading .env.local for overrides (ignore file not found)
	_ = godotenv.Overload(".env.local")

	cfg := &Config{
		ServerAddr:         getEnv("SERVER_ADDR", ":8080"),
		ServerMode:         getEnv("SERVER_MODE", "release"),
		DatabaseDSN:        os.Getenv("DATABASE_DSN"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		MedAgentMode:       getEnv("MEDAGENT_MODE", "http"),
		MedAgentBaseURL:    getEnv("MEDAGENT_BASE_URL", "http://localhost:8080"),
		MedAgentAPIKey:     os.Getenv("MEDAGENT_API_KEY"),
		MedAgentProvider:   getEnv("MEDAGENT_PROVIDER", "deepseek"),
		MedAgentModel:      getEnv("MEDAGENT_MODEL", "deepseek-chat"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func (c *Config) validate() error {
	// DATABASE_DSN is required
	if c.DatabaseDSN == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}

	// JWT_SECRET is required
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	// JWT_SECRET must be at least 32 bytes
	if len([]byte(c.JWTSecret)) < JWTSecretMinLen {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes, got %d bytes", JWTSecretMinLen, len([]byte(c.JWTSecret)))
	}

	// JWT_SECRET weak password blacklist check
	for _, weak := range weakJWTSecrets {
		if strings.Contains(c.JWTSecret, weak) {
			return fmt.Errorf("JWT_SECRET is too weak (matches blacklisted pattern)")
		}
	}

	// Production mode cannot use wildcard CORS
	if c.ServerMode == "release" && c.CORSAllowedOrigins == "*" {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot be '*' in release mode")
	}

	// MEDAGENT_API_KEY is required
	if c.MedAgentAPIKey == "" {
		return fmt.Errorf("MEDAGENT_API_KEY is required")
	}

	return nil
}

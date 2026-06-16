// Package config loads runtime configuration from environment variables / .env
// (SPEC §7.2 — no hardcoded secrets). Required secrets fail fast at startup.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime settings.
type Config struct {
	AppEnv      string
	HTTPPort    string
	DatabaseDSN string
	JWTSecret   string
	JWTTTL      time.Duration
	CORSOrigins string
	LogLevel    string
}

// Load reads configuration, applying defaults for non-secret values and
// returning an error when a required secret is missing.
func Load() (*Config, error) {
	_ = godotenv.Load() // best-effort: real env vars win, .env is a convenience

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	ttlHours, err := strconv.Atoi(getEnv("JWT_TTL_HOURS", "12"))
	if err != nil || ttlHours <= 0 {
		return nil, fmt.Errorf("invalid JWT_TTL_HOURS: %q", os.Getenv("JWT_TTL_HOURS"))
	}
	cfg.JWTTTL = time.Duration(ttlHours) * time.Hour

	if cfg.DatabaseDSN == "" {
		return nil, fmt.Errorf("DATABASE_DSN is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

// DatabaseDSN loads just the database DSN. Used by the migrate/seed tools which
// do not need the full app config (e.g. JWT secret).
func DatabaseDSN() (string, error) {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return "", fmt.Errorf("DATABASE_DSN is required")
	}
	return dsn, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

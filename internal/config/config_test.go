package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
)

func TestLoad_Success(t *testing.T) {
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/his")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_TTL_HOURS", "6")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Equal(t, 6*time.Hour, cfg.JWTTTL)
	assert.Equal(t, "*", cfg.CORSOrigins)
}

func TestLoad_DefaultsTTL(t *testing.T) {
	t.Setenv("DATABASE_DSN", "d")
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("JWT_TTL_HOURS", "")
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 12*time.Hour, cfg.JWTTTL)
}

func TestLoad_Errors(t *testing.T) {
	t.Run("missing dsn", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "")
		t.Setenv("JWT_SECRET", "s")
		_, err := config.Load()
		assert.Error(t, err)
	})
	t.Run("missing secret", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "d")
		t.Setenv("JWT_SECRET", "")
		_, err := config.Load()
		assert.Error(t, err)
	})
	t.Run("bad ttl", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "d")
		t.Setenv("JWT_SECRET", "s")
		t.Setenv("JWT_TTL_HOURS", "not-a-number")
		_, err := config.Load()
		assert.Error(t, err)
	})
}

func TestDatabaseDSN(t *testing.T) {
	t.Setenv("DATABASE_DSN", "the-dsn")
	dsn, err := config.DatabaseDSN()
	require.NoError(t, err)
	assert.Equal(t, "the-dsn", dsn)

	t.Setenv("DATABASE_DSN", "")
	_, err = config.DatabaseDSN()
	assert.Error(t, err)
}

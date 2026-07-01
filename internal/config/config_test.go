package config_test

import (
	"os"
	"testing"

	"github.com/neuhis/software-practice-backend/internal/config"
)

// clearEnv unsets all env vars that config.Load() reads, ensuring test isolation.
func clearEnv(t *testing.T) {
	t.Helper()
	_ = os.Unsetenv("DATABASE_DSN")
	_ = os.Unsetenv("JWT_SECRET")
	_ = os.Unsetenv("SERVER_MODE")
	_ = os.Unsetenv("CORS_ALLOWED_ORIGINS")
	_ = os.Unsetenv("SERVER_ADDR")
	_ = os.Unsetenv("MEDAGENT_MODE")
	_ = os.Unsetenv("MEDAGENT_BASE_URL")
	_ = os.Unsetenv("MEDAGENT_API_KEY")
	_ = os.Unsetenv("MEDAGENT_PROVIDER")
	_ = os.Unsetenv("MEDAGENT_MODEL")
	_ = os.Unsetenv("LOG_LEVEL")
}

func TestLoadValidConfig(t *testing.T) {
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-32-byte-secret-key-here!!")
	t.Setenv("MEDAGENT_API_KEY", "test-medagent-api-key")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.DatabaseDSN != "user:pass@tcp(localhost:3306)/testdb" {
		t.Errorf("unexpected DatabaseDSN: got %q, want %q", cfg.DatabaseDSN, "user:pass@tcp(localhost:3306)/testdb")
	}
	if cfg.JWTSecret != "this-is-a-32-byte-secret-key-here!!" {
		t.Errorf("unexpected JWTSecret: got %q, want %q", cfg.JWTSecret, "this-is-a-32-byte-secret-key-here!!")
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T)
		wantErr string
	}{
		{
			name: "MissingDSN",
			setup: func(t *testing.T) {
				// DATABASE_DSN deliberately not set
			},
			wantErr: "DATABASE_DSN is required",
		},
		{
			name: "MissingJWTSecret",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
			},
			wantErr: "JWT_SECRET is required",
		},
		{
			name: "JWTTooShort",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
				t.Setenv("JWT_SECRET", "short")
			},
			wantErr: "JWT_SECRET must be at least 32 bytes, got 5 bytes",
		},
		{
			name: "JWTBoundary31Bytes",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
				// 31 bytes — one short of the 32-byte minimum
				t.Setenv("JWT_SECRET", "abcdefghijklmnopqrstuvwxyz12345")
				t.Setenv("MEDAGENT_API_KEY", "sk-test-key-for-validation")
			},
			wantErr: "JWT_SECRET must be at least 32 bytes, got 31 bytes",
		},
		{
			name: "JWTWeakPassword",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
				t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
			},
			wantErr: "JWT_SECRET is too weak (matches blacklisted pattern)",
		},
		{
			name: "JWTWeakPasswordPattern2",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
				t.Setenv("JWT_SECRET", "changeme-changeme-changeme-change")
			},
			wantErr: "JWT_SECRET is too weak (matches blacklisted pattern)",
		},
		{
			name: "JWTWeakPasswordPattern3",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
				// Pad the blacklisted pattern to exceed 32-byte minimum
				t.Setenv("JWT_SECRET", "extra-secret-secret-secret-secret-sec")
			},
			wantErr: "JWT_SECRET is too weak (matches blacklisted pattern)",
		},
		{
			name: "ProductionWildcardCORS",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/testdb")
				t.Setenv("JWT_SECRET", "this-is-a-32-byte-secret-key-here!!")
				t.Setenv("SERVER_MODE", "release")
				t.Setenv("CORS_ALLOWED_ORIGINS", "*")
			},
			wantErr: "CORS_ALLOWED_ORIGINS cannot be '*' in release mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)

			tt.setup(t)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("unexpected error message:\ngot:  %q\nwant: %q", err.Error(), tt.wantErr)
			}
		})
	}
}

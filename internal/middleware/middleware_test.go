package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/neuhis/software-practice-backend/internal/auth"
	"github.com/neuhis/software-practice-backend/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter() *gin.Engine {
	return gin.New()
}

func TestCORSMiddleware(t *testing.T) {
	t.Run("allows configured origin", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.CORSMiddleware(middleware.CORSConfig{
			AllowedOrigins: "http://localhost:5173",
		}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		r.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
			t.Errorf("origin header = %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Allow-Methods header should be set")
		}
		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("Allow-Headers header should be set")
		}
	})

	t.Run("rejects mismatched origin", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.CORSMiddleware(middleware.CORSConfig{
			AllowedOrigins: "http://localhost:5173",
		}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		r.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set Allow-Origin for mismatched origin")
		}
	})

	t.Run("wildcard cors", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.CORSMiddleware(middleware.CORSConfig{
			AllowedOrigins: "*",
		}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		r.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("origin header = %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("options request", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.CORSMiddleware(middleware.CORSConfig{
			AllowedOrigins: "*",
		}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != 204 {
			t.Errorf("status = %d, want 204", w.Code)
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	r := setupRouter()
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	r := setupRouter()
	r.Use(middleware.LoggingMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
	// Should have request ID header
	if w.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header missing")
	}
}

func TestAuthMiddleware(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101

	t.Run("missing auth header", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.AuthMiddleware(secret))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.AuthMiddleware(secret))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		token, err := middleware.GenerateToken("p001", secret)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		r := setupRouter()
		r.Use(middleware.AuthMiddleware(secret))
		r.GET("/test", func(c *gin.Context) {
			id := middleware.GetPatientID(c)
			if id != "p001" {
				t.Errorf("patientId = %s, want p001", id)
			}
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid auth format", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.AuthMiddleware(secret))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Token some-token")
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an expired JWT token
		now := time.Now()
		expiredClaims := jwt.RegisteredClaims{
			Subject:   "p001",
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		}
		expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("create expired token: %v", err)
		}

		r := setupRouter()
		r.Use(middleware.AuthMiddleware(secret))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("status = %d, want 401 for expired token, body=%s", w.Code, w.Body.String())
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	r := setupRouter()
	r.Use(middleware.RateLimitMiddleware(100, 200))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// Should allow many requests with high rate limit
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: status = %d, want 200", i, w.Code)
		}
	}
}

func TestRateLimitMiddleware_RejectsWhenExhausted(t *testing.T) {
	r := setupRouter()
	r.Use(middleware.RateLimitMiddleware(0.01, 1))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// The first request creates the bucket and is free.
	// The second request consumes the initial capacity-1 token.
	// The third request should be rate-limited.
	acceptCount := 0
	rejectCount := 0
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		switch w.Code {
		case 200:
			acceptCount++
		case http.StatusTooManyRequests:
			rejectCount++
		}
	}

	if acceptCount == 0 {
		t.Error("expected at least one accepted request")
	}
	if rejectCount == 0 {
		t.Error("expected at least one rejected request when bucket exhausted")
	}

	// Verify rejection body contains RATE_LIMITED
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	if w.Code == http.StatusTooManyRequests {
		if !strings.Contains(w.Body.String(), "RATE_LIMITED") {
			t.Errorf("rejection body does not contain RATE_LIMITED: %s", w.Body.String())
		}
	}
}

func TestGenerateToken(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101
	token, err := middleware.GenerateToken("p001", secret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if token == "" {
		t.Error("token empty")
	}
}

func TestGetPatientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("patientId", "p001")

	id := middleware.GetPatientID(c)
	if id != "p001" {
		t.Errorf("got %s, want p001", id)
	}
}

func TestRequirePatientID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("with patient id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("patientId", "p001")

		r := setupRouter()
		r.Use(middleware.RequirePatientID())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})
		r.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	})

	t.Run("without patient id", func(t *testing.T) {
		r := setupRouter()
		r.Use(middleware.RequirePatientID())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})
}

func TestTokenExpired(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if middleware.TokenExpired(nil) {
			t.Error("nil error should not be expired")
		}
	})

	t.Run("expired error", func(t *testing.T) {
		err := fmt.Errorf("token is expired: %w", jwt.ErrTokenExpired)
		if !middleware.TokenExpired(err) {
			t.Error("expired error should be detected")
		}
	})

	t.Run("other error", func(t *testing.T) {
		err := fmt.Errorf("invalid signature")
		if middleware.TokenExpired(err) {
			t.Error("other error should not be expired")
		}
	})
}

func TestGetPatientIDNone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	id := middleware.GetPatientID(c)
	if id != "" {
		t.Errorf("got %s, want empty", id)
	}
}

func TestGenerateAccessToken(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101
	token, err := auth.GenerateAccessToken("u001", "p001", "13800001111", secret)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
}

func TestAuthMiddleware_NewTokenFormat(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101

	token, err := auth.GenerateAccessToken("u001", "p001", "13800001111", secret)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	r := setupRouter()
	r.Use(middleware.AuthMiddleware(secret))
	r.GET("/test", func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		patientID := middleware.GetPatientID(c)
		if userID != "u001" {
			t.Errorf("userId = %s, want u001", userID)
		}
		if patientID != "p001" {
			t.Errorf("patientId = %s, want p001", patientID)
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware_PatientIDClaimAliases(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        "u001",
		"patient_id": "p001",
		"iat":        now.Unix(),
		"exp":        now.Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	r := setupRouter()
	r.Use(middleware.AuthMiddleware(secret))
	r.GET("/test", func(c *gin.Context) {
		if got := middleware.GetPatientID(c); got != "p001" {
			t.Errorf("patientId = %s, want p001", got)
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware_LegacyTokenBackwardCompat(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101

	token, err := middleware.GenerateToken("p001", secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	r := setupRouter()
	r.Use(middleware.AuthMiddleware(secret))
	r.GET("/test", func(c *gin.Context) {
		patientID := middleware.GetPatientID(c)
		if patientID != "p001" {
			t.Errorf("patientId = %s, want p001", patientID)
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("with userId", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("userId", "u001")

		id := middleware.GetUserID(c)
		if id != "u001" {
			t.Errorf("got %s, want u001", id)
		}
	})

	t.Run("without userId", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		id := middleware.GetUserID(c)
		if id != "" {
			t.Errorf("got %s, want empty", id)
		}
	})
}

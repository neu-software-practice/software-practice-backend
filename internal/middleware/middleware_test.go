package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	secret := "this-is-a-32-byte-secret-key-for-testing!!"

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

func TestGenerateToken(t *testing.T) {
	secret := "this-is-a-32-byte-secret-key-for-testing!!"
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

func TestGetPatientIDNone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	id := middleware.GetPatientID(c)
	if id != "" {
		t.Errorf("got %s, want empty", id)
	}
}

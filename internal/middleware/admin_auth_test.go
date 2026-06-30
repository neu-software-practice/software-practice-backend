package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "this-is-a-32-byte-secret-key-for-testing!!"

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAdminAuthMiddleware_NoAuthHeader(t *testing.T) {
	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthMiddleware_InvalidFormat(t *testing.T) {
	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormatToken")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthMiddleware_InvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthMiddleware_ValidToken(t *testing.T) {
	token, err := GenerateAdminAccessToken("admin-1", "super_admin", testJWTSecret, 900)
	if err != nil {
		t.Fatalf("GenerateAdminAccessToken: %v", err)
	}

	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		adminID := GetAdminID(c)
		role := GetAdminRole(c)
		c.JSON(200, gin.H{"adminId": adminID, "role": role})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !contains(w.Body.String(), `"adminId":"admin-1"`) {
		t.Errorf("response should contain adminId, got: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"role":"super_admin"`) {
		t.Errorf("response should contain role, got: %s", w.Body.String())
	}
}

func TestAdminAuthMiddleware_PatientTokenRejected(t *testing.T) {
	// Create a patient-style token (no role claim)
	patientToken, err := GenerateAccessToken("u1", "p1", "13800001111", testJWTSecret)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+patientToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (patient token should be rejected)", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthMiddleware_WrongSigningKey(t *testing.T) {
	token, err := GenerateAdminAccessToken("admin-1", "admin", "a-different-secret-key-that-is-32-bytes!!", 900)
	if err != nil {
		t.Fatalf("GenerateAdminAccessToken: %v", err)
	}

	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAdminRole_Success(t *testing.T) {
	token, err := GenerateAdminAccessToken("admin-1", "super_admin", testJWTSecret, 900)
	if err != nil {
		t.Fatalf("GenerateAdminAccessToken: %v", err)
	}

	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.Use(RequireAdminRole("super_admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAdminRole_InsufficientRole(t *testing.T) {
	token, err := GenerateAdminAccessToken("admin-1", "operator", testJWTSecret, 900)
	if err != nil {
		t.Fatalf("GenerateAdminAccessToken: %v", err)
	}

	router := gin.New()
	router.Use(AdminAuthMiddleware(testJWTSecret))
	router.Use(RequireAdminRole("super_admin", "admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGenerateAdminAccessToken_Expiration(t *testing.T) {
	token, err := GenerateAdminAccessToken("admin-1", "admin", testJWTSecret, 900)
	if err != nil {
		t.Fatalf("GenerateAdminAccessToken: %v", err)
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "admin-1" {
		t.Errorf("sub = %v, want admin-1", claims["sub"])
	}
	if claims["role"] != "admin" {
		t.Errorf("role = %v, want admin", claims["role"])
	}
	if _, ok := claims["exp"]; !ok {
		t.Error("token missing exp claim")
	}
	if _, ok := claims["iat"]; !ok {
		t.Error("token missing iat claim")
	}
}

func TestGetAdminID_NotSet(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		id := GetAdminID(c)
		if id != "" {
			t.Errorf("GetAdminID = %q, want empty", id)
		}
		c.JSON(200, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	_ = w
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

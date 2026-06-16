package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
)

func init() { gin.SetMode(gin.TestMode) }

func tokens() *jwt.Manager { return jwt.NewManager("middleware-test-secret-0123", time.Hour) }

func do(engine *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestAuth(t *testing.T) {
	tk := tokens()
	good, err := tk.Generate(5, "王医生", constant.DeptTypeOutpatient)
	require.NoError(t, err)

	engine := gin.New()
	engine.GET("/x", middleware.Auth(tk), func(c *gin.Context) {
		assert.EqualValues(t, 5, middleware.CurrentEmployeeID(c))
		assert.Equal(t, constant.DeptTypeOutpatient, middleware.CurrentDeptType(c))
		c.String(http.StatusOK, "ok")
	})

	cases := []struct {
		name, header string
		want         int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"wrong scheme", "Token " + good, http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"garbage token", "Bearer a.b.c", http.StatusUnauthorized},
		{"valid", "Bearer " + good, http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, do(engine, http.MethodGet, "/x", c.header).Code)
		})
	}
}

func TestRequireDeptType(t *testing.T) {
	tk := tokens()
	tok := func(dt string) string {
		s, err := tk.Generate(1, "n", dt)
		require.NoError(t, err)
		return "Bearer " + s
	}

	guarded := func(method string) *gin.Engine {
		e := gin.New()
		g := e.Group("/", middleware.Auth(tk), middleware.RequireDeptType(constant.DeptTypeOutpatient))
		g.Handle(method, "/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
		return e
	}

	assert.Equal(t, http.StatusOK, do(guarded(http.MethodGet), http.MethodGet, "/x", tok(constant.DeptTypeOutpatient)).Code)
	assert.Equal(t, http.StatusForbidden, do(guarded(http.MethodGet), http.MethodGet, "/x", tok(constant.DeptTypeFinance)).Code)
	// root is a read-only observer.
	assert.Equal(t, http.StatusOK, do(guarded(http.MethodGet), http.MethodGet, "/x", tok(constant.DeptTypeRoot)).Code)
	assert.Equal(t, http.StatusForbidden, do(guarded(http.MethodPost), http.MethodPost, "/x", tok(constant.DeptTypeRoot)).Code)

	// RequireDeptType without prior Auth → unauthorized.
	noAuth := gin.New()
	noAuth.GET("/x", middleware.RequireDeptType(constant.DeptTypeOutpatient), func(c *gin.Context) { c.String(200, "ok") })
	assert.Equal(t, http.StatusUnauthorized, do(noAuth, http.MethodGet, "/x", "").Code)
}

func TestCORS(t *testing.T) {
	engine := gin.New()
	engine.Use(middleware.CORS("*"))
	engine.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// Preflight short-circuits with 204.
	w := do(engine, http.MethodOptions, "/x", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	w = do(engine, http.MethodGet, "/x", "")
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Allowlist mode echoes only known origins.
	allow := gin.New()
	allow.Use(middleware.CORS("http://good.com,http://ok.com"))
	allow.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://good.com")
	rec := httptest.NewRecorder()
	allow.ServeHTTP(rec, req)
	assert.Equal(t, "http://good.com", rec.Header().Get("Access-Control-Allow-Origin"))

	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec = httptest.NewRecorder()
	allow.ServeHTTP(rec, req)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestRecoveryAndLogger(t *testing.T) {
	engine := gin.New()
	engine.Use(middleware.Recovery(), middleware.Logger())
	engine.GET("/panic", func(_ *gin.Context) { panic("boom") })
	engine.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	assert.Equal(t, http.StatusInternalServerError, do(engine, http.MethodGet, "/panic", "").Code)
	assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, "/ok", "").Code)
}

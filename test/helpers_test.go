// Package test holds black-box integration tests that drive the assembled HTTP
// server through the unified envelope, against the test database (PLAN §5).
package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/app"
	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/router"
	"github.com/neu-software-practice/software-practice-backend/internal/seed"
	"github.com/neu-software-practice/software-practice-backend/internal/testutil"
)

const seedPassword = "Passw0rd!"

// envelope mirrors response.Body for assertions.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Meta *struct {
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
		Total int64 `json:"total"`
	} `json:"meta"`
}

// newServer builds an engine on a fresh seeded database.
func newServer(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewDB(t)
	require.NoError(t, seed.Run(db, seedPassword))

	cfg := &config.Config{
		AppEnv:      "test",
		HTTPPort:    "0",
		JWTSecret:   "integration-test-secret-0123456789",
		JWTTTL:      time.Hour,
		CORSOrigins: "*",
		LogLevel:    "error",
	}
	return router.New(app.NewContainer(db, cfg).Deps()), db
}

// doJSON performs a request and decodes the envelope.
func doJSON(t *testing.T, engine *gin.Engine, method, path, token string, body interface{}) (*httptest.ResponseRecorder, envelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	var env envelope
	if rec.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "body: %s", rec.Body.String())
	}
	return rec, env
}

// login authenticates and returns the bearer token.
func login(t *testing.T, engine *gin.Engine, username string) string {
	t.Helper()
	rec, env := doJSON(t, engine, http.MethodPost, "/api/auth/login", "",
		dto.LoginRequest{Username: username, Password: seedPassword})
	require.Equalf(t, http.StatusOK, rec.Code, "login %s failed: %s", username, rec.Body.String())
	var lr dto.LoginResponse
	require.NoError(t, json.Unmarshal(env.Data, &lr))
	require.NotEmpty(t, lr.Token)
	return lr.Token
}

// decodeData unmarshals the envelope data into dst.
func decodeData(t *testing.T, env envelope, dst interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(env.Data, dst))
}

package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSwaggerServed verifies the OpenAPI UI and spec are reachable (SPEC §9.5).
func TestSwaggerServed(t *testing.T) {
	engine, _ := newServer(t)

	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		assert.Equalf(t, http.StatusOK, rec.Code, "GET %s", path)
	}

	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	body := rec.Body.String()
	assert.Contains(t, body, "/auth/login")
	assert.Contains(t, body, "/charges")
}

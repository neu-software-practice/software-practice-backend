package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

func init() { gin.SetMode(gin.TestMode) }

func testContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) response.Body {
	t.Helper()
	var b response.Body
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	return b
}

func TestSuccessAndCreated(t *testing.T) {
	c, w := testContext()
	response.Success(c, gin.H{"a": 1})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, decode(t, w).Success)

	c, w = testContext()
	response.Created(c, gin.H{"id": 2})
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, decode(t, w).Success)
}

func TestList(t *testing.T) {
	c, w := testContext()
	response.List(c, []int{1, 2}, response.Meta{Page: 1, Limit: 10, Total: 2})
	assert.Equal(t, http.StatusOK, w.Code)
	body := decode(t, w)
	require.NotNil(t, body.Meta)
	assert.EqualValues(t, 2, body.Meta.Total)
}

func TestError_AppError(t *testing.T) {
	c, w := testContext()
	response.Error(c, apperr.ErrForbidden)
	assert.Equal(t, http.StatusForbidden, w.Code)
	body := decode(t, w)
	assert.False(t, body.Success)
	require.NotNil(t, body.Error)
	assert.Equal(t, "FORBIDDEN", body.Error.Code)
}

func TestError_GenericIsMaskedAs500(t *testing.T) {
	c, w := testContext()
	response.Error(c, errors.New("raw db failure with secrets"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := decode(t, w)
	require.NotNil(t, body.Error)
	assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
	// The raw error text must never reach the client.
	assert.NotContains(t, w.Body.String(), "secrets")
}

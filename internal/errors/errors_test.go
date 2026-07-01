package errors_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
)

func TestNewApiError(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		message       string
		status        int
		wantRetriable bool
	}{
		{"session not found", apperrors.CodeSessionNotFound, "session not found", 404, false},
		{"patient not found", apperrors.CodePatientNotFound, "patient not found", 404, false},
		{"validation error", apperrors.CodeValidationError, "invalid input", 400, false},
		{"card not found", apperrors.CodeCardNotFound, "card not found", 404, true},
		{"internal error 5xx", apperrors.CodeInternalError, "internal error", 500, true},
		{"unknown error 5xx", apperrors.CodeUnknownError, "unknown", 502, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apperrors.NewApiError(tt.code, tt.message, tt.status)
			if err.Code != tt.code {
				t.Errorf("Code = %s, want %s", err.Code, tt.code)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %s, want %s", err.Message, tt.message)
			}
			if err.Status != tt.status {
				t.Errorf("Status = %d, want %d", err.Status, tt.status)
			}
			if err.Retriable != tt.wantRetriable {
				t.Errorf("Retriable = %v, want %v", err.Retriable, tt.wantRetriable)
			}
			if err.Error() == "" {
				t.Error("Error() should return a non-empty string")
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		err := apperrors.NewValidationError("bad input")
		if err.Code != apperrors.CodeValidationError {
			t.Errorf("code = %s", err.Code)
		}
		if err.Status != 422 {
			t.Errorf("status = %d, want 422", err.Status)
		}
	})

	t.Run("NewNotFoundError", func(t *testing.T) {
		err := apperrors.NewNotFoundError(apperrors.CodeSessionNotFound, "gone")
		if err.Status != 404 {
			t.Errorf("status = %d", err.Status)
		}
	})

	t.Run("NewUnauthorizedError", func(t *testing.T) {
		err := apperrors.NewUnauthorizedError("no auth")
		if err.Status != 401 {
			t.Errorf("status = %d", err.Status)
		}
	})

	t.Run("NewForbiddenError", func(t *testing.T) {
		err := apperrors.NewForbiddenError("no access")
		if err.Status != 403 {
			t.Errorf("status = %d", err.Status)
		}
	})

	t.Run("NewInternalError", func(t *testing.T) {
		err := apperrors.NewInternalError("boom")
		if err.Status != 500 {
			t.Errorf("status = %d", err.Status)
		}
	})
}

func TestApiErrorJSON(t *testing.T) {
	err := apperrors.NewApiError(apperrors.CodeSessionNotFound, "not found", 404)
	b, _ := json.Marshal(err)

	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)

	if parsed["code"] != "SESSION_NOT_FOUND" {
		t.Error("code mismatch")
	}
	if parsed["status"] != float64(404) {
		t.Error("status mismatch")
	}
}

func TestWriteError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("write validation error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		apperrors.WriteValidationError(c, "invalid")

		if w.Code != 422 {
			t.Errorf("status = %d, want 422", w.Code)
		}
		// Verify body contains the error code and message
		var body map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if errMap, ok := body["error"].(map[string]interface{}); ok {
			if errMap["code"] != "VALIDATION_ERROR" {
				t.Errorf("error code = %v, want VALIDATION_ERROR", errMap["code"])
			}
		}
	})

	t.Run("write not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "gone")

		if w.Code != 404 {
			t.Errorf("status = %d, want 404", w.Code)
		}
		// Verify body contains the error code
		var body map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if errMap, ok := body["error"].(map[string]interface{}); ok {
			if errMap["code"] != "SESSION_NOT_FOUND" {
				t.Errorf("error code = %v, want SESSION_NOT_FOUND", errMap["code"])
			}
		}
	})

	t.Run("write unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		apperrors.WriteUnauthorized(c, "no token")

		if w.Code != 401 {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("write forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		apperrors.WriteForbidden(c, "no access")

		if w.Code != 403 {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})

	t.Run("write internal error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		apperrors.WriteError(c, apperrors.NewInternalError("boom"))

		if w.Code != 500 {
			t.Errorf("status = %d, want 500", w.Code)
		}
	})

	t.Run("write internal error via helper", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		apperrors.WriteInternalError(c, "server error")

		if w.Code != 500 {
			t.Errorf("status = %d, want 500", w.Code)
		}
	})
}

func TestErrorCodes(t *testing.T) {
	codes := []string{
		apperrors.CodeSessionNotFound,
		apperrors.CodePatientNotFound,
		apperrors.CodeCardNotFound,
		apperrors.CodeValidationError,
		apperrors.CodeUnknownError,
		apperrors.CodeNetworkError,
		apperrors.CodeUnauthorized,
		apperrors.CodeForbidden,
		apperrors.CodeNotFound,
		apperrors.CodeInternalError,
		apperrors.CodeAuthPhoneExists,
		apperrors.CodeAuthInvalidCredentials,
		apperrors.CodeAuthTokenExpired,
		apperrors.CodeAuthRefreshInvalid,
		apperrors.CodeAuthRefreshExpired,
		apperrors.CodeRateLimited,
		apperrors.CodeAddressNotFound,
		apperrors.CodeAddressLimitExceeded,
		apperrors.CodeAddressRequired,
		apperrors.CodeAdminInvalidCredentials,
		apperrors.CodeAdminInvalidRefreshToken,
		apperrors.CodeAdminInvalidSettings,
	}
	for _, c := range codes {
		if c == "" {
			t.Error("found empty error code")
		}
	}
}

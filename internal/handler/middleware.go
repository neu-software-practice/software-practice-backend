package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

// BindJSON binds the request body to the given struct.
func BindJSON[T any](c *gin.Context) (T, error) {
	var input T
	if err := c.ShouldBindJSON(&input); err != nil {
		return input, err
	}
	return input, nil
}

// ParseSessionID extracts the session ID from the URL path.
func ParseSessionID(c *gin.Context) string {
	return c.Param("sessionId")
}

// ParsePatientID extracts the patient ID from the URL path.
func ParsePatientID(c *gin.Context) string {
	return c.Param("patientId")
}

// ParseAddressID extracts the address ID from the URL path.
func ParseAddressID(c *gin.Context) string {
	return c.Param("addressId")
}

// ParseQueryInt parses an integer query parameter with a default value.
func ParseQueryInt(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

// WriteSuccess writes a success response using the unified envelope.
func WriteSuccess[T any](c *gin.Context, status int, data T) {
	c.JSON(status, api.SuccessResponse(data))
}

// WriteSuccessWithMeta writes a success response with metadata.
func WriteSuccessWithMeta[T any](c *gin.Context, status int, data T, meta interface{}) {
	c.JSON(status, api.SuccessResponseWithMeta(data, meta))
}

// WritePageResult writes a paginated response.
func WritePageResult[T any](c *gin.Context, page api.PageResult[T]) {
	c.JSON(http.StatusOK, api.SuccessResponse(page))
}

// GetPatientIDFromContext extracts the authenticated patient ID.
func GetPatientIDFromContext(c *gin.Context) string {
	id, exists := c.Get("patientId")
	if !exists {
		return ""
	}
	s, _ := id.(string)
	return s
}

// RequirePatientID checks that the authenticated patient ID matches the request.
func RequirePatientID(c *gin.Context, requestPatientID string) bool {
	authID := GetPatientIDFromContext(c)
	if authID == "" || authID != requestPatientID {
		apperrors.WriteForbidden(c, "access denied: patient ID mismatch")
		return false
	}
	return true
}

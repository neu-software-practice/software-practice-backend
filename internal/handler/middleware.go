package handler

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

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
	c.JSON(status, data)
}

// WritePageResult writes a paginated response.
func WritePageResult[T any](c *gin.Context, page api.PageResult[T]) {
	c.JSON(http.StatusOK, page)
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

// ResolveOwnPatientID checks access to patient-scoped resources and returns the
// canonical patient ID from the JWT. Some frontends have historically passed a
// display name in the patientId path slot; allow that compatibility form only
// for non-ASCII aliases and still scope data access to the authenticated patient.
func ResolveOwnPatientID(c *gin.Context, requestPatientID string) (string, bool) {
	authID := GetPatientIDFromContext(c)
	if authID == "" {
		apperrors.WriteForbidden(c, "access denied: patient ID mismatch")
		return "", false
	}
	if authID == requestPatientID {
		return authID, true
	}
	if isNonASCIIAlias(requestPatientID) {
		return authID, true
	}

	apperrors.WriteForbidden(c, "access denied: patient ID mismatch")
	return "", false
}

func isNonASCIIAlias(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == utf8.RuneError {
			return false
		}
		if r > 127 {
			return true
		}
	}
	return false
}

package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/neuhis/software-practice-backend/internal/auth"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
)

// AuthMiddleware creates a JWT authentication middleware.
// It extracts the token from the Authorization header and injects patientId into the context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apperrors.WriteUnauthorized(c, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			apperrors.WriteUnauthorized(c, "invalid authorization format, expected 'Bearer <token>'")
			return
		}

		claims, err := auth.ParseJWT(parts[1], jwtSecret)
		if err != nil {
			if TokenExpired(err) {
				apperrors.WriteError(c, apperrors.NewApiError(
					apperrors.CodeAuthTokenExpired,
					"access token expired",
					401,
				))
			} else {
				apperrors.WriteUnauthorized(c, "invalid or expired token")
			}
			return
		}

		userID, _ := claims["sub"].(string)

		patientID, _ := claims["patientId"].(string)
		if patientID == "" {
			patientID = userID
		}

		if patientID == "" {
			apperrors.WriteUnauthorized(c, "token missing required claims")
			return
		}

		c.Set("userId", userID)
		c.Set("patientId", patientID)
		if phone, ok := claims["phone"].(string); ok {
			c.Set("phone", phone)
		}
		c.Next()
	}
}

// GetPatientID extracts the patient ID from the Gin context.
func GetPatientID(c *gin.Context) string {
	id, _ := c.Get("patientId")
	if id == nil {
		return ""
	}
	return id.(string)
}

// RequirePatientID is a middleware that ensures the context has a patientId.
func RequirePatientID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetPatientID(c) == "" {
			apperrors.WriteUnauthorized(c, "patient not authenticated")
			return
		}
		c.Next()
	}
}

// GenerateAccessToken creates a JWT access token with full claims.
// Maintained for backward compatibility; delegates to the shared auth package.
func GenerateAccessToken(userID, patientID, phone, secret string) (string, error) {
	return auth.GenerateAccessToken(userID, patientID, phone, secret)
}

// GenerateToken creates a JWT token for a patient (legacy, used in tests).
func GenerateToken(patientID, secret string) (string, error) {
	return auth.GenerateAccessToken(patientID, patientID, "", secret)
}

// GetUserID extracts the user ID from the Gin context.
func GetUserID(c *gin.Context) string {
	id, _ := c.Get("userId")
	if id == nil {
		return ""
	}
	return id.(string)
}

// TokenExpired checks if the error is due to token expiration.
func TokenExpired(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "token is expired")
}

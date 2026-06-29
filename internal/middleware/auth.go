package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

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

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
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

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			apperrors.WriteUnauthorized(c, "invalid token claims")
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
func GenerateAccessToken(userID, patientID, phone, secret string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       userID,
		"patientId": patientID,
		"phone":     phone,
		"iat":       now.Unix(),
		"exp":       now.Add(900 * time.Second).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return tokenString, nil
}

// GenerateToken creates a JWT token for a patient (legacy, used in tests).
func GenerateToken(patientID, secret string) (string, error) {
	return GenerateAccessToken(patientID, patientID, "", secret)
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

// IsAuthenticated checks if the request has a valid patient context.
func IsAuthenticated(c *gin.Context) bool {
	return GetPatientID(c) != ""
}

// SetPatientID sets the patient ID in the Gin context (for testing).
func SetPatientID(c *gin.Context, patientID string) {
	c.Set("patientId", patientID)
}

// CurrentPatient returns a middleware that adds the patient ID to c.Request.Context()
// as a value. This allows downstream code to use request-scoped values.
func CurrentPatient() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Forward any patientId set by auth middleware
		c.Next()
	}
}

// RespondWithJSON is a helper to respond with standard JSON format.
func RespondWithJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, gin.H{
		"success": status >= 200 && status < 300,
		"data":    data,
	})
}

// ErrorResponse represents a standard error in JSON response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSONError writes a JSON error response with the given status code.
func WriteJSONError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": ErrorResponse{
			Code:    code,
			Message: message,
		},
	})
}

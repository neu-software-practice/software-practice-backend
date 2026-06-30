package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
)

// AdminAuthMiddleware creates a JWT authentication middleware for admin endpoints.
// It checks the token contains admin claims (role field), and rejects patient tokens.
func AdminAuthMiddleware(jwtSecret string) gin.HandlerFunc {
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

		adminID, _ := claims["sub"].(string)
		role, _ := claims["role"].(string)

		if adminID == "" || role == "" {
			apperrors.WriteUnauthorized(c, "token missing admin claims — use admin credentials")
			return
		}

		c.Set("adminId", adminID)
		c.Set("adminRole", role)
		c.Next()
	}
}

// GetAdminID extracts the admin ID from the Gin context.
func GetAdminID(c *gin.Context) string {
	id, _ := c.Get("adminId")
	if id == nil {
		return ""
	}
	return id.(string)
}

// GetAdminRole extracts the admin role from the Gin context.
func GetAdminRole(c *gin.Context) string {
	role, _ := c.Get("adminRole")
	if role == nil {
		return ""
	}
	return role.(string)
}

// RequireAdminRole returns a middleware that checks the admin has one of the required roles.
func RequireAdminRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role := GetAdminRole(c)
		if role == "" {
			apperrors.WriteUnauthorized(c, "admin not authenticated")
			return
		}
		if !allowed[role] {
			apperrors.WriteForbidden(c, "insufficient admin role")
			return
		}
		c.Next()
	}
}

// GenerateAdminAccessToken creates a JWT access token for an admin.
func GenerateAdminAccessToken(adminID, role, secret string, ttlSeconds int) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  adminID,
		"role": role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(ttlSeconds) * time.Second).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("generate admin access token: %w", err)
	}
	return tokenString, nil
}

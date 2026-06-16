package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

// Auth validates the Bearer token and stores its claims on the context.
func Auth(tokens *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			response.Error(c, apperr.ErrUnauthorized)
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if raw == "" {
			response.Error(c, apperr.ErrUnauthorized)
			return
		}
		claims, err := tokens.Parse(raw)
		if err != nil {
			response.Error(c, apperr.ErrUnauthorized)
			return
		}
		setClaims(c, claims)
		c.Next()
	}
}

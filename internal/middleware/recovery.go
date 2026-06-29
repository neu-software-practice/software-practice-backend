package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
)

// RecoveryMiddleware catches panics and returns structured JSON error responses.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				log.Printf("PANIC recovered: %v\n%s", r, stack)

				c.AbortWithStatusJSON(http.StatusInternalServerError, apperrors.NewApiError(
					apperrors.CodeInternalError,
					"internal server error",
					http.StatusInternalServerError,
				))
			}
		}()
		c.Next()
	}
}

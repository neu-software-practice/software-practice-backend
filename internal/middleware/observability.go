package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

// Recovery converts panics into a unified 500 response and logs the stack
// context server-side (never leaking it to the client).
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					"err", r,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
				)
				if !c.Writer.Written() {
					response.Error(c, apperr.ErrInternal)
				}
			}
		}()
		c.Next()
	}
}

// Logger emits a structured access log line per request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(),
		)
	}
}

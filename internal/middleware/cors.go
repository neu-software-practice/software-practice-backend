package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins string
}

// CORSMiddleware creates a CORS middleware from the given config.
func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	allowedOrigins := cfg.AllowedOrigins

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if allowedOrigins == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// Check if the origin is in the allowed list
			origins := strings.Split(allowedOrigins, ",")
			for _, o := range origins {
				if strings.TrimSpace(o) == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Package router wires the HTTP routes and middleware (PLAN §2.1, §4). Routes are
// grouped under /api and guarded per dept_type via the RBAC middleware.
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/handler"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

// Deps carries everything the router needs. The app container populates it.
type Deps struct {
	Cfg    *config.Config
	Tokens *jwt.Manager
	Auth   *handler.AuthHandler
}

// New builds the configured Gin engine.
func New(d Deps) *gin.Engine {
	if d.Cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Logger(), middleware.CORS(d.Cfg.CORSOrigins))

	r.GET("/api/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	registerAuthRoutes(api, d)

	return r
}

func registerAuthRoutes(api *gin.RouterGroup, d Deps) {
	auth := api.Group("/auth")
	auth.POST("/login", d.Auth.Login)
	auth.GET("/me", middleware.Auth(d.Tokens), d.Auth.Me)
}

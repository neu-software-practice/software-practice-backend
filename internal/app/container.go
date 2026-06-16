// Package app is the composition root: it wires repositories → services →
// handlers and exposes the router Deps so cmd/server stays tiny.
package app

import (
	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/handler"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
	"github.com/neu-software-practice/software-practice-backend/internal/router"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// Container holds long-lived application dependencies.
type Container struct {
	cfg    *config.Config
	tokens *jwt.Manager
	deps   router.Deps
}

// NewContainer constructs every layer from the DB connection and config.
func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	tokens := jwt.NewManager(cfg.JWTSecret, cfg.JWTTTL)

	// Repositories.
	employees := repository.NewEmployeeRepository(db)

	// Services.
	authSvc := service.NewAuthService(employees, tokens)

	// Handlers + router deps.
	deps := router.Deps{
		Cfg:    cfg,
		Tokens: tokens,
		Auth:   handler.NewAuthHandler(authSvc),
	}

	return &Container{cfg: cfg, tokens: tokens, deps: deps}
}

// Deps returns the router dependencies.
func (c *Container) Deps() router.Deps { return c.deps }

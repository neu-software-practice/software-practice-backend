package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// AdminRepository defines the data access interface for admin users.
type AdminRepository interface {
	FindByUsername(ctx context.Context, username string) (*model.AdminUser, error)
	FindByID(ctx context.Context, id string) (*model.AdminUser, error)
	Create(ctx context.Context, admin *model.AdminUser) error
}

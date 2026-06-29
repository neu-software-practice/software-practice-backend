package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// UserRepository defines the data access interface for users.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

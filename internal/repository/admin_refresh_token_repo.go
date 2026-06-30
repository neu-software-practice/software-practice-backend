package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// AdminRefreshTokenRepository defines the data access interface for admin refresh tokens.
type AdminRefreshTokenRepository interface {
	Create(ctx context.Context, token *model.AdminRefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.AdminRefreshToken, error)
	MarkUsed(ctx context.Context, id string) error
	RevokeAllByAdminID(ctx context.Context, adminID string) error
}

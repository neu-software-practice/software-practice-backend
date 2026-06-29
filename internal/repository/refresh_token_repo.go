package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// RefreshTokenRepository defines the data access interface for refresh tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	MarkUsed(ctx context.Context, id string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

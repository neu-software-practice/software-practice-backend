package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// FlowCardRepository defines the data access interface for flow cards.
type FlowCardRepository interface {
	Create(ctx context.Context, card *model.FlowCard) error
	FindByID(ctx context.Context, id string) (*model.FlowCard, error)
	ListBySession(ctx context.Context, sessionID string) ([]model.FlowCard, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	Update(ctx context.Context, card *model.FlowCard) error
}

package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// TimelineRepository defines the data access interface for timeline items.
type TimelineRepository interface {
	Append(ctx context.Context, item *model.TimelineItem) error
	AppendBatch(ctx context.Context, items []model.TimelineItem) error
	ListBySession(ctx context.Context, sessionID string, cursor *string, pageSize int) ([]model.TimelineItem, *string, bool, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}

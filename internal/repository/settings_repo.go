package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// SettingsRepository defines the data access interface for system settings.
type SettingsRepository interface {
	Get(ctx context.Context) (*model.SystemSettings, error)
	Update(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error)
}

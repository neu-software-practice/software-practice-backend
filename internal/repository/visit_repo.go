package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// VisitRepository defines the data access interface for visit sessions.
type VisitRepository interface {
	Create(ctx context.Context, visit *model.VisitSession) error
	FindByID(ctx context.Context, id string) (*model.VisitSession, error)
	ListByPatient(ctx context.Context, patientID string, status string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
	UpdateStatus(ctx context.Context, id string, status string, machineState string) error
	Update(ctx context.Context, visit *model.VisitSession) error
}

package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// AddressRepository defines the data access interface for patient delivery addresses.
type AddressRepository interface {
	Create(ctx context.Context, addr *model.Address) error
	FindByID(ctx context.Context, id string) (*model.Address, error)
	ListByPatient(ctx context.Context, patientID string) ([]model.Address, error)
	CountByPatient(ctx context.Context, patientID string) (int, error)
	Update(ctx context.Context, addr *model.Address) error
	Delete(ctx context.Context, id string) error
	ClearDefaultByPatient(ctx context.Context, patientID string) error
	SetDefault(ctx context.Context, id string, patientID string) error
}

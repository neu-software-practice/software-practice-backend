package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// PatientRepository defines the data access interface for patients.
type PatientRepository interface {
	FindByCredential(ctx context.Context, credType, credential string) (*model.PatientProfile, error)
	FindByID(ctx context.Context, id string) (*model.PatientProfile, error)
	Create(ctx context.Context, patient *model.PatientProfile) error
	UpdateProfile(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error)
}

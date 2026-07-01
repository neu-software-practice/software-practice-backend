package patient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

// Service handles patient-related business logic.
type Service struct {
	patientRepo repository.PatientRepository
	visitRepo   repository.VisitRepository
}

// NewService creates a new PatientService.
func NewService(patientRepo repository.PatientRepository, visitRepo repository.VisitRepository) *Service {
	return &Service{
		patientRepo: patientRepo,
		visitRepo:   visitRepo,
	}
}

// VerifyIdentity verifies a patient's identity by credential.
// If the patient is not found, a new patient profile is created.
func (s *Service) VerifyIdentity(ctx context.Context, input model.VerifyIdentityInput) (*model.VerifyIdentityResult, error) {
	patient, err := s.patientRepo.FindByCredential(ctx, input.CredentialType, input.Credential)
	if errors.Is(err, model.ErrPatientNotFound) {
		// Create new patient
		patient = &model.PatientProfile{
			ID:                  uuid.New().String(),
			Name:                input.Name,
			Gender:              string(model.GenderUnknown),
			Age:                 0,
			Allergies:           []string{},
			ChronicDiseases:     []string{},
			LongTermMedications: []string{},
			MedicalHistory:      []string{},
			UpdatedAt:           time.Now(),
		}
		if input.CredentialType == "phone" {
			patient.PhoneMasked = input.Credential
		} else {
			patient.IDCardMasked = input.Credential
		}
		if err := s.patientRepo.Create(ctx, patient); err != nil {
			return nil, fmt.Errorf("create patient: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("find patient: %w", err)
	}

	scopes := []string{"profile", "history", "allergies", "medications"}

	return &model.VerifyIdentityResult{
		Patient:        *patient,
		ReadableScopes: scopes,
		VerifiedAt:     time.Now(),
	}, nil
}

// GetContext retrieves the full patient context including prior visit summary.
func (s *Service) GetContext(ctx context.Context, patientID string) (*model.PatientContext, error) {
	patient, err := s.patientRepo.FindByID(ctx, patientID)
	if err != nil {
		return nil, err
	}

	ctx2 := &model.PatientContext{
		Patient: *patient,
	}

	// Get last completed visit for prior visit summary
	summaries, _, _, err := s.visitRepo.ListByPatient(ctx, patientID, nil, 1)
	if err == nil && len(summaries) > 0 {
		last := summaries[0]
		completedAt := last.UpdatedAt
		if last.EndedAt != nil {
			completedAt = *last.EndedAt
		}
		ctx2.PriorVisit = &model.PatientPriorVisit{
			SessionID:        last.ID,
			CompletedAt:      completedAt,
			Diagnosis:        stringDeref(last.Summary.Diagnosis),
			TreatmentSummary: stringDeref(last.Summary.TreatmentSummary),
		}
	}

	return ctx2, nil
}

// UpdateProfile updates a patient's profile (allergies, chronic diseases, medications).
func (s *Service) UpdateProfile(ctx context.Context, patientID string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
	return s.patientRepo.UpdateProfile(ctx, patientID, input)
}

func stringDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

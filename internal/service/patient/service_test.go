package patient_test

import (
	"context"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/service/patient"
)

// mockPatientRepo implements repository.PatientRepository for testing.
type mockPatientRepo struct {
	findByCredFunc func(ctx context.Context, credType, credential string) (*model.PatientProfile, error)
	findByIDFunc   func(ctx context.Context, id string) (*model.PatientProfile, error)
	createFunc     func(ctx context.Context, p *model.PatientProfile) error
	updateFunc     func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error)
}

func (m *mockPatientRepo) FindByCredential(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
	return m.findByCredFunc(ctx, credType, credential)
}
func (m *mockPatientRepo) FindByID(ctx context.Context, id string) (*model.PatientProfile, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockPatientRepo) Create(ctx context.Context, p *model.PatientProfile) error {
	return m.createFunc(ctx, p)
}
func (m *mockPatientRepo) UpdateProfile(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
	return m.updateFunc(ctx, id, input)
}

// mockVisitRepo implements repository.VisitRepository for testing.
type mockVisitRepo struct {
	listByPatientFunc func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error { return nil }
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return nil, nil
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, patientID, cursor, pageSize)
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id string, status string, machineState string) error {
	return nil
}
func (m *mockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error { return nil }

func TestVerifyIdentity_NewPatient(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			return nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "phone",
		Credential:     "13800001111",
		Name:           "测试",
	}

	result, err := svc.VerifyIdentity(ctx, input)
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if result.Patient.Name != "测试" {
		t.Errorf("name = %s, want 测试", result.Patient.Name)
	}
	if len(result.ReadableScopes) != 4 {
		t.Errorf("scopes = %d, want 4", len(result.ReadableScopes))
	}
}

func TestVerifyIdentity_ExistingPatient(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:        "p001",
				Name:      "张三",
				Gender:    "male",
				Age:       35,
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "phone",
		Credential:     "13800001111",
	}

	result, err := svc.VerifyIdentity(ctx, input)
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if result.Patient.ID != "p001" {
		t.Errorf("id = %s, want p001", result.Patient.ID)
	}
}

func TestGetContext(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:                  "p001",
				Name:                "张三",
				Allergies:           []string{"青霉素"},
				ChronicDiseases:     []string{"高血压"},
				LongTermMedications: []string{"阿司匹林"},
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{}, nil, false, nil
		},
	}

	svc := patient.NewService(patientRepo, visitRepo)

	ctx2, err := svc.GetContext(ctx, "p001")
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if ctx2.Patient.ID != "p001" {
		t.Errorf("id = %s", ctx2.Patient.ID)
	}
	if len(ctx2.Allergies) != 1 {
		t.Errorf("allergies = %d", len(ctx2.Allergies))
	}
}

func TestUpdateProfile(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		updateFunc: func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:                  id,
				Name:                "张三",
				Allergies:           input.Allergies,
				ChronicDiseases:     input.ChronicDiseases,
				LongTermMedications: input.LongTermMedications,
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.ProfileUpdateInput{
		PatientID: "p001",
		Allergies: []string{"头孢"},
	}

	result, err := svc.UpdateProfile(ctx, "p001", input)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if len(result.Allergies) != 1 || result.Allergies[0] != "头孢" {
		t.Error("allergies not updated")
	}
}

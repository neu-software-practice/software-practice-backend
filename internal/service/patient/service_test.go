package patient_test

import (
	"context"
	"errors"
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
	listByPatientFunc func(ctx context.Context, patientID string, _ string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error { return nil }
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return nil, nil
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, patientID string, status string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, patientID, status, cursor, pageSize)
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id string, status string, machineState string) error {
	return nil
}
func (m *mockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error { return nil }

// ---------------------------------------------------------------------------
// VerifyIdentity tests
// ---------------------------------------------------------------------------

func TestVerifyIdentity_NewPatient(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			if p.PhoneMasked != "13800001111" {
				t.Errorf("PhoneMasked = %q, want 13800001111", p.PhoneMasked)
			}
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
	// New patient should start with unknown gender and empty allergies/chronic diseases/medications
	if result.Patient.Gender != "unknown" {
		t.Errorf("gender = %s, want unknown", result.Patient.Gender)
	}
	if result.Patient.PhoneMasked != "13800001111" {
		t.Errorf("phoneMasked = %s, want 13800001111", result.Patient.PhoneMasked)
	}
}

func TestVerifyIdentity_NewPatient_IDCard(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			if p.IDCardMasked != "110101199001011234" {
				t.Errorf("IDCardMasked = %q, want 110101199001011234", p.IDCardMasked)
			}
			return nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "id_card",
		Credential:     "110101199001011234",
		Name:           "张三",
	}

	result, err := svc.VerifyIdentity(ctx, input)
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if result.Patient.Name != "张三" {
		t.Errorf("name = %s, want 张三", result.Patient.Name)
	}
	if result.Patient.IDCardMasked != "110101199001011234" {
		t.Errorf("idCardMasked = %s, want 110101199001011234", result.Patient.IDCardMasked)
	}
	// PhoneMasked should be empty when credential type is not "phone"
	if result.Patient.PhoneMasked != "" {
		t.Errorf("phoneMasked = %s, want empty", result.Patient.PhoneMasked)
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
	if len(result.ReadableScopes) != 4 {
		t.Errorf("scopes = %d, want 4", len(result.ReadableScopes))
	}
}

func TestVerifyIdentity_UnknownCredentialType(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			// For unknown credential types, the else branch sets IDCardMasked
			if p.IDCardMasked != "user@example.com" {
				t.Errorf("IDCardMasked = %q, want user@example.com", p.IDCardMasked)
			}
			if p.PhoneMasked != "" {
				t.Errorf("PhoneMasked = %q, want empty", p.PhoneMasked)
			}
			return nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "email",
		Credential:     "user@example.com",
		Name:           "李四",
	}

	result, err := svc.VerifyIdentity(ctx, input)
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if result.Patient.Name != "李四" {
		t.Errorf("name = %s, want 李四", result.Patient.Name)
	}
	// Unknown credential type falls to else branch and sets IDCardMasked
	if result.Patient.IDCardMasked != "user@example.com" {
		t.Errorf("IDCardMasked = %s, want user@example.com", result.Patient.IDCardMasked)
	}
	if result.Patient.PhoneMasked != "" {
		t.Errorf("PhoneMasked = %s, want empty", result.Patient.PhoneMasked)
	}
}

func TestVerifyIdentity_EmptyCredential(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			// Empty credential should still create a patient with empty masked field
			if p.PhoneMasked != "" {
				t.Errorf("PhoneMasked = %q, want empty", p.PhoneMasked)
			}
			return nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "phone",
		Credential:     "",
		Name:           "王五",
	}

	result, err := svc.VerifyIdentity(ctx, input)
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if result.Patient.Name != "王五" {
		t.Errorf("name = %s, want 王五", result.Patient.Name)
	}
	if result.Patient.PhoneMasked != "" {
		t.Errorf("PhoneMasked = %s, want empty", result.Patient.PhoneMasked)
	}
}

func TestVerifyIdentity_RepoError(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, errors.New("db connection failed")
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "phone",
		Credential:     "13800001111",
	}

	_, err := svc.VerifyIdentity(ctx, input)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "find patient: db connection failed" {
		t.Errorf("error = %q, want %q", err.Error(), "find patient: db connection failed")
	}
}

func TestVerifyIdentity_CreateFails(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			return errors.New("insert failed")
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "phone",
		Credential:     "13800001111",
		Name:           "赵六",
	}

	_, err := svc.VerifyIdentity(ctx, input)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "create patient: insert failed" {
		t.Errorf("error = %q, want %q", err.Error(), "create patient: insert failed")
	}
}

// ---------------------------------------------------------------------------
// GetContext tests
// ---------------------------------------------------------------------------

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
				MedicalHistory:      []string{"腰椎间盘突出"},
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, _ string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
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
	if len(ctx2.Patient.Allergies) != 1 {
		t.Errorf("allergies = %d", len(ctx2.Patient.Allergies))
	}
	if ctx2.PriorVisit != nil {
		t.Error("expected PriorVisit to be nil")
	}
	if len(ctx2.Patient.MedicalHistory) != 1 || ctx2.Patient.MedicalHistory[0] != "腰椎间盘突出" {
		t.Errorf("MedicalHistory = %v, want [腰椎间盘突出]", ctx2.Patient.MedicalHistory)
	}
	if len(ctx2.Patient.LongTermMedications) != 1 || ctx2.Patient.LongTermMedications[0] != "阿司匹林" {
		t.Errorf("LongTermMedications = %v, want [阿司匹林]", ctx2.Patient.LongTermMedications)
	}
}

func TestGetContext_WithPriorVisit_NilSummaryFields(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:                  "p001",
				Name:                "张三",
				Allergies:           []string{},
				ChronicDiseases:     []string{},
				LongTermMedications: []string{},
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, _ string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{
					ID:        "v002",
					PatientID: "p001",
					Status:    "completed",
					Summary: model.VisitSummary{
						Diagnosis:        nil,
						TreatmentSummary: nil,
					},
				},
			}, nil, true, nil
		},
	}

	svc := patient.NewService(patientRepo, visitRepo)

	ctx2, err := svc.GetContext(ctx, "p001")
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if ctx2.PriorVisit == nil {
		t.Fatal("expected PriorVisit to be non-nil")
	}
	if ctx2.PriorVisit.Diagnosis != "" {
		t.Errorf("Diagnosis = %q, want empty string", ctx2.PriorVisit.Diagnosis)
	}
	if ctx2.PriorVisit.TreatmentSummary != "" {
		t.Errorf("TreatmentSummary = %q, want empty string", ctx2.PriorVisit.TreatmentSummary)
	}
}

func TestGetContext_WithPriorVisit(t *testing.T) {
	ctx := context.Background()

	diagnosis := "上呼吸道感染"
	treatmentSummary := "开具头孢类抗生素"

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:                  "p001",
				Name:                "张三",
				Allergies:           []string{"青霉素"},
				ChronicDiseases:     []string{},
				LongTermMedications: []string{},
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, _ string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{
					ID:        "v001",
					PatientID: "p001",
					Status:    "completed",
					Summary: model.VisitSummary{
						Diagnosis:        &diagnosis,
						TreatmentSummary: &treatmentSummary,
					},
				},
			}, nil, true, nil
		},
	}

	svc := patient.NewService(patientRepo, visitRepo)

	ctx2, err := svc.GetContext(ctx, "p001")
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if ctx2.PriorVisit == nil {
		t.Fatal("expected PriorVisit to be non-nil")
	}
	if ctx2.PriorVisit.SessionID != "v001" {
		t.Errorf("SessionID = %s, want v001", ctx2.PriorVisit.SessionID)
	}
	if ctx2.PriorVisit.Diagnosis != diagnosis {
		t.Errorf("Diagnosis = %s, want %s", ctx2.PriorVisit.Diagnosis, diagnosis)
	}
	if ctx2.PriorVisit.TreatmentSummary != treatmentSummary {
		t.Errorf("TreatmentSummary = %s, want %s", ctx2.PriorVisit.TreatmentSummary, treatmentSummary)
	}
}

func TestGetContext_PatientNotFound(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	_, err := svc.GetContext(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !errors.Is(err, model.ErrPatientNotFound) {
		t.Errorf("error = %v, want ErrPatientNotFound", err)
	}
}

func TestGetContext_ListError(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:                  "p001",
				Name:                "张三",
				Allergies:           []string{"青霉素"},
				ChronicDiseases:     []string{"高血压"},
				LongTermMedications: []string{"阿司匹林"},
				MedicalHistory:      []string{"2024年阑尾炎手术"},
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	listErr := errors.New("list failed")
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, _ string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
			return nil, nil, false, listErr
		},
	}

	svc := patient.NewService(patientRepo, visitRepo)

	ctx2, err := svc.GetContext(ctx, "p001")
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if ctx2.Patient.ID != "p001" {
		t.Errorf("id = %s, want p001", ctx2.Patient.ID)
	}
	if ctx2.PriorVisit != nil {
		t.Error("expected PriorVisit to be nil when ListByPatient fails")
	}
	if len(ctx2.Patient.Allergies) != 1 || ctx2.Patient.Allergies[0] != "青霉素" {
		t.Errorf("allergies = %v, want [青霉素]", ctx2.Patient.Allergies)
	}
	if len(ctx2.Patient.MedicalHistory) != 1 || ctx2.Patient.MedicalHistory[0] != "2024年阑尾炎手术" {
		t.Errorf("MedicalHistory = %v, want [2024年阑尾炎手术]", ctx2.Patient.MedicalHistory)
	}
}

// ---------------------------------------------------------------------------
// UpdateProfile tests
// ---------------------------------------------------------------------------

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

func TestUpdateProfile_NotFound(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		updateFunc: func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.ProfileUpdateInput{
		PatientID: "nonexistent",
		Allergies: []string{"头孢"},
	}

	_, err := svc.UpdateProfile(ctx, "nonexistent", input)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !errors.Is(err, model.ErrPatientNotFound) {
		t.Errorf("error = %v, want ErrPatientNotFound", err)
	}
}

func TestUpdateProfile_NilSlices(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		updateFunc: func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
			// Verify all slices are nil (default zero values)
			if input.Allergies != nil {
				t.Error("Allergies should be nil")
			}
			if input.ChronicDiseases != nil {
				t.Error("ChronicDiseases should be nil")
			}
			if input.LongTermMedications != nil {
				t.Error("LongTermMedications should be nil")
			}
			if input.MedicalHistory != nil {
				t.Error("MedicalHistory should be nil")
			}
			return &model.PatientProfile{
				ID:                  id,
				Name:                "张三",
				Allergies:           []string{},
				ChronicDiseases:     []string{},
				LongTermMedications: []string{},
				UpdatedAt:           time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	// Input with nil slices (zero values) — only PatientID set
	input := model.ProfileUpdateInput{
		PatientID: "p001",
	}

	result, err := svc.UpdateProfile(ctx, "p001", input)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if result.ID != "p001" {
		t.Errorf("id = %s, want p001", result.ID)
	}
}

func TestUpdateProfile_MedicalHistory(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		updateFunc: func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:             id,
				Name:           "张三",
				MedicalHistory: input.MedicalHistory,
				UpdatedAt:      time.Now(),
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	t.Run("set medical history", func(t *testing.T) {
		input := model.ProfileUpdateInput{
			PatientID:      "p001",
			MedicalHistory: []string{"慢性咽炎病史3年", "2024年阑尾炎手术"},
		}
		result, err := svc.UpdateProfile(ctx, "p001", input)
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if len(result.MedicalHistory) != 2 {
			t.Errorf("MedicalHistory len = %d, want 2", len(result.MedicalHistory))
		}
	})

	t.Run("clear medical history", func(t *testing.T) {
		input := model.ProfileUpdateInput{
			PatientID:      "p001",
			MedicalHistory: []string{},
		}
		result, err := svc.UpdateProfile(ctx, "p001", input)
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if len(result.MedicalHistory) != 0 {
			t.Errorf("MedicalHistory len = %d, want 0", len(result.MedicalHistory))
		}
	})
}

func TestVerifyIdentity_NewPatient_EmptyMedicalHistory(t *testing.T) {
	ctx := context.Background()

	var created *model.PatientProfile
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			created = p
			return nil
		},
	}
	visitRepo := &mockVisitRepo{}

	svc := patient.NewService(patientRepo, visitRepo)

	input := model.VerifyIdentityInput{
		CredentialType: "phone",
		Credential:     "13900001111",
		Name:           "新患者",
	}

	_, err := svc.VerifyIdentity(ctx, input)
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if created == nil {
		t.Fatal("patient was not created")
	}
	if created.MedicalHistory == nil {
		t.Error("MedicalHistory should not be nil")
	}
	if len(created.MedicalHistory) != 0 {
		t.Errorf("MedicalHistory = %v, want empty slice", created.MedicalHistory)
	}
}

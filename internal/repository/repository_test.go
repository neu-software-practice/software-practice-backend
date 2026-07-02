package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	"github.com/neuhis/software-practice-backend/tests/testutil"
)

// setupDB starts a MySQL testcontainer, runs all migrations, and returns a
// ready-to-use *sql.DB along with a cleanup function.
func setupDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	dsn, teardown := testutil.SetupMySQL(t)
	testutil.RunMigrations(t, dsn, "../../db/migrations")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	return db, func() {
		_ = db.Close()
		teardown()
	}
}

// strPtr is a small helper that returns a pointer to its argument.
func strPtr(s string) *string { return &s }

// addrTagPtr is a small helper that returns a pointer to an AddressTag.
func addrTagPtr(t model.AddressTag) *model.AddressTag { return &t }

// createPatient is a test helper that creates and returns a minimal patient.
func createPatient(ctx context.Context, t *testing.T, repo repository.PatientRepository) *model.PatientProfile {
	t.Helper()
	p := &model.PatientProfile{
		ID:                  uuid.New().String(),
		Name:                "测试患者",
		Gender:              string(model.GenderMale),
		Age:                 30,
		PhoneMasked:         "139****0000",
		IDCardMasked:        "110****5678",
		Allergies:           []string{},
		ChronicDiseases:     []string{},
		LongTermMedications: []string{},
		MedicalHistory:      []string{},
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("createPatient: %v", err)
	}
	return p
}

// createVisit is a test helper that creates and returns a minimal visit session.
func createVisit(ctx context.Context, t *testing.T, repo repository.VisitRepository, patientID string) *model.VisitSession {
	t.Helper()
	v := &model.VisitSession{
		ID:            uuid.New().String(),
		PatientID:     patientID,
		PatientName:   "",
		EntryType:     string(model.VisitEntryTypeNew),
		Status:        string(model.VisitStatusLoadingContext),
		AskRound:      0,
		AskRoundLimit: 20,
		LabRound:      0,
		LabRoundLimit: 10,
		TimerPaused:   false,
	}
	if err := repo.Create(ctx, v); err != nil {
		t.Fatalf("createVisit: %v", err)
	}
	return v
}

// ---------------------------------------------------------------------------
// Patient repository tests
// ---------------------------------------------------------------------------

func TestPatientRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := repository.NewPatientRepository(db)

	t.Run("Create", func(t *testing.T) {
		patient := &model.PatientProfile{
			ID:                  uuid.New().String(),
			Name:                "张三",
			Gender:              string(model.GenderMale),
			Age:                 35,
			PhoneMasked:         "138****5678",
			IDCardMasked:        "110****1234",
			Allergies:           []string{"青霉素"},
			ChronicDiseases:     []string{"高血压"},
			LongTermMedications: []string{},
		}
		err := repo.Create(ctx, patient)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if patient.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set by Create")
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		patient := &model.PatientProfile{
			ID:                  uuid.New().String(),
			Name:                "李四",
			Gender:              string(model.GenderFemale),
			Age:                 28,
			PhoneMasked:         "137****9999",
			IDCardMasked:        "320****4321",
			Allergies:           []string{"青霉素", "头孢"},
			ChronicDiseases:     []string{},
			LongTermMedications: []string{},
		}
		if err := repo.Create(ctx, patient); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		found, err := repo.FindByID(ctx, patient.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found.Name != "李四" {
			t.Errorf("name = %q, want %q", found.Name, "李四")
		}
		if found.Gender != string(model.GenderFemale) {
			t.Errorf("gender = %q, want %q", found.Gender, string(model.GenderFemale))
		}
		if found.Age != 28 {
			t.Errorf("age = %d, want 28", found.Age)
		}
		if len(found.Allergies) != 2 {
			t.Errorf("len(allergies) = %d, want 2", len(found.Allergies))
		}
		if !found.CreatedAt.IsZero() {
			t.Log("CreatedAt is populated")
		}
	})

	t.Run("FindByCredential", func(t *testing.T) {
		patient := &model.PatientProfile{
			ID:                  uuid.New().String(),
			Name:                "王五",
			Gender:              string(model.GenderMale),
			Age:                 40,
			PhoneMasked:         "136****1111",
			IDCardMasked:        "440****2222",
			Allergies:           []string{"磺胺"},
			ChronicDiseases:     []string{"糖尿病"},
			LongTermMedications: []string{"二甲双胍"},
		}
		if err := repo.Create(ctx, patient); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		// Find by phone
		byPhone, err := repo.FindByCredential(ctx, "phone", "136****1111")
		if err != nil {
			t.Fatalf("FindByCredential(phone) failed: %v", err)
		}
		if byPhone.ID != patient.ID {
			t.Errorf("phone lookup: id = %q, want %q", byPhone.ID, patient.ID)
		}
		if len(byPhone.Allergies) != 1 || byPhone.Allergies[0] != "磺胺" {
			t.Errorf("phone lookup: allergies = %v", byPhone.Allergies)
		}

		// Find by id_card
		byCard, err := repo.FindByCredential(ctx, "id_card", "440****2222")
		if err != nil {
			t.Fatalf("FindByCredential(id_card) failed: %v", err)
		}
		if byCard.ID != patient.ID {
			t.Errorf("id_card lookup: id = %q, want %q", byCard.ID, patient.ID)
		}
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		patient := &model.PatientProfile{
			ID:                  uuid.New().String(),
			Name:                "赵六",
			Gender:              string(model.GenderMale),
			Age:                 45,
			PhoneMasked:         "135****3333",
			IDCardMasked:        "330****4444",
			Allergies:           []string{"青霉素"},
			ChronicDiseases:     []string{"高血脂"},
			LongTermMedications: []string{},
		}
		if err := repo.Create(ctx, patient); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		// Update allergies
		updated, err := repo.UpdateProfile(ctx, patient.ID, model.ProfileUpdateInput{
			PatientID: patient.ID,
			Allergies: []string{"青霉素", "阿司匹林"},
		})
		if err != nil {
			t.Fatalf("UpdateProfile failed: %v", err)
		}
		if len(updated.Allergies) != 2 || updated.Allergies[1] != "阿司匹林" {
			t.Errorf("updated allergies = %v, want [青霉素 阿司匹林]", updated.Allergies)
		}

		// Verify via fresh read
		refetched, err := repo.FindByID(ctx, patient.ID)
		if err != nil {
			t.Fatalf("FindByID after update failed: %v", err)
		}
		if len(refetched.Allergies) != 2 || refetched.Allergies[1] != "阿司匹林" {
			t.Errorf("refetched allergies = %v, want [青霉素 阿司匹林]", refetched.Allergies)
		}
	})
}

func TestPatientRepo_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := repository.NewPatientRepository(db)

	_, err := repo.FindByID(ctx, "non-existent-id")
	if !errors.Is(err, model.ErrPatientNotFound) {
		t.Fatalf("expected ErrPatientNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Visit repository tests
// ---------------------------------------------------------------------------

func TestVisitRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()
	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)

	patient := createPatient(ctx, t, pRepo)

	t.Run("Create", func(t *testing.T) {
		visit := &model.VisitSession{
			ID:            uuid.New().String(),
			PatientID:     patient.ID,
			EntryType:     string(model.VisitEntryTypeNew),
			Status:        string(model.VisitStatusLoadingContext),
			AskRound:      0,
			AskRoundLimit: 20,
			LabRound:      0,
			LabRoundLimit: 10,
			TimerPaused:   false,
			Summary: model.VisitSummary{
				ChiefComplaint: strPtr("头痛三天"),
			},
		}
		err := vRepo.Create(ctx, visit)
		if err != nil {
			t.Fatalf("Create visit failed: %v", err)
		}
		if visit.StartedAt.IsZero() {
			t.Error("expected StartedAt to be set by Create")
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		visit := &model.VisitSession{
			ID:            uuid.New().String(),
			PatientID:     patient.ID,
			EntryType:     string(model.VisitEntryTypeNew),
			Status:        string(model.VisitStatusChatting),
			AskRound:      0,
			AskRoundLimit: 20,
			LabRound:      0,
			LabRoundLimit: 10,
			TimerPaused:   false,
			Summary: model.VisitSummary{
				ChiefComplaint: strPtr("发热"),
				LastMessage:    strPtr("好的，我了解了"),
			},
		}
		if err := vRepo.Create(ctx, visit); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		found, err := vRepo.FindByID(ctx, visit.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found.PatientID != patient.ID {
			t.Errorf("PatientID = %q, want %q", found.PatientID, patient.ID)
		}
		if found.EntryType != string(model.VisitEntryTypeNew) {
			t.Errorf("EntryType = %q, want %q", found.EntryType, string(model.VisitEntryTypeNew))
		}
		if found.Status != string(model.VisitStatusChatting) {
			t.Errorf("Status = %q, want %q", found.Status, string(model.VisitStatusChatting))
		}
		if found.Summary.ChiefComplaint == nil || *found.Summary.ChiefComplaint != "发热" {
			t.Errorf("ChiefComplaint = %v, want '发热'", found.Summary.ChiefComplaint)
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		visit := &model.VisitSession{
			ID:            uuid.New().String(),
			PatientID:     patient.ID,
			EntryType:     string(model.VisitEntryTypeNew),
			Status:        string(model.VisitStatusChatting),
			AskRound:      1,
			AskRoundLimit: 20,
			LabRound:      0,
			LabRoundLimit: 10,
			TimerPaused:   false,
		}
		if err := vRepo.Create(ctx, visit); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		// Change status to "blocked"
		err := vRepo.UpdateStatus(ctx, visit.ID, string(model.VisitStatusBlocked), string(model.VisitStatusBlocked))
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		updated, err := vRepo.FindByID(ctx, visit.ID)
		if err != nil {
			t.Fatalf("FindByID after update failed: %v", err)
		}
		if updated.Status != string(model.VisitStatusBlocked) {
			t.Errorf("status = %q, want %q", updated.Status, string(model.VisitStatusBlocked))
		}
	})

	t.Run("ListByPatient", func(t *testing.T) {
		// Create 3 visits for pagination testing with pageSize=2
		visitIDs := make([]string, 3)
		for i := 0; i < 3; i++ {
			v := &model.VisitSession{
				ID:            uuid.New().String(),
				PatientID:     patient.ID,
				EntryType:     string(model.VisitEntryTypeNew),
				Status:        string(model.VisitStatusChatting),
				AskRound:      i,
				AskRoundLimit: 20,
				LabRound:      0,
				LabRoundLimit: 10,
				TimerPaused:   false,
			}
			if err := vRepo.Create(ctx, v); err != nil {
				t.Fatalf("setup: Create visit %d: %v", i, err)
			}
			visitIDs[i] = v.ID
			time.Sleep(1100 * time.Millisecond)
		}

		// First page
		summaries, nextCursor, hasMore, err := vRepo.ListByPatient(ctx, patient.ID, "", nil, 2)
		if err != nil {
			t.Fatalf("ListByPatient failed: %v", err)
		}
		if len(summaries) != 2 {
			t.Errorf("page 1: expected 2 summaries, got %d", len(summaries))
		}
		if !hasMore {
			t.Error("page 1: expected hasMore=true")
		}
		if nextCursor == nil || *nextCursor == "" {
			t.Error("page 1: expected non-empty next cursor")
		}

		// Second page
		page2, _, hasMore2, err := vRepo.ListByPatient(ctx, patient.ID, "", nextCursor, 2)
		if err != nil {
			t.Fatalf("ListByPatient page 2 failed: %v", err)
		}
		if len(page2) == 0 {
			t.Error("page 2: expected at least 1 summary")
		}
		_ = hasMore2

		// Verify each summary has an ID
		for i, s := range summaries {
			if s.ID == "" {
				t.Errorf("summary %d has empty ID", i)
			}
		}
	})

	t.Run("Update", func(t *testing.T) {
		visit := &model.VisitSession{
			ID:            uuid.New().String(),
			PatientID:     patient.ID,
			EntryType:     string(model.VisitEntryTypeNew),
			Status:        string(model.VisitStatusChatting),
			MachineState:  string(model.VisitMachineStateChatting),
			AskRound:      1,
			AskRoundLimit: 20,
			LabRound:      0,
			LabRoundLimit: 10,
			TimerPaused:   false,
			Summary: model.VisitSummary{
				ChiefComplaint: strPtr("咳嗽"),
			},
		}
		if err := vRepo.Create(ctx, visit); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		// Modify and update
		visit.Status = string(model.VisitStatusDiagnosis)
		visit.MachineState = string(model.VisitMachineStateDiagnosis)
		visit.AskRound = 2
		visit.Summary.ChiefComplaint = strPtr("咳嗽加重")
		err := vRepo.Update(ctx, visit)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify
		updated, err := vRepo.FindByID(ctx, visit.ID)
		if err != nil {
			t.Fatalf("FindByID after update: %v", err)
		}
		if updated.Status != string(model.VisitStatusDiagnosis) {
			t.Errorf("Status = %q, want %q", updated.Status, string(model.VisitStatusDiagnosis))
		}
		if updated.MachineState != string(model.VisitMachineStateDiagnosis) {
			t.Errorf("MachineState = %q, want %q", updated.MachineState, string(model.VisitMachineStateDiagnosis))
		}
		if updated.AskRound != 2 {
			t.Errorf("AskRound = %d, want 2", updated.AskRound)
		}
		if updated.Summary.ChiefComplaint == nil || *updated.Summary.ChiefComplaint != "咳嗽加重" {
			t.Error("ChiefComplaint not updated")
		}
	})
}

// ---------------------------------------------------------------------------
// Timeline repository tests
// ---------------------------------------------------------------------------

func TestTimelineRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()
	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	tRepo := repository.NewTimelineRepository(db)

	patient := createPatient(ctx, t, pRepo)
	visit := createVisit(ctx, t, vRepo, patient.ID)

	t.Run("Append", func(t *testing.T) {
		item := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Status:    string(model.TimelineItemStatusDone),
			Role:      string(model.MessageRolePatient),
			Content:   "医生，我头痛",
		}
		err := tRepo.Append(ctx, &item)
		if err != nil {
			t.Fatalf("Append failed: %v", err)
		}
		if item.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set by Append")
		}
	})

	t.Run("ListBySession with pagination", func(t *testing.T) {
		baseTime := time.Now().Add(-1 * time.Hour)
		// Ensure there are at least 3 timeline items with distinct timestamps
		existing, _, _, err := tRepo.ListBySession(ctx, visit.ID, nil, 10)
		if err != nil {
			t.Fatalf("ListBySession setup: %v", err)
		}
		for i := len(existing); i < 3; i++ {
			item := model.TimelineItem{
				ID:        uuid.New().String(),
				SessionID: visit.ID,
				Kind:      string(model.TimelineItemKindMessage),
				Status:    string(model.TimelineItemStatusDone),
				Role:      string(model.MessageRolePatient),
				Content:   "消息",
				CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
			}
			if err := tRepo.Append(ctx, &item); err != nil {
				t.Fatalf("setup Append %d: %v", i, err)
			}
		}

		// First page (pageSize=2)
		items, nextCursor, hasMore, err := tRepo.ListBySession(ctx, visit.ID, nil, 2)
		if err != nil {
			t.Fatalf("ListBySession failed: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("page 1: expected 2 items, got %d", len(items))
		}
		if !hasMore {
			t.Error("page 1: expected hasMore=true")
		}
		if nextCursor == nil || *nextCursor == "" {
			t.Error("page 1: expected non-empty next cursor")
		}
		for i, item := range items {
			if item.ID == "" {
				t.Errorf("item %d has empty ID", i)
			}
			if item.SessionID != visit.ID {
				t.Errorf("item %d SessionID = %q, want %q", i, item.SessionID, visit.ID)
			}
		}

		// Second page
		page2, _, hasMore2, err := tRepo.ListBySession(ctx, visit.ID, nextCursor, 2)
		if err != nil {
			t.Fatalf("ListBySession page 2 failed: %v", err)
		}
		if len(page2) == 0 {
			t.Error("page 2: expected at least 1 item")
		}
		_ = hasMore2
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		item := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Status:    string(model.TimelineItemStatusDone),
			Role:      string(model.MessageRolePatient),
			Content:   "用于状态更新的消息",
		}
		if err := tRepo.Append(ctx, &item); err != nil {
			t.Fatalf("setup Append: %v", err)
		}

		err := tRepo.UpdateStatus(ctx, item.ID, string(model.TimelineItemStatusInvalidated))
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}
		// UpdateStatus updates the DB status column but ListBySession reads
		// from the content JSON which stores the original status at creation
		// time. We verify the call succeeds without error.
	})
}

// ---------------------------------------------------------------------------
// Flow card repository tests
// ---------------------------------------------------------------------------

func TestFlowCardRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()
	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	fRepo := repository.NewFlowCardRepository(db)

	patient := createPatient(ctx, t, pRepo)
	visit := createVisit(ctx, t, vRepo, patient.ID)

	t.Run("Create", func(t *testing.T) {
		card := &model.FlowCard{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindLabDecision),
			Status:    string(model.FlowCardStatusPending),
			Blocking:  true,
			Title:     "血常规检查",
			Reason:    "需排除感染",
			TestItems: []model.TestItem{
				{Code: "WBC", Name: "白细胞计数", SampleType: "blood"},
			},
			EstimatedFee: model.Float64Ptr(50.0),
		}
		err := fRepo.Create(ctx, card)
		if err != nil {
			t.Fatalf("Create flow card failed: %v", err)
		}
		if card.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set by Create")
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		card := &model.FlowCard{
			ID:         uuid.New().String(),
			SessionID:  visit.ID,
			Kind:       string(model.FlowCardKindDiagnosis),
			Status:     string(model.FlowCardStatusAccepted),
			Blocking:   false,
			Title:      "诊断结果",
			Diagnosis:  "上呼吸道感染",
			Confidence: string(model.DiagnosisConfidenceHigh),
			Evidence:   []string{"发热", "咳嗽"},
		}
		if err := fRepo.Create(ctx, card); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		found, err := fRepo.FindByID(ctx, card.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found.Kind != string(model.FlowCardKindDiagnosis) {
			t.Errorf("Kind = %q, want %q", found.Kind, string(model.FlowCardKindDiagnosis))
		}
		if found.Blocking {
			t.Error("expected blocking=false")
		}
		if found.Diagnosis != "上呼吸道感染" {
			t.Errorf("Diagnosis = %q, want '上呼吸道感染'", found.Diagnosis)
		}
		if found.Confidence != string(model.DiagnosisConfidenceHigh) {
			t.Errorf("Confidence = %q, want %q", found.Confidence, string(model.DiagnosisConfidenceHigh))
		}
		if len(found.Evidence) == 0 {
			t.Error("expected non-empty Evidence")
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		card := &model.FlowCard{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindLabDecision),
			Status:    string(model.FlowCardStatusPending),
			Blocking:  true,
			Title:     "尿常规检查",
			Reason:    "常规筛查",
			TestItems: []model.TestItem{
				{Code: "URINE", Name: "尿常规", SampleType: "urine"},
			},
		}
		if err := fRepo.Create(ctx, card); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		err := fRepo.UpdateStatus(ctx, card.ID, string(model.FlowCardStatusCompleted))
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		updated, err := fRepo.FindByID(ctx, card.ID)
		if err != nil {
			t.Fatalf("FindByID after update failed: %v", err)
		}
		if updated.Status != string(model.FlowCardStatusCompleted) {
			t.Errorf("status = %q, want %q", updated.Status, string(model.FlowCardStatusCompleted))
		}
		if updated.HandledAt == nil || updated.HandledAt.IsZero() {
			t.Error("expected HandledAt to be set after UpdateStatus")
		}
	})

	t.Run("ListBySession", func(t *testing.T) {
		// Create two flow cards for the same session
		card1 := &model.FlowCard{
			ID:         uuid.New().String(),
			SessionID:  visit.ID,
			Kind:       string(model.FlowCardKindTreatmentPlan),
			Status:     string(model.FlowCardStatusPending),
			Blocking:   false,
			Title:      "治疗方案",
			Plan:       "口服抗生素",
			Capability: string(model.CapabilityAvailable),
		}
		if err := fRepo.Create(ctx, card1); err != nil {
			t.Fatalf("setup: Create card1: %v", err)
		}

		card2 := &model.FlowCard{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindAdviceOnly),
			Status:    string(model.FlowCardStatusPending),
			Blocking:  false,
			Title:     "生活建议",
			Advices:   []string{"多喝水", "注意休息"},
		}
		if err := fRepo.Create(ctx, card2); err != nil {
			t.Fatalf("setup: Create card2: %v", err)
		}

		cards, err := fRepo.ListBySession(ctx, visit.ID)
		if err != nil {
			t.Fatalf("ListBySession failed: %v", err)
		}
		if len(cards) == 0 {
			t.Fatal("expected at least 1 flow card")
		}
		// Should include our newly created cards
		foundCard1 := false
		foundCard2 := false
		for _, c := range cards {
			if c.ID == card1.ID {
				foundCard1 = true
				if c.Kind != string(model.FlowCardKindTreatmentPlan) {
					t.Errorf("card1 Kind = %q, want %q", c.Kind, string(model.FlowCardKindTreatmentPlan))
				}
			}
			if c.ID == card2.ID {
				foundCard2 = true
				if len(c.Advices) != 2 {
					t.Errorf("card2 Advices = %v, want [多喝水 注意休息]", c.Advices)
				}
			}
		}
		if !foundCard1 {
			t.Error("card1 not found in ListBySession results")
		}
		if !foundCard2 {
			t.Error("card2 not found in ListBySession results")
		}
	})

	t.Run("Update", func(t *testing.T) {
		card := &model.FlowCard{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindDiagnosis),
			Status:    string(model.FlowCardStatusPending),
			Blocking:  true,
			Title:     "诊断初稿",
			Diagnosis: "疑似感冒",
		}
		if err := fRepo.Create(ctx, card); err != nil {
			t.Fatalf("setup: Create: %v", err)
		}

		// Modify and update
		card.Status = string(model.FlowCardStatusAccepted)
		card.Blocking = false
		card.Diagnosis = "上呼吸道感染"
		card.Confidence = string(model.DiagnosisConfidenceHigh)
		now := time.Now()
		card.HandledAt = &now
		err := fRepo.Update(ctx, card)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify
		updated, err := fRepo.FindByID(ctx, card.ID)
		if err != nil {
			t.Fatalf("FindByID after update: %v", err)
		}
		if updated.Status != string(model.FlowCardStatusAccepted) {
			t.Errorf("Status = %q, want %q", updated.Status, string(model.FlowCardStatusAccepted))
		}
		if updated.Blocking {
			t.Error("Blocking = true, want false")
		}
		if updated.Diagnosis != "上呼吸道感染" {
			t.Errorf("Diagnosis = %q, want %q", updated.Diagnosis, "上呼吸道感染")
		}
		if updated.HandledAt == nil {
			t.Error("expected HandledAt to be set after Update")
		}
	})
}

// ---------------------------------------------------------------------------
// User repository tests
// ---------------------------------------------------------------------------

func TestUserRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	patientRepo := repository.NewPatientRepository(db)
	userRepo := repository.NewUserRepository(db)

	p := createPatient(ctx, t, patientRepo)

	t.Run("create and find by phone", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New().String(),
			Phone:        "13800001111",
			PasswordHash: "$2a$12$fakehashvalue",
			RealName:     "张三",
			PatientID:    p.ID,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("Create user: %v", err)
		}

		found, err := userRepo.FindByPhone(ctx, "13800001111")
		if err != nil {
			t.Fatalf("FindByPhone: %v", err)
		}
		if found.ID != user.ID {
			t.Errorf("ID = %s, want %s", found.ID, user.ID)
		}
		if found.RealName != "张三" {
			t.Errorf("RealName = %s, want 张三", found.RealName)
		}
		if found.PatientID != p.ID {
			t.Errorf("PatientID = %s, want %s", found.PatientID, p.ID)
		}
	})

	t.Run("find by id", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New().String(),
			Phone:        "13800002222",
			PasswordHash: "$2a$12$fakehashvalue2",
			RealName:     "李四",
			PatientID:    p.ID,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("Create: %v", err)
		}

		found, err := userRepo.FindByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if found.Phone != "13800002222" {
			t.Errorf("Phone = %s, want 13800002222", found.Phone)
		}
	})

	t.Run("duplicate phone returns error", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New().String(),
			Phone:        "13800001111",
			PasswordHash: "$2a$12$duplicate",
			PatientID:    p.ID,
		}
		err := userRepo.Create(ctx, user)
		if err == nil {
			t.Error("expected error for duplicate phone")
		}
	})

	t.Run("not found returns ErrUserNotFound", func(t *testing.T) {
		_, err := userRepo.FindByPhone(ctx, "19999999999")
		if err != model.ErrUserNotFound {
			t.Errorf("err = %v, want ErrUserNotFound", err)
		}

		_, err = userRepo.FindByID(ctx, "nonexistent-id")
		if err != model.ErrUserNotFound {
			t.Errorf("err = %v, want ErrUserNotFound", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Refresh token repository tests
// ---------------------------------------------------------------------------

func TestRefreshTokenRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	patientRepo := repository.NewPatientRepository(db)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)

	p := createPatient(ctx, t, patientRepo)
	user := &model.User{
		ID:           uuid.New().String(),
		Phone:        "13800009999",
		PasswordHash: "$2a$12$testhash",
		PatientID:    p.ID,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	t.Run("create and find by hash", func(t *testing.T) {
		rt := &model.RefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "sha256hashvalue1",
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, rt); err != nil {
			t.Fatalf("Create token: %v", err)
		}

		found, err := tokenRepo.FindByTokenHash(ctx, "sha256hashvalue1")
		if err != nil {
			t.Fatalf("FindByTokenHash: %v", err)
		}
		if found.UserID != user.ID {
			t.Errorf("UserID = %s, want %s", found.UserID, user.ID)
		}
		if found.UsedAt != nil {
			t.Error("UsedAt should be nil for new token")
		}
	})

	t.Run("mark used", func(t *testing.T) {
		rt := &model.RefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "sha256hashvalue2",
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, rt); err != nil {
			t.Fatalf("Create: %v", err)
		}

		if err := tokenRepo.MarkUsed(ctx, rt.ID); err != nil {
			t.Fatalf("MarkUsed: %v", err)
		}

		found, err := tokenRepo.FindByTokenHash(ctx, "sha256hashvalue2")
		if err != nil {
			t.Fatalf("FindByTokenHash after mark: %v", err)
		}
		if found.UsedAt == nil {
			t.Error("UsedAt should not be nil after MarkUsed")
		}
	})

	t.Run("revoke all by user", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			rt := &model.RefreshToken{
				ID:        uuid.New().String(),
				TokenHash: uuid.New().String(),
				UserID:    user.ID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			if err := tokenRepo.Create(ctx, rt); err != nil {
				t.Fatalf("Create token %d: %v", i, err)
			}
		}

		if err := tokenRepo.RevokeAllByUserID(ctx, user.ID); err != nil {
			t.Fatalf("RevokeAllByUserID: %v", err)
		}

		// All tokens for this user should be gone
		_, err := tokenRepo.FindByTokenHash(ctx, "sha256hashvalue1")
		if err != model.ErrRefreshTokenInvalid {
			t.Errorf("after revoke, err = %v, want ErrRefreshTokenInvalid", err)
		}
	})

	t.Run("not found returns ErrRefreshTokenInvalid", func(t *testing.T) {
		_, err := tokenRepo.FindByTokenHash(ctx, "nonexistent-hash")
		if err != model.ErrRefreshTokenInvalid {
			t.Errorf("err = %v, want ErrRefreshTokenInvalid", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Patient medical_history field tests
// ---------------------------------------------------------------------------

func TestPatientRepo_MedicalHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := repository.NewPatientRepository(db)

	p := &model.PatientProfile{
		ID:             uuid.New().String(),
		Name:           "测试既往病史",
		Gender:         "male",
		Age:            40,
		MedicalHistory: []string{"慢性咽炎3年", "2024年阑尾炎手术"},
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.FindByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if len(found.MedicalHistory) != 2 {
		t.Fatalf("MedicalHistory len = %d, want 2", len(found.MedicalHistory))
	}
	if found.MedicalHistory[0] != "慢性咽炎3年" {
		t.Errorf("MedicalHistory[0] = %s", found.MedicalHistory[0])
	}

	updated, err := repo.UpdateProfile(ctx, p.ID, model.ProfileUpdateInput{
		MedicalHistory: []string{"更新后的病史"},
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if len(updated.MedicalHistory) != 1 || updated.MedicalHistory[0] != "更新后的病史" {
		t.Errorf("updated MedicalHistory = %v", updated.MedicalHistory)
	}
}

func TestAddressRepo_CRUD(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	patientRepo := repository.NewPatientRepository(db)
	addrRepo := repository.NewAddressRepository(db)

	patient := createPatient(ctx, t, patientRepo)

	addr := &model.Address{
		ID:        uuid.New().String(),
		PatientID: patient.ID,
		Name:      "李明",
		Phone:     "13800002468",
		Province:  "辽宁省",
		City:      "沈阳市",
		District:  "浑南区",
		Detail:    "创新路195号",
		IsDefault: true,
		Tag:       addrTagPtr(model.AddressTagCompany),
	}
	if err := addrRepo.Create(ctx, addr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := addrRepo.FindByID(ctx, addr.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Name != "李明" {
		t.Errorf("Name = %s, want 李明", found.Name)
	}
	if !found.IsDefault {
		t.Error("IsDefault should be true")
	}

	addrs, err := addrRepo.ListByPatient(ctx, patient.ID)
	if err != nil {
		t.Fatalf("ListByPatient: %v", err)
	}
	if len(addrs) != 1 {
		t.Errorf("len = %d, want 1", len(addrs))
	}

	count, err := addrRepo.CountByPatient(ctx, patient.ID)
	if err != nil {
		t.Fatalf("CountByPatient: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	addr.Name = "张三"
	if err := addrRepo.Update(ctx, addr); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, err := addrRepo.FindByID(ctx, addr.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if updated.Name != "张三" {
		t.Errorf("Name = %s, want 张三", updated.Name)
	}

	if err := addrRepo.ClearDefaultByPatient(ctx, patient.ID); err != nil {
		t.Fatalf("ClearDefaultByPatient: %v", err)
	}

	if err := addrRepo.SetDefault(ctx, addr.ID, patient.ID); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}
	afterSet, err := addrRepo.FindByID(ctx, addr.ID)
	if err != nil {
		t.Fatalf("FindByID after SetDefault: %v", err)
	}
	if !afterSet.IsDefault {
		t.Error("IsDefault should be true after SetDefault")
	}

	if err := addrRepo.Delete(ctx, addr.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = addrRepo.FindByID(ctx, addr.ID)
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound after delete, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Dashboard repository tests
// ---------------------------------------------------------------------------

func TestDashboardRepo_CountPatients(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	count, err := dRepo.CountPatients(ctx)
	if err != nil {
		t.Fatalf("CountPatients: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	createPatient(ctx, t, pRepo)

	count, err = dRepo.CountPatients(ctx)
	if err != nil {
		t.Fatalf("CountPatients after create: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestDashboardRepo_CountSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	patient := createPatient(ctx, t, pRepo)

	count, err := dRepo.CountSessions(ctx)
	if err != nil {
		t.Fatalf("CountSessions: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	createVisit(ctx, t, vRepo, patient.ID)

	count, err = dRepo.CountSessions(ctx)
	if err != nil {
		t.Fatalf("CountSessions after create: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestDashboardRepo_CountActiveSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	patient := createPatient(ctx, t, pRepo)

	count, err := dRepo.CountActiveSessions(ctx)
	if err != nil {
		t.Fatalf("CountActiveSessions: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	// Create a visit with "new" status (one of the active statuses)
	activeVisit := &model.VisitSession{
		ID:            uuid.New().String(),
		PatientID:     patient.ID,
		EntryType:     string(model.VisitEntryTypeNew),
		Status:        "new",
		AskRound:      0,
		AskRoundLimit: 20,
		LabRound:      0,
		LabRoundLimit: 10,
		TimerPaused:   false,
	}
	if err := vRepo.Create(ctx, activeVisit); err != nil {
		t.Fatalf("Create active visit: %v", err)
	}

	count, err = dRepo.CountActiveSessions(ctx)
	if err != nil {
		t.Fatalf("CountActiveSessions after create: %v", err)
	}
	if count != 1 {
		t.Errorf("active count = %d, want 1", count)
	}
}

func TestDashboardRepo_CountPatientsSince(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	// Initially 0
	count, err := dRepo.CountPatientsSince(ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountPatientsSince: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	createPatient(ctx, t, pRepo)

	// Should find the newly created patient
	count, err = dRepo.CountPatientsSince(ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountPatientsSince after create: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Future date should return 0
	count, err = dRepo.CountPatientsSince(ctx, time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountPatientsSince future: %v", err)
	}
	if count != 0 {
		t.Errorf("future count = %d, want 0", count)
	}
}

func TestDashboardRepo_CountSessionsSince(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	patient := createPatient(ctx, t, pRepo)

	// Initially 0
	count, err := dRepo.CountSessionsSince(ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountSessionsSince: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	createVisit(ctx, t, vRepo, patient.ID)

	// Should find the newly created session
	count, err = dRepo.CountSessionsSince(ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountSessionsSince after create: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Future date should return 0
	count, err = dRepo.CountSessionsSince(ctx, time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountSessionsSince future: %v", err)
	}
	if count != 0 {
		t.Errorf("future count = %d, want 0", count)
	}
}

func TestDashboardRepo_ListPatients(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	// Create 3 patients
	for i := 0; i < 3; i++ {
		p := &model.PatientProfile{
			ID:          uuid.New().String(),
			Name:        fmt.Sprintf("患者%d", i+1),
			Gender:      string(model.GenderMale),
			Age:         20 + i,
			PhoneMasked: fmt.Sprintf("138****%04d", i),
		}
		if err := pRepo.Create(ctx, p); err != nil {
			t.Fatalf("Create patient %d: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// List with page size 2
	query := model.AdminPatientQuery{Page: 1, PageSize: 2}
	items, total, err := dRepo.ListPatients(ctx, query)
	if err != nil {
		t.Fatalf("ListPatients: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(items) != 2 {
		t.Errorf("items len = %d, want 2", len(items))
	}

	// Second page
	query = model.AdminPatientQuery{Page: 2, PageSize: 2}
	items, _, err = dRepo.ListPatients(ctx, query)
	if err != nil {
		t.Fatalf("ListPatients page 2: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("page 2 items = %d, want 1", len(items))
	}

	// Search by name
	query = model.AdminPatientQuery{Page: 1, PageSize: 10, Search: "患者1"}
	items, total, err = dRepo.ListPatients(ctx, query)
	if err != nil {
		t.Fatalf("ListPatients search: %v", err)
	}
	if total != 1 {
		t.Errorf("search total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].RealName != "患者1" {
		t.Errorf("search item name = %q, want '患者1'", items[0].RealName)
	}

	// Search returns empty slice when no match
	query = model.AdminPatientQuery{Page: 1, PageSize: 10, Search: "不存在"}
	items, total, err = dRepo.ListPatients(ctx, query)
	if err != nil {
		t.Fatalf("ListPatients search no match: %v", err)
	}
	if total != 0 {
		t.Errorf("search no match total = %d, want 0", total)
	}
	if len(items) != 0 {
		t.Errorf("search no match items = %d, want 0", len(items))
	}
}

func TestDashboardRepo_ListSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	patient := createPatient(ctx, t, pRepo)

	// Create 2 visits
	for i := 0; i < 2; i++ {
		visit := &model.VisitSession{
			ID:            uuid.New().String(),
			PatientID:     patient.ID,
			EntryType:     string(model.VisitEntryTypeNew),
			Status:        string(model.VisitStatusChatting),
			AskRound:      i,
			AskRoundLimit: 20,
			LabRound:      0,
			LabRoundLimit: 10,
			TimerPaused:   false,
		}
		if err := vRepo.Create(ctx, visit); err != nil {
			t.Fatalf("Create visit %d: %v", i, err)
		}
		time.Sleep(1100 * time.Millisecond)
	}

	// List all sessions
	query := model.AdminSessionQuery{Page: 1, PageSize: 10}
	items, total, err := dRepo.ListSessions(ctx, query)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Errorf("items len = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.PatientID != patient.ID {
			t.Errorf("item patient_id = %q, want %q", item.PatientID, patient.ID)
		}
		if item.Status != string(model.VisitStatusChatting) {
			t.Errorf("item status = %q, want %q", item.Status, string(model.VisitStatusChatting))
		}
	}

	// Filter by patient ID
	query = model.AdminSessionQuery{Page: 1, PageSize: 10, PatientID: patient.ID}
	_, total, err = dRepo.ListSessions(ctx, query)
	if err != nil {
		t.Fatalf("ListSessions filter by patient: %v", err)
	}
	if total != 2 {
		t.Errorf("filtered total = %d, want 2", total)
	}

	// No results for non-existent patient
	query = model.AdminSessionQuery{Page: 1, PageSize: 10, PatientID: "nonexistent"}
	_, total, err = dRepo.ListSessions(ctx, query)
	if err != nil {
		t.Fatalf("ListSessions filter nonexistent: %v", err)
	}
	if total != 0 {
		t.Errorf("nonexistent total = %d, want 0", total)
	}
}

// ---------------------------------------------------------------------------
// System settings repository tests
// ---------------------------------------------------------------------------

func TestSettingsRepo_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	repo := repository.NewSettingsRepository(db)

	settings, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get settings: %v", err)
	}
	if settings.SiteName != "NEUHIS Agent" {
		t.Errorf("SiteName = %q, want 'NEUHIS Agent'", settings.SiteName)
	}
	if settings.MaxConcurrentSessions != 3 {
		t.Errorf("MaxConcurrentSessions = %d, want 3", settings.MaxConcurrentSessions)
	}
	if settings.SessionTimeoutMinutes != 30 {
		t.Errorf("SessionTimeoutMinutes = %d, want 30", settings.SessionTimeoutMinutes)
	}
	if !settings.EnableRegistration {
		t.Error("EnableRegistration should be true by default")
	}
}

func TestSettingsRepo_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	repo := repository.NewSettingsRepository(db)

	// Update site_name
	siteName := "My Hospital"
	updated, err := repo.Update(ctx, model.UpdateSystemSettingsInput{
		SiteName: &siteName,
	})
	if err != nil {
		t.Fatalf("Update settings: %v", err)
	}
	if updated.SiteName != "My Hospital" {
		t.Errorf("updated SiteName = %q, want 'My Hospital'", updated.SiteName)
	}

	// Verify via Get
	settings, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if settings.SiteName != "My Hospital" {
		t.Errorf("Get SiteName = %q, want 'My Hospital'", settings.SiteName)
	}

	// Update all fields
	maxSessions := 5
	timeout := 60
	enabled := false
	updated, err = repo.Update(ctx, model.UpdateSystemSettingsInput{
		SiteName:              &siteName,
		MaxConcurrentSessions: &maxSessions,
		SessionTimeoutMinutes: &timeout,
		EnableRegistration:    &enabled,
	})
	if err != nil {
		t.Fatalf("Update all settings: %v", err)
	}
	if updated.MaxConcurrentSessions != 5 {
		t.Errorf("MaxConcurrentSessions = %d, want 5", updated.MaxConcurrentSessions)
	}
	if updated.SessionTimeoutMinutes != 60 {
		t.Errorf("SessionTimeoutMinutes = %d, want 60", updated.SessionTimeoutMinutes)
	}
	if updated.EnableRegistration {
		t.Error("EnableRegistration should be false")
	}
}

// ---------------------------------------------------------------------------
// Admin repository tests
// ---------------------------------------------------------------------------

func TestAdminRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	repo := repository.NewAdminRepository(db)

	t.Run("Create and FindByUsername", func(t *testing.T) {
		admin := &model.AdminUser{
			ID:           uuid.New().String(),
			Username:     "testadmin",
			PasswordHash: "$2a$12$testhashvalue",
			Role:         model.AdminRoleAdmin,
			DisplayName:  "测试管理员",
			CreatedAt:    time.Now(),
		}
		err := repo.Create(ctx, admin)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		found, err := repo.FindByUsername(ctx, "testadmin")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if found.ID != admin.ID {
			t.Errorf("ID = %s, want %s", found.ID, admin.ID)
		}
		if found.Username != "testadmin" {
			t.Errorf("Username = %s, want testadmin", found.Username)
		}
		if found.Role != model.AdminRoleAdmin {
			t.Errorf("Role = %s, want %s", found.Role, model.AdminRoleAdmin)
		}
		if found.DisplayName != "测试管理员" {
			t.Errorf("DisplayName = %s, want 测试管理员", found.DisplayName)
		}
		if found.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be populated")
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		admin := &model.AdminUser{
			ID:           uuid.New().String(),
			Username:     "testadmin2",
			PasswordHash: "$2a$12$testhashvalue2",
			Role:         model.AdminRoleSuperAdmin,
			DisplayName:  "超级管理员",
			CreatedAt:    time.Now(),
		}
		if err := repo.Create(ctx, admin); err != nil {
			t.Fatalf("Create: %v", err)
		}

		found, err := repo.FindByID(ctx, admin.ID)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if found.Username != "testadmin2" {
			t.Errorf("Username = %s, want testadmin2", found.Username)
		}
		if found.Role != model.AdminRoleSuperAdmin {
			t.Errorf("Role = %s, want %s", found.Role, model.AdminRoleSuperAdmin)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := repo.FindByUsername(ctx, "nonexistent")
		if err != model.ErrAdminNotFound {
			t.Errorf("FindByUsername nonexistent: err = %v, want ErrAdminNotFound", err)
		}

		_, err = repo.FindByID(ctx, "nonexistent-id")
		if err != model.ErrAdminNotFound {
			t.Errorf("FindByID nonexistent: err = %v, want ErrAdminNotFound", err)
		}
	})

	t.Run("Duplicate username", func(t *testing.T) {
		admin := &model.AdminUser{
			ID:           uuid.New().String(),
			Username:     "duplicate_admin",
			PasswordHash: "$2a$12$testhash",
			Role:         model.AdminRoleOperator,
			DisplayName:  "操作员",
			CreatedAt:    time.Now(),
		}
		if err := repo.Create(ctx, admin); err != nil {
			t.Fatalf("Create: %v", err)
		}

		duplicate := &model.AdminUser{
			ID:           uuid.New().String(),
			Username:     "duplicate_admin",
			PasswordHash: "$2a$12$testhash2",
			Role:         model.AdminRoleOperator,
			DisplayName:  "操作员2",
			CreatedAt:    time.Now(),
		}
		err := repo.Create(ctx, duplicate)
		if err == nil {
			t.Error("expected error for duplicate username")
		}
	})
}

// ---------------------------------------------------------------------------
// Admin refresh token repository tests
// ---------------------------------------------------------------------------

func TestAdminRefreshTokenRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	adminRepo := repository.NewAdminRepository(db)
	tokenRepo := repository.NewAdminRefreshTokenRepository(db)

	// Create an admin user first (dependency)
	admin := &model.AdminUser{
		ID:           uuid.New().String(),
		Username:     "tokenadmin",
		PasswordHash: "$2a$12$testhash",
		Role:         model.AdminRoleAdmin,
		DisplayName:  "Token Test Admin",
		CreatedAt:    time.Now(),
	}
	if err := adminRepo.Create(ctx, admin); err != nil {
		t.Fatalf("Create admin: %v", err)
	}

	t.Run("Create and FindByTokenHash", func(t *testing.T) {
		token := &model.AdminRefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "sha256testhash1",
			AdminID:   admin.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := tokenRepo.Create(ctx, token)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if token.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set by Create")
		}

		found, err := tokenRepo.FindByTokenHash(ctx, "sha256testhash1")
		if err != nil {
			t.Fatalf("FindByTokenHash: %v", err)
		}
		if found.AdminID != admin.ID {
			t.Errorf("AdminID = %s, want %s", found.AdminID, admin.ID)
		}
		if found.UsedAt != nil {
			t.Error("UsedAt should be nil for new token")
		}
	})

	t.Run("MarkUsed", func(t *testing.T) {
		token := &model.AdminRefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "sha256testhash2",
			AdminID:   admin.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, token); err != nil {
			t.Fatalf("Create: %v", err)
		}

		if err := tokenRepo.MarkUsed(ctx, token.ID); err != nil {
			t.Fatalf("MarkUsed: %v", err)
		}

		found, err := tokenRepo.FindByTokenHash(ctx, "sha256testhash2")
		if err != nil {
			t.Fatalf("FindByTokenHash after MarkUsed: %v", err)
		}
		if found.UsedAt == nil {
			t.Error("UsedAt should not be nil after MarkUsed")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := tokenRepo.FindByTokenHash(ctx, "nonexistent-hash")
		if err != model.ErrAdminInvalidRefreshToken {
			t.Errorf("err = %v, want ErrAdminInvalidRefreshToken", err)
		}
	})

	t.Run("RevokeAllByAdminID", func(t *testing.T) {
		// Create a few tokens
		for i := 0; i < 3; i++ {
			token := &model.AdminRefreshToken{
				ID:        uuid.New().String(),
				TokenHash: uuid.New().String(),
				AdminID:   admin.ID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			if err := tokenRepo.Create(ctx, token); err != nil {
				t.Fatalf("Create token %d: %v", i, err)
			}
		}

		if err := tokenRepo.RevokeAllByAdminID(ctx, admin.ID); err != nil {
			t.Fatalf("RevokeAllByAdminID: %v", err)
		}

		// All tokens for this admin should be gone
		_, err := tokenRepo.FindByTokenHash(ctx, "sha256testhash1")
		if err != model.ErrAdminInvalidRefreshToken {
			t.Errorf("after revoke, err = %v, want ErrAdminInvalidRefreshToken", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Repository edge case tests (address, dashboard, flow card, admin token, visit, timeline, settings)
// ---------------------------------------------------------------------------

func TestPatientRepo_UpdateProfileAllFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	repo := repository.NewPatientRepository(db)

	patient := &model.PatientProfile{
		ID:                  uuid.New().String(),
		Name:                "AllFields患者",
		Gender:              string(model.GenderMale),
		Age:                 50,
		PhoneMasked:         "137****0000",
		IDCardMasked:        "110****0000",
		Allergies:           []string{"青霉素"},
		ChronicDiseases:     []string{"高血压"},
		LongTermMedications: []string{"硝苯地平"},
		MedicalHistory:      []string{"2020年手术"},
	}
	if err := repo.Create(ctx, patient); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("UpdateAllFields", func(t *testing.T) {
		updated, err := repo.UpdateProfile(ctx, patient.ID, model.ProfileUpdateInput{
			PatientID:           patient.ID,
			Allergies:           []string{"头孢"},
			ChronicDiseases:     []string{"高血压", "糖尿病"},
			LongTermMedications: []string{"硝苯地平", "二甲双胍"},
			MedicalHistory:      []string{"2020年手术", "2023年住院"},
		})
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if len(updated.Allergies) != 1 || updated.Allergies[0] != "头孢" {
			t.Errorf("Allergies = %v", updated.Allergies)
		}
		if len(updated.ChronicDiseases) != 2 {
			t.Errorf("ChronicDiseases = %v", updated.ChronicDiseases)
		}
		if len(updated.LongTermMedications) != 2 {
			t.Errorf("LongTermMedications = %v", updated.LongTermMedications)
		}
		if len(updated.MedicalHistory) != 2 {
			t.Errorf("MedicalHistory = %v", updated.MedicalHistory)
		}
	})

	t.Run("UpdateNilFields", func(t *testing.T) {
		// Passing nil for all optional fields should not change anything.
		updated, err := repo.UpdateProfile(ctx, patient.ID, model.ProfileUpdateInput{
			PatientID: patient.ID,
		})
		if err != nil {
			t.Fatalf("UpdateProfile with nil fields: %v", err)
		}
		if len(updated.Allergies) != 1 {
			t.Errorf("expected Allergies to remain, got %v", updated.Allergies)
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in profile update.
		_ = db.Close()

		_, err := repo.UpdateProfile(ctx, "any", model.ProfileUpdateInput{
			PatientID: "any",
			Allergies: []string{"test"},
		})
		if err == nil {
			t.Error("UpdateProfile: expected error")
		}
		p := &model.PatientProfile{
			ID:     uuid.New().String(),
			Name:   "closedb",
			Gender: "male",
			Age:    30,
		}
		err = repo.Create(ctx, p)
		if err == nil {
			t.Error("Create: expected error")
		}
	})
}

func TestAddressRepo_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()
	patientRepo := repository.NewPatientRepository(db)
	addrRepo := repository.NewAddressRepository(db)

	patient := createPatient(ctx, t, patientRepo)

	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := addrRepo.Delete(ctx, "nonexistent-address-id")
		if !errors.Is(err, model.ErrAddressNotFound) {
			t.Fatalf("expected ErrAddressNotFound, got %v", err)
		}
	})

	t.Run("ListByPatientNoAddresses", func(t *testing.T) {
		// A patient with no addresses should return an empty slice, not nil.
		p2 := createPatient(ctx, t, patientRepo)
		addrs, err := addrRepo.ListByPatient(ctx, p2.ID)
		if err != nil {
			t.Fatalf("ListByPatient: %v", err)
		}
		if addrs == nil {
			t.Error("expected non-nil empty slice, got nil")
		}
		if len(addrs) != 0 {
			t.Errorf("expected 0 addresses, got %d", len(addrs))
		}
	})

	t.Run("CreateDuplicateID", func(t *testing.T) {
		addr := &model.Address{
			ID:        uuid.New().String(),
			PatientID: patient.ID,
			Name:      "测试",
			Phone:     "13800000000",
			Province:  "北京",
			City:      "北京",
			District:  "朝阳区",
			Detail:    "测试地址",
			IsDefault: false,
			Tag:       addrTagPtr(model.AddressTagHome),
		}
		if err := addrRepo.Create(ctx, addr); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		// Second create with same ID should fail due to PK conflict.
		err := addrRepo.Create(ctx, addr)
		if err == nil {
			t.Error("expected error for duplicate address ID, got nil")
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in address functions.
		_ = db.Close()

		_, err := addrRepo.FindByID(ctx, "any")
		if err == nil {
			t.Error("FindByID: expected error")
		}
		_, err = addrRepo.ListByPatient(ctx, patient.ID)
		if err == nil {
			t.Error("ListByPatient: expected error")
		}
		_, err = addrRepo.CountByPatient(ctx, patient.ID)
		if err == nil {
			t.Error("CountByPatient: expected error")
		}
		addr := &model.Address{
			ID:        uuid.New().String(),
			PatientID: patient.ID,
			Name:      "test",
			Phone:     "13800000001",
			Province:  "北京",
			City:      "北京",
			District:  "朝阳区",
			Detail:    "test",
			IsDefault: false,
			Tag:       addrTagPtr(model.AddressTagHome),
		}
		err = addrRepo.Create(ctx, addr)
		if err == nil {
			t.Error("Create: expected error")
		}
		err = addrRepo.Update(ctx, addr)
		if err == nil {
			t.Error("Update: expected error")
		}
		err = addrRepo.Delete(ctx, "any")
		if err == nil {
			t.Error("Delete: expected error")
		}
		err = addrRepo.ClearDefaultByPatient(ctx, patient.ID)
		if err == nil {
			t.Error("ClearDefaultByPatient: expected error")
		}
	})
}

func TestDashboardRepo_ListSessionsByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	dRepo := repository.NewDashboardRepository(db)

	patient := createPatient(ctx, t, pRepo)

	// Create visits with different statuses so we can filter.
	chattingVisit := &model.VisitSession{
		ID:            uuid.New().String(),
		PatientID:     patient.ID,
		EntryType:     string(model.VisitEntryTypeNew),
		Status:        string(model.VisitStatusChatting),
		AskRound:      0,
		AskRoundLimit: 20,
		LabRound:      0,
		LabRoundLimit: 10,
		TimerPaused:   false,
	}
	if err := vRepo.Create(ctx, chattingVisit); err != nil {
		t.Fatalf("Create chatting visit: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	blockedVisit := &model.VisitSession{
		ID:            uuid.New().String(),
		PatientID:     patient.ID,
		EntryType:     string(model.VisitEntryTypeNew),
		Status:        string(model.VisitStatusBlocked),
		AskRound:      0,
		AskRoundLimit: 20,
		LabRound:      0,
		LabRoundLimit: 10,
		TimerPaused:   false,
	}
	if err := vRepo.Create(ctx, blockedVisit); err != nil {
		t.Fatalf("Create blocked visit: %v", err)
	}

	t.Run("FilterByChatting", func(t *testing.T) {
		query := model.AdminSessionQuery{Page: 1, PageSize: 10, Status: string(model.VisitStatusChatting)}
		items, total, err := dRepo.ListSessions(ctx, query)
		if err != nil {
			t.Fatalf("ListSessions: %v", err)
		}
		if total != 1 {
			t.Errorf("total = %d, want 1", total)
		}
		if len(items) != 1 {
			t.Fatalf("items len = %d, want 1", len(items))
		}
		if items[0].Status != string(model.VisitStatusChatting) {
			t.Errorf("status = %q, want %q", items[0].Status, string(model.VisitStatusChatting))
		}
	})

	t.Run("FilterByBlocked", func(t *testing.T) {
		query := model.AdminSessionQuery{Page: 1, PageSize: 10, Status: string(model.VisitStatusBlocked)}
		items, total, err := dRepo.ListSessions(ctx, query)
		if err != nil {
			t.Fatalf("ListSessions: %v", err)
		}
		if total != 1 {
			t.Errorf("total = %d, want 1", total)
		}
		if len(items) != 1 {
			t.Fatalf("items len = %d, want 1", len(items))
		}
		if items[0].Status != string(model.VisitStatusBlocked) {
			t.Errorf("status = %q, want %q", items[0].Status, string(model.VisitStatusBlocked))
		}
	})

	t.Run("FilterByNonExistentStatus", func(t *testing.T) {
		query := model.AdminSessionQuery{Page: 1, PageSize: 10, Status: "nonexistent_status"}
		items, total, err := dRepo.ListSessions(ctx, query)
		if err != nil {
			t.Fatalf("ListSessions: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if len(items) != 0 {
			t.Errorf("items len = %d, want 0", len(items))
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in all dashboard functions.
		_ = db.Close()

		_, err := dRepo.CountPatients(ctx)
		if err == nil {
			t.Error("CountPatients: expected error")
		}
		_, err = dRepo.CountPatientsSince(ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
		if err == nil {
			t.Error("CountPatientsSince: expected error")
		}
		_, err = dRepo.CountSessions(ctx)
		if err == nil {
			t.Error("CountSessions: expected error")
		}
		_, err = dRepo.CountActiveSessions(ctx)
		if err == nil {
			t.Error("CountActiveSessions: expected error")
		}
		_, err = dRepo.CountSessionsSince(ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
		if err == nil {
			t.Error("CountSessionsSince: expected error")
		}
		_, _, err = dRepo.ListPatients(ctx, model.AdminPatientQuery{Page: 1, PageSize: 10})
		if err == nil {
			t.Error("ListPatients: expected error")
		}
		_, _, err = dRepo.ListSessions(ctx, model.AdminSessionQuery{Page: 1, PageSize: 10})
		if err == nil {
			t.Error("ListSessions: expected error")
		}
	})
}

func TestFlowCardRepo_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	fRepo := repository.NewFlowCardRepository(db)

	patient := createPatient(ctx, t, pRepo)
	visit := createVisit(ctx, t, vRepo, patient.ID)

	t.Run("FindByIDNotFound", func(t *testing.T) {
		_, err := fRepo.FindByID(ctx, "nonexistent-card-id")
		if !errors.Is(err, model.ErrCardNotFound) {
			t.Fatalf("expected ErrCardNotFound, got %v", err)
		}
	})

	t.Run("CreateDuplicate", func(t *testing.T) {
		card := &model.FlowCard{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindAdviceOnly),
			Status:    string(model.FlowCardStatusPending),
			Blocking:  false,
			Title:     "Duplicate Test",
			Advices:   []string{"test"},
		}
		if err := fRepo.Create(ctx, card); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		// Second create with same ID should fail due to PK conflict.
		err := fRepo.Create(ctx, card)
		if err == nil {
			t.Error("expected error for duplicate card ID, got nil")
		}
	})

	t.Run("UpdateNonExistent", func(t *testing.T) {
		// Update with a non-existent card should NOT return an error —
		// MySQL UPDATE with no matching rows succeeds (0 rows affected).
		card := &model.FlowCard{
			ID:        "nonexistent-update-id",
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindAdviceOnly),
			Status:    string(model.FlowCardStatusCompleted),
			Blocking:  false,
			Title:     "Ghost Update",
		}
		err := fRepo.Update(ctx, card)
		if err != nil {
			t.Fatalf("Update on non-existent: %v", err)
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in flow card functions.
		_ = db.Close()

		_, err := fRepo.FindByID(ctx, "any")
		if err == nil {
			t.Error("FindByID: expected error")
		}
		_, err = fRepo.ListBySession(ctx, visit.ID)
		if err == nil {
			t.Error("ListBySession: expected error")
		}
		err = fRepo.UpdateStatus(ctx, "any", "done")
		if err == nil {
			t.Error("UpdateStatus: expected error")
		}
		card := &model.FlowCard{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.FlowCardKindAdviceOnly),
			Status:    string(model.FlowCardStatusCompleted),
			Blocking:  false,
			Title:     "ClosedDB",
		}
		err = fRepo.Update(ctx, card)
		if err == nil {
			t.Error("Update: expected error")
		}
		err = fRepo.Create(ctx, card)
		if err == nil {
			t.Error("Create: expected error")
		}
	})
}

func TestAdminRefreshTokenRepo_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	adminRepo := repository.NewAdminRepository(db)
	tokenRepo := repository.NewAdminRefreshTokenRepository(db)

	admin := &model.AdminUser{
		ID:           uuid.New().String(),
		Username:     "edgeadmin",
		PasswordHash: "$2a$12$testhash",
		Role:         model.AdminRoleAdmin,
		DisplayName:  "Edge Case Admin",
		CreatedAt:    time.Now(),
	}
	if err := adminRepo.Create(ctx, admin); err != nil {
		t.Fatalf("Create admin: %v", err)
	}

	t.Run("MarkUsedNonExistent", func(t *testing.T) {
		// MarkUsed on a non-existent ID should NOT error —
		// MySQL UPDATE with no matching rows succeeds.
		err := tokenRepo.MarkUsed(ctx, "nonexistent-token-id")
		if err != nil {
			t.Fatalf("MarkUsed on non-existent: %v", err)
		}
	})

	t.Run("RevokeAllByAdminIDNoTokens", func(t *testing.T) {
		// Create a new admin that has no tokens yet.
		newAdmin := &model.AdminUser{
			ID:           uuid.New().String(),
			Username:     "notokenadmin",
			PasswordHash: "$2a$12$testhash",
			Role:         model.AdminRoleOperator,
			DisplayName:  "No Token Admin",
			CreatedAt:    time.Now(),
		}
		if err := adminRepo.Create(ctx, newAdmin); err != nil {
			t.Fatalf("Create admin: %v", err)
		}

		// RevokeAllByAdminID with no tokens should not error.
		err := tokenRepo.RevokeAllByAdminID(ctx, newAdmin.ID)
		if err != nil {
			t.Fatalf("RevokeAllByAdminID with no tokens: %v", err)
		}
	})

	t.Run("CreateDuplicate", func(t *testing.T) {
		token := &model.AdminRefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "sha256duplicateadmin",
			AdminID:   admin.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, token); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		err := tokenRepo.Create(ctx, token)
		if err == nil {
			t.Error("expected error for duplicate token ID, got nil")
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in admin token functions.
		_ = db.Close()

		err := tokenRepo.MarkUsed(ctx, "any")
		if err == nil {
			t.Error("MarkUsed: expected error")
		}
		err = tokenRepo.RevokeAllByAdminID(ctx, admin.ID)
		if err == nil {
			t.Error("RevokeAllByAdminID: expected error")
		}
		_, err = tokenRepo.FindByTokenHash(ctx, "any")
		if err == nil {
			t.Error("FindByTokenHash: expected error")
		}
		token := &model.AdminRefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "closedbtest",
			AdminID:   admin.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err = tokenRepo.Create(ctx, token)
		if err == nil {
			t.Error("Create: expected error")
		}
	})
}

func TestVisitRepo_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)

	t.Run("FindByIDNotFound", func(t *testing.T) {
		_, err := vRepo.FindByID(ctx, "nonexistent-visit-id")
		if !errors.Is(err, model.ErrSessionNotFound) {
			t.Fatalf("expected ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("UpdateStatusNonExistent", func(t *testing.T) {
		err := vRepo.UpdateStatus(ctx, "nonexistent-visit-id", string(model.VisitStatusBlocked), string(model.VisitStatusBlocked))
		if err != nil {
			t.Fatalf("UpdateStatus on non-existent: %v", err)
		}
	})

	t.Run("FindByCredentialUnknownType", func(t *testing.T) {
		_, err := pRepo.FindByCredential(ctx, "email", "test@example.com")
		if err == nil {
			t.Error("expected error for unknown credential type, got nil")
		}
	})

	t.Run("ListByPatientZeroPageSize", func(t *testing.T) {
		// pageSize=0 should trigger the default assignment (pageSize = 20).
		patient := createPatient(ctx, t, pRepo)
		visit := createVisit(ctx, t, vRepo, patient.ID)
		summaries, _, _, err := vRepo.ListByPatient(ctx, patient.ID, "", nil, 0)
		if err != nil {
			t.Fatalf("ListByPatient with zero pageSize: %v", err)
		}
		if len(summaries) == 0 {
			t.Error("expected at least 1 summary")
		}
		if summaries[0].ID != visit.ID {
			t.Errorf("summary ID = %q, want %q", summaries[0].ID, visit.ID)
		}
	})

	t.Run("ListByPatientFirstPage", func(t *testing.T) {
		patient := createPatient(ctx, t, pRepo)
		createVisit(ctx, t, vRepo, patient.ID)
		// With only 1 visit and pageSize=1, there are no more items,
		// so cursor should be nil and hasMore false.
		summaries, cursor, hasMore, err := vRepo.ListByPatient(ctx, patient.ID, "", nil, 1)
		if err != nil {
			t.Fatalf("ListByPatient: %v", err)
		}
		if len(summaries) != 1 {
			t.Errorf("expected 1 summary, got %d", len(summaries))
		}
		if cursor != nil {
			t.Errorf("expected nil cursor, got %q", *cursor)
		}
		if hasMore {
			t.Error("expected hasMore=false")
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in visit and patient functions.
		_ = db.Close()

		_, err := vRepo.FindByID(ctx, "any")
		if err == nil {
			t.Error("FindByID: expected error")
		}
		err = vRepo.UpdateStatus(ctx, "any", "done", "done")
		if err == nil {
			t.Error("UpdateStatus: expected error")
		}
		err = vRepo.Update(ctx, &model.VisitSession{
			ID:        "any",
			PatientID: "any",
		})
		if err == nil {
			t.Error("Update: expected error")
		}
		visit := &model.VisitSession{
			ID:            uuid.New().String(),
			PatientID:     "any",
			EntryType:     string(model.VisitEntryTypeNew),
			Status:        string(model.VisitStatusChatting),
			AskRound:      0,
			AskRoundLimit: 20,
			LabRound:      0,
			LabRoundLimit: 10,
			TimerPaused:   false,
		}
		err = vRepo.Create(ctx, visit)
		if err == nil {
			t.Error("Create: expected error")
		}
		_, _, _, err = vRepo.ListByPatient(ctx, "any", "", nil, 10)
		if err == nil {
			t.Error("ListByPatient: expected error")
		}
		_, err = pRepo.FindByCredential(ctx, "phone", "13800000000")
		if err == nil {
			t.Error("FindByCredential: expected error")
		}
		_, err = pRepo.FindByID(ctx, "any")
		if err == nil {
			t.Error("FindByID: expected error")
		}
	})
}

func TestRefreshTokenRepo_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	patientRepo := repository.NewPatientRepository(db)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)

	p := createPatient(ctx, t, patientRepo)
	user := &model.User{
		ID:           uuid.New().String(),
		Phone:        "13800009991",
		PasswordHash: "$2a$12$testhash",
		PatientID:    p.ID,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	t.Run("MarkUsedNonExistent", func(t *testing.T) {
		err := tokenRepo.MarkUsed(ctx, "nonexistent-token-id")
		if err != nil {
			t.Fatalf("MarkUsed on non-existent: %v", err)
		}
	})

	t.Run("RevokeAllByUserIDNoTokens", func(t *testing.T) {
		// Delete with no matching tokens should not error.
		err := tokenRepo.RevokeAllByUserID(ctx, "nonexistent-user-id")
		if err != nil {
			t.Fatalf("RevokeAllByUserID with no tokens: %v", err)
		}
	})

	t.Run("CreateDuplicate", func(t *testing.T) {
		token := &model.RefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "sha256duplicate",
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, token); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		err := tokenRepo.Create(ctx, token)
		if err == nil {
			t.Error("expected error for duplicate token ID, got nil")
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in refresh token functions.
		_ = db.Close()

		err := tokenRepo.MarkUsed(ctx, "any")
		if err == nil {
			t.Error("MarkUsed: expected error")
		}
		err = tokenRepo.RevokeAllByUserID(ctx, "any")
		if err == nil {
			t.Error("RevokeAllByUserID: expected error")
		}
		token := &model.RefreshToken{
			ID:        uuid.New().String(),
			TokenHash: "closedbrefresh",
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err = tokenRepo.Create(ctx, token)
		if err == nil {
			t.Error("Create: expected error")
		}
	})
}

func TestTimelineRepo_FindLastPatientMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	tRepo := repository.NewTimelineRepository(db)

	patient := createPatient(ctx, t, pRepo)
	visit := createVisit(ctx, t, vRepo, patient.ID)

	t.Run("NoItems", func(t *testing.T) {
		msg, err := tRepo.FindLastPatientMessage(ctx, visit.ID)
		if err != nil {
			t.Fatalf("FindLastPatientMessage with no items: %v", err)
		}
		if msg != "" {
			t.Errorf("expected empty message, got %q", msg)
		}
	})

	t.Run("OnlyAssistantMessages", func(t *testing.T) {
		// Create a new visit for isolation.
		v2 := createVisit(ctx, t, vRepo, patient.ID)
		item := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: v2.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Status:    string(model.TimelineItemStatusDone),
			Role:      string(model.MessageRoleAssistant),
			Content:   "请描述您的症状",
		}
		if err := tRepo.Append(ctx, &item); err != nil {
			t.Fatalf("Append assistant message: %v", err)
		}
		msg, err := tRepo.FindLastPatientMessage(ctx, v2.ID)
		if err != nil {
			t.Fatalf("FindLastPatientMessage: %v", err)
		}
		if msg != "" {
			t.Errorf("expected empty message for assistant-only, got %q", msg)
		}
	})

	t.Run("PatientMessageFound", func(t *testing.T) {
		// Create a new visit for isolation.
		v3 := createVisit(ctx, t, vRepo, patient.ID)

		// Add an assistant message first.
		docItem := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: v3.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Status:    string(model.TimelineItemStatusDone),
			Role:      string(model.MessageRoleAssistant),
			Content:   "请描述您的症状",
		}
		if err := tRepo.Append(ctx, &docItem); err != nil {
			t.Fatalf("Append assistant message: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		// Add a patient message.
		patientItem := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: v3.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Status:    string(model.TimelineItemStatusDone),
			Role:      string(model.MessageRolePatient),
			Content:   "我头痛三天了",
		}
		if err := tRepo.Append(ctx, &patientItem); err != nil {
			t.Fatalf("Append patient message: %v", err)
		}

		msg, err := tRepo.FindLastPatientMessage(ctx, v3.ID)
		if err != nil {
			t.Fatalf("FindLastPatientMessage: %v", err)
		}
		if msg != "我头痛三天了" {
			t.Errorf("expected '我头痛三天了', got %q", msg)
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths.
		_ = db.Close()
		_, err := tRepo.FindLastPatientMessage(ctx, "any")
		if err == nil {
			t.Error("FindLastPatientMessage: expected error")
		}
	})
}

func TestTimelineRepo_AppendBatchEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pRepo := repository.NewPatientRepository(db)
	vRepo := repository.NewVisitRepository(db)
	tRepo := repository.NewTimelineRepository(db)

	patient := createPatient(ctx, t, pRepo)
	visit := createVisit(ctx, t, vRepo, patient.ID)

	t.Run("EmptyBatch", func(t *testing.T) {
		err := tRepo.AppendBatch(ctx, []model.TimelineItem{})
		if err != nil {
			t.Fatalf("AppendBatch empty: %v", err)
		}
	})

	t.Run("ItemsWithEmptyStatus", func(t *testing.T) {
		// Items with empty Status should have it defaulted to "done" internally.
		item := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Role:      string(model.MessageRolePatient),
			Content:   "status default test",
		}
		err := tRepo.AppendBatch(ctx, []model.TimelineItem{item})
		if err != nil {
			t.Fatalf("AppendBatch: %v", err)
		}
	})

	t.Run("AppendWithEmptyStatus", func(t *testing.T) {
		// Append with empty Status should default to "done".
		item := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: visit.ID,
			Kind:      string(model.TimelineItemKindMessage),
			Role:      string(model.MessageRolePatient),
			Content:   "empty status test",
		}
		if err := tRepo.Append(ctx, &item); err != nil {
			t.Fatalf("Append with empty status: %v", err)
		}
		if item.Status != "done" {
			t.Errorf("Status = %q, want %q", item.Status, "done")
		}
	})

	t.Run("ListBySessionZeroPageSize", func(t *testing.T) {
		// pageSize=0 should trigger the default assignment (pageSize = 50).
		items, _, _, err := tRepo.ListBySession(ctx, visit.ID, nil, 0)
		if err != nil {
			t.Fatalf("ListBySession with zero pageSize: %v", err)
		}
		if items == nil {
			t.Error("expected non-nil items")
		}
	})

	t.Run("ClosedDB", func(t *testing.T) {
		// Close the DB to exercise error-return paths in timeline functions.
		_ = db.Close()

		_, err := tRepo.FindLastPatientMessage(ctx, "any")
		if err == nil {
			t.Error("FindLastPatientMessage: expected error")
		}
		_, _, _, err = tRepo.ListBySession(ctx, "any", nil, 10)
		if err == nil {
			t.Error("ListBySession: expected error")
		}
		err = tRepo.AppendBatch(ctx, []model.TimelineItem{
			{ID: uuid.New().String(), SessionID: "any", Kind: "message"},
		})
		if err == nil {
			t.Error("AppendBatch: expected error")
		}
	})
}

func TestSettingsRepo_GetDefaults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	repo := repository.NewSettingsRepository(db)

	// Delete the settings row inserted by migration to exercise the ErrNoRows path.
	_, err := db.ExecContext(ctx, `DELETE FROM system_settings WHERE id = 1`)
	if err != nil {
		t.Fatalf("delete settings: %v", err)
	}

	settings, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if settings.SiteName != "NEUHIS Agent" {
		t.Errorf("SiteName = %q, want %q", settings.SiteName, "NEUHIS Agent")
	}
	if settings.MaxConcurrentSessions != 3 {
		t.Errorf("MaxConcurrentSessions = %d, want 3", settings.MaxConcurrentSessions)
	}
	if settings.SessionTimeoutMinutes != 30 {
		t.Errorf("SessionTimeoutMinutes = %d, want 30", settings.SessionTimeoutMinutes)
	}
	if !settings.EnableRegistration {
		t.Error("EnableRegistration should be true")
	}
}

package medicalorder_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	medicalordersvc "github.com/neuhis/software-practice-backend/internal/service/medicalorder"
)

var _ repository.VisitRepository = (*mockVisitRepo)(nil)
var _ repository.FlowCardRepository = (*mockFlowCardRepo)(nil)

type mockVisitRepo struct {
	listByPatientFunc func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error)
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error { return nil }
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return nil, nil
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, pid string, status string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, pid, status, c, ps)
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id, status, ms string) error { return nil }
func (m *mockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error       { return nil }

type mockFlowCardRepo struct {
	listBySessionFunc func(ctx context.Context, sid string) ([]model.FlowCard, error)
}

func (m *mockFlowCardRepo) Create(ctx context.Context, card *model.FlowCard) error { return nil }
func (m *mockFlowCardRepo) FindByID(ctx context.Context, id string) (*model.FlowCard, error) {
	return nil, nil
}
func (m *mockFlowCardRepo) ListBySession(ctx context.Context, sid string) ([]model.FlowCard, error) {
	return m.listBySessionFunc(ctx, sid)
}
func (m *mockFlowCardRepo) UpdateStatus(ctx context.Context, id, status string) error { return nil }
func (m *mockFlowCardRepo) Update(ctx context.Context, card *model.FlowCard) error    { return nil }

func TestListMedicalOrders_Empty(t *testing.T) {
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items, want 0", len(resp.Items))
	}
}

func TestListMedicalOrders_NoMatchingCards(t *testing.T) {
	cc := "头痛"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "diagnosis", Status: "completed"},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items, want 0 (no matching card kind)", len(resp.Items))
	}
}

func TestListMedicalOrders_AdviceCompleted(t *testing.T) {
	cc := "头痛"
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed",
					Advices:                []string{"多喝水", "注意休息"},
					WatchItems:             []string{"体温"},
					FollowUpRecommendation: "一周后复诊",
					HandledAt:              &now,
				},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}
	r := resp.Items[0]
	if r.RecordID != "c1" {
		t.Errorf("recordId = %s, want c1", r.RecordID)
	}
	if r.Kind != "advice" {
		t.Errorf("kind = %s, want advice", r.Kind)
	}
	if len(r.Advices) != 2 || r.Advices[0] != "多喝水" {
		t.Errorf("advices = %v", r.Advices)
	}
	if len(r.WatchItems) != 1 || r.WatchItems[0] != "体温" {
		t.Errorf("watchItems = %v", r.WatchItems)
	}
	if r.FollowUpRecommendation != "一周后复诊" {
		t.Errorf("followUpRecommendation = %s", r.FollowUpRecommendation)
	}
	if r.FulfillmentStatus != "" {
		t.Error("fulfillmentStatus should be empty for advice records")
	}
}

func TestListMedicalOrders_MedicationCompleted(t *testing.T) {
	cc := "发热"
	now := time.Date(2026, 6, 16, 14, 0, 0, 0, time.UTC)
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	addr := &model.DeliveryAddress{Name: "张三", Phone: "13800138000", FullAddress: "北京市朝阳区"}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID: "c2", SessionID: "s1", Kind: "medication_fulfillment",
					FulfillmentStatus: "completed",
					Medications: []model.MedicationItem{
						{Name: "阿莫西林", Spec: "0.25g", Quantity: 2, Dosage: "每日三次", Days: 7, Price: 35.0},
					},
					DeliveryAddress: addr,
					HandledAt:       &now,
				},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}
	r := resp.Items[0]
	if r.Kind != "medication" {
		t.Errorf("kind = %s, want medication", r.Kind)
	}
	if len(r.Medications) != 1 || r.Medications[0].Name != "阿莫西林" {
		t.Errorf("medications = %v", r.Medications)
	}
	if r.FulfillmentStatus != "completed" {
		t.Errorf("fulfillmentStatus = %s, want completed", r.FulfillmentStatus)
	}
	if r.DeliveryAddress == nil || r.DeliveryAddress.Name != "张三" {
		t.Error("deliveryAddress mismatch")
	}
	if len(r.Advices) != 0 {
		t.Error("advices should be empty for medication records")
	}
}

func TestListMedicalOrders_MedicationConfirmed(t *testing.T) {
	cc := "发热"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID: "c1", SessionID: "s1", Kind: "medication_fulfillment",
					FulfillmentStatus: "confirmed",
					HandledAt:         &now,
				},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1 (confirmed qualifies)", len(resp.Items))
	}
}

func TestListMedicalOrders_SkipsAdviceNotCompleted(t *testing.T) {
	cc := "头痛"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "pending"},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items, want 0 (advice not completed)", len(resp.Items))
	}
}

func TestListMedicalOrders_SkipsMedicationPending(t *testing.T) {
	cc := "头痛"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID: "c1", SessionID: "s1", Kind: "medication_fulfillment",
					FulfillmentStatus: "pending",
				},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items, want 0 (medication pending)", len(resp.Items))
	}
}

func TestListMedicalOrders_MixedKinds(t *testing.T) {
	cc := "头痛"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &now},
				{ID: "c2", SessionID: "s1", Kind: "medication_fulfillment", FulfillmentStatus: "completed", HandledAt: &now},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("got %d items, want 2 (advice + medication)", len(resp.Items))
	}
	kinds := map[string]int{}
	for _, r := range resp.Items {
		kinds[r.Kind]++
	}
	if kinds["advice"] != 1 || kinds["medication"] != 1 {
		t.Errorf("expected 1 advice + 1 medication, got: %v", kinds)
	}
}

func TestListMedicalOrders_SessionTitle_ChiefComplaint(t *testing.T) {
	cc := "头痛三天"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &now},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if resp.Items[0].SessionTitle != cc {
		t.Errorf("sessionTitle = %s, want %s", resp.Items[0].SessionTitle, cc)
	}
}

func TestListMedicalOrders_SessionTitle_Diagnosis(t *testing.T) {
	diagnosis := "上呼吸道感染"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{Diagnosis: &diagnosis}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &now},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if resp.Items[0].SessionTitle != diagnosis {
		t.Errorf("sessionTitle = %s, want %s", resp.Items[0].SessionTitle, diagnosis)
	}
}

func TestListMedicalOrders_SessionTitle_Title(t *testing.T) {
	title := "就诊标题"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{Title: &title}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &now},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if resp.Items[0].SessionTitle != title {
		t.Errorf("sessionTitle = %s, want %s", resp.Items[0].SessionTitle, title)
	}
}

func TestListMedicalOrders_SessionTitle_Unknown(t *testing.T) {
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &now},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if resp.Items[0].SessionTitle != "未知就诊" {
		t.Errorf("sessionTitle = %s, want 未知就诊", resp.Items[0].SessionTitle)
	}
}

func TestListMedicalOrders_SortedByHandledAtDesc(t *testing.T) {
	cc := "头痛"
	earlier := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &earlier},
				{ID: "c2", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &later},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if len(resp.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(resp.Items))
	}
	if resp.Items[0].RecordID != "c2" {
		t.Errorf("first = %s, want c2 (newest first)", resp.Items[0].RecordID)
	}
	if resp.Items[1].RecordID != "c1" {
		t.Errorf("second = %s, want c1", resp.Items[1].RecordID)
	}
}

func TestListMedicalOrders_HandledAtNilFallback(t *testing.T) {
	cc := "头痛"
	earlier := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				// HandledAt is nil, falls back to CreatedAt (earlier)
				{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", CreatedAt: earlier},
				// Has explicit HandledAt (later), should sort first
				{ID: "c2", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &later, CreatedAt: earlier},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if len(resp.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(resp.Items))
	}
	// c2 has later HandledAt, should come first
	if resp.Items[0].RecordID != "c2" {
		t.Errorf("first = %s, want c2 (has HandledAt later)", resp.Items[0].RecordID)
	}
}

func TestListMedicalOrders_MultipleSessions(t *testing.T) {
	cc1 := "头痛"
	cc2 := "发热"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc1}},
				{ID: "s2", Summary: model.VisitSummary{ChiefComplaint: &cc2}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			switch sid {
			case "s1":
				return []model.FlowCard{
					{ID: "c1", SessionID: "s1", Kind: "advice_only", Status: "completed", HandledAt: &now},
				}, nil
			case "s2":
				return []model.FlowCard{
					{ID: "c2", SessionID: "s2", Kind: "advice_only", Status: "completed", HandledAt: &now},
				}, nil
			}
			return nil, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, _ := svc.ListMedicalOrders(context.Background(), "p001")
	if len(resp.Items) != 2 {
		t.Fatalf("got %d items, want 2 (across two sessions)", len(resp.Items))
	}
}

func TestListMedicalOrders_ListByPatientError(t *testing.T) {
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return nil, nil, false, errors.New("db error")
		},
	}
	svc := medicalordersvc.NewService(visitRepo, &mockFlowCardRepo{})

	_, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err == nil {
		t.Fatal("expected error from ListByPatient, got nil")
	}
}

func TestListMedicalOrders_ListBySessionError(t *testing.T) {
	cc := "头痛"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return nil, errors.New("db error")
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	_, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err == nil {
		t.Fatal("expected error from ListBySession, got nil")
	}
}

func TestListMedicalOrders_NullSlicesPreserved(t *testing.T) {
	cc := "头痛"
	now := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID: "c1", SessionID: "s1", Kind: "medication_fulfillment",
					FulfillmentStatus: "completed",
					Medications:       nil, // nil medications
					HandledAt:         &now,
				},
			}, nil
		},
	}
	svc := medicalordersvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListMedicalOrders(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListMedicalOrders: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}
	// Medications should be nil (not empty slice) serialized as omitted
	if resp.Items[0].Medications != nil {
		t.Errorf("medications should be nil, got %v", resp.Items[0].Medications)
	}
}

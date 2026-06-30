package billing_test

import (
	"context"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	billingsvc "github.com/neuhis/software-practice-backend/internal/service/billing"
)

var _ repository.VisitRepository = (*mockVisitRepo)(nil)
var _ repository.FlowCardRepository = (*mockFlowCardRepo)(nil)

type mockVisitRepo struct {
	listByPatientFunc func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error)
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error { return nil }
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return nil, nil
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, pid, c, ps)
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

func TestListBillingRecords_Empty(t *testing.T) {
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return nil, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items, want 0", len(resp.Items))
	}
}

func TestListBillingRecords_NoPaymentCards(t *testing.T) {
	chiefComplaint := "头痛"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &chiefComplaint}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			// Return non-payment cards only
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "diagnosis", Status: "completed"},
			}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items, want 0 (no paid payment cards)", len(resp.Items))
	}
}

func TestListBillingRecords_WithPaymentCards(t *testing.T) {
	chiefComplaint := "头痛"
	handledAt := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &chiefComplaint}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID:              "c1",
					SessionID:       "s1",
					Kind:            "payment",
					PaymentStatus:   "paid",
					PaymentID:       "pay-1",
					Purpose:         "lab",
					TotalAmount:     150.0,
					InsuranceAmount: 100.0,
					SelfPayAmount:   50.0,
					Items:           []model.PaymentLineItem{{Name: "血常规", Amount: 150.0}},
					HandledAt:       &handledAt,
				},
			}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}
	r := resp.Items[0]
	if r.PaymentID != "pay-1" {
		t.Errorf("paymentId = %s, want pay-1", r.PaymentID)
	}
	if r.SessionID != "s1" {
		t.Errorf("sessionId = %s, want s1", r.SessionID)
	}
	if r.SessionTitle != "头痛" {
		t.Errorf("sessionTitle = %s, want 头痛", r.SessionTitle)
	}
	if r.Purpose != "lab" {
		t.Errorf("purpose = %s, want lab", r.Purpose)
	}
	if r.TotalAmount != 150.0 {
		t.Errorf("totalAmount = %f, want 150.0", r.TotalAmount)
	}
}

func TestListBillingRecords_SessionTitleFallback(t *testing.T) {
	diagnosis := "上呼吸道感染"
	title := "就诊标题"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			// ChiefComplaint is nil, should fall back to diagnosis
			return []model.VisitSessionSummary{
				{
					ID: "s1",
					Summary: model.VisitSummary{
						Title:     &title,
						Diagnosis: &diagnosis,
					},
				},
			}, nil, false, nil
		},
	}
	handledAt := time.Now()
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID:            "c1",
					SessionID:     "s1",
					Kind:          "payment",
					PaymentStatus: "paid",
					Purpose:       "medication",
					TotalAmount:   200.0,
					HandledAt:     &handledAt,
				},
			}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}
	// Should use diagnosis since chiefComplaint is nil
	if resp.Items[0].SessionTitle != diagnosis {
		t.Errorf("sessionTitle = %s, want %s (diagnosis fallback)", resp.Items[0].SessionTitle, diagnosis)
	}
}

func TestListBillingRecords_UnknownTitle(t *testing.T) {
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{}},
			}, nil, false, nil
		},
	}
	handledAt := time.Now()
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{
					ID: "c1", SessionID: "s1", Kind: "payment",
					PaymentStatus: "paid", Purpose: "lab", TotalAmount: 100.0,
					HandledAt: &handledAt,
				},
			}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if resp.Items[0].SessionTitle != "未知就诊" {
		t.Errorf("sessionTitle = %s, want 未知就诊", resp.Items[0].SessionTitle)
	}
}

func TestListBillingRecords_MultipleSessions(t *testing.T) {
	cc1 := "头痛"
	cc2 := "发热"
	handledAt1 := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	handledAt2 := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
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
				return []model.FlowCard{{
					ID: "c1", SessionID: "s1", Kind: "payment", PaymentStatus: "paid",
					Purpose: "lab", TotalAmount: 100.0, HandledAt: &handledAt1,
				}}, nil
			case "s2":
				return []model.FlowCard{{
					ID: "c2", SessionID: "s2", Kind: "payment", PaymentStatus: "paid",
					Purpose: "medication", TotalAmount: 200.0, HandledAt: &handledAt2,
				}}, nil
			}
			return nil, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(resp.Items))
	}
	if resp.Items[0].SessionID != "s2" {
		t.Errorf("first item = %s, want s2 (newest first)", resp.Items[0].SessionID)
	}
}

func TestListBillingRecords_SkipsUnpaid(t *testing.T) {
	cc := "头痛"
	handledAt := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{
				{ID: "c1", SessionID: "s1", Kind: "payment", PaymentStatus: "unpaid", TotalAmount: 100.0},
				{ID: "c2", SessionID: "s1", Kind: "payment", PaymentStatus: "paid", Purpose: "lab", TotalAmount: 150.0, HandledAt: &handledAt},
			}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1 (skip unpaid)", len(resp.Items))
	}
}

func TestListBillingRecords_TitleFromTitleField(t *testing.T) {
	sessionTitle := "会话标题"
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{Title: &sessionTitle}},
			}, nil, false, nil
		},
	}
	handledAt := time.Now()
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{{
				ID: "c1", SessionID: "s1", Kind: "payment", PaymentStatus: "paid",
				Purpose: "lab", TotalAmount: 100.0, HandledAt: &handledAt,
			}}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if resp.Items[0].SessionTitle != sessionTitle {
		t.Errorf("sessionTitle = %s, want %s", resp.Items[0].SessionTitle, sessionTitle)
	}
}

func TestListBillingRecords_WithQuantity(t *testing.T) {
	cc := "头痛"
	qty := 3
	handledAt := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{{
				ID: "c1", SessionID: "s1", Kind: "payment", PaymentStatus: "paid",
				Purpose: "medication", TotalAmount: 300.0,
				Items:     []model.PaymentLineItem{{Name: "阿莫西林", Amount: 100.0, Quantity: qty}},
				HandledAt: &handledAt,
			}}, nil
		},
	}
	svc := billingsvc.NewService(visitRepo, flowCardRepo)

	resp, err := svc.ListBillingRecords(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListBillingRecords: %v", err)
	}
	if len(resp.Items[0].Items) != 1 {
		t.Fatalf("got %d line items, want 1", len(resp.Items[0].Items))
	}
	if resp.Items[0].Items[0].Quantity == nil || *resp.Items[0].Items[0].Quantity != 3 {
		t.Error("quantity should be 3")
	}
}

package workbench_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
)

// ---- Mock Repositories ----

type mockPatientRepo struct {
	findByCredFunc func(ctx context.Context, ct, cred string) (*model.PatientProfile, error)
	findByIDFunc   func(ctx context.Context, id string) (*model.PatientProfile, error)
	createFunc     func(ctx context.Context, p *model.PatientProfile) error
	updateFunc     func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error)
}

func (m *mockPatientRepo) FindByCredential(ctx context.Context, ct, cred string) (*model.PatientProfile, error) {
	return m.findByCredFunc(ctx, ct, cred)
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

type mockVisitRepo struct {
	createFunc        func(ctx context.Context, v *model.VisitSession) error
	findByIDFunc      func(ctx context.Context, id string) (*model.VisitSession, error)
	listByPatientFunc func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error)
	updateStatusFunc  func(ctx context.Context, id, status, ms string) error
	updateFunc        func(ctx context.Context, v *model.VisitSession) error
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error {
	return m.createFunc(ctx, v)
}
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, pid, c, ps)
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id, status, ms string) error {
	return m.updateStatusFunc(ctx, id, status, ms)
}
func (m *mockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error {
	return m.updateFunc(ctx, v)
}

type mockTimelineRepo struct {
	appendFunc      func(ctx context.Context, item *model.TimelineItem) error
	appendBatchFunc func(ctx context.Context, items []model.TimelineItem) error
	listFunc        func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error)
	updateFunc      func(ctx context.Context, id, status string) error
}

func (m *mockTimelineRepo) Append(ctx context.Context, item *model.TimelineItem) error {
	return m.appendFunc(ctx, item)
}
func (m *mockTimelineRepo) AppendBatch(ctx context.Context, items []model.TimelineItem) error {
	return m.appendBatchFunc(ctx, items)
}
func (m *mockTimelineRepo) ListBySession(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
	return m.listFunc(ctx, sid, c, ps)
}
func (m *mockTimelineRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return m.updateFunc(ctx, id, status)
}

type mockFlowCardRepo struct {
	createFunc       func(ctx context.Context, card *model.FlowCard) error
	findByIDFunc     func(ctx context.Context, id string) (*model.FlowCard, error)
	listFunc         func(ctx context.Context, sid string) ([]model.FlowCard, error)
	updateStatusFunc func(ctx context.Context, id, status string) error
	updateFunc       func(ctx context.Context, card *model.FlowCard) error
}

func (m *mockFlowCardRepo) Create(ctx context.Context, card *model.FlowCard) error {
	return m.createFunc(ctx, card)
}
func (m *mockFlowCardRepo) FindByID(ctx context.Context, id string) (*model.FlowCard, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockFlowCardRepo) ListBySession(ctx context.Context, sid string) ([]model.FlowCard, error) {
	return m.listFunc(ctx, sid)
}
func (m *mockFlowCardRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return m.updateStatusFunc(ctx, id, status)
}
func (m *mockFlowCardRepo) Update(ctx context.Context, card *model.FlowCard) error {
	return m.updateFunc(ctx, card)
}

func makeSession(patientID string) *model.VisitSession {
	return &model.VisitSession{
		ID:            uuid.New().String(),
		PatientID:     patientID,
		EntryType:     "new",
		Status:        "chatting",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AskRound:      0,
		AskRoundLimit: 20,
		LabRound:      0,
		LabRoundLimit: 10,
		TimerPaused:   false,
		Summary:       model.VisitSummary{},
	}
}

func makePatient() *model.PatientProfile {
	return &model.PatientProfile{
		ID:        uuid.New().String(),
		Name:      "测试",
		Gender:    "male",
		Age:       30,
		UpdatedAt: time.Now(),
	}
}

func makeCard(cardID, sessionID, kind string, blocking bool) *model.FlowCard {
	return &model.FlowCard{
		ID:        cardID,
		SessionID: sessionID,
		Kind:      kind,
		Status:    "pending",
		Blocking:  blocking,
		Title:     "测试卡片",
		CreatedAt: time.Now(),
	}
}

func newDefaultMocks() (*mockPatientRepo, *mockVisitRepo, *mockTimelineRepo, *mockFlowCardRepo) {
	p := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return makePatient(), nil
		},
	}
	v := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return makeSession("p001"), nil
		},
		updateFunc: func(ctx context.Context, vs *model.VisitSession) error { return nil },
	}
	t := &mockTimelineRepo{
		appendFunc:      func(ctx context.Context, item *model.TimelineItem) error { return nil },
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error { return nil },
		listFunc: func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{{ID: "t1", Role: "patient", Content: "hello", Kind: "message"}}, nil, false, nil
		},
	}
	f := &mockFlowCardRepo{
		createFunc: func(ctx context.Context, card *model.FlowCard) error { return nil },
		findByIDFunc: func(ctx context.Context, id string) (*model.FlowCard, error) {
			return makeCard(id, "s1", "lab_decision", true), nil
		},
		listFunc:         func(ctx context.Context, sid string) ([]model.FlowCard, error) { return nil, nil },
		updateFunc:       func(ctx context.Context, card *model.FlowCard) error { return nil },
		updateStatusFunc: func(ctx context.Context, id, status string) error { return nil },
	}
	return p, v, t, f
}

func newSvc(p *mockPatientRepo, v *mockVisitRepo, t *mockTimelineRepo, f *mockFlowCardRepo) *wbsvc.Service {
	return wbsvc.NewService(p, v, t, f, nil, "http")
}

// ---- Tests ----

func TestSendMessage(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SendMessage(ctx, wbsvc.SendMessageInput{
		SessionID:       "s1",
		Content:         "我头痛",
		ClientMessageID: "msg-001",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if result.PatientMessage.Content != "我头痛" {
		t.Errorf("content = %s", result.PatientMessage.Content)
	}
	if result.AssistantPlaceholder == nil {
		t.Error("expected AssistantPlaceholder")
	}
}

func TestSendMessage_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	_, err := svc.SendMessage(ctx, wbsvc.SendMessageInput{
		SessionID: "bad-id", Content: "test", ClientMessageID: "m1",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSubmitLabDecision_Accepted(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1",
		CardID:    "f1",
		Decision:  "accepted",
	})
	if err != nil {
		t.Fatalf("SubmitLabDecision: %v", err)
	}
	if result.Status != "blocked" {
		t.Errorf("status = %s, want blocked", result.Status)
	}
}

func TestSubmitLabDecision_Skipped(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "skipped",
	})
	if err != nil {
		t.Fatalf("SubmitLabDecision: %v", err)
	}
	if result.Status != "diagnosis" {
		t.Errorf("status = %s, want diagnosis", result.Status)
	}
}

func TestSubmitLabDecision_Vetoed(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "vetoed",
	})
	if err != nil {
		t.Fatalf("SubmitLabDecision: %v", err)
	}
	if result.Status != "chatting" {
		t.Errorf("status = %s, want chatting", result.Status)
	}
}

func TestSubmitLabDecision_Invalid(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid decision")
	}
}

func TestSubmitPayment_Defer(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = 100.0
		return card, nil
	}
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab", Defer: true,
	})
	if err != nil {
		t.Fatalf("SubmitPayment: %v", err)
	}
	if result.Message != "支付已暂缓" {
		t.Errorf("message = %s", result.Message)
	}
}

func TestSubmitPayment_Lab(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = 50.0
		card.Purpose = "lab"
		return card, nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error { return nil }
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab",
	})
	if err != nil {
		t.Fatalf("SubmitPayment: %v", err)
	}
	if result.Status != "diagnosis" {
		t.Errorf("status = %s, want diagnosis", result.Status)
	}
}

func TestSubmitPayment_Medication(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = 100.0
		card.Purpose = "medication"
		return card, nil
	}
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "medication",
	})
	if err != nil {
		t.Fatalf("SubmitPayment: %v", err)
	}
	if result.Status != "blocked" {
		t.Errorf("status = %s, want blocked", result.Status)
	}
}

func TestPauseTimer(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	session, err := svc.PauseTimer(ctx, "s1")
	if err != nil {
		t.Fatalf("PauseTimer: %v", err)
	}
	if !session.TimerPaused {
		t.Error("TimerPaused should be true")
	}
	if session.PausedAt == nil {
		t.Error("PausedAt should be set")
	}
}

func TestResumeTimer(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	session, err := svc.ResumeTimer(ctx, "s1")
	if err != nil {
		t.Fatalf("ResumeTimer: %v", err)
	}
	if session.TimerPaused {
		t.Error("TimerPaused should be false")
	}
}

func TestExitVisit_NoFee(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "chatting"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.TerminalReason != "exited" {
		t.Errorf("terminalReason = %s", result.TerminalReason)
	}
	if result.Consequence == nil || result.Consequence.Kind != "no_fee" {
		t.Error("expected no_fee consequence for chatting exit")
	}
}

func TestExitVisit_Refundable(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "blocked"
		diag := "感冒"
		s.Summary.Diagnosis = &diag
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.Consequence == nil || result.Consequence.Kind != "refundable" {
		t.Errorf("consequence = %v, want refundable", result.Consequence)
	}
}

func TestClassifyIntent_FollowUp(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.ClassifyIntent(ctx, wbsvc.ClassifyIntentInput{
		SessionID: "s1",
		Content:   "我想复查一下",
	})
	if err != nil {
		t.Fatalf("ClassifyIntent: %v", err)
	}
	if result.Intent != "follow_up" {
		t.Errorf("intent = %s, want follow_up", result.Intent)
	}
}

func TestClassifyIntent_Consultation(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.ClassifyIntent(ctx, wbsvc.ClassifyIntentInput{
		SessionID: "s1",
		Content:   "这是什么病",
	})
	if err != nil {
		t.Fatalf("ClassifyIntent: %v", err)
	}
	if result.Intent != "consultation" {
		t.Errorf("intent = %s, want consultation", result.Intent)
	}
}

func TestReportVitals_Normal(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "patient_report",
		Vitals: map[string]interface{}{
			"heartRate": 72.0,
			"spo2":      98.0,
		},
	})
	if err != nil {
		t.Fatalf("ReportVitals: %v", err)
	}
	if result.Emergency {
		t.Error("normal vitals should not trigger emergency")
	}
}

func TestReportVitals_Critical(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "device",
		Vitals: map[string]interface{}{
			"heartRate": 150.0,
			"spo2":      85.0,
		},
	})
	if err != nil {
		t.Fatalf("ReportVitals: %v", err)
	}
	if !result.Emergency {
		t.Error("critical vitals should trigger emergency")
	}
	if result.Severity != "critical" {
		t.Errorf("severity = %s, want critical", result.Severity)
	}
}

func TestGetSession(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	session, err := svc.GetSession(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
}

func TestListTimeline(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	items, _, hasMore, err := svc.ListTimeline(ctx, "s1", nil, 50)
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("items = %d, want 1", len(items))
	}
	if hasMore {
		t.Error("hasMore should be false")
	}
}

func TestDismissEmergency(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "emergency_terminated"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	session, tlItem, err := svc.DismissEmergency(ctx, wbsvc.DismissEmergencyInput{
		SessionID: "s1",
	})
	if err != nil {
		t.Fatalf("DismissEmergency: %v", err)
	}
	if session.Status != "chatting" {
		t.Errorf("status = %s, want chatting", session.Status)
	}
	if tlItem == nil {
		t.Error("timelineItem should not be nil")
	}
}

func TestDismissEmergency_NotEmergency(t *testing.T) {
	mp, mv, mt, mf := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf)
	ctx := context.Background()

	_, _, err := svc.DismissEmergency(ctx, wbsvc.DismissEmergencyInput{
		SessionID: "s1",
	})
	if err != model.ErrValidation {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

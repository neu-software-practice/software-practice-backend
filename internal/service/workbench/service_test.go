package workbench_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
	visitsvc "github.com/neuhis/software-practice-backend/internal/service/visit"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
)

var _ repository.AddressRepository = (*mockAddressRepo)(nil)

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
	listByPatientFunc func(ctx context.Context, pid string, _ string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error)
	updateStatusFunc  func(ctx context.Context, id, status, ms string) error
	updateFunc        func(ctx context.Context, v *model.VisitSession) error
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error {
	return m.createFunc(ctx, v)
}
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, pid string, status string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, pid, status, c, ps)
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
func (m *mockTimelineRepo) FindLastPatientMessage(ctx context.Context, sessionID string) (string, error) {
	return "", nil
}
func (m *mockTimelineRepo) FindLastStreamingMessage(ctx context.Context, sessionID string) (*model.TimelineItem, error) {
	return nil, nil
}
func (m *mockTimelineRepo) UpdateContent(ctx context.Context, id string, item *model.TimelineItem) error {
	return nil
}
func (m *mockTimelineRepo) FindFlowCardByCardID(ctx context.Context, sessionID, cardID string) (*model.TimelineItem, error) {
	return nil, nil
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

type mockAddressRepo struct {
	createFunc                func(ctx context.Context, addr *model.Address) error
	findByIDFunc              func(ctx context.Context, id string) (*model.Address, error)
	listByPatientFunc         func(ctx context.Context, patientID string) ([]model.Address, error)
	countByPatientFunc        func(ctx context.Context, patientID string) (int, error)
	updateFunc                func(ctx context.Context, addr *model.Address) error
	deleteFunc                func(ctx context.Context, id string) error
	clearDefaultByPatientFunc func(ctx context.Context, patientID string) error
	setDefaultFunc            func(ctx context.Context, id, patientID string) error
}

func (m *mockAddressRepo) Create(ctx context.Context, addr *model.Address) error {
	return m.createFunc(ctx, addr)
}
func (m *mockAddressRepo) FindByID(ctx context.Context, id string) (*model.Address, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockAddressRepo) ListByPatient(ctx context.Context, patientID string) ([]model.Address, error) {
	return m.listByPatientFunc(ctx, patientID)
}
func (m *mockAddressRepo) CountByPatient(ctx context.Context, patientID string) (int, error) {
	return m.countByPatientFunc(ctx, patientID)
}
func (m *mockAddressRepo) Update(ctx context.Context, addr *model.Address) error {
	return m.updateFunc(ctx, addr)
}
func (m *mockAddressRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFunc(ctx, id)
}
func (m *mockAddressRepo) ClearDefaultByPatient(ctx context.Context, patientID string) error {
	return m.clearDefaultByPatientFunc(ctx, patientID)
}
func (m *mockAddressRepo) SetDefault(ctx context.Context, id, patientID string) error {
	return m.setDefaultFunc(ctx, id, patientID)
}

// ---- Helpers ----

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

func newDefaultMocks() (*mockPatientRepo, *mockVisitRepo, *mockTimelineRepo, *mockFlowCardRepo, *mockAddressRepo) {
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
	a := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return &model.Address{
				ID:        id,
				PatientID: "p001",
				Name:      "李明",
				Phone:     "13800002468",
				Province:  "辽宁省",
				City:      "沈阳市",
				District:  "浑南区",
				Detail:    "创新路195号",
			}, nil
		},
		createFunc: func(ctx context.Context, addr *model.Address) error { return nil },
		updateFunc: func(ctx context.Context, addr *model.Address) error { return nil },
		deleteFunc: func(ctx context.Context, id string) error { return nil },
	}
	return p, v, t, f, a
}

func newSvc(p *mockPatientRepo, v *mockVisitRepo, t *mockTimelineRepo, f *mockFlowCardRepo, a *mockAddressRepo) *wbsvc.Service {
	visitSvc := visitsvc.NewService(v, t, p)
	return wbsvc.NewService(p, v, t, f, a, visitSvc, nil, "http", nil)
}

// eventCollector collects SSE events for callback-based tests.
type eventCollector struct {
	events []model.AssistantStreamEvent
}

func (ec *eventCollector) callback(event model.AssistantStreamEvent) error {
	ec.events = append(ec.events, event)
	return nil
}

// ============================================================
//  service.go
// ============================================================

func TestGetSession(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	session, err := svc.GetSession(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
}

func TestGetSession_NotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.GetSession(ctx, "bad-id")
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestListTimeline(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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

func TestListTimeline_Empty(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mt.listFunc = func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
		return nil, nil, false, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	items, _, _, err := svc.ListTimeline(ctx, "s1", nil, 50)
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("items = %d, want 0", len(items))
	}
}

// ============================================================
//  chat.go — SendMessage
// ============================================================

func TestSendMessage(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SendMessage(ctx, wbsvc.SendMessageInput{
		SessionID: "bad-id", Content: "test", ClientMessageID: "m1",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSendMessage_TimelineAppendFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mt.appendFunc = func(ctx context.Context, item *model.TimelineItem) error {
		return fmt.Errorf("db error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SendMessage(ctx, wbsvc.SendMessageInput{
		SessionID: "s1", Content: "test", ClientMessageID: "m1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "append patient message: db error" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendMessage_PlaceholderAppendFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	appendCount := 0
	mt.appendFunc = func(ctx context.Context, item *model.TimelineItem) error {
		appendCount++
		if appendCount == 2 {
			return fmt.Errorf("db error")
		}
		return nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SendMessage(ctx, wbsvc.SendMessageInput{
		SessionID: "s1", Content: "test", ClientMessageID: "m1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "append placeholder: db error" {
		t.Errorf("unexpected error: %v", err)
	}
}

// ============================================================
//  lab.go — SubmitLabDecision & SubmitLabResults
// ============================================================

func TestSubmitLabDecision_Accepted(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid decision")
	}
}

func TestSubmitLabDecision_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "bad-id", CardID: "f1", Decision: "accepted",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSubmitLabDecision_CardNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return nil, model.ErrCardNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "bad-card", Decision: "accepted",
	})
	if err != model.ErrCardNotFound {
		t.Errorf("expected ErrCardNotFound, got %v", err)
	}
}

func TestSubmitLabDecision_Accepted_FlowCardCreateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.createFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("create error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "accepted",
	})
	if err == nil {
		t.Fatal("expected error for card creation failure")
	}
}

func TestSubmitLabResults(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	appended := false
	mt.appendFunc = func(ctx context.Context, item *model.TimelineItem) error {
		if item.Kind == "system_event" && item.EventType == string(model.SystemEventTypeLabResultReceived) {
			appended = true
		}
		return nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	err := svc.SubmitLabResults(ctx, wbsvc.SubmitLabResultsInput{
		SessionID: "s1",
		Results: []struct {
			Item  string
			Value string
		}{{Item: "白细胞", Value: "11.2"}},
	})
	if err != nil {
		t.Fatalf("SubmitLabResults: %v", err)
	}
	if !appended {
		t.Error("expected timeline append for lab results")
	}
}

func TestSubmitLabResults_Empty(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	appended := false
	mt.appendFunc = func(ctx context.Context, item *model.TimelineItem) error {
		if item.Kind == "system_event" {
			appended = true
		}
		return nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	err := svc.SubmitLabResults(ctx, wbsvc.SubmitLabResultsInput{
		SessionID: "s1",
	})
	if err != nil {
		t.Fatalf("SubmitLabResults (empty): %v", err)
	}
	if !appended {
		t.Error("expected timeline append for empty results")
	}
}

func TestSubmitLabDecision_Accepted_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return makeCard(id, "s1", "lab_decision", true), nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "accepted",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on accepted")
	}
}

func TestSubmitLabDecision_Accepted_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return makeCard(id, "s1", "lab_decision", true), nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "accepted",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails on accepted")
	}
}

func TestSubmitLabDecision_Skipped_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return makeCard(id, "s1", "lab_decision", true), nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "skipped",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on skipped")
	}
}

func TestSubmitLabDecision_Skipped_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return makeCard(id, "s1", "lab_decision", true), nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "skipped",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails on skipped")
	}
}

func TestSubmitLabDecision_Vetoed_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return makeCard(id, "s1", "lab_decision", true), nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "vetoed",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on vetoed")
	}
}

func TestSubmitLabDecision_Vetoed_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return makeCard(id, "s1", "lab_decision", true), nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitLabDecision(ctx, wbsvc.SubmitLabDecisionInput{
		SessionID: "s1", CardID: "f1", Decision: "vetoed",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails on vetoed")
	}
}

// ============================================================
//  payment.go — SubmitPayment
// ============================================================

func TestSubmitPayment_Defer(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(100.0)
		return card, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(50.0)
		card.Purpose = "lab"
		return card, nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error { return nil }
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(100.0)
		card.Purpose = "medication"
		return card, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
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

func TestSubmitPayment_CardNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return nil, model.ErrCardNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "bad-card", Purpose: "lab",
	})
	if err != model.ErrCardNotFound {
		t.Errorf("expected ErrCardNotFound, got %v", err)
	}
}

// TestSubmitPayment_Lab_WithMedAgent verifies that after lab payment,
// test results are fed back to medAgent and the returned DONE step
// produces diagnosis + treatment plan cards.
func TestSubmitPayment_Lab_WithMedAgent(t *testing.T) {
	medAgentCalled := false
	svc, _, mv, mt, mf, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case containsStr(path, "/test-results"):
			medAgentCalled = true
			j := `{"kind":"DONE","result":{"final":"ADVICE","plan":"ADVICE_ONLY","diagnosis":{"name":"感冒","basis":"血常规确认感染","confidence":0.9},"advice":"多休息，多喝水"}}`
			return 200, j
		default:
			return 404, `{"error":"not found"}`
		}
	})

	// Set up session with medAgent session ID
	maSessID := "ma-test-001"
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.ID = id
		s.MedAgentSessionID = &maSessID
		return s, nil
	}

	// Set up payment card
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(50.0)
		card.Purpose = "lab"
		return card, nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error { return nil }

	// Track created cards and timeline items
	var createdCardKinds []string
	var timelineKinds []string
	mf.createFunc = func(ctx context.Context, card *model.FlowCard) error {
		createdCardKinds = append(createdCardKinds, card.Kind)
		return nil
	}
	mt.appendFunc = func(ctx context.Context, item *model.TimelineItem) error {
		timelineKinds = append(timelineKinds, item.Kind)
		return nil
	}

	ctx := context.Background()
	result, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab",
	})
	if err != nil {
		t.Fatalf("SubmitPayment: %v", err)
	}

	if !medAgentCalled {
		t.Error("medAgent TestResults was not called — agent loop is still broken")
	}
	if result.Message != "检验费支付成功，诊断结果已出" {
		t.Errorf("message = %s, want 检验费支付成功，诊断结果已出", result.Message)
	}

	// ADVICE_ONLY plan should produce: lab_execution, diagnosis, treatment_plan, advice_only
	foundLabExec := false
	foundDiagnosis := false
	foundTreatment := false
	foundAdvice := false
	for _, k := range createdCardKinds {
		switch k {
		case "lab_execution":
			foundLabExec = true
		case "diagnosis":
			foundDiagnosis = true
		case "treatment_plan":
			foundTreatment = true
		case "advice_only":
			foundAdvice = true
		}
	}
	if !foundLabExec {
		t.Error("expected lab_execution card")
	}
	if !foundDiagnosis {
		t.Error("expected diagnosis card")
	}
	if !foundTreatment {
		t.Error("expected treatment_plan card")
	}
	if !foundAdvice {
		t.Error("expected advice_only card")
	}

	// Should have lab_result_received timeline item
	foundResult := false
	for _, k := range timelineKinds {
		if k == "system_event" {
			foundResult = true
		}
	}
	if !foundResult {
		t.Error("expected system_event timeline items")
	}

	// Session should be in advice_only state (blocked with active card)
	if result.Status != "blocked" {
		t.Errorf("status = %s, want blocked (ADVICE_ONLY plan)", result.Status)
	}
}

// TestSubmitPayment_Lab_WithMedAgent_Done_Medication verifies the MEDICATION plan path.
func TestSubmitPayment_Lab_WithMedAgent_Medication(t *testing.T) {
	svc, _, mv, _, mf, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		if containsStr(path, "/test-results") {
			j := `{"kind":"DONE","result":{"final":"MEDICATION","plan":"MEDICATION","diagnosis":{"name":"细菌感染","basis":"血常规白细胞升高","confidence":0.85},"advice":"需要使用抗生素"}}`
			return 200, j
		}
		return 404, `{"error":"not found"}`
	})

	maSessID := "ma-test-002"
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.ID = id
		s.MedAgentSessionID = &maSessID
		return s, nil
	}
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(50.0)
		card.Purpose = "lab"
		return card, nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error { return nil }

	ctx := context.Background()
	result, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab",
	})
	if err != nil {
		t.Fatalf("SubmitPayment: %v", err)
	}

	// MEDICATION plan → session should be in diagnosis state
	if result.Status != "diagnosis" {
		t.Errorf("status = %s, want diagnosis (MEDICATION plan)", result.Status)
	}
	if result.Message != "检验费支付成功，诊断结果已出" {
		t.Errorf("message = %s", result.Message)
	}
}

// TestSubmitPayment_Lab_WithMedAgent_TestResultsFails verifies graceful handling
// when the medAgent TestResults call fails.
func TestSubmitPayment_Lab_WithMedAgent_TestResultsFails(t *testing.T) {
	svc, _, mv, _, mf, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		return 500, `{"error":"internal error"}`
	})

	maSessID := "ma-test-003"
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.ID = id
		s.MedAgentSessionID = &maSessID
		return s, nil
	}
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(50.0)
		card.Purpose = "lab"
		return card, nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error { return nil }

	ctx := context.Background()
	result, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab",
	})
	if err != nil {
		t.Fatalf("SubmitPayment should not fail when TestResults errors: %v", err)
	}
	// Should still complete payment successfully, falling back to diagnosis state.
	if result.Status != "diagnosis" {
		t.Errorf("status = %s, want diagnosis (fallback)", result.Status)
	}
	if result.Message != "检验费支付成功，诊断结果已出" {
		t.Errorf("message = %s", result.Message)
	}
}

func TestSubmitPayment_Lab_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(50.0)
		card.Purpose = "lab"
		return card, nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	// Visit update failure should now be propagated as an error.
	_, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails")
	}
}

func TestSubmitPayment_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(50.0)
		card.Purpose = "lab"
		return card, nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	// Flow card update failure should be propagated as an error.
	_, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails")
	}
}

func TestSubmitPayment_Defer_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "payment", true)
		card.TotalAmount = model.Float64Ptr(100.0)
		return card, nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	// Deferred payment card update failure should be propagated as an error.
	_, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "s1", CardID: "f1", Purpose: "lab", Defer: true,
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on defer")
	}
}

// ============================================================
//  fulfillment.go — SubmitFulfillment
// ============================================================

func TestSubmitFulfillment_Pickup(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "f1", Mode: "pickup",
	})
	if err != nil {
		t.Fatalf("SubmitFulfillment: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("status = %s, want completed", result.Status)
	}
	if len(result.TimelineItems) != 3 {
		t.Errorf("expected 3 timeline items, got %d", len(result.TimelineItems))
	}
}

func TestSubmitFulfillment_Delivery(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "f1", Mode: "delivery", AddressID: "addr-1",
	})
	if err != nil {
		t.Fatalf("SubmitFulfillment: %v", err)
	}
	if result.Message != "已确认配送到家，就诊完成" {
		t.Errorf("message = %s", result.Message)
	}
}

func TestSubmitFulfillment_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "bad-id", CardID: "f1", Mode: "pickup",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSubmitFulfillment_CardNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return nil, model.ErrCardNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "bad-card", Mode: "pickup",
	})
	if err != model.ErrCardNotFound {
		t.Errorf("expected ErrCardNotFound, got %v", err)
	}
}

// ============================================================
//  treatment.go — SubmitTreatmentExecution & AckAdvice
// ============================================================

func TestSubmitTreatmentExecution_Schedule(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "schedule",
	})
	if err != nil {
		t.Fatalf("SubmitTreatmentExecution: %v", err)
	}
	if result.Message != "治疗已预约" {
		t.Errorf("message = %s, want 治疗已预约", result.Message)
	}
}

func TestSubmitTreatmentExecution_ConfirmArrival(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "confirm_arrival",
	})
	if err != nil {
		t.Fatalf("SubmitTreatmentExecution: %v", err)
	}
	if result.Message != "已确认到达" {
		t.Errorf("message = %s, want 已确认到达", result.Message)
	}
}

func TestSubmitTreatmentExecution_Start(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "start",
	})
	if err != nil {
		t.Fatalf("SubmitTreatmentExecution: %v", err)
	}
	if result.Message != "治疗开始" {
		t.Errorf("message = %s, want 治疗开始", result.Message)
	}
}

func TestSubmitTreatmentExecution_Complete(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "complete",
	})
	if err != nil {
		t.Fatalf("SubmitTreatmentExecution: %v", err)
	}
	if result.Message != "治疗完成，就诊结束" {
		t.Errorf("message = %s, want 治疗完成，就诊结束", result.Message)
	}
}

func TestSubmitTreatmentExecution_Cancel(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "cancel",
	})
	if err != nil {
		t.Fatalf("SubmitTreatmentExecution: %v", err)
	}
	if result.Message != "治疗已取消" {
		t.Errorf("message = %s, want 治疗已取消", result.Message)
	}
}

func TestSubmitTreatmentExecution_InvalidAction(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid action")
	}
}

func TestSubmitTreatmentExecution_CardNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return nil, model.ErrCardNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "bad-card", Action: "schedule",
	})
	if err != model.ErrCardNotFound {
		t.Errorf("expected ErrCardNotFound, got %v", err)
	}
}

func TestSubmitTreatmentExecution_Schedule_UpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "schedule",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on schedule")
	}
}

func TestSubmitTreatmentExecution_Complete_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "complete",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on complete")
	}
}

func TestSubmitTreatmentExecution_Complete_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "complete",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails on complete")
	}
}

func TestSubmitTreatmentExecution_Complete_FlowCardCreateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.createFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("create error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "complete",
	})
	if err == nil {
		t.Fatal("expected error when flow card create fails on complete")
	}
}

func TestSubmitTreatmentExecution_Cancel_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "cancel",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails on cancel")
	}
}

func TestSubmitTreatmentExecution_Cancel_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "s1", CardID: "f1", Action: "cancel",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails on cancel")
	}
}

func TestAckAdvice(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.AckAdvice(ctx, wbsvc.AckAdviceInput{
		SessionID: "s1", CardID: "f1",
	})
	if err != nil {
		t.Fatalf("AckAdvice: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("status = %s, want completed", result.Status)
	}
	if result.Message != "医嘱已确认，就诊完成" {
		t.Errorf("message = %s, want 医嘱已确认，就诊完成", result.Message)
	}
}

func TestAckAdvice_CardNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return nil, model.ErrCardNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.AckAdvice(ctx, wbsvc.AckAdviceInput{
		SessionID: "s1", CardID: "bad-card",
	})
	if err != model.ErrCardNotFound {
		t.Errorf("expected ErrCardNotFound, got %v", err)
	}
}

func TestAckAdvice_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.AckAdvice(ctx, wbsvc.AckAdviceInput{
		SessionID: "s1", CardID: "f1",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails")
	}
}

func TestAckAdvice_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.AckAdvice(ctx, wbsvc.AckAdviceInput{
		SessionID: "s1", CardID: "f1",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails")
	}
}

func TestAckAdvice_FlowCardCreateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.createFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("create error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.AckAdvice(ctx, wbsvc.AckAdviceInput{
		SessionID: "s1", CardID: "f1",
	})
	if err == nil {
		t.Fatal("expected error when flow card create fails")
	}
}

// ============================================================
//  exit.go — ExitVisit
// ============================================================

func TestExitVisit_NoFee(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "chatting"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.TerminalReason != "patient_request" {
		t.Errorf("terminalReason = %s, want patient_request", result.TerminalReason)
	}
	if result.Consequence == nil || result.Consequence.Kind != "no_fee" {
		t.Error("expected no_fee consequence for chatting exit")
	}
}

func TestExitVisit_Refundable(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "blocked"
		diag := "感冒"
		s.Summary.Diagnosis = &diag
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
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

func TestExitVisit_ExecutedNoRefund(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "diagnosis"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.Consequence == nil || result.Consequence.Kind != "executed_no_refund" {
		t.Errorf("consequence = %v, want executed_no_refund", result.Consequence)
	}
}

func TestExitVisit_MedicationDispensed(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "completed"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.Consequence == nil || result.Consequence.Kind != "medication_dispensed" {
		t.Errorf("consequence = %v, want medication_dispensed", result.Consequence)
	}
}

func TestExitVisit_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "bad-id", Reason: "patient_request",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestExitVisit_BlockedNoDiagnosis(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "blocked"
		// No diagnosis set -> should fall through to no_fee within blocked case
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.Consequence == nil || result.Consequence.Kind != "no_fee" {
		t.Errorf("consequence = %v, want no_fee", result.Consequence)
	}
}

func TestExitVisit_DefaultCase(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "transferred"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	result, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "patient_request",
	})
	if err != nil {
		t.Fatalf("ExitVisit: %v", err)
	}
	if result.Consequence == nil || result.Consequence.Kind != "no_fee" {
		t.Errorf("consequence = %v, want no_fee (default)", result.Consequence)
	}
}

func TestExitVisit_UpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.ExitVisit(ctx, model.ExitVisitInput{SessionID: "s1", Reason: "patient_request"})
	if err == nil {
		t.Fatal("expected error when visit update fails")
	}
}

func TestSubmitFulfillment_Pickup_FlowCardUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "medication_fulfillment", true)
		return card, nil
	}
	mf.updateFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "f1", Mode: "pickup",
	})
	if err == nil {
		t.Fatal("expected error when flow card update fails")
	}
}

func TestSubmitFulfillment_Pickup_VisitUpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "medication_fulfillment", true)
		return card, nil
	}
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "f1", Mode: "pickup",
	})
	if err == nil {
		t.Fatal("expected error when visit update fails")
	}
}

func TestSubmitFulfillment_Pickup_FlowCardCreateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		card := makeCard(id, "s1", "medication_fulfillment", true)
		return card, nil
	}
	mf.createFunc = func(ctx context.Context, card *model.FlowCard) error {
		return fmt.Errorf("create error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "f1", Mode: "pickup",
	})
	if err == nil {
		t.Fatal("expected error when flow card create fails")
	}
}

// ============================================================
//  vitals.go — ReportVitals & DismissEmergency
// ============================================================

func TestReportVitals_Normal(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	hr := 72
	spo2 := 98.0
	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "patient_report",
		Vitals: &model.VitalsData{
			HeartRate: &hr,
			SpO2:      &spo2,
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	hr := 150
	spo2 := 85.0
	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "device",
		Vitals: &model.VitalsData{
			HeartRate: &hr,
			SpO2:      &spo2,
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

func TestReportVitals_HighTemp(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	updateCalled := false
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		updateCalled = true
		return nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	temp := 42.0
	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "device",
		Vitals: &model.VitalsData{
			Temperature: &temp,
		},
	})
	if err != nil {
		t.Fatalf("ReportVitals: %v", err)
	}
	if !result.Emergency {
		t.Error("high temp should trigger emergency")
	}
	if result.Severity != "suspected" {
		t.Errorf("severity = %s, want suspected", result.Severity)
	}
	if updateCalled {
		t.Error("visit update should not be called for suspected severity")
	}
}

func TestReportVitals_LowTemp(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	temp := 34.5
	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "device",
		Vitals: &model.VitalsData{
			Temperature: &temp,
		},
	})
	if err != nil {
		t.Fatalf("ReportVitals: %v", err)
	}
	if !result.Emergency {
		t.Error("low temp should trigger emergency")
	}
	if result.Severity != "suspected" {
		t.Errorf("severity = %s, want suspected", result.Severity)
	}
}

func TestReportVitals_LowSPO2_NonCritical(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	// spo2 91 is < 90 = false, so no emergency; exercises the spo2 code path.
	spo2 := 91.0
	result, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Source:    "device",
		Vitals: &model.VitalsData{
			SpO2: &spo2,
		},
	})
	if err != nil {
		t.Fatalf("ReportVitals: %v", err)
	}
	if result.Emergency {
		t.Error("spo2 91 should not trigger emergency")
	}
	if result.Message != "体征正常" {
		t.Errorf("message = %s, want 体征正常", result.Message)
	}
}

func TestDismissEmergency(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Status = "emergency_terminated"
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, _, err := svc.DismissEmergency(ctx, wbsvc.DismissEmergencyInput{
		SessionID: "s1",
	})
	if err != model.ErrValidation {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestDismissEmergency_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, _, err := svc.DismissEmergency(ctx, wbsvc.DismissEmergencyInput{
		SessionID: "bad-id",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestDismissEmergency_UpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, _, err := svc.DismissEmergency(ctx, wbsvc.DismissEmergencyInput{SessionID: "s1"})
	if err == nil {
		t.Fatal("expected error when visit update fails on dismiss emergency")
	}
}

func TestReportVitals_Critical_UpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	hr := 150
	_, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Vitals: &model.VitalsData{
			HeartRate: &hr,
		},
		Symptoms: []string{"胸痛"},
	})
	if err == nil {
		t.Fatal("expected error when visit update fails on critical vitals")
	}
}

func TestReportVitals_Critical_FindSessionFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	// FindByID returns error — session termination cannot proceed, but vitals detection succeeds
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, fmt.Errorf("db error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	hr := 150
	_, err := svc.ReportVitals(ctx, wbsvc.ReportVitalsInput{
		SessionID: "s1",
		Vitals: &model.VitalsData{
			HeartRate: &hr,
		},
		Symptoms: []string{"胸痛"},
	})
	// HR 150 + 胸痛 triggers critical severity, but FindByID fails —
	// the function should log a warning and return the emergency detection result without error.
	if err != nil {
		t.Fatalf("expected no error when FindSession fails during emergency detection, got: %v", err)
	}
}

// ============================================================
//  consult.go — ClassifyIntent, StreamConsultationReply, AskLockedQuestion
// ============================================================

func TestClassifyIntent_FollowUp(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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

func TestClassifyIntent_Uncertain(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	// Content with no matching keywords should return uncertain per spec.
	result, err := svc.ClassifyIntent(ctx, wbsvc.ClassifyIntentInput{
		SessionID: "s1",
		Content:   "我没什么特别想说的",
	})
	if err != nil {
		t.Fatalf("ClassifyIntent: %v", err)
	}
	if result.Intent != "uncertain" {
		t.Errorf("intent = %s, want uncertain", result.Intent)
	}
	if result.Confidence > 0.5 {
		t.Errorf("confidence = %f, want <= 0.5", result.Confidence)
	}
}

func TestStreamConsultationReply(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	// Session with diagnosis
	diag := "感冒"
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		s := makeSession("p001")
		s.Summary.Diagnosis = &diag
		return s, nil
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	ec := &eventCollector{}
	err := svc.StreamConsultationReply(ctx, "s1", "我需要注意什么", "req-1", ec.callback)
	if err != nil {
		t.Fatalf("StreamConsultationReply: %v", err)
	}
	if len(ec.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(ec.events))
	}
	if ec.events[0].Type != "delta" {
		t.Errorf("event[0] type = %s, want delta", ec.events[0].Type)
	}
	if ec.events[1].Type != "message_final" {
		t.Errorf("event[1] type = %s, want message_final", ec.events[1].Type)
	}
	if ec.events[2].Type != "done" {
		t.Errorf("event[2] type = %s, want done", ec.events[2].Type)
	}
}

func TestStreamConsultationReply_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	err := svc.StreamConsultationReply(ctx, "bad-id", "test", "req-1", func(evt model.AssistantStreamEvent) error {
		return nil
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestAskLockedQuestion(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	ec := &eventCollector{}
	err := svc.AskLockedQuestion(ctx, "s1", "f1", "这是什么检查", "req-1", ec.callback)
	if err != nil {
		t.Fatalf("AskLockedQuestion: %v", err)
	}
	if len(ec.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(ec.events))
	}
	if ec.events[0].Type != "delta" {
		t.Errorf("event[0] type = %s, want delta", ec.events[0].Type)
	}
	if ec.events[1].Type != "message_final" {
		t.Errorf("event[1] type = %s, want message_final", ec.events[1].Type)
	}
	if ec.events[2].Type != "done" {
		t.Errorf("event[2] type = %s, want done", ec.events[2].Type)
	}
}

func TestAskLockedQuestion_CardNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mf.findByIDFunc = func(ctx context.Context, id string) (*model.FlowCard, error) {
		return nil, model.ErrCardNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	err := svc.AskLockedQuestion(ctx, "s1", "bad-card", "这是什么检查", "req-1", func(evt model.AssistantStreamEvent) error {
		return nil
	})
	if err != model.ErrCardNotFound {
		t.Errorf("expected ErrCardNotFound, got %v", err)
	}
}

// ============================================================
//  timer.go — PauseTimer & ResumeTimer
// ============================================================

func TestPauseTimer(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
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
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	session, err := svc.ResumeTimer(ctx, "s1")
	if err != nil {
		t.Fatalf("ResumeTimer: %v", err)
	}
	if session.TimerPaused {
		t.Error("TimerPaused should be false")
	}
}

func TestPauseTimer_NotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.PauseTimer(ctx, "bad-id")
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestResumeTimer_NotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.ResumeTimer(ctx, "bad-id")
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestPauseTimer_UpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.PauseTimer(ctx, "s1")
	if err == nil {
		t.Fatal("expected error when visit update fails")
	}
}

func TestResumeTimer_UpdateFails(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.updateFunc = func(ctx context.Context, vs *model.VisitSession) error {
		return fmt.Errorf("update error")
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.ResumeTimer(ctx, "s1")
	if err == nil {
		t.Fatal("expected error when visit update fails")
	}
}

// ============================================================
//  chat.go — StreamAssistantMessage & handler functions
// ============================================================

// newMedAgentTestServer starts an httptest server that mimics medAgent API.
func newMedAgentTestServer(handler func(method, path string, body []byte) (int, string)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
		}
		status, resp := handler(r.Method, r.URL.Path, reqBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
}

func newSvcWithMockMedAgent(t *testing.T, medAgentHandler func(method, path string, body []byte) (int, string)) (*wbsvc.Service, *mockPatientRepo, *mockVisitRepo, *mockTimelineRepo, *mockFlowCardRepo, *mockAddressRepo) {
	t.Helper()
	srv := newMedAgentTestServer(medAgentHandler)
	t.Cleanup(srv.Close)
	client := medagent.NewClient(srv.URL)
	mp := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID:        id,
				Name:      "测试",
				Gender:    "male",
				Age:       30,
				Allergies: []string{},
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	mv := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			s := makeSession("p001")
			s.ID = id
			return s, nil
		},
		updateFunc: func(ctx context.Context, vs *model.VisitSession) error { return nil },
	}
	mt := &mockTimelineRepo{
		appendFunc:      func(ctx context.Context, item *model.TimelineItem) error { return nil },
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error { return nil },
		listFunc: func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{{ID: "t1", Role: "patient", Content: "我头痛", Kind: "message"}}, nil, false, nil
		},
	}
	mf := &mockFlowCardRepo{
		createFunc: func(ctx context.Context, card *model.FlowCard) error { return nil },
		listFunc:   func(ctx context.Context, sid string) ([]model.FlowCard, error) { return nil, nil },
	}
	ma := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return &model.Address{
				ID: id, PatientID: "p001", Name: "李明", Phone: "13800002468",
				Province: "辽宁省", City: "沈阳市", District: "浑南区", Detail: "创新路195号",
			}, nil
		},
		createFunc: func(ctx context.Context, addr *model.Address) error { return nil },
		updateFunc: func(ctx context.Context, addr *model.Address) error { return nil },
		deleteFunc: func(ctx context.Context, id string) error { return nil },
	}
	visitSvc := visitsvc.NewService(mv, mt, mp)
	return wbsvc.NewService(mp, mv, mt, mf, ma, visitSvc, client, "http", nil), mp, mv, mt, mf, ma
}

func TestStreamAssistantMessage_Ask(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"ASK","doctor_say":"请问您发烧多久了？"}`
		default:
			return 404, `{"error":"not found"}`
		}
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage: %v", err)
	}
	if len(ec.events) < 4 {
		t.Fatalf("expected >= 4 events, got %d", len(ec.events))
	}
	// Should have delta, message_final, state, done
	types := make(map[string]bool)
	for _, e := range ec.events {
		types[e.Type] = true
	}
	for _, want := range []string{"delta", "message_final", "state", "done"} {
		if !types[want] {
			t.Errorf("missing event type: %s", want)
		}
	}
}

func TestStreamAssistantMessage_Emergency(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"EMERGENCY","emergency":"血压过高，建议立即转急诊"}`
		default:
			return 404, `{"error":"not found"}`
		}
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage: %v", err)
	}
	hasEmergency := false
	for _, e := range ec.events {
		if e.Type == "emergency" {
			hasEmergency = true
		}
	}
	if !hasEmergency {
		t.Error("expected emergency event")
	}
}

func TestStreamAssistantMessage_NeedTests(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"NEED_TESTS","doctor_say":"需要检查血常规","test_items":["血常规"]}`
		default:
			return 404, `{"error":"not found"}`
		}
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage: %v", err)
	}
	cardCount := 0
	hasState := false
	for _, e := range ec.events {
		if e.Type == "card" {
			cardCount++
		}
		if e.Type == "state" && e.Status == "blocked" {
			hasState = true
		}
	}
	if cardCount == 0 {
		t.Error("expected card event for NEED_TESTS")
	}
	if cardCount > 1 {
		t.Errorf("expected exactly 1 card event, got %d — duplicate card event bug", cardCount)
	}
	if !hasState {
		t.Error("expected blocked state event")
	}
}

func TestStreamAssistantMessage_NeedTests_Idempotent(t *testing.T) {
	// When a pending lab_decision card already exists, a second NEED_TESTS step
	// must NOT create a duplicate card.
	cardCreated := false
	svc, _, _, _, mf, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"NEED_TESTS","doctor_say":"需要检查血常规","test_items":["血常规"]}`
		default:
			return 404, `{"error":"not found"}`
		}
	})
	// Override: ListBySession returns an existing pending lab_decision card
	mf.listFunc = func(ctx context.Context, sid string) ([]model.FlowCard, error) {
		return []model.FlowCard{{
			ID:        "existing-card",
			SessionID: sid,
			Kind:      "lab_decision",
			Status:    "pending",
		}}, nil
	}
	mf.createFunc = func(ctx context.Context, card *model.FlowCard) error {
		cardCreated = true
		return nil
	}

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage: %v", err)
	}
	if cardCreated {
		t.Error("duplicate lab_decision card was created — idempotency guard failed")
	}
}

func TestStreamAssistantMessage_DrugQuery(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"DRUG_QUERY","drug_names":["布洛芬缓释胶囊"]}`
		case containsStr(path, "/drug-info"):
			return 200, `{"kind":"PURCHASE","orders":[{"name":"布洛芬缓释胶囊","quantity":2}]}`
		default:
			return 404, `{"error":"not found"}`
		}
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage: %v", err)
	}
	// DrugQuery -> auto drug-info -> Purchase -> card event
	hasCard := false
	for _, e := range ec.events {
		if e.Type == "card" {
			hasCard = true
		}
	}
	if !hasCard {
		t.Error("expected card event after drug query + purchase chain")
	}
}

func TestStreamAssistantMessage_Done(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			j := `{"kind":"DONE","result":{"final":"ADVICE","plan":"ADVICE_ONLY","diagnosis":{"name":"感冒","basis":"症状","confidence":0.9},"advice":"多休息"}}`
			return 200, j
		default:
			return 404, `{"error":"not found"}`
		}
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage: %v", err)
	}
	hasDone := false
	for _, e := range ec.events {
		if e.Type == "done" {
			hasDone = true
		}
	}
	if !hasDone {
		t.Error("expected done event")
	}
}

func TestStreamAssistantMessage_PatientNotFound(t *testing.T) {
	svc, mp, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		return 200, `{"session_id":"ma-001"}`
	})
	mp.findByIDFunc = func(ctx context.Context, id string) (*model.PatientProfile, error) {
		return nil, model.ErrPatientNotFound
	}

	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, func(e model.AssistantStreamEvent) error { return nil })
	if err == nil {
		t.Error("expected error for patient not found")
	}
}

func TestStreamAssistantMessage_NoPatientMessage(t *testing.T) {
	svc, _, _, mt, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"ASK","doctor_say":"请描述您的症状"}`
		default:
			return 404, `{"error":"not found"}`
		}
	})
	// Override timeline to return empty (no patient messages)
	mt.listFunc = func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
		return []model.TimelineItem{}, nil, false, nil
	}

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage (empty timeline): %v", err)
	}
	// Should default to "你好" when no patient message found
	found := false
	for _, e := range ec.events {
		if e.Type == "delta" {
			found = true
		}
	}
	if !found {
		t.Error("expected delta event even with empty timeline")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestStreamAssistantMessage_CreateSessionFailure tests the error path when
// medAgent session creation fails.
func TestStreamAssistantMessage_CreateSessionFailure(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		if path == "/sessions" {
			return 500, `{"error":"internal server error"}`
		}
		return 404, `{"error":"not found"}`
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err == nil {
		t.Fatal("expected error when medAgent session creation fails")
	}
	if !containsStr(err.Error(), "medagent session") && !containsStr(err.Error(), "create") {
		t.Logf("error message (informational): %v", err)
	}
}

// TestStreamAssistantMessage_OK tests the StepOK handling.
func TestStreamAssistantMessage_OK(t *testing.T) {
	svc, _, _, _, _, _ := newSvcWithMockMedAgent(t, func(method, path string, body []byte) (int, string) {
		switch {
		case path == "/sessions":
			return 200, `{"session_id":"ma-001"}`
		case containsStr(path, "/patient-say"):
			return 200, `{"kind":"OK"}`
		default:
			return 404, `{"error":"not found"}`
		}
	})

	ec := &eventCollector{}
	err := svc.StreamAssistantMessage(context.Background(), wbsvc.StreamAssistantInput{
		SessionID: "s1", RequestID: "r1",
	}, ec.callback)
	if err != nil {
		t.Fatalf("StreamAssistantMessage OK: %v", err)
	}
	// OK step should only emit done (no state per A15 fix)
	foundDone := false
	for _, e := range ec.events {
		if e.Type == "state" {
			t.Error("OK step should not emit state event")
		}
		if e.Type == "done" {
			foundDone = true
		}
	}
	if !foundDone {
		t.Error("OK step should emit done event")
	}
}

func TestSubmitPayment_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitPayment(ctx, model.SubmitPaymentInput{
		SessionID: "bad-id", CardID: "f1", Purpose: "lab",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSubmitTreatmentExecution_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitTreatmentExecution(ctx, wbsvc.SubmitTreatmentExecutionInput{
		SessionID: "bad-id", CardID: "f1", Action: "schedule",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestAckAdvice_SessionNotFound(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	mv.findByIDFunc = func(ctx context.Context, id string) (*model.VisitSession, error) {
		return nil, model.ErrSessionNotFound
	}
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.AckAdvice(ctx, wbsvc.AckAdviceInput{
		SessionID: "bad-id", CardID: "f1",
	})
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestExitVisit_InvalidReason(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.ExitVisit(ctx, model.ExitVisitInput{
		SessionID: "s1", Reason: "invalid_reason",
	})
	if err == nil {
		t.Fatal("expected error for invalid exit reason")
	}
}

func TestSubmitFulfillment_Delivery_EmptyAddress(t *testing.T) {
	mp, mv, mt, mf, ma := newDefaultMocks()
	svc := newSvc(mp, mv, mt, mf, ma)
	ctx := context.Background()

	_, err := svc.SubmitFulfillment(ctx, wbsvc.SubmitFulfillmentInput{
		SessionID: "s1", CardID: "f1", Mode: "delivery", AddressID: "",
	})
	if err != model.ErrAddressRequired {
		t.Errorf("expected ErrAddressRequired, got %v", err)
	}
}

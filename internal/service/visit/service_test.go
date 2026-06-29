package visit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/service/visit"
)

type mockVisitRepo struct {
	createFunc        func(ctx context.Context, v *model.VisitSession) error
	findByIDFunc      func(ctx context.Context, id string) (*model.VisitSession, error)
	listByPatientFunc func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
	updateStatusFunc  func(ctx context.Context, id string, status string, machineState string) error
	updateFunc        func(ctx context.Context, v *model.VisitSession) error
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error {
	return m.createFunc(ctx, v)
}
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, patientID, cursor, pageSize)
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id string, status string, machineState string) error {
	return m.updateStatusFunc(ctx, id, status, machineState)
}
func (m *mockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error {
	return m.updateFunc(ctx, v)
}

type mockTimelineRepo struct {
	appendFunc      func(ctx context.Context, item *model.TimelineItem) error
	appendBatchFunc func(ctx context.Context, items []model.TimelineItem) error
	listFunc        func(ctx context.Context, sessionID string, cursor *string, pageSize int) ([]model.TimelineItem, *string, bool, error)
}

func (m *mockTimelineRepo) Append(ctx context.Context, item *model.TimelineItem) error {
	return m.appendFunc(ctx, item)
}
func (m *mockTimelineRepo) AppendBatch(ctx context.Context, items []model.TimelineItem) error {
	return m.appendBatchFunc(ctx, items)
}
func (m *mockTimelineRepo) ListBySession(ctx context.Context, sessionID string, cursor *string, pageSize int) ([]model.TimelineItem, *string, bool, error) {
	return m.listFunc(ctx, sessionID, cursor, pageSize)
}
func (m *mockTimelineRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	return nil
}

var allMachineStates = []string{
	string(model.VisitMachineStateLoadingContext),
	string(model.VisitMachineStateChatting),
	string(model.VisitMachineStateAnalyzing),
	string(model.VisitMachineStateLabDecision),
	string(model.VisitMachineStateLabPayment),
	string(model.VisitMachineStateLabExecution),
	string(model.VisitMachineStateDiagnosis),
	string(model.VisitMachineStateTreatmentDecision),
	string(model.VisitMachineStateMedicationPayment),
	string(model.VisitMachineStateMedicationFulfillment),
	string(model.VisitMachineStateTreatmentExecution),
	string(model.VisitMachineStateAdviceOnly),
	string(model.VisitMachineStateEmergencyPending),
	string(model.VisitMachineStateCompleted),
	string(model.VisitMachineStateTerminated),
	string(model.VisitMachineStateExitSettlement),
	string(model.VisitMachineStateExited),
}

func strPtr(s string) *string { return &s }

// --- Existing Service Tests ---

func TestCreateSession(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error {
			return nil
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateSessionInput{
		PatientID:      "p001",
		EntryType:      "new",
		ChiefComplaint: "头痛",
	}

	result, err := svc.CreateSession(ctx, input)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if result.Session.PatientID != "p001" {
		t.Errorf("patientId = %s", result.Session.PatientID)
	}
	if result.Session.EntryType != "new" {
		t.Errorf("entryType = %s", result.Session.EntryType)
	}
	if len(result.InitialTimeline) < 2 {
		t.Errorf("initialTimeline = %d items, want >= 2", len(result.InitialTimeline))
	}
}

func TestCreateSession_Invalid(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	// Invalid entry type
	input := model.CreateSessionInput{
		PatientID: "p001",
		EntryType: "follow_up",
	}

	_, err := svc.CreateSession(ctx, input)
	if err == nil {
		t.Error("expected error for invalid entry type")
	}
}

func TestCreateSession_EmptyPatientID(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error {
			return nil
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateSessionInput{
		PatientID:      "",
		EntryType:      "new",
		ChiefComplaint: "头痛",
	}

	result, err := svc.CreateSession(ctx, input)
	if err != nil {
		t.Fatalf("CreateSession with empty PatientID: %v", err)
	}
	if result.Session.PatientID != "" {
		t.Errorf("patientID = %q, want empty", result.Session.PatientID)
	}
}

func TestCreateFollowUp(t *testing.T) {
	ctx := context.Background()

	parentSession := &model.VisitSession{
		ID:        "v001",
		PatientID: "p001",
		EntryType: "new",
		Status:    "completed",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Summary: model.VisitSummary{
			ChiefComplaint: strPtr("头痛"),
			Diagnosis:      strPtr("感冒"),
		},
	}

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return parentSession, nil
		},
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error {
			return nil
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateFollowUpInput{
		PatientID:       "p001",
		ParentSessionID: "v001",
		ChiefComplaint:  "仍头痛",
	}

	result, err := svc.CreateFollowUp(ctx, input)
	if err != nil {
		t.Fatalf("CreateFollowUp: %v", err)
	}
	if result.Session.EntryType != "follow_up" {
		t.Errorf("entryType = %s", result.Session.EntryType)
	}
}

func TestGetSession(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID:        id,
				PatientID: "p001",
				Status:    "chatting",
				StartedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	session, err := svc.GetSession(ctx, "v001")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if session.ID != "v001" {
		t.Errorf("id = %s", session.ID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return nil, model.ErrSessionNotFound
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	_, err := svc.GetSession(ctx, "nonexistent")
	if err != model.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestGetSnapshot_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	session := &model.VisitSession{
		ID:        "v001",
		PatientID: "p001",
		Status:    "chatting",
		StartedAt: now,
		UpdatedAt: now,
	}
	timeline := []model.TimelineItem{
		{ID: "t1", SessionID: "v001", Kind: "message", Content: "hello"},
	}

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return session, nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		listFunc: func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
			return timeline, nil, false, nil
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	snapshot, err := svc.GetSnapshot(ctx, "v001")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.Session.ID != "v001" {
		t.Errorf("session id = %s", snapshot.Session.ID)
	}
	if len(snapshot.Timeline) != 1 {
		t.Errorf("timeline = %d items, want 1", len(snapshot.Timeline))
	}
	if !snapshot.Readonly {
		t.Error("snapshot should be readonly")
	}
}

func TestListSessions(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "v001", PatientID: patientID},
			}, nil, false, nil
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	items, _, hasMore, err := svc.ListSessions(ctx, "p001", nil, 20)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("items = %d", len(items))
	}
	if hasMore {
		t.Error("hasMore should be false")
	}
}

func TestListSessions_WithCursor(t *testing.T) {
	ctx := context.Background()
	cursor := "next_cursor"

	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			items := []model.VisitSessionSummary{
				{ID: "v002", PatientID: pid},
				{ID: "v003", PatientID: pid},
			}
			next := "cursor_v2"
			return items, &next, true, nil
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	items, nextCursor, hasMore, err := svc.ListSessions(ctx, "p001", &cursor, 2)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}
	if !hasMore {
		t.Error("hasMore should be true")
	}
	if nextCursor == nil || *nextCursor != "cursor_v2" {
		t.Errorf("nextCursor = %v, want cursor_v2", nextCursor)
	}
}

// --- New Service Tests ---

func TestCreateSession_WithComplaint(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error {
			return nil
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateSessionInput{
		PatientID:      "p001",
		EntryType:      "new",
		ChiefComplaint: "头痛",
	}

	result, err := svc.CreateSession(ctx, input)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if result.Session.Summary.ChiefComplaint == nil {
		t.Fatal("ChiefComplaint should be set in summary")
	}
	if *result.Session.Summary.ChiefComplaint != "头痛" {
		t.Errorf("ChiefComplaint = %q, want %q", *result.Session.Summary.ChiefComplaint, "头痛")
	}
}

func TestCreateSession_TimelineError(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error {
			return errors.New("timeline append failed")
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateSessionInput{
		PatientID:      "p001",
		EntryType:      "new",
		ChiefComplaint: "头痛",
	}

	_, err := svc.CreateSession(ctx, input)
	if err == nil {
		t.Error("expected error from timeline append failure")
	}
}

func TestCreateSession_CreateError(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return errors.New("create failed")
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateSessionInput{
		PatientID:      "p001",
		EntryType:      "new",
		ChiefComplaint: "头痛",
	}

	_, err := svc.CreateSession(ctx, input)
	if err == nil {
		t.Error("expected error from create failure")
	}
}

func TestCreateFollowUp_ParentNotFound(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return nil, model.ErrSessionNotFound
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateFollowUpInput{
		PatientID:       "p001",
		ParentSessionID: "nonexistent",
	}

	_, err := svc.CreateFollowUp(ctx, input)
	if err == nil {
		t.Fatal("expected error for parent not found")
	}
}

func TestCreateFollowUp_TimelineError(t *testing.T) {
	ctx := context.Background()

	parentSession := &model.VisitSession{
		ID:        "v001",
		PatientID: "p001",
		EntryType: "new",
		Status:    "completed",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Summary: model.VisitSummary{
			ChiefComplaint: strPtr("头痛"),
		},
	}

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return parentSession, nil
		},
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		appendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error {
			return errors.New("timeline append failed")
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateFollowUpInput{
		PatientID:       "p001",
		ParentSessionID: "v001",
		ChiefComplaint:  "仍头痛",
	}

	_, err := svc.CreateFollowUp(ctx, input)
	if err == nil {
		t.Error("expected error from timeline append failure")
	}
}

func TestCreateFollowUp_CreateError(t *testing.T) {
	ctx := context.Background()

	parentSession := &model.VisitSession{
		ID:        "v001",
		PatientID: "p001",
		EntryType: "new",
		Status:    "completed",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Summary: model.VisitSummary{
			ChiefComplaint: strPtr("头痛"),
		},
	}

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return parentSession, nil
		},
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return errors.New("create failed")
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	input := model.CreateFollowUpInput{
		PatientID:       "p001",
		ParentSessionID: "v001",
		ChiefComplaint:  "仍头痛",
	}

	_, err := svc.CreateFollowUp(ctx, input)
	if err == nil {
		t.Error("expected error from create failure")
	}
}

func TestListSessions_Empty(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{}, nil, false, nil
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	items, _, hasMore, err := svc.ListSessions(ctx, "p001", nil, 20)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("items = %d, want 0", len(items))
	}
	if hasMore {
		t.Error("hasMore should be false")
	}
}

func TestGetSnapshot_NotFound(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return nil, model.ErrSessionNotFound
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	_, err := svc.GetSnapshot(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestGetSnapshot_TimelineError(t *testing.T) {
	ctx := context.Background()

	session := &model.VisitSession{
		ID:        "v001",
		PatientID: "p001",
		Status:    "chatting",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return session, nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		listFunc: func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
			return nil, nil, false, errors.New("timeline list failed")
		},
	}

	svc := visit.NewService(visitRepo, timelineRepo)

	_, err := svc.GetSnapshot(ctx, "v001")
	if err == nil {
		t.Error("expected error from timeline failure")
	}
}

func TestUpdateStatus_Success(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID:        id,
				PatientID: "p001",
				Status:    "chatting",
				StartedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		updateStatusFunc: func(ctx context.Context, id, status, machineState string) error {
			return nil
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	err := svc.UpdateStatus(ctx, "v001", "analyzing")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return nil, model.ErrSessionNotFound
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	err := svc.UpdateStatus(ctx, "nonexistent", "analyzing")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestUpdateStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID:        id,
				PatientID: "p001",
				Status:    "chatting",
				StartedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	timelineRepo := &mockTimelineRepo{}

	svc := visit.NewService(visitRepo, timelineRepo)

	// chatting -> completed is not a valid transition
	err := svc.UpdateStatus(ctx, "v001", "completed")
	if err == nil {
		t.Error("expected error for invalid transition")
	}
}

// --- Existing State Machine Tests ---

func TestAllTransitions(t *testing.T) {
	for from, targets := range visit.AllowedTransitions {
		for _, to := range targets {
			t.Run(from+"->"+to, func(t *testing.T) {
				if !visit.CanTransition(from, to) {
					t.Errorf("CanTransition(%q, %q) = false, want true", from, to)
				}
			})
		}
	}
}

func TestTerminalStatesReject(t *testing.T) {
	terminalStates := []string{"completed", "terminated", "exited"}
	for _, state := range terminalStates {
		t.Run(state, func(t *testing.T) {
			if visit.CanTransition(state, "chatting") {
				t.Errorf("CanTransition(%q, chatting) = true, want false", state)
			}
			if !visit.IsTerminal(state) {
				t.Errorf("IsTerminal(%q) = false, want true", state)
			}
		})
	}
}

func TestStateMachine(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		valid   bool
	}{
		{"chatting to analyzing", "chatting", "analyzing", true},
		{"chatting to labDecision", "chatting", "labDecision", true},
		{"completed to anything", "completed", "chatting", false},
		{"terminated to anything", "terminated", "chatting", false},
		{"invalid from", "invalid_state", "chatting", false},
		{"invalid to", "chatting", "invalid_state", false},
		{"labDecision to labPayment", "labDecision", "labPayment", true},
		{"labDecision to chatting (veto)", "labDecision", "chatting", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visit.CanTransition(tt.current, tt.next)
			if result != tt.valid {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.current, tt.next, result, tt.valid)
			}
		})
	}
}

// --- Enhanced State Machine Tests ---

// TestIsTerminal verifies terminal detection for all machine states.
func TestIsTerminal(t *testing.T) {
	tests := []struct {
		state    string
		terminal bool
	}{
		{string(model.VisitMachineStateCompleted), true},
		{string(model.VisitMachineStateTerminated), true},
		{string(model.VisitMachineStateExited), true},
		{string(model.VisitMachineStateLoadingContext), false},
		{string(model.VisitMachineStateChatting), false},
		{string(model.VisitMachineStateAnalyzing), false},
		{string(model.VisitMachineStateLabDecision), false},
		{string(model.VisitMachineStateLabPayment), false},
		{string(model.VisitMachineStateLabExecution), false},
		{string(model.VisitMachineStateDiagnosis), false},
		{string(model.VisitMachineStateTreatmentDecision), false},
		{string(model.VisitMachineStateMedicationPayment), false},
		{string(model.VisitMachineStateMedicationFulfillment), false},
		{string(model.VisitMachineStateTreatmentExecution), false},
		{string(model.VisitMachineStateAdviceOnly), false},
		{string(model.VisitMachineStateEmergencyPending), false},
		{string(model.VisitMachineStateExitSettlement), false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := visit.IsTerminal(tt.state)
			if got != tt.terminal {
				t.Errorf("IsTerminal(%q) = %v, want %v", tt.state, got, tt.terminal)
			}
		})
	}
}

// TestGetStatusForState verifies status mappings including the default for unknown states.
func TestGetStatusForState(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{string(model.VisitMachineStateLoadingContext), string(model.VisitStatusLoadingContext)},
		{string(model.VisitMachineStateChatting), string(model.VisitStatusChatting)},
		{string(model.VisitMachineStateAnalyzing), string(model.VisitStatusAnalyzing)},
		{string(model.VisitMachineStateLabDecision), string(model.VisitStatusBlocked)},
		{string(model.VisitMachineStateLabPayment), string(model.VisitStatusBlocked)},
		{string(model.VisitMachineStateLabExecution), string(model.VisitStatusDiagnosis)},
		{string(model.VisitMachineStateDiagnosis), string(model.VisitStatusDiagnosis)},
		{string(model.VisitMachineStateTreatmentDecision), string(model.VisitStatusTreatment)},
		{string(model.VisitMachineStateMedicationPayment), string(model.VisitStatusBlocked)},
		{string(model.VisitMachineStateMedicationFulfillment), string(model.VisitStatusBlocked)},
		{string(model.VisitMachineStateTreatmentExecution), string(model.VisitStatusTreatment)},
		{string(model.VisitMachineStateAdviceOnly), string(model.VisitStatusBlocked)},
		{string(model.VisitMachineStateCompleted), string(model.VisitStatusCompleted)},
		{string(model.VisitMachineStateEmergencyPending), string(model.VisitStatusEmergencyTerminated)},
		{string(model.VisitMachineStateTerminated), string(model.VisitStatusEmergencyTerminated)},
		{string(model.VisitMachineStateExitSettlement), string(model.VisitStatusExited)},
		{string(model.VisitMachineStateExited), string(model.VisitStatusExited)},
		{"unknown", string(model.VisitStatusChatting)},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := visit.GetStatusForState(tt.state)
			if got != tt.expected {
				t.Errorf("GetStatusForState(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

// TestGetStatusForState_All preserves the existing table-driven test for common mappings.
func TestGetStatusForState_All(t *testing.T) {
	cases := []struct {
		state    string
		expected string
	}{
		{"chatting", "chatting"},
		{"labDecision", "blocked"},
		{"completed", "completed"},
	}
	for _, c := range cases {
		t.Run(c.state, func(t *testing.T) {
			got := visit.GetStatusForState(c.state)
			if got != c.expected {
				t.Errorf("GetStatusForState(%q) = %q, want %q", c.state, got, c.expected)
			}
		})
	}
}

// --- New State Machine Tests ---

// TestInvalidTransitions verifies that for each state, transitions not in AllowedTransitions
// are correctly rejected by CanTransition.
func TestInvalidTransitions(t *testing.T) {
	for from, allowed := range visit.AllowedTransitions {
		t.Run(from, func(t *testing.T) {
			allowedSet := make(map[string]bool, len(allowed))
			for _, a := range allowed {
				allowedSet[a] = true
			}
			for _, to := range allMachineStates {
				if !allowedSet[to] {
					if visit.CanTransition(from, to) {
						t.Errorf("CanTransition(%q, %q) = true, want false", from, to)
					}
				}
			}
		})
	}
}

// TestTransition_Success verifies that every valid transition returns the new state without error.
func TestTransition_Success(t *testing.T) {
	for from, targets := range visit.AllowedTransitions {
		for _, to := range targets {
			t.Run(from+"_to_"+to, func(t *testing.T) {
				result, err := visit.Transition(from, to)
				if err != nil {
					t.Errorf("Transition(%q, %q) unexpected error: %v", from, to, err)
				}
				if result != to {
					t.Errorf("Transition(%q, %q) = %q, want %q", from, to, result, to)
				}
			})
		}
	}
}

// TestTransition_Failure verifies that an invalid transition returns an error.
func TestTransition_Failure(t *testing.T) {
	_, err := visit.Transition("chatting", "completed")
	if err == nil {
		t.Error("Transition(chatting, completed) expected error, got nil")
	}
}

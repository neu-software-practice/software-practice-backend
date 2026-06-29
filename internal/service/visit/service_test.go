package visit_test

import (
	"context"
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

func TestIsTerminal(t *testing.T) {
	if !visit.IsTerminal("completed") {
		t.Error("completed should be terminal")
	}
	if !visit.IsTerminal("terminated") {
		t.Error("terminated should be terminal")
	}
	if !visit.IsTerminal("exited") {
		t.Error("exited should be terminal")
	}
	if visit.IsTerminal("chatting") {
		t.Error("chatting should not be terminal")
	}
}

func TestGetStatusForState(t *testing.T) {
	if visit.GetStatusForState("chatting") != "chatting" {
		t.Error("chatting mapping mismatch")
	}
	if visit.GetStatusForState("completed") != "completed" {
		t.Error("completed mapping mismatch")
	}
	if visit.GetStatusForState("labDecision") != "blocked" {
		t.Error("labDecision should map to blocked")
	}
}

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

func strPtr(s string) *string { return &s }

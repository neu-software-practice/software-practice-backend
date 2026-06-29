package visit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

// Service handles visit session business logic.
type Service struct {
	visitRepo    repository.VisitRepository
	timelineRepo repository.TimelineRepository
}

// NewService creates a new VisitService.
func NewService(visitRepo repository.VisitRepository, timelineRepo repository.TimelineRepository) *Service {
	return &Service{
		visitRepo:    visitRepo,
		timelineRepo: timelineRepo,
	}
}

// CreateSession creates a new visit session.
func (s *Service) CreateSession(ctx context.Context, input model.CreateSessionInput) (*model.CreateSessionResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	sessionID := uuid.New().String()
	now := time.Now()

	session := &model.VisitSession{
		ID:            sessionID,
		PatientID:     input.PatientID,
		EntryType:     string(model.VisitEntryTypeNew),
		Status:        string(model.VisitStatusLoadingContext),
		AskRoundLimit: 20,
		LabRoundLimit: 10,
		Summary:       model.VisitSummary{},
	}

	if input.ChiefComplaint != "" {
		cc := input.ChiefComplaint
		session.Summary.ChiefComplaint = &cc
	}

	if err := s.visitRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create visit session: %w", err)
	}

	initialTimeline := adapter.BuildInitialTimeline(sessionID, input.ChiefComplaint)
	if len(initialTimeline) > 0 {
		if err := s.timelineRepo.AppendBatch(ctx, initialTimeline); err != nil {
			return nil, fmt.Errorf("append initial timeline: %w", err)
		}
	}

	// After context loaded, transition to chatting
	session.Status = string(model.VisitStatusChatting)
	session.UpdatedAt = now

	return &model.CreateSessionResult{
		Session:         *session,
		InitialTimeline: initialTimeline,
	}, nil
}

// CreateFollowUp creates a follow-up visit session from a parent session.
func (s *Service) CreateFollowUp(ctx context.Context, input model.CreateFollowUpInput) (*model.CreateSessionResult, error) {
	// Validate parent session exists
	parent, err := s.visitRepo.FindByID(ctx, input.ParentSessionID)
	if err != nil {
		return nil, fmt.Errorf("parent session: %w", err)
	}

	sessionID := uuid.New().String()
	now := time.Now()

	session := &model.VisitSession{
		ID:              sessionID,
		PatientID:       input.PatientID,
		EntryType:       string(model.VisitEntryTypeFollowUp),
		Status:          string(model.VisitStatusLoadingContext),
		ParentSessionID: &input.ParentSessionID,
		AskRoundLimit:   20,
		LabRoundLimit:   10,
		Summary: model.VisitSummary{
			ChiefComplaint:   parent.Summary.ChiefComplaint,
			Diagnosis:        parent.Summary.Diagnosis,
			TreatmentSummary: parent.Summary.TreatmentSummary,
		},
	}

	if input.ChiefComplaint != "" {
		cc := input.ChiefComplaint
		session.Summary.ChiefComplaint = &cc
	}

	if err := s.visitRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create follow-up session: %w", err)
	}

	initialTimeline := adapter.BuildInitialTimeline(sessionID, input.ChiefComplaint)
	// Add follow-up started event
	followUpEvent := adapter.BuildSystemEventTimelineItem(
		sessionID,
		string(model.SystemEventTypeFollowUpStarted),
		"复诊开始",
		fmt.Sprintf("基于上次就诊 %s 创建复诊", input.ParentSessionID),
	)
	initialTimeline = append(initialTimeline, followUpEvent)

	if err := s.timelineRepo.AppendBatch(ctx, initialTimeline); err != nil {
		return nil, fmt.Errorf("append initial timeline: %w", err)
	}

	session.Status = string(model.VisitStatusChatting)
	session.UpdatedAt = now

	return &model.CreateSessionResult{
		Session:         *session,
		InitialTimeline: initialTimeline,
	}, nil
}

// GetSession retrieves a visit session by ID.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*model.VisitSession, error) {
	return s.visitRepo.FindByID(ctx, sessionID)
}

// ListSessions lists visit sessions for a patient with cursor pagination.
func (s *Service) ListSessions(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	return s.visitRepo.ListByPatient(ctx, patientID, cursor, pageSize)
}

// GetSnapshot returns a read-only snapshot of a visit including its full timeline.
func (s *Service) GetSnapshot(ctx context.Context, sessionID string) (*model.VisitSnapshot, error) {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get all timeline items (no pagination for snapshot)
	timeline, _, _, err := s.timelineRepo.ListBySession(ctx, sessionID, nil, 1000)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}

	return &model.VisitSnapshot{
		Session:        *session,
		Timeline:       timeline,
		Readonly:       true,
		TerminalReason: session.TerminalReason,
	}, nil
}

// UpdateStatus updates the session status and machine state.
func (s *Service) UpdateStatus(ctx context.Context, sessionID, newMachineState string) error {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}

	currentState := session.Status // Use status as machine state proxy
	if _, err := Transition(currentState, newMachineState); err != nil {
		return fmt.Errorf("state transition: %w", err)
	}

	status := GetStatusForState(newMachineState)
	return s.visitRepo.UpdateStatus(ctx, sessionID, status, newMachineState)
}

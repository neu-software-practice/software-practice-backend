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

// SuspendVisitResult is the response for the suspend visit endpoint.
type SuspendVisitResult struct {
	Session      model.VisitSession `json:"session"`
	TimelineItem model.TimelineItem `json:"timelineItem"`
}

// createSessionParams holds the parameters for creating a new or follow-up session.
type createSessionParams struct {
	PatientID       string
	ChiefComplaint  string
	EntryType       string
	ParentSessionID *string
	ParentSummary   *model.VisitSummary
}

// createSession is the shared session-creation helper used by CreateSession and CreateFollowUp.
// It creates the session in DB, appends the initial timeline, and transitions to chatting.
func (s *Service) createSession(ctx context.Context, params createSessionParams) (*model.CreateSessionResult, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	session := &model.VisitSession{
		ID:              sessionID,
		PatientID:       params.PatientID,
		EntryType:       params.EntryType,
		Status:          string(model.VisitStatusLoadingContext),
		MachineState:    string(model.VisitMachineStateLoadingContext),
		ParentSessionID: params.ParentSessionID,
		AskRoundLimit:   20,
		LabRoundLimit:   10,
		LastActivityAt:  &now,
		Summary:         model.VisitSummary{},
	}

	if params.ParentSummary != nil {
		session.Summary.ChiefComplaint = params.ParentSummary.ChiefComplaint
		session.Summary.Diagnosis = params.ParentSummary.Diagnosis
		session.Summary.TreatmentSummary = params.ParentSummary.TreatmentSummary
	}

	if params.ChiefComplaint != "" {
		cc := params.ChiefComplaint
		session.Summary.ChiefComplaint = &cc
	}

	if err := s.visitRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create visit session: %w", err)
	}

	initialTimeline := adapter.BuildInitialTimeline(sessionID, params.ChiefComplaint)
	if err := s.timelineRepo.AppendBatch(ctx, initialTimeline); err != nil {
		return nil, fmt.Errorf("append initial timeline: %w", err)
	}

	// After context loaded, transition to chatting
	session.Status = string(model.VisitStatusChatting)
	session.MachineState = string(model.VisitMachineStateChatting)
	session.UpdatedAt = now

	if err := s.visitRepo.UpdateStatus(ctx, sessionID, session.Status, session.MachineState); err != nil {
		return nil, fmt.Errorf("update session status to chatting: %w", err)
	}

	return &model.CreateSessionResult{
		Session:         *session,
		InitialTimeline: initialTimeline,
	}, nil
}

// CreateSession creates a new visit session.
func (s *Service) CreateSession(ctx context.Context, input model.CreateSessionInput) (*model.CreateSessionResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return s.createSession(ctx, createSessionParams{
		PatientID:      input.PatientID,
		ChiefComplaint: input.ChiefComplaint,
		EntryType:      string(model.VisitEntryTypeNew),
	})
}

// CreateFollowUp creates a follow-up visit session from a parent session.
func (s *Service) CreateFollowUp(ctx context.Context, input model.CreateFollowUpInput) (*model.CreateSessionResult, error) {
	parent, err := s.visitRepo.FindByID(ctx, input.ParentSessionID)
	if err != nil {
		return nil, fmt.Errorf("parent session: %w", err)
	}

	result, err := s.createSession(ctx, createSessionParams{
		PatientID:       input.PatientID,
		ChiefComplaint:  input.ChiefComplaint,
		EntryType:       string(model.VisitEntryTypeFollowUp),
		ParentSessionID: &input.ParentSessionID,
		ParentSummary:   &parent.Summary,
	})
	if err != nil {
		return nil, err
	}

	// Append follow-up started event to the new session's timeline
	followUpEvent := adapter.BuildSystemEventTimelineItem(
		result.Session.ID,
		string(model.SystemEventTypeFollowUpStarted),
		"复诊开始",
		fmt.Sprintf("基于上次就诊 %s 创建复诊", input.ParentSessionID),
	)
	if err := s.timelineRepo.Append(ctx, &followUpEvent); err != nil {
		return nil, fmt.Errorf("append follow-up event: %w", err)
	}
	result.InitialTimeline = append(result.InitialTimeline, followUpEvent)

	return result, nil
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

	currentState := session.MachineState
	if _, err := Transition(currentState, newMachineState); err != nil {
		return fmt.Errorf("state transition: %w", err)
	}

	status := GetStatusForState(newMachineState)
	return s.visitRepo.UpdateStatus(ctx, sessionID, status, newMachineState)
}

// SuspendVisit suspends a visit session due to idle timeout.
// It is not a terminal state — endedAt and terminalReason are not set.
func (s *Service) SuspendVisit(ctx context.Context, sessionID string) (*SuspendVisitResult, error) {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Verify the session can transition to suspended state
	if _, err := Transition(session.MachineState, string(model.VisitMachineStateSuspended)); err != nil {
		return nil, fmt.Errorf("%w: %s", model.ErrInvalidState, err.Error())
	}

	now := time.Now()

	// If there is an active streaming assistant message, mark it as idle-interrupted
	streamingMsg, err := s.timelineRepo.FindLastStreamingMessage(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("find streaming message: %w", err)
	}
	if streamingMsg != nil {
		idle := string(model.InterruptedByIdle)
		streamingMsg.InterruptedBy = &idle
		streamingMsg.Status = string(model.TimelineItemStatusDone)
		if err := s.timelineRepo.UpdateContent(ctx, streamingMsg.ID, streamingMsg); err != nil {
			return nil, fmt.Errorf("update streaming message: %w", err)
		}
	}

	// Clear the active card ID and transition to suspended
	session.ActiveCardID = nil
	session.Status = GetStatusForState(string(model.VisitMachineStateSuspended))
	session.MachineState = string(model.VisitMachineStateSuspended)
	session.UpdatedAt = now
	// NOTE: LastActivityAt is NOT refreshed here — suspend is a result of inactivity

	if err := s.visitRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Create system event for timeline
	suspendEvent := adapter.BuildSystemEventTimelineItem(
		sessionID,
		string(model.SystemEventTypeSessionSuspended),
		"会话已暂停",
		"会话因空闲超时已暂停，患者可直接输入或按复诊流程继续",
	)
	if err := s.timelineRepo.Append(ctx, &suspendEvent); err != nil {
		return nil, fmt.Errorf("append suspend event: %w", err)
	}

	return &SuspendVisitResult{
		Session:      *session,
		TimelineItem: suspendEvent,
	}, nil
}

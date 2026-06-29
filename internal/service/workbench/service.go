package workbench

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// Service orchestrates all workbench operations (chat, lab, payment, etc.).
type Service struct {
	patientRepo    repository.PatientRepository
	visitRepo      repository.VisitRepository
	timelineRepo   repository.TimelineRepository
	flowCardRepo   repository.FlowCardRepository
	medAgentClient *medagent.Client
	medAgentMode   string
}

// NewService creates a new WorkbenchService.
func NewService(
	patientRepo repository.PatientRepository,
	visitRepo repository.VisitRepository,
	timelineRepo repository.TimelineRepository,
	flowCardRepo repository.FlowCardRepository,
	medAgentClient *medagent.Client,
	medAgentMode string,
) *Service {
	return &Service{
		patientRepo:    patientRepo,
		visitRepo:      visitRepo,
		timelineRepo:   timelineRepo,
		flowCardRepo:   flowCardRepo,
		medAgentClient: medAgentClient,
		medAgentMode:   medAgentMode,
	}
}

// GetSession retrieves a visit session by ID.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*model.VisitSession, error) {
	return s.visitRepo.FindByID(ctx, sessionID)
}

// ListTimeline lists timeline items for a session with cursor pagination.
func (s *Service) ListTimeline(ctx context.Context, sessionID string, cursor *string, pageSize int) ([]model.TimelineItem, *string, bool, error) {
	return s.timelineRepo.ListBySession(ctx, sessionID, cursor, pageSize)
}

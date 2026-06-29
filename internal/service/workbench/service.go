package workbench

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// LLMClient defines the interface for LLM text completion used by title generation.
type LLMClient interface {
	ChatComplete(ctx context.Context, system, user string) (string, error)
}

// Service orchestrates all workbench operations (chat, lab, payment, etc.).
type Service struct {
	patientRepo    repository.PatientRepository
	visitRepo      repository.VisitRepository
	timelineRepo   repository.TimelineRepository
	flowCardRepo   repository.FlowCardRepository
	medAgentClient *medagent.Client
	medAgentMode   string
	llmClient      LLMClient
}

// NewService creates a new WorkbenchService.
func NewService(
	patientRepo repository.PatientRepository,
	visitRepo repository.VisitRepository,
	timelineRepo repository.TimelineRepository,
	flowCardRepo repository.FlowCardRepository,
	medAgentClient *medagent.Client,
	medAgentMode string,
	llmClient LLMClient,
) *Service {
	return &Service{
		patientRepo:    patientRepo,
		visitRepo:      visitRepo,
		timelineRepo:   timelineRepo,
		flowCardRepo:   flowCardRepo,
		medAgentClient: medAgentClient,
		medAgentMode:   medAgentMode,
		llmClient:      llmClient,
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

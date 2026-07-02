package workbench

import (
	"context"
	"log/slog"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
	"github.com/neuhis/software-practice-backend/internal/service/visit"
)

// LLMClient defines the interface for LLM text completion used by title generation.
type LLMClient interface {
	ChatComplete(ctx context.Context, system, user string) (string, error)
}

// medAgentClient defines the interface for medAgent interactions used by the workbench.
type medAgentClient interface {
	CreateSession(ctx context.Context, profile map[string]interface{}, initial bool, prior []interface{}) (string, error)
	PatientSay(ctx context.Context, sessionID string, message string) (*medagent.Step, error)
	DrugInfo(ctx context.Context, sessionID string, infos []medagent.DrugInfo) (*medagent.Step, error)
	TestResults(ctx context.Context, sessionID string, results []medagent.TestResult) (*medagent.Step, error)
}

// Service orchestrates all workbench operations (chat, lab, payment, etc.).
type Service struct {
	patientRepo  repository.PatientRepository
	visitRepo    repository.VisitRepository
	timelineRepo repository.TimelineRepository
	flowCardRepo repository.FlowCardRepository
	addressRepo  repository.AddressRepository
	drugRepo     repository.DrugRepository
	visitSvc     *visit.Service
	maClient     medAgentClient
	medAgentMode string
	llmClient    LLMClient
}

// NewService creates a new WorkbenchService.
func NewService(
	patientRepo repository.PatientRepository,
	visitRepo repository.VisitRepository,
	timelineRepo repository.TimelineRepository,
	flowCardRepo repository.FlowCardRepository,
	addressRepo repository.AddressRepository,
	visitSvc *visit.Service,
	maClient medAgentClient,
	medAgentMode string,
	llmClient LLMClient,
	drugRepos ...repository.DrugRepository,
) *Service {
	var drugRepo repository.DrugRepository
	if len(drugRepos) > 0 {
		drugRepo = drugRepos[0]
	}
	return &Service{
		patientRepo:  patientRepo,
		visitRepo:    visitRepo,
		timelineRepo: timelineRepo,
		flowCardRepo: flowCardRepo,
		addressRepo:  addressRepo,
		drugRepo:     drugRepo,
		visitSvc:     visitSvc,
		maClient:     maClient,
		medAgentMode: medAgentMode,
		llmClient:    llmClient,
	}
}

// GetSession retrieves a visit session by ID.
// Delegates to visitSvc when available; falls back to visitRepo for backward compatibility.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*model.VisitSession, error) {
	if s.visitSvc != nil {
		return s.visitSvc.GetSession(ctx, sessionID)
	}
	return s.visitRepo.FindByID(ctx, sessionID)
}

// ListTimeline lists timeline items for a session with cursor pagination.
func (s *Service) ListTimeline(ctx context.Context, sessionID string, cursor *string, pageSize int) ([]model.TimelineItem, *string, bool, error) {
	return s.timelineRepo.ListBySession(ctx, sessionID, cursor, pageSize)
}

// syncCardToTimeline updates the flow_card timeline item's embedded card snapshot
// after a flow card has been updated in the flow_cards table. This prevents the
// frontend from seeing stale card state when reading timeline items.
func (s *Service) syncCardToTimeline(ctx context.Context, card *model.FlowCard) {
	tlItem, err := s.timelineRepo.FindFlowCardByCardID(ctx, card.SessionID, card.ID)
	if err != nil {
		slog.Warn("failed to find timeline item for card sync", "card_id", card.ID, "error", err)
		return
	}
	if tlItem == nil {
		return
	}
	tlItem.Card = card
	if err := s.timelineRepo.UpdateContent(ctx, tlItem.ID, tlItem); err != nil {
		slog.Warn("failed to sync card to timeline", "card_id", card.ID, "error", err)
	}
}

func markCardProcessed(card *model.FlowCard, now time.Time) {
	card.Blocking = false
	card.HandledAt = &now
}

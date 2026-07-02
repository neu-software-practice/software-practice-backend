package medicalorder

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

// Service handles medical order records business logic.
type Service struct {
	visitRepo    repository.VisitRepository
	flowCardRepo repository.FlowCardRepository
}

// NewService creates a new MedicalOrderService.
func NewService(visitRepo repository.VisitRepository, flowCardRepo repository.FlowCardRepository) *Service {
	return &Service{
		visitRepo:    visitRepo,
		flowCardRepo: flowCardRepo,
	}
}

// ListMedicalOrders returns all medical order records for a patient, aggregated
// from completed advice_only and confirmed/completed medication_fulfillment
// flow cards across all their visit sessions.
func (s *Service) ListMedicalOrders(ctx context.Context, patientID string) (*model.MedicalOrdersResponse, error) {
	sessions, _, _, err := s.visitRepo.ListByPatient(ctx, patientID, "", nil, 1000)
	if err != nil {
		return nil, fmt.Errorf("list visits: %w", err)
	}

	var records []model.MedicalOrderRecord
	for _, session := range sessions {
		cards, err := s.flowCardRepo.ListBySession(ctx, session.ID)
		if err != nil {
			return nil, fmt.Errorf("list flow cards for session %s: %w", session.ID, err)
		}

		for _, card := range cards {
			record := s.buildRecordFromCard(session, card)
			if record != nil {
				records = append(records, *record)
			}
		}
	}

	// Sort by handledAt descending (newest first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].HandledAt > records[j].HandledAt
	})

	if records == nil {
		records = []model.MedicalOrderRecord{}
	}

	result := api.NewPageResult(records, nil, false)
	return &result, nil
}

// buildSessionTitle builds a human-readable title for a visit session.
func (s *Service) buildSessionTitle(summary model.VisitSummary) string {
	if summary.ChiefComplaint != nil && *summary.ChiefComplaint != "" {
		return *summary.ChiefComplaint
	}
	if summary.Diagnosis != nil && *summary.Diagnosis != "" {
		return *summary.Diagnosis
	}
	if summary.Title != nil && *summary.Title != "" {
		return *summary.Title
	}
	return "未知就诊"
}

// buildRecordFromCard converts a flow card to a MedicalOrderRecord, or returns nil
// if the card doesn't match the required kind/status criteria.
func (s *Service) buildRecordFromCard(session model.VisitSessionSummary, card model.FlowCard) *model.MedicalOrderRecord {
	base := model.MedicalOrderRecord{
		RecordID:     card.ID,
		SessionID:    card.SessionID,
		SessionTitle: s.buildSessionTitle(session.Summary),
		HandledAt:    s.resolveHandledAt(card).UTC().Format("2006-01-02T15:04:05.000Z"),
		CreatedAt:    card.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	switch card.Kind {
	case string(model.FlowCardKindAdviceOnly):
		if card.Status != string(model.FlowCardStatusCompleted) {
			return nil
		}
		base.Kind = string(model.MedicalOrderKindAdvice)
		base.Advices = copyStrings(card.Advices)
		base.WatchItems = copyStrings(card.WatchItems)
		base.FollowUpRecommendation = card.FollowUpRecommendation
		return &base

	case string(model.FlowCardKindMedicationFulfillment):
		if card.FulfillmentStatus != model.MedicationFulfillmentStatusCompleted &&
			card.FulfillmentStatus != model.MedicationFulfillmentStatusConfirmed {
			return nil
		}
		base.Kind = string(model.MedicalOrderKindMedication)
		base.Medications = copyMedications(card.Medications)
		base.FulfillmentStatus = model.FulfillmentStatus(card.FulfillmentStatus)
		base.DeliveryAddress = card.DeliveryAddress
		return &base
	}

	return nil
}

// resolveHandledAt returns the handledAt time, falling back to createdAt if nil.
func (s *Service) resolveHandledAt(card model.FlowCard) time.Time {
	if card.HandledAt != nil {
		return *card.HandledAt
	}
	return card.CreatedAt
}

// copyStrings returns a copy of the string slice (or nil) to avoid aliasing.
func copyStrings(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

// copyMedications returns a copy of the MedicationItem slice (or nil) to avoid aliasing.
func copyMedications(src []model.MedicationItem) []model.MedicationItem {
	if src == nil {
		return nil
	}
	dst := make([]model.MedicationItem, len(src))
	copy(dst, src)
	return dst
}

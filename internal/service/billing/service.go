package billing

import (
	"context"
	"fmt"
	"sort"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

// Service handles billing records business logic.
type Service struct {
	visitRepo    repository.VisitRepository
	flowCardRepo repository.FlowCardRepository
}

// NewService creates a new BillingService.
func NewService(visitRepo repository.VisitRepository, flowCardRepo repository.FlowCardRepository) *Service {
	return &Service{
		visitRepo:    visitRepo,
		flowCardRepo: flowCardRepo,
	}
}

// ListBillingRecords returns all billing records for a patient, aggregated
// from paid payment flow cards across all their visit sessions.
func (s *Service) ListBillingRecords(ctx context.Context, patientID string) (*model.BillingRecordsResponse, error) {
	// List all sessions for the patient (use a large page size to get everything)
	sessions, _, _, err := s.visitRepo.ListByPatient(ctx, patientID, nil, 1000)
	if err != nil {
		return nil, fmt.Errorf("list visits: %w", err)
	}

	var records []model.BillingRecord
	for _, session := range sessions {
		cards, err := s.flowCardRepo.ListBySession(ctx, session.ID)
		if err != nil {
			return nil, fmt.Errorf("list flow cards for session %s: %w", session.ID, err)
		}

		for _, card := range cards {
			// Only include paid payment cards
			if card.Kind != string(model.FlowCardKindPayment) {
				continue
			}
			if card.PaymentStatus != string(model.PaymentStatusPaid) {
				continue
			}

			// Build session title
			sessionTitle := ""
			if session.Summary.ChiefComplaint != nil && *session.Summary.ChiefComplaint != "" {
				sessionTitle = *session.Summary.ChiefComplaint
			} else if session.Summary.Diagnosis != nil && *session.Summary.Diagnosis != "" {
				sessionTitle = *session.Summary.Diagnosis
			} else if session.Summary.Title != nil && *session.Summary.Title != "" {
				sessionTitle = *session.Summary.Title
			} else {
				sessionTitle = "未知就诊"
			}

			var lineItems []model.BillingLineItem
			for _, item := range card.Items {
				qty := item.Quantity
				li := model.BillingLineItem{
					Name:   item.Name,
					Amount: item.Amount,
				}
				if qty > 0 {
					li.Quantity = &qty
				}
				lineItems = append(lineItems, li)
			}

			createdAt := ""
			if card.HandledAt != nil {
				createdAt = card.HandledAt.UTC().Format("2006-01-02T15:04:05.000Z")
			} else {
				createdAt = card.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z")
			}

			totalAmount := 0.0
			if card.TotalAmount != nil {
				totalAmount = *card.TotalAmount
			}
			insuranceAmount := 0.0
			if card.InsuranceAmount != nil {
				insuranceAmount = *card.InsuranceAmount
			}
			selfPayAmount := 0.0
			if card.SelfPayAmount != nil {
				selfPayAmount = *card.SelfPayAmount
			}
			records = append(records, model.BillingRecord{
				PaymentID:       card.PaymentID,
				SessionID:       card.SessionID,
				SessionTitle:    sessionTitle,
				Purpose:         card.Purpose,
				Items:           lineItems,
				TotalAmount:     totalAmount,
				InsuranceAmount: insuranceAmount,
				SelfPayAmount:   selfPayAmount,
				PaymentStatus:   card.PaymentStatus,
				CreatedAt:       createdAt,
			})
		}
	}

	// Sort by createdAt descending (newest first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})

	if records == nil {
		records = []model.BillingRecord{}
	}

	return &model.BillingRecordsResponse{Items: records}, nil
}

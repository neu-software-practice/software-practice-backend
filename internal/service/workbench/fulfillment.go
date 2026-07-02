package workbench

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// SubmitFulfillmentInput is the input for medication fulfillment.
type SubmitFulfillmentInput struct {
	SessionID string `json:"sessionId"`
	CardID    string `json:"cardId"`
	Mode      string `json:"mode"`                // pickup, delivery
	AddressID string `json:"addressId,omitempty"` // required when mode=delivery
}

// SubmitFulfillment processes medication fulfillment (pickup/delivery).
func (s *Service) SubmitFulfillment(ctx context.Context, input SubmitFulfillmentInput) (*model.FlowActionResult, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	card, err := s.flowCardRepo.FindByID(ctx, input.CardID)
	if err != nil {
		return nil, err
	}

	// Validate and resolve delivery address
	if input.Mode == string(model.FulfillmentModeDelivery) {
		if input.AddressID == "" {
			return nil, model.ErrAddressRequired
		}
		addr, err := s.addressRepo.FindByID(ctx, input.AddressID)
		if err != nil {
			return nil, err
		}
		if addr.PatientID != session.PatientID {
			return nil, model.ErrAddressNotFound
		}
		// Write address summary to the card
		card.DeliveryAddress = &model.DeliveryAddress{
			Name:        addr.Name,
			Phone:       addr.Phone,
			FullAddress: fmt.Sprintf("%s%s%s%s", addr.Province, addr.City, addr.District, addr.Detail),
		}
	}

	if err := s.decrementMedicationStock(ctx, card.Medications); err != nil {
		return nil, fmt.Errorf("decrement medication stock: %w", err)
	}

	now := time.Now()
	card.SelectedMode = &input.Mode
	card.FulfillmentStatus = model.MedicationFulfillmentStatusConfirmed
	card.Status = string(model.FlowCardStatusCompleted)
	markCardProcessed(card, now)
	if err := s.flowCardRepo.Update(ctx, card); err != nil {
		return nil, fmt.Errorf("update flow card on fulfillment: %w", err)
	}
	s.syncCardToTimeline(ctx, card)

	modeText := "到院取药"
	if input.Mode == "delivery" {
		modeText = "配送到家"
	}

	// Create drug purchased timeline event
	drugTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		string(model.SystemEventTypeDrugPurchased),
		"取药确认",
		fmt.Sprintf("已确认%s", modeText),
	)
	if err := s.timelineRepo.Append(ctx, &drugTL); err != nil {
		slog.Warn("failed to append drug purchased timeline", "session_id", input.SessionID, "error", err)
	}

	// Complete the session
	status := string(model.VisitStatusCompleted)
	reason := "completed"
	session.Status = status
	session.MachineState = string(model.VisitMachineStateCompleted)
	session.EndedAt = &now
	session.TerminalReason = &reason
	session.ActiveCardID = nil
	ts := "药物治疗完成"
	session.Summary.TreatmentSummary = &ts
	session.UpdatedAt = now
	session.LastActivityAt = &now
	if err := s.visitRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session after fulfillment: %w", err)
	}

	// Create completed visit card
	completedCard := &model.FlowCard{
		ID:               uuid.New().String(),
		SessionID:        input.SessionID,
		Kind:             string(model.FlowCardKindCompletedVisit),
		Status:           string(model.FlowCardStatusCompleted),
		Blocking:         false,
		Title:            "就诊完成",
		CreatedAt:        now,
		CompletedAt:      now,
		TreatmentSummary: "药物治疗完成",
	}
	if err := s.flowCardRepo.Create(ctx, completedCard); err != nil {
		return nil, fmt.Errorf("create completed visit card: %w", err)
	}

	completedTL := adapter.BuildFlowCardTimelineItem(input.SessionID, completedCard)
	if err := s.timelineRepo.Append(ctx, &completedTL); err != nil {
		slog.Warn("failed to append completed visit timeline", "session_id", input.SessionID, "error", err)
	}

	// Terminal timeline item
	termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
		"completed",
		"就诊完成",
		fmt.Sprintf("已%s，就诊结束", modeText),
	)
	if err := s.timelineRepo.Append(ctx, &termTL); err != nil {
		slog.Warn("failed to append terminal timeline", "session_id", input.SessionID, "error", err)
	}

	return &model.FlowActionResult{
		SessionID:     input.SessionID,
		Status:        status,
		Card:          card,
		TimelineItems: []model.TimelineItem{drugTL, completedTL, termTL},
		Message:       fmt.Sprintf("已确认%s，就诊完成", modeText),
	}, nil
}

func (s *Service) decrementMedicationStock(ctx context.Context, medications []model.MedicationItem) error {
	if s.drugRepo == nil || len(medications) == 0 {
		return nil
	}
	for _, medication := range medications {
		if medication.Quantity <= 0 {
			return fmt.Errorf("%w: drug quantity must be positive for %s", model.ErrValidation, medication.Name)
		}
		if err := s.drugRepo.DecrementStock(ctx, medication.Name, medication.Quantity); err != nil {
			return err
		}
	}
	return nil
}

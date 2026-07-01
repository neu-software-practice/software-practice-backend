package workbench

import (
	"context"
	"fmt"
	"time"

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

	now := time.Now()
	card.SelectedMode = &input.Mode
	card.FulfillmentStatus = "confirmed"
	card.Status = string(model.FlowCardStatusCompleted)
	card.HandledAt = &now
	_ = s.flowCardRepo.Update(ctx, card)

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
	_ = s.timelineRepo.Append(ctx, &drugTL)

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
	_ = s.visitRepo.Update(ctx, session)

	// Create completed visit card
	completedCard := &model.FlowCard{
		ID:               "",
		SessionID:        input.SessionID,
		Kind:             string(model.FlowCardKindCompletedVisit),
		Status:           string(model.FlowCardStatusCompleted),
		Blocking:         false,
		Title:            "就诊完成",
		CreatedAt:        now,
		CompletedAt:      now,
		TreatmentSummary: "药物治疗完成",
	}
	_ = s.flowCardRepo.Create(ctx, completedCard)

	completedTL := adapter.BuildFlowCardTimelineItem(input.SessionID, completedCard)
	_ = s.timelineRepo.Append(ctx, &completedTL)

	// Terminal timeline item
	termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
		"completed",
		"就诊完成",
		fmt.Sprintf("已%s，就诊结束", modeText),
	)
	_ = s.timelineRepo.Append(ctx, &termTL)

	return &model.FlowActionResult{
		SessionID:     input.SessionID,
		Status:        status,
		Card:          card,
		TimelineItems: []model.TimelineItem{drugTL, completedTL, termTL},
		Message:       fmt.Sprintf("已确认%s，就诊完成", modeText),
	}, nil
}

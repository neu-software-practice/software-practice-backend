package workbench

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// SubmitPayment processes a payment submission.
func (s *Service) SubmitPayment(ctx context.Context, input model.SubmitPaymentInput) (*model.FlowActionResult, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	card, err := s.flowCardRepo.FindByID(ctx, input.CardID)
	if err != nil {
		return nil, err
	}

	if input.Defer {
		// Defer payment
		card.Status = string(model.FlowCardStatusPending)
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on defer: %w", err)
		}

		return &model.FlowActionResult{
			SessionID: input.SessionID,
			Status:    session.Status,
			Message:   "支付已暂缓",
			TimelineItems: []model.TimelineItem{
				adapter.BuildSystemEventTimelineItem(input.SessionID,
					"payment_deferred",
					"支付暂缓",
					"患者选择暂缓支付",
				),
			},
		}, nil
	}

	// Process payment
	now := time.Now()
	card.PaymentStatus = string(model.PaymentStatusPaid)
	card.Status = string(model.FlowCardStatusPaid)
	card.HandledAt = &now
	if err := s.flowCardRepo.Update(ctx, card); err != nil {
		return nil, fmt.Errorf("update flow card on payment: %w", err)
	}

	result := &model.FlowActionResult{
		SessionID: input.SessionID,
		Card:      card,
	}

	// Create payment success timeline
	payTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		string(model.SystemEventTypePaymentSucceeded),
		"支付成功",
		fmt.Sprintf("%s 费用已支付 ¥%.2f", input.Purpose, model.DerefFloat64(card.TotalAmount)),
	)
	_ = s.timelineRepo.Append(ctx, &payTL)

	switch input.Purpose {
	case "lab":
		// After lab payment, simulate lab results and advance to lab execution
		session.MachineState = string(model.VisitMachineStateLabExecution)
		status := string(model.VisitStatusDiagnosis)
		session.Status = status
		session.ActiveCardID = nil
		session.UpdatedAt = now
		session.LastActivityAt = &now
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after lab payment: %w", err)
		}

		// Auto-generate lab results
		labResults := []struct {
			Item  string
			Value string
		}{{Item: "血常规-白细胞", Value: "11.2×10⁹/L"}}
		if err := s.SubmitLabResults(ctx, SubmitLabResultsInput{
			SessionID: input.SessionID,
			Results:   labResults,
		}); err != nil {
			slog.Warn("failed to submit auto-generated lab results", "session_id", input.SessionID, "error", err)
		}

		result.Status = status
		result.Message = "检验费支付成功，检验进行中"

	case "medication":
		// After medication payment, go to medication fulfillment
		session.MachineState = string(model.VisitMachineStateMedicationFulfillment)
		status := string(model.VisitStatusBlocked)
		session.UpdatedAt = now
		session.LastActivityAt = &now
		session.Status = status
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after medication payment: %w", err)
		}

		result.Status = status
		result.Message = "药费支付成功，请确认取药方式"
	}

	result.TimelineItems = []model.TimelineItem{payTL}
	return result, nil
}

package workbench

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// SubmitLabDecisionInput is the input for submitting a lab decision.
type SubmitLabDecisionInput struct {
	SessionID string
	CardID    string
	Decision  string // accepted, skipped, vetoed
}

// SubmitLabDecision processes a patient's lab decision.
func (s *Service) SubmitLabDecision(ctx context.Context, input SubmitLabDecisionInput) (*model.FlowActionResult, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	card, err := s.flowCardRepo.FindByID(ctx, input.CardID)
	if err != nil {
		return nil, err
	}

	result := &model.FlowActionResult{
		SessionID: input.SessionID,
	}

	now := time.Now()
	card.HandledAt = &now

	switch input.Decision {
	case "accepted":
		card.Status = string(model.FlowCardStatusAccepted)
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on lab accepted: %w", err)
		}
		s.syncCardToTimeline(ctx, card)

		// Create payment card for lab tests
		items := []model.PaymentLineItem{
			{Name: "血常规", Amount: 50.0, Quantity: 1},
		}
		paymentCard := adapter.BuildPaymentCard(input.SessionID, "lab", items, 50.0)
		if err := s.flowCardRepo.Create(ctx, paymentCard); err != nil {
			return nil, fmt.Errorf("create payment card: %w", err)
		}

		// Update state to labPayment
		status := string(model.VisitStatusBlocked)
		cardID := paymentCard.ID
		session.Status = status
		session.MachineState = string(model.VisitMachineStateLabPayment)
		session.ActiveCardID = &cardID
		session.UpdatedAt = now
		session.LastActivityAt = &now
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after lab accepted: %w", err)
		}

		result.Status = status
		result.ActiveCardID = &cardID
		result.Card = paymentCard
		result.Message = "检验已确认，请完成缴费"

		// Create timeline items
		decisionTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
			string(model.SystemEventTypeLabResultReceived),
			"检验决定",
			fmt.Sprintf("患者选择：%s", input.Decision),
		)
		if err := s.timelineRepo.Append(ctx, &decisionTL); err != nil {
			slog.Warn("failed to append lab decision timeline", "session_id", input.SessionID, "error", err)
		}

		cardTL := adapter.BuildFlowCardTimelineItem(input.SessionID, paymentCard)
		if err := s.timelineRepo.Append(ctx, &cardTL); err != nil {
			slog.Warn("failed to append payment card timeline", "session_id", input.SessionID, "error", err)
		}
		result.TimelineItems = []model.TimelineItem{decisionTL, cardTL}

	case "skipped":
		card.Status = string(model.FlowCardStatusSkipped)
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on lab skipped: %w", err)
		}
		s.syncCardToTimeline(ctx, card)

		// Go straight to diagnosis
		status := string(model.VisitStatusDiagnosis)
		session.Status = status
		session.MachineState = string(model.VisitMachineStateDiagnosis)
		session.UpdatedAt = now
		session.LastActivityAt = &now
		session.ActiveCardID = nil
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after lab skipped: %w", err)
		}

		result.Status = status
		result.Message = "已跳过检验，进入诊断阶段"

		skipTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
			"lab_skipped",
			"跳过检验",
			"患者选择不进行检验",
		)
		if err := s.timelineRepo.Append(ctx, &skipTL); err != nil {
			slog.Warn("failed to append lab skipped timeline", "session_id", input.SessionID, "error", err)
		}
		result.TimelineItems = []model.TimelineItem{skipTL}

	case "vetoed":
		card.Status = string(model.FlowCardStatusVetoed)
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on lab vetoed: %w", err)
		}
		s.syncCardToTimeline(ctx, card)

		// Return to chatting
		status := string(model.VisitStatusChatting)
		session.Status = status
		session.UpdatedAt = now
		session.LastActivityAt = &now
		session.MachineState = string(model.VisitMachineStateChatting)
		session.ActiveCardID = nil
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after lab vetoed: %w", err)
		}

		result.Status = status
		result.Message = "已暂不决定，回到问诊"

		vetoTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
			"lab_vetoed",
			"暂不决定",
			"患者选择暂不决定是否检验",
		)
		if err := s.timelineRepo.Append(ctx, &vetoTL); err != nil {
			slog.Warn("failed to append lab vetoed timeline", "session_id", input.SessionID, "error", err)
		}
		result.TimelineItems = []model.TimelineItem{vetoTL}

	default:
		return nil, fmt.Errorf("invalid lab decision: %s", input.Decision)
	}

	return result, nil
}

// SubmitLabResultsInput is the input for submitting lab results.
type SubmitLabResultsInput struct {
	SessionID string
	Results   []struct {
		Item  string
		Value string
	}
}

// SubmitLabResults processes lab test results (typically called internally after payment).
func (s *Service) SubmitLabResults(ctx context.Context, input SubmitLabResultsInput) error {
	// Create lab_execution card with results
	card := &model.FlowCard{
		ID:              "", // generated by DB
		SessionID:       input.SessionID,
		Kind:            string(model.FlowCardKindLabExecution),
		Status:          string(model.FlowCardStatusCompleted),
		Blocking:        false,
		Title:           "检验结果",
		CreatedAt:       time.Now(),
		ExecutionStatus: string(model.LabExecutionStatusResultReady),
	}

	summary := "检验完成"
	if len(input.Results) > 0 {
		summary = fmt.Sprintf("%s: %s", input.Results[0].Item, input.Results[0].Value)
	}
	card.ResultSummary = &summary

	// Persist the lab execution card
	if err := s.flowCardRepo.Create(ctx, card); err != nil {
		return fmt.Errorf("create lab execution card: %w", err)
	}

	// Create result timeline item
	resultTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		string(model.SystemEventTypeLabResultReceived),
		"检验结果已出",
		summary,
	)

	return s.timelineRepo.Append(ctx, &resultTL)
}

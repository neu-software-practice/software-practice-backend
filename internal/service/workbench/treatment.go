package workbench

import (
	"context"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// SubmitTreatmentExecutionInput is the input for treatment execution.
type SubmitTreatmentExecutionInput struct {
	SessionID string
	CardID    string
	Action    string // schedule, confirm_arrival, start, complete, cancel
}

// SubmitTreatmentExecution processes a treatment execution action.
func (s *Service) SubmitTreatmentExecution(ctx context.Context, input SubmitTreatmentExecutionInput) (*model.FlowActionResult, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	card, err := s.flowCardRepo.FindByID(ctx, input.CardID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := &model.FlowActionResult{
		SessionID: input.SessionID,
	}

	switch input.Action {
	case "schedule":
		card.ExecutionStatus = string(model.ExecutionStatusScheduled)
		card.Status = string(model.FlowCardStatusProcessing)
		future := now.Add(30 * time.Minute)
		card.AppointmentAt = &future
		_ = s.flowCardRepo.Update(ctx, card)

		result.Status = session.Status
		result.Card = card
		result.Message = "治疗已预约"

	case "confirm_arrival":
		card.ExecutionStatus = string(model.ExecutionStatusArrived)
		_ = s.flowCardRepo.Update(ctx, card)

		result.Status = session.Status
		result.Card = card
		result.Message = "已确认到达"

	case "start":
		card.ExecutionStatus = string(model.ExecutionStatusInProgress)
		_ = s.flowCardRepo.Update(ctx, card)

		result.Status = session.Status
		result.Card = card
		result.Message = "治疗开始"

	case "complete":
		card.ExecutionStatus = string(model.ExecutionStatusCompleted)
		card.Status = string(model.FlowCardStatusCompleted)
		card.HandledAt = &now
		_ = s.flowCardRepo.Update(ctx, card)

		// Complete the session
		reason := "completed"
		session.Status = string(model.VisitStatusCompleted)
		session.EndedAt = &now
		session.TerminalReason = &reason
		session.ActiveCardID = nil
		ts := "治疗完成"
		session.Summary.TreatmentSummary = &ts
		_ = s.visitRepo.Update(ctx, session)

		// Completed visit card
		completedCard := &model.FlowCard{
			ID:               "",
			SessionID:        input.SessionID,
			Kind:             string(model.FlowCardKindCompletedVisit),
			Status:           string(model.FlowCardStatusCompleted),
			Blocking:         false,
			Title:            "就诊完成",
			CreatedAt:        now,
			CompletedAt:      now,
			TreatmentSummary: "治疗完成",
		}
		_ = s.flowCardRepo.Create(ctx, completedCard)

		termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
			"completed",
			"治疗完成",
			"治疗已全部完成，就诊结束",
		)
		_ = s.timelineRepo.Append(ctx, &termTL)

		result.Status = string(model.VisitStatusCompleted)
		result.Card = card
		result.Message = "治疗完成，就诊结束"
		result.TimelineItems = []model.TimelineItem{termTL}

	case "cancel":
		card.ExecutionStatus = string(model.ExecutionStatusCanceled)
		card.Status = string(model.FlowCardStatusInvalidated)
		card.HandledAt = &now
		_ = s.flowCardRepo.Update(ctx, card)

		// Return to treatment decision
		session.Status = string(model.VisitStatusTreatment)
		session.ActiveCardID = nil
		_ = s.visitRepo.Update(ctx, session)

		result.Status = string(model.VisitStatusTreatment)
		result.Card = card
		result.Message = "治疗已取消"

	default:
		return nil, fmt.Errorf("invalid treatment action: %s", input.Action)
	}

	// Timeline event
	actionTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		"treatment_action",
		"治疗进度",
		fmt.Sprintf("操作：%s", input.Action),
	)
	_ = s.timelineRepo.Append(ctx, &actionTL)
	result.TimelineItems = append(result.TimelineItems, actionTL)

	return result, nil
}

// AckAdviceInput is the input for acknowledging advice-only treatment.
type AckAdviceInput struct {
	SessionID string
	CardID    string
}

// AckAdvice acknowledges an advice-only treatment plan and completes the session.
func (s *Service) AckAdvice(ctx context.Context, input AckAdviceInput) (*model.FlowActionResult, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	card, err := s.flowCardRepo.FindByID(ctx, input.CardID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	card.Status = string(model.FlowCardStatusCompleted)
	card.HandledAt = &now
	_ = s.flowCardRepo.Update(ctx, card)

	// Complete session
	reason := "completed"
	session.Status = string(model.VisitStatusCompleted)
	session.EndedAt = &now
	session.TerminalReason = &reason
	session.ActiveCardID = nil
	_ = s.visitRepo.Update(ctx, session)

	// Completed visit card
	completedCard := &model.FlowCard{
		ID:               "",
		SessionID:        input.SessionID,
		Kind:             string(model.FlowCardKindCompletedVisit),
		Status:           string(model.FlowCardStatusCompleted),
		Blocking:         false,
		Title:            "就诊完成",
		CreatedAt:        now,
		CompletedAt:      now,
		TreatmentSummary: "医嘱确认完成",
	}
	_ = s.flowCardRepo.Create(ctx, completedCard)

	ackTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		"advice_acknowledged",
		"医嘱已确认",
		"患者已确认医嘱",
	)
	_ = s.timelineRepo.Append(ctx, &ackTL)

	termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
		"completed",
		"就诊完成",
		"医嘱确认完成，就诊结束",
	)
	_ = s.timelineRepo.Append(ctx, &termTL)

	return &model.FlowActionResult{
		SessionID:     input.SessionID,
		Status:        string(model.VisitStatusCompleted),
		Card:          card,
		TimelineItems: []model.TimelineItem{ackTL, termTL},
		Message:       "医嘱已确认，就诊完成",
	}, nil
}

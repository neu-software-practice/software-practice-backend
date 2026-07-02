package workbench

import (
	"context"
	"fmt"
	"log/slog"
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
		card.ExecutionStatus = string(model.TreatmentExecutionStatusScheduled)
		card.Status = string(model.FlowCardStatusProcessing)
		future := now.Add(30 * time.Minute)
		card.AppointmentAt = &future
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on schedule: %w", err)
		}

		result.Status = session.Status
		result.Card = card
		result.Message = "治疗已预约"

	case "confirm_arrival":
		card.ExecutionStatus = string(model.TreatmentExecutionStatusArrived)
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on confirm arrival: %w", err)
		}

		result.Status = session.Status
		result.Card = card
		result.Message = "已确认到达"

	case "start":
		card.ExecutionStatus = string(model.TreatmentExecutionStatusInProgress)
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on start: %w", err)
		}

		result.Status = session.Status
		result.Card = card
		result.Message = "治疗开始"

	case "complete":
		card.ExecutionStatus = string(model.TreatmentExecutionStatusCompleted)
		card.Status = string(model.FlowCardStatusCompleted)
		card.HandledAt = &now
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on complete: %w", err)
		}

		// Complete the session
		reason := "completed"
		session.Status = string(model.VisitStatusCompleted)
		session.MachineState = string(model.VisitMachineStateCompleted)
		session.EndedAt = &now
		session.TerminalReason = &reason
		session.ActiveCardID = nil
		ts := "治疗完成"
		session.Summary.TreatmentSummary = &ts
		session.UpdatedAt = now
		session.LastActivityAt = &now
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after treatment complete: %w", err)
		}

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
		if err := s.flowCardRepo.Create(ctx, completedCard); err != nil {
			return nil, fmt.Errorf("create completed visit card: %w", err)
		}

		termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
			"completed",
			"治疗完成",
			"治疗已全部完成，就诊结束",
		)
		if err := s.timelineRepo.Append(ctx, &termTL); err != nil {
			slog.Warn("failed to append terminal timeline on treatment complete", "session_id", input.SessionID, "error", err)
		}

		result.Status = string(model.VisitStatusCompleted)
		result.Card = card
		result.Message = "治疗完成，就诊结束"
		result.TimelineItems = []model.TimelineItem{termTL}

	case "cancel":
		card.ExecutionStatus = string(model.TreatmentExecutionStatusCanceled)
		card.Status = string(model.FlowCardStatusInvalidated)
		card.HandledAt = &now
		if err := s.flowCardRepo.Update(ctx, card); err != nil {
			return nil, fmt.Errorf("update flow card on cancel: %w", err)
		}

		// Return to treatment decision
		session.Status = string(model.VisitStatusTreatment)
		session.MachineState = string(model.VisitMachineStateTreatmentDecision)
		session.ActiveCardID = nil
		session.UpdatedAt = now
		session.LastActivityAt = &now
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return nil, fmt.Errorf("update session after treatment cancel: %w", err)
		}

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
	if err := s.timelineRepo.Append(ctx, &actionTL); err != nil {
		slog.Warn("failed to append treatment action timeline", "session_id", input.SessionID, "action", input.Action, "error", err)
	}
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
	if err := s.flowCardRepo.Update(ctx, card); err != nil {
		return nil, fmt.Errorf("update flow card on ack advice: %w", err)
	}

	// Complete session
	reason := "completed"
	session.Status = string(model.VisitStatusCompleted)
	session.MachineState = string(model.VisitMachineStateCompleted)
	session.EndedAt = &now
	session.TerminalReason = &reason
	session.ActiveCardID = nil
	session.UpdatedAt = now
	session.LastActivityAt = &now
	if err := s.visitRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session after ack advice: %w", err)
	}

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
	if err := s.flowCardRepo.Create(ctx, completedCard); err != nil {
		return nil, fmt.Errorf("create completed visit card: %w", err)
	}

	ackTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		"advice_acknowledged",
		"医嘱已确认",
		"患者已确认医嘱",
	)
	if err := s.timelineRepo.Append(ctx, &ackTL); err != nil {
		slog.Warn("failed to append advice ack timeline", "session_id", input.SessionID, "error", err)
	}

	termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
		"completed",
		"就诊完成",
		"医嘱确认完成，就诊结束",
	)
	if err := s.timelineRepo.Append(ctx, &termTL); err != nil {
		slog.Warn("failed to append terminal timeline on ack advice", "session_id", input.SessionID, "error", err)
	}

	return &model.FlowActionResult{
		SessionID:     input.SessionID,
		Status:        string(model.VisitStatusCompleted),
		Card:          card,
		TimelineItems: []model.TimelineItem{ackTL, termTL},
		Message:       "医嘱已确认，就诊完成",
	}, nil
}

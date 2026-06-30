package workbench

import (
	"context"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// validExitReasons is the set of allowed reason values for ExitVisit.
var validExitReasons = map[string]bool{
	"patient_request": true,
	"timeout":         true,
	"emergency":       true,
	"other":           true,
}

// ExitVisit processes a patient's request to exit the visit.
// Returns the settlement result with one of four consequences.
func (s *Service) ExitVisit(ctx context.Context, input model.ExitVisitInput) (*model.ExitSettlementResult, error) {
	if !validExitReasons[input.Reason] {
		return nil, fmt.Errorf("invalid exit reason: %q", input.Reason)
	}

	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Determine consequence based on session state
	consequence := determineExitConsequence(session)

	reason := input.Reason
	session.Status = string(model.VisitStatusExited)
	session.MachineState = string(model.VisitMachineStateExited)
	session.EndedAt = &now
	session.TerminalReason = &reason
	_ = s.visitRepo.Update(ctx, session)

	// Create exit settlement timeline item
	exitTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		string(model.SystemEventTypeExitSettled),
		"退出结算",
		consequence.Text,
	)

	terminalTL := adapter.BuildTerminalTimelineItem(input.SessionID,
		reason,
		"主动退出",
		consequence.Text,
	)

	_ = s.timelineRepo.Append(ctx, &exitTL)
	_ = s.timelineRepo.Append(ctx, &terminalTL)

	return &model.ExitSettlementResult{
		SessionID:      input.SessionID,
		TerminalReason: reason,
		RefundAmount:   consequence.Amount,
		PayableAmount:  0,
		TimelineItem:   terminalTL,
		Consequence:    consequence,
	}, nil
}

// determineExitConsequence determines the exit consequence based on session state.
func determineExitConsequence(session *model.VisitSession) *model.ExitConsequence {
	status := session.Status

	switch status {
	case string(model.VisitStatusLoadingContext),
		string(model.VisitStatusChatting),
		string(model.VisitStatusAnalyzing):
		// No fee incurred yet
		return &model.ExitConsequence{
			Kind:   string(model.ExitConsequenceNoFee),
			Amount: 0,
			Text:   "未产生任何费用，已退出就诊",
		}

	case string(model.VisitStatusBlocked):
		// Check if payment was made
		if session.Summary.Diagnosis != nil {
			// Diagnosis was made but treatment incomplete
			return &model.ExitConsequence{
				Kind:   string(model.ExitConsequenceRefundable),
				Amount: 50.0,
				Text:   "已付费用可退款 ¥50.00，已退出就诊",
			}
		}
		return &model.ExitConsequence{
			Kind:   string(model.ExitConsequenceNoFee),
			Amount: 0,
			Text:   "未产生费用，已退出就诊",
		}

	case string(model.VisitStatusDiagnosis),
		string(model.VisitStatusTreatment):
		return &model.ExitConsequence{
			Kind:   string(model.ExitConsequenceExecutedNoRefund),
			Amount: 0,
			Text:   "诊疗已执行，不可退款，已退出就诊",
		}

	case string(model.VisitStatusCompleted):
		return &model.ExitConsequence{
			Kind:   string(model.ExitConsequenceMedicationDispensed),
			Amount: 0,
			Text:   "药品已发出，按已购计费，已退出就诊",
		}

	default:
		return &model.ExitConsequence{
			Kind:   string(model.ExitConsequenceNoFee),
			Amount: 0,
			Text:   "已退出就诊",
		}
	}
}

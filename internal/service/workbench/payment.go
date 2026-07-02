package workbench

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
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
		s.syncCardToTimeline(ctx, card)

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
	s.syncCardToTimeline(ctx, card)

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
	if err := s.timelineRepo.Append(ctx, &payTL); err != nil {
		slog.Warn("failed to append payment success timeline", "session_id", input.SessionID, "error", err)
	}

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

		// Feed test results back to medAgent to continue the agent loop.
		// The agent has been waiting for test-results since the NEED_TESTS step.
		medAgentDone := false
		if s.maClient != nil && session.MedAgentSessionID != nil && *session.MedAgentSessionID != "" {
			testResults := []medagent.TestResult{
				{Item: "血常规-白细胞", Value: "11.2×10⁹/L"},
			}
			nextStep, err := s.maClient.TestResults(ctx, *session.MedAgentSessionID, testResults)
			if err != nil {
				slog.Warn("failed to send test results to medAgent", "session_id", input.SessionID, "error", err)
			} else if nextStep != nil {
				s.applyMedAgentStep(ctx, session, nextStep)
				medAgentDone = (nextStep.Kind == medagent.StepDone)
				session.UpdatedAt = time.Now()
				session.LastActivityAt = &now
				if err := s.visitRepo.Update(ctx, session); err != nil {
					slog.Warn("failed to persist session after medAgent step", "session_id", input.SessionID, "error", err)
				}
			}
		}

		result.Status = session.Status
		if medAgentDone {
			result.Message = "检验费支付成功，诊断结果已出"
		} else {
			result.Message = "检验费支付成功，AI医生正在分析结果，请继续对话"
		}

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

// applyMedAgentStep processes a medAgent step response without SSE streaming.
// It creates cards and timeline items directly, mutating the session in place.
// The caller is responsible for persisting the session afterward.
func (s *Service) applyMedAgentStep(ctx context.Context, session *model.VisitSession, step *medagent.Step) {
	switch step.Kind {
	case medagent.StepDone:
		s.applyDoneStep(ctx, session, step)

	case medagent.StepAsk:
		// Agent wants more information — return to chatting so the frontend can send messages.
		session.Status = string(model.VisitStatusChatting)
		session.MachineState = string(model.VisitMachineStateChatting)
		session.ActiveCardID = nil

		// Create assistant message timeline (mirrors chat.go handleAsk behavior)
		content := step.DoctorSay
		if content == "" {
			content = "请继续描述您的症状。"
		}
		msgItem := adapter.BuildMessageTimelineItem(session.ID, "assistant", content)
		if err := s.timelineRepo.Append(ctx, &msgItem); err != nil {
			slog.Warn("failed to append assistant message timeline in payment flow", "session_id", session.ID, "error", err)
		}

	case medagent.StepNeedTests:
		// Agent needs additional tests (uncommon after test results, but handle gracefully).
		card := adapter.BuildLabDecisionCard(session.ID, step)
		if err := s.flowCardRepo.Create(ctx, card); err != nil {
			slog.Warn("failed to create lab decision card in payment flow", "session_id", session.ID, "error", err)
			return
		}
		tlItem := adapter.BuildFlowCardTimelineItem(session.ID, card)
		if err := s.timelineRepo.Append(ctx, &tlItem); err != nil {
			slog.Warn("failed to append need-tests card timeline", "session_id", session.ID, "error", err)
		}
		cardID := card.ID
		session.Status = string(model.VisitStatusBlocked)
		session.MachineState = string(model.VisitMachineStateLabDecision)
		session.ActiveCardID = &cardID

	case medagent.StepEmergency:
		termItem := adapter.BuildTerminalTimelineItem(session.ID,
			string(model.TerminalReasonEmergency),
			"急症",
			step.Emergency,
		)
		if err := s.timelineRepo.Append(ctx, &termItem); err != nil {
			slog.Warn("failed to append emergency terminal timeline", "session_id", session.ID, "error", err)
		}
		now := time.Now()
		reason := string(model.TerminalReasonEmergency)
		session.Status = string(model.VisitStatusEmergencyTerminated)
		session.MachineState = string(model.VisitMachineStateTerminated)
		session.EndedAt = &now
		session.TerminalReason = &reason

	default:
		slog.Warn("unexpected medAgent step after test results", "session_id", session.ID, "kind", string(step.Kind))
		// Default to diagnosis state so the frontend can at least see results.
		session.Status = string(model.VisitStatusDiagnosis)
		session.MachineState = string(model.VisitMachineStateDiagnosis)
		session.ActiveCardID = nil
	}
}

// applyDoneStep processes a DONE step from medAgent — creates diagnosis and
// treatment plan cards, then updates session state based on the plan type.
// This mirrors handleDone in chat.go but without SSE streaming callbacks.
func (s *Service) applyDoneStep(ctx context.Context, session *model.VisitSession, step *medagent.Step) {
	result := step.Result
	if result == nil {
		slog.Warn("DONE step has no result", "session_id", session.ID)
		return
	}

	// 1. Create diagnosis card
	diagCard := adapter.BuildDiagnosisCard(session.ID, result)
	if err := s.flowCardRepo.Create(ctx, diagCard); err != nil {
		slog.Warn("failed to create diagnosis card in payment flow", "session_id", session.ID, "error", err)
	} else {
		diagTL := adapter.BuildFlowCardTimelineItem(session.ID, diagCard)
		if err := s.timelineRepo.Append(ctx, &diagTL); err != nil {
			slog.Warn("failed to append diagnosis timeline in payment flow", "session_id", session.ID, "error", err)
		}
	}

	// 2. Create treatment plan card
	planCard := adapter.BuildTreatmentPlanCard(session.ID, result)
	if err := s.flowCardRepo.Create(ctx, planCard); err != nil {
		slog.Warn("failed to create treatment plan card in payment flow", "session_id", session.ID, "error", err)
	} else {
		planTL := adapter.BuildFlowCardTimelineItem(session.ID, planCard)
		if err := s.timelineRepo.Append(ctx, &planTL); err != nil {
			slog.Warn("failed to append treatment plan timeline in payment flow", "session_id", session.ID, "error", err)
		}
	}

	// 3. Handle plan-specific state and cards
	switch result.Plan {
	case "MEDICATION":
		session.Status = string(model.VisitStatusDiagnosis)
		session.MachineState = string(model.VisitMachineStateDiagnosis)
		session.ActiveCardID = nil

	case "ADVICE_ONLY":
		adviceCard := adapter.BuildAdviceOnlyCard(session.ID, result)
		if err := s.flowCardRepo.Create(ctx, adviceCard); err != nil {
			slog.Warn("failed to create advice card in payment flow", "session_id", session.ID, "error", err)
		} else {
			adviceTL := adapter.BuildFlowCardTimelineItem(session.ID, adviceCard)
			if err := s.timelineRepo.Append(ctx, &adviceTL); err != nil {
				slog.Warn("failed to append advice timeline in payment flow", "session_id", session.ID, "error", err)
			}
		}
		cardID := adviceCard.ID
		session.Status = string(model.VisitStatusBlocked)
		session.MachineState = string(model.VisitMachineStateAdviceOnly)
		session.ActiveCardID = &cardID

	case "REFERRAL":
		completedCard := adapter.BuildCompletedVisitCard(session.ID, result)
		if err := s.flowCardRepo.Create(ctx, completedCard); err != nil {
			slog.Warn("failed to create completed visit card in payment flow", "session_id", session.ID, "error", err)
		} else {
			compTL := adapter.BuildFlowCardTimelineItem(session.ID, completedCard)
			if err := s.timelineRepo.Append(ctx, &compTL); err != nil {
				slog.Warn("failed to append completed visit timeline in payment flow", "session_id", session.ID, "error", err)
			}
		}
		termItem := adapter.BuildTerminalTimelineItem(session.ID,
			string(model.TerminalReasonReferral),
			"转诊",
			result.Advice,
		)
		if err := s.timelineRepo.Append(ctx, &termItem); err != nil {
			slog.Warn("failed to append referral terminal timeline in payment flow", "session_id", session.ID, "error", err)
		}
		now := time.Now()
		reason := string(model.TerminalReasonReferral)
		session.Status = string(model.VisitStatusTransferred)
		session.MachineState = string(model.VisitMachineStateTransferred)
		session.EndedAt = &now
		session.TerminalReason = &reason
		ts := result.Plan
		session.Summary.TreatmentSummary = &ts

	default:
		// Unknown plan — default to diagnosis state.
		slog.Warn("unknown treatment plan from medAgent", "session_id", session.ID, "plan", result.Plan)
		session.Status = string(model.VisitStatusDiagnosis)
		session.MachineState = string(model.VisitMachineStateDiagnosis)
		session.ActiveCardID = nil
	}

	// 4. Set diagnosis in session summary
	if result.Diagnosis != nil {
		diag := result.Diagnosis.Name
		session.Summary.Diagnosis = &diag
	}
}

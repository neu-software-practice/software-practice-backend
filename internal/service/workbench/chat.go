package workbench

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// SendMessageInput represents a patient message submission.
type SendMessageInput struct {
	SessionID       string
	Content         string
	ClientMessageID string
}

// SendMessageResult is the result of sending a patient message.
type SendMessageResult struct {
	Session              model.VisitSession
	PatientMessage       model.TimelineItem
	AssistantPlaceholder *model.TimelineItem
}

// SendMessage sends a patient message and returns a placeholder for the AI response.
func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*SendMessageResult, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	// Create patient message timeline item
	patientMsg := adapter.BuildMessageTimelineItem(input.SessionID, "patient", input.Content)
	patientMsg.ID = input.ClientMessageID
	if err := s.timelineRepo.Append(ctx, &patientMsg); err != nil {
		return nil, fmt.Errorf("append patient message: %w", err)
	}

	// Update session
	session.AskRound++
	session.UpdatedAt = time.Now()
	lm := input.Content
	session.Summary.LastMessage = &lm

	var placeholder *model.TimelineItem
	// Create placeholder for assistant response
	placeholderItem := adapter.BuildMessageTimelineItem(input.SessionID, "assistant", "")
	placeholderItem.Status = string(model.TimelineItemStatusPending)
	placeholderItem.ID = uuid.New().String()
	if err := s.timelineRepo.Append(ctx, &placeholderItem); err != nil {
		return nil, fmt.Errorf("append placeholder: %w", err)
	}
	placeholder = &placeholderItem

	return &SendMessageResult{
		Session:              *session,
		PatientMessage:       patientMsg,
		AssistantPlaceholder: placeholder,
	}, nil
}

// StreamAssistantInput represents an assistant stream request.
type StreamAssistantInput struct {
	SessionID       string
	RequestID       string
	ClientMessageID string
}

// StreamAssistantEventCallback is called for each SSE event during streaming.
type StreamAssistantEventCallback func(event model.AssistantStreamEvent) error

// StreamAssistantMessage orchestrates calling medAgent and streaming SSE events.
// It handles the full conversation loop: patient-say → step processing → SSE events.
func (s *Service) StreamAssistantMessage(ctx context.Context, input StreamAssistantInput, callback StreamAssistantEventCallback) error {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return err
	}

	// Determine the last patient message
	var lastMessage string
	timeline, _, _, _ := s.timelineRepo.ListBySession(ctx, input.SessionID, nil, 50)
	for i := len(timeline) - 1; i >= 0; i-- {
		if timeline[i].Kind == "message" && timeline[i].Role == "patient" {
			lastMessage = timeline[i].Content
			break
		}
	}
	if lastMessage == "" {
		lastMessage = "你好"
	}

	// Build patient profile for medAgent
	patient, err := s.patientRepo.FindByID(ctx, session.PatientID)
	if err != nil {
		return fmt.Errorf("find patient: %w", err)
	}

	profile := map[string]interface{}{
		"age":    patient.Age,
		"gender": patient.Gender,
		"allergies": patient.Allergies,
	}

	// Create medAgent session
	maSessionID, err := s.medAgentClient.CreateSession(ctx, profile, session.EntryType == "new", nil)
	if err != nil {
		return fmt.Errorf("create medagent session: %w", err)
	}

	// Send patient message to medAgent
	step, err := s.medAgentClient.PatientSay(ctx, maSessionID, lastMessage)
	if err != nil {
		return fmt.Errorf("patient say: %w", err)
	}

	// Process the step and stream events
	return s.processStep(ctx, input.SessionID, input.RequestID, maSessionID, session, step, callback)
}

// processStep handles a medAgent Step and produces SSE events.
func (s *Service) processStep(
	ctx context.Context,
	sessionID, requestID, maSessionID string,
	session *model.VisitSession,
	step *medagent.Step,
	callback StreamAssistantEventCallback,
) error {
	if step == nil {
		return fmt.Errorf("nil step from medAgent")
	}

	switch step.Kind {
	case medagent.StepAsk:
		return s.handleAsk(ctx, sessionID, requestID, session, step, callback)

	case medagent.StepNeedTests:
		return s.handleNeedTests(ctx, sessionID, requestID, session, step, callback)

	case medagent.StepDrugQuery:
		return s.handleDrugQuery(ctx, sessionID, requestID, maSessionID, session, step, callback)

	case medagent.StepPurchase:
		return s.handlePurchase(ctx, sessionID, requestID, session, step, callback)

	case medagent.StepEmergency:
		return s.handleEmergency(ctx, sessionID, requestID, session, step, callback)

	case medagent.StepDone:
		return s.handleDone(ctx, sessionID, requestID, session, step, callback)

	case medagent.StepOK:
		return s.handleOK(ctx, sessionID, requestID, session, callback)

	default:
		return fmt.Errorf("unknown step kind: %s", step.Kind)
	}
}

func (s *Service) handleAsk(ctx context.Context, sessionID, requestID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Stream delta events
	content := step.DoctorSay
	if content == "" {
		content = "请继续描述您的症状。"
	}

	// Send delta
	_ = callback(model.AssistantStreamEvent{
		Type:      "delta",
		SessionID: sessionID,
		RequestID: requestID,
		Content:   content,
	})

	// Create message timeline item
	msgItem := adapter.BuildMessageTimelineItem(sessionID, "assistant", content)
	if err := s.timelineRepo.Append(ctx, &msgItem); err != nil {
		return fmt.Errorf("append assistant message: %w", err)
	}

	// Send message_final
	_ = callback(model.AssistantStreamEvent{
		Type:      "message_final",
		SessionID: sessionID,
		RequestID: requestID,
		Item:      &msgItem,
	})

	// Send state update
	_ = callback(model.AssistantStreamEvent{
		Type:      "state",
		SessionID: sessionID,
		State:     string(model.VisitMachineStateChatting),
		Status:    string(model.VisitStatusChatting),
	})

	// Send done
	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	// Update session
	session.AskRound++
	session.UpdatedAt = time.Now()
	lm := content
	session.Summary.LastMessage = &lm

	return nil
}

func (s *Service) handleNeedTests(ctx context.Context, sessionID, requestID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Build lab_decision card
	card := adapter.BuildLabDecisionCard(sessionID, step)
	if err := s.flowCardRepo.Create(ctx, card); err != nil {
		return fmt.Errorf("create lab decision card: %w", err)
	}

	// Send card event
	_ = callback(model.AssistantStreamEvent{
		Type:      "card",
		SessionID: sessionID,
		RequestID: requestID,
		Card:      card,
	})

	// Create timeline item for the card
	tlItem := adapter.BuildFlowCardTimelineItem(sessionID, card)
	if err := s.timelineRepo.Append(ctx, &tlItem); err != nil {
		return fmt.Errorf("append card timeline: %w", err)
	}

	_ = callback(model.AssistantStreamEvent{
		Type:         "card",
		SessionID:    sessionID,
		RequestID:    requestID,
		Card:         card,
		TimelineItem: &tlItem,
	})

	// Update state to labDecision (blocked)
	newState := string(model.VisitMachineStateLabDecision)
	status := string(model.VisitStatusBlocked)
	cardID := card.ID
	session.Status = status
	session.ActiveCardID = &cardID

	_ = callback(model.AssistantStreamEvent{
		Type:         "state",
		SessionID:    sessionID,
		State:        newState,
		Status:       status,
		ActiveCardID: &cardID,
	})

	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	return nil
}

func (s *Service) handleDrugQuery(ctx context.Context, sessionID, requestID, maSessionID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Send state event (drug query is transparent to user)
	_ = callback(model.AssistantStreamEvent{
		Type:      "state",
		SessionID: sessionID,
		State:     string(model.VisitMachineStateAnalyzing),
		Status:    string(model.VisitStatusAnalyzing),
	})

	// Auto-fill drug info with mock data and continue
	infos := make([]medagent.DrugInfo, len(step.DrugNames))
	for i, name := range step.DrugNames {
		infos[i] = medagent.DrugInfo{
			Name: name,
			Spec: fmt.Sprintf("每盒24粒×0.3g (%s)", name),
		}
	}

	// Send drug info to medAgent
	nextStep, err := s.medAgentClient.DrugInfo(ctx, maSessionID, infos)
	if err != nil {
		return fmt.Errorf("drug info: %w", err)
	}

	// Process next step
	return s.processStep(ctx, sessionID, requestID, maSessionID, session, nextStep, callback)
}

func (s *Service) handlePurchase(ctx context.Context, sessionID, requestID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Build medication fulfillment card
	card := adapter.BuildMedicationFulfillmentCard(sessionID, step)
	if err := s.flowCardRepo.Create(ctx, card); err != nil {
		return fmt.Errorf("create medication card: %w", err)
	}

	// Send card event
	_ = callback(model.AssistantStreamEvent{
		Type:      "card",
		SessionID: sessionID,
		RequestID: requestID,
		Card:      card,
	})

	tlItem := adapter.BuildFlowCardTimelineItem(sessionID, card)
	if err := s.timelineRepo.Append(ctx, &tlItem); err != nil {
		return fmt.Errorf("append card timeline: %w", err)
	}

	// Update state to blocked
	newState := string(model.VisitMachineStateMedicationFulfillment)
	status := string(model.VisitStatusBlocked)
	cardID := card.ID
	session.Status = status
	session.ActiveCardID = &cardID

	_ = callback(model.AssistantStreamEvent{
		Type:         "state",
		SessionID:    sessionID,
		State:        newState,
		Status:       status,
		ActiveCardID: &cardID,
	})

	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	return nil
}

func (s *Service) handleEmergency(ctx context.Context, sessionID, requestID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Send emergency event
	_ = callback(model.AssistantStreamEvent{
		Type:      "emergency",
		SessionID: sessionID,
		Severity:  string(model.EmergencySeverityCritical),
		Message:   step.Emergency,
	})

	// Create terminal timeline item
	termItem := adapter.BuildTerminalTimelineItem(sessionID,
		string(model.TerminalReasonEmergency),
		"急症",
		step.Emergency,
	)
	if err := s.timelineRepo.Append(ctx, &termItem); err != nil {
		return fmt.Errorf("append terminal timeline: %w", err)
	}

	// Terminate session
	now := time.Now()
	reason := string(model.TerminalReasonEmergency)
	session.Status = string(model.VisitStatusEmergencyTerminated)
	session.EndedAt = &now
	session.TerminalReason = &reason

	return nil
}

func (s *Service) handleDone(ctx context.Context, sessionID, requestID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	result := step.Result
	if result == nil {
		return fmt.Errorf("DONE step has no result")
	}

	// 1. Send diagnosis card
	diagCard := adapter.BuildDiagnosisCard(sessionID, result)
	if err := s.flowCardRepo.Create(ctx, diagCard); err != nil {
		return fmt.Errorf("create diagnosis card: %w", err)
	}
	_ = callback(model.AssistantStreamEvent{
		Type:      "card",
		SessionID: sessionID,
		RequestID: requestID,
		Card:      diagCard,
	})
	diagTL := adapter.BuildFlowCardTimelineItem(sessionID, diagCard)
	_ = s.timelineRepo.Append(ctx, &diagTL)

	// 2. Send treatment plan card
	planCard := adapter.BuildTreatmentPlanCard(sessionID, result)
	if err := s.flowCardRepo.Create(ctx, planCard); err != nil {
		return fmt.Errorf("create treatment plan card: %w", err)
	}

	// 3. Based on plan, determine next state
	switch result.Plan {
	case "MEDICATION":
		// Will be followed by PURCHASE step
		_ = callback(model.AssistantStreamEvent{
			Type:      "state",
			SessionID: sessionID,
			State:     string(model.VisitMachineStateDiagnosis),
			Status:    string(model.VisitStatusDiagnosis),
		})

	case "ADVICE_ONLY":
		// Create advice_only card
		adviceCard := adapter.BuildAdviceOnlyCard(sessionID, result)
		if err := s.flowCardRepo.Create(ctx, adviceCard); err != nil {
			return fmt.Errorf("create advice card: %w", err)
		}
		_ = callback(model.AssistantStreamEvent{
			Type:      "card",
			SessionID: sessionID,
			RequestID: requestID,
			Card:      adviceCard,
		})
		adviceTL := adapter.BuildFlowCardTimelineItem(sessionID, adviceCard)
		_ = s.timelineRepo.Append(ctx, &adviceTL)

		cardID := adviceCard.ID
		session.ActiveCardID = &cardID

		_ = callback(model.AssistantStreamEvent{
			Type:         "state",
			SessionID:    sessionID,
			State:        string(model.VisitMachineStateAdviceOnly),
			Status:       string(model.VisitStatusBlocked),
			ActiveCardID: &cardID,
		})

	case "REFERRAL":
		// Create completed visit card with referral
		completedCard := adapter.BuildCompletedVisitCard(sessionID, result)
		if err := s.flowCardRepo.Create(ctx, completedCard); err != nil {
			return fmt.Errorf("create completed card: %w", err)
		}
		_ = callback(model.AssistantStreamEvent{
			Type:      "card",
			SessionID: sessionID,
			RequestID: requestID,
			Card:      completedCard,
		})
		compTL := adapter.BuildFlowCardTimelineItem(sessionID, completedCard)
		_ = s.timelineRepo.Append(ctx, &compTL)

		// Terminal - referral
		termItem := adapter.BuildTerminalTimelineItem(sessionID,
			string(model.TerminalReasonReferral),
			"转诊",
			result.Advice,
		)
		_ = s.timelineRepo.Append(ctx, &termItem)

		now := time.Now()
		reason := string(model.TerminalReasonReferral)
		session.EndedAt = &now
		session.TerminalReason = &reason
		session.Status = string(model.VisitStatusTransferred)

		diag := result.Diagnosis.Name
		session.Summary.Diagnosis = &diag
		ts := result.Plan
		session.Summary.TreatmentSummary = &ts

		_ = callback(model.AssistantStreamEvent{
			Type:      "state",
			SessionID: sessionID,
			State:     string(model.VisitMachineStateCompleted),
			Status:    string(model.VisitStatusTransferred),
		})
	}

	// Set diagnosis in session summary
	if result.Diagnosis != nil {
		diag := result.Diagnosis.Name
		session.Summary.Diagnosis = &diag
	}

	// Send done
	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	return nil
}

func (s *Service) handleOK(ctx context.Context, sessionID, requestID string, session *model.VisitSession, callback StreamAssistantEventCallback) error {
	_ = callback(model.AssistantStreamEvent{
		Type:      "state",
		SessionID: sessionID,
		State:     string(model.VisitMachineStateChatting),
		Status:    string(model.VisitStatusChatting),
	})

	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	return nil
}

// Helper to stringify interface for medAgent messages

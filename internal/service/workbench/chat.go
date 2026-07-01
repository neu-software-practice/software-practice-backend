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

// NOTE: Methods in this file load a *model.VisitSession via FindByID, mutate its fields
// (e.g. AskRound, Status, MachineState, Summary, etc.), then persist via visitRepo.Update.
// These mutations are intentional per-request modifications on a session loaded for that
// specific request and are safe from goroutine sharing. Each request obtains its own pointer
// from the database. The pattern is pragmatic — the session is a write-model aggregate loaded
// per-request, not a shared value.
type SendMessageInput struct {
	SessionID       string
	Content         string
	ClientMessageID string
}

// SendMessageResult is the result of sending a patient message.
type SendMessageResult struct {
	Session              model.VisitSession  `json:"session"`
	PatientMessage       model.TimelineItem  `json:"patientMessage"`
	AssistantPlaceholder *model.TimelineItem `json:"assistantPlaceholder,omitempty"`
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
	now := time.Now()
	session.UpdatedAt = now
	session.LastActivityAt = &now
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

	// Persist session mutations
	if err := s.visitRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update visit session: %w", err)
	}

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
	lastMessage, err := s.timelineRepo.FindLastPatientMessage(ctx, input.SessionID)
	if err != nil {
		return fmt.Errorf("find last patient message: %w", err)
	}
	if lastMessage == "" {
		lastMessage = "你好"
	}

	// Reuse existing medAgent session or create a new one
	var maSessionID string
	if session.MedAgentSessionID != nil && *session.MedAgentSessionID != "" {
		maSessionID = *session.MedAgentSessionID
	} else {
		// Build patient profile for medAgent
		patient, err := s.patientRepo.FindByID(ctx, session.PatientID)
		if err != nil {
			return fmt.Errorf("find patient: %w", err)
		}

		profile := map[string]interface{}{
			"age":       patient.Age,
			"gender":    patient.Gender,
			"allergies": patient.Allergies,
		}

		// Create medAgent session only on first invocation
		maSessionID, err = s.maClient.CreateSession(ctx, profile, session.EntryType == "new", nil)
		if err != nil {
			return fmt.Errorf("create medagent session: %w", err)
		}

		// Persist the medAgent session ID for subsequent calls
		session.MedAgentSessionID = &maSessionID
		session.UpdatedAt = time.Now()
		if err := s.visitRepo.Update(ctx, session); err != nil {
			return fmt.Errorf("persist medagent session id: %w", err)
		}
	}

	// Send patient message to medAgent
	step, err := s.maClient.PatientSay(ctx, maSessionID, lastMessage)
	if err != nil {
		return fmt.Errorf("patient say: %w", err)
	}

	// Process the step and stream events
	if err := s.processStep(ctx, input.SessionID, input.RequestID, maSessionID, session, step, callback); err != nil {
		return err
	}

	// Persist session mutations made by the step handler
	if err := s.visitRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("update visit session: %w", err)
	}

	return nil
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
	if err := callback(model.AssistantStreamEvent{
		Type:      "delta",
		SessionID: sessionID,
		RequestID: requestID,
		Content:   content,
	}); err != nil {
		return err
	}

	// Create message timeline item
	msgItem := adapter.BuildMessageTimelineItem(sessionID, "assistant", content)
	if err := s.timelineRepo.Append(ctx, &msgItem); err != nil {
		return fmt.Errorf("append assistant message: %w", err)
	}

	// Send message_final
	if err := callback(model.AssistantStreamEvent{
		Type:             "message_final",
		SessionID:        sessionID,
		RequestID:        requestID,
		MessageFinalItem: &msgItem,
	}); err != nil {
		return err
	}

	// Send state update
	if err := callback(model.AssistantStreamEvent{
		Type:      "state",
		SessionID: sessionID,
		State:     string(model.VisitMachineStateChatting),
		Status:    string(model.VisitStatusChatting),
	}); err != nil {
		return err
	}

	// Send done
	if err := callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	}); err != nil {
		return err
	}

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
	if err := callback(model.AssistantStreamEvent{
		Type:      "card",
		SessionID: sessionID,
		RequestID: requestID,
		Card:      card,
	}); err != nil {
		return err
	}

	// Create timeline item for the card
	tlItem := adapter.BuildFlowCardTimelineItem(sessionID, card)
	if err := s.timelineRepo.Append(ctx, &tlItem); err != nil {
		return fmt.Errorf("append card timeline: %w", err)
	}

	if err := callback(model.AssistantStreamEvent{
		Type:             "card",
		SessionID:        sessionID,
		RequestID:        requestID,
		Card:             card,
		CardTimelineItem: &tlItem,
	}); err != nil {
		return err
	}

	// Update state to labDecision (blocked)
	newState := string(model.VisitMachineStateLabDecision)
	status := string(model.VisitStatusBlocked)
	cardID := card.ID
	session.Status = status
	session.ActiveCardID = &cardID

	if err := callback(model.AssistantStreamEvent{
		Type:         "state",
		SessionID:    sessionID,
		State:        newState,
		Status:       status,
		ActiveCardID: &cardID,
	}); err != nil {
		return err
	}

	if err := callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleDrugQuery(ctx context.Context, sessionID, requestID, maSessionID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Send state event (drug query is transparent to user)
	if err := callback(model.AssistantStreamEvent{
		Type:      "state",
		SessionID: sessionID,
		State:     string(model.VisitMachineStateAnalyzing),
		Status:    string(model.VisitStatusAnalyzing),
	}); err != nil {
		return err
	}

	// Auto-fill drug info with mock data and continue
	infos := make([]medagent.DrugInfo, len(step.DrugNames))
	for i, name := range step.DrugNames {
		infos[i] = medagent.DrugInfo{
			Name: name,
			Spec: fmt.Sprintf("每盒24粒×0.3g (%s)", name),
		}
	}

	// Send drug info to medAgent
	nextStep, err := s.maClient.DrugInfo(ctx, maSessionID, infos)
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
	if err := callback(model.AssistantStreamEvent{
		Type:      "card",
		SessionID: sessionID,
		RequestID: requestID,
		Card:      card,
	}); err != nil {
		return err
	}

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

	if err := callback(model.AssistantStreamEvent{
		Type:         "state",
		SessionID:    sessionID,
		State:        newState,
		Status:       status,
		ActiveCardID: &cardID,
	}); err != nil {
		return err
	}

	if err := callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleEmergency(ctx context.Context, sessionID, requestID string, session *model.VisitSession, step *medagent.Step, callback StreamAssistantEventCallback) error {
	// Send emergency event
	if err := callback(model.AssistantStreamEvent{
		Type:      "emergency",
		SessionID: sessionID,
		Severity:  string(model.EmergencySeverityCritical),
		Message:   step.Emergency,
	}); err != nil {
		return err
	}

	// Send done event
	if err := callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	}); err != nil {
		return err
	}

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
	if err := callback(model.AssistantStreamEvent{
		Type:      "card",
		SessionID: sessionID,
		RequestID: requestID,
		Card:      diagCard,
	}); err != nil {
		return err
	}
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
		if err := callback(model.AssistantStreamEvent{
			Type:      "state",
			SessionID: sessionID,
			State:     string(model.VisitMachineStateDiagnosis),
			Status:    string(model.VisitStatusDiagnosis),
		}); err != nil {
			return err
		}

	case "ADVICE_ONLY":
		// Create advice_only card
		adviceCard := adapter.BuildAdviceOnlyCard(sessionID, result)
		if err := s.flowCardRepo.Create(ctx, adviceCard); err != nil {
			return fmt.Errorf("create advice card: %w", err)
		}
		if err := callback(model.AssistantStreamEvent{
			Type:      "card",
			SessionID: sessionID,
			RequestID: requestID,
			Card:      adviceCard,
		}); err != nil {
			return err
		}
		adviceTL := adapter.BuildFlowCardTimelineItem(sessionID, adviceCard)
		_ = s.timelineRepo.Append(ctx, &adviceTL)

		cardID := adviceCard.ID
		session.ActiveCardID = &cardID

		if err := callback(model.AssistantStreamEvent{
			Type:         "state",
			SessionID:    sessionID,
			State:        string(model.VisitMachineStateAdviceOnly),
			Status:       string(model.VisitStatusBlocked),
			ActiveCardID: &cardID,
		}); err != nil {
			return err
		}

	case "REFERRAL":
		// Create completed visit card with referral
		completedCard := adapter.BuildCompletedVisitCard(sessionID, result)
		if err := s.flowCardRepo.Create(ctx, completedCard); err != nil {
			return fmt.Errorf("create completed card: %w", err)
		}
		if err := callback(model.AssistantStreamEvent{
			Type:      "card",
			SessionID: sessionID,
			RequestID: requestID,
			Card:      completedCard,
		}); err != nil {
			return err
		}
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

		ts := result.Plan
		session.Summary.TreatmentSummary = &ts

		if err := callback(model.AssistantStreamEvent{
			Type:      "state",
			SessionID: sessionID,
			State:     string(model.VisitMachineStateTransferred),
			Status:    string(model.VisitStatusTransferred),
		}); err != nil {
			return err
		}
	}

	// Set diagnosis in session summary
	if result.Diagnosis != nil {
		diag := result.Diagnosis.Name
		session.Summary.Diagnosis = &diag
	}

	// Send done
	if err := callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleOK(ctx context.Context, sessionID, requestID string, session *model.VisitSession, callback StreamAssistantEventCallback) error {
	// OK is a confirmation step — per spec, no SSE state events are emitted.
	if err := callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	}); err != nil {
		return err
	}

	return nil
}

// Helper to stringify interface for medAgent messages

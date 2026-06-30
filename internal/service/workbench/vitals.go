package workbench

import (
	"context"
	"strings"
	"time"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// emergencySymptomKeywords are symptom keywords that indicate potential emergency.
var emergencySymptomKeywords = []string{
	"胸痛", "胸闷", "呼吸困难", "意识模糊", "昏迷", "大出血", "抽搐", "窒息",
}

// ReportVitalsInput is the input for reporting vital signs.
type ReportVitalsInput struct {
	SessionID string
	Source    string
	Vitals    map[string]interface{}
	Symptoms  []string
}

// EmergencyRecheckResult is the result of an emergency check.
type EmergencyRecheckResult struct {
	Emergency bool   `json:"emergency"`
	Severity  string `json:"severity,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ReportVitals processes vital signs and checks for emergency.
func (s *Service) ReportVitals(ctx context.Context, input ReportVitalsInput) (*EmergencyRecheckResult, error) {
	// Simple emergency detection logic
	emergency := false
	severity := ""

	if hr, ok := input.Vitals["heartRate"].(float64); ok {
		if hr > 120 || hr < 40 {
			emergency = true
			severity = string(model.EmergencySeverityCritical)
		}
	}
	if spo2, ok := input.Vitals["spo2"].(float64); ok {
		if spo2 < 90 {
			emergency = true
			severity = string(model.EmergencySeverityCritical)
		}
	}
	if temp, ok := input.Vitals["temperature"].(float64); ok {
		if temp > 41 || temp < 35 {
			emergency = true
			severity = string(model.EmergencySeveritySuspected)
		}
	}

	// Symptom-based emergency detection
	for _, symptom := range input.Symptoms {
		lower := strings.ToLower(symptom)
		for _, keyword := range emergencySymptomKeywords {
			if strings.Contains(lower, keyword) {
				emergency = true
				if severity == "" {
					severity = string(model.EmergencySeveritySuspected)
				}
				break
			}
		}
		if emergency {
			break
		}
	}

	result := &EmergencyRecheckResult{
		Emergency: emergency,
		Severity:  severity,
	}

	if emergency {
		result.Message = "检测到紧急体征，建议立即转急诊处理"

		// Create emergency timeline event
		emergencyTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
			"vitals_emergency",
			"体征异常",
			result.Message,
		)
		_ = s.timelineRepo.Append(ctx, &emergencyTL)

		// Check if we should terminate
		if severity == string(model.EmergencySeverityCritical) {
			session, err := s.visitRepo.FindByID(ctx, input.SessionID)
			if err == nil {
				now := time.Now()
				reason := string(model.TerminalReasonEmergency)
				session.Status = string(model.VisitStatusEmergencyTerminated)
				session.MachineState = string(model.VisitMachineStateEmergencyPending)
				session.EndedAt = &now
				session.TerminalReason = &reason
				_ = s.visitRepo.Update(ctx, session)

				termTL := adapter.BuildTerminalTimelineItem(input.SessionID,
					string(model.TerminalReasonEmergency),
					"急症终止",
					result.Message,
				)
				_ = s.timelineRepo.Append(ctx, &termTL)
			}
		}

	} else {
		result.Message = "体征正常"
	}

	return result, nil
}

// DismissEmergencyInput is the input for dismissing an emergency.
type DismissEmergencyInput struct {
	SessionID string
}

// DismissEmergency dismisses a false emergency alarm.
func (s *Service) DismissEmergency(ctx context.Context, input DismissEmergencyInput) (*model.VisitSession, *model.TimelineItem, error) {
	session, err := s.visitRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, nil, err
	}

	// Only can dismiss if in emergency state
	if session.Status != string(model.VisitStatusEmergencyTerminated) {
		return nil, nil, model.ErrValidation
	}

	// Recover to chatting
	session.Status = string(model.VisitStatusChatting)
	session.MachineState = string(model.VisitMachineStateChatting)
	session.TerminalReason = nil
	session.EndedAt = nil
	_ = s.visitRepo.Update(ctx, session)

	// Create dismissal timeline event
	dismissTL := adapter.BuildSystemEventTimelineItem(input.SessionID,
		string(model.SystemEventTypeEmergencyDismissed),
		"急症已解除",
		"误报申诉通过，急症态已解除",
	)
	_ = s.timelineRepo.Append(ctx, &dismissTL)

	return session, &dismissTL, nil
}

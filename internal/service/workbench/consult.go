package workbench

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// ClassifyIntentInput is the input for classifying user intent.
type ClassifyIntentInput struct {
	SessionID string
	Content   string
}

// ClassifyIntentResult is the result of intent classification.
type ClassifyIntentResult struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
}

// ClassifyIntent classifies the user's intent in a completed visit.
func (s *Service) ClassifyIntent(ctx context.Context, input ClassifyIntentInput) (*ClassifyIntentResult, error) {
	// Simple keyword-based classification
	content := input.Content

	var intent string
	var confidence float64
	reason := ""

	if containsAny(content, []string{"复查", "复诊", "再看", "还有症状", "没好"}) {
		intent = "follow_up"
		confidence = 0.85
		reason = "检测到复查意向"
	} else if containsAny(content, []string{"咨询", "问问", "了解", "是什么", "为什么", "怎么"}) {
		intent = "consultation"
		confidence = 0.8
		reason = "检测到咨询意向"
	} else {
		// Confidence below threshold — return uncertain per spec
		intent = "uncertain"
		confidence = 0.3
		reason = "无法确定意图"
	}

	return &ClassifyIntentResult{
		Intent:     intent,
		Confidence: math.Round(confidence*100) / 100,
		Reason:     reason,
	}, nil
}

// StreamConsultationReply streams a consultation reply for completed visits.
func (s *Service) StreamConsultationReply(ctx context.Context, sessionID, content, requestID string, callback func(model.AssistantStreamEvent) error) error {
	// For completed visits, provide simple consultation reply
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}

	reply := fmt.Sprintf("关于您的问题「%s」，根据本次就诊记录，建议您咨询专业医生获取更详细的解答。", content)

	if session.Summary.Diagnosis != nil {
		reply = fmt.Sprintf("根据您本次的诊断「%s」，关于「%s」的问题，建议：1. 遵医嘱按时服药 2. 注意休息 3. 如有不适及时复诊",
			*session.Summary.Diagnosis, content)
	}

	// Stream delta
	_ = callback(model.AssistantStreamEvent{
		Type:      "delta",
		SessionID: sessionID,
		RequestID: requestID,
		Content:   reply,
	})

	// Create message item
	msgItem := adapter.BuildMessageTimelineItem(sessionID, "assistant", reply)
	_ = s.timelineRepo.Append(ctx, &msgItem)

	// Message final
	_ = callback(model.AssistantStreamEvent{
		Type:             "message_final",
		SessionID:        sessionID,
		RequestID:        requestID,
		MessageFinalItem: &msgItem,
	})

	// Done
	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	return nil
}

// AskLockedQuestion streams a reply for a locked-card question.
func (s *Service) AskLockedQuestion(ctx context.Context, sessionID, cardID, content, requestID string, callback func(model.AssistantStreamEvent) error) error {
	card, err := s.flowCardRepo.FindByID(ctx, cardID)
	if err != nil {
		return err
	}

	reply := fmt.Sprintf("关于流程卡「%s」中您的问题「%s」，这是一项常规医疗流程，建议您按照提示完成操作。如有疑问，请咨询现场医务人员。",
		card.Title, content)

	// Stream delta
	_ = callback(model.AssistantStreamEvent{
		Type:      "delta",
		SessionID: sessionID,
		RequestID: requestID,
		Content:   reply,
	})

	msgItem := adapter.BuildMessageTimelineItem(sessionID, "assistant", reply)
	_ = s.timelineRepo.Append(ctx, &msgItem)

	_ = callback(model.AssistantStreamEvent{
		Type:             "message_final",
		SessionID:        sessionID,
		RequestID:        requestID,
		MessageFinalItem: &msgItem,
	})

	_ = callback(model.AssistantStreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RequestID: requestID,
	})

	return nil
}

func containsAny(s string, keywords []string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

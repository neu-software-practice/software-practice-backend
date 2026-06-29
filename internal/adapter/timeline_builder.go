package adapter

import (
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// BuildMessageTimelineItem creates a message TimelineItem.
func BuildMessageTimelineItem(sessionID, role, content string) model.TimelineItem {
	return model.TimelineItem{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.TimelineItemKindMessage),
		Status:    string(model.TimelineItemStatusDone),
		CreatedAt: time.Now(),
		Role:      role,
		Content:   content,
	}
}

// BuildFlowCardTimelineItem creates a flow_card TimelineItem.
func BuildFlowCardTimelineItem(sessionID string, card *model.FlowCard) model.TimelineItem {
	return model.TimelineItem{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.TimelineItemKindFlowCard),
		Status:    string(model.TimelineItemStatusDone),
		CreatedAt: time.Now(),
		Card:      card,
	}
}

// BuildSystemEventTimelineItem creates a system_event TimelineItem.
func BuildSystemEventTimelineItem(sessionID, eventType, title, description string) model.TimelineItem {
	return model.TimelineItem{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Kind:        string(model.TimelineItemKindSystemEvent),
		Status:      string(model.TimelineItemStatusDone),
		CreatedAt:   time.Now(),
		EventType:   eventType,
		Title:       title,
		Description: description,
	}
}

// BuildTerminalTimelineItem creates a terminal TimelineItem.
func BuildTerminalTimelineItem(sessionID, reason, title, description string) model.TimelineItem {
	terminationReason := reason
	return model.TimelineItem{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Kind:        string(model.TimelineItemKindTerminal),
		Status:      string(model.TimelineItemStatusDone),
		CreatedAt:   time.Now(),
		Reason:      &terminationReason,
		Title:       title,
		Description: description,
	}
}

// BuildInitialTimeline creates the initial timeline items for a new session.
func BuildInitialTimeline(sessionID, chiefComplaint string) []model.TimelineItem {
	items := []model.TimelineItem{
		BuildSystemEventTimelineItem(sessionID,
			string(model.SystemEventTypeContextLoaded),
			"上下文加载完成",
			"已加载患者信息和就诊上下文",
		),
	}

	if chiefComplaint != "" {
		items = append(items, BuildMessageTimelineItem(sessionID, "patient", chiefComplaint))
	}

	return items
}

// BuildTimelineFromRecord creates TimelineItems from a medAgent SessionRecord.
func BuildTimelineFromRecord(sessionID string, record *medagent.SessionRecord) []model.TimelineItem {
	var items []model.TimelineItem

	for _, turn := range record.Turns {
		item := model.TimelineItem{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Kind:      string(model.TimelineItemKindMessage),
			Status:    string(model.TimelineItemStatusDone),
			CreatedAt: turn.At,
		}

		switch turn.Kind {
		case "patient":
			item.Role = "patient"
			item.Content = turn.Text
		case "doctor":
			item.Role = "assistant"
			item.Content = turn.Text
		case "test_request", "test_result", "drug_query", "drug_info",
			"purchase_request", "purchase_result":
			item.Kind = string(model.TimelineItemKindSystemEvent)
			item.EventType = turn.Kind
			item.Title = turn.Text
		case "advice":
			item.Kind = string(model.TimelineItemKindSystemEvent)
			item.EventType = "advice"
			item.Title = turn.Text
		case "emergency":
			item.Kind = string(model.TimelineItemKindTerminal)
			reason := "emergency"
			item.Reason = &reason
			item.Title = "急症"
			item.Description = turn.Text
		default:
			item.Kind = string(model.TimelineItemKindSystemEvent)
			item.EventType = turn.Kind
			item.Title = turn.Text
		}

		items = append(items, item)
	}

	return items
}

package model

// SSEEventError is the structured error payload carried by error-type SSE events.
type SSEEventError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Status    int         `json:"status,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Retriable bool        `json:"retriable,omitempty"`
}

// AssistantStreamEvent is a discriminated union over its Type field.
//
// Valid Type values: delta, message_final, card, state, emergency, done, error.
// Only the fields relevant to the current Type are populated; all others are
// left at their zero value.
type AssistantStreamEvent struct {
	Type             string         `json:"type"`
	SessionID        string         `json:"sessionId,omitempty"`
	RequestID        string         `json:"requestId,omitempty"`
	Content          string         `json:"content,omitempty"`
	MessageFinalItem *TimelineItem  `json:"item,omitempty"`
	Card             *FlowCard      `json:"card,omitempty"`
	CardTimelineItem *TimelineItem  `json:"timelineItem,omitempty"`
	State            string         `json:"state,omitempty"`
	Status           string         `json:"status,omitempty"`
	ActiveCardID     *string        `json:"activeCardId,omitempty"`
	Severity         string         `json:"severity,omitempty"`
	Message          string         `json:"message,omitempty"`
	Error            *SSEEventError `json:"error,omitempty"`
}

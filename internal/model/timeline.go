package model

import "time"

// TimelineItem is a discriminated union based on the Kind field.
//
// Common fields (ID, SessionID, Kind, Status, CreatedAt) are always present.
// The remaining fields are kind-specific and tagged with omitempty so they
// are omitted from JSON when empty.
type TimelineItem struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`

	// message kind fields
	Role          string  `json:"role,omitempty"`
	Content       string  `json:"content,omitempty"`
	LocalKey      *string `json:"localKey,omitempty"`
	InterruptedBy *string `json:"interruptedBy,omitempty"`

	// flow_card kind fields
	Card *FlowCard `json:"card,omitempty"`

	// system_event kind fields
	EventType   string `json:"eventType,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// terminal kind fields (also uses Title and Description)
	Reason              *string `json:"reason,omitempty"`
	SuggestedDepartment *string `json:"suggestedDepartment,omitempty"`
}

// TimelineContent is the JSON subset stored in the content column.
// It excludes fields already stored as dedicated SQL columns (ID, SessionID, Kind, Status, CreatedAt)
// to avoid redundant storage.
type TimelineContent struct {
	Role                string    `json:"role,omitempty"`
	Content             string    `json:"content,omitempty"`
	LocalKey            *string   `json:"localKey,omitempty"`
	InterruptedBy       *string   `json:"interruptedBy,omitempty"`
	Card                *FlowCard `json:"card,omitempty"`
	EventType           string    `json:"eventType,omitempty"`
	Title               string    `json:"title,omitempty"`
	Description         string    `json:"description,omitempty"`
	Reason              *string   `json:"reason,omitempty"`
	SuggestedDepartment *string   `json:"suggestedDepartment,omitempty"`
}

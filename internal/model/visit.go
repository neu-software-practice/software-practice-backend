package model

import (
	"errors"
	"time"
)

// VisitSession represents a complete visit session entity.
type VisitSession struct {
	ID                string       `json:"id"`
	PatientID         string       `json:"patientId"`
	EntryType         string       `json:"entryType"`
	Status            string       `json:"status"`
	MachineState      string       `json:"-"`
	StartedAt         time.Time    `json:"startedAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	EndedAt           *time.Time   `json:"endedAt,omitempty"`
	TimeoutAt         *time.Time   `json:"timeoutAt,omitempty"`
	PausedAt          *time.Time   `json:"pausedAt,omitempty"`
	LastActivityAt    *time.Time   `json:"lastActivityAt,omitempty"`
	AskRound          int          `json:"askRound"`
	AskRoundLimit     int          `json:"askRoundLimit"`
	LabRound          int          `json:"labRound"`
	LabRoundLimit     int          `json:"labRoundLimit"`
	ParentSessionID   *string      `json:"parentSessionId,omitempty"`
	TerminalReason    *string      `json:"terminalReason,omitempty"`
	ActiveCardID      *string      `json:"activeCardId,omitempty"`
	MedAgentSessionID *string      `json:"-"`
	TimerPaused       bool         `json:"timerPaused"`
	Summary           VisitSummary `json:"summary"`
}

// VisitSummary contains optional summary fields for a visit session.
type VisitSummary struct {
	Title            *string `json:"title,omitempty"`
	ChiefComplaint   *string `json:"chiefComplaint,omitempty"`
	Diagnosis        *string `json:"diagnosis,omitempty"`
	TreatmentSummary *string `json:"treatmentSummary,omitempty"`
	LastMessage      *string `json:"lastMessage,omitempty"`
}

// VisitSessionSummary is a lightweight summary representation of a visit session for list views.
type VisitSessionSummary struct {
	ID              string       `json:"id"`
	PatientID       string       `json:"patientId"`
	EntryType       string       `json:"entryType"`
	Status          string       `json:"status"`
	StartedAt       time.Time    `json:"startedAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	LastActivityAt  *time.Time   `json:"lastActivityAt,omitempty"`
	EndedAt         *time.Time   `json:"endedAt,omitempty"`
	ParentSessionID *string      `json:"parentSessionId,omitempty"`
	TerminalReason  *string      `json:"terminalReason,omitempty"`
	Summary         VisitSummary `json:"summary"`
}

// VisitSnapshot is a read-only snapshot of a complete visit.
type VisitSnapshot struct {
	Session        VisitSession   `json:"session"`
	Timeline       []TimelineItem `json:"timeline"`
	Readonly       bool           `json:"readonly,omitempty"`
	TerminalReason *string        `json:"terminalReason,omitempty"`
}

// CreateSessionInput is the request body for creating a new visit session.
type CreateSessionInput struct {
	PatientID      string `json:"patientId"`
	EntryType      string `json:"entryType"`
	ChiefComplaint string `json:"chiefComplaint,omitempty"`
}

// Validate checks that CreateSessionInput has valid fields.
// A new-session input must use entry type "new"; parent-related fields
// are not applicable here.
func (c CreateSessionInput) Validate() error {
	if c.EntryType != "new" {
		return errors.New("entry type must be 'new' for a new session")
	}
	return nil
}

// CreateFollowUpInput is the request body for creating a follow-up visit.
type CreateFollowUpInput struct {
	PatientID       string `json:"patientId"`
	ParentSessionID string `json:"parentSessionId"`
	ChiefComplaint  string `json:"chiefComplaint,omitempty"`
}

// CreateSessionResult is the response for session creation (both new and follow-up).
type CreateSessionResult struct {
	Session         VisitSession   `json:"session"`
	InitialTimeline []TimelineItem `json:"initialTimeline"`
}

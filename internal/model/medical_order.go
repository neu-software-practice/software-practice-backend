package model

import "github.com/neuhis/software-practice-backend/pkg/api"

// MedicalOrderKind enumerates the kind of medical order record.
type MedicalOrderKind string

const (
	MedicalOrderKindAdvice     MedicalOrderKind = "advice"
	MedicalOrderKindMedication MedicalOrderKind = "medication"
)

// MedicalOrderRecord represents a single medical order record aggregated from
// completed advice_only and medication_fulfillment flow cards.
// Kind-specific fields use omitempty so only relevant fields serialize for each kind.
type MedicalOrderRecord struct {
	RecordID     string `json:"recordId"`
	SessionID    string `json:"sessionId"`
	SessionTitle string `json:"sessionTitle"`
	Kind         string `json:"kind"`

	// --- advice-only fields ---
	Advices                []string `json:"advices,omitempty"`
	WatchItems             []string `json:"watchItems,omitempty"`
	FollowUpRecommendation string   `json:"followUpRecommendation,omitempty"`

	// --- medication-only fields ---
	Medications       []MedicationItem  `json:"medications,omitempty"`
	FulfillmentStatus FulfillmentStatus `json:"fulfillmentStatus,omitempty"`
	DeliveryAddress   *DeliveryAddress  `json:"deliveryAddress,omitempty"`

	// --- common ---
	HandledAt string `json:"handledAt"`
	CreatedAt string `json:"createdAt"`
}

// MedicalOrdersResponse is the response for GET /medical-orders.
type MedicalOrdersResponse = api.PageResult[MedicalOrderRecord]

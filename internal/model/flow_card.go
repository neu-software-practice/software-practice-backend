package model

import "time"

// TestItem represents a single lab test item requested in a lab_decision card.
type TestItem struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	SampleType string `json:"sampleType,omitempty"`
}

// PaymentLineItem represents a single line in a payment card.
type PaymentLineItem struct {
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Quantity int     `json:"quantity,omitempty"`
}

// DeliveryAddress holds the delivery address summary stored on a medication_fulfillment card.
type DeliveryAddress struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	FullAddress string `json:"fullAddress"`
}

// MedicationItem represents a single medication prescribed in a medication_fulfillment card.
type MedicationItem struct {
	Name     string  `json:"name"`
	Spec     string  `json:"spec"`
	Quantity int     `json:"quantity"`
	Dosage   string  `json:"dosage"`
	Days     int     `json:"days"`
	Price    float64 `json:"price"`
}

// FlowCard is a discriminated union identified by the Kind field.
// Each supported Kind variant is listed in the FlowCardKind constants.
// Type-specific fields use omitempty so that only fields relevant to the
// current Kind are serialized.
type FlowCard struct {
	// --- common base fields ---
	ID         string     `json:"id"`
	SessionID  string     `json:"sessionId"`
	Kind       string     `json:"kind"`
	Status     string     `json:"status"`
	Blocking   bool       `json:"blocking"`
	Title      string     `json:"title"`
	CreatedAt  time.Time  `json:"createdAt"`
	HandledAt  *time.Time `json:"handledAt,omitempty"`
	LockReason *string    `json:"lockReason,omitempty"`

	// --- lab_decision ---
	TestItems           []TestItem `json:"testItems,omitempty"`
	Reason              string     `json:"reason,omitempty"`
	DifferentialTargets []string   `json:"differentialTargets,omitempty"`
	EstimatedFee        *float64   `json:"estimatedFee,omitempty"`

	// --- payment ---
	PaymentID       string            `json:"paymentId,omitempty"`
	Purpose         string            `json:"purpose,omitempty"`
	Items           []PaymentLineItem `json:"items,omitempty"`
	TotalAmount     *float64          `json:"totalAmount,omitempty"`
	InsuranceAmount *float64          `json:"insuranceAmount,omitempty"`
	SelfPayAmount   *float64          `json:"selfPayAmount,omitempty"`
	PaymentStatus   string            `json:"paymentStatus,omitempty"`

	// --- lab_execution ---
	LabOrderID       string     `json:"labOrderId,omitempty"`
	ExecutionStatus  string     `json:"executionStatus,omitempty"`
	ResultSummary    *string    `json:"resultSummary,omitempty"`
	ResultReturnedAt *time.Time `json:"resultReturnedAt,omitempty"`

	// --- diagnosis ---
	Diagnosis       string           `json:"diagnosis,omitempty"`
	Confidence      string           `json:"confidence,omitempty"`
	Evidence        []string         `json:"evidence,omitempty"`
	EvidenceSources []EvidenceSource `json:"evidenceSources,omitempty"`
	RiskSignals     []string         `json:"riskSignals,omitempty"`

	// --- treatment_plan ---
	Plan       string   `json:"plan,omitempty"`
	Capability string   `json:"capability,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Actions    []string `json:"actions,omitempty"`

	// --- medication_fulfillment ---
	Medications       []MedicationItem            `json:"medications,omitempty"`
	AvailableModes    []string                    `json:"availableModes,omitempty"`
	SelectedMode      *string                     `json:"selectedMode,omitempty"`
	FulfillmentStatus MedicationFulfillmentStatus `json:"fulfillmentStatus,omitempty"`
	DeliveryAddress   *DeliveryAddress            `json:"deliveryAddress,omitempty"`

	// --- treatment_execution ---
	TreatmentName    string     `json:"treatmentName,omitempty"`
	AppointmentAt    *time.Time `json:"appointmentAt,omitempty"`
	QueueNo          *string    `json:"queueNo,omitempty"`
	Notices          []string   `json:"notices,omitempty"`
	AvailableActions []string   `json:"availableActions,omitempty"`

	// --- advice_only ---
	Advices                []string `json:"advices,omitempty"`
	WatchItems             []string `json:"watchItems,omitempty"`
	FollowUpRecommendation string   `json:"followUpRecommendation,omitempty"`

	// --- completed_visit ---
	TreatmentSummary   string    `json:"treatmentSummary,omitempty"`
	FollowUpSuggestion string    `json:"followUpSuggestion,omitempty"`
	CompletedAt        time.Time `json:"completedAt,omitempty"`
}

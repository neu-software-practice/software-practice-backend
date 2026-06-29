package medagent

import "time"

// These types mirror medAgent/agent/types.go for the HTTP client.

// StepKind represents the type of a medAgent step.
type StepKind string

const (
	StepAsk       StepKind = "ASK"
	StepNeedTests StepKind = "NEED_TESTS"
	StepPurchase  StepKind = "PURCHASE"
	StepDrugQuery StepKind = "DRUG_QUERY"
	StepEmergency StepKind = "EMERGENCY"
	StepDone      StepKind = "DONE"
	StepOK        StepKind = "OK"
)

// Step represents a medAgent next-step instruction.
type Step struct {
	Kind      StepKind    `json:"kind"`
	DoctorSay string      `json:"doctor_say,omitempty"`
	TestItems []string    `json:"test_items,omitempty"`
	DrugNames []string    `json:"drug_names,omitempty"`
	Orders    []DrugOrder `json:"orders,omitempty"`
	Emergency string      `json:"emergency,omitempty"`
	Result    *Result     `json:"result,omitempty"`
}

// Result is the diagnosis/treatment result from medAgent.
type Result struct {
	Final       string       `json:"final"`
	Diagnosis   *Diagnosis   `json:"diagnosis,omitempty"`
	Plan        string       `json:"plan"`
	Medications []Medication `json:"medications,omitempty"`
	Advice      string       `json:"advice"`
}

// Diagnosis is the diagnostic conclusion.
type Diagnosis struct {
	Name       string  `json:"name"`
	Basis      string  `json:"basis"`
	Confidence float64 `json:"confidence"`
}

// Medication is a prescribed medication.
type Medication struct {
	Name     string `json:"name"`
	Dosage   string `json:"dosage"`
	Schedule string `json:"schedule"`
	Quantity int    `json:"quantity"`
}

// DrugOrder is a medication purchase order.
type DrugOrder struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// DrugPurchase is a purchase result.
type DrugPurchase struct {
	Name     string `json:"name"`
	Bought   bool   `json:"bought"`
	Quantity int    `json:"quantity"`
}

// TestResult is a lab test result item.
type TestResult struct {
	Item  string `json:"item"`
	Value string `json:"value"`
}

// DrugInfo is drug specification information.
type DrugInfo struct {
	Name string `json:"name"`
	Spec string `json:"spec"`
}

// SessionRecord is the full session record from medAgent.
type SessionRecord struct {
	SessionID string         `json:"session_id"`
	Initial   bool           `json:"initial"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"`
	Profile   interface{}    `json:"profile,omitempty"`
	Turns     []RecordedTurn `json:"turns"`
	Outcome   *Result        `json:"outcome,omitempty"`
}

// RecordedTurn is a single turn in the session record.
type RecordedTurn struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"`
	Text string    `json:"text"`
}

// Package model defines domain entities, enums, DTOs, and business error sentinels.
package model

// VisitStatus represents the external-facing visit session status.
type VisitStatus string

const (
	VisitStatusLoadingContext      VisitStatus = "loading_context"
	VisitStatusChatting            VisitStatus = "chatting"
	VisitStatusAnalyzing           VisitStatus = "analyzing"
	VisitStatusBlocked             VisitStatus = "blocked"
	VisitStatusDiagnosis           VisitStatus = "diagnosis"
	VisitStatusTreatment           VisitStatus = "treatment"
	VisitStatusCompleted           VisitStatus = "completed"
	VisitStatusSuspended           VisitStatus = "suspended"
	VisitStatusTransferred         VisitStatus = "transferred"
	VisitStatusEmergencyTerminated VisitStatus = "emergency_terminated"
	VisitStatusExited              VisitStatus = "exited"
)

// IsValidVisitStatus returns true if the given status string is a valid VisitStatus value.
func IsValidVisitStatus(status string) bool {
	switch VisitStatus(status) {
	case VisitStatusLoadingContext, VisitStatusChatting, VisitStatusAnalyzing,
		VisitStatusBlocked, VisitStatusDiagnosis, VisitStatusTreatment,
		VisitStatusCompleted, VisitStatusSuspended, VisitStatusTransferred,
		VisitStatusEmergencyTerminated, VisitStatusExited:
		return true
	}
	return false
}

// VisitMachineState represents the internal state machine state of a visit.
type VisitMachineState string

const (
	VisitMachineStateLoadingContext        VisitMachineState = "loadingContext"
	VisitMachineStateChatting              VisitMachineState = "chatting"
	VisitMachineStateAnalyzing             VisitMachineState = "analyzing"
	VisitMachineStateLabDecision           VisitMachineState = "labDecision"
	VisitMachineStateLabPayment            VisitMachineState = "labPayment"
	VisitMachineStateLabExecution          VisitMachineState = "labExecution"
	VisitMachineStateDiagnosis             VisitMachineState = "diagnosis"
	VisitMachineStateTreatmentDecision     VisitMachineState = "treatmentDecision"
	VisitMachineStateMedicationPayment     VisitMachineState = "medicationPayment"
	VisitMachineStateMedicationFulfillment VisitMachineState = "medicationFulfillment"
	VisitMachineStateTreatmentExecution    VisitMachineState = "treatmentExecution"
	VisitMachineStateAdviceOnly            VisitMachineState = "adviceOnly"
	VisitMachineStateCompleted             VisitMachineState = "completed"
	VisitMachineStateSuspended             VisitMachineState = "suspended"
	VisitMachineStateEmergencyPending      VisitMachineState = "emergencyPending"
	VisitMachineStateTerminated            VisitMachineState = "terminated"
	VisitMachineStateExitSettlement        VisitMachineState = "exitSettlement"
	VisitMachineStateExited                VisitMachineState = "exited"
	VisitMachineStateTransferred           VisitMachineState = "transferred"
)

// TerminalReason describes why a visit was terminated.
type TerminalReason string

const (
	TerminalReasonEmergency              TerminalReason = "emergency"
	TerminalReasonTimeout                TerminalReason = "timeout"
	TerminalReasonAskLimitReached        TerminalReason = "ask_limit_reached"
	TerminalReasonLabLimitReached        TerminalReason = "lab_limit_reached"
	TerminalReasonReferral               TerminalReason = "referral"
	TerminalReasonCapabilityInsufficient TerminalReason = "capability_insufficient"
	TerminalReasonExited                 TerminalReason = "exited"
)

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	PaymentStatusUnpaid   PaymentStatus = "unpaid"
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// VisitEntryType indicates how a visit was initiated.
type VisitEntryType string

const (
	VisitEntryTypeNew      VisitEntryType = "new"
	VisitEntryTypeFollowUp VisitEntryType = "follow_up"
)

// FlowCardKind categorises the type of a flow card.
type FlowCardKind string

const (
	FlowCardKindLabDecision           FlowCardKind = "lab_decision"
	FlowCardKindPayment               FlowCardKind = "payment"
	FlowCardKindLabExecution          FlowCardKind = "lab_execution"
	FlowCardKindDiagnosis             FlowCardKind = "diagnosis"
	FlowCardKindTreatmentPlan         FlowCardKind = "treatment_plan"
	FlowCardKindMedicationFulfillment FlowCardKind = "medication_fulfillment"
	FlowCardKindTreatmentExecution    FlowCardKind = "treatment_execution"
	FlowCardKindAdviceOnly            FlowCardKind = "advice_only"
	FlowCardKindCompletedVisit        FlowCardKind = "completed_visit"
)

// FlowCardStatus represents the lifecycle status of a flow card.
type FlowCardStatus string

const (
	FlowCardStatusPending     FlowCardStatus = "pending"
	FlowCardStatusAccepted    FlowCardStatus = "accepted"
	FlowCardStatusSkipped     FlowCardStatus = "skipped"
	FlowCardStatusVetoed      FlowCardStatus = "vetoed"
	FlowCardStatusPaid        FlowCardStatus = "paid"
	FlowCardStatusProcessing  FlowCardStatus = "processing"
	FlowCardStatusCompleted   FlowCardStatus = "completed"
	FlowCardStatusFailed      FlowCardStatus = "failed"
	FlowCardStatusInvalidated FlowCardStatus = "invalidated"
)

// TimelineItemKind categorises a timeline entry.
type TimelineItemKind string

const (
	TimelineItemKindMessage     TimelineItemKind = "message"
	TimelineItemKindFlowCard    TimelineItemKind = "flow_card"
	TimelineItemKindSystemEvent TimelineItemKind = "system_event"
	TimelineItemKindTerminal    TimelineItemKind = "terminal"
)

// TimelineItemStatus represents the processing status of a timeline item.
type TimelineItemStatus string

const (
	TimelineItemStatusPending     TimelineItemStatus = "pending"
	TimelineItemStatusStreaming   TimelineItemStatus = "streaming"
	TimelineItemStatusDone        TimelineItemStatus = "done"
	TimelineItemStatusFailed      TimelineItemStatus = "failed"
	TimelineItemStatusInvalidated TimelineItemStatus = "invalidated"
)

// SystemEventType enumerates known system events that can appear on the timeline.
type SystemEventType string

const (
	SystemEventTypeContextLoaded      SystemEventType = "context_loaded"
	SystemEventTypeAgentThinking      SystemEventType = "agent_thinking"
	SystemEventTypeLabResultReceived  SystemEventType = "lab_result_received"
	SystemEventTypePaymentSucceeded   SystemEventType = "payment_succeeded"
	SystemEventTypeDrugPurchased      SystemEventType = "drug_purchased"
	SystemEventTypeFollowUpStarted    SystemEventType = "follow_up_started"
	SystemEventTypeEmergencyDismissed SystemEventType = "emergency_dismissed"
	SystemEventTypeExitSettled        SystemEventType = "exit_settled"
	SystemEventTypeSessionSuspended   SystemEventType = "session_suspended"
)

// SSEEventType identifies the type of a server-sent event in the assistant stream.
type SSEEventType string

const (
	SSEEventTypeDelta        SSEEventType = "delta"
	SSEEventTypeMessageFinal SSEEventType = "message_final"
	SSEEventTypeCard         SSEEventType = "card"
	SSEEventTypeState        SSEEventType = "state"
	SSEEventTypeEmergency    SSEEventType = "emergency"
	SSEEventTypeDone         SSEEventType = "done"
	SSEEventTypeError        SSEEventType = "error"
)

// CredentialType represents the type of identity credential presented.
type CredentialType string

const (
	CredentialTypeIDCard CredentialType = "id_card"
	CredentialTypePhone  CredentialType = "phone"
)

// ReadableScope identifies a category of patient data the caller may read.
type ReadableScope string

const (
	ReadableScopeProfile     ReadableScope = "profile"
	ReadableScopeHistory     ReadableScope = "history"
	ReadableScopeAllergies   ReadableScope = "allergies"
	ReadableScopeMedications ReadableScope = "medications"
)

// Gender represents the patient's gender.
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderOther   Gender = "other"
	GenderUnknown Gender = "unknown"
)

// MessageRole identifies the sender of a chat message.
type MessageRole string

const (
	MessageRolePatient   MessageRole = "patient"
	MessageRoleAssistant MessageRole = "assistant"
)

// LabDecision represents the patient's decision regarding a lab test proposal.
type LabDecision string

const (
	LabDecisionAccepted LabDecision = "accepted"
	LabDecisionSkipped  LabDecision = "skipped"
	LabDecisionVetoed   LabDecision = "vetoed"
)

// PaymentPurpose indicates what a payment is for.
type PaymentPurpose string

const (
	PaymentPurposeLab        PaymentPurpose = "lab"
	PaymentPurposeMedication PaymentPurpose = "medication"
)

// FulfillmentMode describes how medication is dispensed.
type FulfillmentMode string

const (
	FulfillmentModePickup   FulfillmentMode = "pickup"
	FulfillmentModeDelivery FulfillmentMode = "delivery"
)

// TreatmentAction enumerates the actions available for treatment execution.
type TreatmentAction string

const (
	TreatmentActionSchedule       TreatmentAction = "schedule"
	TreatmentActionConfirmArrival TreatmentAction = "confirm_arrival"
	TreatmentActionStart          TreatmentAction = "start"
	TreatmentActionComplete       TreatmentAction = "complete"
	TreatmentActionCancel         TreatmentAction = "cancel"
)

// ConsultationIntent classifies the intent behind a completed-visit user input.
type ConsultationIntent string

const (
	ConsultationIntentConsultation ConsultationIntent = "consultation"
	ConsultationIntentFollowUp     ConsultationIntent = "follow_up"
	ConsultationIntentUncertain    ConsultationIntent = "uncertain"
)

// EmergencySeverity represents the severity level of an emergency event.
type EmergencySeverity string

const (
	EmergencySeveritySuspected EmergencySeverity = "suspected"
	EmergencySeverityCritical  EmergencySeverity = "critical"
)

// ExitReason describes the reason a patient exits a visit.
type ExitReason string

const (
	ExitReasonPatientRequest ExitReason = "patient_request"
	ExitReasonTimeout        ExitReason = "timeout"
	ExitReasonEmergency      ExitReason = "emergency"
	ExitReasonOther          ExitReason = "other"
)

// ExitConsequenceKind categorises the financial consequence of an exit.
type ExitConsequenceKind string

const (
	ExitConsequenceNoFee               ExitConsequenceKind = "no_fee"
	ExitConsequenceRefundable          ExitConsequenceKind = "refundable"
	ExitConsequenceExecutedNoRefund    ExitConsequenceKind = "executed_no_refund"
	ExitConsequenceMedicationDispensed ExitConsequenceKind = "medication_dispensed"
)

// VitalsSource indicates where vital-sign data originated.
type VitalsSource string

const (
	VitalsSourcePatientReport VitalsSource = "patient_report"
	VitalsSourceDevice        VitalsSource = "device"
	VitalsSourceManual        VitalsSource = "manual"
)

// DiagnosisConfidence expresses the confidence level of a diagnosis.
type DiagnosisConfidence string

const (
	DiagnosisConfidenceLow    DiagnosisConfidence = "low"
	DiagnosisConfidenceMedium DiagnosisConfidence = "medium"
	DiagnosisConfidenceHigh   DiagnosisConfidence = "high"
)

// TreatmentPlan categorises the recommended treatment path.
type TreatmentPlan string

const (
	TreatmentPlanMedication TreatmentPlan = "medication"
	TreatmentPlanTreatment  TreatmentPlan = "treatment"
	TreatmentPlanAdviceOnly TreatmentPlan = "advice_only"
	TreatmentPlanReferral   TreatmentPlan = "referral"
)

// TreatmentCapability describes whether the facility can provide a treatment.
type TreatmentCapability string

const (
	TreatmentCapabilityAvailable   TreatmentCapability = "available"
	TreatmentCapabilityLimited     TreatmentCapability = "limited"
	TreatmentCapabilityUnavailable TreatmentCapability = "unavailable"
)

// ExecutionStatus tracks the lifecycle of a treatment execution.
type ExecutionStatus string

const (
	ExecutionStatusPending    ExecutionStatus = "pending"
	ExecutionStatusScheduled  ExecutionStatus = "scheduled"
	ExecutionStatusArrived    ExecutionStatus = "arrived"
	ExecutionStatusInProgress ExecutionStatus = "in_progress"
	ExecutionStatusCompleted  ExecutionStatus = "completed"
	ExecutionStatusCanceled   ExecutionStatus = "canceled"
)

// LabExecutionStatus tracks the lifecycle of a lab order execution.
type LabExecutionStatus string

const (
	LabExecutionStatusWaitingPayment LabExecutionStatus = "waiting_payment"
	LabExecutionStatusQueued         LabExecutionStatus = "queued"
	LabExecutionStatusCollecting     LabExecutionStatus = "collecting"
	LabExecutionStatusTesting        LabExecutionStatus = "testing"
	LabExecutionStatusResultReady    LabExecutionStatus = "result_ready"
	LabExecutionStatusCompleted      LabExecutionStatus = "completed"
)

// FinalOutcome represents the terminal outcome of a visit from the agent's perspective.
type FinalOutcome string

const (
	FinalOutcomeAdvice   FinalOutcome = "ADVICE"
	FinalOutcomeReferral FinalOutcome = "REFERRAL"
)

// EvidenceSource identifies the origin of a diagnostic evidence item.
type EvidenceSource string

const (
	EvidenceSourceHistory   EvidenceSource = "history"
	EvidenceSourceAnswer    EvidenceSource = "answer"
	EvidenceSourceLabResult EvidenceSource = "lab_result"
)

// InterruptedBy represents the reason an assistant message was interrupted.
type InterruptedBy string

const (
	InterruptedByIdle      InterruptedBy = "idle"
	InterruptedByEmergency InterruptedBy = "emergency"
	InterruptedByTimeout   InterruptedBy = "timeout"
	InterruptedByExit      InterruptedBy = "exit"
)

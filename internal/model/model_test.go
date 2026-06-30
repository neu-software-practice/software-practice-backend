package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

func TestVisitSessionJSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	s := model.VisitSession{
		ID:            "v001",
		PatientID:     "p001",
		EntryType:     "new",
		Status:        "chatting",
		StartedAt:     now,
		UpdatedAt:     now,
		AskRound:      0,
		AskRoundLimit: 20,
		LabRound:      0,
		LabRoundLimit: 10,
		TimerPaused:   false,
		Summary:       model.VisitSummary{},
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["id"] != "v001" {
		t.Errorf("id: got %v, want v001", parsed["id"])
	}
	if parsed["patientId"] != "p001" {
		t.Errorf("patientId: got %v, want p001", parsed["patientId"])
	}
	if parsed["entryType"] != "new" {
		t.Errorf("entryType: got %v, want new", parsed["entryType"])
	}
	if parsed["timerPaused"] != false {
		t.Errorf("timerPaused: got %v, want false", parsed["timerPaused"])
	}
}

func TestPatientProfileJSON(t *testing.T) {
	p := model.PatientProfile{
		ID:                  "p001",
		Name:                "张三",
		Gender:              "male",
		Age:                 35,
		PhoneMasked:         "138****1234",
		Allergies:           []string{"青霉素"},
		ChronicDiseases:     []string{"高血压"},
		LongTermMedications: []string{"硝苯地平"},
		UpdatedAt:           time.Now(),
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)

	if parsed["phoneMasked"] != "138****1234" {
		t.Error("phoneMasked mismatch")
	}
}

func TestTimelineItemMessageJSON(t *testing.T) {
	item := model.TimelineItem{
		ID:        "t001",
		SessionID: "v001",
		Kind:      "message",
		Status:    "done",
		CreatedAt: time.Now(),
		Role:      "patient",
		Content:   "我头疼",
	}

	b, _ := json.Marshal(item)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)

	if parsed["kind"] != "message" {
		t.Error("kind mismatch")
	}
	if parsed["role"] != "patient" {
		t.Error("role mismatch")
	}
	// Verify non-message fields are omitted
	if _, ok := parsed["card"]; ok {
		t.Error("card should be omitted for message kind")
	}
}

func TestTimelineItemFlowCardJSON(t *testing.T) {
	card := &model.FlowCard{
		ID:        "f001",
		SessionID: "v001",
		Kind:      "lab_decision",
		Status:    "pending",
		Blocking:  true,
		Title:     "检验决定",
		CreatedAt: time.Now(),
	}
	item := model.TimelineItem{
		ID:        "t002",
		SessionID: "v001",
		Kind:      "flow_card",
		Status:    "done",
		CreatedAt: time.Now(),
		Card:      card,
	}

	b, _ := json.Marshal(item)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)

	if parsed["kind"] != "flow_card" {
		t.Error("kind mismatch")
	}
}

func TestFlowCardJSON(t *testing.T) {
	card := model.FlowCard{
		ID:        "f001",
		SessionID: "v001",
		Kind:      "lab_decision",
		Status:    "pending",
		Blocking:  true,
		Title:     "检验决定",
		CreatedAt: time.Now(),
		TestItems: []model.TestItem{
			{Code: "blood_rt", Name: "血常规"},
		},
		Reason:       "需要进一步检查",
		EstimatedFee: model.Float64Ptr(50.0),
	}

	b, _ := json.Marshal(card)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)

	if parsed["kind"] != "lab_decision" {
		t.Error("kind mismatch")
	}
	if parsed["blocking"] != true {
		t.Error("blocking should be true")
	}
}

func TestAssistantStreamEventJSON(t *testing.T) {
	event := model.AssistantStreamEvent{
		Type:      "delta",
		SessionID: "v001",
		RequestID: "r001",
		Content:   "您好",
	}

	b, _ := json.Marshal(event)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)

	if parsed["type"] != "delta" {
		t.Error("type mismatch")
	}
	if parsed["content"] != "您好" {
		t.Error("content mismatch")
	}
}

func TestCreateSessionInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   model.CreateSessionInput
		wantErr bool
	}{
		{"valid new", model.CreateSessionInput{PatientID: "p1", EntryType: "new"}, false},
		{"follow_up entry type rejected for new session", model.CreateSessionInput{PatientID: "p1", EntryType: "follow_up"}, true},
		{"empty entry type", model.CreateSessionInput{PatientID: "p1", EntryType: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnumConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		// VisitStatus — 10 values
		{name: "VisitStatusLoadingContext", value: string(model.VisitStatusLoadingContext), expected: "loading_context"},
		{name: "VisitStatusChatting", value: string(model.VisitStatusChatting), expected: "chatting"},
		{name: "VisitStatusAnalyzing", value: string(model.VisitStatusAnalyzing), expected: "analyzing"},
		{name: "VisitStatusBlocked", value: string(model.VisitStatusBlocked), expected: "blocked"},
		{name: "VisitStatusDiagnosis", value: string(model.VisitStatusDiagnosis), expected: "diagnosis"},
		{name: "VisitStatusTreatment", value: string(model.VisitStatusTreatment), expected: "treatment"},
		{name: "VisitStatusCompleted", value: string(model.VisitStatusCompleted), expected: "completed"},
		{name: "VisitStatusTransferred", value: string(model.VisitStatusTransferred), expected: "transferred"},
		{name: "VisitStatusEmergencyTerminated", value: string(model.VisitStatusEmergencyTerminated), expected: "emergency_terminated"},
		{name: "VisitStatusExited", value: string(model.VisitStatusExited), expected: "exited"},

		// VisitMachineState — 17 values
		{name: "VisitMachineStateLoadingContext", value: string(model.VisitMachineStateLoadingContext), expected: "loadingContext"},
		{name: "VisitMachineStateChatting", value: string(model.VisitMachineStateChatting), expected: "chatting"},
		{name: "VisitMachineStateAnalyzing", value: string(model.VisitMachineStateAnalyzing), expected: "analyzing"},
		{name: "VisitMachineStateLabDecision", value: string(model.VisitMachineStateLabDecision), expected: "labDecision"},
		{name: "VisitMachineStateLabPayment", value: string(model.VisitMachineStateLabPayment), expected: "labPayment"},
		{name: "VisitMachineStateLabExecution", value: string(model.VisitMachineStateLabExecution), expected: "labExecution"},
		{name: "VisitMachineStateDiagnosis", value: string(model.VisitMachineStateDiagnosis), expected: "diagnosis"},
		{name: "VisitMachineStateTreatmentDecision", value: string(model.VisitMachineStateTreatmentDecision), expected: "treatmentDecision"},
		{name: "VisitMachineStateMedicationPayment", value: string(model.VisitMachineStateMedicationPayment), expected: "medicationPayment"},
		{name: "VisitMachineStateMedicationFulfillment", value: string(model.VisitMachineStateMedicationFulfillment), expected: "medicationFulfillment"},
		{name: "VisitMachineStateTreatmentExecution", value: string(model.VisitMachineStateTreatmentExecution), expected: "treatmentExecution"},
		{name: "VisitMachineStateAdviceOnly", value: string(model.VisitMachineStateAdviceOnly), expected: "adviceOnly"},
		{name: "VisitMachineStateCompleted", value: string(model.VisitMachineStateCompleted), expected: "completed"},
		{name: "VisitMachineStateEmergencyPending", value: string(model.VisitMachineStateEmergencyPending), expected: "emergencyPending"},
		{name: "VisitMachineStateTerminated", value: string(model.VisitMachineStateTerminated), expected: "terminated"},
		{name: "VisitMachineStateExitSettlement", value: string(model.VisitMachineStateExitSettlement), expected: "exitSettlement"},
		{name: "VisitMachineStateExited", value: string(model.VisitMachineStateExited), expected: "exited"},

		// TerminalReason — 7 values
		{name: "TerminalReasonEmergency", value: string(model.TerminalReasonEmergency), expected: "emergency"},
		{name: "TerminalReasonTimeout", value: string(model.TerminalReasonTimeout), expected: "timeout"},
		{name: "TerminalReasonAskLimitReached", value: string(model.TerminalReasonAskLimitReached), expected: "ask_limit_reached"},
		{name: "TerminalReasonLabLimitReached", value: string(model.TerminalReasonLabLimitReached), expected: "lab_limit_reached"},
		{name: "TerminalReasonReferral", value: string(model.TerminalReasonReferral), expected: "referral"},
		{name: "TerminalReasonCapabilityInsufficient", value: string(model.TerminalReasonCapabilityInsufficient), expected: "capability_insufficient"},
		{name: "TerminalReasonExited", value: string(model.TerminalReasonExited), expected: "exited"},

		// PaymentStatus — 5 values
		{name: "PaymentStatusUnpaid", value: string(model.PaymentStatusUnpaid), expected: "unpaid"},
		{name: "PaymentStatusPending", value: string(model.PaymentStatusPending), expected: "pending"},
		{name: "PaymentStatusPaid", value: string(model.PaymentStatusPaid), expected: "paid"},
		{name: "PaymentStatusFailed", value: string(model.PaymentStatusFailed), expected: "failed"},
		{name: "PaymentStatusRefunded", value: string(model.PaymentStatusRefunded), expected: "refunded"},

		// FlowCardKind — 9 values
		{name: "FlowCardKindLabDecision", value: string(model.FlowCardKindLabDecision), expected: "lab_decision"},
		{name: "FlowCardKindPayment", value: string(model.FlowCardKindPayment), expected: "payment"},
		{name: "FlowCardKindLabExecution", value: string(model.FlowCardKindLabExecution), expected: "lab_execution"},
		{name: "FlowCardKindDiagnosis", value: string(model.FlowCardKindDiagnosis), expected: "diagnosis"},
		{name: "FlowCardKindTreatmentPlan", value: string(model.FlowCardKindTreatmentPlan), expected: "treatment_plan"},
		{name: "FlowCardKindMedicationFulfillment", value: string(model.FlowCardKindMedicationFulfillment), expected: "medication_fulfillment"},
		{name: "FlowCardKindTreatmentExecution", value: string(model.FlowCardKindTreatmentExecution), expected: "treatment_execution"},
		{name: "FlowCardKindAdviceOnly", value: string(model.FlowCardKindAdviceOnly), expected: "advice_only"},
		{name: "FlowCardKindCompletedVisit", value: string(model.FlowCardKindCompletedVisit), expected: "completed_visit"},

		// FlowCardStatus — 9 values
		{name: "FlowCardStatusPending", value: string(model.FlowCardStatusPending), expected: "pending"},
		{name: "FlowCardStatusAccepted", value: string(model.FlowCardStatusAccepted), expected: "accepted"},
		{name: "FlowCardStatusSkipped", value: string(model.FlowCardStatusSkipped), expected: "skipped"},
		{name: "FlowCardStatusVetoed", value: string(model.FlowCardStatusVetoed), expected: "vetoed"},
		{name: "FlowCardStatusPaid", value: string(model.FlowCardStatusPaid), expected: "paid"},
		{name: "FlowCardStatusProcessing", value: string(model.FlowCardStatusProcessing), expected: "processing"},
		{name: "FlowCardStatusCompleted", value: string(model.FlowCardStatusCompleted), expected: "completed"},
		{name: "FlowCardStatusFailed", value: string(model.FlowCardStatusFailed), expected: "failed"},
		{name: "FlowCardStatusInvalidated", value: string(model.FlowCardStatusInvalidated), expected: "invalidated"},

		// TimelineItemKind — 4 values
		{name: "TimelineItemKindMessage", value: string(model.TimelineItemKindMessage), expected: "message"},
		{name: "TimelineItemKindFlowCard", value: string(model.TimelineItemKindFlowCard), expected: "flow_card"},
		{name: "TimelineItemKindSystemEvent", value: string(model.TimelineItemKindSystemEvent), expected: "system_event"},
		{name: "TimelineItemKindTerminal", value: string(model.TimelineItemKindTerminal), expected: "terminal"},

		// TimelineItemStatus — 5 values
		{name: "TimelineItemStatusPending", value: string(model.TimelineItemStatusPending), expected: "pending"},
		{name: "TimelineItemStatusStreaming", value: string(model.TimelineItemStatusStreaming), expected: "streaming"},
		{name: "TimelineItemStatusDone", value: string(model.TimelineItemStatusDone), expected: "done"},
		{name: "TimelineItemStatusFailed", value: string(model.TimelineItemStatusFailed), expected: "failed"},
		{name: "TimelineItemStatusInvalidated", value: string(model.TimelineItemStatusInvalidated), expected: "invalidated"},

		// SystemEventType — 8 values
		{name: "SystemEventTypeContextLoaded", value: string(model.SystemEventTypeContextLoaded), expected: "context_loaded"},
		{name: "SystemEventTypeAgentThinking", value: string(model.SystemEventTypeAgentThinking), expected: "agent_thinking"},
		{name: "SystemEventTypeLabResultReceived", value: string(model.SystemEventTypeLabResultReceived), expected: "lab_result_received"},
		{name: "SystemEventTypePaymentSucceeded", value: string(model.SystemEventTypePaymentSucceeded), expected: "payment_succeeded"},
		{name: "SystemEventTypeDrugPurchased", value: string(model.SystemEventTypeDrugPurchased), expected: "drug_purchased"},
		{name: "SystemEventTypeFollowUpStarted", value: string(model.SystemEventTypeFollowUpStarted), expected: "follow_up_started"},
		{name: "SystemEventTypeEmergencyDismissed", value: string(model.SystemEventTypeEmergencyDismissed), expected: "emergency_dismissed"},
		{name: "SystemEventTypeExitSettled", value: string(model.SystemEventTypeExitSettled), expected: "exit_settled"},

		// SSEEventType — 7 values
		{name: "SSEEventTypeDelta", value: string(model.SSEEventTypeDelta), expected: "delta"},
		{name: "SSEEventTypeMessageFinal", value: string(model.SSEEventTypeMessageFinal), expected: "message_final"},
		{name: "SSEEventTypeCard", value: string(model.SSEEventTypeCard), expected: "card"},
		{name: "SSEEventTypeState", value: string(model.SSEEventTypeState), expected: "state"},
		{name: "SSEEventTypeEmergency", value: string(model.SSEEventTypeEmergency), expected: "emergency"},
		{name: "SSEEventTypeDone", value: string(model.SSEEventTypeDone), expected: "done"},
		{name: "SSEEventTypeError", value: string(model.SSEEventTypeError), expected: "error"},

		// Minor enums
		{name: "VisitEntryTypeNew", value: string(model.VisitEntryTypeNew), expected: "new"},
		{name: "VisitEntryTypeFollowUp", value: string(model.VisitEntryTypeFollowUp), expected: "follow_up"},
		{name: "GenderMale", value: string(model.GenderMale), expected: "male"},
		{name: "GenderFemale", value: string(model.GenderFemale), expected: "female"},
		{name: "GenderOther", value: string(model.GenderOther), expected: "other"},
		{name: "GenderUnknown", value: string(model.GenderUnknown), expected: "unknown"},
		{name: "ExitConsequenceNoFee", value: string(model.ExitConsequenceNoFee), expected: "no_fee"},
		{name: "ExitConsequenceRefundable", value: string(model.ExitConsequenceRefundable), expected: "refundable"},
		{name: "ExitConsequenceExecutedNoRefund", value: string(model.ExitConsequenceExecutedNoRefund), expected: "executed_no_refund"},
		{name: "ExitConsequenceMedicationDispensed", value: string(model.ExitConsequenceMedicationDispensed), expected: "medication_dispensed"},
		{name: "ConsultationIntentConsultation", value: string(model.ConsultationIntentConsultation), expected: "consultation"},
		{name: "ConsultationIntentFollowUp", value: string(model.ConsultationIntentFollowUp), expected: "follow_up"},
		{name: "ConsultationIntentUncertain", value: string(model.ConsultationIntentUncertain), expected: "uncertain"},
		{name: "TreatmentPlanMedication", value: string(model.TreatmentPlanMedication), expected: "medication"},
		{name: "TreatmentPlanTreatment", value: string(model.TreatmentPlanTreatment), expected: "treatment"},
		{name: "TreatmentPlanAdviceOnly", value: string(model.TreatmentPlanAdviceOnly), expected: "advice_only"},
		{name: "TreatmentPlanReferral", value: string(model.TreatmentPlanReferral), expected: "referral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestTerminalReasons(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "TerminalReasonEmergency", value: string(model.TerminalReasonEmergency), expected: "emergency"},
		{name: "TerminalReasonTimeout", value: string(model.TerminalReasonTimeout), expected: "timeout"},
		{name: "TerminalReasonAskLimitReached", value: string(model.TerminalReasonAskLimitReached), expected: "ask_limit_reached"},
		{name: "TerminalReasonLabLimitReached", value: string(model.TerminalReasonLabLimitReached), expected: "lab_limit_reached"},
		{name: "TerminalReasonReferral", value: string(model.TerminalReasonReferral), expected: "referral"},
		{name: "TerminalReasonCapabilityInsufficient", value: string(model.TerminalReasonCapabilityInsufficient), expected: "capability_insufficient"},
		{name: "TerminalReasonExited", value: string(model.TerminalReasonExited), expected: "exited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	if model.ErrSessionNotFound.Error() == "" {
		t.Error("ErrSessionNotFound should have message")
	}
	if model.ErrPatientNotFound.Error() == "" {
		t.Error("ErrPatientNotFound should have message")
	}
	if model.ErrCardNotFound.Error() == "" {
		t.Error("ErrCardNotFound should have message")
	}
	if model.ErrValidation.Error() == "" {
		t.Error("ErrValidation should have message")
	}
}

func TestExitConsequence(t *testing.T) {
	c := model.ExitConsequence{
		Kind:   "no_fee",
		Amount: 0,
		Text:   "未产生费用",
	}
	b, _ := json.Marshal(c)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)
	if parsed["kind"] != "no_fee" {
		t.Error("kind mismatch")
	}
}

func TestPaymentTypes(t *testing.T) {
	input := model.SubmitPaymentInput{
		SessionID:       "v001",
		CardID:          "f001",
		Purpose:         "lab",
		PaymentMethodID: "pm001",
		Defer:           false,
	}
	b, _ := json.Marshal(input)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)
	if parsed["purpose"] != "lab" {
		t.Error("purpose mismatch")
	}
	if parsed["cardId"] != "f001" {
		t.Error("cardId mismatch")
	}
}

func TestFloat64Ptr(t *testing.T) {
	p := model.Float64Ptr(42.5)
	if p == nil {
		t.Fatal("Float64Ptr returned nil")
	}
	if *p != 42.5 {
		t.Errorf("Float64Ptr: got %v, want 42.5", *p)
	}
}

func TestDerefFloat64(t *testing.T) {
	// Non-nil pointer
	val := 99.9
	if got := model.DerefFloat64(&val); got != 99.9 {
		t.Errorf("DerefFloat64(non-nil): got %v, want 99.9", got)
	}
	// Nil pointer
	if got := model.DerefFloat64(nil); got != 0 {
		t.Errorf("DerefFloat64(nil): got %v, want 0", got)
	}
}

func TestFloat64PtrZero(t *testing.T) {
	p := model.Float64Ptr(0)
	if p == nil {
		t.Fatal("Float64Ptr(0) returned nil — must return a valid pointer to 0")
	}
	if *p != 0 {
		t.Errorf("Float64Ptr(0): got %v, want 0", *p)
	}
}

func TestFlowActionResult(t *testing.T) {
	result := model.FlowActionResult{
		SessionID: "v001",
		Status:    "chatting",
		Message:   "完成",
	}
	b, _ := json.Marshal(result)
	var parsed map[string]interface{}
	_ = json.Unmarshal(b, &parsed)
	if parsed["sessionId"] != "v001" {
		t.Error("sessionId mismatch")
	}
}

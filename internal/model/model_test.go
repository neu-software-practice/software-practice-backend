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
		{"invalid entry type", model.CreateSessionInput{PatientID: "p1", EntryType: "follow_up"}, true},
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
	// Verify key enum values match front-api.md specs
	if model.VisitStatusChatting != "chatting" {
		t.Error("VisitStatusChatting mismatch")
	}
	if model.VisitStatusBlocked != "blocked" {
		t.Error("VisitStatusBlocked mismatch")
	}
	if model.VisitStatusCompleted != "completed" {
		t.Error("VisitStatusCompleted mismatch")
	}

	if model.TerminalReasonEmergency != "emergency" {
		t.Error("TerminalReasonEmergency mismatch")
	}

	if model.FlowCardKindLabDecision != "lab_decision" {
		t.Error("FlowCardKindLabDecision mismatch")
	}
	if model.FlowCardKindDiagnosis != "diagnosis" {
		t.Error("FlowCardKindDiagnosis mismatch")
	}

	if model.PaymentStatusUnpaid != "unpaid" {
		t.Error("PaymentStatusUnpaid mismatch")
	}
	if model.PaymentStatusPaid != "paid" {
		t.Error("PaymentStatusPaid mismatch")
	}

	if string(model.GenderMale) != "male" {
		t.Error("GenderMale mismatch")
	}

	if string(model.ExitConsequenceNoFee) != "no_fee" {
		t.Error("ExitConsequenceNoFee mismatch")
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

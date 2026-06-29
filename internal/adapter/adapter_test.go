package adapter_test

import (
	"testing"

	"github.com/neuhis/software-practice-backend/internal/adapter"
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

func TestStepMappingTable(t *testing.T) {
	tests := []struct {
		kind         medagent.StepKind
		producesCard bool
		isTerminal   bool
		minSSETypes  int
	}{
		{medagent.StepAsk, false, false, 2},
		{medagent.StepNeedTests, true, false, 2},
		{medagent.StepDrugQuery, false, false, 1},
		{medagent.StepPurchase, true, false, 2},
		{medagent.StepEmergency, false, true, 1},
		{medagent.StepDone, true, true, 3},
		{medagent.StepOK, false, false, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			mapping, ok := adapter.GetMapping(tt.kind)
			if !ok {
				t.Fatalf("no mapping found for %s", tt.kind)
			}
			if mapping.ProducesCard != tt.producesCard {
				t.Errorf("ProducesCard = %v, want %v", mapping.ProducesCard, tt.producesCard)
			}
			if mapping.IsTerminal != tt.isTerminal {
				t.Errorf("IsTerminal = %v, want %v", mapping.IsTerminal, tt.isTerminal)
			}
			if len(mapping.SSETypes) < tt.minSSETypes {
				t.Errorf("SSETypes length = %d, want >= %d", len(mapping.SSETypes), tt.minSSETypes)
			}
		})
	}
}

func TestGetMappingUnknown(t *testing.T) {
	_, ok := adapter.GetMapping("UNKNOWN_KIND")
	if ok {
		t.Error("expected no mapping for unknown kind")
	}
}

func TestBuildLabDecisionCard(t *testing.T) {
	step := &medagent.Step{
		Kind:      medagent.StepNeedTests,
		DoctorSay: "需要检查血常规",
		TestItems: []string{"血常规"},
	}

	card := adapter.BuildLabDecisionCard("s1", step)
	if card.Kind != string(model.FlowCardKindLabDecision) {
		t.Errorf("kind = %s", card.Kind)
	}
	if !card.Blocking {
		t.Error("lab decision card should be blocking")
	}
	if card.SessionID != "s1" {
		t.Errorf("sessionID = %s", card.SessionID)
	}
	if len(card.TestItems) != 1 {
		t.Errorf("testItems length = %d, want 1", len(card.TestItems))
	}
}

func TestBuildDiagnosisCard(t *testing.T) {
	result := &medagent.Result{
		Final: "ADVICE",
		Plan:  "MEDICATION",
		Diagnosis: &medagent.Diagnosis{
			Name:       "急性上呼吸道感染",
			Basis:      "发热+咽痛+血象",
			Confidence: 0.88,
		},
		Advice: "多休息",
	}

	card := adapter.BuildDiagnosisCard("s1", result)
	if card.Kind != string(model.FlowCardKindDiagnosis) {
		t.Errorf("kind = %s", card.Kind)
	}
	if card.Confidence != "high" {
		t.Errorf("confidence = %s, want high", card.Confidence)
	}
}

func TestBuildMedicationFulfillmentCard(t *testing.T) {
	step := &medagent.Step{
		Kind: medagent.StepPurchase,
		Orders: []medagent.DrugOrder{
			{Name: "布洛芬缓释胶囊", Quantity: 2},
		},
	}

	card := adapter.BuildMedicationFulfillmentCard("s1", step)
	if card.Kind != string(model.FlowCardKindMedicationFulfillment) {
		t.Errorf("kind = %s", card.Kind)
	}
	if len(card.Medications) != 1 {
		t.Errorf("medications length = %d, want 1", len(card.Medications))
	}
	if len(card.AvailableModes) != 2 {
		t.Errorf("availableModes length = %d, want 2", len(card.AvailableModes))
	}
}

func TestBuildCompletedVisitCard(t *testing.T) {
	result := &medagent.Result{
		Final: "ADVICE",
		Plan:  "MEDICATION",
		Diagnosis: &medagent.Diagnosis{
			Name: "感冒",
		},
		Advice: "多喝水",
	}

	card := adapter.BuildCompletedVisitCard("s1", result)
	if card.Kind != string(model.FlowCardKindCompletedVisit) {
		t.Errorf("kind = %s", card.Kind)
	}
}

func TestBuildAdviceOnlyCard(t *testing.T) {
	result := &medagent.Result{
		Final:  "ADVICE",
		Plan:   "ADVICE_ONLY",
		Advice: "多休息，三天后复诊",
	}

	card := adapter.BuildAdviceOnlyCard("s1", result)
	if card.Kind != string(model.FlowCardKindAdviceOnly) {
		t.Errorf("kind = %s", card.Kind)
	}
	if !card.Blocking {
		t.Error("advice only card should be blocking")
	}
}

func TestBuildPaymentCard(t *testing.T) {
	items := []model.PaymentLineItem{
		{Name: "血常规", Amount: 50.0, Quantity: 1},
	}

	card := adapter.BuildPaymentCard("s1", "lab", items, 50.0)
	if card.Kind != string(model.FlowCardKindPayment) {
		t.Errorf("kind = %s", card.Kind)
	}
	if card.PaymentStatus != "unpaid" {
		t.Errorf("paymentStatus = %s", card.PaymentStatus)
	}
}

func TestTimelineBuilders(t *testing.T) {
	t.Run("BuildMessageTimelineItem", func(t *testing.T) {
		item := adapter.BuildMessageTimelineItem("s1", "patient", "hello")
		if item.Kind != "message" {
			t.Errorf("kind = %s", item.Kind)
		}
		if item.Role != "patient" {
			t.Errorf("role = %s", item.Role)
		}
	})

	t.Run("BuildSystemEventTimelineItem", func(t *testing.T) {
		item := adapter.BuildSystemEventTimelineItem("s1", "context_loaded", "Title", "Desc")
		if item.Kind != "system_event" {
			t.Errorf("kind = %s", item.Kind)
		}
		if item.EventType != "context_loaded" {
			t.Errorf("eventType = %s", item.EventType)
		}
	})

	t.Run("BuildTerminalTimelineItem", func(t *testing.T) {
		item := adapter.BuildTerminalTimelineItem("s1", "completed", "Done", "All done")
		if item.Kind != "terminal" {
			t.Errorf("kind = %s", item.Kind)
		}
		if item.Reason == nil || *item.Reason != "completed" {
			t.Error("reason mismatch")
		}
	})

	t.Run("BuildInitialTimeline", func(t *testing.T) {
		items := adapter.BuildInitialTimeline("s1", "头疼")
		if len(items) < 2 {
			t.Errorf("items length = %d, want >= 2", len(items))
		}
	})
}

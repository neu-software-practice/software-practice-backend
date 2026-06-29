package adapter_test

import (
	"testing"
	"time"

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

func TestBuildTreatmentPlanCard(t *testing.T) {
	t.Run("MEDICATION plan", func(t *testing.T) {
		result := &medagent.Result{
			Final:  "ADVICE",
			Plan:   "MEDICATION",
			Advice: "按时服药",
		}
		card := adapter.BuildTreatmentPlanCard("s1", result)
		if card.Kind != string(model.FlowCardKindTreatmentPlan) {
			t.Errorf("kind = %s, want treatment_plan", card.Kind)
		}
		if card.Plan != "MEDICATION" {
			t.Errorf("plan = %s, want MEDICATION", card.Plan)
		}
		if card.Capability != "available" {
			t.Errorf("capability = %s, want available", card.Capability)
		}
		if card.Summary != "按时服药" {
			t.Errorf("summary = %s, want 按时服药", card.Summary)
		}
	})

	t.Run("REFERRAL plan", func(t *testing.T) {
		result := &medagent.Result{
			Final:  "REFER",
			Plan:   "REFERRAL",
			Advice: "建议转诊至专科医院",
		}
		card := adapter.BuildTreatmentPlanCard("s2", result)
		if card.Capability != "unavailable" {
			t.Errorf("capability = %s, want unavailable", card.Capability)
		}
	})
}

func TestBuildDiagnosisCard_AllConfidenceLevels(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		want       string
	}{
		{"high", 0.88, "high"},
		{"medium_boundary", 0.79, "medium"},
		{"medium_mid", 0.60, "medium"},
		{"low_boundary", 0.49, "low"},
		{"low", 0.20, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &medagent.Result{
				Final: "ADVICE",
				Plan:  "MEDICATION",
				Diagnosis: &medagent.Diagnosis{
					Name:       "测试诊断",
					Basis:      "测试依据",
					Confidence: tt.confidence,
				},
			}
			card := adapter.BuildDiagnosisCard("s1", result)
			if card.Confidence != tt.want {
				t.Errorf("confidence = %s, want %s", card.Confidence, tt.want)
			}
		})
	}

	t.Run("nil_diagnosis", func(t *testing.T) {
		result := &medagent.Result{
			Final: "ADVICE",
			Plan:  "MEDICATION",
		}
		card := adapter.BuildDiagnosisCard("s1", result)
		if card.Diagnosis != "" {
			t.Errorf("diagnosis = %s, want empty", card.Diagnosis)
		}
		if card.Confidence != "" {
			t.Errorf("confidence = %s, want empty", card.Confidence)
		}
	})
}

func TestBuildFlowCardTimelineItem(t *testing.T) {
	card := &model.FlowCard{
		ID:        "card-1",
		SessionID: "s1",
		Kind:      string(model.FlowCardKindLabDecision),
		Status:    "pending",
		Title:     "检验决策",
	}

	item := adapter.BuildFlowCardTimelineItem("s1", card)
	if item.Kind != string(model.TimelineItemKindFlowCard) {
		t.Errorf("kind = %s, want flow_card", item.Kind)
	}
	if item.SessionID != "s1" {
		t.Errorf("sessionID = %s", item.SessionID)
	}
	if item.Card == nil {
		t.Fatal("Card should not be nil")
	}
	if item.Card.ID != "card-1" {
		t.Errorf("card.ID = %s, want card-1", item.Card.ID)
	}
}

func TestBuildTimelineFromRecord(t *testing.T) {
	now := time.Now()
	sessionID := "s1"

	tests := []struct {
		name     string
		record   *medagent.SessionRecord
		wantKind string
		wantRole string
	}{
		{
			name: "patient_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "patient", Text: "我头疼"},
				},
			},
			wantKind: "message",
			wantRole: "patient",
		},
		{
			name: "doctor_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "doctor", Text: "请描述症状"},
				},
			},
			wantKind: "message",
			wantRole: "assistant",
		},
		{
			name: "test_request_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "test_request", Text: "血常规"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "test_result_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "test_result", Text: "正常"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "drug_query_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "drug_query", Text: "布洛芬"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "drug_info_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "drug_info", Text: "布洛芬缓释胶囊 0.3g*20粒"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "purchase_request_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "purchase_request", Text: "购买布洛芬"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "purchase_result_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "purchase_result", Text: "已购买"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "advice_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "advice", Text: "多休息"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "emergency_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "emergency", Text: "血压过高"},
				},
			},
			wantKind: "terminal",
		},
		{
			name: "unknown_turn",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "unknown_kind", Text: "未知事件"},
				},
			},
			wantKind: "system_event",
		},
		{
			name: "multiple_turns",
			record: &medagent.SessionRecord{
				SessionID: sessionID,
				Turns: []medagent.RecordedTurn{
					{At: now, Kind: "patient", Text: "头疼"},
					{At: now, Kind: "doctor", Text: "需要检查"},
					{At: now, Kind: "test_request", Text: "血常规"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := adapter.BuildTimelineFromRecord(sessionID, tt.record)
			if len(items) != len(tt.record.Turns) {
				t.Fatalf("items length = %d, want %d", len(items), len(tt.record.Turns))
			}

			for i, item := range items {
				if item.SessionID != sessionID {
					t.Errorf("item[%d].SessionID = %s, want %s", i, item.SessionID, sessionID)
				}

				if tt.wantKind != "" && item.Kind != tt.wantKind {
					t.Errorf("item[%d].Kind = %s, want %s", i, item.Kind, tt.wantKind)
				}

				if tt.wantRole != "" && item.Role != tt.wantRole {
					t.Errorf("item[%d].Role = %s, want %s", i, item.Role, tt.wantRole)
				}
			}

			// Verify emergency turn has Reason set
			for _, item := range items {
				if item.Kind == "terminal" {
					if item.Reason == nil {
						t.Error("emergency turn should have Reason set")
					}
				}
			}
		})
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

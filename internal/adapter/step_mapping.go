package adapter

import (
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// StepMapping defines how a medAgent Step.kind maps to SSE events and FlowCard kinds.
type StepMapping struct {
	SSETypes          []string           // SSE event types to emit
	CardKind          model.FlowCardKind // Primary FlowCardKind if a card is produced (non-empty means a card is produced)
	SecondaryCardKind model.FlowCardKind // Optional secondary FlowCardKind (e.g. StepDone produces diagnosis + completed_visit/advice_only)
	IsTerminal        bool
}

// StepMappingTable maps medAgent StepKind to StepMapping.
var StepMappingTable = map[medagent.StepKind]StepMapping{
	medagent.StepAsk: {
		SSETypes: []string{"delta", "message_final"},
	},
	medagent.StepNeedTests: {
		SSETypes: []string{"card", "state"},
		CardKind: model.FlowCardKindLabDecision,
	},
	medagent.StepDrugQuery: {
		SSETypes: []string{"state"},
	},
	medagent.StepPurchase: {
		SSETypes: []string{"card", "state"},
		CardKind: model.FlowCardKindMedicationFulfillment,
	},
	medagent.StepEmergency: {
		SSETypes:   []string{"emergency"},
		IsTerminal: true,
	},
	// StepDone produces multiple cards (diagnosis + treatment_plan / advice_only / completed_visit)
	// based on the plan field. handleDone in chat.go contains the full branching logic.
	medagent.StepDone: {
		SSETypes:          []string{"card", "card", "done"},
		CardKind:          model.FlowCardKindDiagnosis,
		SecondaryCardKind: model.FlowCardKindTreatmentPlan,
		IsTerminal:        true,
	},
	medagent.StepOK: {
		SSETypes: []string{},
	},
}

// GetMapping returns the StepMapping for a given StepKind.
func GetMapping(kind medagent.StepKind) (StepMapping, bool) {
	m, ok := StepMappingTable[kind]
	return m, ok
}

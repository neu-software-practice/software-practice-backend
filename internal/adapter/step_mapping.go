package adapter

import (
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// StepMapping defines how a medAgent Step.kind maps to SSE events and FlowCard kinds.
type StepMapping struct {
	SSETypes      []string          // SSE event types to emit
	CardKind      model.FlowCardKind // FlowCardKind if a card is produced
	ProducesCard  bool
	IsTerminal    bool
}

// StepMappingTable maps medAgent StepKind to StepMapping.
var StepMappingTable = map[medagent.StepKind]StepMapping{
	medagent.StepAsk: {
		SSETypes:     []string{"delta", "message_final"},
		ProducesCard: false,
	},
	medagent.StepNeedTests: {
		SSETypes:     []string{"card", "state"},
		CardKind:     model.FlowCardKindLabDecision,
		ProducesCard: true,
	},
	medagent.StepDrugQuery: {
		SSETypes:     []string{"state"},
		ProducesCard: false,
	},
	medagent.StepPurchase: {
		SSETypes:     []string{"card", "state"},
		CardKind:     model.FlowCardKindMedicationFulfillment,
		ProducesCard: true,
	},
	medagent.StepEmergency: {
		SSETypes:     []string{"emergency"},
		ProducesCard: false,
		IsTerminal:   true,
	},
	medagent.StepDone: {
		SSETypes:     []string{"card", "card", "done"},
		CardKind:     model.FlowCardKindDiagnosis,
		ProducesCard: true,
		IsTerminal:   true,
	},
	medagent.StepOK: {
		SSETypes:     []string{"state"},
		ProducesCard: false,
	},
}

// GetMapping returns the StepMapping for a given StepKind.
func GetMapping(kind medagent.StepKind) (StepMapping, bool) {
	m, ok := StepMappingTable[kind]
	return m, ok
}

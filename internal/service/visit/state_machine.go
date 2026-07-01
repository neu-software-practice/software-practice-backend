package visit

import (
	"fmt"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// TerminalStates are states from which no further transitions are allowed.
var TerminalStates = map[string]bool{
	string(model.VisitMachineStateCompleted):   true,
	string(model.VisitMachineStateTerminated):  true,
	string(model.VisitMachineStateExited):      true,
	string(model.VisitMachineStateTransferred): true,
}

// IsTerminal checks if a machine state is terminal.
func IsTerminal(machineState string) bool {
	return TerminalStates[machineState]
}

// AllowedTransitions defines valid state transitions for VisitMachineState.
var AllowedTransitions = map[string][]string{
	string(model.VisitMachineStateLoadingContext): {
		string(model.VisitMachineStateChatting),
	},
	string(model.VisitMachineStateChatting): {
		string(model.VisitMachineStateAnalyzing),
		string(model.VisitMachineStateLabDecision),
		string(model.VisitMachineStateEmergencyPending),
		string(model.VisitMachineStateExitSettlement),
	},
	string(model.VisitMachineStateAnalyzing): {
		string(model.VisitMachineStateChatting),
		string(model.VisitMachineStateLabDecision),
		string(model.VisitMachineStateDiagnosis),
		string(model.VisitMachineStateEmergencyPending),
	},
	string(model.VisitMachineStateLabDecision): {
		string(model.VisitMachineStateChatting),   // vetoed
		string(model.VisitMachineStateDiagnosis),  // skipped
		string(model.VisitMachineStateLabPayment), // accepted
		string(model.VisitMachineStateExitSettlement),
	},
	string(model.VisitMachineStateLabPayment): {
		string(model.VisitMachineStateLabExecution),
		string(model.VisitMachineStateExitSettlement),
	},
	string(model.VisitMachineStateLabExecution): {
		string(model.VisitMachineStateAnalyzing), // result returned → continue
		string(model.VisitMachineStateDiagnosis),
	},
	string(model.VisitMachineStateDiagnosis): {
		string(model.VisitMachineStateTreatmentDecision),
	},
	string(model.VisitMachineStateTreatmentDecision): {
		string(model.VisitMachineStateMedicationPayment),
		string(model.VisitMachineStateMedicationFulfillment),
		string(model.VisitMachineStateTreatmentExecution),
		string(model.VisitMachineStateAdviceOnly),
		string(model.VisitMachineStateCompleted),   // referral
		string(model.VisitMachineStateTransferred), // transferred
	},
	string(model.VisitMachineStateMedicationPayment): {
		string(model.VisitMachineStateMedicationFulfillment),
		string(model.VisitMachineStateExitSettlement),
	},
	string(model.VisitMachineStateMedicationFulfillment): {
		string(model.VisitMachineStateCompleted),
		string(model.VisitMachineStateExitSettlement),
	},
	string(model.VisitMachineStateTreatmentExecution): {
		string(model.VisitMachineStateCompleted),
		string(model.VisitMachineStateExitSettlement),
	},
	string(model.VisitMachineStateAdviceOnly): {
		string(model.VisitMachineStateCompleted),
	},
	string(model.VisitMachineStateEmergencyPending): {
		string(model.VisitMachineStateTerminated),
	},
	string(model.VisitMachineStateExitSettlement): {
		string(model.VisitMachineStateExited),
	},
}

// CanTransition checks if a transition from current to next state is valid.
func CanTransition(current, next string) bool {
	if IsTerminal(current) {
		return false
	}
	allowed, ok := AllowedTransitions[current]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}

// Transition attempts to move from current to next state.
// Returns the new state and an error if the transition is invalid.
func Transition(current, next string) (string, error) {
	if !CanTransition(current, next) {
		return current, fmt.Errorf("invalid state transition: %s -> %s", current, next)
	}
	return next, nil
}

// MachineStateToStatus maps internal machine states to external VisitStatus values.
var MachineStateToStatus = map[string]string{
	string(model.VisitMachineStateLoadingContext):        string(model.VisitStatusLoadingContext),
	string(model.VisitMachineStateChatting):              string(model.VisitStatusChatting),
	string(model.VisitMachineStateAnalyzing):             string(model.VisitStatusAnalyzing),
	string(model.VisitMachineStateLabDecision):           string(model.VisitStatusBlocked),
	string(model.VisitMachineStateLabPayment):            string(model.VisitStatusBlocked),
	string(model.VisitMachineStateLabExecution):          string(model.VisitStatusDiagnosis),
	string(model.VisitMachineStateDiagnosis):             string(model.VisitStatusDiagnosis),
	string(model.VisitMachineStateTreatmentDecision):     string(model.VisitStatusTreatment),
	string(model.VisitMachineStateMedicationPayment):     string(model.VisitStatusBlocked),
	string(model.VisitMachineStateMedicationFulfillment): string(model.VisitStatusBlocked),
	string(model.VisitMachineStateTreatmentExecution):    string(model.VisitStatusTreatment),
	string(model.VisitMachineStateAdviceOnly):            string(model.VisitStatusBlocked),
	string(model.VisitMachineStateCompleted):             string(model.VisitStatusCompleted),
	string(model.VisitMachineStateEmergencyPending):      string(model.VisitStatusEmergencyTerminated),
	string(model.VisitMachineStateTerminated):            string(model.VisitStatusEmergencyTerminated),
	string(model.VisitMachineStateExitSettlement):        string(model.VisitStatusExited),
	string(model.VisitMachineStateExited):                string(model.VisitStatusExited),
	string(model.VisitMachineStateTransferred):           string(model.VisitStatusTransferred),
}

// GetStatusForState returns the external VisitStatus for a given machine state.
func GetStatusForState(machineState string) string {
	if status, ok := MachineStateToStatus[machineState]; ok {
		return status
	}
	return string(model.VisitStatusChatting)
}

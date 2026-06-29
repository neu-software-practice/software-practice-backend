package model

// PaymentInfo holds the full payment breakdown for a visit or purpose.
type PaymentInfo struct {
	PaymentID       string            `json:"paymentId"`
	Purpose         string            `json:"purpose"`
	Items           []PaymentLineItem `json:"items"`
	TotalAmount     float64           `json:"totalAmount"`
	InsuranceAmount float64           `json:"insuranceAmount"`
	SelfPayAmount   float64           `json:"selfPayAmount"`
	Status          string            `json:"status"`
}

// SubmitPaymentInput is the request payload to initiate a payment.
type SubmitPaymentInput struct {
	SessionID       string `json:"sessionId"`
	CardID          string `json:"cardId"`
	Purpose         string `json:"purpose"`
	PaymentMethodID string `json:"paymentMethodId,omitempty"`
	SimulateStatus  string `json:"simulateStatus,omitempty"`
	Defer           bool   `json:"defer,omitempty"`
}

// FlowActionResult is the generic result returned after a flow action.
type FlowActionResult struct {
	SessionID     string         `json:"sessionId"`
	Status        string         `json:"status"`
	ActiveCardID  *string        `json:"activeCardId,omitempty"`
	Card          *FlowCard      `json:"card,omitempty"`
	TimelineItems []TimelineItem `json:"timelineItems"`
	Message       string         `json:"message,omitempty"`
}

// ExitVisitInput is the request payload to exit a visit.
type ExitVisitInput struct {
	SessionID string `json:"sessionId"`
	Reason    string `json:"reason"`
}

// ExitSettlementResult is the result returned after a visit exit settlement.
type ExitSettlementResult struct {
	SessionID      string           `json:"sessionId"`
	TerminalReason string           `json:"terminalReason"`
	RefundAmount   float64          `json:"refundAmount"`
	PayableAmount  float64          `json:"payableAmount"`
	TimelineItem   TimelineItem     `json:"timelineItem"`
	Consequence    *ExitConsequence `json:"consequence,omitempty"`
}

// ExitConsequence describes a consequence of exiting a visit.
type ExitConsequence struct {
	Kind   string  `json:"kind"`
	Amount float64 `json:"amount,omitempty"`
	Text   string  `json:"text"`
}

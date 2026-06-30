package model

// BillingRecord aggregates payment information across all sessions for a patient.
type BillingRecord struct {
	PaymentID       string            `json:"paymentId"`
	SessionID       string            `json:"sessionId"`
	SessionTitle    string            `json:"sessionTitle"`
	Purpose         string            `json:"purpose"`
	Items           []BillingLineItem `json:"items"`
	TotalAmount     float64           `json:"totalAmount"`
	InsuranceAmount float64           `json:"insuranceAmount"`
	SelfPayAmount   float64           `json:"selfPayAmount"`
	PaymentStatus   string            `json:"paymentStatus"`
	CreatedAt       string            `json:"createdAt"`
}

// BillingLineItem is a single line item in a billing record.
type BillingLineItem struct {
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Quantity *int    `json:"quantity,omitempty"`
}

// BillingRecordsResponse is the response for GET /billing/records.
type BillingRecordsResponse struct {
	Items []BillingRecord `json:"items"`
}

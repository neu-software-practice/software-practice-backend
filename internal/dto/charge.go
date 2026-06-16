package dto

// ChargeItemRef identifies one payable item to settle (F1-3).
type ChargeItemRef struct {
	ItemType string `json:"item_type" binding:"required"` // check/inspection/disposal/prescription
	ID       uint   `json:"id" binding:"required"`
}

// ChargeRequest is the F1-3 settlement body.
type ChargeRequest struct {
	CaseNumber string          `json:"case_number" binding:"required"`
	Items      []ChargeItemRef `json:"items" binding:"required,min=1,dive"`
}

// PendingItemsResponse is the F1-3 pending-charges screen payload.
type PendingItemsResponse struct {
	Register RegisterBrief `json:"register"`
	Items    []PendingItem `json:"items"`
	Total    float64       `json:"total"`
}

// ChargeResult summarizes a settlement.
type ChargeResult struct {
	RegisterID uint    `json:"register_id"`
	Count      int     `json:"count"`
	Total      float64 `json:"total"`
}

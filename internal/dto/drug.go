package dto

// DrugRequest is the create/update body for drug inventory management (F5-3).
type DrugRequest struct {
	DrugCode     string  `json:"drug_code" binding:"required"`
	DrugName     string  `json:"drug_name" binding:"required"`
	DrugFormat   string  `json:"drug_format"`
	DrugUnit     string  `json:"drug_unit"`
	Manufacturer string  `json:"manufacturer"`
	DrugDosage   string  `json:"drug_dosage"`
	DrugType     string  `json:"drug_type"`
	DrugPrice    float64 `json:"drug_price" binding:"min=0"`
	DrugStock    int     `json:"drug_stock" binding:"min=0"`
	MnemonicCode string  `json:"mnemonic_code"`
}

// StockRequest adjusts a drug's stock (F5-3 入库/调整). Positive = restock.
type StockRequest struct {
	Delta int `json:"delta" binding:"required"`
}

// RefundRequest reverses paid items (F1-4). Mirrors ChargeRequest.
type RefundRequest struct {
	CaseNumber string          `json:"case_number" binding:"required"`
	Items      []ChargeItemRef `json:"items" binding:"required,min=1,dive"`
}

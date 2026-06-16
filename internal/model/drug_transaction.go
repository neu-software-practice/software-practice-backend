package model

import "time"

// DrugTransaction records a pharmacy dispense (F5-1) or refund-to-stock (F5-2)
// event, backing the F5-4 transaction history. Kept separate from ChargeRecord
// because dispensing is an inventory event, not a payment event.
type DrugTransaction struct {
	ID             uint      `gorm:"column:id;primaryKey" json:"id"`
	PrescriptionID uint      `gorm:"column:prescription_id;index" json:"prescription_id"`
	RegisterID     uint      `gorm:"column:register_id;index" json:"register_id"`
	DrugID         uint      `gorm:"column:drug_id" json:"drug_id"`
	DrugName       string    `gorm:"column:drug_name;size:255" json:"drug_name"`
	Quantity       int       `gorm:"column:quantity" json:"quantity"`
	Action         string    `gorm:"column:action;size:16;index" json:"action"` // 发药 / 退药
	OperatorID     uint      `gorm:"column:operator_id" json:"operator_id"`
	CreatedAt      time.Time `gorm:"column:created_at;index" json:"created_at"`
}

func (DrugTransaction) TableName() string { return "drug_transaction" }

package model

import "time"

// ChargeRecord is a financial transaction line (SPEC §6 補全: 收费流水 for
// auditability). Each charge (F1-3) or refund (F1-4) writes one row, which
// F1-5 / F2-11 query. item_type is one of constant.ChargeItem*.
type ChargeRecord struct {
	ID         uint      `gorm:"column:id;primaryKey" json:"id"`
	RegisterID uint      `gorm:"column:register_id;index" json:"register_id"`
	ItemType   string    `gorm:"column:item_type;size:32;index" json:"item_type"`
	ItemID     uint      `gorm:"column:item_id" json:"item_id"`
	ItemName   string    `gorm:"column:item_name;size:128" json:"item_name"`
	Amount     float64   `gorm:"column:amount;type:decimal(8,2)" json:"amount"`
	Action     string    `gorm:"column:action;size:16;index" json:"action"` // 收费 / 退费
	OperatorID uint      `gorm:"column:operator_id" json:"operator_id"`
	CreatedAt  time.Time `gorm:"column:created_at;index" json:"created_at"`
}

func (ChargeRecord) TableName() string { return "charge_record" }

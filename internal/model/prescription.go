package model

import "time"

// Prescription is one prescribed drug line (SPEC §6 table 14). drug_state follows
// §5.3. drug_number is modelled as an integer quantity (SPEC §6 補全) so the line
// amount can be computed as drug_info.drug_price × drug_number.
type Prescription struct {
	ID           uint      `gorm:"column:id;primaryKey" json:"id"`
	RegisterID   uint      `gorm:"column:register_id;index" json:"register_id"`
	DrugID       uint      `gorm:"column:drug_id" json:"drug_id"`
	DrugUsage    string    `gorm:"column:drug_usage;size:255" json:"drug_usage"`
	DrugNumber   int       `gorm:"column:drug_number" json:"drug_number"`
	CreationTime time.Time `gorm:"column:creation_time" json:"creation_time"`
	DrugState    string    `gorm:"column:drug_state;size:64;index" json:"drug_state"`

	Drug *DrugInfo `gorm:"foreignKey:DrugID" json:"drug,omitempty"`
}

func (Prescription) TableName() string { return "prescription" }

package model

import "time"

// DrugInfo is a drug master record (SPEC §6 table 15). drug_stock is a §6 補全
// addition so dispense (F5-1) / refund-to-stock (F5-2) and pharmacy inventory
// management (F5-3) have an inventory quantity to operate on.
type DrugInfo struct {
	ID           uint      `gorm:"column:id;primaryKey" json:"id"`
	DrugCode     string    `gorm:"column:drug_code;size:255;index" json:"drug_code"`
	DrugName     string    `gorm:"column:drug_name;size:255;index" json:"drug_name"`
	DrugFormat   string    `gorm:"column:drug_format;size:255" json:"drug_format"`
	DrugUnit     string    `gorm:"column:drug_unit;size:16" json:"drug_unit"`
	Manufacturer string    `gorm:"column:manufacturer;size:255" json:"manufacturer"`
	DrugDosage   string    `gorm:"column:drug_dosage;size:64" json:"drug_dosage"`
	DrugType     string    `gorm:"column:drug_type;size:64" json:"drug_type"`
	DrugPrice    float64   `gorm:"column:drug_price;type:decimal(8,2)" json:"drug_price"`
	DrugStock    int       `gorm:"column:drug_stock;default:0" json:"drug_stock"`
	MnemonicCode string    `gorm:"column:mnemonic_code;size:255;index" json:"mnemonic_code"`
	CreationDate time.Time `gorm:"column:creation_date;type:date" json:"creation_date"`
	Delmark      int       `gorm:"column:delmark;default:1;index" json:"delmark"`
}

func (DrugInfo) TableName() string { return "drug_info" }

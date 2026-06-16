package model

// MedicalTechnology is a medical-technology project (SPEC §6 table 10): check,
// inspection or disposal item. tech_type partitions which request it belongs to.
type MedicalTechnology struct {
	ID         uint    `gorm:"column:id;primaryKey" json:"id"`
	TechCode   string  `gorm:"column:tech_code;size:64;index" json:"tech_code"`
	TechName   string  `gorm:"column:tech_name;size:64;index" json:"tech_name"`
	TechFormat string  `gorm:"column:tech_format;size:64" json:"tech_format"`
	TechPrice  float64 `gorm:"column:tech_price;type:decimal(8,2)" json:"tech_price"`
	TechType   string  `gorm:"column:tech_type;size:64;index" json:"tech_type"`
	PriceType  string  `gorm:"column:price_type;size:64" json:"price_type"`
	DeptmentID uint    `gorm:"column:deptment_id" json:"deptment_id"`
}

func (MedicalTechnology) TableName() string { return "medical_technology" }

package model

// SettleCategory is a settlement category (SPEC §6 table 4), e.g. 自费/医保/新农合.
type SettleCategory struct {
	ID         uint   `gorm:"column:id;primaryKey" json:"id"`
	SettleCode string `gorm:"column:settle_code;size:64" json:"settle_code"`
	SettleName string `gorm:"column:settle_name;size:64" json:"settle_name"`
	SequenceNo int    `gorm:"column:sequence_no" json:"sequence_no"`
	Delmark    int    `gorm:"column:delmark;default:1;index" json:"delmark"`
}

func (SettleCategory) TableName() string { return "settle_category" }

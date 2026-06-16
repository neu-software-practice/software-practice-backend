package model

// RegistLevel is a registration level / ticket grade (SPEC §6 table 3),
// e.g. 专家号 / 普通号, each carrying its own fee.
type RegistLevel struct {
	ID          uint    `gorm:"column:id;primaryKey" json:"id"`
	RegistCode  string  `gorm:"column:regist_code;size:64" json:"regist_code"`
	RegistName  string  `gorm:"column:regist_name;size:64" json:"regist_name"`
	RegistFee   float64 `gorm:"column:regist_fee;type:decimal(8,2)" json:"regist_fee"`
	RegistQuota int     `gorm:"column:regist_quota" json:"regist_quota"`
	SequenceNo  int     `gorm:"column:sequence_no" json:"sequence_no"`
	Delmark     int     `gorm:"column:delmark;default:1;index" json:"delmark"`
}

func (RegistLevel) TableName() string { return "regist_level" }

package model

// MedicalRecord is the patient's medical record for a visit (SPEC §6 table 11).
// One record per register_id; filled at F2-2 and updated at F2-8 confirmation.
type MedicalRecord struct {
	ID           uint   `gorm:"column:id;primaryKey" json:"id"`
	RegisterID   uint   `gorm:"column:register_id;uniqueIndex" json:"register_id"`
	Readme       string `gorm:"column:readme;size:512" json:"readme"`               // 主诉
	Present      string `gorm:"column:present;size:512" json:"present"`             // 现病史
	PresentTreat string `gorm:"column:present_treat;size:512" json:"present_treat"` // 现病治疗情况
	History      string `gorm:"column:history;size:512" json:"history"`             // 既往史
	Allergy      string `gorm:"column:allergy;size:512" json:"allergy"`             // 过敏史
	Physique     string `gorm:"column:physique;size:512" json:"physique"`           // 体格检查
	Proposal     string `gorm:"column:proposal;size:512" json:"proposal"`           // 检查/检验建议
	Careful      string `gorm:"column:careful;size:512" json:"careful"`             // 注意事项
	Diagnosis    string `gorm:"column:diagnosis;size:512" json:"diagnosis"`         // 诊断结果
	Cure         string `gorm:"column:cure;size:512" json:"cure"`                   // 处理意见

	// Diseases is populated by the repository via medical_record_disease; it is
	// not a GORM association (gorm:"-") to keep AutoMigrate and the explicit
	// join-table model from colliding.
	Diseases []Disease `gorm:"-" json:"diseases,omitempty"`
}

func (MedicalRecord) TableName() string { return "medical_record" }

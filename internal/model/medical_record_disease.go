package model

// MedicalRecordDisease links a medical record to its diagnosed diseases
// (SPEC §6 table 12). A surrogate id plus a unique (record, disease) pair keeps
// the join table easy to manage from the repository layer.
type MedicalRecordDisease struct {
	ID              uint `gorm:"column:id;primaryKey" json:"id"`
	MedicalRecordID uint `gorm:"column:medical_record_id;uniqueIndex:uniq_record_disease" json:"medical_record_id"`
	DiseaseID       uint `gorm:"column:disease_id;uniqueIndex:uniq_record_disease" json:"disease_id"`
}

func (MedicalRecordDisease) TableName() string { return "medical_record_disease" }

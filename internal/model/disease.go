package model

// Disease is a diagnosable disease (SPEC §6 table 13).
type Disease struct {
	ID              uint   `gorm:"column:id;primaryKey" json:"id"`
	DiseaseCode     string `gorm:"column:disease_code;size:64;index" json:"disease_code"`
	DiseaseName     string `gorm:"column:disease_name;size:255;index" json:"disease_name"`
	DiseaseICD      string `gorm:"column:disease_icd;size:64" json:"disease_icd"`
	DiseaseCategory string `gorm:"column:disease_category;size:64" json:"disease_category"`
}

func (Disease) TableName() string { return "disease" }

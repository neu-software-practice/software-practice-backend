package model

// All returns every persisted model. Used by the test harness for AutoMigrate
// and table truncation, and as a single inventory of the 17 tables.
func All() []interface{} {
	return []interface{}{
		&Department{},
		&RegistLevel{},
		&SettleCategory{},
		&Scheduling{},
		&Employee{},
		&Register{},
		&MedicalTechnology{},
		&CheckRequest{},
		&InspectionRequest{},
		&DisposalRequest{},
		&MedicalRecord{},
		&Disease{},
		&MedicalRecordDisease{},
		&DrugInfo{},
		&Prescription{},
		&ChargeRecord{},
		&DrugTransaction{},
	}
}

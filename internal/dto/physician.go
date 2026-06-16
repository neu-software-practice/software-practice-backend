package dto

// MedicalRecordRequest is the F2-2 病历首页 save body.
type MedicalRecordRequest struct {
	Readme       string `json:"readme"`
	Present      string `json:"present"`
	PresentTreat string `json:"present_treat"`
	History      string `json:"history"`
	Allergy      string `json:"allergy"`
	Physique     string `json:"physique"`
	Proposal     string `json:"proposal"`
	Careful      string `json:"careful"`
	DiseaseIDs   []uint `json:"disease_ids"`
}

// DiagnoseRequest is the F2-8 门诊确诊 body. finish=true ends the visit (state 3).
type DiagnoseRequest struct {
	Diagnosis string `json:"diagnosis" binding:"required"`
	Cure      string `json:"cure"`
	Finish    bool   `json:"finish"`
}

// PrescriptionItemInput is one drug line of F2-9.
type PrescriptionItemInput struct {
	DrugID     uint   `json:"drug_id" binding:"required"`
	DrugUsage  string `json:"drug_usage"`
	DrugNumber int    `json:"drug_number" binding:"required,min=1"`
}

// PrescriptionRequest is the F2-9 开立处方 body.
type PrescriptionRequest struct {
	Items []PrescriptionItemInput `json:"items" binding:"required,min=1,dive"`
}

// PrescriptionResult summarizes a prescription submission.
type PrescriptionResult struct {
	RegisterID uint    `json:"register_id"`
	Count      int     `json:"count"`
	Total      float64 `json:"total"`
}

// PatientCounts is the F2-1 header counter (排队 / 已看诊).
type PatientCounts struct {
	Queued int64 `json:"queued"`
	Seen   int64 `json:"seen"`
}

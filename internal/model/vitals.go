package model

// VitalsData represents a set of patient vital signs.
// All fields are pointers to distinguish unset from zero values.
type VitalsData struct {
	Temperature       *float64 `json:"temperature"`
	HeartRate         *int     `json:"heartRate" binding:"omitempty,gt=0"`
	SystolicPressure  *int     `json:"systolicPressure" binding:"omitempty,gt=0"`
	DiastolicPressure *int     `json:"diastolicPressure" binding:"omitempty,gt=0"`
	SpO2              *float64 `json:"spo2" binding:"omitempty,min=0,max=100"`
}

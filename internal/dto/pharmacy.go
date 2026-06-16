package dto

import "github.com/neu-software-practice/software-practice-backend/internal/model"

// DispenseList is the F5-1 pharmacy screen payload: a patient and their
// prescriptions in the requested state.
type DispenseList struct {
	Register RegisterBrief        `json:"register"`
	Items    []model.Prescription `json:"items"`
}

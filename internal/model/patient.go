// Package model defines domain entities, enums, DTOs, and business error sentinels.
package model

import "time"

// PatientProfile represents the patient's basic profile information.
// It is returned as part of identity verification and context retrieval.
type PatientProfile struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Gender              string    `json:"gender"`
	Age                 int       `json:"age"`
	PhoneMasked         string    `json:"phoneMasked,omitempty"`
	IDCardMasked        string    `json:"idCardMasked,omitempty"`
	Allergies           []string  `json:"allergies"`
	ChronicDiseases     []string  `json:"chronicDiseases"`
	LongTermMedications []string  `json:"longTermMedications"`
	MedicalHistory      []string  `json:"-"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
}

// PatientContext represents the full consultation context for a patient.
// It wraps the patient profile along with their chief complaint, medical history,
// allergies, long-term medications, and optionally the prior visit summary.
type PatientContext struct {
	Patient        PatientProfile     `json:"patient"`
	ChiefComplaint string             `json:"chiefComplaint,omitempty"`
	PriorVisit     *PatientPriorVisit `json:"priorVisit,omitempty"`
}

// PatientPriorVisit summarizes the patient's most recent visit.
// It includes the session identifier, completion time, diagnosis,
// optional lab result summary, and treatment summary.
type PatientPriorVisit struct {
	SessionID        string    `json:"sessionId"`
	CompletedAt      time.Time `json:"completedAt"`
	Diagnosis        string    `json:"diagnosis"`
	LabResultSummary string    `json:"labResultSummary,omitempty"`
	TreatmentSummary string    `json:"treatmentSummary"`
}

// ProfileUpdateInput carries the fields that can be updated on a patient's profile.
// All slices are optional; only provided fields will be updated.
type ProfileUpdateInput struct {
	PatientID           string   `json:"patientId"`
	Allergies           []string `json:"allergies,omitempty"`
	ChronicDiseases     []string `json:"chronicDiseases,omitempty"`
	LongTermMedications []string `json:"longTermMedications,omitempty"`
	MedicalHistory      []string `json:"medicalHistory,omitempty"`
}

// VerifyIdentityInput represents the request body for patient identity verification.
// CredentialType specifies the type of credential (e.g. "id_card" or "phone"),
// Credential holds the credential value, and Name is the optional patient name.
type VerifyIdentityInput struct {
	CredentialType string `json:"credentialType"`
	Credential     string `json:"credential"`
	Name           string `json:"name,omitempty"`
}

// VerifyIdentityResult is the response returned after successful identity verification.
// It contains the patient profile, the list of readable scopes, and the verification timestamp.
type VerifyIdentityResult struct {
	Patient        PatientProfile `json:"patient"`
	ReadableScopes []string       `json:"readableScopes"`
	VerifiedAt     time.Time      `json:"verifiedAt"`
}

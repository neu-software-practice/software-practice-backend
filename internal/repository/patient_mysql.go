package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type patientMySQLRepo struct {
	db *sql.DB
}

// NewPatientRepository creates a new MySQL-based PatientRepository.
func NewPatientRepository(db *sql.DB) PatientRepository {
	return &patientMySQLRepo{db: db}
}

// scanPatient scans a patient row from the given scanner and parses JSON array columns.
func scanPatient(scanner rowScanner) (*model.PatientProfile, error) {
	var p model.PatientProfile
	var allergiesJSON, chronicJSON, medsJSON, medHistJSON string

	err := scanner.Scan(
		&p.ID, &p.Name, &p.Gender, &p.Age, &p.PhoneMasked, &p.IDCardMasked,
		&allergiesJSON, &chronicJSON, &medsJSON, &medHistJSON, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrPatientNotFound
	}
	if err != nil {
		return nil, err
	}

	p.Allergies = parseJSONStringArray(allergiesJSON)
	p.ChronicDiseases = parseJSONStringArray(chronicJSON)
	p.LongTermMedications = parseJSONStringArray(medsJSON)
	p.MedicalHistory = parseJSONStringArray(medHistJSON)

	return &p, nil
}

func (r *patientMySQLRepo) FindByCredential(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
	query := `SELECT id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, medical_history, updated_at
		FROM patients WHERE `

	switch credType {
	case "id_card":
		query += `id_card_masked = ?`
	case "phone":
		query += `phone_masked = ?`
	default:
		return nil, fmt.Errorf("unknown credential type: %s", credType)
	}

	p, err := scanPatient(r.db.QueryRowContext(ctx, query, credential))
	if err != nil {
		return nil, fmt.Errorf("find patient by credential: %w", err)
	}
	return p, nil
}

func (r *patientMySQLRepo) FindByID(ctx context.Context, id string) (*model.PatientProfile, error) {
	p, err := scanPatient(r.db.QueryRowContext(ctx,
		`SELECT id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, medical_history, updated_at
		FROM patients WHERE id = ?`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("find patient by id: %w", err)
	}
	return p, nil
}

func (r *patientMySQLRepo) Create(ctx context.Context, patient *model.PatientProfile) error {
	allergiesJSON, _ := json.Marshal(patient.Allergies)
	chronicJSON, _ := json.Marshal(patient.ChronicDiseases)
	medsJSON, _ := json.Marshal(patient.LongTermMedications)
	medHistJSON, _ := json.Marshal(patient.MedicalHistory)

	now := time.Now()
	patient.CreatedAt = now
	patient.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO patients (id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, medical_history, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patient.ID, patient.Name, patient.Gender, patient.Age,
		patient.PhoneMasked, patient.IDCardMasked,
		string(allergiesJSON), string(chronicJSON), string(medsJSON), string(medHistJSON),
		patient.CreatedAt, patient.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create patient: %w", err)
	}
	return nil
}

func (r *patientMySQLRepo) UpdateProfile(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var p model.PatientProfile
	var allergiesJSON, chronicJSON, medsJSON, medHistJSON string

	err = tx.QueryRowContext(ctx,
		`SELECT id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, medical_history, updated_at
		FROM patients WHERE id = ? FOR UPDATE`, id,
	).Scan(&p.ID, &p.Name, &p.Gender, &p.Age, &p.PhoneMasked, &p.IDCardMasked,
		&allergiesJSON, &chronicJSON, &medsJSON, &medHistJSON, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrPatientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find patient for update: %w", err)
	}

	p.Allergies = parseJSONStringArray(allergiesJSON)
	p.ChronicDiseases = parseJSONStringArray(chronicJSON)
	p.LongTermMedications = parseJSONStringArray(medsJSON)
	p.MedicalHistory = parseJSONStringArray(medHistJSON)

	if input.Allergies != nil {
		p.Allergies = input.Allergies
	}
	if input.ChronicDiseases != nil {
		p.ChronicDiseases = input.ChronicDiseases
	}
	if input.LongTermMedications != nil {
		p.LongTermMedications = input.LongTermMedications
	}
	if input.MedicalHistory != nil {
		p.MedicalHistory = input.MedicalHistory
	}

	now := time.Now()

	allergiesJSONB, _ := json.Marshal(p.Allergies)
	chronicJSONB, _ := json.Marshal(p.ChronicDiseases)
	medsJSONB, _ := json.Marshal(p.LongTermMedications)
	medHistJSONB, _ := json.Marshal(p.MedicalHistory)

	_, err = tx.ExecContext(ctx,
		`UPDATE patients SET allergies=?, chronic_diseases=?, long_term_medications=?, medical_history=?, updated_at=? WHERE id=?`,
		string(allergiesJSONB), string(chronicJSONB), string(medsJSONB), string(medHistJSONB), now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update patient profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	p.UpdatedAt = now
	return &p, nil
}

func parseJSONStringArray(raw string) []string {
	var arr []string
	if raw == "" || raw == "null" {
		return []string{}
	}
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return []string{}
	}
	return arr
}

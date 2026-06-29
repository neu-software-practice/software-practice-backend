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

func (r *patientMySQLRepo) FindByCredential(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
	var p model.PatientProfile
	var allergiesJSON, chronicJSON, medsJSON string

	query := `SELECT id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, updated_at
		FROM patients WHERE `

	switch credType {
	case "id_card":
		query += `id_card_masked = ?`
	case "phone":
		query += `phone_masked = ?`
	default:
		return nil, fmt.Errorf("unknown credential type: %s", credType)
	}

	err := r.db.QueryRowContext(ctx, query, credential).Scan(
		&p.ID, &p.Name, &p.Gender, &p.Age, &p.PhoneMasked, &p.IDCardMasked,
		&allergiesJSON, &chronicJSON, &medsJSON, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrPatientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find patient by credential: %w", err)
	}

	p.Allergies = parseJSONStringArray(allergiesJSON)
	p.ChronicDiseases = parseJSONStringArray(chronicJSON)
	p.LongTermMedications = parseJSONStringArray(medsJSON)

	return &p, nil
}

func (r *patientMySQLRepo) FindByID(ctx context.Context, id string) (*model.PatientProfile, error) {
	var p model.PatientProfile
	var allergiesJSON, chronicJSON, medsJSON string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, updated_at
		FROM patients WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Gender, &p.Age, &p.PhoneMasked, &p.IDCardMasked,
		&allergiesJSON, &chronicJSON, &medsJSON, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrPatientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find patient by id: %w", err)
	}

	p.Allergies = parseJSONStringArray(allergiesJSON)
	p.ChronicDiseases = parseJSONStringArray(chronicJSON)
	p.LongTermMedications = parseJSONStringArray(medsJSON)

	return &p, nil
}

func (r *patientMySQLRepo) Create(ctx context.Context, patient *model.PatientProfile) error {
	allergiesJSON, _ := json.Marshal(patient.Allergies)
	chronicJSON, _ := json.Marshal(patient.ChronicDiseases)
	medsJSON, _ := json.Marshal(patient.LongTermMedications)

	now := time.Now()
	patient.CreatedAt = now
	patient.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO patients (id, name, gender, age, phone_masked, id_card_masked,
		allergies, chronic_diseases, long_term_medications, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patient.ID, patient.Name, patient.Gender, patient.Age,
		patient.PhoneMasked, patient.IDCardMasked,
		string(allergiesJSON), string(chronicJSON), string(medsJSON),
		patient.CreatedAt, patient.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create patient: %w", err)
	}
	return nil
}

func (r *patientMySQLRepo) UpdateProfile(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
	p, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Allergies != nil {
		p.Allergies = input.Allergies
	}
	if input.ChronicDiseases != nil {
		p.ChronicDiseases = input.ChronicDiseases
	}
	if input.LongTermMedications != nil {
		p.LongTermMedications = input.LongTermMedications
	}

	allergiesJSON, _ := json.Marshal(p.Allergies)
	chronicJSON, _ := json.Marshal(p.ChronicDiseases)
	medsJSON, _ := json.Marshal(p.LongTermMedications)

	_, err = r.db.ExecContext(ctx,
		`UPDATE patients SET allergies=?, chronic_diseases=?, long_term_medications=?, updated_at=? WHERE id=?`,
		string(allergiesJSON), string(chronicJSON), string(medsJSON), time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update patient profile: %w", err)
	}

	p.UpdatedAt = time.Now()
	return p, nil
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

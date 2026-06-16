package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// MedicalRecordRepository persists the per-visit medical record and its disease
// links (F2-2, F2-8).
type MedicalRecordRepository interface {
	FindByRegisterID(ctx context.Context, registerID uint) (*model.MedicalRecord, error)
	// Upsert creates or updates the record for a visit and replaces its disease
	// links. Caller should wrap in a transaction for atomicity.
	Upsert(ctx context.Context, rec *model.MedicalRecord, diseaseIDs []uint) error
	UpdateDiagnosis(ctx context.Context, registerID uint, diagnosis, cure string) error
}

type medicalRecordRepository struct{ base }

// NewMedicalRecordRepository builds the GORM-backed MedicalRecordRepository.
func NewMedicalRecordRepository(db *gorm.DB) MedicalRecordRepository {
	return &medicalRecordRepository{base{db}}
}

func (r *medicalRecordRepository) FindByRegisterID(ctx context.Context, registerID uint) (*model.MedicalRecord, error) {
	var rec model.MedicalRecord
	if err := r.conn(ctx).Where("register_id = ?", registerID).First(&rec).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	var diseases []model.Disease
	if err := r.conn(ctx).
		Table("disease").
		Joins("JOIN medical_record_disease mrd ON mrd.disease_id = disease.id").
		Where("mrd.medical_record_id = ?", rec.ID).
		Find(&diseases).Error; err != nil {
		return nil, err
	}
	rec.Diseases = diseases
	return &rec, nil
}

func (r *medicalRecordRepository) Upsert(ctx context.Context, rec *model.MedicalRecord, diseaseIDs []uint) error {
	db := r.conn(ctx)

	var existing model.MedicalRecord
	err := db.Where("register_id = ?", rec.RegisterID).First(&existing).Error
	switch {
	case err == nil:
		rec.ID = existing.ID
	case errors.Is(err, gorm.ErrRecordNotFound):
		rec.ID = 0
	default:
		return err
	}

	if err := db.Save(rec).Error; err != nil {
		return err
	}

	// Replace disease links.
	if err := db.Where("medical_record_id = ?", rec.ID).Delete(&model.MedicalRecordDisease{}).Error; err != nil {
		return err
	}
	if len(diseaseIDs) == 0 {
		return nil
	}
	links := make([]model.MedicalRecordDisease, 0, len(diseaseIDs))
	for _, did := range diseaseIDs {
		links = append(links, model.MedicalRecordDisease{MedicalRecordID: rec.ID, DiseaseID: did})
	}
	return db.Create(&links).Error
}

func (r *medicalRecordRepository) UpdateDiagnosis(ctx context.Context, registerID uint, diagnosis, cure string) error {
	return r.conn(ctx).
		Model(&model.MedicalRecord{}).
		Where("register_id = ?", registerID).
		Updates(map[string]interface{}{"diagnosis": diagnosis, "cure": cure}).Error
}

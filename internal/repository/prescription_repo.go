package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// PrescriptionRepository persists prescription lines (F2-9, F1-3, F5-1/2).
type PrescriptionRepository interface {
	CreateBatch(ctx context.Context, items []*model.Prescription) error
	FindByID(ctx context.Context, id uint) (*model.Prescription, error)
	ListByRegister(ctx context.Context, registerID uint) ([]model.Prescription, error)
	ListByRegisterAndState(ctx context.Context, registerID uint, state string) ([]model.Prescription, error)
	UpdateState(ctx context.Context, id uint, state string) error
}

type prescriptionRepository struct{ base }

// NewPrescriptionRepository builds the GORM-backed PrescriptionRepository.
func NewPrescriptionRepository(db *gorm.DB) PrescriptionRepository {
	return &prescriptionRepository{base{db}}
}

func (r *prescriptionRepository) CreateBatch(ctx context.Context, items []*model.Prescription) error {
	if len(items) == 0 {
		return nil
	}
	return r.conn(ctx).Create(&items).Error
}

func (r *prescriptionRepository) FindByID(ctx context.Context, id uint) (*model.Prescription, error) {
	var p model.Prescription
	if err := r.conn(ctx).Preload("Drug").First(&p, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &p, nil
}

func (r *prescriptionRepository) ListByRegister(ctx context.Context, registerID uint) ([]model.Prescription, error) {
	var rows []model.Prescription
	err := r.conn(ctx).
		Preload("Drug").
		Where("register_id = ?", registerID).
		Order("creation_time DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

func (r *prescriptionRepository) ListByRegisterAndState(ctx context.Context, registerID uint, state string) ([]model.Prescription, error) {
	var rows []model.Prescription
	err := r.conn(ctx).
		Preload("Drug").
		Where("register_id = ? AND drug_state = ?", registerID, state).
		Order("creation_time DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

func (r *prescriptionRepository) UpdateState(ctx context.Context, id uint, state string) error {
	return r.conn(ctx).
		Model(&model.Prescription{}).
		Where("id = ?", id).
		Update("drug_state", state).Error
}

package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// MedicalTechnologyRepository serves the check/inspection/disposal project catalog.
type MedicalTechnologyRepository interface {
	Search(ctx context.Context, keyword, techType string, page Page) ([]model.MedicalTechnology, int64, error)
	FindByID(ctx context.Context, id uint) (*model.MedicalTechnology, error)
}

type medicalTechnologyRepository struct{ base }

// NewMedicalTechnologyRepository builds the GORM-backed repository.
func NewMedicalTechnologyRepository(db *gorm.DB) MedicalTechnologyRepository {
	return &medicalTechnologyRepository{base{db}}
}

func (r *medicalTechnologyRepository) Search(ctx context.Context, keyword, techType string, page Page) ([]model.MedicalTechnology, int64, error) {
	apply := func(db *gorm.DB) *gorm.DB {
		db = db.Model(&model.MedicalTechnology{})
		if techType != "" {
			db = db.Where("tech_type = ?", techType)
		}
		if keyword != "" {
			like := "%" + keyword + "%"
			db = db.Where("tech_code LIKE ? OR tech_name LIKE ?", like, like)
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.MedicalTechnology
	err := page.apply(apply(r.conn(ctx)).Order("id ASC")).Find(&rows).Error
	return rows, total, err
}

func (r *medicalTechnologyRepository) FindByID(ctx context.Context, id uint) (*model.MedicalTechnology, error) {
	var row model.MedicalTechnology
	if err := r.conn(ctx).First(&row, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &row, nil
}

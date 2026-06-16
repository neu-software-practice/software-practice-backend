package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// DiseaseRepository serves the disease catalog (病历首页初步诊断选择).
type DiseaseRepository interface {
	Search(ctx context.Context, keyword string, page Page) ([]model.Disease, int64, error)
	FindByIDs(ctx context.Context, ids []uint) ([]model.Disease, error)
}

type diseaseRepository struct{ base }

// NewDiseaseRepository builds the GORM-backed DiseaseRepository.
func NewDiseaseRepository(db *gorm.DB) DiseaseRepository {
	return &diseaseRepository{base{db}}
}

func (r *diseaseRepository) Search(ctx context.Context, keyword string, page Page) ([]model.Disease, int64, error) {
	apply := func(db *gorm.DB) *gorm.DB {
		db = db.Model(&model.Disease{})
		if keyword != "" {
			like := "%" + keyword + "%"
			db = db.Where("disease_code LIKE ? OR disease_name LIKE ? OR disease_icd LIKE ?", like, like, like)
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Disease
	err := page.apply(apply(r.conn(ctx)).Order("id ASC")).Find(&rows).Error
	return rows, total, err
}

func (r *diseaseRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.Disease, error) {
	if len(ids) == 0 {
		return []model.Disease{}, nil
	}
	var rows []model.Disease
	err := r.conn(ctx).Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

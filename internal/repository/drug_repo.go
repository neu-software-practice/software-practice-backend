package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
)

// DrugInfoRepository serves the drug catalog and inventory (F2-9, F5-1/2/3).
type DrugInfoRepository interface {
	Search(ctx context.Context, keyword string, page Page) ([]model.DrugInfo, int64, error)
	FindByID(ctx context.Context, id uint) (*model.DrugInfo, error)
	Create(ctx context.Context, drug *model.DrugInfo) error
	Update(ctx context.Context, drug *model.DrugInfo) error
	SoftDelete(ctx context.Context, id uint) error
	// AdjustStock atomically applies delta to drug_stock. For negative deltas it
	// only succeeds when stock stays ≥ 0, returning ok=false otherwise.
	AdjustStock(ctx context.Context, id uint, delta int) (ok bool, err error)
}

type drugInfoRepository struct{ base }

// NewDrugInfoRepository builds the GORM-backed DrugInfoRepository.
func NewDrugInfoRepository(db *gorm.DB) DrugInfoRepository {
	return &drugInfoRepository{base{db}}
}

func (r *drugInfoRepository) Search(ctx context.Context, keyword string, page Page) ([]model.DrugInfo, int64, error) {
	apply := func(db *gorm.DB) *gorm.DB {
		db = db.Model(&model.DrugInfo{}).Where("delmark = ?", constant.DelmarkActive)
		if keyword != "" {
			like := "%" + keyword + "%"
			db = db.Where("drug_code LIKE ? OR drug_name LIKE ? OR mnemonic_code LIKE ?", like, like, like)
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.DrugInfo
	err := page.apply(apply(r.conn(ctx)).Order("id ASC")).Find(&rows).Error
	return rows, total, err
}

func (r *drugInfoRepository) FindByID(ctx context.Context, id uint) (*model.DrugInfo, error) {
	var row model.DrugInfo
	err := r.conn(ctx).
		Where("id = ? AND delmark = ?", id, constant.DelmarkActive).
		First(&row).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &row, nil
}

func (r *drugInfoRepository) Create(ctx context.Context, drug *model.DrugInfo) error {
	return r.conn(ctx).Create(drug).Error
}

func (r *drugInfoRepository) Update(ctx context.Context, drug *model.DrugInfo) error {
	return r.conn(ctx).Save(drug).Error
}

func (r *drugInfoRepository) SoftDelete(ctx context.Context, id uint) error {
	return r.conn(ctx).
		Model(&model.DrugInfo{}).
		Where("id = ?", id).
		Update("delmark", constant.DelmarkDeleted).Error
}

func (r *drugInfoRepository) AdjustStock(ctx context.Context, id uint, delta int) (bool, error) {
	res := r.conn(ctx).
		Model(&model.DrugInfo{}).
		Where("id = ? AND drug_stock + ? >= 0", id, delta).
		Update("drug_stock", gorm.Expr("drug_stock + ?", delta))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

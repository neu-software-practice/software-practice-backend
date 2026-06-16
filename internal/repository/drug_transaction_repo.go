package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// DrugTransactionFilter narrows the pharmacy transaction history (F5-4).
type DrugTransactionFilter struct {
	RegisterID uint
	Action     string // 发药 / 退药; empty = both
}

// DrugTransactionRepository persists and queries pharmacy dispense/refund events.
type DrugTransactionRepository interface {
	Create(ctx context.Context, rec *model.DrugTransaction) error
	List(ctx context.Context, f DrugTransactionFilter, page Page) ([]model.DrugTransaction, int64, error)
}

type drugTransactionRepository struct{ base }

// NewDrugTransactionRepository builds the GORM-backed DrugTransactionRepository.
func NewDrugTransactionRepository(db *gorm.DB) DrugTransactionRepository {
	return &drugTransactionRepository{base{db}}
}

func (r *drugTransactionRepository) Create(ctx context.Context, rec *model.DrugTransaction) error {
	return r.conn(ctx).Create(rec).Error
}

func (r *drugTransactionRepository) List(ctx context.Context, f DrugTransactionFilter, page Page) ([]model.DrugTransaction, int64, error) {
	apply := func(db *gorm.DB) *gorm.DB {
		db = db.Model(&model.DrugTransaction{})
		if f.RegisterID != 0 {
			db = db.Where("register_id = ?", f.RegisterID)
		}
		if f.Action != "" {
			db = db.Where("action = ?", f.Action)
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.DrugTransaction
	err := page.apply(apply(r.conn(ctx)).Order("created_at DESC, id DESC")).Find(&rows).Error
	return rows, total, err
}

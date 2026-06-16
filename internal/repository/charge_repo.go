package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// ChargeFilter narrows the financial ledger query (F1-5, F2-11).
type ChargeFilter struct {
	RegisterID uint
	Action     string // 收费 / 退费; empty = both
	ItemType   string // empty = all
}

// ChargeRecordRepository persists and queries the financial ledger.
type ChargeRecordRepository interface {
	Create(ctx context.Context, rec *model.ChargeRecord) error
	List(ctx context.Context, f ChargeFilter, page Page) ([]model.ChargeRecord, int64, error)
}

type chargeRecordRepository struct{ base }

// NewChargeRecordRepository builds the GORM-backed ChargeRecordRepository.
func NewChargeRecordRepository(db *gorm.DB) ChargeRecordRepository {
	return &chargeRecordRepository{base{db}}
}

func (r *chargeRecordRepository) Create(ctx context.Context, rec *model.ChargeRecord) error {
	return r.conn(ctx).Create(rec).Error
}

func (r *chargeRecordRepository) List(ctx context.Context, f ChargeFilter, page Page) ([]model.ChargeRecord, int64, error) {
	apply := func(db *gorm.DB) *gorm.DB {
		db = db.Model(&model.ChargeRecord{})
		if f.RegisterID != 0 {
			db = db.Where("register_id = ?", f.RegisterID)
		}
		if f.Action != "" {
			db = db.Where("action = ?", f.Action)
		}
		if f.ItemType != "" {
			db = db.Where("item_type = ?", f.ItemType)
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.ChargeRecord
	err := page.apply(apply(r.conn(ctx)).Order("created_at DESC, id DESC")).Find(&rows).Error
	return rows, total, err
}

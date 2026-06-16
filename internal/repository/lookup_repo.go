package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
)

// RegistLevelRepository serves registration levels (挂号级别).
type RegistLevelRepository interface {
	List(ctx context.Context) ([]model.RegistLevel, error)
	FindByID(ctx context.Context, id uint) (*model.RegistLevel, error)
}

type registLevelRepository struct{ base }

// NewRegistLevelRepository builds the GORM-backed RegistLevelRepository.
func NewRegistLevelRepository(db *gorm.DB) RegistLevelRepository {
	return &registLevelRepository{base{db}}
}

func (r *registLevelRepository) List(ctx context.Context) ([]model.RegistLevel, error) {
	var rows []model.RegistLevel
	err := r.conn(ctx).
		Where("delmark = ?", constant.DelmarkActive).
		Order("sequence_no ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *registLevelRepository) FindByID(ctx context.Context, id uint) (*model.RegistLevel, error) {
	var row model.RegistLevel
	err := r.conn(ctx).
		Where("id = ? AND delmark = ?", id, constant.DelmarkActive).
		First(&row).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &row, nil
}

// SettleCategoryRepository serves settlement categories (结算类别).
type SettleCategoryRepository interface {
	List(ctx context.Context) ([]model.SettleCategory, error)
	FindByID(ctx context.Context, id uint) (*model.SettleCategory, error)
}

type settleCategoryRepository struct{ base }

// NewSettleCategoryRepository builds the GORM-backed SettleCategoryRepository.
func NewSettleCategoryRepository(db *gorm.DB) SettleCategoryRepository {
	return &settleCategoryRepository{base{db}}
}

func (r *settleCategoryRepository) List(ctx context.Context) ([]model.SettleCategory, error) {
	var rows []model.SettleCategory
	err := r.conn(ctx).
		Where("delmark = ?", constant.DelmarkActive).
		Order("sequence_no ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *settleCategoryRepository) FindByID(ctx context.Context, id uint) (*model.SettleCategory, error) {
	var row model.SettleCategory
	err := r.conn(ctx).
		Where("id = ? AND delmark = ?", id, constant.DelmarkActive).
		First(&row).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &row, nil
}

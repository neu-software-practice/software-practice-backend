package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
)

// DepartmentRepository serves departments (科室).
type DepartmentRepository interface {
	List(ctx context.Context) ([]model.Department, error)
	ListByType(ctx context.Context, deptType string) ([]model.Department, error)
	FindByID(ctx context.Context, id uint) (*model.Department, error)
}

type departmentRepository struct{ base }

// NewDepartmentRepository builds the GORM-backed DepartmentRepository.
func NewDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return &departmentRepository{base{db}}
}

func (r *departmentRepository) List(ctx context.Context) ([]model.Department, error) {
	var rows []model.Department
	err := r.conn(ctx).
		Where("delmark = ?", constant.DelmarkActive).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *departmentRepository) ListByType(ctx context.Context, deptType string) ([]model.Department, error) {
	var rows []model.Department
	err := r.conn(ctx).
		Where("dept_type = ? AND delmark = ?", deptType, constant.DelmarkActive).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *departmentRepository) FindByID(ctx context.Context, id uint) (*model.Department, error) {
	var row model.Department
	err := r.conn(ctx).
		Where("id = ? AND delmark = ?", id, constant.DelmarkActive).
		First(&row).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &row, nil
}

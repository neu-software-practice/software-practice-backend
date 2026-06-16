package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
)

// EmployeeRepository accesses employee records (login + identity lookups).
type EmployeeRepository interface {
	FindByUsername(ctx context.Context, username string) (*model.Employee, error)
	FindByID(ctx context.Context, id uint) (*model.Employee, error)
	// ListDoctors returns active doctors in a department at a registration level,
	// used by F1-1 to populate the on-duty doctor picker.
	ListDoctors(ctx context.Context, deptID, registLevelID uint) ([]model.Employee, error)
}

type employeeRepository struct{ base }

// NewEmployeeRepository builds the GORM-backed EmployeeRepository.
func NewEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &employeeRepository{base{db}}
}

func (r *employeeRepository) FindByUsername(ctx context.Context, username string) (*model.Employee, error) {
	var e model.Employee
	err := r.conn(ctx).
		Preload("Department").
		Where("username = ? AND delmark = ?", username, constant.DelmarkActive).
		First(&e).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &e, nil
}

func (r *employeeRepository) FindByID(ctx context.Context, id uint) (*model.Employee, error) {
	var e model.Employee
	err := r.conn(ctx).
		Preload("Department").
		Where("id = ? AND delmark = ?", id, constant.DelmarkActive).
		First(&e).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &e, nil
}

func (r *employeeRepository) ListDoctors(ctx context.Context, deptID, registLevelID uint) ([]model.Employee, error) {
	var rows []model.Employee
	q := r.conn(ctx).
		Preload("RegistLevel").
		Preload("Scheduling").
		Where("deptment_id = ? AND delmark = ?", deptID, constant.DelmarkActive)
	if registLevelID != 0 {
		q = q.Where("regist_level_id = ?", registLevelID)
	}
	err := q.Order("id ASC").Find(&rows).Error
	return rows, err
}

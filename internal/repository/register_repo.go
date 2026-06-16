package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// RegisterFilter narrows a doctor's patient list (F2-1 / F2-5).
type RegisterFilter struct {
	EmployeeID uint   // owning doctor; 0 means "any doctor"
	CaseNumber string // partial match
	Name       string // partial match
	States     []int  // visit_state set; empty means "any"
}

// RegisterRepository persists and queries the visit master record.
type RegisterRepository interface {
	Create(ctx context.Context, reg *model.Register) error
	Save(ctx context.Context, reg *model.Register) error
	UpdateState(ctx context.Context, id uint, state int) error
	FindByID(ctx context.Context, id uint) (*model.Register, error)
	FindByCaseNumber(ctx context.Context, caseNumber string) (*model.Register, error)
	List(ctx context.Context, f RegisterFilter, page Page) ([]model.Register, int64, error)
	CountByState(ctx context.Context, employeeID uint, states ...int) (int64, error)
}

type registerRepository struct{ base }

// NewRegisterRepository builds the GORM-backed RegisterRepository.
func NewRegisterRepository(db *gorm.DB) RegisterRepository {
	return &registerRepository{base{db}}
}

func (r *registerRepository) Create(ctx context.Context, reg *model.Register) error {
	return r.conn(ctx).Create(reg).Error
}

func (r *registerRepository) Save(ctx context.Context, reg *model.Register) error {
	return r.conn(ctx).Save(reg).Error
}

func (r *registerRepository) UpdateState(ctx context.Context, id uint, state int) error {
	return r.conn(ctx).
		Model(&model.Register{}).
		Where("id = ?", id).
		Update("visit_state", state).Error
}

func (r *registerRepository) FindByID(ctx context.Context, id uint) (*model.Register, error) {
	var reg model.Register
	err := r.conn(ctx).
		Preload("Department").Preload("Employee").
		Preload("RegistLevel").Preload("SettleCategory").
		First(&reg, id).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &reg, nil
}

func (r *registerRepository) FindByCaseNumber(ctx context.Context, caseNumber string) (*model.Register, error) {
	var reg model.Register
	err := r.conn(ctx).
		Preload("Department").Preload("Employee").
		Preload("RegistLevel").Preload("SettleCategory").
		Where("case_number = ?", caseNumber).
		First(&reg).Error
	if err != nil {
		return nil, wrapNotFound(err)
	}
	return &reg, nil
}

func (r *registerRepository) List(ctx context.Context, f RegisterFilter, page Page) ([]model.Register, int64, error) {
	apply := func(db *gorm.DB) *gorm.DB {
		db = db.Model(&model.Register{})
		if f.EmployeeID != 0 {
			db = db.Where("employee_id = ?", f.EmployeeID)
		}
		if f.CaseNumber != "" {
			db = db.Where("case_number LIKE ?", "%"+f.CaseNumber+"%")
		}
		if f.Name != "" {
			db = db.Where("real_name LIKE ?", "%"+f.Name+"%")
		}
		if len(f.States) > 0 {
			db = db.Where("visit_state IN ?", f.States)
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.Register
	err := page.apply(apply(r.conn(ctx)).
		Preload("RegistLevel").Preload("Department").
		Order("visit_date DESC, id DESC")).
		Find(&rows).Error
	return rows, total, err
}

func (r *registerRepository) CountByState(ctx context.Context, employeeID uint, states ...int) (int64, error) {
	var total int64
	db := r.conn(ctx).Model(&model.Register{})
	if employeeID != 0 {
		db = db.Where("employee_id = ?", employeeID)
	}
	if len(states) > 0 {
		db = db.Where("visit_state IN ?", states)
	}
	err := db.Count(&total).Error
	return total, err
}

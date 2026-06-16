package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// RequestPtr binds a pointer type *T that satisfies model.MedTechRequest, the
// classic two-type-parameter generics trick so the repo can allocate new(T)
// while still calling the interface methods. Exported so the service layer can
// build matching generic services.
type RequestPtr[T any] interface {
	*T
	model.MedTechRequest
}

// RequestRepository is a generic data-access layer shared by check_request,
// inspection_request and disposal_request. State-column differences are resolved
// at runtime via the StateColumn()/TableName() accessors.
type RequestRepository[T any, PT RequestPtr[T]] struct {
	base
}

// NewRequestRepository builds a generic RequestRepository for entity T.
func NewRequestRepository[T any, PT RequestPtr[T]](db *gorm.DB) *RequestRepository[T, PT] {
	return &RequestRepository[T, PT]{base{db}}
}

func (r *RequestRepository[T, PT]) meta() (table, stateCol string) {
	var zero T
	p := PT(&zero)
	return p.TableName(), p.StateColumn()
}

// Create inserts a new request row.
func (r *RequestRepository[T, PT]) Create(ctx context.Context, e PT) error {
	return r.conn(ctx).Create(e).Error
}

// Save persists changes to an existing request row.
func (r *RequestRepository[T, PT]) Save(ctx context.Context, e PT) error {
	return r.conn(ctx).Save(e).Error
}

// FindByID loads one request with its medical-technology project.
func (r *RequestRepository[T, PT]) FindByID(ctx context.Context, id uint) (PT, error) {
	var m T
	p := PT(&m)
	if err := r.conn(ctx).Preload("MedicalTechnology").First(p, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return p, nil
}

// ListByRegister returns all requests for a visit, newest first.
func (r *RequestRepository[T, PT]) ListByRegister(ctx context.Context, registerID uint) ([]T, error) {
	var rows []T
	err := r.conn(ctx).
		Preload("MedicalTechnology").
		Where("register_id = ?", registerID).
		Order("creation_time DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

// ListByRegisterAndState filters a visit's requests by state.
func (r *RequestRepository[T, PT]) ListByRegisterAndState(ctx context.Context, registerID uint, state string) ([]T, error) {
	_, stateCol := r.meta()
	var rows []T
	err := r.conn(ctx).
		Preload("MedicalTechnology").
		Where("register_id = ? AND "+stateCol+" = ?", registerID, state).
		Order("creation_time DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

// ListPendingRegisters returns the distinct patients (registers) that have at
// least one request in the given state — the apply-acceptance worklist
// (F3-1/F4-1/F6-1).
func (r *RequestRepository[T, PT]) ListPendingRegisters(ctx context.Context, state string, f RegisterFilter, page Page) ([]model.Register, int64, error) {
	table, stateCol := r.meta()
	apply := func(db *gorm.DB) *gorm.DB {
		sub := r.conn(ctx).Table(table).Select("register_id").Where(stateCol+" = ?", state)
		db = db.Model(&model.Register{}).Where("id IN (?)", sub)
		if f.CaseNumber != "" {
			db = db.Where("case_number LIKE ?", "%"+f.CaseNumber+"%")
		}
		if f.Name != "" {
			db = db.Where("real_name LIKE ?", "%"+f.Name+"%")
		}
		return db
	}

	var total int64
	if err := apply(r.conn(ctx)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Register
	err := page.apply(apply(r.conn(ctx)).Order("id DESC")).Find(&rows).Error
	return rows, total, err
}

// CountDistinctRegisters counts distinct patients with a request in the state,
// powering the "已检查 / 排队" header counters.
func (r *RequestRepository[T, PT]) CountDistinctRegisters(ctx context.Context, state string) (int64, error) {
	table, stateCol := r.meta()
	var n int64
	err := r.conn(ctx).
		Table(table).
		Where(stateCol+" = ?", state).
		Distinct("register_id").
		Count(&n).Error
	return n, err
}

package service

import (
	"context"
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// RequestBuilder constructs a fresh typed request from neutral input. Injected at
// wiring time so the otherwise-generic service needs no per-table setters.
type RequestBuilder[T any, PT repository.RequestPtr[T]] func(registerID, techID uint, info, position, remark string) PT

// RequestService is the shared business logic for the three isomorphic request
// families (check / inspection / disposal). One generic implementation drives
// the doctor side (open order, F2-3/4/10), the tech-doctor side (accept →
// execute → result, F3/F4/F6) and the charging integration.
type RequestService[T any, PT repository.RequestPtr[T]] struct {
	repo      *repository.RequestRepository[T, PT]
	registers repository.RegisterRepository
	techs     repository.MedicalTechnologyRepository
	techType  string // expected medical_technology.tech_type
	itemType  string // constant.ChargeItem* label
	build     RequestBuilder[T, PT]
}

// NewRequestService wires a generic RequestService for entity T.
func NewRequestService[T any, PT repository.RequestPtr[T]](
	repo *repository.RequestRepository[T, PT],
	registers repository.RegisterRepository,
	techs repository.MedicalTechnologyRepository,
	techType, itemType string,
	build RequestBuilder[T, PT],
) *RequestService[T, PT] {
	return &RequestService[T, PT]{repo: repo, registers: registers, techs: techs, techType: techType, itemType: itemType, build: build}
}

// ItemType identifies this request family to the charging module.
func (s *RequestService[T, PT]) ItemType() string { return s.itemType }

// Create opens a new request for a visit (F2-3/F2-4/F2-10). It validates that the
// chosen project actually belongs to this family's tech_type.
func (s *RequestService[T, PT]) Create(ctx context.Context, registerID, techID uint, info, position, remark string) (dto.RequestView, error) {
	if _, err := s.registers.FindByID(ctx, registerID); err != nil {
		return dto.RequestView{}, notFoundAs(err, apperr.ErrRegisterNotFound)
	}
	tech, err := s.techs.FindByID(ctx, techID)
	if err != nil {
		return dto.RequestView{}, notFoundAs(err, apperr.ErrNotFound.WithMessage("医技项目不存在"))
	}
	if tech.TechType != s.techType {
		return dto.RequestView{}, apperr.ErrTechTypeMismatch
	}

	p := s.build(registerID, techID, info, position, remark)
	p.SetCreation(time.Now())
	p.SetState(constant.RequestStateCreated)
	if err := s.repo.Create(ctx, p); err != nil {
		return dto.RequestView{}, err
	}

	view := dto.NewRequestView(p)
	view.TechName = tech.TechName
	view.TechPrice = tech.TechPrice
	return view, nil
}

// PendingPatients lists patients (registers) awaiting execution — paid requests
// not yet executed (F3-1/F4-1/F6-1).
func (s *RequestService[T, PT]) PendingPatients(ctx context.Context, f repository.RegisterFilter, page repository.Page) ([]model.Register, int64, error) {
	return s.repo.ListPendingRegisters(ctx, constant.RequestStatePaid, f, page)
}

// Counts returns the "已完成 / 排队" header counters.
func (s *RequestService[T, PT]) Counts(ctx context.Context) (waiting, done int64, err error) {
	if waiting, err = s.repo.CountDistinctRegisters(ctx, constant.RequestStatePaid); err != nil {
		return 0, 0, err
	}
	done, err = s.repo.CountDistinctRegisters(ctx, constant.RequestStateCompleted)
	return waiting, done, err
}

// PatientRequests lists a visit's requests in a given state.
func (s *RequestService[T, PT]) PatientRequests(ctx context.Context, registerID uint, state string) ([]dto.RequestView, error) {
	rows, err := s.repo.ListByRegisterAndState(ctx, registerID, state)
	if err != nil {
		return nil, err
	}
	return s.views(rows), nil
}

// Results lists a visit's completed requests (F2-6/F2-7 result viewing).
func (s *RequestService[T, PT]) Results(ctx context.Context, registerID uint) ([]dto.RequestView, error) {
	return s.PatientRequests(ctx, registerID, constant.RequestStateCompleted)
}

// ByRegister lists every request of a visit regardless of state (F3-4/F4-4/F6-4
// 管理/历史).
func (s *RequestService[T, PT]) ByRegister(ctx context.Context, registerID uint) ([]dto.RequestView, error) {
	rows, err := s.repo.ListByRegister(ctx, registerID)
	if err != nil {
		return nil, err
	}
	return s.views(rows), nil
}

// Execute assigns an executor and moves a paid request into 执行中 (F3-2/F4-2/F6-2).
func (s *RequestService[T, PT]) Execute(ctx context.Context, requestID, executorID uint) (dto.RequestView, error) {
	p, err := s.repo.FindByID(ctx, requestID)
	if err != nil {
		return dto.RequestView{}, notFoundAs(err, apperr.ErrRequestNotFound)
	}
	if p.State() != constant.RequestStatePaid {
		return dto.RequestView{}, apperr.ErrRequestState
	}
	p.AssignExecutor(executorID)
	p.SetState(constant.RequestStateExecuting)
	if err := s.repo.Save(ctx, p); err != nil {
		return dto.RequestView{}, err
	}
	return dto.NewRequestView(p), nil
}

// RecordResult records the outcome and completes a request (F3-3/F4-3/F6-3).
func (s *RequestService[T, PT]) RecordResult(ctx context.Context, requestID, inputterID uint, result string) (dto.RequestView, error) {
	p, err := s.repo.FindByID(ctx, requestID)
	if err != nil {
		return dto.RequestView{}, notFoundAs(err, apperr.ErrRequestNotFound)
	}
	if p.State() != constant.RequestStateExecuting {
		return dto.RequestView{}, apperr.ErrRequestState
	}
	p.RecordResult(result, inputterID, time.Now())
	p.SetState(constant.RequestStateCompleted)
	if err := s.repo.Save(ctx, p); err != nil {
		return dto.RequestView{}, err
	}
	return dto.NewRequestView(p), nil
}

// --- RequestCharger implementation (consumed by ChargeService) ---

// BillableViews lists this family's items for a visit in a billing-relevant
// state (已开立 for charging F1-3, 已缴费 for refunding F1-4).
func (s *RequestService[T, PT]) BillableViews(ctx context.Context, registerID uint, state string) ([]dto.PendingItem, error) {
	rows, err := s.repo.ListByRegisterAndState(ctx, registerID, state)
	if err != nil {
		return nil, err
	}
	items := make([]dto.PendingItem, 0, len(rows))
	for i := range rows {
		items = append(items, s.pendingItem(PT(&rows[i])))
	}
	return items, nil
}

// PayItem flips a visit's 已开立 request to 已缴费 and returns its billable line
// (F1-3). registerID guards against charging an item under the wrong patient.
func (s *RequestService[T, PT]) PayItem(ctx context.Context, registerID, id uint) (dto.PendingItem, error) {
	return s.transition(ctx, registerID, id, constant.RequestStateCreated, constant.RequestStatePaid)
}

// RefundItem flips a visit's 已缴费 request to 已退费 and returns its line (F1-4).
func (s *RequestService[T, PT]) RefundItem(ctx context.Context, registerID, id uint) (dto.PendingItem, error) {
	return s.transition(ctx, registerID, id, constant.RequestStatePaid, constant.RequestStateRefunded)
}

func (s *RequestService[T, PT]) transition(ctx context.Context, registerID, id uint, from, to string) (dto.PendingItem, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return dto.PendingItem{}, notFoundAs(err, apperr.ErrRequestNotFound)
	}
	if p.RequestRegisterID() != registerID {
		return dto.PendingItem{}, apperr.ErrRequestNotFound
	}
	if p.State() != from {
		return dto.PendingItem{}, apperr.ErrRequestState
	}
	p.SetState(to)
	if err := s.repo.Save(ctx, p); err != nil {
		return dto.PendingItem{}, err
	}
	return s.pendingItem(p), nil
}

func (s *RequestService[T, PT]) pendingItem(p PT) dto.PendingItem {
	item := dto.PendingItem{ItemType: s.itemType, ID: p.RequestID(), Quantity: 1, CreationTime: p.GetCreationTime()}
	if t := p.GetMedicalTechnology(); t != nil {
		item.Name = t.TechName
		item.Spec = t.TechFormat
		item.Amount = t.TechPrice
	}
	return item
}

func (s *RequestService[T, PT]) views(rows []T) []dto.RequestView {
	out := make([]dto.RequestView, 0, len(rows))
	for i := range rows {
		out = append(out, dto.NewRequestView(PT(&rows[i])))
	}
	return out
}

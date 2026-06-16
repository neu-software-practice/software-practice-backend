package service

import (
	"context"
	"fmt"
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// RegistrationService handles window registration, cancellation and the
// registration record query (F1-1, F1-2).
type RegistrationService struct {
	registers   repository.RegisterRepository
	levels      repository.RegistLevelRepository
	departments repository.DepartmentRepository
	settles     repository.SettleCategoryRepository
	employees   repository.EmployeeRepository
	charges     repository.ChargeRecordRepository
	tx          repository.TxManager
}

// NewRegistrationService wires the RegistrationService.
func NewRegistrationService(
	registers repository.RegisterRepository,
	levels repository.RegistLevelRepository,
	departments repository.DepartmentRepository,
	settles repository.SettleCategoryRepository,
	employees repository.EmployeeRepository,
	charges repository.ChargeRecordRepository,
	tx repository.TxManager,
) *RegistrationService {
	return &RegistrationService{registers: registers, levels: levels, departments: departments, settles: settles, employees: employees, charges: charges, tx: tx}
}

// Register creates a visit (F1-1): the fee is taken from the chosen level, the
// visit date/half-day are set server-side, the case number is derived from the
// new row's id, and the registration fee is recorded in the financial ledger —
// all atomically. visit_state starts at 已挂号.
func (s *RegistrationService) Register(ctx context.Context, operatorID uint, in dto.RegisterRequest) (*model.Register, error) {
	level, err := s.levels.FindByID(ctx, in.RegistLevelID)
	if err != nil {
		return nil, notFoundAs(err, apperr.ErrNotFound.WithMessage("挂号级别不存在"))
	}
	if _, err := s.departments.FindByID(ctx, in.DeptmentID); err != nil {
		return nil, notFoundAs(err, apperr.ErrNotFound.WithMessage("挂号科室不存在"))
	}
	if _, err := s.settles.FindByID(ctx, in.SettleCategoryID); err != nil {
		return nil, notFoundAs(err, apperr.ErrNotFound.WithMessage("结算类别不存在"))
	}
	doctor, err := s.employees.FindByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, notFoundAs(err, apperr.ErrEmployeeNotFound)
	}
	if doctor.DeptmentID != in.DeptmentID {
		return nil, apperr.ErrBadRequest.WithMessage("所选医生不属于挂号科室")
	}

	birth, err := parseOptionalDate(in.Birthdate)
	if err != nil {
		return nil, apperr.ErrValidation.WithMessage("出生日期格式应为 YYYY-MM-DD")
	}

	now := time.Now()
	reg := &model.Register{
		RealName: in.RealName, Gender: in.Gender, CardNumber: in.CardNumber, Birthdate: birth,
		Age: in.Age, AgeType: orDefault(in.AgeType, "年"), HomeAddress: in.HomeAddress,
		VisitDate: now, Noon: currentNoon(now), DeptmentID: in.DeptmentID, EmployeeID: in.EmployeeID,
		RegistLevelID: in.RegistLevelID, SettleCategoryID: in.SettleCategoryID,
		IsBook: orDefault(in.IsBook, "否"), RegistMethod: in.RegistMethod, RegistMoney: level.RegistFee,
		VisitState: constant.VisitStateRegistered,
	}

	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.registers.Create(ctx, reg); err != nil {
			return err
		}
		reg.CaseNumber = fmt.Sprintf("MR%08d", reg.ID)
		if err := s.registers.Save(ctx, reg); err != nil {
			return err
		}
		return s.charges.Create(ctx, &model.ChargeRecord{
			RegisterID: reg.ID, ItemType: constant.ChargeItemRegister, ItemID: reg.ID, ItemName: "挂号费(" + level.RegistName + ")",
			Amount: level.RegistFee, Action: constant.ChargeActionPay, OperatorID: operatorID, CreatedAt: now,
		})
	})
	if err != nil {
		return nil, err
	}
	return reg, nil
}

// Cancel voids an un-consulted registration (F1-2): visit_state → 已退号 and the
// registration fee is refunded in the ledger.
func (s *RegistrationService) Cancel(ctx context.Context, operatorID, registerID uint) (*model.Register, error) {
	reg, err := s.registers.FindByID(ctx, registerID)
	if err != nil {
		return nil, notFoundAs(err, apperr.ErrRegisterNotFound)
	}
	if reg.VisitState != constant.VisitStateRegistered {
		return nil, apperr.ErrRegisterState.WithMessage("仅可对未接诊的挂号退号")
	}

	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.registers.UpdateState(ctx, registerID, constant.VisitStateCanceled); err != nil {
			return err
		}
		return s.charges.Create(ctx, &model.ChargeRecord{
			RegisterID: reg.ID, ItemType: constant.ChargeItemRegister, ItemID: reg.ID, ItemName: "挂号费退号",
			Amount: reg.RegistMoney, Action: constant.ChargeActionRefund, OperatorID: operatorID, CreatedAt: time.Now(),
		})
	})
	if err != nil {
		return nil, err
	}
	reg.VisitState = constant.VisitStateCanceled
	return reg, nil
}

// List queries registration records (F1-2 查询).
func (s *RegistrationService) List(ctx context.Context, f repository.RegisterFilter, page repository.Page) ([]dto.RegisterBrief, int64, error) {
	rows, total, err := s.registers.List(ctx, f, page)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewRegisterBriefs(rows), total, nil
}

func currentNoon(t time.Time) string {
	if t.Hour() < 12 {
		return constant.NoonMorning
	}
	return constant.NoonAfternoon
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func parseOptionalDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

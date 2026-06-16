package service

import (
	"context"
	"sort"
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// RequestCharger is the slice of behavior ChargeService needs from each request
// family (check/inspection/disposal). Implemented by RequestService.
type RequestCharger interface {
	ItemType() string
	BillableViews(ctx context.Context, registerID uint, state string) ([]dto.PendingItem, error)
	PayItem(ctx context.Context, registerID, id uint) (dto.PendingItem, error)
	RefundItem(ctx context.Context, registerID, id uint) (dto.PendingItem, error)
}

// ChargeService settles and refunds the heterogeneous payable items of a visit
// (F1-3 / F1-4) and queries the financial ledger (F1-5 / F2-11). Writes are
// atomic.
type ChargeService struct {
	registers     repository.RegisterRepository
	prescriptions repository.PrescriptionRepository
	charges       repository.ChargeRecordRepository
	chargers      map[string]RequestCharger
	tx            repository.TxManager
}

// NewChargeService wires the ChargeService with one charger per request family.
func NewChargeService(
	registers repository.RegisterRepository,
	prescriptions repository.PrescriptionRepository,
	charges repository.ChargeRecordRepository,
	tx repository.TxManager,
	chargers ...RequestCharger,
) *ChargeService {
	m := make(map[string]RequestCharger, len(chargers))
	for _, c := range chargers {
		m[c.ItemType()] = c
	}
	return &ChargeService{registers: registers, prescriptions: prescriptions, charges: charges, chargers: m, tx: tx}
}

// PendingItems aggregates the visit's unpaid items, newest first (F1-3).
func (s *ChargeService) PendingItems(ctx context.Context, caseNumber string) (dto.PendingItemsResponse, error) {
	return s.billableItems(ctx, caseNumber, constant.RequestStateCreated, constant.PrescriptionStateCreated)
}

// RefundableItems aggregates the visit's paid (refundable) items (F1-4).
func (s *ChargeService) RefundableItems(ctx context.Context, caseNumber string) (dto.PendingItemsResponse, error) {
	return s.billableItems(ctx, caseNumber, constant.RequestStatePaid, constant.PrescriptionStatePaid)
}

func (s *ChargeService) billableItems(ctx context.Context, caseNumber, reqState, presState string) (dto.PendingItemsResponse, error) {
	reg, err := s.registers.FindByCaseNumber(ctx, caseNumber)
	if err != nil {
		return dto.PendingItemsResponse{}, notFoundAs(err, apperr.ErrRegisterNotFound)
	}

	var items []dto.PendingItem
	for _, c := range s.chargers {
		got, err := c.BillableViews(ctx, reg.ID, reqState)
		if err != nil {
			return dto.PendingItemsResponse{}, err
		}
		items = append(items, got...)
	}

	pres, err := s.prescriptions.ListByRegisterAndState(ctx, reg.ID, presState)
	if err != nil {
		return dto.PendingItemsResponse{}, err
	}
	for i := range pres {
		items = append(items, prescriptionItem(&pres[i]))
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].CreationTime.After(items[j].CreationTime) })

	var total float64
	for _, it := range items {
		total += it.Amount
	}
	return dto.PendingItemsResponse{Register: dto.NewRegisterBrief(reg), Items: items, Total: round2(total)}, nil
}

// Charge settles the selected items in one transaction (F1-3).
func (s *ChargeService) Charge(ctx context.Context, operatorID uint, in dto.ChargeRequest) (dto.ChargeResult, error) {
	return s.settle(ctx, operatorID, in.CaseNumber, in.Items, constant.ChargeActionPay)
}

// Refund reverses the selected paid items in one transaction (F1-4).
func (s *ChargeService) Refund(ctx context.Context, operatorID uint, in dto.RefundRequest) (dto.ChargeResult, error) {
	return s.settle(ctx, operatorID, in.CaseNumber, in.Items, constant.ChargeActionRefund)
}

func (s *ChargeService) settle(ctx context.Context, operatorID uint, caseNumber string, items []dto.ChargeItemRef, action string) (dto.ChargeResult, error) {
	reg, err := s.registers.FindByCaseNumber(ctx, caseNumber)
	if err != nil {
		return dto.ChargeResult{}, notFoundAs(err, apperr.ErrRegisterNotFound)
	}
	if len(items) == 0 {
		return dto.ChargeResult{}, apperr.ErrNoChargeItems
	}

	var total float64
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		for _, ref := range items {
			item, err := s.applyOne(ctx, reg.ID, ref, action)
			if err != nil {
				return err
			}
			if err := s.charges.Create(ctx, &model.ChargeRecord{
				RegisterID: reg.ID, ItemType: item.ItemType, ItemID: item.ID, ItemName: item.Name,
				Amount: item.Amount, Action: action, OperatorID: operatorID, CreatedAt: time.Now(),
			}); err != nil {
				return err
			}
			total += item.Amount
		}
		return nil
	})
	if err != nil {
		return dto.ChargeResult{}, err
	}
	return dto.ChargeResult{RegisterID: reg.ID, Count: len(items), Total: round2(total)}, nil
}

func (s *ChargeService) applyOne(ctx context.Context, registerID uint, ref dto.ChargeItemRef, action string) (dto.PendingItem, error) {
	if ref.ItemType == constant.ChargeItemPrescription {
		return s.applyPrescription(ctx, registerID, ref.ID, action)
	}
	c, ok := s.chargers[ref.ItemType]
	if !ok {
		return dto.PendingItem{}, apperr.ErrBadRequest.WithMessage("未知费用项目类型: " + ref.ItemType)
	}
	if action == constant.ChargeActionRefund {
		return c.RefundItem(ctx, registerID, ref.ID)
	}
	return c.PayItem(ctx, registerID, ref.ID)
}

func (s *ChargeService) applyPrescription(ctx context.Context, registerID, id uint, action string) (dto.PendingItem, error) {
	p, err := s.prescriptions.FindByID(ctx, id)
	if err != nil {
		return dto.PendingItem{}, notFoundAs(err, apperr.ErrNotFound.WithMessage("处方不存在"))
	}
	if p.RegisterID != registerID {
		return dto.PendingItem{}, apperr.ErrNotFound.WithMessage("处方不属于该患者")
	}

	from, to := constant.PrescriptionStateCreated, constant.PrescriptionStatePaid
	if action == constant.ChargeActionRefund {
		from, to = constant.PrescriptionStatePaid, constant.PrescriptionStateCreated
	}
	if p.DrugState != from {
		return dto.PendingItem{}, apperr.ErrPrescriptionState
	}
	if err := s.prescriptions.UpdateState(ctx, id, to); err != nil {
		return dto.PendingItem{}, err
	}
	return prescriptionItem(p), nil
}

// Records lists ledger entries for a visit (F1-5 by case number / F2-11 by id).
func (s *ChargeService) Records(ctx context.Context, caseNumber string, registerID uint, action string, page repository.Page) ([]model.ChargeRecord, int64, error) {
	if caseNumber != "" {
		reg, err := s.registers.FindByCaseNumber(ctx, caseNumber)
		if err != nil {
			return nil, 0, notFoundAs(err, apperr.ErrRegisterNotFound)
		}
		registerID = reg.ID
	}
	return s.charges.List(ctx, repository.ChargeFilter{RegisterID: registerID, Action: action}, page)
}

// prescriptionItem builds the billable line for a prescription (price × qty).
func prescriptionItem(p *model.Prescription) dto.PendingItem {
	item := dto.PendingItem{
		ItemType: constant.ChargeItemPrescription, ID: p.ID,
		Quantity: p.DrugNumber, CreationTime: p.CreationTime,
	}
	if p.Drug != nil {
		item.Name = p.Drug.DrugName
		item.Spec = p.Drug.DrugFormat
		item.Amount = round2(p.Drug.DrugPrice * float64(p.DrugNumber))
	}
	return item
}

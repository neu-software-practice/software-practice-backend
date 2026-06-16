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
	PendingViews(ctx context.Context, registerID uint) ([]dto.PendingItem, error)
	PayItem(ctx context.Context, registerID, id uint) (dto.PendingItem, error)
	RefundItem(ctx context.Context, registerID, id uint) (dto.PendingItem, error)
}

// ChargeService settles and refunds the heterogeneous payable items of a visit
// (F1-3 / F1-4) and writes the financial ledger atomically.
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

// PendingItems aggregates the visit's unpaid items across requests and
// prescriptions, newest first (F1-3).
func (s *ChargeService) PendingItems(ctx context.Context, caseNumber string) (dto.PendingItemsResponse, error) {
	reg, err := s.registers.FindByCaseNumber(ctx, caseNumber)
	if err != nil {
		return dto.PendingItemsResponse{}, notFoundAs(err, apperr.ErrRegisterNotFound)
	}

	var items []dto.PendingItem
	for _, c := range s.chargers {
		got, err := c.PendingViews(ctx, reg.ID)
		if err != nil {
			return dto.PendingItemsResponse{}, err
		}
		items = append(items, got...)
	}

	pres, err := s.prescriptions.ListByRegisterAndState(ctx, reg.ID, constant.PrescriptionStateCreated)
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

// Charge settles the selected items in one transaction, flipping each to 已缴费
// and writing a ledger row per item (F1-3).
func (s *ChargeService) Charge(ctx context.Context, operatorID uint, in dto.ChargeRequest) (dto.ChargeResult, error) {
	reg, err := s.registers.FindByCaseNumber(ctx, in.CaseNumber)
	if err != nil {
		return dto.ChargeResult{}, notFoundAs(err, apperr.ErrRegisterNotFound)
	}
	if len(in.Items) == 0 {
		return dto.ChargeResult{}, apperr.ErrNoChargeItems
	}

	var total float64
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		for _, ref := range in.Items {
			item, err := s.payOne(ctx, reg.ID, ref)
			if err != nil {
				return err
			}
			if err := s.charges.Create(ctx, &model.ChargeRecord{
				RegisterID: reg.ID, ItemType: item.ItemType, ItemID: item.ID, ItemName: item.Name,
				Amount: item.Amount, Action: "收费", OperatorID: operatorID, CreatedAt: time.Now(),
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
	return dto.ChargeResult{RegisterID: reg.ID, Count: len(in.Items), Total: round2(total)}, nil
}

func (s *ChargeService) payOne(ctx context.Context, registerID uint, ref dto.ChargeItemRef) (dto.PendingItem, error) {
	if ref.ItemType == constant.ChargeItemPrescription {
		return s.payPrescription(ctx, registerID, ref.ID)
	}
	c, ok := s.chargers[ref.ItemType]
	if !ok {
		return dto.PendingItem{}, apperr.ErrBadRequest.WithMessage("未知费用项目类型: " + ref.ItemType)
	}
	return c.PayItem(ctx, registerID, ref.ID)
}

func (s *ChargeService) payPrescription(ctx context.Context, registerID, id uint) (dto.PendingItem, error) {
	p, err := s.prescriptions.FindByID(ctx, id)
	if err != nil {
		return dto.PendingItem{}, notFoundAs(err, apperr.ErrNotFound.WithMessage("处方不存在"))
	}
	if p.RegisterID != registerID {
		return dto.PendingItem{}, apperr.ErrNotFound.WithMessage("处方不属于该患者")
	}
	if p.DrugState != constant.PrescriptionStateCreated {
		return dto.PendingItem{}, apperr.ErrPrescriptionState
	}
	if err := s.prescriptions.UpdateState(ctx, id, constant.PrescriptionStatePaid); err != nil {
		return dto.PendingItem{}, err
	}
	return prescriptionItem(p), nil
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

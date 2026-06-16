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

// PharmacyService implements dispensing, returns, drug inventory management and
// the transaction history (F5-1/2/3/4).
type PharmacyService struct {
	prescriptions repository.PrescriptionRepository
	registers     repository.RegisterRepository
	drugs         repository.DrugInfoRepository
	txns          repository.DrugTransactionRepository
	tx            repository.TxManager
}

// NewPharmacyService wires the PharmacyService.
func NewPharmacyService(
	prescriptions repository.PrescriptionRepository,
	registers repository.RegisterRepository,
	drugs repository.DrugInfoRepository,
	txns repository.DrugTransactionRepository,
	tx repository.TxManager,
) *PharmacyService {
	return &PharmacyService{prescriptions: prescriptions, registers: registers, drugs: drugs, txns: txns, tx: tx}
}

// Prescriptions lists a patient's prescriptions in a state (default 已缴费 = 待发药).
func (s *PharmacyService) Prescriptions(ctx context.Context, caseNumber, state string) (dto.DispenseList, error) {
	if state == "" {
		state = constant.PrescriptionStatePaid
	}
	reg, err := s.registers.FindByCaseNumber(ctx, caseNumber)
	if err != nil {
		return dto.DispenseList{}, notFoundAs(err, apperr.ErrRegisterNotFound)
	}
	items, err := s.prescriptions.ListByRegisterAndState(ctx, reg.ID, state)
	if err != nil {
		return dto.DispenseList{}, err
	}
	return dto.DispenseList{Register: dto.NewRegisterBrief(reg), Items: items}, nil
}

// Dispense issues a paid prescription: 已缴费 → 已发药, decrement stock, record the
// transaction — all atomically (F5-1).
func (s *PharmacyService) Dispense(ctx context.Context, operatorID, prescriptionID uint) (*model.Prescription, error) {
	return s.move(ctx, operatorID, prescriptionID,
		constant.PrescriptionStatePaid, constant.PrescriptionStateDispensed,
		constant.DrugActionDispense, -1)
}

// Refund returns a dispensed prescription: 已发药 → 已退药, restock, record the
// transaction (F5-2).
func (s *PharmacyService) Refund(ctx context.Context, operatorID, prescriptionID uint) (*model.Prescription, error) {
	return s.move(ctx, operatorID, prescriptionID,
		constant.PrescriptionStateDispensed, constant.PrescriptionStateRefunded,
		constant.DrugActionReturn, +1)
}

// move performs a dispense/return state transition with the matching inventory
// delta and transaction row. sign is -1 (dispense, out) or +1 (return, in).
func (s *PharmacyService) move(ctx context.Context, operatorID, prescriptionID uint, from, to, action string, sign int) (*model.Prescription, error) {
	var result *model.Prescription
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		p, err := s.prescriptions.FindByID(ctx, prescriptionID)
		if err != nil {
			return notFoundAs(err, apperr.ErrNotFound.WithMessage("处方不存在"))
		}
		if p.DrugState != from {
			return apperr.ErrPrescriptionState.WithMessage("处方状态不允许该操作")
		}

		ok, err := s.drugs.AdjustStock(ctx, p.DrugID, sign*p.DrugNumber)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.ErrConflict.WithMessage("药品库存不足，无法发药")
		}
		if err := s.prescriptions.UpdateState(ctx, p.ID, to); err != nil {
			return err
		}

		name := ""
		if p.Drug != nil {
			name = p.Drug.DrugName
		}
		if err := s.txns.Create(ctx, &model.DrugTransaction{
			PrescriptionID: p.ID, RegisterID: p.RegisterID, DrugID: p.DrugID, DrugName: name,
			Quantity: p.DrugNumber, Action: action, OperatorID: operatorID, CreatedAt: time.Now(),
		}); err != nil {
			return err
		}

		p.DrugState = to
		result = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Transactions lists the dispense/return history (F5-4), optionally by patient.
func (s *PharmacyService) Transactions(ctx context.Context, caseNumber, action string, page repository.Page) ([]model.DrugTransaction, int64, error) {
	var registerID uint
	if caseNumber != "" {
		reg, err := s.registers.FindByCaseNumber(ctx, caseNumber)
		if err != nil {
			return nil, 0, notFoundAs(err, apperr.ErrRegisterNotFound)
		}
		registerID = reg.ID
	}
	return s.txns.List(ctx, repository.DrugTransactionFilter{RegisterID: registerID, Action: action}, page)
}

// CreateDrug adds a drug to the catalog (F5-3).
func (s *PharmacyService) CreateDrug(ctx context.Context, in dto.DrugRequest) (*model.DrugInfo, error) {
	drug := &model.DrugInfo{
		DrugCode: in.DrugCode, DrugName: in.DrugName, DrugFormat: in.DrugFormat, DrugUnit: in.DrugUnit,
		Manufacturer: in.Manufacturer, DrugDosage: in.DrugDosage, DrugType: in.DrugType, DrugPrice: in.DrugPrice,
		DrugStock: in.DrugStock, MnemonicCode: in.MnemonicCode, CreationDate: time.Now(), Delmark: constant.DelmarkActive,
	}
	if err := s.drugs.Create(ctx, drug); err != nil {
		return nil, err
	}
	return drug, nil
}

// UpdateDrug edits a drug (F5-3).
func (s *PharmacyService) UpdateDrug(ctx context.Context, id uint, in dto.DrugRequest) (*model.DrugInfo, error) {
	drug, err := s.drugs.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundAs(err, apperr.ErrNotFound.WithMessage("药品不存在"))
	}
	drug.DrugCode, drug.DrugName, drug.DrugFormat, drug.DrugUnit = in.DrugCode, in.DrugName, in.DrugFormat, in.DrugUnit
	drug.Manufacturer, drug.DrugDosage, drug.DrugType = in.Manufacturer, in.DrugDosage, in.DrugType
	drug.DrugPrice, drug.DrugStock, drug.MnemonicCode = in.DrugPrice, in.DrugStock, in.MnemonicCode
	if err := s.drugs.Update(ctx, drug); err != nil {
		return nil, err
	}
	return drug, nil
}

// DeleteDrug soft-deletes a drug (F5-3).
func (s *PharmacyService) DeleteDrug(ctx context.Context, id uint) error {
	if _, err := s.drugs.FindByID(ctx, id); err != nil {
		return notFoundAs(err, apperr.ErrNotFound.WithMessage("药品不存在"))
	}
	return s.drugs.SoftDelete(ctx, id)
}

// Restock adjusts a drug's stock (F5-3 入库/调整).
func (s *PharmacyService) Restock(ctx context.Context, id uint, delta int) (*model.DrugInfo, error) {
	ok, err := s.drugs.AdjustStock(ctx, id, delta)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperr.ErrConflict.WithMessage("库存调整后不能为负")
	}
	drug, err := s.drugs.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundAs(err, apperr.ErrNotFound.WithMessage("药品不存在"))
	}
	return drug, nil
}

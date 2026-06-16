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

// PharmacyService implements dispensing (F5-1).
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

// Dispense issues a paid prescription: 已缴费 → 已发药, decrement stock and record
// the transaction — all atomically (F5-1).
func (s *PharmacyService) Dispense(ctx context.Context, operatorID, prescriptionID uint) (*model.Prescription, error) {
	var result *model.Prescription
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		p, err := s.prescriptions.FindByID(ctx, prescriptionID)
		if err != nil {
			return notFoundAs(err, apperr.ErrNotFound.WithMessage("处方不存在"))
		}
		if p.DrugState != constant.PrescriptionStatePaid {
			return apperr.ErrPrescriptionState.WithMessage("仅可对已缴费处方发药")
		}

		ok, err := s.drugs.AdjustStock(ctx, p.DrugID, -p.DrugNumber)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.ErrConflict.WithMessage("药品库存不足，无法发药")
		}
		if err := s.prescriptions.UpdateState(ctx, p.ID, constant.PrescriptionStateDispensed); err != nil {
			return err
		}

		name := ""
		if p.Drug != nil {
			name = p.Drug.DrugName
		}
		if err := s.txns.Create(ctx, &model.DrugTransaction{
			PrescriptionID: p.ID, RegisterID: p.RegisterID, DrugID: p.DrugID, DrugName: name,
			Quantity: p.DrugNumber, Action: "发药", OperatorID: operatorID, CreatedAt: time.Now(),
		}); err != nil {
			return err
		}

		p.DrugState = constant.PrescriptionStateDispensed
		result = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

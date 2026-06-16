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

// PhysicianService implements the outpatient doctor's workflow (F2-1/2/5/8/9).
type PhysicianService struct {
	registers     repository.RegisterRepository
	records       repository.MedicalRecordRepository
	prescriptions repository.PrescriptionRepository
	drugs         repository.DrugInfoRepository
	tx            repository.TxManager
}

// NewPhysicianService wires the PhysicianService.
func NewPhysicianService(
	registers repository.RegisterRepository,
	records repository.MedicalRecordRepository,
	prescriptions repository.PrescriptionRepository,
	drugs repository.DrugInfoRepository,
	tx repository.TxManager,
) *PhysicianService {
	return &PhysicianService{registers: registers, records: records, prescriptions: prescriptions, drugs: drugs, tx: tx}
}

// Patients lists the logged-in doctor's patients (F2-1).
func (s *PhysicianService) Patients(ctx context.Context, doctorID uint, f repository.RegisterFilter, page repository.Page) ([]dto.RegisterBrief, int64, error) {
	f.EmployeeID = doctorID
	rows, total, err := s.registers.List(ctx, f, page)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewRegisterBriefs(rows), total, nil
}

// PatientCounts returns the F2-1 header counters: queued (已挂号) vs seen (接诊+结束).
func (s *PhysicianService) PatientCounts(ctx context.Context, doctorID uint) (dto.PatientCounts, error) {
	queued, err := s.registers.CountByState(ctx, doctorID, constant.VisitStateRegistered)
	if err != nil {
		return dto.PatientCounts{}, err
	}
	seen, err := s.registers.CountByState(ctx, doctorID, constant.VisitStateInConsult, constant.VisitStateFinished)
	if err != nil {
		return dto.PatientCounts{}, err
	}
	return dto.PatientCounts{Queued: queued, Seen: seen}, nil
}

// Consult starts a consultation: 已挂号 → 医生接诊 (F2-1 创建病历).
func (s *PhysicianService) Consult(ctx context.Context, doctorID, registerID uint) (*model.Register, error) {
	reg, err := s.ownedRegister(ctx, doctorID, registerID)
	if err != nil {
		return nil, err
	}
	if reg.VisitState != constant.VisitStateRegistered {
		return nil, apperr.ErrRegisterState.WithMessage("仅可对已挂号患者创建病历")
	}
	if err := s.registers.UpdateState(ctx, registerID, constant.VisitStateInConsult); err != nil {
		return nil, err
	}
	reg.VisitState = constant.VisitStateInConsult
	return reg, nil
}

// MedicalRecord loads a visit's record (nil when not yet created).
func (s *PhysicianService) MedicalRecord(ctx context.Context, doctorID, registerID uint) (*model.MedicalRecord, error) {
	if _, err := s.ownedRegister(ctx, doctorID, registerID); err != nil {
		return nil, err
	}
	rec, err := s.records.FindByRegisterID(ctx, registerID)
	if err != nil {
		return nil, notFoundAs(err, nil)
	}
	return rec, nil
}

// SaveMedicalRecord upserts the F2-2 病历首页 with its diagnosed diseases.
func (s *PhysicianService) SaveMedicalRecord(ctx context.Context, doctorID, registerID uint, in dto.MedicalRecordRequest) (*model.MedicalRecord, error) {
	if err := s.ensureInConsult(ctx, doctorID, registerID); err != nil {
		return nil, err
	}
	rec := &model.MedicalRecord{
		RegisterID: registerID, Readme: in.Readme, Present: in.Present, PresentTreat: in.PresentTreat,
		History: in.History, Allergy: in.Allergy, Physique: in.Physique, Proposal: in.Proposal, Careful: in.Careful,
	}
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		return s.records.Upsert(ctx, rec, in.DiseaseIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.records.FindByRegisterID(ctx, registerID)
}

// Diagnose records the diagnosis/handling and optionally ends the visit (F2-8).
func (s *PhysicianService) Diagnose(ctx context.Context, doctorID, registerID uint, in dto.DiagnoseRequest) (*model.Register, error) {
	reg, err := s.ownedRegister(ctx, doctorID, registerID)
	if err != nil {
		return nil, err
	}
	if reg.VisitState != constant.VisitStateInConsult && reg.VisitState != constant.VisitStateFinished {
		return nil, apperr.ErrRegisterState.WithMessage("仅可对接诊中的患者确诊")
	}
	if err := s.records.UpdateDiagnosis(ctx, registerID, in.Diagnosis, in.Cure); err != nil {
		return nil, err
	}
	if in.Finish {
		if err := s.registers.UpdateState(ctx, registerID, constant.VisitStateFinished); err != nil {
			return nil, err
		}
		reg.VisitState = constant.VisitStateFinished
	}
	return reg, nil
}

// WritePrescription opens prescription lines for a visit (F2-9).
func (s *PhysicianService) WritePrescription(ctx context.Context, doctorID, registerID uint, in dto.PrescriptionRequest) (dto.PrescriptionResult, error) {
	if err := s.ensureInConsult(ctx, doctorID, registerID); err != nil {
		return dto.PrescriptionResult{}, err
	}

	now := time.Now()
	var total float64
	items := make([]*model.Prescription, 0, len(in.Items))
	for _, it := range in.Items {
		drug, err := s.drugs.FindByID(ctx, it.DrugID)
		if err != nil {
			return dto.PrescriptionResult{}, notFoundAs(err, apperr.ErrNotFound.WithMessage("药品不存在"))
		}
		items = append(items, &model.Prescription{
			RegisterID: registerID, DrugID: drug.ID, DrugUsage: it.DrugUsage, DrugNumber: it.DrugNumber,
			CreationTime: now, DrugState: constant.PrescriptionStateCreated,
		})
		total += round2(drug.DrugPrice * float64(it.DrugNumber))
	}
	if err := s.prescriptions.CreateBatch(ctx, items); err != nil {
		return dto.PrescriptionResult{}, err
	}
	return dto.PrescriptionResult{RegisterID: registerID, Count: len(items), Total: round2(total)}, nil
}

// History lists the doctor's already-consulted patients (F2-5).
func (s *PhysicianService) History(ctx context.Context, doctorID uint, f repository.RegisterFilter, page repository.Page) ([]dto.RegisterBrief, int64, error) {
	f.EmployeeID = doctorID
	f.States = []int{constant.VisitStateInConsult, constant.VisitStateFinished}
	rows, total, err := s.registers.List(ctx, f, page)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewRegisterBriefs(rows), total, nil
}

// ownedRegister loads a register and asserts it belongs to the doctor.
func (s *PhysicianService) ownedRegister(ctx context.Context, doctorID, registerID uint) (*model.Register, error) {
	reg, err := s.registers.FindByID(ctx, registerID)
	if err != nil {
		return nil, notFoundAs(err, apperr.ErrRegisterNotFound)
	}
	if reg.EmployeeID != doctorID {
		return nil, apperr.ErrForbidden.WithMessage("无权操作其他医生的患者")
	}
	return reg, nil
}

func (s *PhysicianService) ensureInConsult(ctx context.Context, doctorID, registerID uint) error {
	reg, err := s.ownedRegister(ctx, doctorID, registerID)
	if err != nil {
		return err
	}
	if reg.VisitState != constant.VisitStateInConsult && reg.VisitState != constant.VisitStateFinished {
		return apperr.ErrRegisterState.WithMessage("请先创建病历开始看诊")
	}
	return nil
}

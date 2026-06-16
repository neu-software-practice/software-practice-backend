// Package app is the composition root: it wires repositories → services →
// handlers and exposes the router Deps so cmd/server stays tiny.
package app

import (
	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/handler"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
	"github.com/neu-software-practice/software-practice-backend/internal/router"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// Container holds long-lived application dependencies.
type Container struct {
	cfg    *config.Config
	tokens *jwt.Manager
	deps   router.Deps
}

// NewContainer constructs every layer from the DB connection and config.
func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	tokens := jwt.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	tx := repository.NewTxManager(db)

	// Repositories.
	employees := repository.NewEmployeeRepository(db)
	departments := repository.NewDepartmentRepository(db)
	levels := repository.NewRegistLevelRepository(db)
	settles := repository.NewSettleCategoryRepository(db)
	registers := repository.NewRegisterRepository(db)
	techs := repository.NewMedicalTechnologyRepository(db)
	diseases := repository.NewDiseaseRepository(db)
	drugs := repository.NewDrugInfoRepository(db)
	records := repository.NewMedicalRecordRepository(db)
	prescriptions := repository.NewPrescriptionRepository(db)
	charges := repository.NewChargeRecordRepository(db)
	drugTxns := repository.NewDrugTransactionRepository(db)
	checkRepo := repository.NewRequestRepository[model.CheckRequest, *model.CheckRequest](db)
	inspRepo := repository.NewRequestRepository[model.InspectionRequest, *model.InspectionRequest](db)
	dispRepo := repository.NewRequestRepository[model.DisposalRequest, *model.DisposalRequest](db)

	// Generic request services (one per isomorphic family).
	checkSvc := service.NewRequestService[model.CheckRequest](checkRepo, registers, techs,
		constant.TechTypeCheck, constant.ChargeItemCheck,
		func(reg, tech uint, info, pos, rem string) *model.CheckRequest {
			return &model.CheckRequest{RegisterID: reg, MedicalTechnologyID: tech, CheckInfo: info, CheckPosition: pos, CheckRemark: rem}
		})
	inspSvc := service.NewRequestService[model.InspectionRequest](inspRepo, registers, techs,
		constant.TechTypeInspection, constant.ChargeItemInspection,
		func(reg, tech uint, info, pos, rem string) *model.InspectionRequest {
			return &model.InspectionRequest{RegisterID: reg, MedicalTechnologyID: tech, InspectionInfo: info, InspectionPosition: pos, InspectionRemark: rem}
		})
	dispSvc := service.NewRequestService[model.DisposalRequest](dispRepo, registers, techs,
		constant.TechTypeDisposal, constant.ChargeItemDisposal,
		func(reg, tech uint, info, pos, rem string) *model.DisposalRequest {
			return &model.DisposalRequest{RegisterID: reg, MedicalTechnologyID: tech, DisposalInfo: info, DisposalPosition: pos, DisposalRemark: rem}
		})

	// Services.
	authSvc := service.NewAuthService(employees, tokens)
	catalogSvc := service.NewCatalogService(departments, levels, settles, employees, techs, diseases, drugs)
	regSvc := service.NewRegistrationService(registers, levels, departments, settles, employees, charges, tx)
	chargeSvc := service.NewChargeService(registers, prescriptions, charges, tx, checkSvc, inspSvc, dispSvc)
	physSvc := service.NewPhysicianService(registers, records, prescriptions, drugs, tx)
	pharmSvc := service.NewPharmacyService(prescriptions, registers, drugs, drugTxns, tx)

	deps := router.Deps{
		Cfg:          cfg,
		Tokens:       tokens,
		Auth:         handler.NewAuthHandler(authSvc),
		Catalog:      handler.NewCatalogHandler(catalogSvc),
		Registration: handler.NewRegistrationHandler(regSvc),
		Charge:       handler.NewChargeHandler(chargeSvc),
		Physician:    handler.NewPhysicianHandler(physSvc),
		Pharmacy:     handler.NewPharmacyHandler(pharmSvc),
		Check:        handler.NewRequestHandler[model.CheckRequest](checkSvc),
		Inspection:   handler.NewRequestHandler[model.InspectionRequest](inspSvc),
		Disposal:     handler.NewRequestHandler[model.DisposalRequest](dispSvc),
	}

	return &Container{cfg: cfg, tokens: tokens, deps: deps}
}

// Deps returns the router dependencies.
func (c *Container) Deps() router.Deps { return c.deps }

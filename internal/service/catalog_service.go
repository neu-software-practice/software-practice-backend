package service

import (
	"context"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// CatalogService serves reference data and search endpoints used by the UI's
// dropdowns and project/disease/drug pickers (F1-1, F2-2, F2-3/4, F2-9).
type CatalogService struct {
	departments repository.DepartmentRepository
	levels      repository.RegistLevelRepository
	settles     repository.SettleCategoryRepository
	employees   repository.EmployeeRepository
	techs       repository.MedicalTechnologyRepository
	diseases    repository.DiseaseRepository
	drugs       repository.DrugInfoRepository
}

// NewCatalogService wires the CatalogService.
func NewCatalogService(
	departments repository.DepartmentRepository,
	levels repository.RegistLevelRepository,
	settles repository.SettleCategoryRepository,
	employees repository.EmployeeRepository,
	techs repository.MedicalTechnologyRepository,
	diseases repository.DiseaseRepository,
	drugs repository.DrugInfoRepository,
) *CatalogService {
	return &CatalogService{departments: departments, levels: levels, settles: settles, employees: employees, techs: techs, diseases: diseases, drugs: drugs}
}

// Departments lists departments, optionally filtered by dept_type.
func (s *CatalogService) Departments(ctx context.Context, deptType string) ([]model.Department, error) {
	if deptType != "" {
		return s.departments.ListByType(ctx, deptType)
	}
	return s.departments.List(ctx)
}

// RegistLevels lists active registration levels.
func (s *CatalogService) RegistLevels(ctx context.Context) ([]model.RegistLevel, error) {
	return s.levels.List(ctx)
}

// SettleCategories lists active settlement categories.
func (s *CatalogService) SettleCategories(ctx context.Context) ([]model.SettleCategory, error) {
	return s.settles.List(ctx)
}

// Doctors lists on-duty doctors in a department at a registration level (F1-1).
func (s *CatalogService) Doctors(ctx context.Context, deptID, levelID uint) ([]model.Employee, error) {
	return s.employees.ListDoctors(ctx, deptID, levelID)
}

// MedicalTechnologies searches the project catalog (F2-3/F2-4/F2-10).
func (s *CatalogService) MedicalTechnologies(ctx context.Context, keyword, techType string, page repository.Page) ([]model.MedicalTechnology, int64, error) {
	return s.techs.Search(ctx, keyword, techType, page)
}

// Diseases searches the disease catalog (F2-2).
func (s *CatalogService) Diseases(ctx context.Context, keyword string, page repository.Page) ([]model.Disease, int64, error) {
	return s.diseases.Search(ctx, keyword, page)
}

// Drugs searches the drug catalog (F2-9, F5-1).
func (s *CatalogService) Drugs(ctx context.Context, keyword string, page repository.Page) ([]model.DrugInfo, int64, error) {
	return s.drugs.Search(ctx, keyword, page)
}

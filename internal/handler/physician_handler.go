package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// PhysicianHandler serves the outpatient doctor workflow (F2-*).
type PhysicianHandler struct{ svc *service.PhysicianService }

// NewPhysicianHandler builds the PhysicianHandler.
func NewPhysicianHandler(svc *service.PhysicianService) *PhysicianHandler {
	return &PhysicianHandler{svc: svc}
}

func (h *PhysicianHandler) listFilter(c *gin.Context) repository.RegisterFilter {
	f := repository.RegisterFilter{CaseNumber: c.Query("case_number"), Name: c.Query("name")}
	if s := c.Query("state"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			f.States = []int{v}
		}
	}
	return f
}

// Patients lists the logged-in doctor's patients (F2-1).
func (h *PhysicianHandler) Patients(c *gin.Context) {
	page := parsePage(c)
	briefs, total, err := h.svc.Patients(c.Request.Context(), middleware.CurrentEmployeeID(c), h.listFilter(c), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, briefs, metaFor(page, total))
}

// Counts returns the F2-1 header counters.
func (h *PhysicianHandler) Counts(c *gin.Context) {
	counts, err := h.svc.PatientCounts(c.Request.Context(), middleware.CurrentEmployeeID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, counts)
}

// Consult starts a consultation (F2-1 创建病历).
func (h *PhysicianHandler) Consult(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	reg, err := h.svc.Consult(c.Request.Context(), middleware.CurrentEmployeeID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, dto.NewRegisterBrief(reg))
}

// GetMedicalRecord loads a visit's record (F2-2).
func (h *PhysicianHandler) GetMedicalRecord(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	rec, err := h.svc.MedicalRecord(c.Request.Context(), middleware.CurrentEmployeeID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rec)
}

// SaveMedicalRecord upserts a visit's record (F2-2).
func (h *PhysicianHandler) SaveMedicalRecord(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.MedicalRecordRequest
	if !bindJSON(c, &in) {
		return
	}
	rec, err := h.svc.SaveMedicalRecord(c.Request.Context(), middleware.CurrentEmployeeID(c), id, in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rec)
}

// History lists the doctor's consulted patients (F2-5).
func (h *PhysicianHandler) History(c *gin.Context) {
	page := parsePage(c)
	briefs, total, err := h.svc.History(c.Request.Context(), middleware.CurrentEmployeeID(c), h.listFilter(c), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, briefs, metaFor(page, total))
}

// Diagnose records the diagnosis and optionally ends the visit (F2-8).
func (h *PhysicianHandler) Diagnose(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.DiagnoseRequest
	if !bindJSON(c, &in) {
		return
	}
	reg, err := h.svc.Diagnose(c.Request.Context(), middleware.CurrentEmployeeID(c), id, in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, dto.NewRegisterBrief(reg))
}

// WritePrescription opens prescription lines (F2-9).
func (h *PhysicianHandler) WritePrescription(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.PrescriptionRequest
	if !bindJSON(c, &in) {
		return
	}
	res, err := h.svc.WritePrescription(c.Request.Context(), middleware.CurrentEmployeeID(c), id, in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res)
}

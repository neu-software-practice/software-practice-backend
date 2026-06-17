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

// Patients godoc
// @Summary  患者查看 (F2-1)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    name         query     string  false  "姓名"
// @Param    state        query     int     false  "看诊状态"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /physician/patients [get]
func (h *PhysicianHandler) Patients(c *gin.Context) {
	page := parsePage(c)
	briefs, total, err := h.svc.Patients(c.Request.Context(), middleware.CurrentEmployeeID(c), h.listFilter(c), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, briefs, metaFor(page, total))
}

// Counts godoc
// @Summary  患者统计 (排队/已看诊, F2-1)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Router   /physician/patients/counts [get]
func (h *PhysicianHandler) Counts(c *gin.Context) {
	counts, err := h.svc.PatientCounts(c.Request.Context(), middleware.CurrentEmployeeID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, counts)
}

// Consult godoc
// @Summary  创建病历开始看诊 (F2-1)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    id   path      int  true  "挂号ID"
// @Success  200  {object}  response.Body
// @Router   /physician/registers/{id}/consult [post]
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

// GetMedicalRecord godoc
// @Summary  读取病历首页 (F2-2)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    id   path      int  true  "挂号ID"
// @Success  200  {object}  response.Body
// @Router   /physician/registers/{id}/medical-record [get]
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

// SaveMedicalRecord godoc
// @Summary  保存病历首页 (F2-2)
// @Tags     physician
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                       true  "挂号ID"
// @Param    body  body      dto.MedicalRecordRequest  true  "病历内容"
// @Success  200   {object}  response.Body
// @Router   /physician/registers/{id}/medical-record [put]
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

// History godoc
// @Summary  看诊记录 (F2-5)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    name         query     string  false  "姓名"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /physician/history [get]
func (h *PhysicianHandler) History(c *gin.Context) {
	page := parsePage(c)
	briefs, total, err := h.svc.History(c.Request.Context(), middleware.CurrentEmployeeID(c), h.listFilter(c), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, briefs, metaFor(page, total))
}

// Diagnose godoc
// @Summary  门诊确诊 (F2-8)
// @Tags     physician
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                  true  "挂号ID"
// @Param    body  body      dto.DiagnoseRequest  true  "诊断结果"
// @Success  200   {object}  response.Body
// @Router   /physician/registers/{id}/diagnosis [put]
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

// WritePrescription godoc
// @Summary  开立处方 (F2-9)
// @Tags     physician
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                      true  "挂号ID"
// @Param    body  body      dto.PrescriptionRequest  true  "处方明细"
// @Success  201   {object}  response.Body
// @Router   /physician/registers/{id}/prescriptions [post]
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

package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// RequestHandler is the generic HTTP handler for the three isomorphic request
// families. The same instance serves the doctor side (Create, Results — guarded
// 门诊) and the tech-doctor side (PendingPatients/Execute/RecordResult — guarded
// 检查/检验/处置); the router applies the right RBAC per route.
type RequestHandler[T any, PT repository.RequestPtr[T]] struct {
	svc *service.RequestService[T, PT]
}

// NewRequestHandler builds a generic RequestHandler.
func NewRequestHandler[T any, PT repository.RequestPtr[T]](svc *service.RequestService[T, PT]) *RequestHandler[T, PT] {
	return &RequestHandler[T, PT]{svc: svc}
}

// Create opens a request (F2-3/F2-4/F2-10). The three concrete paths
// (/check-requests, /inspection-requests, /disposal-requests) are documented in
// api_docs.go since one generic method cannot carry three @Router annotations.
func (h *RequestHandler[T, PT]) Create(c *gin.Context) {
	var in dto.CreateRequestInput
	if !bindJSON(c, &in) {
		return
	}
	view, err := h.svc.Create(c.Request.Context(), in.RegisterID, in.TechID, in.Info, in.Position, in.Remark)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, view)
}

// PendingPatients lists patients awaiting execution (F3-1/F4-1/F6-1).
func (h *RequestHandler[T, PT]) PendingPatients(c *gin.Context) {
	page := parsePage(c)
	f := repository.RegisterFilter{CaseNumber: c.Query("case_number"), Name: c.Query("name")}
	rows, total, err := h.svc.PendingPatients(c.Request.Context(), f, page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, dto.NewRegisterBriefs(rows), metaFor(page, total))
}

// Counts returns the 排队/已完成 header counters.
func (h *RequestHandler[T, PT]) Counts(c *gin.Context) {
	waiting, done, err := h.svc.Counts(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"waiting": waiting, "done": done})
}

// PatientRequests lists a patient's requests in a state (default 已缴费) — F3-2.
func (h *RequestHandler[T, PT]) PatientRequests(c *gin.Context) {
	registerID := parseUintQuery(c, "register_id")
	if registerID == 0 {
		response.Error(c, apperr.ErrBadRequest.WithMessage("缺少 register_id 参数"))
		return
	}
	state := c.DefaultQuery("state", constant.RequestStatePaid)
	views, err := h.svc.PatientRequests(c.Request.Context(), registerID, state)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, views)
}

// Results lists a patient's completed requests (F2-6/F2-7).
func (h *RequestHandler[T, PT]) Results(c *gin.Context) {
	registerID := parseUintQuery(c, "register_id")
	if registerID == 0 {
		response.Error(c, apperr.ErrBadRequest.WithMessage("缺少 register_id 参数"))
		return
	}
	views, err := h.svc.Results(c.Request.Context(), registerID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, views)
}

// Manage lists all of a patient's requests regardless of state (F3-4/F4-4/F6-4).
func (h *RequestHandler[T, PT]) Manage(c *gin.Context) {
	registerID := parseUintQuery(c, "register_id")
	if registerID == 0 {
		response.Error(c, apperr.ErrBadRequest.WithMessage("缺少 register_id 参数"))
		return
	}
	views, err := h.svc.ByRegister(c.Request.Context(), registerID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, views)
}

// Execute assigns an executor (F3-2/F4-2/F6-2). Defaults executor to current user.
func (h *RequestHandler[T, PT]) Execute(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.ExecuteRequestInput
	_ = c.ShouldBindJSON(&in) // body is optional
	executor := in.ExecutorID
	if executor == 0 {
		executor = middleware.CurrentEmployeeID(c)
	}
	view, err := h.svc.Execute(c.Request.Context(), id, executor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, view)
}

// RecordResult records a result (F3-3/F4-3/F6-3). Defaults inputter to current user.
func (h *RequestHandler[T, PT]) RecordResult(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.ResultRequestInput
	if !bindJSON(c, &in) {
		return
	}
	inputter := in.InputterID
	if inputter == 0 {
		inputter = middleware.CurrentEmployeeID(c)
	}
	view, err := h.svc.RecordResult(c.Request.Context(), id, inputter, in.Result)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, view)
}

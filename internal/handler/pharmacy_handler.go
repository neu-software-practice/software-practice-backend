package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// PharmacyHandler serves dispensing (F5-1).
type PharmacyHandler struct{ svc *service.PharmacyService }

// NewPharmacyHandler builds the PharmacyHandler.
func NewPharmacyHandler(svc *service.PharmacyService) *PharmacyHandler {
	return &PharmacyHandler{svc: svc}
}

// Prescriptions lists a patient's prescriptions by case number + state (F5-1).
func (h *PharmacyHandler) Prescriptions(c *gin.Context) {
	caseNumber := c.Query("case_number")
	if caseNumber == "" {
		response.Error(c, apperr.ErrBadRequest.WithMessage("缺少 case_number 参数"))
		return
	}
	list, err := h.svc.Prescriptions(c.Request.Context(), caseNumber, c.Query("state"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

// Dispense godoc
// @Summary  药房发药 (F5-1)
// @Tags     pharmacy
// @Produce  json
// @Security BearerAuth
// @Param    id   path      int  true  "处方ID"
// @Success  200  {object}  response.Body
// @Router   /pharmacy/prescriptions/{id}/dispense [post]
func (h *PharmacyHandler) Dispense(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	p, err := h.svc.Dispense(c.Request.Context(), middleware.CurrentEmployeeID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, p)
}

// Refund returns a dispensed prescription (F5-2 退药).
func (h *PharmacyHandler) Refund(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	p, err := h.svc.Refund(c.Request.Context(), middleware.CurrentEmployeeID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, p)
}

// Transactions lists the dispense/return history (F5-4).
func (h *PharmacyHandler) Transactions(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Transactions(c.Request.Context(), c.Query("case_number"), c.Query("action"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

// CreateDrug adds a drug to the catalog (F5-3).
func (h *PharmacyHandler) CreateDrug(c *gin.Context) {
	var in dto.DrugRequest
	if !bindJSON(c, &in) {
		return
	}
	drug, err := h.svc.CreateDrug(c.Request.Context(), in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, drug)
}

// UpdateDrug edits a drug (F5-3).
func (h *PharmacyHandler) UpdateDrug(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.DrugRequest
	if !bindJSON(c, &in) {
		return
	}
	drug, err := h.svc.UpdateDrug(c.Request.Context(), id, in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, drug)
}

// DeleteDrug soft-deletes a drug (F5-3).
func (h *PharmacyHandler) DeleteDrug(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteDrug(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "deleted": true})
}

// Restock adjusts a drug's stock (F5-3 入库/调整).
func (h *PharmacyHandler) Restock(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in dto.StockRequest
	if !bindJSON(c, &in) {
		return
	}
	drug, err := h.svc.Restock(c.Request.Context(), id, in.Delta)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, drug)
}

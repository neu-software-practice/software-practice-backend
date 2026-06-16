package handler

import (
	"github.com/gin-gonic/gin"

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

// Dispense issues a paid prescription (F5-1).
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

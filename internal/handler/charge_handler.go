package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// ChargeHandler serves charging/refunding (F1-3/F1-4).
type ChargeHandler struct{ svc *service.ChargeService }

// NewChargeHandler builds the ChargeHandler.
func NewChargeHandler(svc *service.ChargeService) *ChargeHandler { return &ChargeHandler{svc: svc} }

// Pending lists a visit's payable items by case number (F1-3).
func (h *ChargeHandler) Pending(c *gin.Context) {
	caseNumber := c.Query("case_number")
	if caseNumber == "" {
		response.Error(c, apperr.ErrBadRequest.WithMessage("缺少 case_number 参数"))
		return
	}
	resp, err := h.svc.PendingItems(c.Request.Context(), caseNumber)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Charge settles selected items (F1-3).
func (h *ChargeHandler) Charge(c *gin.Context) {
	var in dto.ChargeRequest
	if !bindJSON(c, &in) {
		return
	}
	res, err := h.svc.Charge(c.Request.Context(), middleware.CurrentEmployeeID(c), in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

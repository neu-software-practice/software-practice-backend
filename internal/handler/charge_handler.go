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

// Pending godoc
// @Summary  待缴费项目 (F1-3)
// @Tags     charge
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  true  "病历号"
// @Success  200          {object}  response.Body
// @Router   /charges/pending [get]
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

// Charge godoc
// @Summary  收费结算 (F1-3)
// @Tags     charge
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.ChargeRequest  true  "结算项目"
// @Success  200   {object}  response.Body
// @Router   /charges [post]
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

// RefundPending godoc
// @Summary  可退费项目 (F1-4)
// @Tags     charge
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  true  "病历号"
// @Success  200          {object}  response.Body
// @Router   /charges/refund-pending [get]
func (h *ChargeHandler) RefundPending(c *gin.Context) {
	caseNumber := c.Query("case_number")
	if caseNumber == "" {
		response.Error(c, apperr.ErrBadRequest.WithMessage("缺少 case_number 参数"))
		return
	}
	resp, err := h.svc.RefundableItems(c.Request.Context(), caseNumber)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Refund godoc
// @Summary  退费 (F1-4)
// @Tags     charge
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.RefundRequest  true  "退费项目"
// @Success  200   {object}  response.Body
// @Router   /charges/refund [post]
func (h *ChargeHandler) Refund(c *gin.Context) {
	var in dto.RefundRequest
	if !bindJSON(c, &in) {
		return
	}
	res, err := h.svc.Refund(c.Request.Context(), middleware.CurrentEmployeeID(c), in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// Records godoc
// @Summary  费用记录查询 (F1-5 财务 / F2-11 门诊)
// @Tags     charge
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    register_id  query     int     false  "挂号ID"
// @Param    action       query     string  false  "动作(收费/退费)"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /charge-records [get]
func (h *ChargeHandler) Records(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Records(c.Request.Context(), c.Query("case_number"), parseUintQuery(c, "register_id"), c.Query("action"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

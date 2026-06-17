package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// PharmacyHandler serves dispensing, returns, inventory and transactions (F5-*).
type PharmacyHandler struct{ svc *service.PharmacyService }

// NewPharmacyHandler builds the PharmacyHandler.
func NewPharmacyHandler(svc *service.PharmacyService) *PharmacyHandler {
	return &PharmacyHandler{svc: svc}
}

// Prescriptions godoc
// @Summary  待发药/处方查询 (F5-1)
// @Tags     pharmacy
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  true   "病历号"
// @Param    state        query     string  false  "处方状态(默认 已缴费)"
// @Success  200          {object}  response.Body
// @Router   /pharmacy/prescriptions [get]
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

// Refund godoc
// @Summary  药房退药 (F5-2)
// @Tags     pharmacy
// @Produce  json
// @Security BearerAuth
// @Param    id   path      int  true  "处方ID"
// @Success  200  {object}  response.Body
// @Router   /pharmacy/prescriptions/{id}/refund [post]
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

// Transactions godoc
// @Summary  药品交易记录 (F5-4)
// @Tags     pharmacy
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    action       query     string  false  "动作(发药/退药)"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /pharmacy/transactions [get]
func (h *PharmacyHandler) Transactions(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Transactions(c.Request.Context(), c.Query("case_number"), c.Query("action"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

// CreateDrug godoc
// @Summary  药品入库/新增 (F5-3)
// @Tags     pharmacy
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.DrugRequest  true  "药品信息"
// @Success  201   {object}  response.Body
// @Router   /pharmacy/drugs [post]
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

// UpdateDrug godoc
// @Summary  药品信息维护 (F5-3)
// @Tags     pharmacy
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int              true  "药品ID"
// @Param    body  body      dto.DrugRequest  true  "药品信息"
// @Success  200   {object}  response.Body
// @Router   /pharmacy/drugs/{id} [put]
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

// DeleteDrug godoc
// @Summary  药品删除 (F5-3)
// @Tags     pharmacy
// @Produce  json
// @Security BearerAuth
// @Param    id   path      int  true  "药品ID"
// @Success  200  {object}  response.Body
// @Router   /pharmacy/drugs/{id} [delete]
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

// Restock godoc
// @Summary  库存调整/入库 (F5-3)
// @Tags     pharmacy
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int               true  "药品ID"
// @Param    body  body      dto.StockRequest  true  "库存增量(正数入库)"
// @Success  200   {object}  response.Body
// @Router   /pharmacy/drugs/{id}/restock [post]
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

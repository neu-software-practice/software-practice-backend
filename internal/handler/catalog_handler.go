package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// CatalogHandler serves reference data and search endpoints.
type CatalogHandler struct{ svc *service.CatalogService }

// NewCatalogHandler builds the CatalogHandler.
func NewCatalogHandler(svc *service.CatalogService) *CatalogHandler { return &CatalogHandler{svc: svc} }

// Departments godoc
// @Summary  科室列表
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Param    type  query     string  false  "科室类型(财务/门诊/检查/检验/药房/处置)"
// @Success  200   {object}  response.Body
// @Router   /departments [get]
func (h *CatalogHandler) Departments(c *gin.Context) {
	rows, err := h.svc.Departments(c.Request.Context(), c.Query("type"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// RegistLevels godoc
// @Summary  挂号级别列表
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Router   /regist-levels [get]
func (h *CatalogHandler) RegistLevels(c *gin.Context) {
	rows, err := h.svc.RegistLevels(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// SettleCategories godoc
// @Summary  结算类别列表
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Router   /settle-categories [get]
func (h *CatalogHandler) SettleCategories(c *gin.Context) {
	rows, err := h.svc.SettleCategories(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// Doctors godoc
// @Summary  出诊医生列表 (F1-1)
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Param    dept_id          query     int  false  "挂号科室ID"
// @Param    regist_level_id  query     int  false  "挂号级别ID"
// @Success  200              {object}  response.Body
// @Router   /doctors [get]
func (h *CatalogHandler) Doctors(c *gin.Context) {
	rows, err := h.svc.Doctors(c.Request.Context(), parseUintQuery(c, "dept_id"), parseUintQuery(c, "regist_level_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// MedicalTechnologies godoc
// @Summary  医技项目检索 (F2-3/F2-4/F2-10)
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Param    keyword  query     string  false  "编码/名称关键字"
// @Param    type     query     string  false  "类型(检查/检验/处置)"
// @Param    page     query     int     false  "页码"
// @Param    limit    query     int     false  "每页条数"
// @Success  200      {object}  response.Body
// @Router   /medical-technologies [get]
func (h *CatalogHandler) MedicalTechnologies(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.MedicalTechnologies(c.Request.Context(), c.Query("keyword"), c.Query("type"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

// Diseases godoc
// @Summary  疾病检索 (F2-2)
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Param    keyword  query     string  false  "编码/名称/ICD 关键字"
// @Param    page     query     int     false  "页码"
// @Param    limit    query     int     false  "每页条数"
// @Success  200      {object}  response.Body
// @Router   /diseases [get]
func (h *CatalogHandler) Diseases(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Diseases(c.Request.Context(), c.Query("keyword"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

// Drugs godoc
// @Summary  药品检索 (F2-9/F5-1)
// @Tags     catalog
// @Produce  json
// @Security BearerAuth
// @Param    keyword  query     string  false  "编码/名称/拼音码关键字"
// @Param    page     query     int     false  "页码"
// @Param    limit    query     int     false  "每页条数"
// @Success  200      {object}  response.Body
// @Router   /drugs [get]
func (h *CatalogHandler) Drugs(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Drugs(c.Request.Context(), c.Query("keyword"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

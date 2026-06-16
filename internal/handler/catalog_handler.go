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
// @Summary 科室列表 @Tags catalog @Produce json @Param type query string false "科室类型" @Success 200 {object} response.Body @Router /departments [get]
func (h *CatalogHandler) Departments(c *gin.Context) {
	rows, err := h.svc.Departments(c.Request.Context(), c.Query("type"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// RegistLevels lists registration levels.
func (h *CatalogHandler) RegistLevels(c *gin.Context) {
	rows, err := h.svc.RegistLevels(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// SettleCategories lists settlement categories.
func (h *CatalogHandler) SettleCategories(c *gin.Context) {
	rows, err := h.svc.SettleCategories(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// Doctors lists on-duty doctors for a department + level (F1-1).
func (h *CatalogHandler) Doctors(c *gin.Context) {
	rows, err := h.svc.Doctors(c.Request.Context(), parseUintQuery(c, "dept_id"), parseUintQuery(c, "regist_level_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// MedicalTechnologies searches the project catalog.
func (h *CatalogHandler) MedicalTechnologies(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.MedicalTechnologies(c.Request.Context(), c.Query("keyword"), c.Query("type"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

// Diseases searches the disease catalog.
func (h *CatalogHandler) Diseases(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Diseases(c.Request.Context(), c.Query("keyword"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

// Drugs searches the drug catalog.
func (h *CatalogHandler) Drugs(c *gin.Context) {
	page := parsePage(c)
	rows, total, err := h.svc.Drugs(c.Request.Context(), c.Query("keyword"), page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, metaFor(page, total))
}

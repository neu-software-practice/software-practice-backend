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

// RegistrationHandler serves window registration (F1-1/F1-2).
type RegistrationHandler struct{ svc *service.RegistrationService }

// NewRegistrationHandler builds the RegistrationHandler.
func NewRegistrationHandler(svc *service.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{svc: svc}
}

// Register godoc
// @Summary  窗口挂号
// @Tags     registration
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.RegisterRequest  true  "挂号信息"
// @Success  201   {object}  response.Body
// @Failure  404   {object}  response.Body
// @Router   /registers [post]
func (h *RegistrationHandler) Register(c *gin.Context) {
	var in dto.RegisterRequest
	if !bindJSON(c, &in) {
		return
	}
	reg, err := h.svc.Register(c.Request.Context(), middleware.CurrentEmployeeID(c), in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.NewRegisterBrief(reg))
}

// Cancel voids an un-consulted registration (F1-2 退号).
func (h *RegistrationHandler) Cancel(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	reg, err := h.svc.Cancel(c.Request.Context(), middleware.CurrentEmployeeID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, dto.NewRegisterBrief(reg))
}

// List queries registration records (F1-2 查询).
func (h *RegistrationHandler) List(c *gin.Context) {
	page := parsePage(c)
	f := repository.RegisterFilter{CaseNumber: c.Query("case_number"), Name: c.Query("name")}
	if s := c.Query("state"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			f.States = []int{v}
		}
	}
	briefs, total, err := h.svc.List(c.Request.Context(), f, page)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, briefs, metaFor(page, total))
}

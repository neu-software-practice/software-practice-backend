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

// Cancel godoc
// @Summary  窗口退号 (F1-2)
// @Tags     registration
// @Produce  json
// @Security BearerAuth
// @Param    id   path      int  true  "挂号ID"
// @Success  200  {object}  response.Body
// @Failure  409  {object}  response.Body
// @Router   /registers/{id}/cancel [post]
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

// List godoc
// @Summary  挂号记录查询 (F1-2)
// @Tags     registration
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    name         query     string  false  "姓名"
// @Param    state        query     int     false  "看诊状态(1已挂号/2接诊/3结束/4退号)"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /registers [get]
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

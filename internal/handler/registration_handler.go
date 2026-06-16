package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// RegistrationHandler serves window registration (F1-1).
type RegistrationHandler struct{ svc *service.RegistrationService }

// NewRegistrationHandler builds the RegistrationHandler.
func NewRegistrationHandler(svc *service.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{svc: svc}
}

// Register godoc
// @Summary 窗口挂号 @Tags registration @Accept json @Produce json @Param body body dto.RegisterRequest true "挂号信息" @Success 201 {object} response.Body @Router /registers [post]
func (h *RegistrationHandler) Register(c *gin.Context) {
	var in dto.RegisterRequest
	if !bindJSON(c, &in) {
		return
	}
	reg, err := h.svc.Register(c.Request.Context(), in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.NewRegisterBrief(reg))
}

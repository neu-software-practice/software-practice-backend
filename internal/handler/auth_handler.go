package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/service"
)

// AuthHandler exposes login and identity endpoints.
type AuthHandler struct{ svc *service.AuthService }

// NewAuthHandler builds the AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

// Login godoc
// @Summary  登录
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      dto.LoginRequest  true  "登录凭据"
// @Success  200   {object}  response.Body
// @Failure  401   {object}  response.Body
// @Router   /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Me godoc
// @Summary  当前用户
// @Tags     auth
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Failure  401  {object}  response.Body
// @Router   /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	u, err := h.svc.Me(c.Request.Context(), middleware.CurrentEmployeeID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, u)
}

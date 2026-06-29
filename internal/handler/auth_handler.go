package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	authsvc "github.com/neuhis/software-practice-backend/internal/service/auth"
)

// AuthHandler handles authentication HTTP endpoints.
type AuthHandler struct {
	svc *authsvc.Service
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *authsvc.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	input, err := BindJSON[model.RegisterInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrPhoneExists):
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAuthPhoneExists,
				"phone already registered",
				http.StatusConflict,
			))
		case errors.Is(err, model.ErrValidation):
			apperrors.WriteValidationError(c, err.Error())
		default:
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	WriteSuccess(c, http.StatusCreated, resp)
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	input, err := BindJSON[model.LoginInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAuthInvalidCredentials,
				"invalid phone or password",
				http.StatusUnauthorized,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, resp)
}

// Refresh handles POST /auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	input, err := BindJSON[model.RefreshInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	resp, err := h.svc.Refresh(c.Request.Context(), input.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrRefreshTokenInvalid), errors.Is(err, model.ErrRefreshTokenReuse):
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAuthRefreshInvalid,
				"refresh token invalid",
				http.StatusUnauthorized,
			))
		case errors.Is(err, model.ErrRefreshTokenExpired):
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAuthRefreshExpired,
				"refresh token expired",
				http.StatusUnauthorized,
			))
		default:
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	WriteSuccess(c, http.StatusOK, resp)
}

// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	input, err := BindJSON[model.LogoutInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	_ = h.svc.Logout(c.Request.Context(), input.RefreshToken)
	c.Status(http.StatusNoContent)
}

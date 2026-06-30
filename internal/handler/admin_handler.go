package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	adminsvc "github.com/neuhis/software-practice-backend/internal/service/admin"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

// AdminHandler handles admin panel HTTP endpoints.
type AdminHandler struct {
	svc *adminsvc.Service
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(svc *adminsvc.Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// Login handles POST /admin/auth/login
func (h *AdminHandler) Login(c *gin.Context) {
	input, err := BindJSON[model.AdminLoginInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, model.ErrAdminInvalidCredentials) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAdminInvalidCredentials,
				"invalid username or password",
				http.StatusUnauthorized,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, resp)
}

// Logout handles POST /admin/auth/logout
func (h *AdminHandler) Logout(c *gin.Context) {
	input, err := BindJSON[model.AdminLogoutInput](c)
	if err != nil {
		// Even on malformed input, return success (idempotent per spec)
		WriteSuccess(c, http.StatusOK, model.AdminLogoutResult{Success: true})
		return
	}

	_ = h.svc.Logout(c.Request.Context(), input.RefreshToken)

	WriteSuccess(c, http.StatusOK, model.AdminLogoutResult{Success: true})
}

// Refresh handles POST /admin/auth/refresh
func (h *AdminHandler) Refresh(c *gin.Context) {
	input, err := BindJSON[model.AdminRefreshInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	resp, err := h.svc.Refresh(c.Request.Context(), input.RefreshToken)
	if err != nil {
		if errors.Is(err, model.ErrAdminInvalidRefreshToken) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAdminInvalidRefreshToken,
				"refresh token invalid or expired",
				http.StatusUnauthorized,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, model.AdminRefreshResult{Tokens: *resp})
}

// GetDashboardStats handles GET /admin/dashboard/stats
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	stats, err := h.svc.GetDashboardStats(c.Request.Context())
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, stats)
}

// ListPatients handles GET /admin/patients
func (h *AdminHandler) ListPatients(c *gin.Context) {
	query := model.AdminPatientQuery{
		Page:     ParseQueryInt(c, "page", 1),
		PageSize: ParseQueryInt(c, "pageSize", 20),
		Search:   c.Query("search"),
	}

	result, err := h.svc.ListPatients(c.Request.Context(), query)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WritePageResponse(c, result)
}

// GetPatientDetail handles GET /admin/patients/:id
func (h *AdminHandler) GetPatientDetail(c *gin.Context) {
	patientID := c.Param("id")
	if patientID == "" {
		apperrors.WriteValidationError(c, "patient id is required")
		return
	}

	patient, err := h.svc.GetPatientProfile(c.Request.Context(), patientID)
	if err != nil {
		if errors.Is(err, model.ErrPatientNotFound) {
			apperrors.WriteNotFound(c, apperrors.CodePatientNotFound, "patient not found")
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, patient)
}

// ListSessions handles GET /admin/sessions
func (h *AdminHandler) ListSessions(c *gin.Context) {
	query := model.AdminSessionQuery{
		Page:      ParseQueryInt(c, "page", 1),
		PageSize:  ParseQueryInt(c, "pageSize", 20),
		Status:    c.Query("status"),
		PatientID: c.Query("patientId"),
	}

	result, err := h.svc.ListSessions(c.Request.Context(), query)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WritePageResponse(c, result)
}

// GetSessionDetail handles GET /admin/sessions/:id
func (h *AdminHandler) GetSessionDetail(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		apperrors.WriteValidationError(c, "session id is required")
		return
	}

	session, err := h.svc.GetSessionDetail(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, model.ErrSessionNotFound) {
			apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, session)
}

// GetSettings handles GET /admin/settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, settings)
}

// UpdateSettings handles PUT /admin/settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	input, err := BindJSON[model.UpdateSystemSettingsInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	settings, err := h.svc.UpdateSettings(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, model.ErrValidation) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAdminInvalidSettings,
				err.Error(),
				http.StatusBadRequest,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, settings)
}

// WritePageResponse writes a page-based paginated response.
func WritePageResponse[T any](c *gin.Context, pageData *T) {
	// Directly write success with the page response containing items/total/page/pageSize
	// Use type switch to extract fields
	c.JSON(http.StatusOK, api.SuccessResponse(pageData))
}

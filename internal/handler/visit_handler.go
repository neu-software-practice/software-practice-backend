package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	visitsvc "github.com/neuhis/software-practice-backend/internal/service/visit"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

// VisitHandler handles visit session HTTP endpoints.
type VisitHandler struct {
	svc *visitsvc.Service
}

// NewVisitHandler creates a new VisitHandler.
func NewVisitHandler(svc *visitsvc.Service) *VisitHandler {
	return &VisitHandler{svc: svc}
}

// CreateSession handles POST /visits
func (h *VisitHandler) CreateSession(c *gin.Context) {
	input, err := BindJSON[model.CreateSessionInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	if !RequirePatientID(c, input.PatientID) {
		return
	}

	result, err := h.svc.CreateSession(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, model.ErrPatientNotFound) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodePatientNotFound,
				"patient not found",
				http.StatusNotFound,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewValidationError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// CreateFollowUp handles POST /visits/:sessionId/follow-up
func (h *VisitHandler) CreateFollowUp(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[model.CreateFollowUpInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.ParentSessionID = sessionID

	if !RequirePatientID(c, input.PatientID) {
		return
	}

	result, err := h.svc.CreateFollowUp(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, model.ErrPatientNotFound) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodePatientNotFound,
				"patient not found",
				http.StatusNotFound,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewApiError(
			apperrors.CodeSessionNotFound,
			err.Error(),
			http.StatusNotFound,
		))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// ListSessions handles GET /visits
func (h *VisitHandler) ListSessions(c *gin.Context) {
	patientID := c.Query("patientId")
	cursor := api.CursorFromQuery(c.Query("cursor"))
	pageSize := ParseQueryInt(c, "pageSize", 20)

	if patientID == "" {
		apperrors.WriteValidationError(c, "patientId query parameter is required")
		return
	}

	if !RequirePatientID(c, patientID) {
		return
	}

	items, nextCursor, hasMore, err := h.svc.ListSessions(c.Request.Context(), patientID, cursor, pageSize)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WritePageResult(c, api.NewPageResult(items, nextCursor, hasMore))
}

// GetSession handles GET /visits/:sessionId
func (h *VisitHandler) GetSession(c *gin.Context) {
	sessionID := ParseSessionID(c)

	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if errors.Is(err, model.ErrSessionNotFound) {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	if !RequirePatientID(c, session.PatientID) {
		return
	}

	WriteSuccess(c, http.StatusOK, session)
}

// GetSnapshot handles GET /visits/:sessionId/snapshot
func (h *VisitHandler) GetSnapshot(c *gin.Context) {
	sessionID := ParseSessionID(c)

	snapshot, err := h.svc.GetSnapshot(c.Request.Context(), sessionID)
	if errors.Is(err, model.ErrSessionNotFound) {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	if !RequirePatientID(c, snapshot.Session.PatientID) {
		return
	}

	WriteSuccess(c, http.StatusOK, snapshot)
}

// SuspendVisit handles POST /visits/:sessionId/suspend
func (h *VisitHandler) SuspendVisit(c *gin.Context) {
	sessionID := ParseSessionID(c)

	// Verify session exists and patient has access
	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if errors.Is(err, model.ErrSessionNotFound) {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	if !RequirePatientID(c, session.PatientID) {
		return
	}

	result, err := h.svc.SuspendVisit(c.Request.Context(), sessionID)
	if errors.Is(err, model.ErrInvalidState) {
		apperrors.WriteError(c, apperrors.NewApiError(
			apperrors.CodeInvalidState,
			err.Error(),
			http.StatusUnprocessableEntity,
		))
		return
	}
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

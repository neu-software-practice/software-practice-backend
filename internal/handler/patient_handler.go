package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	patientsvc "github.com/neuhis/software-practice-backend/internal/service/patient"
)

// PatientHandler handles patient-related HTTP endpoints.
type PatientHandler struct {
	svc *patientsvc.Service
}

// NewPatientHandler creates a new PatientHandler.
func NewPatientHandler(svc *patientsvc.Service) *PatientHandler {
	return &PatientHandler{svc: svc}
}

// VerifyIdentity handles POST /patients/verify
func (h *PatientHandler) VerifyIdentity(c *gin.Context) {
	input, err := BindJSON[model.VerifyIdentityInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	result, err := h.svc.VerifyIdentity(c.Request.Context(), input)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewApiError(
			apperrors.CodePatientNotFound,
			err.Error(),
			http.StatusNotFound,
		))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// GetContext handles GET /patients/:patientId/context
func (h *PatientHandler) GetContext(c *gin.Context) {
	patientID := ParsePatientID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	ctx2, err := h.svc.GetContext(c.Request.Context(), patientID)
	if errors.Is(err, model.ErrPatientNotFound) {
		apperrors.WriteNotFound(c, apperrors.CodePatientNotFound, "patient not found")
		return
	}
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, ctx2)
}

// UpdateProfile handles PATCH /patients/:patientId/profile
func (h *PatientHandler) UpdateProfile(c *gin.Context) {
	patientID := ParsePatientID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	input, err := BindJSON[model.ProfileUpdateInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.PatientID = patientID

	updated, err := h.svc.UpdateProfile(c.Request.Context(), patientID, input)
	if errors.Is(err, model.ErrPatientNotFound) {
		apperrors.WriteNotFound(c, apperrors.CodePatientNotFound, "patient not found")
		return
	}
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, updated)
}

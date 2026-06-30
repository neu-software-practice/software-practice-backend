package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	medicalordersvc "github.com/neuhis/software-practice-backend/internal/service/medicalorder"
)

// MedicalOrderHandler handles medical-order-related HTTP endpoints.
type MedicalOrderHandler struct {
	svc *medicalordersvc.Service
}

// NewMedicalOrderHandler creates a new MedicalOrderHandler.
func NewMedicalOrderHandler(svc *medicalordersvc.Service) *MedicalOrderHandler {
	return &MedicalOrderHandler{svc: svc}
}

// ListMedicalOrders handles GET /medical-orders
func (h *MedicalOrderHandler) ListMedicalOrders(c *gin.Context) {
	patientID := GetPatientIDFromContext(c)
	if patientID == "" {
		apperrors.WriteUnauthorized(c, "authentication required")
		return
	}

	result, err := h.svc.ListMedicalOrders(c.Request.Context(), patientID)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	billingsvc "github.com/neuhis/software-practice-backend/internal/service/billing"
)

// BillingHandler handles billing-related HTTP endpoints.
type BillingHandler struct {
	svc *billingsvc.Service
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(svc *billingsvc.Service) *BillingHandler {
	return &BillingHandler{svc: svc}
}

// ListBillingRecords handles GET /billing/records
func (h *BillingHandler) ListBillingRecords(c *gin.Context) {
	patientID := GetPatientIDFromContext(c)
	if patientID == "" {
		apperrors.WriteUnauthorized(c, "authentication required")
		return
	}

	result, err := h.svc.ListBillingRecords(c.Request.Context(), patientID)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

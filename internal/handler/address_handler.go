package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	addresssvc "github.com/neuhis/software-practice-backend/internal/service/address"
)

// AddressHandler handles address book HTTP endpoints.
type AddressHandler struct {
	svc *addresssvc.Service
}

// NewAddressHandler creates a new AddressHandler.
func NewAddressHandler(svc *addresssvc.Service) *AddressHandler {
	return &AddressHandler{svc: svc}
}

// ListAddresses handles GET /patients/:patientId/addresses
func (h *AddressHandler) ListAddresses(c *gin.Context) {
	patientID := ParsePatientID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	result, err := h.svc.ListAddresses(c.Request.Context(), patientID)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// CreateAddress handles POST /patients/:patientId/addresses
func (h *AddressHandler) CreateAddress(c *gin.Context) {
	patientID := ParsePatientID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	input, err := BindJSON[model.CreateAddressInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.PatientID = patientID

	result, err := h.svc.CreateAddress(c.Request.Context(), patientID, input)
	if err != nil {
		switch err {
		case model.ErrAddressLimitExceeded:
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeAddressLimitExceeded, err.Error(), http.StatusBadRequest))
		default:
			apperrors.WriteValidationError(c, err.Error())
		}
		return
	}

	WriteSuccess(c, http.StatusCreated, result)
}

// UpdateAddress handles PATCH /patients/:patientId/addresses/:addressId
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	patientID := ParsePatientID(c)
	addressID := ParseAddressID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	input, err := BindJSON[model.UpdateAddressInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	result, err := h.svc.UpdateAddress(c.Request.Context(), patientID, addressID, input)
	if err != nil {
		switch err {
		case model.ErrAddressNotFound:
			apperrors.WriteNotFound(c, apperrors.CodeAddressNotFound, "address not found")
		default:
			apperrors.WriteValidationError(c, err.Error())
		}
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// DeleteAddress handles DELETE /patients/:patientId/addresses/:addressId
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	patientID := ParsePatientID(c)
	addressID := ParseAddressID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	result, err := h.svc.DeleteAddress(c.Request.Context(), patientID, addressID)
	if err != nil {
		switch err {
		case model.ErrAddressNotFound:
			apperrors.WriteNotFound(c, apperrors.CodeAddressNotFound, "address not found")
		default:
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// SetDefaultAddress handles PUT /patients/:patientId/addresses/:addressId/default
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	patientID := ParsePatientID(c)
	addressID := ParseAddressID(c)

	if !RequirePatientID(c, patientID) {
		return
	}

	result, err := h.svc.SetDefaultAddress(c.Request.Context(), patientID, addressID)
	if err != nil {
		switch err {
		case model.ErrAddressNotFound:
			apperrors.WriteNotFound(c, apperrors.CodeAddressNotFound, "address not found")
		default:
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

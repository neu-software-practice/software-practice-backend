package errors

// fmt is used for error formatting

// ApiError is a structured error for API responses.
type ApiError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Status    int         `json:"status,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Retriable bool        `json:"retriable,omitempty"`
}

// NewApiError creates a new ApiError with the given code, message, and status.
// Retriable is set to false for SESSION_NOT_FOUND, PATIENT_NOT_FOUND, and
// VALIDATION_ERROR. CARD_NOT_FOUND is always retriable (spec: card
// updated/invalidated, prompt refresh). For all other codes, retriable is
// true when the status is 5xx.
func NewApiError(code, message string, status int) *ApiError {
	retriable := status >= 500
	switch code {
	case CodeSessionNotFound, CodePatientNotFound, CodeValidationError:
		retriable = false
	case CodeCardNotFound:
		retriable = true
	}
	return &ApiError{
		Code:      code,
		Message:   message,
		Status:    status,
		Retriable: retriable,
	}
}

// Error returns the error message string, implementing the error interface.
func (e *ApiError) Error() string {
	return e.Message
}

// NewValidationError creates a validation error (422).
func NewValidationError(message string) *ApiError {
	return NewApiError(CodeValidationError, message, 422)
}

// NewNotFoundError creates a not found error (404).
func NewNotFoundError(code, message string) *ApiError {
	return NewApiError(code, message, 404)
}

// NewUnauthorizedError creates an unauthorized error (401).
func NewUnauthorizedError(message string) *ApiError {
	return NewApiError(CodeUnauthorized, message, 401)
}

// NewForbiddenError creates a forbidden error (403).
func NewForbiddenError(message string) *ApiError {
	return NewApiError(CodeForbidden, message, 403)
}

// NewInternalError creates an internal server error (500).
func NewInternalError(message string) *ApiError {
	return NewApiError(CodeInternalError, message, 500)
}

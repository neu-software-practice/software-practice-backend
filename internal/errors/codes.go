package errors

const (
	CodeSessionNotFound = "SESSION_NOT_FOUND"
	CodePatientNotFound = "PATIENT_NOT_FOUND"
	CodeCardNotFound    = "CARD_NOT_FOUND"
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnknownError    = "UNKNOWN_ERROR"
	CodeNetworkError    = "NETWORK_ERROR"
	CodeUnauthorized    = "HTTP_401"
	CodeForbidden       = "HTTP_403"
	CodeNotFound        = "HTTP_404"
	CodeTimeout         = "HTTP_408"
	CodeInternalError   = "HTTP_500"
)

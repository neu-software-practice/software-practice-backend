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

	CodeAuthPhoneExists        = "AUTH_PHONE_EXISTS"
	CodeAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS" // #nosec G101
	CodeAuthTokenExpired       = "AUTH_TOKEN_EXPIRED"       // #nosec G101
	CodeAuthRefreshInvalid     = "AUTH_REFRESH_INVALID"
	CodeAuthRefreshExpired     = "AUTH_REFRESH_EXPIRED"
	CodeRateLimited            = "RATE_LIMITED"

	CodeTitleAlreadyExists = "TITLE_ALREADY_EXISTS"
	CodeLLMUnavailable     = "LLM_UNAVAILABLE"
)

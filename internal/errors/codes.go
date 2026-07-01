package errors

const (
	CodeSessionNotFound = "SESSION_NOT_FOUND"
	CodePatientNotFound = "PATIENT_NOT_FOUND"
	CodeCardNotFound    = "CARD_NOT_FOUND"
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnknownError    = "UNKNOWN_ERROR"
	CodeNetworkError    = "NETWORK_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeInternalError   = "INTERNAL_ERROR"

	CodeAuthPhoneExists        = "AUTH_PHONE_EXISTS"
	CodeAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS" // #nosec G101
	CodeAuthTokenExpired       = "AUTH_TOKEN_EXPIRED"       // #nosec G101
	CodeAuthRefreshInvalid     = "AUTH_REFRESH_INVALID"
	CodeAuthRefreshExpired     = "AUTH_REFRESH_EXPIRED"
	CodeRateLimited            = "RATE_LIMITED"

	CodeInvalidState = "INVALID_STATE"

	CodeAddressNotFound      = "ADDRESS_NOT_FOUND"
	CodeAddressLimitExceeded = "ADDRESS_LIMIT_EXCEEDED"
	CodeAddressRequired      = "ADDRESS_REQUIRED"

	// Admin error codes
	CodeAdminInvalidCredentials  = "INVALID_CREDENTIALS"   // #nosec G101
	CodeAdminInvalidRefreshToken = "INVALID_REFRESH_TOKEN" // #nosec G101
	CodeAdminInvalidSettings     = "INVALID_SETTINGS"
)

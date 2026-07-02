package errors

import (
	"github.com/gin-gonic/gin"
)

// WriteError sends the ApiError as a JSON response using the standard API envelope.
func WriteError(c *gin.Context, err *ApiError) {
	c.AbortWithStatusJSON(err.Status, err)
}

// WriteValidationError sends a 422 VALIDATION_ERROR response.
func WriteValidationError(c *gin.Context, message string) {
	WriteError(c, NewApiError(CodeValidationError, message, 422))
}

// WriteNotFound sends a 404 not found response with the given code and message.
func WriteNotFound(c *gin.Context, code, message string) {
	WriteError(c, NewApiError(code, message, 404))
}

// WriteUnauthorized sends a 401 unauthorized response.
func WriteUnauthorized(c *gin.Context, message string) {
	WriteError(c, NewApiError(CodeUnauthorized, message, 401))
}

// WriteForbidden sends a 403 forbidden response.
func WriteForbidden(c *gin.Context, message string) {
	WriteError(c, NewApiError(CodeForbidden, message, 403))
}

// WriteInternalError sends a 500 internal server error response.
func WriteInternalError(c *gin.Context, message string) {
	WriteError(c, NewApiError(CodeInternalError, message, 500))
}

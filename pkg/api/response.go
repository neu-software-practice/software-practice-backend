package api

// ApiResponse is a generic response envelope for all API responses.
type ApiResponse[T any] struct {
	Success bool        `json:"success"`
	Data    *T          `json:"data"`
	Error   interface{} `json:"error"`
}

// SuccessResponse creates a success response with the provided data.
func SuccessResponse[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{
		Success: true,
		Data:    &data,
		Error:   nil,
	}
}

// ErrorResponse creates an error response with the provided error details.
func ErrorResponse(err interface{}) ApiResponse[interface{}] {
	return ApiResponse[interface{}]{
		Success: false,
		Data:    nil,
		Error:   err,
	}
}

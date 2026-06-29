package api

// ApiResponse is a generic response envelope for all API responses.
type ApiResponse[T any] struct {
	Success bool        `json:"success"`
	Data    *T          `json:"data"`
	Error   interface{} `json:"error"`
	Meta    interface{} `json:"meta,omitempty"`
}

// SuccessResponse creates a success response with the provided data.
func SuccessResponse[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{
		Success: true,
		Data:    &data,
		Error:   nil,
	}
}

// SuccessResponseWithMeta creates a success response with metadata.
func SuccessResponseWithMeta[T any](data T, meta interface{}) ApiResponse[T] {
	return ApiResponse[T]{
		Success: true,
		Data:    &data,
		Meta:    meta,
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

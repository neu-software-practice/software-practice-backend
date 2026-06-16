// Package apperr defines business error codes mapped to HTTP statuses (PLAN §4).
// Handlers return these from the service layer; the response package renders
// them into the unified envelope. AppError values are treated as immutable —
// WithMessage returns a copy rather than mutating the shared sentinel.
package apperr

import "net/http"

// AppError is a business error carrying a stable machine code, a human message
// and the HTTP status it maps to.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

// New constructs an AppError. Prefer reusing the package-level sentinels below.
func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

// WithMessage returns a copy of the error with a caller-supplied message,
// leaving the original sentinel untouched (immutable style, coding-style rule).
func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{Code: e.Code, Message: message, Status: e.Status}
}

// Generic errors.
var (
	ErrBadRequest   = New("BAD_REQUEST", "请求参数错误", http.StatusBadRequest)
	ErrValidation   = New("VALIDATION_ERROR", "输入校验失败", http.StatusUnprocessableEntity)
	ErrUnauthorized = New("UNAUTHORIZED", "未认证或登录态已失效", http.StatusUnauthorized)
	ErrForbidden    = New("FORBIDDEN", "无权访问该资源", http.StatusForbidden)
	ErrNotFound     = New("NOT_FOUND", "资源不存在", http.StatusNotFound)
	ErrConflict     = New("CONFLICT", "资源状态冲突，操作无法完成", http.StatusConflict)
	ErrInternal     = New("INTERNAL_ERROR", "服务器内部错误", http.StatusInternalServerError)
)

// Domain errors.
var (
	ErrInvalidCredentials = New("INVALID_CREDENTIALS", "账号或密码错误", http.StatusUnauthorized)
	ErrEmployeeNotFound   = New("EMPLOYEE_NOT_FOUND", "员工不存在", http.StatusNotFound)
	ErrRegisterNotFound   = New("REGISTER_NOT_FOUND", "挂号记录不存在", http.StatusNotFound)
	ErrRegisterState      = New("REGISTER_STATE_INVALID", "挂号状态不允许该操作", http.StatusConflict)
	ErrRequestNotFound    = New("REQUEST_NOT_FOUND", "申请单不存在", http.StatusNotFound)
	ErrRequestState       = New("REQUEST_STATE_INVALID", "申请单状态不允许该操作", http.StatusConflict)
	ErrPrescriptionState  = New("PRESCRIPTION_STATE_INVALID", "处方状态不允许该操作", http.StatusConflict)
	ErrMedicalRecord      = New("MEDICAL_RECORD_INVALID", "病历数据无效", http.StatusUnprocessableEntity)
	ErrNoChargeItems      = New("NO_CHARGE_ITEMS", "没有可结算的费用项目", http.StatusUnprocessableEntity)
	ErrTechTypeMismatch   = New("TECH_TYPE_MISMATCH", "医技项目类型与申请类型不匹配", http.StatusUnprocessableEntity)
)

package model

import "errors"

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrPatientNotFound = errors.New("patient not found")
	ErrCardNotFound    = errors.New("flow card not found")
	ErrValidation      = errors.New("validation error")
	ErrSessionClosed   = errors.New("session is closed")
	ErrWrongStep       = errors.New("wrong step for current session state")
	ErrUnauthorized    = errors.New("unauthorized access")
	ErrForbidden       = errors.New("forbidden")
	ErrEmergency       = errors.New("emergency detected")

	ErrUserNotFound        = errors.New("user not found")
	ErrPhoneExists         = errors.New("phone already registered")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrRefreshTokenInvalid = errors.New("refresh token invalid")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenReuse   = errors.New("refresh token reuse detected")
)

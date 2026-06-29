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
)

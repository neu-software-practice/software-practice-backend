package service

import (
	"errors"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// notFoundAs maps a repository.ErrNotFound to a domain apperr, passing any other
// error through unchanged. Centralizes the repo→service error translation.
func notFoundAs(err error, appErr *apperr.AppError) error {
	if errors.Is(err, repository.ErrNotFound) {
		return appErr
	}
	return err
}

// round2 rounds a money amount to 2 decimals, taming float64 accumulation in
// totals (acceptable for this demo's DECIMAL(8,2) range).
func round2(v float64) float64 {
	if v < 0 {
		return -round2(-v)
	}
	return float64(int64(v*100+0.5)) / 100
}

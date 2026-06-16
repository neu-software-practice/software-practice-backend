// Package service holds the business logic (PLAN §2.1). Services depend on
// repository interfaces (mockable) and translate repository.ErrNotFound and
// domain rules into apperr values for the handler layer.
package service

import (
	"context"
	"errors"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/hash"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// AuthService handles login and identity (SPEC §7.1).
type AuthService struct {
	employees repository.EmployeeRepository
	tokens    *jwt.Manager
}

// NewAuthService wires the AuthService.
func NewAuthService(employees repository.EmployeeRepository, tokens *jwt.Manager) *AuthService {
	return &AuthService{employees: employees, tokens: tokens}
}

// Login verifies credentials and issues a JWT. To avoid user enumeration, both
// "unknown user" and "wrong password" return the same generic error.
func (s *AuthService) Login(ctx context.Context, in dto.LoginRequest) (*dto.LoginResponse, error) {
	emp, err := s.employees.FindByUsername(ctx, in.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperr.ErrInvalidCredentials
		}
		return nil, err
	}
	if !hash.Verify(emp.Password, in.Password) {
		return nil, apperr.ErrInvalidCredentials
	}

	deptType := ""
	if emp.Department != nil {
		deptType = emp.Department.DeptType
	}
	token, err := s.tokens.Generate(emp.ID, emp.Realname, deptType)
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{Token: token, User: dto.NewUserInfo(emp)}, nil
}

// Me returns the current user's profile from a validated token's employee id.
func (s *AuthService) Me(ctx context.Context, employeeID uint) (*dto.UserInfo, error) {
	emp, err := s.employees.FindByID(ctx, employeeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperr.ErrUnauthorized
		}
		return nil, err
	}
	u := dto.NewUserInfo(emp)
	return &u, nil
}

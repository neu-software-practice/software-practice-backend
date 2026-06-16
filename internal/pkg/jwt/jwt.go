// Package jwt signs and parses the access tokens issued at login (SPEC §7.1).
// The token carries the employee id, realname and dept_type so the auth
// middleware can authorize by role without a DB round-trip.
package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const issuer = "his-backend"

// Claims is the JWT payload.
type Claims struct {
	EmployeeID uint   `json:"employee_id"`
	Realname   string `json:"realname"`
	DeptType   string `json:"dept_type"`
	jwtlib.RegisteredClaims
}

// Manager signs and verifies tokens with a single HMAC secret.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager builds a Manager. The secret must come from configuration/env.
func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// Generate signs a token for the given employee.
func (m *Manager) Generate(employeeID uint, realname, deptType string) (string, error) {
	now := time.Now()
	claims := Claims{
		EmployeeID: employeeID,
		Realname:   realname,
		DeptType:   deptType,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.ttl)),
		},
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates the token string and returns its claims.
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

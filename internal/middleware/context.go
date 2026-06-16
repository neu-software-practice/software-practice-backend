// Package middleware provides Gin middleware: JWT auth, RBAC by dept_type, CORS,
// panic recovery and request logging (PLAN §2.1).
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
)

const claimsKey = "auth_claims"

// setClaims stores validated claims on the request context.
func setClaims(c *gin.Context, claims *jwt.Claims) {
	c.Set(claimsKey, claims)
}

// CurrentClaims returns the authenticated user's claims, if present.
func CurrentClaims(c *gin.Context) (*jwt.Claims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*jwt.Claims)
	return claims, ok
}

// CurrentEmployeeID returns the authenticated employee id (0 if unauthenticated).
func CurrentEmployeeID(c *gin.Context) uint {
	if claims, ok := CurrentClaims(c); ok {
		return claims.EmployeeID
	}
	return 0
}

// CurrentDeptType returns the authenticated user's dept_type ("" if none).
func CurrentDeptType(c *gin.Context) string {
	if claims, ok := CurrentClaims(c); ok {
		return claims.DeptType
	}
	return ""
}

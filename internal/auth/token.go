package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateAccessToken creates a patient JWT access token with standard claims.
// TTL is fixed at 900 seconds (15 minutes).
func GenerateAccessToken(userID, patientID, phone, secret string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       userID,
		"patientId": patientID,
		"phone":     phone,
		"iat":       now.Unix(),
		"exp":       now.Add(900 * time.Second).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return tokenString, nil
}

// GenerateAdminAccessToken creates an admin JWT access token with a configurable TTL.
func GenerateAdminAccessToken(adminID, role, secret string, ttlSeconds int) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  adminID,
		"role": role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(ttlSeconds) * time.Second).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("generate admin access token: %w", err)
	}
	return tokenString, nil
}

// ParseJWT parses and validates a JWT token string, returning the claims map.
func ParseJWT(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

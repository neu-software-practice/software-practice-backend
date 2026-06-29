package model

import "time"

// User represents a registered user account.
type User struct {
	ID           string    `json:"id"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"`
	RealName     string    `json:"realName"`
	PatientID    string    `json:"patientId"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// RefreshToken represents a stored refresh token record.
type RefreshToken struct {
	ID        string     `json:"id"`
	TokenHash string     `json:"-"`
	UserID    string     `json:"userId"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// RegisterInput carries the fields for user registration.
type RegisterInput struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
	RealName string `json:"realName,omitempty"`
}

// LoginInput carries the fields for user login.
type LoginInput struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// RefreshInput carries the refresh token for token rotation.
type RefreshInput struct {
	RefreshToken string `json:"refreshToken"`
}

// LogoutInput carries the refresh token to invalidate.
type LogoutInput struct {
	RefreshToken string `json:"refreshToken"`
}

// AuthResponse is the response returned by auth endpoints.
type AuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int       `json:"expiresIn"`
	User         *UserInfo `json:"user,omitempty"`
}

// UserInfo contains basic user information returned in auth responses.
type UserInfo struct {
	UserID    string `json:"userId"`
	PatientID string `json:"patientId"`
	Phone     string `json:"phone"`
	RealName  string `json:"realName"`
}

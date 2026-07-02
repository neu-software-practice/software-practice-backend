package model

import "time"

// AdminRole represents the role of an admin user.
// Used in tests only.
type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "super_admin"
	AdminRoleAdmin      AdminRole = "admin"
	AdminRoleOperator   AdminRole = "operator"
)

// AdminUser represents an administrator account stored in the database.
type AdminUser struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         AdminRole `json:"role"`
	DisplayName  string    `json:"displayName"`
	CreatedAt    time.Time `json:"createdAt"`
}

// AdminRefreshToken represents a stored admin refresh token.
type AdminRefreshToken struct {
	ID        string     `json:"id"`
	TokenHash string     `json:"-"`
	AdminID   string     `json:"adminId"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// AdminTokens contains the access and refresh token pair for admin authentication.
type AdminTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// AdminLoginInput carries the fields for admin login.
type AdminLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password" binding:"required"`
}

// AdminLoginResult is the response for a successful admin login.
type AdminLoginResult struct {
	Tokens AdminTokens `json:"tokens"`
	User   AdminUser   `json:"user"`
}

// AdminLogoutInput carries the refresh token to invalidate.
type AdminLogoutInput struct {
	RefreshToken string `json:"refreshToken"`
}

// AdminLogoutResult is the response for admin logout.
type AdminLogoutResult struct {
	Success bool `json:"success"`
}

// AdminRefreshInput carries the refresh token for token rotation.
type AdminRefreshInput struct {
	RefreshToken string `json:"refreshToken" binding:"required,min=1"`
}

// AdminRefreshResult is the response for a successful token refresh.
type AdminRefreshResult struct {
	Tokens AdminTokens `json:"tokens"`
}

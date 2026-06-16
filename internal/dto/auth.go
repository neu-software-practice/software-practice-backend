// Package dto holds the request/response shapes exchanged over the API. Keeping
// them separate from GORM models lets the wire contract evolve independently and
// keeps validation tags out of the persistence layer.
package dto

import "github.com/neu-software-practice/software-practice-backend/internal/model"

// LoginRequest is the POST /api/auth/login body.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserInfo is the authenticated user's public profile.
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Realname string `json:"realname"`
	DeptID   uint   `json:"dept_id"`
	DeptName string `json:"dept_name"`
	DeptType string `json:"dept_type"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// NewUserInfo projects an employee (with its Department preloaded) into UserInfo.
func NewUserInfo(e *model.Employee) UserInfo {
	u := UserInfo{
		ID:       e.ID,
		Username: e.Username,
		Realname: e.Realname,
		DeptID:   e.DeptmentID,
	}
	if e.Department != nil {
		u.DeptName = e.Department.DeptName
		u.DeptType = e.Department.DeptType
	}
	return u
}

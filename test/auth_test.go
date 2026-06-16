package test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
)

func TestHealth(t *testing.T) {
	engine, _ := newServer(t)
	rec, env := doJSON(t, engine, http.MethodGet, "/api/health", "", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, env.Success)
}

func TestAuth_LoginSuccess(t *testing.T) {
	engine, _ := newServer(t)
	rec, env := doJSON(t, engine, http.MethodPost, "/api/auth/login", "",
		dto.LoginRequest{Username: "doctor", Password: seedPassword})
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, env.Success)

	var lr dto.LoginResponse
	decodeData(t, env, &lr)
	assert.NotEmpty(t, lr.Token)
	assert.Equal(t, "门诊", lr.User.DeptType)
	assert.Equal(t, "doctor", lr.User.Username)
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	engine, _ := newServer(t)
	rec, env := doJSON(t, engine, http.MethodPost, "/api/auth/login", "",
		dto.LoginRequest{Username: "doctor", Password: "nope"})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	require.NotNil(t, env.Error)
	assert.Equal(t, "INVALID_CREDENTIALS", env.Error.Code)
}

func TestAuth_MeRequiresToken(t *testing.T) {
	engine, _ := newServer(t)
	rec, _ := doJSON(t, engine, http.MethodGet, "/api/auth/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_MeWithToken(t *testing.T) {
	engine, _ := newServer(t)
	token := login(t, engine, "finance")

	rec, env := doJSON(t, engine, http.MethodGet, "/api/auth/me", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var u dto.UserInfo
	decodeData(t, env, &u)
	assert.Equal(t, "财务", u.DeptType)
	assert.Equal(t, "收费处", u.DeptName)
}

func TestAuth_RejectsTamperedToken(t *testing.T) {
	engine, _ := newServer(t)
	rec, _ := doJSON(t, engine, http.MethodGet, "/api/auth/me", "garbage.token.value", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

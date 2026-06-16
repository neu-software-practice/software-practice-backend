package jwt_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
)

func TestGenerateAndParse(t *testing.T) {
	m := jwt.NewManager("a-sufficiently-long-secret-0123456789", time.Hour)
	token, err := m.Generate(42, "王医生", "门诊")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := m.Parse(token)
	require.NoError(t, err)
	assert.EqualValues(t, 42, claims.EmployeeID)
	assert.Equal(t, "王医生", claims.Realname)
	assert.Equal(t, "门诊", claims.DeptType)
}

func TestParse_WrongSecret(t *testing.T) {
	token, err := jwt.NewManager("secret-one-0123456789", time.Hour).Generate(1, "n", "门诊")
	require.NoError(t, err)

	_, err = jwt.NewManager("secret-two-0123456789", time.Hour).Parse(token)
	assert.Error(t, err)
}

func TestParse_Expired(t *testing.T) {
	m := jwt.NewManager("secret-0123456789", -time.Minute) // already expired
	token, err := m.Generate(1, "n", "门诊")
	require.NoError(t, err)

	_, err = m.Parse(token)
	assert.Error(t, err)
}

func TestParse_Garbage(t *testing.T) {
	_, err := jwt.NewManager("secret-0123456789", time.Hour).Parse("not.a.jwt")
	assert.Error(t, err)
}

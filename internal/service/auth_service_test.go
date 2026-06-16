package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/hash"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// mockEmployeeRepo is a function-field mock of repository.EmployeeRepository.
type mockEmployeeRepo struct {
	findByUsername func(ctx context.Context, username string) (*model.Employee, error)
	findByID       func(ctx context.Context, id uint) (*model.Employee, error)
	listDoctors    func(ctx context.Context, deptID, levelID uint) ([]model.Employee, error)
}

func (m mockEmployeeRepo) FindByUsername(ctx context.Context, username string) (*model.Employee, error) {
	return m.findByUsername(ctx, username)
}
func (m mockEmployeeRepo) FindByID(ctx context.Context, id uint) (*model.Employee, error) {
	return m.findByID(ctx, id)
}
func (m mockEmployeeRepo) ListDoctors(ctx context.Context, deptID, levelID uint) ([]model.Employee, error) {
	return m.listDoctors(ctx, deptID, levelID)
}

func testEmployee(t *testing.T, password string) *model.Employee {
	t.Helper()
	hashed, err := hash.Password(password)
	require.NoError(t, err)
	return &model.Employee{
		ID:         7,
		Username:   "doctor",
		Password:   hashed,
		Realname:   "王医生",
		DeptmentID: 2,
		Department: &model.Department{ID: 2, DeptName: "门诊内科", DeptType: "门诊"},
	}
}

func newAuthService(repo repository.EmployeeRepository) *AuthService {
	return NewAuthService(repo, jwt.NewManager("test-secret-0123456789", time.Hour))
}

func TestAuthService_Login_Success(t *testing.T) {
	emp := testEmployee(t, "secret123")
	svc := newAuthService(mockEmployeeRepo{
		findByUsername: func(_ context.Context, u string) (*model.Employee, error) {
			assert.Equal(t, "doctor", u)
			return emp, nil
		},
	})

	resp, err := svc.Login(context.Background(), dto.LoginRequest{Username: "doctor", Password: "secret123"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "门诊", resp.User.DeptType)
	assert.Equal(t, "王医生", resp.User.Realname)

	// The issued token must parse back to the same identity.
	claims, err := jwt.NewManager("test-secret-0123456789", time.Hour).Parse(resp.Token)
	require.NoError(t, err)
	assert.Equal(t, uint(7), claims.EmployeeID)
	assert.Equal(t, "门诊", claims.DeptType)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	emp := testEmployee(t, "secret123")
	svc := newAuthService(mockEmployeeRepo{
		findByUsername: func(_ context.Context, _ string) (*model.Employee, error) { return emp, nil },
	})

	_, err := svc.Login(context.Background(), dto.LoginRequest{Username: "doctor", Password: "wrong"})
	assert.ErrorIs(t, err, apperr.ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	svc := newAuthService(mockEmployeeRepo{
		findByUsername: func(_ context.Context, _ string) (*model.Employee, error) {
			return nil, repository.ErrNotFound
		},
	})

	_, err := svc.Login(context.Background(), dto.LoginRequest{Username: "ghost", Password: "x"})
	assert.ErrorIs(t, err, apperr.ErrInvalidCredentials)
}

func TestAuthService_Me(t *testing.T) {
	emp := testEmployee(t, "secret123")
	svc := newAuthService(mockEmployeeRepo{
		findByID: func(_ context.Context, id uint) (*model.Employee, error) {
			assert.Equal(t, uint(7), id)
			return emp, nil
		},
	})

	u, err := svc.Me(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, "doctor", u.Username)
	assert.Equal(t, "门诊内科", u.DeptName)
}

func TestAuthService_Me_NotFound(t *testing.T) {
	svc := newAuthService(mockEmployeeRepo{
		findByID: func(_ context.Context, _ uint) (*model.Employee, error) {
			return nil, repository.ErrNotFound
		},
	})

	_, err := svc.Me(context.Background(), 99)
	assert.ErrorIs(t, err, apperr.ErrUnauthorized)
}

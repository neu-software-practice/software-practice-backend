package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/service/auth"
)

const testSecret = "this-is-a-test-secret-that-is-at-least-32-bytes-long!!"

// --- Mock repositories ---

type mockUserRepo struct {
	createFunc      func(ctx context.Context, user *model.User) error
	findByPhoneFunc func(ctx context.Context, phone string) (*model.User, error)
	findByIDFunc    func(ctx context.Context, id string) (*model.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	return m.createFunc(ctx, user)
}
func (m *mockUserRepo) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	return m.findByPhoneFunc(ctx, phone)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	return m.findByIDFunc(ctx, id)
}

type mockRefreshTokenRepo struct {
	createFunc          func(ctx context.Context, token *model.RefreshToken) error
	findByTokenHashFunc func(ctx context.Context, hash string) (*model.RefreshToken, error)
	markUsedFunc        func(ctx context.Context, id string) error
	revokeAllFunc       func(ctx context.Context, userID string) error
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	return m.createFunc(ctx, token)
}
func (m *mockRefreshTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	return m.findByTokenHashFunc(ctx, hash)
}
func (m *mockRefreshTokenRepo) MarkUsed(ctx context.Context, id string) error {
	return m.markUsedFunc(ctx, id)
}
func (m *mockRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID string) error {
	return m.revokeAllFunc(ctx, userID)
}

type mockPatientRepo struct {
	findByCredFunc func(ctx context.Context, credType, credential string) (*model.PatientProfile, error)
	findByIDFunc   func(ctx context.Context, id string) (*model.PatientProfile, error)
	createFunc     func(ctx context.Context, p *model.PatientProfile) error
	updateFunc     func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error)
}

func (m *mockPatientRepo) FindByCredential(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
	return m.findByCredFunc(ctx, credType, credential)
}
func (m *mockPatientRepo) FindByID(ctx context.Context, id string) (*model.PatientProfile, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockPatientRepo) Create(ctx context.Context, p *model.PatientProfile) error {
	return m.createFunc(ctx, p)
}
func (m *mockPatientRepo) UpdateProfile(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
	return m.updateFunc(ctx, id, input)
}

// --- Helper ---

func newTestService(
	userRepo *mockUserRepo,
	tokenRepo *mockRefreshTokenRepo,
	patientRepo *mockPatientRepo,
) *auth.Service {
	return auth.NewService(userRepo, tokenRepo, patientRepo, testSecret)
}

// --- Register tests ---

func TestRegister_Success_NewPatient(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
		createFunc: func(ctx context.Context, user *model.User) error {
			if user.Phone != "13800001111" {
				t.Errorf("Phone = %q, want 13800001111", user.Phone)
			}
			if user.RealName != "张三" {
				t.Errorf("RealName = %q, want 张三", user.RealName)
			}
			if user.PatientID == "" {
				t.Error("PatientID should not be empty")
			}
			return nil
		},
	}
	tokenRepo := &mockRefreshTokenRepo{
		createFunc: func(ctx context.Context, token *model.RefreshToken) error { return nil },
	}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error { return nil },
	}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	resp, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
		RealName: "张三",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if resp.ExpiresIn != auth.AccessTokenTTL {
		t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, auth.AccessTokenTTL)
	}
	if resp.User == nil {
		t.Fatal("User should not be nil")
	}
	if resp.User.Phone != "13800001111" {
		t.Errorf("User.Phone = %q, want 13800001111", resp.User.Phone)
	}
}

func TestRegister_Success_ExistingPatient(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
		createFunc: func(ctx context.Context, user *model.User) error {
			if user.PatientID != "existing-patient-id" {
				t.Errorf("PatientID = %q, want existing-patient-id", user.PatientID)
			}
			return nil
		},
	}
	tokenRepo := &mockRefreshTokenRepo{
		createFunc: func(ctx context.Context, token *model.RefreshToken) error { return nil },
	}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return &model.PatientProfile{ID: "existing-patient-id"}, nil
		},
	}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	resp, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800002222",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User.PatientID != "existing-patient-id" {
		t.Errorf("PatientID = %q, want existing-patient-id", resp.User.PatientID)
	}
}

func TestRegister_PhoneExists(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return &model.User{ID: "u1"}, nil
		},
	}
	tokenRepo := &mockRefreshTokenRepo{}
	patientRepo := &mockPatientRepo{}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
	})
	if !errors.Is(err, model.ErrPhoneExists) {
		t.Errorf("err = %v, want ErrPhoneExists", err)
	}
}

func TestRegister_InvalidPhone(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockPatientRepo{})

	tests := []struct {
		name  string
		phone string
	}{
		{"too short", "1380000"},
		{"too long", "138000011112"},
		{"invalid prefix", "10800001111"},
		{"non-numeric", "1380000111a"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, model.RegisterInput{
				Phone:    tt.phone,
				Password: "password123",
			})
			if !errors.Is(err, model.ErrValidation) {
				t.Errorf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockPatientRepo{})

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "short",
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

// --- Login tests ---

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return &model.User{
				ID:           "u1",
				Phone:        "13800001111",
				PasswordHash: "$2a$12$LJ3m4ys/Y4BnEm4y6GXbku8lSxGPYhDqNfOJpGqse5MYFnTnNADXe",
				PatientID:    "p1",
			}, nil
		},
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
	}
	tokenRepo := &mockRefreshTokenRepo{
		createFunc: func(ctx context.Context, token *model.RefreshToken) error { return nil },
	}
	patientRepo := &mockPatientRepo{}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	// Use Register to create a properly hashed user first
	userRepo.findByPhoneFunc = func(ctx context.Context, phone string) (*model.User, error) {
		return nil, model.ErrUserNotFound
	}
	userRepo.createFunc = func(ctx context.Context, user *model.User) error { return nil }
	patientRepo.findByCredFunc = func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
		return &model.PatientProfile{ID: "p1"}, nil
	}

	regResp, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "mypassword",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Now set up for login - capture the hash from Register
	var capturedUser *model.User
	userRepo.createFunc = func(ctx context.Context, user *model.User) error {
		capturedUser = user
		return nil
	}
	_, _ = svc.Register(ctx, model.RegisterInput{
		Phone:    "13800003333",
		Password: "mypassword",
	})

	if capturedUser == nil {
		t.Fatal("failed to capture user")
	}

	userRepo.findByPhoneFunc = func(ctx context.Context, phone string) (*model.User, error) {
		return capturedUser, nil
	}

	resp, err := svc.Login(ctx, model.LoginInput{
		Phone:    "13800003333",
		Password: "mypassword",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if resp.User == nil {
		t.Fatal("User should not be nil")
	}
	_ = regResp
}

func TestLogin_UserNotFound(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, &mockRefreshTokenRepo{}, &mockPatientRepo{})

	_, err := svc.Login(ctx, model.LoginInput{
		Phone:    "13800001111",
		Password: "password",
	})
	if !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
		createFunc: func(ctx context.Context, user *model.User) error { return nil },
	}
	tokenRepo := &mockRefreshTokenRepo{
		createFunc: func(ctx context.Context, token *model.RefreshToken) error { return nil },
	}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return &model.PatientProfile{ID: "p1"}, nil
		},
	}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	// Register first
	var capturedUser *model.User
	userRepo.createFunc = func(ctx context.Context, user *model.User) error {
		capturedUser = user
		return nil
	}
	_, _ = svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "correctpassword",
	})

	userRepo.findByPhoneFunc = func(ctx context.Context, phone string) (*model.User, error) {
		return capturedUser, nil
	}

	_, err := svc.Login(ctx, model.LoginInput{
		Phone:    "13800001111",
		Password: "wrongpassword",
	})
	if !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

// --- Refresh tests ---

func TestRefresh_Success(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.RefreshToken{
		ID:        "rt1",
		TokenHash: "somehash",
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	var markedID string
	userRepo := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return &model.User{
				ID:        "u1",
				Phone:     "13800001111",
				PatientID: "p1",
			}, nil
		},
	}
	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error {
			markedID = id
			return nil
		},
		createFunc: func(ctx context.Context, token *model.RefreshToken) error { return nil },
	}

	svc := newTestService(userRepo, tokenRepo, &mockPatientRepo{})

	resp, err := svc.Refresh(ctx, "raw-refresh-token")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("new RefreshToken should not be empty")
	}
	if resp.User != nil {
		t.Error("User should be nil for refresh response")
	}
	if markedID != "rt1" {
		t.Errorf("markedID = %q, want rt1", markedID)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	ctx := context.Background()

	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return nil, model.ErrRefreshTokenInvalid
		},
	}

	svc := newTestService(&mockUserRepo{}, tokenRepo, &mockPatientRepo{})

	_, err := svc.Refresh(ctx, "unknown-token")
	if !errors.Is(err, model.ErrRefreshTokenInvalid) {
		t.Errorf("err = %v, want ErrRefreshTokenInvalid", err)
	}
}

func TestRefresh_TokenTheftDetection(t *testing.T) {
	ctx := context.Background()

	usedAt := time.Now().Add(-time.Minute)
	storedToken := &model.RefreshToken{
		ID:        "rt1",
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    &usedAt,
	}

	var revokedUserID string
	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return storedToken, nil
		},
		revokeAllFunc: func(ctx context.Context, userID string) error {
			revokedUserID = userID
			return nil
		},
	}

	svc := newTestService(&mockUserRepo{}, tokenRepo, &mockPatientRepo{})

	_, err := svc.Refresh(ctx, "reused-token")
	if !errors.Is(err, model.ErrRefreshTokenReuse) {
		t.Errorf("err = %v, want ErrRefreshTokenReuse", err)
	}
	if revokedUserID != "u1" {
		t.Errorf("revokedUserID = %q, want u1", revokedUserID)
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.RefreshToken{
		ID:        "rt1",
		UserID:    "u1",
		ExpiresAt: time.Now().Add(-time.Hour),
		UsedAt:    nil,
	}

	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return storedToken, nil
		},
	}

	svc := newTestService(&mockUserRepo{}, tokenRepo, &mockPatientRepo{})

	_, err := svc.Refresh(ctx, "expired-token")
	if !errors.Is(err, model.ErrRefreshTokenExpired) {
		t.Errorf("err = %v, want ErrRefreshTokenExpired", err)
	}
}

// --- Logout tests ---

func TestLogout_Success(t *testing.T) {
	ctx := context.Background()

	var markedID string
	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return &model.RefreshToken{ID: "rt1", UserID: "u1"}, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error {
			markedID = id
			return nil
		},
	}

	svc := newTestService(&mockUserRepo{}, tokenRepo, &mockPatientRepo{})

	err := svc.Logout(ctx, "some-token")
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if markedID != "rt1" {
		t.Errorf("markedID = %q, want rt1", markedID)
	}
}

func TestLogout_UnknownToken_Idempotent(t *testing.T) {
	ctx := context.Background()

	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return nil, model.ErrRefreshTokenInvalid
		},
	}

	svc := newTestService(&mockUserRepo{}, tokenRepo, &mockPatientRepo{})

	err := svc.Logout(ctx, "unknown-token")
	if err != nil {
		t.Fatalf("Logout should be idempotent, got: %v", err)
	}
}

// --- Register edge cases ---

func TestRegister_CreateUserFails(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
		createFunc: func(ctx context.Context, user *model.User) error {
			return errors.New("db error")
		},
	}
	tokenRepo := &mockRefreshTokenRepo{}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return &model.PatientProfile{ID: "p1"}, nil
		},
	}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_FindPhoneError(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, errors.New("db connection lost")
		},
	}

	svc := newTestService(userRepo, &mockRefreshTokenRepo{}, &mockPatientRepo{})

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_PatientLookupError(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
	}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, errors.New("patient db error")
		},
	}

	svc := newTestService(userRepo, &mockRefreshTokenRepo{}, patientRepo)

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error from patient lookup")
	}
}

func TestRegister_CreatePatientFails(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
	}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			return errors.New("patient insert failed")
		},
	}

	svc := newTestService(userRepo, &mockRefreshTokenRepo{}, patientRepo)

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error from patient creation")
	}
}

func TestRegister_StoreRefreshTokenFails(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
		createFunc: func(ctx context.Context, user *model.User) error { return nil },
	}
	tokenRepo := &mockRefreshTokenRepo{
		createFunc: func(ctx context.Context, token *model.RefreshToken) error {
			return errors.New("token store failed")
		},
	}
	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
			return &model.PatientProfile{ID: "p1"}, nil
		},
	}

	svc := newTestService(userRepo, tokenRepo, patientRepo)

	_, err := svc.Register(ctx, model.RegisterInput{
		Phone:    "13800001111",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error from token storage")
	}
}

func TestLogin_FindError(t *testing.T) {
	ctx := context.Background()

	userRepo := &mockUserRepo{
		findByPhoneFunc: func(ctx context.Context, phone string) (*model.User, error) {
			return nil, errors.New("db down")
		},
	}

	svc := newTestService(userRepo, &mockRefreshTokenRepo{}, &mockPatientRepo{})

	_, err := svc.Login(ctx, model.LoginInput{
		Phone:    "13800001111",
		Password: "password",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, model.ErrInvalidCredentials) {
		t.Error("should not be ErrInvalidCredentials for db error")
	}
}

func TestRefresh_MarkUsedFails(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.RefreshToken{
		ID:        "rt1",
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error {
			return errors.New("mark used failed")
		},
	}

	svc := newTestService(&mockUserRepo{}, tokenRepo, &mockPatientRepo{})

	_, err := svc.Refresh(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error from MarkUsed")
	}
}

func TestRefresh_UserNotFound(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.RefreshToken{
		ID:        "rt1",
		UserID:    "u-gone",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	userRepo := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, model.ErrUserNotFound
		},
	}
	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error { return nil },
	}

	svc := newTestService(userRepo, tokenRepo, &mockPatientRepo{})

	_, err := svc.Refresh(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error when user not found during refresh")
	}
}

func TestRefresh_StoreNewTokenFails(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.RefreshToken{
		ID:        "rt1",
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	userRepo := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return &model.User{ID: "u1", Phone: "13800001111", PatientID: "p1"}, nil
		},
	}
	tokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error { return nil },
		createFunc: func(ctx context.Context, token *model.RefreshToken) error {
			return errors.New("store new token failed")
		},
	}

	svc := newTestService(userRepo, tokenRepo, &mockPatientRepo{})

	_, err := svc.Refresh(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error from storing new refresh token")
	}
}

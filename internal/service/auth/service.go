package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

const (
	AccessTokenTTL  = 900
	RefreshTokenTTL = 604800
	BcryptCost      = 12
	RefreshTokenLen = 32
)

var phoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Service handles authentication business logic.
type Service struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.RefreshTokenRepository
	patientRepo repository.PatientRepository
	jwtSecret   string
}

// NewService creates a new auth Service.
func NewService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	patientRepo repository.PatientRepository,
	jwtSecret string,
) *Service {
	return &Service{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		patientRepo: patientRepo,
		jwtSecret:   jwtSecret,
	}
}

// Register creates a new user account, reusing or creating a patient profile.
func (s *Service) Register(ctx context.Context, input model.RegisterInput) (*model.AuthResponse, error) {
	if !phoneRegexp.MatchString(input.Phone) {
		return nil, fmt.Errorf("%w: invalid phone format", model.ErrValidation)
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", model.ErrValidation)
	}

	_, err := s.userRepo.FindByPhone(ctx, input.Phone)
	if err == nil {
		return nil, model.ErrPhoneExists
	}
	if err != model.ErrUserNotFound {
		return nil, fmt.Errorf("check phone uniqueness: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	patientID, err := s.resolvePatientID(ctx, input.Phone, input.RealName)
	if err != nil {
		return nil, fmt.Errorf("resolve patient: %w", err)
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Phone:        input.Phone,
		PasswordHash: string(hash),
		RealName:     input.RealName,
		PatientID:    patientID,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.buildAuthResponse(ctx, user)
}

// Login authenticates a user by phone and password.
func (s *Service) Login(ctx context.Context, input model.LoginInput) (*model.AuthResponse, error) {
	user, err := s.userRepo.FindByPhone(ctx, input.Phone)
	if err == model.ErrUserNotFound {
		return nil, model.ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, model.ErrInvalidCredentials
	}

	return s.buildAuthResponse(ctx, user)
}

// Refresh rotates a refresh token, issuing a new token pair.
func (s *Service) Refresh(ctx context.Context, rawToken string) (*model.AuthResponse, error) {
	tokenHash := hashToken(rawToken)

	stored, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, model.ErrRefreshTokenInvalid
	}

	if stored.UsedAt != nil {
		_ = s.tokenRepo.RevokeAllByUserID(ctx, stored.UserID)
		return nil, model.ErrRefreshTokenReuse
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, model.ErrRefreshTokenExpired
	}

	if err := s.tokenRepo.MarkUsed(ctx, stored.ID); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user for refresh: %w", err)
	}

	resp, err := s.buildAuthResponse(ctx, user)
	if err != nil {
		return nil, err
	}
	resp.User = nil
	return resp, nil
}

// Logout invalidates a refresh token. Idempotent — always succeeds.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	tokenHash := hashToken(rawToken)

	stored, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil
	}

	_ = s.tokenRepo.MarkUsed(ctx, stored.ID)
	return nil
}

func (s *Service) resolvePatientID(ctx context.Context, phone, realName string) (string, error) {
	existing, err := s.patientRepo.FindByCredential(ctx, "phone", phone)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, model.ErrPatientNotFound) {
		return "", fmt.Errorf("find patient by phone: %w", err)
	}

	p := &model.PatientProfile{
		ID:                  uuid.New().String(),
		Name:                realName,
		Gender:              "unknown",
		Allergies:           []string{},
		ChronicDiseases:     []string{},
		LongTermMedications: []string{},
		MedicalHistory:      []string{},
		PhoneMasked:         phone,
	}
	if err := s.patientRepo.Create(ctx, p); err != nil {
		return "", fmt.Errorf("create patient: %w", err)
	}
	return p.ID, nil
}

func (s *Service) buildAuthResponse(ctx context.Context, user *model.User) (*model.AuthResponse, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateRefreshTokenRaw()
	if err != nil {
		return nil, err
	}

	rt := &model.RefreshToken{
		ID:        uuid.New().String(),
		TokenHash: hashToken(rawRefresh),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Duration(RefreshTokenTTL) * time.Second),
	}
	if err := s.tokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    AccessTokenTTL,
		User: &model.UserInfo{
			UserID:    user.ID,
			PatientID: user.PatientID,
			Phone:     user.Phone,
			RealName:  user.RealName,
		},
	}, nil
}

func (s *Service) generateAccessToken(user *model.User) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       user.ID,
		"patientId": user.PatientID,
		"phone":     user.Phone,
		"iat":       now.Unix(),
		"exp":       now.Add(time.Duration(AccessTokenTTL) * time.Second).Unix(),
	})
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func generateRefreshTokenRaw() (string, error) {
	b := make([]byte, RefreshTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

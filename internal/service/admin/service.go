package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/neuhis/software-practice-backend/internal/auth"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

const (
	adminAccessTokenTTL  = 900
	adminRefreshTokenTTL = 604800
	adminBcryptCost      = 12
	adminRefreshTokenLen = 32
)

// Service handles admin business logic.
type Service struct {
	adminRepo      repository.AdminRepository
	adminTokenRepo repository.AdminRefreshTokenRepository
	dashboardRepo  repository.DashboardRepository
	settingsRepo   repository.SettingsRepository
	patientRepo    repository.PatientRepository
	visitRepo      repository.VisitRepository
	jwtSecret      string
}

// NewService creates a new admin Service.
func NewService(
	adminRepo repository.AdminRepository,
	adminTokenRepo repository.AdminRefreshTokenRepository,
	dashboardRepo repository.DashboardRepository,
	settingsRepo repository.SettingsRepository,
	patientRepo repository.PatientRepository,
	visitRepo repository.VisitRepository,
	jwtSecret string,
) *Service {
	return &Service{
		adminRepo:      adminRepo,
		adminTokenRepo: adminTokenRepo,
		dashboardRepo:  dashboardRepo,
		settingsRepo:   settingsRepo,
		patientRepo:    patientRepo,
		visitRepo:      visitRepo,
		jwtSecret:      jwtSecret,
	}
}

// Login authenticates an admin by username and password.
func (s *Service) Login(ctx context.Context, input model.AdminLoginInput) (*model.AdminLoginResult, error) {
	if input.Username == "" || input.Password == "" {
		return nil, fmt.Errorf("%w: username and password are required", model.ErrValidation)
	}

	admin, err := s.adminRepo.FindByUsername(ctx, input.Username)
	if err == model.ErrAdminNotFound {
		return nil, model.ErrAdminInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find admin: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.Password)); err != nil {
		return nil, model.ErrAdminInvalidCredentials
	}

	accessToken, err := auth.GenerateAdminAccessToken(admin.ID, string(admin.Role), s.jwtSecret, adminAccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, err := generateAdminRefreshTokenRaw()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	rt := &model.AdminRefreshToken{
		ID:        uuid.New().String(),
		TokenHash: hashAdminToken(rawRefresh),
		AdminID:   admin.ID,
		ExpiresAt: time.Now().Add(time.Duration(adminRefreshTokenTTL) * time.Second),
	}
	if err := s.adminTokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	adminUser := *admin
	adminUser.PasswordHash = ""

	return &model.AdminLoginResult{
		Tokens: model.AdminTokens{
			AccessToken:  accessToken,
			RefreshToken: rawRefresh,
			ExpiresIn:    adminAccessTokenTTL,
		},
		User: adminUser,
	}, nil
}

// Logout invalidates an admin refresh token. Idempotent — always succeeds.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	tokenHash := hashAdminToken(rawToken)

	stored, err := s.adminTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		// Idempotent: return success even if token not found
		return nil
	}

	_ = s.adminTokenRepo.MarkUsed(ctx, stored.ID)
	return nil
}

// Refresh rotates an admin refresh token, issuing a new token pair.
func (s *Service) Refresh(ctx context.Context, rawToken string) (*model.AdminTokens, error) {
	tokenHash := hashAdminToken(rawToken)

	stored, err := s.adminTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, model.ErrAdminInvalidRefreshToken
	}

	if stored.UsedAt != nil {
		_ = s.adminTokenRepo.RevokeAllByAdminID(ctx, stored.AdminID)
		return nil, model.ErrAdminInvalidRefreshToken
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, model.ErrAdminInvalidRefreshToken
	}

	if err := s.adminTokenRepo.MarkUsed(ctx, stored.ID); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	admin, err := s.adminRepo.FindByID(ctx, stored.AdminID)
	if err != nil {
		return nil, fmt.Errorf("find admin for refresh: %w", err)
	}

	accessToken, err := auth.GenerateAdminAccessToken(admin.ID, string(admin.Role), s.jwtSecret, adminAccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, err := generateAdminRefreshTokenRaw()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	rt := &model.AdminRefreshToken{
		ID:        uuid.New().String(),
		TokenHash: hashAdminToken(rawRefresh),
		AdminID:   admin.ID,
		ExpiresAt: time.Now().Add(time.Duration(adminRefreshTokenTTL) * time.Second),
	}
	if err := s.adminTokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.AdminTokens{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    adminAccessTokenTTL,
	}, nil
}

// GetDashboardStats returns aggregated statistics for the admin dashboard.
func (s *Service) GetDashboardStats(ctx context.Context) (*model.DashboardStats, error) {
	todayStart := time.Now().Format("2006-01-02") + "T00:00:00Z"

	totalPatients, err := s.dashboardRepo.CountPatients(ctx)
	if err != nil {
		return nil, fmt.Errorf("count patients: %w", err)
	}

	totalSessions, err := s.dashboardRepo.CountSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("count sessions: %w", err)
	}

	activeSessions, err := s.dashboardRepo.CountActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active sessions: %w", err)
	}

	todayNewPatients, err := s.dashboardRepo.CountPatientsSince(ctx, todayStart)
	if err != nil {
		return nil, fmt.Errorf("count today patients: %w", err)
	}

	todayNewSessions, err := s.dashboardRepo.CountSessionsSince(ctx, todayStart)
	if err != nil {
		return nil, fmt.Errorf("count today sessions: %w", err)
	}

	return &model.DashboardStats{
		TotalPatients:    totalPatients,
		TotalSessions:    totalSessions,
		ActiveSessions:   activeSessions,
		TodayNewPatients: todayNewPatients,
		TodayNewSessions: todayNewSessions,
	}, nil
}

// ListPatients returns a paginated list of patients for the admin panel.
func (s *Service) ListPatients(ctx context.Context, query model.AdminPatientQuery) (*model.AdminPatientListResult, error) {
	items, total, err := s.dashboardRepo.ListPatients(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list patients: %w", err)
	}

	return &model.AdminPatientListResult{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetPatientProfile returns the full profile of a patient.
func (s *Service) GetPatientProfile(ctx context.Context, patientID string) (*model.PatientProfile, error) {
	patient, err := s.patientRepo.FindByID(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("get patient profile: %w", err)
	}
	return patient, nil
}

// ListSessions returns a paginated list of visit sessions for the admin panel.
func (s *Service) ListSessions(ctx context.Context, query model.AdminSessionQuery) (*model.AdminSessionListResult, error) {
	items, total, err := s.dashboardRepo.ListSessions(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	return &model.AdminSessionListResult{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetSessionDetail returns the full detail of a visit session.
func (s *Service) GetSessionDetail(ctx context.Context, sessionID string) (*model.VisitSession, error) {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session detail: %w", err)
	}
	return session, nil
}

// GetSettings returns the current system settings.
func (s *Service) GetSettings(ctx context.Context) (*model.SystemSettings, error) {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return settings, nil
}

// UpdateSettings updates system settings with partial input.
func (s *Service) UpdateSettings(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error) {
	// Validate
	if input.SiteName != nil && *input.SiteName == "" {
		return nil, fmt.Errorf("%w: siteName cannot be empty", model.ErrValidation)
	}
	if input.MaxConcurrentSessions != nil && *input.MaxConcurrentSessions <= 0 {
		return nil, fmt.Errorf("%w: maxConcurrentSessions must be positive", model.ErrValidation)
	}
	if input.SessionTimeoutMinutes != nil && *input.SessionTimeoutMinutes <= 0 {
		return nil, fmt.Errorf("%w: sessionTimeoutMinutes must be positive", model.ErrValidation)
	}

	settings, err := s.settingsRepo.Update(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("update settings: %w", err)
	}
	return settings, nil
}

// BuildAccessToken generates an admin access token for testing or direct use.
func (s *Service) BuildAccessToken(adminID, role string) (string, error) {
	return auth.GenerateAdminAccessToken(adminID, role, s.jwtSecret, adminAccessTokenTTL)
}

func generateAdminRefreshTokenRaw() (string, error) {
	b := make([]byte, adminRefreshTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate admin refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashAdminToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

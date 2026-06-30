package admin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/service/admin"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "this-is-a-test-secret-that-is-at-least-32-bytes-long!!"

// --- Mock repositories ---

type mockAdminRepo struct {
	findByUsernameFunc func(ctx context.Context, username string) (*model.AdminUser, error)
	findByIDFunc       func(ctx context.Context, id string) (*model.AdminUser, error)
	createFunc         func(ctx context.Context, admin *model.AdminUser) error
}

func (m *mockAdminRepo) FindByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	return m.findByUsernameFunc(ctx, username)
}
func (m *mockAdminRepo) FindByID(ctx context.Context, id string) (*model.AdminUser, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockAdminRepo) Create(ctx context.Context, admin *model.AdminUser) error {
	return m.createFunc(ctx, admin)
}

type mockAdminTokenRepo struct {
	createFunc          func(ctx context.Context, token *model.AdminRefreshToken) error
	findByTokenHashFunc func(ctx context.Context, hash string) (*model.AdminRefreshToken, error)
	markUsedFunc        func(ctx context.Context, id string) error
	revokeAllFunc       func(ctx context.Context, adminID string) error
}

func (m *mockAdminTokenRepo) Create(ctx context.Context, token *model.AdminRefreshToken) error {
	return m.createFunc(ctx, token)
}
func (m *mockAdminTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
	return m.findByTokenHashFunc(ctx, hash)
}
func (m *mockAdminTokenRepo) MarkUsed(ctx context.Context, id string) error {
	return m.markUsedFunc(ctx, id)
}
func (m *mockAdminTokenRepo) RevokeAllByAdminID(ctx context.Context, adminID string) error {
	return m.revokeAllFunc(ctx, adminID)
}

type mockDashboardRepo struct {
	countPatientsFunc       func(ctx context.Context) (int, error)
	countPatientsSinceFunc  func(ctx context.Context, since string) (int, error)
	countSessionsFunc       func(ctx context.Context) (int, error)
	countActiveSessionsFunc func(ctx context.Context) (int, error)
	countSessionsSinceFunc  func(ctx context.Context, since string) (int, error)
	listPatientsFunc        func(ctx context.Context, query model.AdminPatientQuery) ([]model.AdminPatientItem, int, error)
	listSessionsFunc        func(ctx context.Context, query model.AdminSessionQuery) ([]model.AdminSessionItem, int, error)
}

func (m *mockDashboardRepo) CountPatients(ctx context.Context) (int, error) {
	return m.countPatientsFunc(ctx)
}
func (m *mockDashboardRepo) CountPatientsSince(ctx context.Context, since string) (int, error) {
	return m.countPatientsSinceFunc(ctx, since)
}
func (m *mockDashboardRepo) CountSessions(ctx context.Context) (int, error) {
	return m.countSessionsFunc(ctx)
}
func (m *mockDashboardRepo) CountActiveSessions(ctx context.Context) (int, error) {
	return m.countActiveSessionsFunc(ctx)
}
func (m *mockDashboardRepo) CountSessionsSince(ctx context.Context, since string) (int, error) {
	return m.countSessionsSinceFunc(ctx, since)
}
func (m *mockDashboardRepo) ListPatients(ctx context.Context, query model.AdminPatientQuery) ([]model.AdminPatientItem, int, error) {
	return m.listPatientsFunc(ctx, query)
}
func (m *mockDashboardRepo) ListSessions(ctx context.Context, query model.AdminSessionQuery) ([]model.AdminSessionItem, int, error) {
	return m.listSessionsFunc(ctx, query)
}

type mockSettingsRepo struct {
	getFunc    func(ctx context.Context) (*model.SystemSettings, error)
	updateFunc func(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error)
}

func (m *mockSettingsRepo) Get(ctx context.Context) (*model.SystemSettings, error) {
	return m.getFunc(ctx)
}
func (m *mockSettingsRepo) Update(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error) {
	return m.updateFunc(ctx, input)
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

type mockVisitRepo struct {
	createFunc        func(ctx context.Context, visit *model.VisitSession) error
	findByIDFunc      func(ctx context.Context, id string) (*model.VisitSession, error)
	listByPatientFunc func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
	updateStatusFunc  func(ctx context.Context, id string, status string, machineState string) error
	updateFunc        func(ctx context.Context, visit *model.VisitSession) error
}

func (m *mockVisitRepo) Create(ctx context.Context, visit *model.VisitSession) error {
	return m.createFunc(ctx, visit)
}
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	return m.listByPatientFunc(ctx, patientID, cursor, pageSize)
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id string, status string, machineState string) error {
	return m.updateStatusFunc(ctx, id, status, machineState)
}
func (m *mockVisitRepo) Update(ctx context.Context, visit *model.VisitSession) error {
	return m.updateFunc(ctx, visit)
}

// --- Helper ---

func newTestService(
	adminRepo *mockAdminRepo,
	adminTokenRepo *mockAdminTokenRepo,
	dashboardRepo *mockDashboardRepo,
	settingsRepo *mockSettingsRepo,
	patientRepo *mockPatientRepo,
	visitRepo *mockVisitRepo,
) *admin.Service {
	return admin.NewService(adminRepo, adminTokenRepo, dashboardRepo, settingsRepo, patientRepo, visitRepo, testSecret)
}

func defaultMocks() (*mockAdminRepo, *mockAdminTokenRepo, *mockDashboardRepo, *mockSettingsRepo, *mockPatientRepo, *mockVisitRepo) {
	return &mockAdminRepo{}, &mockAdminTokenRepo{}, &mockDashboardRepo{}, &mockSettingsRepo{}, &mockPatientRepo{}, &mockVisitRepo{}
}

// --- Login tests ---

func TestAdminLogin_Success(t *testing.T) {
	ctx := context.Background()

	const testPassword = "admin123"
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), 12)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	adminRepo := &mockAdminRepo{
		findByUsernameFunc: func(ctx context.Context, username string) (*model.AdminUser, error) {
			return &model.AdminUser{
				ID:           "a1",
				Username:     "admin",
				PasswordHash: string(hash),
				Role:         model.AdminRoleSuperAdmin,
				DisplayName:  "系统管理员",
			}, nil
		},
	}
	adminTokenRepo := &mockAdminTokenRepo{
		createFunc: func(ctx context.Context, token *model.AdminRefreshToken) error { return nil },
	}
	_, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	resp, err := svc.Login(ctx, model.AdminLoginInput{
		Username: "admin",
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.Tokens.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if resp.Tokens.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if resp.Tokens.ExpiresIn <= 0 {
		t.Errorf("ExpiresIn = %d, want >0", resp.Tokens.ExpiresIn)
	}
	if resp.User.ID != "a1" {
		t.Errorf("User.ID = %q, want a1", resp.User.ID)
	}
	if resp.User.Username != "admin" {
		t.Errorf("User.Username = %q, want admin", resp.User.Username)
	}
	if resp.User.Role != model.AdminRoleSuperAdmin {
		t.Errorf("User.Role = %q, want super_admin", resp.User.Role)
	}
	if resp.User.DisplayName != "系统管理员" {
		t.Errorf("User.DisplayName = %q, want 系统管理员", resp.User.DisplayName)
	}
	if resp.User.PasswordHash != "" {
		t.Error("PasswordHash should be cleared in response")
	}
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), 12)

	adminRepo := &mockAdminRepo{
		findByUsernameFunc: func(ctx context.Context, username string) (*model.AdminUser, error) {
			return &model.AdminUser{
				ID:           "a1",
				Username:     "admin",
				PasswordHash: string(hash),
				Role:         model.AdminRoleAdmin,
			}, nil
		},
	}
	_, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Login(ctx, model.AdminLoginInput{
		Username: "admin",
		Password: "wrongpassword",
	})
	if !errors.Is(err, model.ErrAdminInvalidCredentials) {
		t.Errorf("err = %v, want ErrAdminInvalidCredentials", err)
	}
}

func TestAdminLogin_UserNotFound(t *testing.T) {
	ctx := context.Background()

	adminRepo := &mockAdminRepo{
		findByUsernameFunc: func(ctx context.Context, username string) (*model.AdminUser, error) {
			return nil, model.ErrAdminNotFound
		},
	}
	_, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Login(ctx, model.AdminLoginInput{
		Username: "nonexistent",
		Password: "password",
	})
	if !errors.Is(err, model.ErrAdminInvalidCredentials) {
		t.Errorf("err = %v, want ErrAdminInvalidCredentials", err)
	}
}

func TestAdminLogin_EmptyCredentials(t *testing.T) {
	ctx := context.Background()

	aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()
	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"empty username", "", "password"},
		{"empty password", "admin", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(ctx, model.AdminLoginInput{
				Username: tt.username,
				Password: tt.password,
			})
			if !errors.Is(err, model.ErrValidation) {
				t.Errorf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestAdminLogin_DBError(t *testing.T) {
	ctx := context.Background()

	adminRepo := &mockAdminRepo{
		findByUsernameFunc: func(ctx context.Context, username string) (*model.AdminUser, error) {
			return nil, errors.New("db down")
		},
	}
	_, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Login(ctx, model.AdminLoginInput{
		Username: "admin",
		Password: "password",
	})
	if err == nil {
		t.Fatal("expected error from db")
	}
	if errors.Is(err, model.ErrAdminInvalidCredentials) {
		t.Error("db error should not be treated as invalid credentials")
	}
}

// --- Logout tests ---

func TestAdminLogout_Success(t *testing.T) {
	ctx := context.Background()

	var markedID string
	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return &model.AdminRefreshToken{ID: "rt1", AdminID: "a1"}, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error {
			markedID = id
			return nil
		},
	}
	aRepo, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	err := svc.Logout(ctx, "some-token")
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if markedID != "rt1" {
		t.Errorf("markedID = %q, want rt1", markedID)
	}
}

func TestAdminLogout_UnknownToken_Idempotent(t *testing.T) {
	ctx := context.Background()

	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return nil, model.ErrAdminInvalidRefreshToken
		},
	}
	aRepo, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	err := svc.Logout(ctx, "unknown-token")
	if err != nil {
		t.Fatalf("Logout should be idempotent, got: %v", err)
	}
}

// --- Refresh tests ---

func TestAdminRefresh_Success(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.AdminRefreshToken{
		ID:        "rt1",
		TokenHash: "somehash",
		AdminID:   "a1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	var markedID string
	adminRepo := &mockAdminRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.AdminUser, error) {
			return &model.AdminUser{
				ID:   "a1",
				Role: model.AdminRoleAdmin,
			}, nil
		},
	}
	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error {
			markedID = id
			return nil
		},
		createFunc: func(ctx context.Context, token *model.AdminRefreshToken) error { return nil },
	}
	_, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

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
	if markedID != "rt1" {
		t.Errorf("markedID = %q, want rt1", markedID)
	}
}

func TestAdminRefresh_InvalidToken(t *testing.T) {
	ctx := context.Background()

	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return nil, model.ErrAdminInvalidRefreshToken
		},
	}
	aRepo, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Refresh(ctx, "unknown-token")
	if !errors.Is(err, model.ErrAdminInvalidRefreshToken) {
		t.Errorf("err = %v, want ErrAdminInvalidRefreshToken", err)
	}
}

func TestAdminRefresh_TokenTheftDetection(t *testing.T) {
	ctx := context.Background()

	usedAt := time.Now().Add(-time.Minute)
	storedToken := &model.AdminRefreshToken{
		ID:        "rt1",
		AdminID:   "a1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    &usedAt,
	}

	var revokedAdminID string
	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return storedToken, nil
		},
		revokeAllFunc: func(ctx context.Context, adminID string) error {
			revokedAdminID = adminID
			return nil
		},
	}
	aRepo, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Refresh(ctx, "reused-token")
	if !errors.Is(err, model.ErrAdminInvalidRefreshToken) {
		t.Errorf("err = %v, want ErrAdminInvalidRefreshToken", err)
	}
	if revokedAdminID != "a1" {
		t.Errorf("revokedAdminID = %q, want a1", revokedAdminID)
	}
}

func TestAdminRefresh_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.AdminRefreshToken{
		ID:        "rt1",
		AdminID:   "a1",
		ExpiresAt: time.Now().Add(-time.Hour),
		UsedAt:    nil,
	}

	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return storedToken, nil
		},
	}
	aRepo, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Refresh(ctx, "expired-token")
	if !errors.Is(err, model.ErrAdminInvalidRefreshToken) {
		t.Errorf("err = %v, want ErrAdminInvalidRefreshToken", err)
	}
}

// --- Dashboard stats tests ---

func TestGetDashboardStats_Success(t *testing.T) {
	ctx := context.Background()

	dashRepo := &mockDashboardRepo{
		countPatientsFunc:       func(ctx context.Context) (int, error) { return 100, nil },
		countSessionsFunc:       func(ctx context.Context) (int, error) { return 500, nil },
		countActiveSessionsFunc: func(ctx context.Context) (int, error) { return 12, nil },
		countPatientsSinceFunc:  func(ctx context.Context, since string) (int, error) { return 5, nil },
		countSessionsSinceFunc:  func(ctx context.Context, since string) (int, error) { return 8, nil },
	}
	aRepo, tRepo, _, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	stats, err := svc.GetDashboardStats(ctx)
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}
	if stats.TotalPatients != 100 {
		t.Errorf("TotalPatients = %d, want 100", stats.TotalPatients)
	}
	if stats.TotalSessions != 500 {
		t.Errorf("TotalSessions = %d, want 500", stats.TotalSessions)
	}
	if stats.ActiveSessions != 12 {
		t.Errorf("ActiveSessions = %d, want 12", stats.ActiveSessions)
	}
	if stats.TodayNewPatients != 5 {
		t.Errorf("TodayNewPatients = %d, want 5", stats.TodayNewPatients)
	}
	if stats.TodayNewSessions != 8 {
		t.Errorf("TodayNewSessions = %d, want 8", stats.TodayNewSessions)
	}
}

// --- Patient listing tests ---

func TestListPatients_Success(t *testing.T) {
	ctx := context.Background()

	dashRepo := &mockDashboardRepo{
		listPatientsFunc: func(ctx context.Context, query model.AdminPatientQuery) ([]model.AdminPatientItem, int, error) {
			return []model.AdminPatientItem{
				{ID: "p1", RealName: "张三", Phone: "13800001111", Gender: "male", SessionCount: 3},
				{ID: "p2", RealName: "李四", Phone: "13800002222", Gender: "female", SessionCount: 1},
			}, 25, nil
		},
	}
	aRepo, tRepo, _, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	result, err := svc.ListPatients(ctx, model.AdminPatientQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListPatients: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(result.Items))
	}
	if result.Total != 25 {
		t.Errorf("Total = %d, want 25", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("Page = %d, want 1", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", result.PageSize)
	}
}

// --- Session listing tests ---

func TestListSessions_Success(t *testing.T) {
	ctx := context.Background()

	dashRepo := &mockDashboardRepo{
		listSessionsFunc: func(ctx context.Context, query model.AdminSessionQuery) ([]model.AdminSessionItem, int, error) {
			return []model.AdminSessionItem{
				{ID: "s1", PatientID: "p1", PatientName: "张三", Status: "in_progress"},
			}, 1, nil
		},
	}
	aRepo, tRepo, _, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	result, err := svc.ListSessions(ctx, model.AdminSessionQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(result.Items))
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

// --- Patient profile tests ---

func TestGetPatientProfile_Success(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{ID: "p1", Name: "张三", Gender: "male"}, nil
		},
	}
	aRepo, tRepo, dashRepo, settingsRepo, _, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patientRepo, visitRepo)

	patient, err := svc.GetPatientProfile(ctx, "p1")
	if err != nil {
		t.Fatalf("GetPatientProfile: %v", err)
	}
	if patient.ID != "p1" {
		t.Errorf("ID = %q, want p1", patient.ID)
	}
	if patient.Name != "张三" {
		t.Errorf("Name = %q, want 张三", patient.Name)
	}
}

func TestGetPatientProfile_NotFound(t *testing.T) {
	ctx := context.Background()

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
	}
	aRepo, tRepo, dashRepo, settingsRepo, _, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patientRepo, visitRepo)

	_, err := svc.GetPatientProfile(ctx, "nonexistent")
	if !errors.Is(err, model.ErrPatientNotFound) {
		t.Errorf("err = %v, want ErrPatientNotFound", err)
	}
}

// --- Session detail tests ---

func TestGetSessionDetail_Success(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{ID: "s1", PatientID: "p1", Status: "in_progress"}, nil
		},
	}
	aRepo, tRepo, dashRepo, settingsRepo, patRepo, _ := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	session, err := svc.GetSessionDetail(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSessionDetail: %v", err)
	}
	if session.ID != "s1" {
		t.Errorf("ID = %q, want s1", session.ID)
	}
}

func TestGetSessionDetail_NotFound(t *testing.T) {
	ctx := context.Background()

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return nil, model.ErrSessionNotFound
		},
	}
	aRepo, tRepo, dashRepo, settingsRepo, patRepo, _ := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.GetSessionDetail(ctx, "nonexistent")
	if !errors.Is(err, model.ErrSessionNotFound) {
		t.Errorf("err = %v, want ErrSessionNotFound", err)
	}
}

// --- Settings tests ---

func TestGetSettings_Success(t *testing.T) {
	ctx := context.Background()

	settingsRepo := &mockSettingsRepo{
		getFunc: func(ctx context.Context) (*model.SystemSettings, error) {
			return &model.SystemSettings{
				SiteName:              "NEUHIS",
				MaxConcurrentSessions: 3,
				SessionTimeoutMinutes: 30,
				EnableRegistration:    true,
			}, nil
		},
	}
	aRepo, tRepo, dashRepo, _, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.SiteName != "NEUHIS" {
		t.Errorf("SiteName = %q, want NEUHIS", settings.SiteName)
	}
}

func TestUpdateSettings_Success(t *testing.T) {
	ctx := context.Background()

	newName := "新名称"
	settingsRepo := &mockSettingsRepo{
		updateFunc: func(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error) {
			return &model.SystemSettings{
				SiteName:              *input.SiteName,
				MaxConcurrentSessions: 5,
				SessionTimeoutMinutes: 60,
				EnableRegistration:    false,
			}, nil
		},
	}
	aRepo, tRepo, dashRepo, _, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	settings, err := svc.UpdateSettings(ctx, model.UpdateSystemSettingsInput{
		SiteName: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if settings.SiteName != "新名称" {
		t.Errorf("SiteName = %q, want 新名称", settings.SiteName)
	}
}

func TestUpdateSettings_EmptySiteName(t *testing.T) {
	ctx := context.Background()

	emptyName := ""
	aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.UpdateSettings(ctx, model.UpdateSystemSettingsInput{
		SiteName: &emptyName,
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestUpdateSettings_NegativeMaxConcurrent(t *testing.T) {
	ctx := context.Background()

	neg := -1
	aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.UpdateSettings(ctx, model.UpdateSystemSettingsInput{
		MaxConcurrentSessions: &neg,
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestUpdateSettings_ZeroTimeout(t *testing.T) {
	ctx := context.Background()

	zero := 0
	aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.UpdateSettings(ctx, model.UpdateSystemSettingsInput{
		SessionTimeoutMinutes: &zero,
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

// --- Error path tests for coverage ---

func TestGetDashboardStats_Error(t *testing.T) {
	ctx := context.Background()

	dashRepo := &mockDashboardRepo{
		countPatientsFunc: func(ctx context.Context) (int, error) {
			return 0, errors.New("db error")
		},
	}
	aRepo, tRepo, _, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.GetDashboardStats(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListPatients_Error(t *testing.T) {
	ctx := context.Background()

	dashRepo := &mockDashboardRepo{
		listPatientsFunc: func(ctx context.Context, query model.AdminPatientQuery) ([]model.AdminPatientItem, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	aRepo, tRepo, _, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.ListPatients(ctx, model.AdminPatientQuery{Page: 1, PageSize: 10})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListSessions_Error(t *testing.T) {
	ctx := context.Background()

	dashRepo := &mockDashboardRepo{
		listSessionsFunc: func(ctx context.Context, query model.AdminSessionQuery) ([]model.AdminSessionItem, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	aRepo, tRepo, _, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.ListSessions(ctx, model.AdminSessionQuery{Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetSettings_Error(t *testing.T) {
	ctx := context.Background()

	settingsRepo := &mockSettingsRepo{
		getFunc: func(ctx context.Context) (*model.SystemSettings, error) {
			return nil, errors.New("db error")
		},
	}
	aRepo, tRepo, dashRepo, _, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.GetSettings(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateSettings_RepoError(t *testing.T) {
	ctx := context.Background()

	newName := "test"
	settingsRepo := &mockSettingsRepo{
		updateFunc: func(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error) {
			return nil, errors.New("db error")
		},
	}
	aRepo, tRepo, dashRepo, _, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.UpdateSettings(ctx, model.UpdateSystemSettingsInput{
		SiteName: &newName,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAdminRefresh_MarkUsedFails(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.AdminRefreshToken{
		ID:        "rt1",
		AdminID:   "a1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error {
			return errors.New("mark used failed")
		},
	}
	aRepo, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(aRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Refresh(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error from MarkUsed")
	}
}

func TestAdminRefresh_AdminNotFound(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.AdminRefreshToken{
		ID:        "rt1",
		AdminID:   "a-gone",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	adminRepo := &mockAdminRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.AdminUser, error) {
			return nil, model.ErrAdminNotFound
		},
	}
	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error { return nil },
	}
	_, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Refresh(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error when admin not found during refresh")
	}
}

func TestAdminRefresh_StoreNewTokenFails(t *testing.T) {
	ctx := context.Background()

	storedToken := &model.AdminRefreshToken{
		ID:        "rt1",
		AdminID:   "a1",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	adminRepo := &mockAdminRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.AdminUser, error) {
			return &model.AdminUser{ID: "a1", Role: model.AdminRoleAdmin}, nil
		},
	}
	adminTokenRepo := &mockAdminTokenRepo{
		findByTokenHashFunc: func(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
			return storedToken, nil
		},
		markUsedFunc: func(ctx context.Context, id string) error { return nil },
		createFunc: func(ctx context.Context, token *model.AdminRefreshToken) error {
			return errors.New("store new token failed")
		},
	}
	_, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Refresh(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error from storing new refresh token")
	}
}

func TestAdminLogin_StoreTokenFails(t *testing.T) {
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 12)

	adminRepo := &mockAdminRepo{
		findByUsernameFunc: func(ctx context.Context, username string) (*model.AdminUser, error) {
			return &model.AdminUser{
				ID:           "a1",
				Username:     "admin",
				PasswordHash: string(hash),
				Role:         model.AdminRoleSuperAdmin,
			}, nil
		},
	}
	adminTokenRepo := &mockAdminTokenRepo{
		createFunc: func(ctx context.Context, token *model.AdminRefreshToken) error {
			return errors.New("store failed")
		},
	}
	_, _, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()

	svc := newTestService(adminRepo, adminTokenRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	_, err := svc.Login(ctx, model.AdminLoginInput{
		Username: "admin",
		Password: "admin123",
	})
	if err == nil {
		t.Fatal("expected error from token storage")
	}
}

func TestBuildAccessToken(t *testing.T) {
	adminRepo := &mockAdminRepo{
		findByUsernameFunc: func(ctx context.Context, username string) (*model.AdminUser, error) {
			return &model.AdminUser{
				ID:   "a1",
				Role: model.AdminRoleAdmin,
			}, nil
		},
	}
	aRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo := defaultMocks()
	_ = aRepo

	svc := newTestService(adminRepo, tRepo, dashRepo, settingsRepo, patRepo, visitRepo)

	token, err := svc.BuildAccessToken("a1", "admin")
	if err != nil {
		t.Fatalf("BuildAccessToken: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
}

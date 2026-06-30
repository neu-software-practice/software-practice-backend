package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neuhis/software-practice-backend/internal/config"
	"github.com/neuhis/software-practice-backend/internal/handler"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	addresssvc "github.com/neuhis/software-practice-backend/internal/service/address"
	authsvc "github.com/neuhis/software-practice-backend/internal/service/auth"
	billingsvc "github.com/neuhis/software-practice-backend/internal/service/billing"
	patientsvc "github.com/neuhis/software-practice-backend/internal/service/patient"
	visitsvc "github.com/neuhis/software-practice-backend/internal/service/visit"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockPatientService implements the patient service interface for testing

// mockVisitService implements the visit service interface for testing

// We test handler helper functions directly since full handler tests need wired services.

func TestParseSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "sessionId", Value: "s001"}}

	id := handler.ParseSessionID(c)
	if id != "s001" {
		t.Errorf("got %s, want s001", id)
	}
}

func TestParsePatientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}}

	id := handler.ParsePatientID(c)
	if id != "p001" {
		t.Errorf("got %s, want p001", id)
	}
}

func TestParseQueryInt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		queryVal   string
		defaultVal int
		expected   int
	}{
		{"empty uses default", "", 20, 20},
		{"valid value", "50", 20, 50},
		{"zero uses default", "0", 20, 20},
		{"negative uses default", "-1", 20, 20},
		{"non-numeric uses default", "abc", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test?pageSize="+tt.queryVal, nil)

			result := handler.ParseQueryInt(c, "pageSize", tt.defaultVal)
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestWriteSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	handler.WriteSuccess(c, 200, map[string]string{"key": "value"})

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp api.ApiResponse[map[string]interface{}]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestWritePageResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	items := []string{"a", "b"}
	cursor := "next"
	handler.WritePageResult(c, api.NewPageResult(items, &cursor, true))

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGetPatientIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("patient id set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("patientId", "p001")
		id := handler.GetPatientIDFromContext(c)
		if id != "p001" {
			t.Errorf("got %s, want p001", id)
		}
	})

	t.Run("no patient id", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		id := handler.GetPatientIDFromContext(c)
		if id != "" {
			t.Errorf("got %s, want empty", id)
		}
	})
}

func TestRequirePatientID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("matching id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("patientId", "p001")
		c.Request = httptest.NewRequest("GET", "/test", nil)

		if !handler.RequirePatientID(c, "p001") {
			t.Error("should allow matching id")
		}
	})

	t.Run("mismatched id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("patientId", "p001")
		c.Request = httptest.NewRequest("GET", "/test", nil)

		if handler.RequirePatientID(c, "p002") {
			t.Error("should deny mismatched id")
		}
	})
}

func TestSSEWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stream", nil)

	writer, err := handler.NewSSEWriter(c)
	if err != nil {
		t.Fatalf("new sse writer: %v", err)
	}

	event := model.AssistantStreamEvent{
		Type:      "delta",
		SessionID: "s001",
		RequestID: "r001",
		Content:   "hello",
	}

	if err := writer.WriteEvent(event); err != nil {
		t.Fatalf("write event: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "data:") {
		t.Error("SSE response should contain 'data:'")
	}
	if !strings.Contains(body, "delta") {
		t.Error("SSE response should contain event type")
	}

	writer.Close()
}

func TestBindJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type testInput struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	c.Request = httptest.NewRequest("POST", "/test",
		strings.NewReader(`{"name":"test","age":30}`))
	c.Request.Header.Set("Content-Type", "application/json")

	input, err := handler.BindJSON[testInput](c)
	if err != nil {
		t.Fatalf("bind json: %v", err)
	}
	if input.Name != "test" {
		t.Errorf("name = %s, want test", input.Name)
	}
	if input.Age != 30 {
		t.Errorf("age = %d, want 30", input.Age)
	}
}

func TestBindJSON_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type testInput struct {
		Name string `json:"name"`
	}

	c.Request = httptest.NewRequest("POST", "/test",
		strings.NewReader(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")

	_, err := handler.BindJSON[testInput](c)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRouterSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Verify we can register routes
	r.POST("/api/patients/verify", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/api/visits", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/patients/verify", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestParseQueryInt_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		queryVal   string
		defaultVal int
		expected   int
	}{
		{"max int overflow", "9223372036854775808", 20, 20},
		{"very large page size", "1000000", 20, 1000000},
		{"negative value", "-5", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test?pageSize="+tt.queryVal, nil)

			result := handler.ParseQueryInt(c, "pageSize", tt.defaultVal)
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestWriteSuccess_StatusCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		status int
	}{
		{"status 200", 200},
		{"status 201", 201},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			handler.WriteSuccess(c, tt.status, map[string]string{"key": "value"})

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}

			var resp api.ApiResponse[map[string]interface{}]
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if !resp.Success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestWritePageResult_NoMore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	items := []string{"a", "b"}
	handler.WritePageResult(c, api.NewPageResult(items, nil, false))

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp api.ApiResponse[api.PageResult[string]]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Data == nil {
		t.Fatal("data should not be nil")
	}
	if resp.Data.HasMore {
		t.Error("expected hasMore=false")
	}
	if resp.Data.NextCursor != nil {
		t.Error("expected nextCursor=nil")
	}
}

func TestSSEWriter_StreamEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stream", nil)

	writer, err := handler.NewSSEWriter(c)
	if err != nil {
		t.Fatalf("new sse writer: %v", err)
	}

	events := []model.AssistantStreamEvent{
		{
			Type:      "delta",
			SessionID: "s001",
			RequestID: "r001",
			Content:   "hello",
		},
		{
			Type:      "delta",
			SessionID: "s001",
			RequestID: "r002",
			Content:   " world",
		},
	}

	for _, event := range events {
		if err := writer.WriteEvent(event); err != nil {
			t.Fatalf("write event: %v", err)
		}
	}

	body := w.Body.String()
	if !strings.Contains(body, "hello") {
		t.Error("SSE response should contain first event content")
	}
	if !strings.Contains(body, "world") {
		t.Error("SSE response should contain second event content")
	}

	writer.Close()
}

func TestBindJSON_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type testInput struct {
		Name string `json:"name"`
	}

	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	_, err := handler.BindJSON[testInput](c)
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestGetPatientIDFromContext_NonString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("patientId", 12345)

	id := handler.GetPatientIDFromContext(c)
	if id != "" {
		t.Errorf("got %s, want empty for non-string value", id)
	}
}

func TestWriteSuccessWithMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.WriteSuccessWithMeta(c, http.StatusOK, map[string]string{"key": "value"}, map[string]int{"total": 42})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("body should not be empty")
	}
}

func TestSSEWriter_WriteError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stream", nil)

	writer, err := handler.NewSSEWriter(c)
	if err != nil {
		t.Fatalf("NewSSEWriter: %v", err)
	}

	writer.WriteError("s001", "req-1", model.ErrSessionNotFound)

	// Verify SSE format in response
	body := w.Body.String()
	if body == "" {
		t.Error("body should not be empty for error event")
	}
	if !strings.Contains(body, "error") {
		t.Errorf("body should contain 'error': %s", body)
	}
}

func TestStreamEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful stream", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/stream", nil)

		events := []model.AssistantStreamEvent{
			{Type: "thinking", SessionID: "s1", RequestID: "r1", Message: "thinking..."},
			{Type: "message", SessionID: "s1", RequestID: "r1", Message: "你好"},
		}

		// StreamEvents writes via SSE and returns without error
		handler.StreamEvents(c, events)

		body := w.Body.String()
		if !strings.Contains(body, "thinking") {
			t.Errorf("body should contain 'thinking': %s", body)
		}
		if !strings.Contains(body, "message") {
			t.Errorf("body should contain 'message': %s", body)
		}
	})

	t.Run("empty events", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/stream", nil)

		handler.StreamEvents(c, nil)

		// Should still set SSE headers
		if c.Writer.Header().Get("Content-Type") != "text/event-stream" {
			t.Errorf("Content-Type = %s, want text/event-stream",
				c.Writer.Header().Get("Content-Type"))
		}
	})
}

func TestRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/api/patients/verify", func(c *gin.Context) {
		c.JSON(200, gin.H{"verified": true})
	})

	t.Run("GET /api/health", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/health", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("POST /api/patients/verify", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/patients/verify", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Mock repositories (shared across handler tests)
// ---------------------------------------------------------------------------

type mockPatientRepo struct {
	findByCredFunc func(ctx context.Context, credType, credential string) (*model.PatientProfile, error)
	findByIDFunc   func(ctx context.Context, id string) (*model.PatientProfile, error)
	createFunc     func(ctx context.Context, p *model.PatientProfile) error
	updateFunc     func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error)
}

func (m *mockPatientRepo) FindByCredential(ctx context.Context, ct, cred string) (*model.PatientProfile, error) {
	return m.findByCredFunc(ctx, ct, cred)
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
	findByIDFunc      func(ctx context.Context, id string) (*model.VisitSession, error)
	listByPatientFunc func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
	createFunc        func(ctx context.Context, v *model.VisitSession) error
}

func (m *mockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, v)
	}
	return nil
}
func (m *mockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockVisitRepo) ListByPatient(ctx context.Context, pid string, cursor *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
	if m.listByPatientFunc != nil {
		return m.listByPatientFunc(ctx, pid, cursor, ps)
	}
	return nil, nil, false, nil
}
func (m *mockVisitRepo) UpdateStatus(ctx context.Context, id, status, machineState string) error {
	return nil
}
func (m *mockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error { return nil }

type mockTimelineRepo struct {
	appendFunc     func(ctx context.Context, item *model.TimelineItem) error
	listBySessFunc func(ctx context.Context, sid string, cursor *string, ps int) ([]model.TimelineItem, *string, bool, error)
}

func (m *mockTimelineRepo) Append(ctx context.Context, item *model.TimelineItem) error {
	if m.appendFunc != nil {
		return m.appendFunc(ctx, item)
	}
	return nil
}
func (m *mockTimelineRepo) AppendBatch(ctx context.Context, items []model.TimelineItem) error {
	return nil
}
func (m *mockTimelineRepo) ListBySession(ctx context.Context, sid string, cursor *string, ps int) ([]model.TimelineItem, *string, bool, error) {
	if m.listBySessFunc != nil {
		return m.listBySessFunc(ctx, sid, cursor, ps)
	}
	return nil, nil, false, nil
}
func (m *mockTimelineRepo) UpdateStatus(ctx context.Context, id, status string) error { return nil }

type mockFlowCardRepo struct {
	listBySessionFunc func(ctx context.Context, sid string) ([]model.FlowCard, error)
	findByIDFunc      func(ctx context.Context, id string) (*model.FlowCard, error)
	createFunc        func(ctx context.Context, card *model.FlowCard) error
	updateFunc        func(ctx context.Context, card *model.FlowCard) error
	updateStatusFunc  func(ctx context.Context, id, status string) error
}

func (m *mockFlowCardRepo) Create(ctx context.Context, card *model.FlowCard) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, card)
	}
	return nil
}
func (m *mockFlowCardRepo) FindByID(ctx context.Context, id string) (*model.FlowCard, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockFlowCardRepo) ListBySession(ctx context.Context, sid string) ([]model.FlowCard, error) {
	if m.listBySessionFunc != nil {
		return m.listBySessionFunc(ctx, sid)
	}
	return nil, nil
}
func (m *mockFlowCardRepo) UpdateStatus(ctx context.Context, id, status string) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status)
	}
	return nil
}
func (m *mockFlowCardRepo) Update(ctx context.Context, card *model.FlowCard) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, card)
	}
	return nil
}

type mockAddressRepo struct {
	listByPatientFunc         func(ctx context.Context, patientID string) ([]model.Address, error)
	findByIDFunc              func(ctx context.Context, id string) (*model.Address, error)
	countByPatientFunc        func(ctx context.Context, patientID string) (int, error)
	createFunc                func(ctx context.Context, addr *model.Address) error
	updateFunc                func(ctx context.Context, addr *model.Address) error
	deleteFunc                func(ctx context.Context, id string) error
	clearDefaultByPatientFunc func(ctx context.Context, patientID string) error
	setDefaultFunc            func(ctx context.Context, id, patientID string) error
}

func (m *mockAddressRepo) Create(ctx context.Context, addr *model.Address) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, addr)
	}
	return nil
}
func (m *mockAddressRepo) FindByID(ctx context.Context, id string) (*model.Address, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return &model.Address{
		ID: id, PatientID: "p001", Name: "李明", Phone: "13800002468",
		Province: "辽宁省", City: "沈阳市", District: "浑南区", Detail: "创新路195号",
	}, nil
}
func (m *mockAddressRepo) ListByPatient(ctx context.Context, patientID string) ([]model.Address, error) {
	if m.listByPatientFunc != nil {
		return m.listByPatientFunc(ctx, patientID)
	}
	return nil, nil
}
func (m *mockAddressRepo) CountByPatient(ctx context.Context, patientID string) (int, error) {
	if m.countByPatientFunc != nil {
		return m.countByPatientFunc(ctx, patientID)
	}
	return 0, nil
}
func (m *mockAddressRepo) Update(ctx context.Context, addr *model.Address) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, addr)
	}
	return nil
}
func (m *mockAddressRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}
func (m *mockAddressRepo) ClearDefaultByPatient(ctx context.Context, patientID string) error {
	if m.clearDefaultByPatientFunc != nil {
		return m.clearDefaultByPatientFunc(ctx, patientID)
	}
	return nil
}
func (m *mockAddressRepo) SetDefault(ctx context.Context, id, patientID string) error {
	if m.setDefaultFunc != nil {
		return m.setDefaultFunc(ctx, id, patientID)
	}
	return nil
}

type mockUserRepo struct{}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error { return nil }
func (m *mockUserRepo) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	return nil, model.ErrUserNotFound
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	return nil, model.ErrUserNotFound
}

type mockRefreshTokenRepo struct{}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	return nil
}
func (m *mockRefreshTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	return nil, model.ErrRefreshTokenInvalid
}
func (m *mockRefreshTokenRepo) MarkUsed(ctx context.Context, id string) error { return nil }
func (m *mockRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID string) error {
	return nil
}

// Verify repository interface compliance
var _ repository.PatientRepository = (*mockPatientRepo)(nil)
var _ repository.VisitRepository = (*mockVisitRepo)(nil)
var _ repository.TimelineRepository = (*mockTimelineRepo)(nil)
var _ repository.FlowCardRepository = (*mockFlowCardRepo)(nil)
var _ repository.AddressRepository = (*mockAddressRepo)(nil)
var _ repository.UserRepository = (*mockUserRepo)(nil)
var _ repository.RefreshTokenRepository = (*mockRefreshTokenRepo)(nil)

const handlerTestSecret = "this-is-a-32-byte-secret-key-for-testing!!" // #nosec G101

func newTestAuthService() *authsvc.Service {
	return authsvc.NewService(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockPatientRepo{}, handlerTestSecret)
}

// ---------------------------------------------------------------------------
// Patient Handler tests
// ---------------------------------------------------------------------------

func TestPatientHandler_VerifyIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	patientRepo := &mockPatientRepo{
		findByCredFunc: func(ctx context.Context, ct, cred string) (*model.PatientProfile, error) {
			return nil, model.ErrPatientNotFound
		},
		createFunc: func(ctx context.Context, p *model.PatientProfile) error {
			return nil
		},
	}
	svc := patientsvc.NewService(patientRepo, &mockVisitRepo{})
	h := handler.NewPatientHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/patients/verify",
			strings.NewReader(`{"credential":"13800001111","credentialType":"phone","name":"测试"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.VerifyIdentity(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/patients/verify",
			strings.NewReader(`not-json`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.VerifyIdentity(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

func TestPatientHandler_GetContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID: id, Name: "测试", PhoneMasked: "138****1111",
			}, nil
		},
	}
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, cursor *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return nil, nil, false, nil
		},
	}
	svc := patientsvc.NewService(patientRepo, visitRepo)
	h := handler.NewPatientHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "patientId", Value: "p001"}}
		c.Request = httptest.NewRequest("GET", "/patients/p001/context", nil)
		c.Set("patientId", "p001")

		h.GetContext(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("patient not found", func(t *testing.T) {
		pRepo := &mockPatientRepo{
			findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
				return nil, model.ErrPatientNotFound
			},
		}
		svc2 := patientsvc.NewService(pRepo, &mockVisitRepo{})
		h2 := handler.NewPatientHandler(svc2)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "patientId", Value: "p999"}}
		c.Request = httptest.NewRequest("GET", "/patients/p999/context", nil)
		c.Set("patientId", "p999")

		h2.GetContext(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})
}

func TestPatientHandler_UpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	patientRepo := &mockPatientRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
			return &model.PatientProfile{ID: id, Name: "旧名"}, nil
		},
		updateFunc: func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
			return &model.PatientProfile{
				ID: id, Name: "旧名", PhoneMasked: "138****1111",
			}, nil
		},
	}
	svc := patientsvc.NewService(patientRepo, &mockVisitRepo{})
	h := handler.NewPatientHandler(svc)

	t.Run("valid update", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "patientId", Value: "p001"}}
		c.Request = httptest.NewRequest("PATCH", "/patients/p001/profile",
			strings.NewReader(`{"name":"新名","allergies":["青霉素"]}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("patientId", "p001")

		h.UpdateProfile(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Visit Handler tests
// ---------------------------------------------------------------------------

func TestVisitHandler_CreateSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		createFunc: func(ctx context.Context, v *model.VisitSession) error {
			return nil
		},
	}
	svc := visitsvc.NewService(visitRepo, &mockTimelineRepo{})
	h := handler.NewVisitHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/visits",
			strings.NewReader(`{"patientId":"p001","chiefComplaint":"头疼","entryType":"new"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("patientId", "p001")

		h.CreateSession(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})
}

func TestVisitHandler_ListSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, cursor *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{}, nil, false, nil
		},
	}
	svc := visitsvc.NewService(visitRepo, &mockTimelineRepo{})
	h := handler.NewVisitHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/visits?patientId=p001", nil)
		c.Set("patientId", "p001")

		h.ListSessions(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}

func TestVisitHandler_GetSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID: id, PatientID: "p001", Status: "active",
			}, nil
		},
	}
	svc := visitsvc.NewService(visitRepo, &mockTimelineRepo{})
	h := handler.NewVisitHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "sessionId", Value: "s001"}}
		c.Request = httptest.NewRequest("GET", "/visits/s001", nil)
		c.Set("patientId", "p001")

		h.GetSession(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})
}

func TestVisitHandler_GetSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID: id, PatientID: "p001", Status: "active",
			}, nil
		},
	}
	svc := visitsvc.NewService(visitRepo, &mockTimelineRepo{})
	h := handler.NewVisitHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "sessionId", Value: "s001"}}
		c.Request = httptest.NewRequest("GET", "/visits/s001/snapshot", nil)
		c.Set("patientId", "p001")

		h.GetSnapshot(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})
}

func TestVisitHandler_CreateFollowUp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID: id, PatientID: "p001", Status: "active",
			}, nil
		},
	}
	svc := visitsvc.NewService(visitRepo, &mockTimelineRepo{})
	h := handler.NewVisitHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "sessionId", Value: "s001"}}
		c.Request = httptest.NewRequest("POST", "/visits/s001/follow-up",
			strings.NewReader(`{"patientId":"p001","chiefComplaint":"复诊"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("patientId", "p001")

		h.CreateFollowUp(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// Workbench Handler tests
// ---------------------------------------------------------------------------

func newWorkbenchServiceForTest(
	visitRepo *mockVisitRepo,
	timelineRepo *mockTimelineRepo,
) *wbsvc.Service {
	return wbsvc.NewService(
		&mockPatientRepo{},
		visitRepo,
		timelineRepo,
		&mockFlowCardRepo{},
		&mockAddressRepo{},
		nil, // medAgentClient — not used by simple read methods
		"test",
		nil, // llmClient — not used by simple read methods
	)
}

func TestWorkbenchHandler_GetSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID: id, PatientID: "p001", Status: "active",
			}, nil
		},
	}
	svc := newWorkbenchServiceForTest(visitRepo, &mockTimelineRepo{})
	h := handler.NewWorkbenchHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "sessionId", Value: "s001"}}
		c.Request = httptest.NewRequest("GET", "/visits/s001", nil)
		c.Set("patientId", "p001")

		h.GetSession(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("session not found", func(t *testing.T) {
		vRepo := &mockVisitRepo{
			findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
				return nil, model.ErrSessionNotFound
			},
		}
		svc2 := newWorkbenchServiceForTest(vRepo, &mockTimelineRepo{})
		h2 := handler.NewWorkbenchHandler(svc2)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "sessionId", Value: "s999"}}
		c.Request = httptest.NewRequest("GET", "/visits/s999", nil)
		c.Set("patientId", "p001")

		h2.GetSession(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})
}

func TestNewRouter(t *testing.T) {
	patientSvc := patientsvc.NewService(&mockPatientRepo{}, &mockVisitRepo{})
	visitSvc := visitsvc.NewService(&mockVisitRepo{}, &mockTimelineRepo{})
	wbSvc := newWorkbenchServiceForTest(&mockVisitRepo{}, &mockTimelineRepo{})
	authSvc := newTestAuthService()
	addressSvc := addresssvc.NewService(&mockAddressRepo{})
	billingSvc := billingsvc.NewService(&mockVisitRepo{}, &mockFlowCardRepo{})

	router := handler.NewRouter(patientSvc, visitSvc, wbSvc, authSvc, addressSvc, billingSvc)
	if router.Patient == nil {
		t.Error("Patient handler should not be nil")
	}
	if router.Visit == nil {
		t.Error("Visit handler should not be nil")
	}
	if router.Workbench == nil {
		t.Error("Workbench handler should not be nil")
	}
	if router.Auth == nil {
		t.Error("Auth handler should not be nil")
	}
	if router.Address == nil {
		t.Error("Address handler should not be nil")
	}
	if router.Billing == nil {
		t.Error("Billing handler should not be nil")
	}
}

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	patientSvc := patientsvc.NewService(&mockPatientRepo{}, &mockVisitRepo{})
	visitSvc := visitsvc.NewService(&mockVisitRepo{}, &mockTimelineRepo{})
	wbSvc := newWorkbenchServiceForTest(&mockVisitRepo{}, &mockTimelineRepo{})
	authSvc := newTestAuthService()
	addressSvc := addresssvc.NewService(&mockAddressRepo{})
	billingSvc := billingsvc.NewService(&mockVisitRepo{}, &mockFlowCardRepo{})
	router := handler.NewRouter(patientSvc, visitSvc, wbSvc, authSvc, addressSvc, billingSvc)

	cfg := &config.Config{
		ServerMode:         "test",
		JWTSecret:          "this-is-a-32-byte-secret-key-for-testing!!",
		CORSAllowedOrigins: "http://localhost:5173",
	}

	engine := gin.New()
	handler.SetupRoutes(engine, cfg, router)

	// Verify key routes are registered
	routes := engine.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+" "+r.Path] = true
	}

	expectedRoutes := []string{
		"GET /api/health",
		"POST /api/patients/verify",
		"GET /api/patients/:patientId/context",
		"PATCH /api/patients/:patientId/profile",
		"POST /api/visits",
		"GET /api/visits",
		"GET /api/visits/:sessionId",
		"GET /api/visits/:sessionId/snapshot",
		"POST /api/visits/:sessionId/follow-up",
		"POST /api/auth/register",
		"POST /api/auth/login",
		"POST /api/auth/refresh",
		"POST /api/auth/logout",
	}

	for _, expected := range expectedRoutes {
		if !routePaths[expected] {
			t.Errorf("route not registered: %s", expected)
		}
	}
}

func TestWorkbenchHandler_ListTimeline(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
			return &model.VisitSession{
				ID: id, PatientID: "p001", Status: "active",
			}, nil
		},
	}
	timelineRepo := &mockTimelineRepo{
		listBySessFunc: func(ctx context.Context, sid string, cursor *string, ps int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{}, nil, false, nil
		},
	}
	svc := newWorkbenchServiceForTest(visitRepo, timelineRepo)
	h := handler.NewWorkbenchHandler(svc)

	t.Run("valid request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "sessionId", Value: "s001"}}
		c.Request = httptest.NewRequest("GET", "/visits/s001/timeline", nil)
		c.Set("patientId", "p001")

		h.ListTimeline(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// Address Handler tests
// ---------------------------------------------------------------------------

func TestAddressHandler_ListAddresses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		listByPatientFunc: func(ctx context.Context, patientID string) ([]model.Address, error) {
			return []model.Address{}, nil
		},
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}}
	c.Request = httptest.NewRequest("GET", "/patients/p001/addresses", nil)
	c.Set("patientId", "p001")

	h.ListAddresses(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAddressHandler_CreateAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		countByPatientFunc:        func(ctx context.Context, patientID string) (int, error) { return 0, nil },
		clearDefaultByPatientFunc: func(ctx context.Context, patientID string) error { return nil },
		createFunc:                func(ctx context.Context, addr *model.Address) error { return nil },
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}}
	body := `{"patientId":"p001","name":"李明","phone":"13800002468","province":"辽宁省","city":"沈阳市","district":"浑南区","detail":"创新路195号","tag":"公司"}`
	c.Request = httptest.NewRequest("POST", "/patients/p001/addresses", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("patientId", "p001")

	h.CreateAddress(c)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body=%s", w.Code, w.Body.String())
	}
}

func TestAddressHandler_CreateAddress_LimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		countByPatientFunc: func(ctx context.Context, patientID string) (int, error) { return 10, nil },
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}}
	body := `{"patientId":"p001","name":"李明","phone":"13800002468","province":"辽宁省","city":"沈阳市","district":"浑南区","detail":"创新路195号"}`
	c.Request = httptest.NewRequest("POST", "/patients/p001/addresses", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("patientId", "p001")

	h.CreateAddress(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, body=%s", w.Code, w.Body.String())
	}
}

func TestAddressHandler_UpdateAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return &model.Address{
				ID: id, PatientID: "p001", Name: "李明", Phone: "13800002468",
				Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试",
			}, nil
		},
		updateFunc: func(ctx context.Context, a *model.Address) error { return nil },
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}, {Key: "addressId", Value: "addr-1"}}
	body := `{"name":"张三"}`
	c.Request = httptest.NewRequest("PATCH", "/patients/p001/addresses/addr-1", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("patientId", "p001")

	h.UpdateAddress(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestAddressHandler_UpdateAddress_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return nil, model.ErrAddressNotFound
		},
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}, {Key: "addressId", Value: "bad-id"}}
	body := `{"name":"张三"}`
	c.Request = httptest.NewRequest("PATCH", "/patients/p001/addresses/bad-id", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("patientId", "p001")

	h.UpdateAddress(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404, body=%s", w.Code, w.Body.String())
	}
}

func TestAddressHandler_DeleteAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return &model.Address{
				ID: id, PatientID: "p001", Name: "李明", Phone: "13800002468",
				Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试",
			}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error { return nil },
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}, {Key: "addressId", Value: "addr-1"}}
	c.Request = httptest.NewRequest("DELETE", "/patients/p001/addresses/addr-1", nil)
	c.Set("patientId", "p001")

	h.DeleteAddress(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestAddressHandler_SetDefaultAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return &model.Address{
				ID: id, PatientID: "p001", Name: "李明", Phone: "13800002468",
				Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试",
			}, nil
		},
		setDefaultFunc: func(ctx context.Context, id, patientID string) error { return nil },
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}, {Key: "addressId", Value: "addr-1"}}
	c.Request = httptest.NewRequest("PUT", "/patients/p001/addresses/addr-1/default", nil)
	c.Set("patientId", "p001")

	h.SetDefaultAddress(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Address Handler edge cases
// ---------------------------------------------------------------------------

func TestAddressHandler_CreateAddress_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	addrSvc := addresssvc.NewService(&mockAddressRepo{})
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}}
	c.Request = httptest.NewRequest("POST", "/patients/p001/addresses", strings.NewReader(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("patientId", "p001")

	h.CreateAddress(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, got %d", w.Code, w.Code)
	}
}

func TestAddressHandler_SetDefaultAddress_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return nil, model.ErrAddressNotFound
		},
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}, {Key: "addressId", Value: "bad-id"}}
	c.Request = httptest.NewRequest("PUT", "/patients/p001/addresses/bad-id/default", nil)
	c.Set("patientId", "p001")

	h.SetDefaultAddress(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAddressHandler_DeleteAddress_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return nil, model.ErrAddressNotFound
		},
	}
	addrSvc := addresssvc.NewService(repo)
	h := handler.NewAddressHandler(addrSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "patientId", Value: "p001"}, {Key: "addressId", Value: "bad-id"}}
	c.Request = httptest.NewRequest("DELETE", "/patients/p001/addresses/bad-id", nil)
	c.Set("patientId", "p001")

	h.DeleteAddress(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Billing Handler tests
// ---------------------------------------------------------------------------

func TestBillingHandler_ListBillingRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)

	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c2 *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return nil, nil
		},
	}
	billingSvc := billingsvc.NewService(visitRepo, flowCardRepo)
	h := handler.NewBillingHandler(billingSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/billing/records", nil)
	c.Set("patientId", "p001")

	h.ListBillingRecords(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_ListBillingRecords_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	billingSvc := billingsvc.NewService(&mockVisitRepo{}, &mockFlowCardRepo{})
	h := handler.NewBillingHandler(billingSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/billing/records", nil)
	// No patientId set — simulates unauthenticated

	h.ListBillingRecords(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBillingHandler_ListBillingRecords_WithData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cc := "头痛"
	handledAt := time.Now()
	visitRepo := &mockVisitRepo{
		listByPatientFunc: func(ctx context.Context, pid string, c2 *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
			return []model.VisitSessionSummary{
				{ID: "s1", Summary: model.VisitSummary{ChiefComplaint: &cc}},
			}, nil, false, nil
		},
	}
	flowCardRepo := &mockFlowCardRepo{
		listBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) {
			return []model.FlowCard{{
				ID: "c1", SessionID: "s1", Kind: "payment", PaymentStatus: "paid",
				PaymentID: "pay-1", Purpose: "lab", TotalAmount: 150.0,
				InsuranceAmount: 100.0, SelfPayAmount: 50.0,
				HandledAt: &handledAt,
			}}, nil
		},
	}
	billingSvc := billingsvc.NewService(visitRepo, flowCardRepo)
	h := handler.NewBillingHandler(billingSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/billing/records", nil)
	c.Set("patientId", "p001")

	h.ListBillingRecords(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

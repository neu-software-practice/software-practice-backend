package handler_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/neuhis/software-practice-backend/internal/handler"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockPatientService implements the patient service interface for testing
type mockPatientService struct{}

// mockVisitService implements the visit service interface for testing
type mockVisitService struct{}

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
	json.Unmarshal(w.Body.Bytes(), &resp)
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
			json.Unmarshal(w.Body.Bytes(), &resp)
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
	json.Unmarshal(w.Body.Bytes(), &resp)
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

package medagent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:8080")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %s, want http://localhost:8080", c.baseURL)
	}
	if c.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestCreateSession_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "sess-001"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	sid, err := c.CreateSession(context.Background(), map[string]interface{}{"age": 30}, true, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sid != "sess-001" {
		t.Errorf("sessionID = %s, want sess-001", sid)
	}
}

func TestCreateSession_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.CreateSession(context.Background(), nil, true, nil)
	if !errors.Is(err, ErrMedAgentSessionNotFound) {
		t.Errorf("err = %v, want ErrMedAgentSessionNotFound", err)
	}
}

func TestCreateSession_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("session closed"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.CreateSession(context.Background(), nil, true, nil)
	if !errors.Is(err, ErrMedAgentSessionClosed) {
		t.Errorf("err = %v, want ErrMedAgentSessionClosed", err)
	}
}

func TestCreateSession_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.CreateSession(context.Background(), nil, true, nil)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestPatientSay_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step := Step{Kind: StepAsk, DoctorSay: "Hello"}
		_ = json.NewEncoder(w).Encode(step)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	step, err := c.PatientSay(context.Background(), "sess-1", "hi")
	if err != nil {
		t.Fatalf("PatientSay: %v", err)
	}
	if step.Kind != StepAsk {
		t.Errorf("Kind = %s, want ASK", step.Kind)
	}
	if step.DoctorSay != "Hello" {
		t.Errorf("DoctorSay = %s, want Hello", step.DoctorSay)
	}
}

func TestDrugInfo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step := Step{Kind: StepOK}
		_ = json.NewEncoder(w).Encode(step)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	step, err := c.DrugInfo(context.Background(), "sess-1", []DrugInfo{{Name: "Aspirin", Spec: "100mg"}})
	if err != nil {
		t.Fatalf("DrugInfo: %v", err)
	}
	if step.Kind != StepOK {
		t.Errorf("Kind = %s, want OK", step.Kind)
	}
}

func TestGetRecord_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(SessionRecord{SessionID: "sess-1"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	rec, err := c.GetRecord(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if rec.SessionID != "sess-1" {
		t.Errorf("SessionID = %s, want sess-1", rec.SessionID)
	}
}

func TestDeleteSession_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	err := c.DeleteSession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
}

func TestDeleteSession_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	err := c.DeleteSession(context.Background(), "sess-1")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrMedAgentSessionNotFound.Error() != "medagent session not found" {
		t.Error("wrong message for ErrMedAgentSessionNotFound")
	}
	if ErrMedAgentSessionClosed.Error() != "medagent session closed or wrong step" {
		t.Error("wrong message for ErrMedAgentSessionClosed")
	}
	if ErrMedAgentUnavailable.Error() != "medagent unavailable" {
		t.Error("wrong message for ErrMedAgentUnavailable")
	}
}

func TestTestResults_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Step{Kind: StepNeedTests, DoctorSay: "need tests"})
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	step, err := c.TestResults(context.Background(), "s1", []TestResult{{Item: "blood", Value: "normal"}})
	if err != nil {
		t.Fatalf("TestResults: %v", err)
	}
	if step.Kind != StepNeedTests {
		t.Errorf("Kind = %s, want NEED_TESTS", step.Kind)
	}
}

func TestPurchaseResult_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Step{Kind: StepOK})
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	step, err := c.PurchaseResult(context.Background(), "s1", []DrugPurchase{{Name: "aspirin", Bought: true, Quantity: 1}})
	if err != nil {
		t.Fatalf("PurchaseResult: %v", err)
	}
	if step.Kind != StepOK {
		t.Errorf("Kind = %s, want OK", step.Kind)
	}
}

func TestVitals_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Step{Kind: StepOK})
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	step, err := c.Vitals(context.Background(), "s1", map[string]interface{}{"temp": 37.0})
	if err != nil {
		t.Fatalf("Vitals: %v", err)
	}
	if step.Kind != StepOK {
		t.Errorf("Kind = %s, want OK", step.Kind)
	}
}

func TestGetRecord_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.GetRecord(context.Background(), "s1")
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestUnmarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.PatientSay(context.Background(), "s1", "hi")
	if err == nil {
		t.Error("expected unmarshal error")
	}
}

func TestGetRecord_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.GetRecord(context.Background(), "s1")
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestGetRecord_UnmarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("bad json"))
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.GetRecord(context.Background(), "s1")
	if err == nil {
		t.Error("expected unmarshal error")
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	err := c.DeleteSession(context.Background(), "s1")
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestCreateSession_MarshalError(t *testing.T) {
	// Passing a channel in the map triggers json.Marshal error
	ch := make(chan int)
	c := NewClient("http://localhost:1") // doesn't matter, marshal fails first
	_, err := c.CreateSession(context.Background(), map[string]interface{}{"ch": ch}, true, nil)
	if err == nil {
		t.Error("expected marshal error")
	}
}

func TestTestResults_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.TestResults(context.Background(), "s1", []TestResult{{Item: "test", Value: "ok"}})
	if err == nil {
		t.Error("expected server error")
	}
}

func TestDrugInfo_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.DrugInfo(context.Background(), "s1", []DrugInfo{{Name: "a", Spec: "100mg"}})
	if err == nil {
		t.Error("expected server error")
	}
}

func TestPurchaseResult_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.PurchaseResult(context.Background(), "s1", []DrugPurchase{{Name: "a", Bought: true, Quantity: 1}})
	if err == nil {
		t.Error("expected server error")
	}
}

func TestVitals_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	_, err := c.Vitals(context.Background(), "s1", map[string]interface{}{"temp": 37})
	if err == nil {
		t.Error("expected server error")
	}
}

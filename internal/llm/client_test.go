package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/neuhis/software-practice-backend/internal/llm"
)

func TestChatComplete_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}

		// Decode request body to verify structure
		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "test-model" {
			t.Errorf("unexpected model: %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"choices":[{"message":{"content":"发热伴咳嗽3天"}}]}`
		_, _ = w.Write([]byte(resp))
	}))
	defer ts.Close()

	client := llm.New(llm.Config{
		BaseURL: ts.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	})

	result, err := client.ChatComplete(context.Background(), "system prompt", "user input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "发热伴咳嗽3天" {
		t.Errorf("got %q, want %q", result, "发热伴咳嗽3天")
	}
}

func TestChatComplete_EmptyChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer ts.Close()

	client := llm.New(llm.Config{BaseURL: ts.URL, APIKey: "key", Model: "m"})
	_, err := client.ChatComplete(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
	if !errors.Is(err, llm.ErrLLMUnavailable) {
		t.Errorf("expected ErrLLMUnavailable, got: %v", err)
	}
}

func TestChatComplete_NonJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer ts.Close()

	client := llm.New(llm.Config{BaseURL: ts.URL, APIKey: "key", Model: "m"})
	_, err := client.ChatComplete(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !errors.Is(err, llm.ErrLLMUnavailable) {
		t.Errorf("expected ErrLLMUnavailable, got: %v", err)
	}
}

func TestChatComplete_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server down"}`))
	}))
	defer ts.Close()

	client := llm.New(llm.Config{BaseURL: ts.URL, APIKey: "key", Model: "m"})
	_, err := client.ChatComplete(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !errors.Is(err, llm.ErrLLMUnavailable) {
		t.Errorf("expected ErrLLMUnavailable, got: %v", err)
	}
}

func TestChatComplete_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := llm.New(llm.Config{
		BaseURL: ts.URL,
		APIKey:  "key",
		Model:   "m",
		Timeout: 100 * time.Millisecond,
	})
	_, err := client.ChatComplete(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, llm.ErrLLMUnavailable) {
		t.Errorf("expected ErrLLMUnavailable, got: %v", err)
	}
}

func TestChatComplete_ContextCancelled(t *testing.T) {
	client := llm.New(llm.Config{BaseURL: "http://localhost:1", APIKey: "key", Model: "m"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.ChatComplete(ctx, "sys", "usr")
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestChatComplete_LargeErrorBody(t *testing.T) {
	largeBody := strings.Repeat("x", 1024)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(largeBody))
	}))
	defer ts.Close()

	client := llm.New(llm.Config{BaseURL: ts.URL, APIKey: "key", Model: "m"})
	_, err := client.ChatComplete(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should be truncated via snippet (max 512 chars)
	errMsg := err.Error()
	if len(errMsg) > 600 {
		t.Errorf("error message too long (%d chars), snippet should truncate", len(errMsg))
	}
}

func TestNewFromProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		baseURL  string
	}{
		{"deepseek", "deepseek", ""},
		{"qwen", "qwen", ""},
		{"custom baseURL", "ignored", "http://custom.example.com/v1"},
		{"empty provider uses deepseek default", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := llm.NewFromProvider(tt.provider, "key", "model", tt.baseURL)
			if client == nil {
				t.Fatal("expected non-nil client")
			}
		})
	}
}

func TestNewDeepSeek(t *testing.T) {
	client := llm.NewDeepSeek("key", "model")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewQwen(t *testing.T) {
	client := llm.NewQwen("key", "model")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

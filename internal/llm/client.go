package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	deepSeekBaseURL = "https://api.deepseek.com/v1"
	qwenBaseURL     = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	defaultTimeout  = 60 * time.Second
)

// ErrLLMUnavailable indicates the LLM service is unreachable or returned an error.
var ErrLLMUnavailable = errors.New("llm service unavailable")

// Config holds constructor parameters for Client.
// BaseURL, APIKey, and Model are required.
// Timeout defaults to 60s when zero; HTTPClient can be injected for testing.
type Config struct {
	BaseURL    string
	APIKey     string
	Model      string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Client is an OpenAI-compatible LLM client for simple text chat completions.
type Client struct {
	cfg  Config
	http *http.Client
}

// New creates a Client from Config.
func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		to := cfg.Timeout
		if to == 0 {
			to = defaultTimeout
		}
		hc = &http.Client{Timeout: to}
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{cfg: cfg, http: hc}
}

// NewDeepSeek creates a Client pre-filled with DeepSeek's base URL.
func NewDeepSeek(apiKey, model string) *Client {
	return New(Config{BaseURL: deepSeekBaseURL, APIKey: apiKey, Model: model})
}

// NewQwen creates a Client pre-filled with Qwen (DashScope compatible mode) base URL.
func NewQwen(apiKey, model string) *Client {
	return New(Config{BaseURL: qwenBaseURL, APIKey: apiKey, Model: model})
}

// NewFromProvider creates a Client based on provider name, API key, and model.
// Supported providers: "deepseek", "qwen". Falls back to using baseURL directly for unknown providers.
func NewFromProvider(provider, apiKey, model, baseURL string) *Client {
	switch strings.ToLower(provider) {
	case "deepseek":
		return NewDeepSeek(apiKey, model)
	case "qwen":
		return NewQwen(apiKey, model)
	default:
		if baseURL == "" {
			baseURL = deepSeekBaseURL
		}
		return New(Config{BaseURL: baseURL, APIKey: apiKey, Model: model})
	}
}

// ChatComplete sends a simple system+user message pair to the chat completions endpoint
// and returns the assistant's text response.
func (c *Client) ChatComplete(ctx context.Context, system, user string) (string, error) {
	msgs := []wireMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
	reqBody := chatRequest{
		Model:    c.cfg.Model,
		Messages: msgs,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("llm: marshal request: %w", err)
	}

	respBody, err := c.post(ctx, body)
	if err != nil {
		return "", err
	}

	var resp chatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("llm: unmarshal response (%v): %w", err, ErrLLMUnavailable)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("llm: empty choices: %w", ErrLLMUnavailable)
	}

	return resp.Choices[0].Message.Content, nil
}

// post sends a POST request to /chat/completions and returns the response body.
func (c *Client) post(ctx context.Context, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: create request (%v): %w", err, ErrLLMUnavailable)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm: request failed (%v): %w", err, ErrLLMUnavailable)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read response (%v): %w", err, ErrLLMUnavailable)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm: non-2xx response %d: %s: %w", resp.StatusCode, snippet(respBody), ErrLLMUnavailable)
	}
	return respBody, nil
}

// snippet truncates response body for error messages.
func snippet(b []byte) string {
	const max = 512
	if len(b) <= max {
		return string(b)
	}
	end := max
	for end > 0 && !utf8.RuneStart(b[end]) {
		end--
	}
	return string(b[:end]) + "…"
}

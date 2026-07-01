package medagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	// ErrMedAgentSessionNotFound indicates the medAgent session does not exist.
	ErrMedAgentSessionNotFound = errors.New("medagent session not found")
	// ErrMedAgentSessionClosed indicates the session is closed or the step is wrong.
	ErrMedAgentSessionClosed = errors.New("medagent session closed or wrong step")
	// ErrMedAgentUnavailable indicates medAgent is unreachable or returned a server error.
	ErrMedAgentUnavailable = errors.New("medagent unavailable")
)

// Client is an HTTP client for the medAgent independent process.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new medAgent HTTP client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMedAgentUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w", ErrMedAgentSessionNotFound)
	}
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("%w: %s", ErrMedAgentSessionClosed, string(respBody))
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("medagent error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMedAgentUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("medagent error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// CreateSession creates a new medAgent session.
func (c *Client) CreateSession(ctx context.Context, profile map[string]interface{}, initial bool, prior []interface{}) (string, error) {
	body := map[string]interface{}{
		"profile": profile,
		"initial": initial,
		"prior":   prior,
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := c.post(ctx, "/sessions", body, &result); err != nil {
		return "", err
	}
	return result.SessionID, nil
}

// PatientSay sends a patient message to medAgent and returns the Step.
func (c *Client) PatientSay(ctx context.Context, sessionID string, message string) (*Step, error) {
	body := map[string]string{"message": message}
	var step Step
	if err := c.post(ctx, fmt.Sprintf("/sessions/%s/patient-say", sessionID), body, &step); err != nil {
		return nil, err
	}
	return &step, nil
}

// TestResults sends test results to medAgent.
func (c *Client) TestResults(ctx context.Context, sessionID string, results []TestResult) (*Step, error) {
	body := map[string]interface{}{"results": results}
	var step Step
	if err := c.post(ctx, fmt.Sprintf("/sessions/%s/test-results", sessionID), body, &step); err != nil {
		return nil, err
	}
	return &step, nil
}

// DrugInfo sends drug information to medAgent.
func (c *Client) DrugInfo(ctx context.Context, sessionID string, infos []DrugInfo) (*Step, error) {
	body := map[string]interface{}{"infos": infos}
	var step Step
	if err := c.post(ctx, fmt.Sprintf("/sessions/%s/drug-info", sessionID), body, &step); err != nil {
		return nil, err
	}
	return &step, nil
}

// PurchaseResult sends purchase results to medAgent.
func (c *Client) PurchaseResult(ctx context.Context, sessionID string, results []DrugPurchase) (*Step, error) {
	body := map[string]interface{}{"results": results}
	var step Step
	if err := c.post(ctx, fmt.Sprintf("/sessions/%s/purchase-result", sessionID), body, &step); err != nil {
		return nil, err
	}
	return &step, nil
}

// Vitals sends vital signs to medAgent for guardian check.
func (c *Client) Vitals(ctx context.Context, sessionID string, vitals map[string]interface{}) (*Step, error) {
	body := map[string]interface{}{"vitals": vitals}
	var step Step
	if err := c.post(ctx, fmt.Sprintf("/sessions/%s/vitals", sessionID), body, &step); err != nil {
		return nil, err
	}
	return &step, nil
}

// GetRecord exports the session record from medAgent.
func (c *Client) GetRecord(ctx context.Context, sessionID string) (*SessionRecord, error) {
	var record SessionRecord
	if err := c.get(ctx, fmt.Sprintf("/sessions/%s/record", sessionID), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteSession deletes a medAgent session.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+fmt.Sprintf("/sessions/%s", sessionID), nil)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMedAgentUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete session error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

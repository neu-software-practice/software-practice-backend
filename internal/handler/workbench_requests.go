package handler

// SendMessageRequest represents the request body for sending a message.
type SendMessageRequest struct {
	SessionID       string `json:"sessionId"`
	Content         string `json:"content"`
	ClientMessageID string `json:"clientMessageId"`
}

// StreamAssistantRequest represents the request body for streaming assistant response.
type StreamAssistantRequest struct {
	SessionID       string `json:"sessionId"`
	RequestID       string `json:"requestId"`
	ClientMessageID string `json:"clientMessageId,omitempty"`
}

// LabDecisionRequest represents the request body for lab decision.
type LabDecisionRequest struct {
	SessionID string `json:"sessionId"`
	CardID    string `json:"cardId"`
	Decision  string `json:"decision"`
}

// AckAdviceRequest represents the request body for acknowledging advice.
type AckAdviceRequest struct {
	SessionID string `json:"sessionId"`
	CardID    string `json:"cardId"`
}

// ClassifyIntentRequest represents the request body for intent classification.
type ClassifyIntentRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

// VitalsRequest represents the request body for reporting vitals.
type VitalsRequest struct {
	SessionID string                 `json:"sessionId"`
	Source    string                 `json:"source"`
	Vitals    map[string]interface{} `json:"vitals,omitempty"`
	Symptoms  []string               `json:"symptoms"`
}

// TimerRequest represents the request body for timer operations.
type TimerRequest struct {
	SessionID string `json:"sessionId"`
	Action    string `json:"action"`
}

// LockQuestionRequest represents the request body for asking a locked question.
type LockQuestionRequest struct {
	SessionID string `json:"sessionId"`
	CardID    string `json:"cardId"`
	Content   string `json:"content"`
	RequestID string `json:"requestId"`
}

// ConsultRequest represents the request body for consultation reply.
type ConsultRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
	RequestID string `json:"requestId"`
}

// GenerateTitleRequest represents the request body for generating a title.
type GenerateTitleRequest struct {
	SessionID string `json:"sessionId"`
}

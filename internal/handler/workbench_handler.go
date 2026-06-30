package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
	"github.com/neuhis/software-practice-backend/pkg/api"
)

// WorkbenchHandler handles workbench-related HTTP endpoints.
type WorkbenchHandler struct {
	svc *wbsvc.Service
}

// NewWorkbenchHandler creates a new WorkbenchHandler.
func NewWorkbenchHandler(svc *wbsvc.Service) *WorkbenchHandler {
	return &WorkbenchHandler{svc: svc}
}

// getSessionAndVerify loads the session and verifies patient ownership.
// It writes the appropriate HTTP error response on failure so callers can simply return.
func (h *WorkbenchHandler) getSessionAndVerify(c *gin.Context) (*model.VisitSession, error) {
	sessionID := ParseSessionID(c)
	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if err == model.ErrSessionNotFound {
			apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		} else {
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return nil, err
	}
	if !RequirePatientID(c, session.PatientID) {
		return nil, apperrors.NewForbiddenError("access denied")
	}
	return session, nil
}

// GetSession is an alias for workbench's session retrieval.
func (h *WorkbenchHandler) GetSession(c *gin.Context) {
	session, err := h.getSessionAndVerify(c)
	if err != nil {
		return
	}
	WriteSuccess(c, http.StatusOK, session)
}

// ListTimeline handles GET /visits/:sessionId/timeline
func (h *WorkbenchHandler) ListTimeline(c *gin.Context) {
	sessionID := ParseSessionID(c)
	cursor := api.CursorFromQuery(c.Query("cursor"))
	pageSize := ParseQueryInt(c, "pageSize", 50)

	// Verify session access
	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if !RequirePatientID(c, session.PatientID) {
		return
	}

	items, nextCursor, hasMore, err := h.svc.ListTimeline(c.Request.Context(), sessionID, cursor, pageSize)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WritePageResult(c, api.NewPageResult(items, nextCursor, hasMore))
}

// SendMessage handles POST /visits/:sessionId/messages
func (h *WorkbenchHandler) SendMessage(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type sendMessageInput struct {
		SessionID       string `json:"sessionId"`
		Content         string `json:"content"`
		ClientMessageID string `json:"clientMessageId"`
	}
	input, err := BindJSON[sendMessageInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if !RequirePatientID(c, session.PatientID) {
		return
	}

	result, err := h.svc.SendMessage(c.Request.Context(), wbsvc.SendMessageInput{
		SessionID:       input.SessionID,
		Content:         input.Content,
		ClientMessageID: input.ClientMessageID,
	})
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// StreamAssistantMessage handles POST /visits/:sessionId/assistant-stream (SSE)
func (h *WorkbenchHandler) StreamAssistantMessage(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type streamInput struct {
		SessionID       string `json:"sessionId"`
		RequestID       string `json:"requestId"`
		ClientMessageID string `json:"clientMessageId,omitempty"`
	}
	input, err := BindJSON[streamInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if !RequirePatientID(c, session.PatientID) {
		return
	}

	writer, err := NewSSEWriter(c)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError("SSE not supported"))
		return
	}

	err = h.svc.StreamAssistantMessage(c.Request.Context(), wbsvc.StreamAssistantInput{
		SessionID:       input.SessionID,
		RequestID:       input.RequestID,
		ClientMessageID: input.ClientMessageID,
	}, func(event model.AssistantStreamEvent) error {
		return writer.WriteEvent(event)
	})

	if err != nil {
		writer.WriteError(input.SessionID, input.RequestID, err)
	}
	writer.Close()
}

// SubmitLabDecision handles POST /visits/:sessionId/lab-decision
func (h *WorkbenchHandler) SubmitLabDecision(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type labDecisionInput struct {
		SessionID string `json:"sessionId"`
		CardID    string `json:"cardId"`
		Decision  string `json:"decision"`
	}
	input, err := BindJSON[labDecisionInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.SubmitLabDecision(c.Request.Context(), wbsvc.SubmitLabDecisionInput{
		SessionID: input.SessionID,
		CardID:    input.CardID,
		Decision:  input.Decision,
	})
	if err != nil {
		apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeCardNotFound, err.Error(), http.StatusNotFound))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// SubmitPayment handles POST /visits/:sessionId/payments
func (h *WorkbenchHandler) SubmitPayment(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[model.SubmitPaymentInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.SubmitPayment(c.Request.Context(), input)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeCardNotFound, err.Error(), http.StatusNotFound))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// SubmitFulfillment handles POST /visits/:sessionId/fulfillment
func (h *WorkbenchHandler) SubmitFulfillment(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[wbsvc.SubmitFulfillmentInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.SubmitFulfillment(c.Request.Context(), input)
	if err != nil {
		switch err {
		case model.ErrCardNotFound:
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeCardNotFound, err.Error(), http.StatusNotFound))
		case model.ErrAddressRequired:
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeAddressRequired, err.Error(), http.StatusBadRequest))
		case model.ErrAddressNotFound:
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeAddressNotFound, err.Error(), http.StatusNotFound))
		default:
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// SubmitTreatmentExecution handles POST /visits/:sessionId/treatment-execution
func (h *WorkbenchHandler) SubmitTreatmentExecution(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[wbsvc.SubmitTreatmentExecutionInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.SubmitTreatmentExecution(c.Request.Context(), input)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeCardNotFound, err.Error(), http.StatusNotFound))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// AckAdvice handles POST /visits/:sessionId/advice-ack
func (h *WorkbenchHandler) AckAdvice(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type ackInput struct {
		SessionID string `json:"sessionId"`
		CardID    string `json:"cardId"`
	}
	input, err := BindJSON[ackInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.AckAdvice(c.Request.Context(), wbsvc.AckAdviceInput{
		SessionID: input.SessionID,
		CardID:    input.CardID,
	})
	if err != nil {
		apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeCardNotFound, err.Error(), http.StatusNotFound))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// ClassifyIntent handles POST /visits/:sessionId/classify-intent
func (h *WorkbenchHandler) ClassifyIntent(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type classifyInput struct {
		SessionID string `json:"sessionId"`
		Content   string `json:"content"`
	}
	input, err := BindJSON[classifyInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.ClassifyIntent(c.Request.Context(), wbsvc.ClassifyIntentInput{
		SessionID: input.SessionID,
		Content:   input.Content,
	})
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// ReportVitals handles POST /visits/:sessionId/vitals
func (h *WorkbenchHandler) ReportVitals(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type vitalsInput struct {
		SessionID string                 `json:"sessionId"`
		Source    string                 `json:"source"`
		Vitals    map[string]interface{} `json:"vitals,omitempty"`
		Symptoms  []string               `json:"symptoms"`
	}
	input, err := BindJSON[vitalsInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if len(input.Symptoms) == 0 {
		apperrors.WriteValidationError(c, "symptoms is required")
		return
	}

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.ReportVitals(c.Request.Context(), wbsvc.ReportVitalsInput{
		SessionID: input.SessionID,
		Source:    input.Source,
		Vitals:    input.Vitals,
		Symptoms:  input.Symptoms,
	})
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// ExitVisit handles POST /visits/:sessionId/exit
func (h *WorkbenchHandler) ExitVisit(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[model.ExitVisitInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.ExitVisit(c.Request.Context(), input)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// ToggleTimer handles POST /visits/:sessionId/timer (pause/resume)
func (h *WorkbenchHandler) ToggleTimer(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type timerInput struct {
		SessionID string `json:"sessionId"`
		Action    string `json:"action"` // pause, resume
	}
	input, err := BindJSON[timerInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	var result *model.VisitSession

	switch input.Action {
	case "pause":
		result, err = h.svc.PauseTimer(c.Request.Context(), input.SessionID)
	case "resume":
		result, err = h.svc.ResumeTimer(c.Request.Context(), input.SessionID)
	default:
		apperrors.WriteValidationError(c, "action must be 'pause' or 'resume'")
		return
	}

	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// DismissEmergency handles POST /visits/:sessionId/dismiss-emergency
func (h *WorkbenchHandler) DismissEmergency(c *gin.Context) {
	sessionID := ParseSessionID(c)

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, tlItem, err := h.svc.DismissEmergency(c.Request.Context(), wbsvc.DismissEmergencyInput{
		SessionID: sessionID,
	})
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, gin.H{
		"session":      result,
		"timelineItem": tlItem,
	})
}

// AskLockedQuestion handles POST /visits/:sessionId/lock-question (SSE)
func (h *WorkbenchHandler) AskLockedQuestion(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type lockQuestionInput struct {
		SessionID string `json:"sessionId"`
		CardID    string `json:"cardId"`
		Content   string `json:"content"`
		RequestID string `json:"requestId"`
	}
	input, err := BindJSON[lockQuestionInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	writer, _ := NewSSEWriter(c)
	err = h.svc.AskLockedQuestion(c.Request.Context(), input.SessionID, input.CardID, input.Content, input.RequestID,
		func(event model.AssistantStreamEvent) error {
			return writer.WriteEvent(event)
		})
	if err != nil {
		writer.WriteError(input.SessionID, input.RequestID, err)
	}
	writer.Close()
}

// StreamConsultationReply handles POST /visits/:sessionId/consult (SSE)
func (h *WorkbenchHandler) StreamConsultationReply(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type consultInput struct {
		SessionID string `json:"sessionId"`
		Content   string `json:"content"`
		RequestID string `json:"requestId"`
	}
	input, err := BindJSON[consultInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	writer, _ := NewSSEWriter(c)
	err = h.svc.StreamConsultationReply(c.Request.Context(), input.SessionID, input.Content, input.RequestID,
		func(event model.AssistantStreamEvent) error {
			return writer.WriteEvent(event)
		})
	if err != nil {
		writer.WriteError(input.SessionID, input.RequestID, err)
	}
	writer.Close()
}

// GenerateTitle handles POST /visits/:sessionId/generate-title
func (h *WorkbenchHandler) GenerateTitle(c *gin.Context) {
	sessionID := ParseSessionID(c)

	type generateTitleInput struct {
		SessionID string `json:"sessionId"`
	}
	input, err := BindJSON[generateTitleInput](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}

	// Validate path param matches body
	if input.SessionID != "" && input.SessionID != sessionID {
		apperrors.WriteValidationError(c, "sessionId in body does not match path parameter")
		return
	}

	// Verify session access
	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		return
	}
	if !RequirePatientID(c, session.PatientID) {
		return
	}

	title, err := h.svc.GenerateTitle(c.Request.Context(), sessionID)
	if err != nil {
		switch {
		case err.Error() == "session not found":
			apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		case err.Error() == "title already exists":
			// Idempotent: return existing title
			c.JSON(http.StatusOK, api.SuccessResponse(gin.H{
				"sessionId": sessionID,
				"title":     *session.Summary.Title,
			}))
		case err.Error() == "llm unavailable":
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeLLMUnavailable, "LLM service unavailable", 503))
		default:
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse(gin.H{
		"sessionId": sessionID,
		"title":     title,
	}))
}

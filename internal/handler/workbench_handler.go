package handler

import (
	"errors"
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
		if errors.Is(err, model.ErrSessionNotFound) {
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
	if _, err := h.getSessionAndVerify(c); err != nil {
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

	input, err := BindJSON[SendMessageRequest](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	result, err := h.svc.SendMessage(c.Request.Context(), wbsvc.SendMessageInput{
		SessionID:       input.SessionID,
		Content:         input.Content,
		ClientMessageID: input.ClientMessageID,
	})
	if err != nil {
		if errors.Is(err, model.ErrInvalidState) {
			apperrors.WriteError(c, apperrors.NewApiError(
				apperrors.CodeInvalidState,
				err.Error(),
				http.StatusUnprocessableEntity,
			))
			return
		}
		apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		return
	}

	WriteSuccess(c, http.StatusOK, result)
}

// StreamAssistantMessage handles POST /visits/:sessionId/assistant-stream (SSE)
func (h *WorkbenchHandler) StreamAssistantMessage(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[StreamAssistantRequest](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
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
		var apiErr *apperrors.ApiError
		if errors.As(err, &apiErr) {
			writer.WriteError(input.SessionID, input.RequestID, apiErr)
		} else if errors.Is(err, model.ErrInvalidState) {
			writer.WriteError(input.SessionID, input.RequestID, apperrors.NewApiError(
				apperrors.CodeInvalidState,
				err.Error(),
				http.StatusUnprocessableEntity,
			))
		} else if errors.Is(err, model.ErrDrugNotFound) {
			writer.WriteError(input.SessionID, input.RequestID, apperrors.NewApiError(
				apperrors.CodeDrugNotFound,
				err.Error(),
				http.StatusUnprocessableEntity,
			))
		} else if errors.Is(err, model.ErrDrugStockInsufficient) {
			writer.WriteError(input.SessionID, input.RequestID, apperrors.NewApiError(
				apperrors.CodeDrugStockInsufficient,
				err.Error(),
				http.StatusConflict,
			))
		} else {
			writer.WriteError(input.SessionID, input.RequestID, apperrors.NewInternalError(err.Error()))
		}
	}
	writer.Close()
}

// SubmitLabDecision handles POST /visits/:sessionId/lab-decision
func (h *WorkbenchHandler) SubmitLabDecision(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[LabDecisionRequest](c)
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
		switch {
		case errors.Is(err, model.ErrCardNotFound):
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeCardNotFound, err.Error(), http.StatusNotFound))
		case errors.Is(err, model.ErrAddressRequired):
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeAddressRequired, err.Error(), http.StatusBadRequest))
		case errors.Is(err, model.ErrAddressNotFound):
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeAddressNotFound, err.Error(), http.StatusNotFound))
		case errors.Is(err, model.ErrDrugNotFound):
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeDrugNotFound, err.Error(), http.StatusUnprocessableEntity))
		case errors.Is(err, model.ErrDrugStockInsufficient):
			apperrors.WriteError(c, apperrors.NewApiError(apperrors.CodeDrugStockInsufficient, err.Error(), http.StatusConflict))
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

	input, err := BindJSON[AckAdviceRequest](c)
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

	input, err := BindJSON[ClassifyIntentRequest](c)
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

	input, err := BindJSON[VitalsRequest](c)
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

	input, err := BindJSON[TimerRequest](c)
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
	case model.TimerActionPause:
		result, err = h.svc.PauseTimer(c.Request.Context(), input.SessionID)
	case model.TimerActionResume:
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
		if errors.Is(err, model.ErrSessionNotFound) {
			apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, "session not found")
		} else if errors.Is(err, model.ErrValidation) {
			apperrors.WriteValidationError(c, "session is not in emergency state")
		} else {
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	WriteSuccess(c, http.StatusOK, DismissEmergencyResult{
		Session:      result,
		TimelineItem: tlItem,
	})
}

// AskLockedQuestion handles POST /visits/:sessionId/lock-question (SSE)
func (h *WorkbenchHandler) AskLockedQuestion(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[LockQuestionRequest](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	writer, err := NewSSEWriter(c)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError("SSE not supported"))
		return
	}
	err = h.svc.AskLockedQuestion(c.Request.Context(), input.SessionID, input.CardID, input.Content, input.RequestID,
		func(event model.AssistantStreamEvent) error {
			return writer.WriteEvent(event)
		})
	if err != nil {
		var apiErr *apperrors.ApiError
		if errors.As(err, &apiErr) {
			writer.WriteError(input.SessionID, input.RequestID, apiErr)
		} else {
			writer.WriteError(input.SessionID, input.RequestID, apperrors.NewInternalError(err.Error()))
		}
	}
	writer.Close()
}

// StreamConsultationReply handles POST /visits/:sessionId/consult (SSE)
func (h *WorkbenchHandler) StreamConsultationReply(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[ConsultRequest](c)
	if err != nil {
		apperrors.WriteValidationError(c, "invalid request body")
		return
	}
	input.SessionID = sessionID

	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	writer, err := NewSSEWriter(c)
	if err != nil {
		apperrors.WriteError(c, apperrors.NewInternalError("SSE not supported"))
		return
	}
	err = h.svc.StreamConsultationReply(c.Request.Context(), input.SessionID, input.Content, input.RequestID,
		func(event model.AssistantStreamEvent) error {
			return writer.WriteEvent(event)
		})
	if err != nil {
		var apiErr *apperrors.ApiError
		if errors.As(err, &apiErr) {
			writer.WriteError(input.SessionID, input.RequestID, apiErr)
		} else {
			writer.WriteError(input.SessionID, input.RequestID, apperrors.NewInternalError(err.Error()))
		}
	}
	writer.Close()
}

// GenerateTitle handles POST /visits/:sessionId/generate-title
func (h *WorkbenchHandler) GenerateTitle(c *gin.Context) {
	sessionID := ParseSessionID(c)

	input, err := BindJSON[GenerateTitleRequest](c)
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
	if _, err := h.getSessionAndVerify(c); err != nil {
		return
	}

	title, err := h.svc.GenerateTitle(c.Request.Context(), sessionID)
	if err != nil {
		var apiErr *apperrors.ApiError
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case apperrors.CodeSessionNotFound:
				apperrors.WriteNotFound(c, apperrors.CodeSessionNotFound, apiErr.Message)
			default:
				apperrors.WriteError(c, apiErr)
			}
		} else {
			apperrors.WriteError(c, apperrors.NewInternalError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, GenerateTitleResult{
		SessionID: sessionID,
		Title:     title,
	})
}

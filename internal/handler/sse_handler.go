package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neuhis/software-practice-backend/internal/model"
)

// SSEWriter handles Server-Sent Events streaming.
type SSEWriter struct {
	c       *gin.Context
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer for the given context.
func NewSSEWriter(c *gin.Context) (*SSEWriter, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &SSEWriter{
		c:       c,
		flusher: flusher,
	}, nil
}

// WriteEvent writes a single SSE event.
func (w *SSEWriter) WriteEvent(event model.AssistantStreamEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal sse event: %w", err)
	}

	// SSE format: "data: <JSON>\n\n"
	_, err = fmt.Fprintf(w.c.Writer, "data: %s\n\n", data)
	if err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

// WriteError writes an SSE error event.
func (w *SSEWriter) WriteError(sessionID, requestID string, err error) {
	event := model.AssistantStreamEvent{
		Type:      "error",
		SessionID: sessionID,
		RequestID: requestID,
		Message:   err.Error(),
	}
	_ = w.WriteEvent(event)
}

// Heartbeat sends periodic keep-alive comments to maintain the connection.
func (w *SSEWriter) Heartbeat(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = fmt.Fprintf(w.c.Writer, ": heartbeat\n\n")
			w.flusher.Flush()
		case <-stop:
			return
		case <-w.c.Request.Context().Done():
			return
		}
	}
}

// Close is a no-op kept for compatibility with callers that defer its execution.
func (w *SSEWriter) Close() {
}

// StreamEvents streams a batch of assistant events via SSE, then closes.
func StreamEvents(c *gin.Context, events []model.AssistantStreamEvent) {
	writer, err := NewSSEWriter(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	for _, event := range events {
		if err := writer.WriteEvent(event); err != nil {
			return
		}
	}
}

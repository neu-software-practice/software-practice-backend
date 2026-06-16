// Package response renders the unified API envelope { success, data, error, meta }
// (SPEC §8 / PLAN §4). All handlers funnel success and error paths through here
// so the contract stays consistent and internal error details never leak.
package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
)

// Meta carries pagination info for list endpoints.
type Meta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// ErrorBody is the error payload returned on failure.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Body is the response envelope shared by every endpoint.
type Body struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *ErrorBody  `json:"error"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Success writes a 200 envelope with the given data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Success: true, Data: data})
}

// Created writes a 201 envelope with the given data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{Success: true, Data: data})
}

// List writes a 200 envelope with data plus pagination meta.
func List(c *gin.Context, data interface{}, meta Meta) {
	c.JSON(http.StatusOK, Body{Success: true, Data: data, Meta: &meta})
}

// Error maps an error to the envelope. Known AppErrors are rendered with their
// code/message/status; anything else is logged server-side and returned as a
// generic 500 so implementation details never reach the client (SPEC §7.2).
func Error(c *gin.Context, err error) {
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		c.AbortWithStatusJSON(appErr.Status, Body{
			Success: false,
			Error:   &ErrorBody{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	slog.Error("unhandled error", "path", c.FullPath(), "method", c.Request.Method, "err", err)
	c.AbortWithStatusJSON(apperr.ErrInternal.Status, Body{
		Success: false,
		Error:   &ErrorBody{Code: apperr.ErrInternal.Code, Message: apperr.ErrInternal.Message},
	})
}

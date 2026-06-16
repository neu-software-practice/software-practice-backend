// Package handler holds the Gin HTTP handlers (PLAN §2.1). Handlers bind/validate
// input, delegate to services and render via the unified response envelope.
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

// bindJSON binds and validates the JSON body, writing a 422 on failure.
func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, apperr.ErrValidation.WithMessage("请求参数校验失败: "+err.Error()))
		return false
	}
	return true
}

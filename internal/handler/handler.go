// Package handler holds the Gin HTTP handlers (PLAN §2.1). Handlers bind/validate
// input, delegate to services and render via the unified response envelope.
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
)

// bindJSON binds and validates the JSON body, writing a 422 on failure.
func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, apperr.ErrValidation.WithMessage("请求参数校验失败: "+err.Error()))
		return false
	}
	return true
}

// parsePage reads ?page= & ?limit= with defaults.
func parsePage(c *gin.Context) repository.Page {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	return repository.Page{Page: page, Limit: limit}
}

// parseIDParam parses a positive uint path parameter, writing a 400 on failure.
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, apperr.ErrBadRequest.WithMessage("无效的 "+name+" 参数"))
		return 0, false
	}
	return uint(id), true
}

// parseUintQuery parses a uint query parameter (0 when absent/invalid).
func parseUintQuery(c *gin.Context, name string) uint {
	v, _ := strconv.ParseUint(c.Query(name), 10, 64)
	return uint(v)
}

// metaFor builds pagination meta from the (normalized) page and total count.
func metaFor(page repository.Page, total int64) response.Meta {
	p, l := page.Normalized()
	return response.Meta{Page: p, Limit: l, Total: total}
}

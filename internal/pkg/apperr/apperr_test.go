package apperr_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
)

func TestAppError(t *testing.T) {
	e := apperr.New("CODE", "消息", http.StatusBadRequest)
	assert.Equal(t, "消息", e.Error())
	assert.Equal(t, "CODE", e.Code)
	assert.Equal(t, http.StatusBadRequest, e.Status)
}

func TestWithMessage_IsImmutable(t *testing.T) {
	base := apperr.ErrNotFound
	custom := base.WithMessage("挂号记录不存在")

	assert.Equal(t, "挂号记录不存在", custom.Message)
	assert.Equal(t, base.Code, custom.Code)
	assert.Equal(t, base.Status, custom.Status)
	// The shared sentinel must be untouched.
	assert.Equal(t, "资源不存在", base.Message)
}

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/apperr"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

// RequireDeptType authorizes the route for the listed dept_types (SPEC §3, §7.1).
// The root account is a non-business, read-only observer: it may GET any guarded
// route but is forbidden from mutating requests.
func RequireDeptType(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}

	return func(c *gin.Context) {
		claims, ok := CurrentClaims(c)
		if !ok {
			response.Error(c, apperr.ErrUnauthorized)
			return
		}

		if claims.DeptType == constant.DeptTypeRoot {
			if isReadOnly(c.Request.Method) {
				c.Next()
				return
			}
			response.Error(c, apperr.ErrForbidden)
			return
		}

		if _, allowed := allowedSet[claims.DeptType]; !allowed {
			response.Error(c, apperr.ErrForbidden)
			return
		}
		c.Next()
	}
}

func isReadOnly(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

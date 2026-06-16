package test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
)

// TestRBAC_CrossRoleForbidden verifies dept_type isolation (SPEC §3): a role may
// only reach its own endpoints.
func TestRBAC_CrossRoleForbidden(t *testing.T) {
	engine, _ := newServer(t)
	doctor := login(t, engine, "doctor")
	finance := login(t, engine, "finance")
	checker := login(t, engine, "checker")

	cases := []struct {
		name, method, path, token string
		body                      interface{}
		want                      int
	}{
		{"doctor cannot register", http.MethodPost, "/api/registers", doctor, dto.RegisterRequest{RealName: "x", Gender: "男", DeptmentID: 1, EmployeeID: 1, RegistLevelID: 1, SettleCategoryID: 1}, http.StatusForbidden},
		{"finance cannot consult", http.MethodPost, "/api/physician/registers/1/consult", finance, nil, http.StatusForbidden},
		{"checker cannot charge", http.MethodPost, "/api/charges", checker, dto.ChargeRequest{CaseNumber: "X", Items: []dto.ChargeItemRef{{ItemType: "check", ID: 1}}}, http.StatusForbidden},
		{"finance cannot dispense", http.MethodPost, "/api/pharmacy/prescriptions/1/dispense", finance, nil, http.StatusForbidden},
		{"doctor cannot accept checks", http.MethodGet, "/api/check/pending", doctor, nil, http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec, _ := doJSON(t, engine, c.method, c.path, c.token, c.body)
			assert.Equal(t, c.want, rec.Code)
		})
	}
}

// TestRBAC_RootIsReadOnly verifies the root observer: GET allowed, writes 403.
func TestRBAC_RootIsReadOnly(t *testing.T) {
	engine, _ := newServer(t)
	root := login(t, engine, "root")

	rec, _ := doJSON(t, engine, http.MethodGet, "/api/physician/patients", root, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "root may read any guarded route")

	rec, _ = doJSON(t, engine, http.MethodPost, "/api/registers", root,
		dto.RegisterRequest{RealName: "x", Gender: "男", DeptmentID: 1, EmployeeID: 1, RegistLevelID: 1, SettleCategoryID: 1})
	assert.Equal(t, http.StatusForbidden, rec.Code, "root must not mutate")
}

// TestRBAC_Unauthenticated verifies guarded routes reject anonymous access.
func TestRBAC_Unauthenticated(t *testing.T) {
	engine, _ := newServer(t)
	rec, _ := doJSON(t, engine, http.MethodGet, "/api/departments", "", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

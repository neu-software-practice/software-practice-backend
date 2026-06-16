package test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
)

// registerPatient registers a fresh patient and returns its case number + id.
func registerPatient(t *testing.T, engine *gin.Engine, finance, doctor string) (string, uint, dto.UserInfo) {
	t.Helper()
	_, env := doJSON(t, engine, http.MethodGet, "/api/auth/me", doctor, nil)
	var doc dto.UserInfo
	decodeData(t, env, &doc)

	levelID := firstID(t, engine, finance, "/api/regist-levels")
	settleID := firstID(t, engine, finance, "/api/settle-categories")
	rec, env := doJSON(t, engine, http.MethodPost, "/api/registers", finance, dto.RegisterRequest{
		RealName: "测试", Gender: "女", DeptmentID: doc.DeptID, EmployeeID: doc.ID, RegistLevelID: levelID, SettleCategoryID: settleID,
	})
	require.Equalf(t, http.StatusCreated, rec.Code, "register: %s", rec.Body.String())
	var reg dto.RegisterBrief
	decodeData(t, env, &reg)
	return reg.CaseNumber, reg.ID, doc
}

func TestRegistrationErrors(t *testing.T) {
	engine, _ := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	_, env := doJSON(t, engine, http.MethodGet, "/api/auth/me", doctor, nil)
	var doc dto.UserInfo
	decodeData(t, env, &doc)
	levelID := firstID(t, engine, finance, "/api/regist-levels")
	settleID := firstID(t, engine, finance, "/api/settle-categories")
	financeDeptID := firstID(t, engine, finance, "/api/departments?type="+url.QueryEscape(constant.DeptTypeFinance))

	t.Run("missing required fields", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/registers", finance, dto.RegisterRequest{RealName: "x"})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
	t.Run("unknown level", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/registers", finance, dto.RegisterRequest{
			RealName: "x", Gender: "男", DeptmentID: doc.DeptID, EmployeeID: doc.ID, RegistLevelID: 99999, SettleCategoryID: settleID})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("doctor not in department", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/registers", finance, dto.RegisterRequest{
			RealName: "x", Gender: "男", DeptmentID: financeDeptID, EmployeeID: doc.ID, RegistLevelID: levelID, SettleCategoryID: settleID})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("bad birthdate", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/registers", finance, dto.RegisterRequest{
			RealName: "x", Gender: "男", Birthdate: "not-a-date", DeptmentID: doc.DeptID, EmployeeID: doc.ID, RegistLevelID: levelID, SettleCategoryID: settleID})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
}

func TestPhysicianErrors(t *testing.T) {
	engine, _ := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	_, regID, _ := registerPatient(t, engine, finance, doctor)

	t.Run("bad id param", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/physician/registers/abc/consult", doctor, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("consult missing register", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/physician/registers/99999/consult", doctor, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("save record before consult", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPut, p("/api/physician/registers/%d/medical-record", regID), doctor, dto.MedicalRecordRequest{Readme: "x"})
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	// Now consult, then double-consult must conflict.
	rec, _ := doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/consult", regID), doctor, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/consult", regID), doctor, nil)
	assert.Equal(t, http.StatusConflict, rec.Code)

	t.Run("diagnose validation", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPut, p("/api/physician/registers/%d/diagnosis", regID), doctor, dto.DiagnoseRequest{})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
	t.Run("prescription unknown drug", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/prescriptions", regID), doctor, dto.PrescriptionRequest{
			Items: []dto.PrescriptionItemInput{{DrugID: 99999, DrugUsage: "x", DrugNumber: 1}}})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestRequestAndChargeErrors(t *testing.T) {
	engine, _ := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	checker := login(t, engine, "checker")
	caseNumber, regID, _ := registerPatient(t, engine, finance, doctor)
	rec, _ := doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/consult", regID), doctor, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	inspTechID := firstID(t, engine, doctor, "/api/medical-technologies?type="+url.QueryEscape(constant.TechTypeInspection))

	t.Run("tech type mismatch", func(t *testing.T) {
		// Posting an inspection project to the check endpoint must be rejected.
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/check-requests", doctor, dto.CreateRequestInput{RegisterID: regID, TechID: inspTechID})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
	t.Run("create request missing register", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/inspection-requests", doctor, dto.CreateRequestInput{RegisterID: 99999, TechID: inspTechID})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("execute missing request", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/check-requests/99999/execute", checker, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("charge pending missing case_number", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodGet, "/api/charges/pending", finance, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("charge pending unknown case", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodGet, "/api/charges/pending?case_number=NOPE", finance, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("charge empty items", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: caseNumber, Items: []dto.ChargeItemRef{}})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
	t.Run("charge unknown item type", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: caseNumber, Items: []dto.ChargeItemRef{{ItemType: "bogus", ID: 1}}})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestPharmacyErrors(t *testing.T) {
	engine, _ := newServer(t)
	pharmacist := login(t, engine, "pharmacist")

	t.Run("missing case_number", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodGet, "/api/pharmacy/prescriptions", pharmacist, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("dispense missing prescription", func(t *testing.T) {
		rec, _ := doJSON(t, engine, http.MethodPost, "/api/pharmacy/prescriptions/99999/dispense", pharmacist, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

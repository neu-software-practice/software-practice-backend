package test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
)

// firstID fetches a list endpoint and returns the first row's id.
func firstID(t *testing.T, engine *gin.Engine, token, path string) uint {
	t.Helper()
	rec, env := doJSON(t, engine, http.MethodGet, path, token, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "GET %s -> %s", path, rec.Body.String())
	var rows []struct {
		ID uint `json:"id"`
	}
	decodeData(t, env, &rows)
	require.NotEmptyf(t, rows, "expected at least one row at %s", path)
	return rows[0].ID
}

func p(format string, args ...interface{}) string { return fmt.Sprintf(format, args...) }

func refs(items []dto.PendingItem) []dto.ChargeItemRef {
	out := make([]dto.ChargeItemRef, 0, len(items))
	for _, it := range items {
		out = append(out, dto.ChargeItemRef{ItemType: it.ItemType, ID: it.ID})
	}
	return out
}

func containsRegister(rows []dto.RegisterBrief, id uint) bool {
	for _, r := range rows {
		if r.ID == id {
			return true
		}
	}
	return false
}

func pendingItems(t *testing.T, engine *gin.Engine, token, caseNumber string) dto.PendingItemsResponse {
	t.Helper()
	_, env := doJSON(t, engine, http.MethodGet, "/api/charges/pending?case_number="+caseNumber, token, nil)
	var resp dto.PendingItemsResponse
	decodeData(t, env, &resp)
	return resp
}

// runTechFlow performs 受理 → 执行 → 结果 for one request family (check/inspection).
func runTechFlow(t *testing.T, engine *gin.Engine, token, prefix string, registerID uint) {
	t.Helper()

	rec, env := doJSON(t, engine, http.MethodGet, p("/api/%s/pending", prefix), token, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "%s pending: %s", prefix, rec.Body.String())
	var pendingPatients []dto.RegisterBrief
	decodeData(t, env, &pendingPatients)
	require.Truef(t, containsRegister(pendingPatients, registerID), "register must be pending for %s", prefix)

	_, env = doJSON(t, engine, http.MethodGet, p("/api/%s-requests?register_id=%d", prefix, registerID), token, nil)
	var paid []dto.RequestView
	decodeData(t, env, &paid)
	require.Len(t, paid, 1)
	reqID := paid[0].ID

	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/%s-requests/%d/execute", prefix, reqID), token, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "%s execute", prefix)

	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/%s-requests/%d/result", prefix, reqID), token, dto.ResultRequestInput{Result: "未见明显异常"})
	require.Equalf(t, http.StatusOK, rec.Code, "%s result", prefix)
}

// TestMainLineFlow drives the entire 挂号→看病→缴费→发药 closed loop end-to-end
// across all six roles (SPEC §1 本质目标).
func TestMainLineFlow(t *testing.T) {
	engine, db := newServer(t)

	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	checker := login(t, engine, "checker")
	inspector := login(t, engine, "inspector")
	pharmacist := login(t, engine, "pharmacist")

	_, env := doJSON(t, engine, http.MethodGet, "/api/auth/me", doctor, nil)
	var doc dto.UserInfo
	decodeData(t, env, &doc)

	levelID := firstID(t, engine, finance, "/api/regist-levels")
	settleID := firstID(t, engine, finance, "/api/settle-categories")
	checkTechID := firstID(t, engine, doctor, "/api/medical-technologies?type="+url.QueryEscape(constant.TechTypeCheck))
	inspTechID := firstID(t, engine, doctor, "/api/medical-technologies?type="+url.QueryEscape(constant.TechTypeInspection))
	diseaseID := firstID(t, engine, doctor, "/api/diseases")

	_, env = doJSON(t, engine, http.MethodGet, "/api/drugs", doctor, nil)
	var drugs []model.DrugInfo
	decodeData(t, env, &drugs)
	require.NotEmpty(t, drugs)
	drug := drugs[0]

	// --- F1-1 窗口挂号 ---
	rec, env := doJSON(t, engine, http.MethodPost, "/api/registers", finance, dto.RegisterRequest{
		RealName: "李雷", Gender: "男", Birthdate: "1990-01-01", Age: 36,
		DeptmentID: doc.DeptID, EmployeeID: doc.ID, RegistLevelID: levelID, SettleCategoryID: settleID,
		RegistMethod: "现金",
	})
	require.Equalf(t, http.StatusCreated, rec.Code, "register: %s", rec.Body.String())
	var reg dto.RegisterBrief
	decodeData(t, env, &reg)
	require.NotEmpty(t, reg.CaseNumber)
	assert.Equal(t, constant.VisitStateRegistered, reg.VisitState)
	regID := reg.ID

	// --- F2-1 创建病历 (医生接诊) ---
	rec, env = doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/consult", regID), doctor, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "consult: %s", rec.Body.String())
	var consulted dto.RegisterBrief
	decodeData(t, env, &consulted)
	assert.Equal(t, constant.VisitStateInConsult, consulted.VisitState)

	// --- F2-2 病历首页 ---
	rec, _ = doJSON(t, engine, http.MethodPut, p("/api/physician/registers/%d/medical-record", regID), doctor, dto.MedicalRecordRequest{
		Readme: "咳嗽三天", Present: "无发热", DiseaseIDs: []uint{diseaseID},
	})
	require.Equal(t, http.StatusOK, rec.Code)

	// --- F2-3 检查申请 / F2-4 检验申请 ---
	rec, _ = doJSON(t, engine, http.MethodPost, "/api/check-requests", doctor, dto.CreateRequestInput{RegisterID: regID, TechID: checkTechID, Info: "排查肺炎", Position: "胸部"})
	require.Equalf(t, http.StatusCreated, rec.Code, "check req: %s", rec.Body.String())
	rec, _ = doJSON(t, engine, http.MethodPost, "/api/inspection-requests", doctor, dto.CreateRequestInput{RegisterID: regID, TechID: inspTechID})
	require.Equalf(t, http.StatusCreated, rec.Code, "insp req: %s", rec.Body.String())

	// --- F1-3 收费 (检查 + 检验) ---
	pend := pendingItems(t, engine, finance, reg.CaseNumber)
	require.Len(t, pend.Items, 2)
	assert.Greater(t, pend.Total, 0.0)
	rec, env = doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: reg.CaseNumber, Items: refs(pend.Items)})
	require.Equalf(t, http.StatusOK, rec.Code, "charge: %s", rec.Body.String())
	var charge dto.ChargeResult
	decodeData(t, env, &charge)
	assert.Equal(t, 2, charge.Count)

	// --- F3 检查 / F4 检验: 受理 → 执行 → 结果 ---
	runTechFlow(t, engine, checker, "check", regID)
	runTechFlow(t, engine, inspector, "inspection", regID)

	// --- F2-6/F2-7 医生查看结果 ---
	_, env = doJSON(t, engine, http.MethodGet, p("/api/check-requests/results?register_id=%d", regID), doctor, nil)
	var checkResults []dto.RequestView
	decodeData(t, env, &checkResults)
	require.Len(t, checkResults, 1)
	assert.Equal(t, constant.RequestStateCompleted, checkResults[0].State)
	assert.NotEmpty(t, checkResults[0].Result)

	// --- F2-8 门诊确诊 ---
	rec, _ = doJSON(t, engine, http.MethodPut, p("/api/physician/registers/%d/diagnosis", regID), doctor, dto.DiagnoseRequest{Diagnosis: "急性上呼吸道感染", Cure: "多休息多饮水"})
	require.Equal(t, http.StatusOK, rec.Code)

	// --- F2-9 开立处方 ---
	rec, env = doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/prescriptions", regID), doctor, dto.PrescriptionRequest{
		Items: []dto.PrescriptionItemInput{{DrugID: drug.ID, DrugUsage: "口服 一日三次", DrugNumber: 2}},
	})
	require.Equalf(t, http.StatusCreated, rec.Code, "prescription: %s", rec.Body.String())
	var presResult dto.PrescriptionResult
	decodeData(t, env, &presResult)
	assert.Equal(t, 1, presResult.Count)
	assert.InDelta(t, drug.DrugPrice*2, presResult.Total, 0.001)

	// --- F1-3 收费 (处方) ---
	pend = pendingItems(t, engine, finance, reg.CaseNumber)
	require.Len(t, pend.Items, 1)
	assert.Equal(t, constant.ChargeItemPrescription, pend.Items[0].ItemType)
	rec, _ = doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: reg.CaseNumber, Items: refs(pend.Items)})
	require.Equal(t, http.StatusOK, rec.Code)

	// --- F5-1 药房发药 ---
	_, env = doJSON(t, engine, http.MethodGet, "/api/pharmacy/prescriptions?case_number="+reg.CaseNumber, pharmacist, nil)
	var dispense dto.DispenseList
	decodeData(t, env, &dispense)
	require.Len(t, dispense.Items, 1)
	prescriptionID := dispense.Items[0].ID
	assert.Equal(t, constant.PrescriptionStatePaid, dispense.Items[0].DrugState)

	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/pharmacy/prescriptions/%d/dispense", prescriptionID), pharmacist, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "dispense: %s", rec.Body.String())

	// --- 最终状态核验 ---
	var pres model.Prescription
	require.NoError(t, db.First(&pres, prescriptionID).Error)
	assert.Equal(t, constant.PrescriptionStateDispensed, pres.DrugState)

	var refreshed model.DrugInfo
	require.NoError(t, db.First(&refreshed, drug.ID).Error)
	assert.Equal(t, drug.DrugStock-2, refreshed.DrugStock, "stock must drop by dispensed quantity")

	var chargeRows int64
	require.NoError(t, db.Model(&model.ChargeRecord{}).Where("register_id = ?", regID).Count(&chargeRows).Error)
	assert.EqualValues(t, 3, chargeRows, "check + inspection + prescription = 3 ledger rows")

	var txnRows int64
	require.NoError(t, db.Model(&model.DrugTransaction{}).Where("register_id = ? AND action = ?", regID, "发药").Count(&txnRows).Error)
	assert.EqualValues(t, 1, txnRows)
}

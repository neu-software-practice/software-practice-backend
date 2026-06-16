package test

import (
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

func consult(t *testing.T, engine *gin.Engine, doctor string, regID uint) {
	t.Helper()
	rec, _ := doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/consult", regID), doctor, nil)
	require.Equal(t, http.StatusOK, rec.Code)
}

// paidCheckRequest registers a patient, opens a check order and charges it,
// returning the case number, register id and the now-paid check-request id.
func paidCheckRequest(t *testing.T, engine *gin.Engine, finance, doctor string) (string, uint, uint) {
	t.Helper()
	caseNumber, regID, _ := registerPatient(t, engine, finance, doctor)
	consult(t, engine, doctor, regID)
	techID := firstID(t, engine, doctor, "/api/medical-technologies?type="+url.QueryEscape(constant.TechTypeCheck))
	rec, _ := doJSON(t, engine, http.MethodPost, "/api/check-requests", doctor, dto.CreateRequestInput{RegisterID: regID, TechID: techID})
	require.Equal(t, http.StatusCreated, rec.Code)

	pend := pendingItems(t, engine, finance, caseNumber)
	require.Len(t, pend.Items, 1)
	checkReqID := pend.Items[0].ID
	rec, _ = doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: caseNumber, Items: refs(pend.Items)})
	require.Equal(t, http.StatusOK, rec.Code)
	return caseNumber, regID, checkReqID
}

// TestRegistrationCancelAndList covers F1-2 退号 + 挂号记录查询.
func TestRegistrationCancelAndList(t *testing.T) {
	engine, db := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	caseNumber, regID, _ := registerPatient(t, engine, finance, doctor)

	rec, _ := doJSON(t, engine, http.MethodPost, p("/api/registers/%d/cancel", regID), finance, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "cancel: %s", rec.Body.String())

	var reg model.Register
	require.NoError(t, db.First(&reg, regID).Error)
	assert.Equal(t, constant.VisitStateCanceled, reg.VisitState)

	var refunds int64
	db.Model(&model.ChargeRecord{}).Where("register_id = ? AND action = ?", regID, constant.ChargeActionRefund).Count(&refunds)
	assert.EqualValues(t, 1, refunds, "退号必须退挂号费")

	// Cancelling again must conflict.
	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/registers/%d/cancel", regID), finance, nil)
	assert.Equal(t, http.StatusConflict, rec.Code)

	// Registration record query returns the visit.
	_, env := doJSON(t, engine, http.MethodGet, "/api/registers?case_number="+caseNumber, finance, nil)
	var rows []dto.RegisterBrief
	decodeData(t, env, &rows)
	require.Len(t, rows, 1)
	assert.Equal(t, regID, rows[0].ID)
}

// TestRefundAndChargeRecords covers F1-4 退费 + F1-5 费用记录查询.
func TestRefundAndChargeRecords(t *testing.T) {
	engine, db := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	caseNumber, regID, checkReqID := paidCheckRequest(t, engine, finance, doctor)

	// Refundable list shows the paid check item.
	_, env := doJSON(t, engine, http.MethodGet, "/api/charges/refund-pending?case_number="+caseNumber, finance, nil)
	var refundable dto.PendingItemsResponse
	decodeData(t, env, &refundable)
	require.Len(t, refundable.Items, 1)

	rec, _ := doJSON(t, engine, http.MethodPost, "/api/charges/refund", finance, dto.RefundRequest{
		CaseNumber: caseNumber, Items: []dto.ChargeItemRef{{ItemType: constant.ChargeItemCheck, ID: checkReqID}},
	})
	require.Equalf(t, http.StatusOK, rec.Code, "refund: %s", rec.Body.String())

	var cr model.CheckRequest
	require.NoError(t, db.First(&cr, checkReqID).Error)
	assert.Equal(t, constant.RequestStateRefunded, cr.CheckState)

	// Ledger query (F1-5) shows both 收费 (挂号 + check) and 退费 rows.
	_, env = doJSON(t, engine, http.MethodGet, "/api/charge-records?case_number="+caseNumber, finance, nil)
	var records []model.ChargeRecord
	decodeData(t, env, &records)
	assert.GreaterOrEqual(t, len(records), 3)

	_, env = doJSON(t, engine, http.MethodGet, "/api/charge-records?register_id="+p("%d", regID)+"&action="+url.QueryEscape(constant.ChargeActionRefund), finance, nil)
	var refunds []model.ChargeRecord
	decodeData(t, env, &refunds)
	require.Len(t, refunds, 1)
	assert.Equal(t, constant.ChargeItemCheck, refunds[0].ItemType)
}

// TestDisposalFlow covers F2-10 处置申请 + F6 处置受理/录入/结果 (generic path).
func TestDisposalFlow(t *testing.T) {
	engine, db := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	disposer := login(t, engine, "disposer")

	caseNumber, regID, _ := registerPatient(t, engine, finance, doctor)
	consult(t, engine, doctor, regID)
	dispTechID := firstID(t, engine, doctor, "/api/medical-technologies?type="+url.QueryEscape(constant.TechTypeDisposal))

	rec, _ := doJSON(t, engine, http.MethodPost, "/api/disposal-requests", doctor, dto.CreateRequestInput{RegisterID: regID, TechID: dispTechID, Info: "清创"})
	require.Equalf(t, http.StatusCreated, rec.Code, "disposal req: %s", rec.Body.String())

	pend := pendingItems(t, engine, finance, caseNumber)
	require.Len(t, pend.Items, 1)
	assert.Equal(t, constant.ChargeItemDisposal, pend.Items[0].ItemType)
	rec, _ = doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: caseNumber, Items: refs(pend.Items)})
	require.Equal(t, http.StatusOK, rec.Code)

	runTechFlow(t, engine, disposer, "disposal", regID)

	var dr model.DisposalRequest
	require.NoError(t, db.Where("register_id = ?", regID).First(&dr).Error)
	assert.Equal(t, constant.RequestStateCompleted, dr.DisposalState)
	assert.NotEmpty(t, dr.DisposalResult)

	// 管理 (F6-4) lists the patient's disposal requests.
	_, env := doJSON(t, engine, http.MethodGet, p("/api/disposal-requests/manage?register_id=%d", regID), disposer, nil)
	var manage []dto.RequestView
	decodeData(t, env, &manage)
	require.Len(t, manage, 1)
}

// TestDrugAdmin covers F5-3 药库管理 (增改删 + 入库).
func TestDrugAdmin(t *testing.T) {
	engine, db := newServer(t)
	pharmacist := login(t, engine, "pharmacist")

	rec, env := doJSON(t, engine, http.MethodPost, "/api/pharmacy/drugs", pharmacist, dto.DrugRequest{
		DrugCode: "NEW001", DrugName: "新药测试", DrugUnit: "盒", DrugPrice: 9.9, DrugStock: 10, MnemonicCode: "XYCS",
	})
	require.Equalf(t, http.StatusCreated, rec.Code, "create drug: %s", rec.Body.String())
	var created model.DrugInfo
	decodeData(t, env, &created)
	require.NotZero(t, created.ID)

	rec, _ = doJSON(t, engine, http.MethodPut, p("/api/pharmacy/drugs/%d", created.ID), pharmacist, dto.DrugRequest{
		DrugCode: "NEW001", DrugName: "新药测试改", DrugPrice: 12.0, DrugStock: 10,
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec, env = doJSON(t, engine, http.MethodPost, p("/api/pharmacy/drugs/%d/restock", created.ID), pharmacist, dto.StockRequest{Delta: 40})
	require.Equal(t, http.StatusOK, rec.Code)
	var restocked model.DrugInfo
	decodeData(t, env, &restocked)
	assert.Equal(t, 50, restocked.DrugStock)

	rec, _ = doJSON(t, engine, http.MethodDelete, p("/api/pharmacy/drugs/%d", created.ID), pharmacist, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var gone model.DrugInfo
	require.NoError(t, db.First(&gone, created.ID).Error)
	assert.Equal(t, constant.DelmarkDeleted, gone.Delmark)
}

// TestDispenseRefundAndTransactions covers F5-2 退药 + F5-4 交易记录.
func TestDispenseRefundAndTransactions(t *testing.T) {
	engine, db := newServer(t)
	finance := login(t, engine, "finance")
	doctor := login(t, engine, "doctor")
	pharmacist := login(t, engine, "pharmacist")

	caseNumber, regID, _ := registerPatient(t, engine, finance, doctor)
	consult(t, engine, doctor, regID)

	_, env := doJSON(t, engine, http.MethodGet, "/api/drugs", doctor, nil)
	var drugs []model.DrugInfo
	decodeData(t, env, &drugs)
	require.NotEmpty(t, drugs)
	drug := drugs[0]

	rec, _ := doJSON(t, engine, http.MethodPost, p("/api/physician/registers/%d/prescriptions", regID), doctor, dto.PrescriptionRequest{
		Items: []dto.PrescriptionItemInput{{DrugID: drug.ID, DrugUsage: "口服", DrugNumber: 3}},
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	pend := pendingItems(t, engine, finance, caseNumber)
	require.Len(t, pend.Items, 1)
	doJSON(t, engine, http.MethodPost, "/api/charges", finance, dto.ChargeRequest{CaseNumber: caseNumber, Items: refs(pend.Items)})

	_, env = doJSON(t, engine, http.MethodGet, "/api/pharmacy/prescriptions?case_number="+caseNumber, pharmacist, nil)
	var list dto.DispenseList
	decodeData(t, env, &list)
	require.Len(t, list.Items, 1)
	presID := list.Items[0].ID

	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/pharmacy/prescriptions/%d/dispense", presID), pharmacist, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// 退药 (F5-2): 已发药 → 已退药, stock restored.
	rec, _ = doJSON(t, engine, http.MethodPost, p("/api/pharmacy/prescriptions/%d/refund", presID), pharmacist, nil)
	require.Equalf(t, http.StatusOK, rec.Code, "refund medicine: %s", rec.Body.String())

	var refreshed model.DrugInfo
	require.NoError(t, db.First(&refreshed, drug.ID).Error)
	assert.Equal(t, drug.DrugStock, refreshed.DrugStock, "退药后库存恢复")

	// 交易记录 (F5-4): one 发药 + one 退药.
	_, env = doJSON(t, engine, http.MethodGet, "/api/pharmacy/transactions?case_number="+caseNumber, pharmacist, nil)
	var txns []model.DrugTransaction
	decodeData(t, env, &txns)
	assert.Len(t, txns, 2)
}

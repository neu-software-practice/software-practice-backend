package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// TestMedTechRequest_Accessors verifies the generic interface implementation for
// all three isomorphic request types in one table-driven pass.
func TestMedTechRequest_Accessors(t *testing.T) {
	now := time.Now()
	tech := &model.MedicalTechnology{TechName: "胸部CT"}

	cases := []struct {
		name     string
		stateCol string
		table    string
		req      model.MedTechRequest
	}{
		{"check", "check_state", "check_request", &model.CheckRequest{
			ID: 1, RegisterID: 2, MedicalTechnologyID: 3, CheckInfo: "i", CheckPosition: "p",
			CheckRemark: "r", CheckState: "已开立", CreationTime: now, MedicalTechnology: tech,
		}},
		{"inspection", "inspection_state", "inspection_request", &model.InspectionRequest{
			ID: 1, RegisterID: 2, MedicalTechnologyID: 3, InspectionInfo: "i", InspectionPosition: "p",
			InspectionRemark: "r", InspectionState: "已开立", CreationTime: now, MedicalTechnology: tech,
		}},
		{"disposal", "disposal_state", "disposal_request", &model.DisposalRequest{
			ID: 1, RegisterID: 2, MedicalTechnologyID: 3, DisposalInfo: "i", DisposalPosition: "p",
			DisposalRemark: "r", DisposalState: "已开立", CreationTime: now, MedicalTechnology: tech,
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.req
			assert.EqualValues(t, 1, r.RequestID())
			assert.EqualValues(t, 2, r.RequestRegisterID())
			assert.EqualValues(t, 3, r.RequestTechID())
			assert.Equal(t, "已开立", r.State())
			assert.Equal(t, c.stateCol, r.StateColumn())
			assert.Equal(t, c.table, r.TableName())
			assert.Equal(t, "i", r.Info())
			assert.Equal(t, "p", r.Position())
			assert.Equal(t, "r", r.Remark())
			require.NotNil(t, r.GetMedicalTechnology())
			assert.Equal(t, "胸部CT", r.GetMedicalTechnology().TechName)
			assert.False(t, r.GetCreationTime().IsZero())
			assert.Nil(t, r.GetExecutorID())
			assert.Nil(t, r.GetInputterID())
			assert.Nil(t, r.GetResultTime())
			assert.Empty(t, r.Result())

			// Mutating setters (idiomatic pointer-receiver state machine).
			r.SetState("已缴费")
			assert.Equal(t, "已缴费", r.State())

			r.AssignExecutor(42)
			require.NotNil(t, r.GetExecutorID())
			assert.EqualValues(t, 42, *r.GetExecutorID())

			at := time.Now().Add(time.Hour)
			r.RecordResult("结果正常", 7, at)
			assert.Equal(t, "结果正常", r.Result())
			require.NotNil(t, r.GetInputterID())
			assert.EqualValues(t, 7, *r.GetInputterID())
			require.NotNil(t, r.GetResultTime())

			r.SetCreation(at)
			assert.Equal(t, at, r.GetCreationTime())
		})
	}
}

func TestModel_TableNames(t *testing.T) {
	names := map[string]string{
		model.Department{}.TableName():           "department",
		model.Employee{}.TableName():             "employee",
		model.Register{}.TableName():             "register",
		model.RegistLevel{}.TableName():          "regist_level",
		model.SettleCategory{}.TableName():       "settle_category",
		model.Scheduling{}.TableName():           "scheduling",
		model.MedicalTechnology{}.TableName():    "medical_technology",
		model.MedicalRecord{}.TableName():        "medical_record",
		model.MedicalRecordDisease{}.TableName(): "medical_record_disease",
		model.Disease{}.TableName():              "disease",
		model.Prescription{}.TableName():         "prescription",
		model.DrugInfo{}.TableName():             "drug_info",
		model.ChargeRecord{}.TableName():         "charge_record",
		model.DrugTransaction{}.TableName():      "drug_transaction",
	}
	for got, want := range names {
		assert.Equal(t, want, got)
	}
	assert.Len(t, model.All(), 17)
}

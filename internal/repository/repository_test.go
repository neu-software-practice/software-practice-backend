package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"
	"github.com/neu-software-practice/software-practice-backend/internal/testutil"
)

func ctx() context.Context { return context.Background() }

// seedFixtures inserts a minimal, connected graph used across repository tests.
func seedFixtures(t *testing.T, db *gorm.DB) (dept model.Department, doctor model.Employee, level model.RegistLevel) {
	t.Helper()
	dept = model.Department{DeptCode: "MZ", DeptName: "门诊内科", DeptType: constant.DeptTypeOutpatient, Delmark: 1}
	require.NoError(t, db.Create(&dept).Error)
	level = model.RegistLevel{RegistCode: "PT", RegistName: "普通号", RegistFee: 10, SequenceNo: 1, Delmark: 1}
	require.NoError(t, db.Create(&level).Error)
	doctor = model.Employee{Username: "doc", Password: "x", Realname: "王医生", DeptmentID: dept.ID, RegistLevelID: &level.ID, Delmark: 1}
	require.NoError(t, db.Create(&doctor).Error)
	return dept, doctor, level
}

func newRegister(t *testing.T, db *gorm.DB, repo repository.RegisterRepository, doctor model.Employee, dept model.Department, level model.RegistLevel, caseNo string, state int) model.Register {
	t.Helper()
	bd := time.Now().AddDate(-30, 0, 0)
	reg := model.Register{
		CaseNumber: caseNo, RealName: "张三", Gender: "男", Birthdate: &bd,
		Age: 30, AgeType: "年", VisitDate: time.Now(), Noon: constant.NoonMorning,
		DeptmentID: dept.ID, EmployeeID: doctor.ID, RegistLevelID: level.ID, SettleCategoryID: 1,
		RegistMoney: level.RegistFee, VisitState: state,
	}
	require.NoError(t, repo.Create(ctx(), &reg))
	return reg
}

func TestPageNormalized(t *testing.T) {
	cases := []struct {
		in              repository.Page
		wantPage, wantL int
	}{
		{repository.Page{}, 1, 10},
		{repository.Page{Page: -5, Limit: 0}, 1, 10},
		{repository.Page{Page: 3, Limit: 25}, 3, 25},
		{repository.Page{Page: 1, Limit: 1000}, 1, 100},
	}
	for _, c := range cases {
		p, l := c.in.Normalized()
		assert.Equal(t, c.wantPage, p)
		assert.Equal(t, c.wantL, l)
	}
}

func TestEmployeeRepository(t *testing.T) {
	db := testutil.NewDB(t)
	dept, doctor, _ := seedFixtures(t, db)
	repo := repository.NewEmployeeRepository(db)

	got, err := repo.FindByUsername(ctx(), "doc")
	require.NoError(t, err)
	assert.Equal(t, doctor.ID, got.ID)
	require.NotNil(t, got.Department)
	assert.Equal(t, dept.DeptType, got.Department.DeptType)

	got, err = repo.FindByID(ctx(), doctor.ID)
	require.NoError(t, err)
	assert.Equal(t, "王医生", got.Realname)

	docs, err := repo.ListDoctors(ctx(), dept.ID, *doctor.RegistLevelID)
	require.NoError(t, err)
	assert.Len(t, docs, 1)

	_, err = repo.FindByUsername(ctx(), "ghost")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestLookupRepositories(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, level := seedFixtures(t, db)
	require.NoError(t, db.Create(&model.SettleCategory{SettleCode: "ZF", SettleName: "自费", SequenceNo: 1, Delmark: 1}).Error)

	rl := repository.NewRegistLevelRepository(db)
	levels, err := rl.List(ctx())
	require.NoError(t, err)
	assert.Len(t, levels, 1)
	one, err := rl.FindByID(ctx(), level.ID)
	require.NoError(t, err)
	assert.Equal(t, "普通号", one.RegistName)
	_, err = rl.FindByID(ctx(), 999)
	assert.ErrorIs(t, err, repository.ErrNotFound)

	sc := repository.NewSettleCategoryRepository(db)
	cats, err := sc.List(ctx())
	require.NoError(t, err)
	assert.Len(t, cats, 1)
	_, err = sc.FindByID(ctx(), cats[0].ID)
	require.NoError(t, err)
}

func TestDepartmentRepository(t *testing.T) {
	db := testutil.NewDB(t)
	dept, _, _ := seedFixtures(t, db)
	repo := repository.NewDepartmentRepository(db)

	all, err := repo.List(ctx())
	require.NoError(t, err)
	assert.Len(t, all, 1)

	byType, err := repo.ListByType(ctx(), constant.DeptTypeOutpatient)
	require.NoError(t, err)
	assert.Len(t, byType, 1)

	one, err := repo.FindByID(ctx(), dept.ID)
	require.NoError(t, err)
	assert.Equal(t, "门诊内科", one.DeptName)
}

func TestRegisterRepository(t *testing.T) {
	db := testutil.NewDB(t)
	dept, doctor, level := seedFixtures(t, db)
	repo := repository.NewRegisterRepository(db)

	reg := newRegister(t, db, repo, doctor, dept, level, "MR001", constant.VisitStateRegistered)
	assert.NotZero(t, reg.ID)

	byID, err := repo.FindByID(ctx(), reg.ID)
	require.NoError(t, err)
	require.NotNil(t, byID.Employee)
	assert.Equal(t, "张三", byID.RealName)

	byCase, err := repo.FindByCaseNumber(ctx(), "MR001")
	require.NoError(t, err)
	assert.Equal(t, reg.ID, byCase.ID)

	require.NoError(t, repo.UpdateState(ctx(), reg.ID, constant.VisitStateInConsult))
	byID, _ = repo.FindByID(ctx(), reg.ID)
	assert.Equal(t, constant.VisitStateInConsult, byID.VisitState)

	newRegister(t, db, repo, doctor, dept, level, "MR002", constant.VisitStateRegistered)
	list, total, err := repo.List(ctx(), repository.RegisterFilter{EmployeeID: doctor.ID}, repository.Page{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, list, 2)

	list, total, err = repo.List(ctx(), repository.RegisterFilter{EmployeeID: doctor.ID, CaseNumber: "MR002"}, repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, "MR002", list[0].CaseNumber)

	list, _, err = repo.List(ctx(), repository.RegisterFilter{Name: "张三", States: []int{constant.VisitStateRegistered}}, repository.Page{})
	require.NoError(t, err)
	assert.Len(t, list, 1)

	n, err := repo.CountByState(ctx(), doctor.ID, constant.VisitStateInConsult)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	_, err = repo.FindByCaseNumber(ctx(), "NOPE")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestMedicalTechnologyRepository(t *testing.T) {
	db := testutil.NewDB(t)
	repo := repository.NewMedicalTechnologyRepository(db)
	require.NoError(t, db.Create(&model.MedicalTechnology{TechCode: "CT", TechName: "胸部CT", TechPrice: 200, TechType: constant.TechTypeCheck}).Error)
	require.NoError(t, db.Create(&model.MedicalTechnology{TechCode: "XJ", TechName: "血常规", TechPrice: 35, TechType: constant.TechTypeInspection}).Error)

	rows, total, err := repo.Search(ctx(), "", constant.TechTypeCheck, repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, "胸部CT", rows[0].TechName)

	rows, _, err = repo.Search(ctx(), "血", "", repository.Page{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	one, err := repo.FindByID(ctx(), rows[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "血常规", one.TechName)
}

func TestDiseaseRepository(t *testing.T) {
	db := testutil.NewDB(t)
	repo := repository.NewDiseaseRepository(db)
	require.NoError(t, db.Create(&model.Disease{DiseaseCode: "GM", DiseaseName: "感冒"}).Error)
	require.NoError(t, db.Create(&model.Disease{DiseaseCode: "FY", DiseaseName: "肺炎"}).Error)

	rows, total, err := repo.Search(ctx(), "感", repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)

	all, err := repo.FindByIDs(ctx(), []uint{rows[0].ID})
	require.NoError(t, err)
	assert.Len(t, all, 1)

	empty, err := repo.FindByIDs(ctx(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestDrugInfoRepository(t *testing.T) {
	db := testutil.NewDB(t)
	repo := repository.NewDrugInfoRepository(db)

	drug := &model.DrugInfo{DrugCode: "YP1", DrugName: "阿莫西林", DrugPrice: 18.5, DrugStock: 100, MnemonicCode: "AMXL", Delmark: 1, CreationDate: time.Now()}
	require.NoError(t, repo.Create(ctx(), drug))

	rows, total, err := repo.Search(ctx(), "阿莫", repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Len(t, rows, 1)

	drug.DrugPrice = 20
	require.NoError(t, repo.Update(ctx(), drug))
	got, err := repo.FindByID(ctx(), drug.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 20, got.DrugPrice)

	ok, err := repo.AdjustStock(ctx(), drug.ID, -30)
	require.NoError(t, err)
	assert.True(t, ok)
	got, _ = repo.FindByID(ctx(), drug.ID)
	assert.Equal(t, 70, got.DrugStock)

	ok, err = repo.AdjustStock(ctx(), drug.ID, -1000)
	require.NoError(t, err)
	assert.False(t, ok, "must refuse to oversell")

	ok, err = repo.AdjustStock(ctx(), drug.ID, 50)
	require.NoError(t, err)
	assert.True(t, ok)

	require.NoError(t, repo.SoftDelete(ctx(), drug.ID))
	_, err = repo.FindByID(ctx(), drug.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestMedicalRecordRepository(t *testing.T) {
	db := testutil.NewDB(t)
	dept, doctor, level := seedFixtures(t, db)
	regRepo := repository.NewRegisterRepository(db)
	reg := newRegister(t, db, regRepo, doctor, dept, level, "MR100", constant.VisitStateInConsult)
	d1 := model.Disease{DiseaseCode: "A", DiseaseName: "感冒"}
	d2 := model.Disease{DiseaseCode: "B", DiseaseName: "发热"}
	require.NoError(t, db.Create(&d1).Error)
	require.NoError(t, db.Create(&d2).Error)

	repo := repository.NewMedicalRecordRepository(db)
	rec := &model.MedicalRecord{RegisterID: reg.ID, Readme: "咳嗽三天"}
	require.NoError(t, repo.Upsert(ctx(), rec, []uint{d1.ID, d2.ID}))

	got, err := repo.FindByRegisterID(ctx(), reg.ID)
	require.NoError(t, err)
	assert.Equal(t, "咳嗽三天", got.Readme)
	assert.Len(t, got.Diseases, 2)

	// Upsert again updates in place and replaces disease links.
	rec2 := &model.MedicalRecord{RegisterID: reg.ID, Readme: "咳嗽五天"}
	require.NoError(t, repo.Upsert(ctx(), rec2, []uint{d1.ID}))
	got, err = repo.FindByRegisterID(ctx(), reg.ID)
	require.NoError(t, err)
	assert.Equal(t, "咳嗽五天", got.Readme)
	assert.Len(t, got.Diseases, 1)

	require.NoError(t, repo.UpdateDiagnosis(ctx(), reg.ID, "上呼吸道感染", "多喝水"))
	got, _ = repo.FindByRegisterID(ctx(), reg.ID)
	assert.Equal(t, "上呼吸道感染", got.Diagnosis)
	assert.Equal(t, "多喝水", got.Cure)

	_, err = repo.FindByRegisterID(ctx(), 9999)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPrescriptionRepository(t *testing.T) {
	db := testutil.NewDB(t)
	dept, doctor, level := seedFixtures(t, db)
	regRepo := repository.NewRegisterRepository(db)
	reg := newRegister(t, db, regRepo, doctor, dept, level, "MR200", constant.VisitStateInConsult)
	drug := model.DrugInfo{DrugCode: "YP1", DrugName: "阿莫西林", DrugPrice: 18.5, DrugStock: 100, Delmark: 1, CreationDate: time.Now()}
	require.NoError(t, db.Create(&drug).Error)

	repo := repository.NewPrescriptionRepository(db)
	items := []*model.Prescription{
		{RegisterID: reg.ID, DrugID: drug.ID, DrugUsage: "口服 一次1粒", DrugNumber: 2, CreationTime: time.Now(), DrugState: constant.PrescriptionStateCreated},
	}
	require.NoError(t, repo.CreateBatch(ctx(), items))
	require.NoError(t, repo.CreateBatch(ctx(), nil)) // no-op

	all, err := repo.ListByRegister(ctx(), reg.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.NotNil(t, all[0].Drug)
	assert.Equal(t, "阿莫西林", all[0].Drug.DrugName)

	pending, err := repo.ListByRegisterAndState(ctx(), reg.ID, constant.PrescriptionStateCreated)
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	require.NoError(t, repo.UpdateState(ctx(), all[0].ID, constant.PrescriptionStatePaid))
	one, err := repo.FindByID(ctx(), all[0].ID)
	require.NoError(t, err)
	assert.Equal(t, constant.PrescriptionStatePaid, one.DrugState)
}

func TestChargeAndDrugTransactionRepositories(t *testing.T) {
	db := testutil.NewDB(t)
	charges := repository.NewChargeRecordRepository(db)
	require.NoError(t, charges.Create(ctx(), &model.ChargeRecord{RegisterID: 1, ItemType: constant.ChargeItemCheck, ItemID: 5, ItemName: "CT", Amount: 200, Action: "收费", OperatorID: 9, CreatedAt: time.Now()}))
	require.NoError(t, charges.Create(ctx(), &model.ChargeRecord{RegisterID: 1, ItemType: constant.ChargeItemCheck, ItemID: 5, ItemName: "CT", Amount: 200, Action: "退费", OperatorID: 9, CreatedAt: time.Now()}))

	rows, total, err := charges.List(ctx(), repository.ChargeFilter{RegisterID: 1}, repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, rows, 2)

	_, total, err = charges.List(ctx(), repository.ChargeFilter{RegisterID: 1, Action: "收费"}, repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)

	tx := repository.NewDrugTransactionRepository(db)
	require.NoError(t, tx.Create(ctx(), &model.DrugTransaction{PrescriptionID: 3, RegisterID: 1, DrugID: 2, DrugName: "阿莫西林", Quantity: 2, Action: "发药", OperatorID: 9, CreatedAt: time.Now()}))
	trows, ttotal, err := tx.List(ctx(), repository.DrugTransactionFilter{RegisterID: 1, Action: "发药"}, repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, ttotal)
	assert.Len(t, trows, 1)
}

func TestRequestRepository_Generic(t *testing.T) {
	db := testutil.NewDB(t)
	dept, doctor, level := seedFixtures(t, db)
	regRepo := repository.NewRegisterRepository(db)
	reg := newRegister(t, db, regRepo, doctor, dept, level, "MR300", constant.VisitStateInConsult)
	tech := model.MedicalTechnology{TechCode: "CT", TechName: "胸部CT", TechPrice: 200, TechType: constant.TechTypeCheck}
	require.NoError(t, db.Create(&tech).Error)

	repo := repository.NewRequestRepository[model.CheckRequest, *model.CheckRequest](db)
	req := &model.CheckRequest{RegisterID: reg.ID, MedicalTechnologyID: tech.ID, CheckInfo: "排查肺炎", CreationTime: time.Now(), CheckState: constant.RequestStateCreated}
	require.NoError(t, repo.Create(ctx(), req))
	assert.NotZero(t, req.ID)

	got, err := repo.FindByID(ctx(), req.ID)
	require.NoError(t, err)
	require.NotNil(t, got.MedicalTechnology)
	assert.Equal(t, "胸部CT", got.MedicalTechnology.TechName)

	got.SetState(constant.RequestStatePaid)
	require.NoError(t, repo.Save(ctx(), got))

	byReg, err := repo.ListByRegister(ctx(), reg.ID)
	require.NoError(t, err)
	assert.Len(t, byReg, 1)

	paid, err := repo.ListByRegisterAndState(ctx(), reg.ID, constant.RequestStatePaid)
	require.NoError(t, err)
	assert.Len(t, paid, 1)

	pendingRegs, total, err := repo.ListPendingRegisters(ctx(), constant.RequestStatePaid, repository.RegisterFilter{}, repository.Page{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, reg.ID, pendingRegs[0].ID)

	n, err := repo.CountDistinctRegisters(ctx(), constant.RequestStatePaid)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	_, err = repo.FindByID(ctx(), 9999)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestTxManager_CommitAndRollback(t *testing.T) {
	db := testutil.NewDB(t)
	tm := repository.NewTxManager(db)
	drugs := repository.NewDrugInfoRepository(db)

	// Commit path.
	err := tm.Do(ctx(), func(c context.Context) error {
		return drugs.Create(c, &model.DrugInfo{DrugCode: "T1", DrugName: "药A", Delmark: 1, CreationDate: time.Now()})
	})
	require.NoError(t, err)
	_, total, _ := drugs.Search(ctx(), "药A", repository.Page{})
	assert.EqualValues(t, 1, total)

	// Rollback path: returning an error must undo the insert.
	sentinel := errors.New("boom")
	err = tm.Do(ctx(), func(c context.Context) error {
		require.NoError(t, drugs.Create(c, &model.DrugInfo{DrugCode: "T2", DrugName: "药B", Delmark: 1, CreationDate: time.Now()}))
		return sentinel
	})
	assert.ErrorIs(t, err, sentinel)
	_, total, _ = drugs.Search(ctx(), "药B", repository.Page{})
	assert.EqualValues(t, 0, total, "rollback must remove the row")
}

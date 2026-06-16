// Package seed loads demonstration base data (SPEC §7.3): one account per role,
// departments, registration levels, settlement categories, scheduling, medical
// technology projects, diseases and drugs. It is idempotent — every row is
// inserted via FirstOrCreate keyed on a natural key, so `make seed` can run
// repeatedly without duplicating data.
package seed

import (
	"time"

	"gorm.io/gorm"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/hash"
)

// Run seeds the database. defaultPassword is applied (hashed) to every account.
func Run(db *gorm.DB, defaultPassword string) error {
	pw, err := hash.Password(defaultPassword)
	if err != nil {
		return err
	}

	depts, err := seedDepartments(db)
	if err != nil {
		return err
	}
	levels, err := seedRegistLevels(db)
	if err != nil {
		return err
	}
	if err := seedSettleCategories(db); err != nil {
		return err
	}
	scheds, err := seedScheduling(db)
	if err != nil {
		return err
	}
	if err := seedEmployees(db, pw, depts, levels, scheds); err != nil {
		return err
	}
	if err := seedMedicalTechnologies(db, depts); err != nil {
		return err
	}
	if err := seedDiseases(db); err != nil {
		return err
	}
	return seedDrugs(db)
}

func seedDepartments(db *gorm.DB) (map[string]model.Department, error) {
	rows := []model.Department{
		{DeptCode: "CWS", DeptName: "收费处", DeptType: constant.DeptTypeFinance},
		{DeptCode: "MZNK", DeptName: "门诊内科", DeptType: constant.DeptTypeOutpatient},
		{DeptCode: "FSK", DeptName: "放射检查科", DeptType: constant.DeptTypeCheck},
		{DeptCode: "JYK", DeptName: "检验科", DeptType: constant.DeptTypeInspection},
		{DeptCode: "XYF", DeptName: "西药房", DeptType: constant.DeptTypePharmacy},
		{DeptCode: "CZS", DeptName: "处置室", DeptType: constant.DeptTypeDisposal},
		{DeptCode: "SYS", DeptName: "系统管理", DeptType: constant.DeptTypeRoot},
	}
	out := make(map[string]model.Department, len(rows))
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.Department{DeptCode: r.DeptCode}).
			Attrs(model.Department{DeptName: r.DeptName, DeptType: r.DeptType, Delmark: constant.DelmarkActive}).
			FirstOrCreate(&r).Error; err != nil {
			return nil, err
		}
		out[r.DeptType] = r
	}
	return out, nil
}

func seedRegistLevels(db *gorm.DB) (map[string]model.RegistLevel, error) {
	rows := []model.RegistLevel{
		{RegistCode: "PT", RegistName: "普通号", RegistFee: 10.00, RegistQuota: 100, SequenceNo: 1},
		{RegistCode: "ZJ", RegistName: "专家号", RegistFee: 50.00, RegistQuota: 30, SequenceNo: 2},
		{RegistCode: "ZR", RegistName: "主任号", RegistFee: 100.00, RegistQuota: 10, SequenceNo: 3},
	}
	out := make(map[string]model.RegistLevel, len(rows))
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.RegistLevel{RegistCode: r.RegistCode}).
			Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return nil, err
		}
		out[r.RegistCode] = r
	}
	return out, nil
}

func seedSettleCategories(db *gorm.DB) error {
	rows := []model.SettleCategory{
		{SettleCode: "ZF", SettleName: "自费", SequenceNo: 1},
		{SettleCode: "YB", SettleName: "医保", SequenceNo: 2},
		{SettleCode: "XNH", SettleName: "新农合", SequenceNo: 3},
	}
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.SettleCategory{SettleCode: r.SettleCode}).
			Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedScheduling(db *gorm.DB) (map[string]model.Scheduling, error) {
	rows := []model.Scheduling{
		{RuleName: "全周出诊", WeekRule: "1,2,3,4,5,6,7"},
		{RuleName: "工作日出诊", WeekRule: "1,2,3,4,5"},
	}
	out := make(map[string]model.Scheduling, len(rows))
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.Scheduling{RuleName: r.RuleName}).
			Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return nil, err
		}
		out[r.RuleName] = r
	}
	return out, nil
}

func seedEmployees(db *gorm.DB, pw string, depts map[string]model.Department, levels map[string]model.RegistLevel, scheds map[string]model.Scheduling) error {
	ptLevel := levels["PT"].ID
	sched := scheds["全周出诊"].ID

	type acct struct {
		username string
		realname string
		deptType string
		level    *uint
		sched    *uint
	}
	accts := []acct{
		{"finance", "收费员小财", constant.DeptTypeFinance, nil, nil},
		{"doctor", "门诊王医生", constant.DeptTypeOutpatient, &ptLevel, &sched},
		{"checker", "检查李医生", constant.DeptTypeCheck, nil, nil},
		{"inspector", "检验赵医生", constant.DeptTypeInspection, nil, nil},
		{"pharmacist", "药房孙药师", constant.DeptTypePharmacy, nil, nil},
		{"disposer", "处置周医生", constant.DeptTypeDisposal, nil, nil},
		{"root", "系统管理员", constant.DeptTypeRoot, nil, nil},
	}
	for _, a := range accts {
		emp := model.Employee{Username: a.username}
		if err := db.Where(model.Employee{Username: a.username}).
			Attrs(model.Employee{
				Password:      pw,
				Realname:      a.realname,
				DeptmentID:    depts[a.deptType].ID,
				RegistLevelID: a.level,
				SchedulingID:  a.sched,
				Delmark:       constant.DelmarkActive,
			}).FirstOrCreate(&emp).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedMedicalTechnologies(db *gorm.DB, depts map[string]model.Department) error {
	checkDept := depts[constant.DeptTypeCheck].ID
	inspDept := depts[constant.DeptTypeInspection].ID
	dispDept := depts[constant.DeptTypeDisposal].ID

	rows := []model.MedicalTechnology{
		{TechCode: "CT001", TechName: "胸部CT", TechFormat: "平扫", TechPrice: 220.00, TechType: constant.TechTypeCheck, PriceType: "检查费", DeptmentID: checkDept},
		{TechCode: "BC001", TechName: "腹部B超", TechFormat: "常规", TechPrice: 120.00, TechType: constant.TechTypeCheck, PriceType: "检查费", DeptmentID: checkDept},
		{TechCode: "XS001", TechName: "胸部X光", TechFormat: "正位", TechPrice: 80.00, TechType: constant.TechTypeCheck, PriceType: "检查费", DeptmentID: checkDept},
		{TechCode: "XJ001", TechName: "血常规", TechFormat: "五分类", TechPrice: 35.00, TechType: constant.TechTypeInspection, PriceType: "检验费", DeptmentID: inspDept},
		{TechCode: "XJ002", TechName: "尿常规", TechFormat: "常规", TechPrice: 25.00, TechType: constant.TechTypeInspection, PriceType: "检验费", DeptmentID: inspDept},
		{TechCode: "XJ003", TechName: "肝功能", TechFormat: "全套", TechPrice: 90.00, TechType: constant.TechTypeInspection, PriceType: "检验费", DeptmentID: inspDept},
		{TechCode: "CZ001", TechName: "清创缝合", TechFormat: "小", TechPrice: 60.00, TechType: constant.TechTypeDisposal, PriceType: "处置费", DeptmentID: dispDept},
		{TechCode: "CZ002", TechName: "雾化吸入", TechFormat: "单次", TechPrice: 40.00, TechType: constant.TechTypeDisposal, PriceType: "处置费", DeptmentID: dispDept},
	}
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.MedicalTechnology{TechCode: r.TechCode}).
			Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDiseases(db *gorm.DB) error {
	rows := []model.Disease{
		{DiseaseCode: "GM", DiseaseName: "急性上呼吸道感染", DiseaseICD: "J06.900", DiseaseCategory: "呼吸系统疾病"},
		{DiseaseCode: "GXY", DiseaseName: "高血压病", DiseaseICD: "I10.x00", DiseaseCategory: "循环系统疾病"},
		{DiseaseCode: "TNB", DiseaseName: "2型糖尿病", DiseaseICD: "E11.900", DiseaseCategory: "内分泌疾病"},
		{DiseaseCode: "WY", DiseaseName: "急性胃炎", DiseaseICD: "K29.100", DiseaseCategory: "消化系统疾病"},
		{DiseaseCode: "QGY", DiseaseName: "急性支气管炎", DiseaseICD: "J20.900", DiseaseCategory: "呼吸系统疾病"},
	}
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.Disease{DiseaseCode: r.DiseaseCode}).
			Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDrugs(db *gorm.DB) error {
	now := time.Now()
	rows := []model.DrugInfo{
		{DrugCode: "YP001", DrugName: "阿莫西林胶囊", DrugFormat: "0.25g*24粒", DrugUnit: "盒", Manufacturer: "华北制药", DrugDosage: "胶囊", DrugType: "抗生素", DrugPrice: 18.50, DrugStock: 500, MnemonicCode: "AMXL", CreationDate: now},
		{DrugCode: "YP002", DrugName: "布洛芬缓释胶囊", DrugFormat: "0.3g*20粒", DrugUnit: "盒", Manufacturer: "中美史克", DrugDosage: "胶囊", DrugType: "解热镇痛", DrugPrice: 22.00, DrugStock: 300, MnemonicCode: "BLF", CreationDate: now},
		{DrugCode: "YP003", DrugName: "盐酸二甲双胍片", DrugFormat: "0.5g*48片", DrugUnit: "盒", Manufacturer: "格华止", DrugDosage: "片剂", DrugType: "降糖药", DrugPrice: 36.80, DrugStock: 200, MnemonicCode: "EJSG", CreationDate: now},
		{DrugCode: "YP004", DrugName: "氨氯地平片", DrugFormat: "5mg*14片", DrugUnit: "盒", Manufacturer: "辉瑞", DrugDosage: "片剂", DrugType: "降压药", DrugPrice: 28.00, DrugStock: 250, MnemonicCode: "ALDP", CreationDate: now},
		{DrugCode: "YP005", DrugName: "蒙脱石散", DrugFormat: "3g*10袋", DrugUnit: "盒", Manufacturer: "博福益普生", DrugDosage: "散剂", DrugType: "消化系统", DrugPrice: 15.20, DrugStock: 400, MnemonicCode: "MTSS", CreationDate: now},
	}
	for i := range rows {
		r := rows[i]
		if err := db.Where(model.DrugInfo{DrugCode: r.DrugCode}).
			Attrs(r).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

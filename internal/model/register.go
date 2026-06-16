package model

import "time"

// Register is the visit master record — one row per registration (SPEC §6 table 6).
// It anchors the whole encounter: medical record, requests and prescriptions all
// reference register_id. visit_state follows the SPEC §5.1 state machine.
type Register struct {
	ID               uint       `gorm:"column:id;primaryKey" json:"id"`
	CaseNumber       string     `gorm:"column:case_number;size:64;index" json:"case_number"`
	RealName         string     `gorm:"column:real_name;size:64;index" json:"real_name"`
	Gender           string     `gorm:"column:gender;size:6" json:"gender"`
	CardNumber       string     `gorm:"column:card_number;size:18" json:"card_number"`
	Birthdate        *time.Time `gorm:"column:birthdate;type:date" json:"birthdate"`
	Age              int        `gorm:"column:age" json:"age"`
	AgeType          string     `gorm:"column:age_type;size:6" json:"age_type"`
	HomeAddress      string     `gorm:"column:home_address;size:128" json:"home_address"`
	VisitDate        time.Time  `gorm:"column:visit_date" json:"visit_date"`
	Noon             string     `gorm:"column:noon;size:6" json:"noon"`
	DeptmentID       uint       `gorm:"column:deptment_id;index" json:"deptment_id"`
	EmployeeID       uint       `gorm:"column:employee_id;index" json:"employee_id"`
	RegistLevelID    uint       `gorm:"column:regist_level_id" json:"regist_level_id"`
	SettleCategoryID uint       `gorm:"column:settle_category_id" json:"settle_category_id"`
	IsBook           string     `gorm:"column:is_book;size:2" json:"is_book"`
	RegistMethod     string     `gorm:"column:regist_method;size:10" json:"regist_method"`
	RegistMoney      float64    `gorm:"column:regist_money;type:decimal(8,2)" json:"regist_money"`
	VisitState       int        `gorm:"column:visit_state;index" json:"visit_state"`

	// Associations.
	Department     *Department     `gorm:"foreignKey:DeptmentID" json:"department,omitempty"`
	Employee       *Employee       `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	RegistLevel    *RegistLevel    `gorm:"foreignKey:RegistLevelID" json:"regist_level,omitempty"`
	SettleCategory *SettleCategory `gorm:"foreignKey:SettleCategoryID" json:"settle_category,omitempty"`
}

func (Register) TableName() string { return "register" }

package model

// Department is a hospital department (SPEC §6 table 2). dept_type drives RBAC.
type Department struct {
	ID       uint   `gorm:"column:id;primaryKey" json:"id"`
	DeptCode string `gorm:"column:dept_code;size:64" json:"dept_code"`
	DeptName string `gorm:"column:dept_name;size:64" json:"dept_name"`
	DeptType string `gorm:"column:dept_type;size:64;index" json:"dept_type"`
	Delmark  int    `gorm:"column:delmark;default:1;index" json:"delmark"`
}

func (Department) TableName() string { return "department" }

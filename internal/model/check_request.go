package model

import "time"

// CheckRequest is a check order (SPEC §6 table 7). check_state follows §5.2:
// 已开立 → 已缴费 → 执行中 → 已出结果 (or 已退费).
type CheckRequest struct {
	ID                   uint       `gorm:"column:id;primaryKey" json:"id"`
	RegisterID           uint       `gorm:"column:register_id;index" json:"register_id"`
	MedicalTechnologyID  uint       `gorm:"column:medical_technology_id" json:"medical_technology_id"`
	CheckInfo            string     `gorm:"column:check_info;size:512" json:"check_info"`
	CheckPosition        string     `gorm:"column:check_position;size:255" json:"check_position"`
	CreationTime         time.Time  `gorm:"column:creation_time" json:"creation_time"`
	CheckEmployeeID      *uint      `gorm:"column:check_employee_id" json:"check_employee_id"`
	InputcheckEmployeeID *uint      `gorm:"column:inputcheck_employee_id" json:"inputcheck_employee_id"`
	CheckTime            *time.Time `gorm:"column:check_time" json:"check_time"`
	CheckResult          string     `gorm:"column:check_result;size:512" json:"check_result"`
	CheckState           string     `gorm:"column:check_state;size:64;index" json:"check_state"`
	CheckRemark          string     `gorm:"column:check_remark;size:512" json:"check_remark"`

	MedicalTechnology *MedicalTechnology `gorm:"foreignKey:MedicalTechnologyID" json:"medical_technology,omitempty"`
	Register          *Register          `gorm:"foreignKey:RegisterID" json:"register,omitempty"`
}

func (CheckRequest) TableName() string { return "check_request" }

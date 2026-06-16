package model

import "time"

// DisposalRequest is a disposal/treatment order (SPEC §6 table 9). Isomorphic to
// CheckRequest with disposal_* columns.
type DisposalRequest struct {
	ID                      uint       `gorm:"column:id;primaryKey" json:"id"`
	RegisterID              uint       `gorm:"column:register_id;index" json:"register_id"`
	MedicalTechnologyID     uint       `gorm:"column:medical_technology_id" json:"medical_technology_id"`
	DisposalInfo            string     `gorm:"column:disposal_info;size:512" json:"disposal_info"`
	DisposalPosition        string     `gorm:"column:disposal_position;size:255" json:"disposal_position"`
	CreationTime            time.Time  `gorm:"column:creation_time" json:"creation_time"`
	DisposalEmployeeID      *uint      `gorm:"column:disposal_employee_id" json:"disposal_employee_id"`
	InputdisposalEmployeeID *uint      `gorm:"column:inputdisposal_employee_id" json:"inputdisposal_employee_id"`
	DisposalTime            *time.Time `gorm:"column:disposal_time" json:"disposal_time"`
	DisposalResult          string     `gorm:"column:disposal_result;size:512" json:"disposal_result"`
	DisposalState           string     `gorm:"column:disposal_state;size:64;index" json:"disposal_state"`
	DisposalRemark          string     `gorm:"column:disposal_remark;size:512" json:"disposal_remark"`

	MedicalTechnology *MedicalTechnology `gorm:"foreignKey:MedicalTechnologyID" json:"medical_technology,omitempty"`
	Register          *Register          `gorm:"foreignKey:RegisterID" json:"register,omitempty"`
}

func (DisposalRequest) TableName() string { return "disposal_request" }

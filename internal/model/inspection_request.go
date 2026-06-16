package model

import "time"

// InspectionRequest is a lab-inspection order (SPEC §6 table 8). Structurally
// isomorphic to CheckRequest but on its own table with inspection_* columns.
type InspectionRequest struct {
	ID                        uint       `gorm:"column:id;primaryKey" json:"id"`
	RegisterID                uint       `gorm:"column:register_id;index" json:"register_id"`
	MedicalTechnologyID       uint       `gorm:"column:medical_technology_id" json:"medical_technology_id"`
	InspectionInfo            string     `gorm:"column:inspection_info;size:512" json:"inspection_info"`
	InspectionPosition        string     `gorm:"column:inspection_position;size:255" json:"inspection_position"`
	CreationTime              time.Time  `gorm:"column:creation_time" json:"creation_time"`
	InspectionEmployeeID      *uint      `gorm:"column:inspection_employee_id" json:"inspection_employee_id"`
	InputinspectionEmployeeID *uint      `gorm:"column:inputinspection_employee_id" json:"inputinspection_employee_id"`
	InspectionTime            *time.Time `gorm:"column:inspection_time" json:"inspection_time"`
	InspectionResult          string     `gorm:"column:inspection_result;size:512" json:"inspection_result"`
	InspectionState           string     `gorm:"column:inspection_state;size:64;index" json:"inspection_state"`
	InspectionRemark          string     `gorm:"column:inspection_remark;size:512" json:"inspection_remark"`

	MedicalTechnology *MedicalTechnology `gorm:"foreignKey:MedicalTechnologyID" json:"medical_technology,omitempty"`
	Register          *Register          `gorm:"foreignKey:RegisterID" json:"register,omitempty"`
}

func (InspectionRequest) TableName() string { return "inspection_request" }

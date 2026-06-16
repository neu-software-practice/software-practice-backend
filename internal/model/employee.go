package model

// Employee is a hospital staff member, mostly doctors (SPEC §6 table 1).
// username + password (bcrypt hash) are the §6 補全 login fields.
type Employee struct {
	ID            uint   `gorm:"column:id;primaryKey" json:"id"`
	Username      string `gorm:"column:username;size:64;uniqueIndex" json:"username"`
	Password      string `gorm:"column:password;size:128" json:"-"`
	Realname      string `gorm:"column:realname;size:64" json:"realname"`
	DeptmentID    uint   `gorm:"column:deptment_id;index" json:"deptment_id"`
	RegistLevelID *uint  `gorm:"column:regist_level_id" json:"regist_level_id"`
	SchedulingID  *uint  `gorm:"column:scheduling_id" json:"scheduling_id"`
	Delmark       int    `gorm:"column:delmark;default:1;index" json:"delmark"`

	// Associations (populated via Preload where useful).
	Department  *Department  `gorm:"foreignKey:DeptmentID" json:"department,omitempty"`
	RegistLevel *RegistLevel `gorm:"foreignKey:RegistLevelID" json:"regist_level,omitempty"`
	Scheduling  *Scheduling  `gorm:"foreignKey:SchedulingID" json:"scheduling,omitempty"`
}

func (Employee) TableName() string { return "employee" }

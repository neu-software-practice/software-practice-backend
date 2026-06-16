package model

// Scheduling is a doctor's weekly on-duty rule (SPEC §6 table 5). week_rule
// encodes which half-days the doctor sees patients.
type Scheduling struct {
	ID       uint   `gorm:"column:id;primaryKey" json:"id"`
	RuleName string `gorm:"column:rule_name;size:64" json:"rule_name"`
	WeekRule string `gorm:"column:week_rule;size:32" json:"week_rule"`
	Delmark  int    `gorm:"column:delmark;default:1;index" json:"delmark"`
}

func (Scheduling) TableName() string { return "scheduling" }

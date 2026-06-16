package dto

import (
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// RegisterRequest is the F1-1 window-registration body. visit_date/noon are set
// server-side (today / current half-day) and are not client-supplied.
type RegisterRequest struct {
	RealName         string `json:"real_name" binding:"required"`
	Gender           string `json:"gender" binding:"required"`
	CardNumber       string `json:"card_number"`
	Birthdate        string `json:"birthdate"` // YYYY-MM-DD
	Age              int    `json:"age"`
	AgeType          string `json:"age_type"`
	HomeAddress      string `json:"home_address"`
	DeptmentID       uint   `json:"deptment_id" binding:"required"`
	EmployeeID       uint   `json:"employee_id" binding:"required"`
	RegistLevelID    uint   `json:"regist_level_id" binding:"required"`
	SettleCategoryID uint   `json:"settle_category_id" binding:"required"`
	IsBook           string `json:"is_book"`
	RegistMethod     string `json:"regist_method"`
}

// RegisterBrief is a light projection of a visit for lists and headers.
type RegisterBrief struct {
	ID            uint      `json:"id"`
	CaseNumber    string    `json:"case_number"`
	RealName      string    `json:"real_name"`
	Gender        string    `json:"gender"`
	Age           int       `json:"age"`
	AgeType       string    `json:"age_type"`
	CardNumber    string    `json:"card_number"`
	VisitDate     time.Time `json:"visit_date"`
	Noon          string    `json:"noon"`
	VisitState    int       `json:"visit_state"`
	DeptmentID    uint      `json:"deptment_id"`
	EmployeeID    uint      `json:"employee_id"`
	RegistLevelID uint      `json:"regist_level_id"`
	RegistMoney   float64   `json:"regist_money"`
}

// NewRegisterBrief projects a register row.
func NewRegisterBrief(r *model.Register) RegisterBrief {
	return RegisterBrief{
		ID: r.ID, CaseNumber: r.CaseNumber, RealName: r.RealName, Gender: r.Gender,
		Age: r.Age, AgeType: r.AgeType, CardNumber: r.CardNumber, VisitDate: r.VisitDate,
		Noon: r.Noon, VisitState: r.VisitState, DeptmentID: r.DeptmentID, EmployeeID: r.EmployeeID,
		RegistLevelID: r.RegistLevelID, RegistMoney: r.RegistMoney,
	}
}

// NewRegisterBriefs projects a slice of register rows.
func NewRegisterBriefs(rows []model.Register) []RegisterBrief {
	out := make([]RegisterBrief, 0, len(rows))
	for i := range rows {
		out = append(out, NewRegisterBrief(&rows[i]))
	}
	return out
}

package dto

import (
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// RequestView is the uniform projection of a check/inspection/disposal request.
type RequestView struct {
	ID           uint       `json:"id"`
	RegisterID   uint       `json:"register_id"`
	TechID       uint       `json:"tech_id"`
	TechName     string     `json:"tech_name"`
	TechPrice    float64    `json:"tech_price"`
	Info         string     `json:"info"`
	Position     string     `json:"position"`
	Remark       string     `json:"remark"`
	State        string     `json:"state"`
	CreationTime time.Time  `json:"creation_time"`
	Result       string     `json:"result"`
	ResultTime   *time.Time `json:"result_time"`
	ExecutorID   *uint      `json:"executor_id"`
	InputterID   *uint      `json:"inputter_id"`
}

// NewRequestView projects any MedTechRequest into the uniform view.
func NewRequestView(r model.MedTechRequest) RequestView {
	v := RequestView{
		ID:           r.RequestID(),
		RegisterID:   r.RequestRegisterID(),
		TechID:       r.RequestTechID(),
		Info:         r.Info(),
		Position:     r.Position(),
		Remark:       r.Remark(),
		State:        r.State(),
		CreationTime: r.GetCreationTime(),
		Result:       r.Result(),
		ResultTime:   r.GetResultTime(),
		ExecutorID:   r.GetExecutorID(),
		InputterID:   r.GetInputterID(),
	}
	if t := r.GetMedicalTechnology(); t != nil {
		v.TechName = t.TechName
		v.TechPrice = t.TechPrice
	}
	return v
}

// CreateRequestInput opens a check/inspection/disposal order (F2-3/F2-4/F2-10).
type CreateRequestInput struct {
	RegisterID uint   `json:"register_id" binding:"required"`
	TechID     uint   `json:"tech_id" binding:"required"`
	Info       string `json:"info"`
	Position   string `json:"position"`
	Remark     string `json:"remark"`
}

// ExecuteRequestInput assigns an executor (F3-2/F4-2/F6-2). executor_id is
// optional and defaults to the logged-in tech doctor.
type ExecuteRequestInput struct {
	ExecutorID uint `json:"executor_id"`
}

// ResultRequestInput records a result (F3-3/F4-3/F6-3). inputter_id defaults to
// the logged-in tech doctor.
type ResultRequestInput struct {
	Result     string `json:"result" binding:"required"`
	InputterID uint   `json:"inputter_id"`
}

// PendingItem is one payable line shown on the charging screen (F1-3).
type PendingItem struct {
	ItemType     string    `json:"item_type"` // check / inspection / disposal / prescription
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	Spec         string    `json:"spec,omitempty"`
	Quantity     int       `json:"quantity,omitempty"`
	Amount       float64   `json:"amount"`
	CreationTime time.Time `json:"creation_time"`
}

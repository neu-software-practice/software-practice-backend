package model

import "time"

// MedTechRequest is the common behavior shared by the three isomorphic request
// tables (check / inspection / disposal). It lets one generic repository and one
// generic service drive all three despite their differing column names, instead
// of triplicating the state-machine logic (PLAN §7 "抽公共 service/handler 模板").
//
// Mutating setters use pointer receivers — idiomatic Go, the explicit exception
// to the immutability rule. Read getters let a generic mapper project any of the
// three into a uniform view DTO.
type MedTechRequest interface {
	RequestID() uint
	RequestRegisterID() uint
	RequestTechID() uint
	State() string
	SetState(state string)
	StateColumn() string
	TableName() string
	AssignExecutor(employeeID uint)
	RecordResult(result string, inputterID uint, at time.Time)
	SetCreation(at time.Time)

	// Read accessors used by the generic view mapper.
	GetMedicalTechnology() *MedicalTechnology
	GetCreationTime() time.Time
	Info() string
	Position() string
	Remark() string
	Result() string
	GetResultTime() *time.Time
	GetExecutorID() *uint
	GetInputterID() *uint
}

// --- CheckRequest ---

func (r *CheckRequest) RequestID() uint                          { return r.ID }
func (r *CheckRequest) RequestRegisterID() uint                  { return r.RegisterID }
func (r *CheckRequest) RequestTechID() uint                      { return r.MedicalTechnologyID }
func (r *CheckRequest) State() string                            { return r.CheckState }
func (r *CheckRequest) SetState(state string)                    { r.CheckState = state }
func (r *CheckRequest) StateColumn() string                      { return "check_state" }
func (r *CheckRequest) AssignExecutor(id uint)                   { r.CheckEmployeeID = &id }
func (r *CheckRequest) SetCreation(at time.Time)                 { r.CreationTime = at }
func (r *CheckRequest) GetMedicalTechnology() *MedicalTechnology { return r.MedicalTechnology }
func (r *CheckRequest) GetCreationTime() time.Time               { return r.CreationTime }
func (r *CheckRequest) Info() string                             { return r.CheckInfo }
func (r *CheckRequest) Position() string                         { return r.CheckPosition }
func (r *CheckRequest) Remark() string                           { return r.CheckRemark }
func (r *CheckRequest) Result() string                           { return r.CheckResult }
func (r *CheckRequest) GetResultTime() *time.Time                { return r.CheckTime }
func (r *CheckRequest) GetExecutorID() *uint                     { return r.CheckEmployeeID }
func (r *CheckRequest) GetInputterID() *uint                     { return r.InputcheckEmployeeID }
func (r *CheckRequest) RecordResult(result string, inputterID uint, at time.Time) {
	r.CheckResult = result
	r.InputcheckEmployeeID = &inputterID
	r.CheckTime = &at
}

// --- InspectionRequest ---

func (r *InspectionRequest) RequestID() uint                          { return r.ID }
func (r *InspectionRequest) RequestRegisterID() uint                  { return r.RegisterID }
func (r *InspectionRequest) RequestTechID() uint                      { return r.MedicalTechnologyID }
func (r *InspectionRequest) State() string                            { return r.InspectionState }
func (r *InspectionRequest) SetState(state string)                    { r.InspectionState = state }
func (r *InspectionRequest) StateColumn() string                      { return "inspection_state" }
func (r *InspectionRequest) AssignExecutor(id uint)                   { r.InspectionEmployeeID = &id }
func (r *InspectionRequest) SetCreation(at time.Time)                 { r.CreationTime = at }
func (r *InspectionRequest) GetMedicalTechnology() *MedicalTechnology { return r.MedicalTechnology }
func (r *InspectionRequest) GetCreationTime() time.Time               { return r.CreationTime }
func (r *InspectionRequest) Info() string                             { return r.InspectionInfo }
func (r *InspectionRequest) Position() string                         { return r.InspectionPosition }
func (r *InspectionRequest) Remark() string                           { return r.InspectionRemark }
func (r *InspectionRequest) Result() string                           { return r.InspectionResult }
func (r *InspectionRequest) GetResultTime() *time.Time                { return r.InspectionTime }
func (r *InspectionRequest) GetExecutorID() *uint                     { return r.InspectionEmployeeID }
func (r *InspectionRequest) GetInputterID() *uint                     { return r.InputinspectionEmployeeID }
func (r *InspectionRequest) RecordResult(result string, inputterID uint, at time.Time) {
	r.InspectionResult = result
	r.InputinspectionEmployeeID = &inputterID
	r.InspectionTime = &at
}

// --- DisposalRequest ---

func (r *DisposalRequest) RequestID() uint                          { return r.ID }
func (r *DisposalRequest) RequestRegisterID() uint                  { return r.RegisterID }
func (r *DisposalRequest) RequestTechID() uint                      { return r.MedicalTechnologyID }
func (r *DisposalRequest) State() string                            { return r.DisposalState }
func (r *DisposalRequest) SetState(state string)                    { r.DisposalState = state }
func (r *DisposalRequest) StateColumn() string                      { return "disposal_state" }
func (r *DisposalRequest) AssignExecutor(id uint)                   { r.DisposalEmployeeID = &id }
func (r *DisposalRequest) SetCreation(at time.Time)                 { r.CreationTime = at }
func (r *DisposalRequest) GetMedicalTechnology() *MedicalTechnology { return r.MedicalTechnology }
func (r *DisposalRequest) GetCreationTime() time.Time               { return r.CreationTime }
func (r *DisposalRequest) Info() string                             { return r.DisposalInfo }
func (r *DisposalRequest) Position() string                         { return r.DisposalPosition }
func (r *DisposalRequest) Remark() string                           { return r.DisposalRemark }
func (r *DisposalRequest) Result() string                           { return r.DisposalResult }
func (r *DisposalRequest) GetResultTime() *time.Time                { return r.DisposalTime }
func (r *DisposalRequest) GetExecutorID() *uint                     { return r.DisposalEmployeeID }
func (r *DisposalRequest) GetInputterID() *uint                     { return r.InputdisposalEmployeeID }
func (r *DisposalRequest) RecordResult(result string, inputterID uint, at time.Time) {
	r.DisposalResult = result
	r.InputdisposalEmployeeID = &inputterID
	r.DisposalTime = &at
}

// Package constant defines the shared enumerations used across the HIS domain.
// Centralizing these string/int literals avoids magic values scattered through
// the codebase (SPEC §5 state machines, §3 RBAC dept_type).
package constant

// Department types — RBAC is enforced by the dept_type of the employee's
// department (SPEC §3). The JWT carries dept_type and the router guards by it.
const (
	DeptTypeFinance    = "财务"
	DeptTypeOutpatient = "门诊"
	DeptTypeCheck      = "检查"
	DeptTypeInspection = "检验"
	DeptTypePharmacy   = "药房"
	DeptTypeDisposal   = "处置"
	DeptTypeRoot       = "root"
)

// Visit states for register.visit_state (SPEC §5.1).
const (
	VisitStateRegistered = 1 // 已挂号
	VisitStateInConsult  = 2 // 医生接诊
	VisitStateFinished   = 3 // 看诊结束
	VisitStateCanceled   = 4 // 已退号
)

// Request states shared by check_request / inspection_request / disposal_request
// (*_state VARCHAR, SPEC §5.2).
const (
	RequestStateCreated   = "已开立" // 待缴费
	RequestStatePaid      = "已缴费" // 待执行
	RequestStateExecuting = "执行中" // 已分配执行医生
	RequestStateCompleted = "已出结果"
	RequestStateRefunded  = "已退费"
)

// Prescription states for prescription.drug_state (SPEC §5.3).
const (
	PrescriptionStateCreated   = "已开立" // 待缴费
	PrescriptionStatePaid      = "已缴费" // 待发药
	PrescriptionStateDispensed = "已发药"
	PrescriptionStateRefunded  = "已退药"
)

// Noon (午别) for register.noon.
const (
	NoonMorning   = "上午"
	NoonAfternoon = "下午"
)

// Medical technology categories for medical_technology.tech_type. Determines
// whether a project is orderable as a check / inspection / disposal request.
const (
	TechTypeCheck      = "检查"
	TechTypeInspection = "检验"
	TechTypeDisposal   = "处置"
)

// Charge item categories used by the charging module (F1-3/F1-4) to address a
// pending payable item across the heterogeneous request/prescription tables.
const (
	ChargeItemCheck        = "check"
	ChargeItemInspection   = "inspection"
	ChargeItemDisposal     = "disposal"
	ChargeItemPrescription = "prescription"
)

// DelmarkActive / DelmarkDeleted are the soft-delete sentinels (delmark column).
const (
	DelmarkActive  = 1
	DelmarkDeleted = 0
)

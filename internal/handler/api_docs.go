package handler

import (
	"github.com/neu-software-practice/software-practice-backend/internal/dto"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
)

// Keep dto/response referenced so swaggo can resolve the types named in the
// annotations below — they appear only in comments, not in code.
var (
	_ = dto.CreateRequestInput{}
	_ = response.Body{}
)

// This file carries OpenAPI (swaggo) annotations for the check / inspection /
// disposal endpoints. They are served by the single generic RequestHandler, so
// one Go method maps to three concrete URL paths — and a method can hold only one
// @Router. These no-op stubs give swag one declaration per path. They are never
// called; the reference slice at the bottom keeps the unused linter satisfied
// without exporting them.

// docCheckCreate godoc
// @Summary  开立检查申请 (F2-3)
// @Tags     check
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.CreateRequestInput  true  "检查申请"
// @Success  201   {object}  response.Body
// @Router   /check-requests [post]
func docCheckCreate() {}

// docCheckPending godoc
// @Summary  检查申请受理-待检查患者 (F3-1)
// @Tags     check
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    name         query     string  false  "姓名"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /check/pending [get]
func docCheckPending() {}

// docCheckCounts godoc
// @Summary  检查统计 (已检查/排队, F3-1)
// @Tags     check
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Router   /check/counts [get]
func docCheckCounts() {}

// docCheckRequests godoc
// @Summary  患者检查项目 (F3-2)
// @Tags     check
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int     true   "挂号ID"
// @Param    state        query     string  false  "状态(默认 已缴费)"
// @Success  200          {object}  response.Body
// @Router   /check-requests [get]
func docCheckRequests() {}

// docCheckManage godoc
// @Summary  检查管理-患者全部检查申请 (F3-4)
// @Tags     check
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int  true  "挂号ID"
// @Success  200          {object}  response.Body
// @Router   /check-requests/manage [get]
func docCheckManage() {}

// docCheckExecute godoc
// @Summary  检查患者录入-分配执行医生 (F3-2)
// @Tags     check
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                      true   "检查申请ID"
// @Param    body  body      dto.ExecuteRequestInput  false  "执行医生(默认当前用户)"
// @Success  200   {object}  response.Body
// @Router   /check-requests/{id}/execute [post]
func docCheckExecute() {}

// docCheckResult godoc
// @Summary  检查结果录入 (F3-3)
// @Tags     check
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                     true  "检查申请ID"
// @Param    body  body      dto.ResultRequestInput  true  "检查结果"
// @Success  200   {object}  response.Body
// @Router   /check-requests/{id}/result [post]
func docCheckResult() {}

// docInspectionCreate godoc
// @Summary  开立检验申请 (F2-4)
// @Tags     inspection
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.CreateRequestInput  true  "检验申请"
// @Success  201   {object}  response.Body
// @Router   /inspection-requests [post]
func docInspectionCreate() {}

// docInspectionPending godoc
// @Summary  检验申请受理-待检验患者 (F4-1)
// @Tags     inspection
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    name         query     string  false  "姓名"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /inspection/pending [get]
func docInspectionPending() {}

// docInspectionCounts godoc
// @Summary  检验统计 (已检验/排队, F4-1)
// @Tags     inspection
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Router   /inspection/counts [get]
func docInspectionCounts() {}

// docInspectionRequests godoc
// @Summary  患者检验项目 (F4-2)
// @Tags     inspection
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int     true   "挂号ID"
// @Param    state        query     string  false  "状态(默认 已缴费)"
// @Success  200          {object}  response.Body
// @Router   /inspection-requests [get]
func docInspectionRequests() {}

// docInspectionManage godoc
// @Summary  检验管理-患者全部检验申请 (F4-4)
// @Tags     inspection
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int  true  "挂号ID"
// @Success  200          {object}  response.Body
// @Router   /inspection-requests/manage [get]
func docInspectionManage() {}

// docInspectionExecute godoc
// @Summary  检验患者录入-分配执行医生 (F4-2)
// @Tags     inspection
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                      true   "检验申请ID"
// @Param    body  body      dto.ExecuteRequestInput  false  "执行医生(默认当前用户)"
// @Success  200   {object}  response.Body
// @Router   /inspection-requests/{id}/execute [post]
func docInspectionExecute() {}

// docInspectionResult godoc
// @Summary  检验结果录入 (F4-3)
// @Tags     inspection
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                     true  "检验申请ID"
// @Param    body  body      dto.ResultRequestInput  true  "检验结果"
// @Success  200   {object}  response.Body
// @Router   /inspection-requests/{id}/result [post]
func docInspectionResult() {}

// docDisposalCreate godoc
// @Summary  开立处置申请 (F2-10)
// @Tags     disposal
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body      dto.CreateRequestInput  true  "处置申请"
// @Success  201   {object}  response.Body
// @Router   /disposal-requests [post]
func docDisposalCreate() {}

// docDisposalPending godoc
// @Summary  处置申请受理-待处置患者 (F6-1)
// @Tags     disposal
// @Produce  json
// @Security BearerAuth
// @Param    case_number  query     string  false  "病历号"
// @Param    name         query     string  false  "姓名"
// @Param    page         query     int     false  "页码"
// @Param    limit        query     int     false  "每页条数"
// @Success  200          {object}  response.Body
// @Router   /disposal/pending [get]
func docDisposalPending() {}

// docDisposalCounts godoc
// @Summary  处置统计 (已处置/排队, F6-1)
// @Tags     disposal
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.Body
// @Router   /disposal/counts [get]
func docDisposalCounts() {}

// docDisposalRequests godoc
// @Summary  患者处置项目 (F6-2)
// @Tags     disposal
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int     true   "挂号ID"
// @Param    state        query     string  false  "状态(默认 已缴费)"
// @Success  200          {object}  response.Body
// @Router   /disposal-requests [get]
func docDisposalRequests() {}

// docDisposalManage godoc
// @Summary  处置管理-患者全部处置申请 (F6-4)
// @Tags     disposal
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int  true  "挂号ID"
// @Success  200          {object}  response.Body
// @Router   /disposal-requests/manage [get]
func docDisposalManage() {}

// docDisposalExecute godoc
// @Summary  处置患者录入-分配执行医生 (F6-2)
// @Tags     disposal
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                      true   "处置申请ID"
// @Param    body  body      dto.ExecuteRequestInput  false  "执行医生(默认当前用户)"
// @Success  200   {object}  response.Body
// @Router   /disposal-requests/{id}/execute [post]
func docDisposalExecute() {}

// docDisposalResult godoc
// @Summary  处置结果录入 (F6-3)
// @Tags     disposal
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id    path      int                     true  "处置申请ID"
// @Param    body  body      dto.ResultRequestInput  true  "处置结果"
// @Success  200   {object}  response.Body
// @Router   /disposal-requests/{id}/result [post]
func docDisposalResult() {}

// docCheckResults godoc
// @Summary  查看检查结果 (F2-6)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int  true  "挂号ID"
// @Success  200          {object}  response.Body
// @Router   /check-requests/results [get]
func docCheckResults() {}

// docInspectionResults godoc
// @Summary  查看检验结果 (F2-7)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int  true  "挂号ID"
// @Success  200          {object}  response.Body
// @Router   /inspection-requests/results [get]
func docInspectionResults() {}

// docDisposalResults godoc
// @Summary  查看处置结果
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int  true  "挂号ID"
// @Success  200          {object}  response.Body
// @Router   /disposal-requests/results [get]
func docDisposalResults() {}

// docPhysicianChargeRecords godoc
// @Summary  费用查询 (F2-11)
// @Tags     physician
// @Produce  json
// @Security BearerAuth
// @Param    register_id  query     int     false  "挂号ID"
// @Param    case_number  query     string  false  "病历号"
// @Success  200          {object}  response.Body
// @Router   /physician/charge-records [get]
func docPhysicianChargeRecords() {}

// The blank reference keeps every documentation stub "used" for the linter
// without exporting them or being a named (and itself unused) variable.
var _ = []func(){
	docCheckCreate, docCheckPending, docCheckCounts, docCheckRequests, docCheckManage, docCheckExecute, docCheckResult,
	docInspectionCreate, docInspectionPending, docInspectionCounts, docInspectionRequests, docInspectionManage, docInspectionExecute, docInspectionResult,
	docDisposalCreate, docDisposalPending, docDisposalCounts, docDisposalRequests, docDisposalManage, docDisposalExecute, docDisposalResult,
	docCheckResults, docInspectionResults, docDisposalResults,
	docPhysicianChargeRecords,
}

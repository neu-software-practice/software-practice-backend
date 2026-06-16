// Package router wires the HTTP routes and middleware (PLAN §2.1, §4). Routes are
// grouped under /api and guarded per dept_type via the RBAC middleware.
package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/handler"
	"github.com/neu-software-practice/software-practice-backend/internal/middleware"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/constant"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/jwt"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/response"
	"github.com/neu-software-practice/software-practice-backend/internal/repository"

	// Registers the generated OpenAPI spec with the swagger handler.
	_ "github.com/neu-software-practice/software-practice-backend/internal/swagger"
)

// Deps carries everything the router needs. The app container populates it.
type Deps struct {
	Cfg          *config.Config
	Tokens       *jwt.Manager
	Auth         *handler.AuthHandler
	Catalog      *handler.CatalogHandler
	Registration *handler.RegistrationHandler
	Charge       *handler.ChargeHandler
	Physician    *handler.PhysicianHandler
	Pharmacy     *handler.PharmacyHandler
	Check        *handler.RequestHandler[model.CheckRequest, *model.CheckRequest]
	Inspection   *handler.RequestHandler[model.InspectionRequest, *model.InspectionRequest]
	Disposal     *handler.RequestHandler[model.DisposalRequest, *model.DisposalRequest]
}

// New builds the configured Gin engine.
func New(d Deps) *gin.Engine {
	if d.Cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Logger(), middleware.CORS(d.Cfg.CORSOrigins))

	r.GET("/api/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	// OpenAPI / Swagger UI at /swagger/index.html (SPEC §9.5).
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	auth := func() gin.HandlerFunc { return middleware.Auth(d.Tokens) }
	dept := middleware.RequireDeptType

	registerAuthRoutes(api, d)

	// Reference data & search — any authenticated user.
	cat := api.Group("", auth())
	{
		cat.GET("/departments", d.Catalog.Departments)
		cat.GET("/regist-levels", d.Catalog.RegistLevels)
		cat.GET("/settle-categories", d.Catalog.SettleCategories)
		cat.GET("/doctors", d.Catalog.Doctors)
		cat.GET("/medical-technologies", d.Catalog.MedicalTechnologies)
		cat.GET("/diseases", d.Catalog.Diseases)
		cat.GET("/drugs", d.Catalog.Drugs)
	}

	// 财务: 挂号/退号 + 收费/退费 + 费用记录.
	fin := api.Group("", auth(), dept(constant.DeptTypeFinance))
	{
		fin.POST("/registers", d.Registration.Register)
		fin.GET("/registers", d.Registration.List)
		fin.POST("/registers/:id/cancel", d.Registration.Cancel)
		fin.GET("/charges/pending", d.Charge.Pending)
		fin.POST("/charges", d.Charge.Charge)
		fin.GET("/charges/refund-pending", d.Charge.RefundPending)
		fin.POST("/charges/refund", d.Charge.Refund)
		fin.GET("/charge-records", d.Charge.Records)
	}

	// 门诊医生: 诊疗.
	phy := api.Group("/physician", auth(), dept(constant.DeptTypeOutpatient))
	{
		phy.GET("/patients", d.Physician.Patients)
		phy.GET("/patients/counts", d.Physician.Counts)
		phy.GET("/history", d.Physician.History)
		phy.POST("/registers/:id/consult", d.Physician.Consult)
		phy.GET("/registers/:id/medical-record", d.Physician.GetMedicalRecord)
		phy.PUT("/registers/:id/medical-record", d.Physician.SaveMedicalRecord)
		phy.PUT("/registers/:id/diagnosis", d.Physician.Diagnose)
		phy.POST("/registers/:id/prescriptions", d.Physician.WritePrescription)
		phy.GET("/charge-records", d.Charge.Records) // F2-11 费用查询
	}
	// 门诊医生: 开立检查/检验/处置 + 查看结果.
	orders := api.Group("", auth(), dept(constant.DeptTypeOutpatient))
	{
		orders.POST("/check-requests", d.Check.Create)
		orders.POST("/inspection-requests", d.Inspection.Create)
		orders.POST("/disposal-requests", d.Disposal.Create)
		orders.GET("/check-requests/results", d.Check.Results)
		orders.GET("/inspection-requests/results", d.Inspection.Results)
		orders.GET("/disposal-requests/results", d.Disposal.Results)
	}

	// 检查/检验/处置医生: 受理 → 执行 → 结果录入.
	registerTechRoutes(api, "check", d.Check, auth(), dept(constant.DeptTypeCheck))
	registerTechRoutes(api, "inspection", d.Inspection, auth(), dept(constant.DeptTypeInspection))
	registerTechRoutes(api, "disposal", d.Disposal, auth(), dept(constant.DeptTypeDisposal))

	// 药房: 发药/退药 + 药库管理 + 交易记录.
	ph := api.Group("/pharmacy", auth(), dept(constant.DeptTypePharmacy))
	{
		ph.GET("/prescriptions", d.Pharmacy.Prescriptions)
		ph.POST("/prescriptions/:id/dispense", d.Pharmacy.Dispense)
		ph.POST("/prescriptions/:id/refund", d.Pharmacy.Refund)
		ph.GET("/transactions", d.Pharmacy.Transactions)
		ph.POST("/drugs", d.Pharmacy.CreateDrug)
		ph.PUT("/drugs/:id", d.Pharmacy.UpdateDrug)
		ph.DELETE("/drugs/:id", d.Pharmacy.DeleteDrug)
		ph.POST("/drugs/:id/restock", d.Pharmacy.Restock)
	}

	return r
}

func registerAuthRoutes(api *gin.RouterGroup, d Deps) {
	auth := api.Group("/auth")
	auth.POST("/login", d.Auth.Login)
	auth.GET("/me", middleware.Auth(d.Tokens), d.Auth.Me)
}

// registerTechRoutes wires the tech-doctor side (受理/执行/结果) for one request
// family. The generic RequestHandler is reused across check/inspection/disposal;
// only the URL prefix differs.
func registerTechRoutes[T any, PT repository.RequestPtr[T]](api *gin.RouterGroup, prefix string, h *handler.RequestHandler[T, PT], guards ...gin.HandlerFunc) {
	g := api.Group("", guards...)
	g.GET("/"+prefix+"/pending", h.PendingPatients)
	g.GET("/"+prefix+"/counts", h.Counts)
	g.GET("/"+prefix+"-requests", h.PatientRequests)
	g.GET("/"+prefix+"-requests/manage", h.Manage)
	g.POST("/"+prefix+"-requests/:id/execute", h.Execute)
	g.POST("/"+prefix+"-requests/:id/result", h.RecordResult)
}

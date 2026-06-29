package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/neuhis/software-practice-backend/internal/config"
	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/middleware"
	authsvc "github.com/neuhis/software-practice-backend/internal/service/auth"
	patientsvc "github.com/neuhis/software-practice-backend/internal/service/patient"
	visitsvc "github.com/neuhis/software-practice-backend/internal/service/visit"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
)

// Router holds all route handlers.
type Router struct {
	Patient   *PatientHandler
	Visit     *VisitHandler
	Workbench *WorkbenchHandler
	Auth      *AuthHandler
}

// NewRouter creates a new Router.
func NewRouter(
	patientSvc *patientsvc.Service,
	visitSvc *visitsvc.Service,
	workbenchSvc *wbsvc.Service,
	authSvc *authsvc.Service,
) *Router {
	return &Router{
		Patient:   NewPatientHandler(patientSvc),
		Visit:     NewVisitHandler(visitSvc),
		Workbench: NewWorkbenchHandler(workbenchSvc),
		Auth:      NewAuthHandler(authSvc),
	}
}

// SetupRoutes registers all routes on the Gin engine.
func SetupRoutes(engine *gin.Engine, cfg *config.Config, router *Router) {
	// Health check (no auth)
	engine.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := engine.Group("/api")

	// Auth routes (public, stricter rate limit: 5 req/min/IP)
	authGroup := api.Group("/auth")
	authGroup.Use(middleware.RateLimitMiddleware(5.0/60.0, 5))
	{
		authGroup.POST("/register", router.Auth.Register)
		authGroup.POST("/login", router.Auth.Login)
		authGroup.POST("/refresh", router.Auth.Refresh)
		authGroup.POST("/logout", router.Auth.Logout)
	}

	// Public routes (no auth)
	api.POST("/patients/verify", router.Patient.VerifyIdentity)

	// Authenticated routes
	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// Patient routes
		auth.GET("/patients/:patientId/context", router.Patient.GetContext)
		auth.PATCH("/patients/:patientId/profile", router.Patient.UpdateProfile)

		// Visit routes
		auth.POST("/visits", router.Visit.CreateSession)
		auth.POST("/visits/:sessionId/follow-up", router.Visit.CreateFollowUp)
		auth.GET("/visits", router.Visit.ListSessions)
		auth.GET("/visits/:sessionId", router.Visit.GetSession)
		auth.GET("/visits/:sessionId/snapshot", router.Visit.GetSnapshot)

		// Workbench routes
		auth.GET("/visits/:sessionId/timeline", router.Workbench.ListTimeline)
		auth.POST("/visits/:sessionId/messages", router.Workbench.SendMessage)
		auth.POST("/visits/:sessionId/assistant-stream", router.Workbench.StreamAssistantMessage)
		auth.POST("/visits/:sessionId/lab-decision", router.Workbench.SubmitLabDecision)
		auth.POST("/visits/:sessionId/payments", router.Workbench.SubmitPayment)
		auth.POST("/visits/:sessionId/fulfillment", router.Workbench.SubmitFulfillment)
		auth.POST("/visits/:sessionId/treatment-execution", router.Workbench.SubmitTreatmentExecution)
		auth.POST("/visits/:sessionId/advice-ack", router.Workbench.AckAdvice)
		auth.POST("/visits/:sessionId/lock-question", router.Workbench.AskLockedQuestion)
		auth.POST("/visits/:sessionId/classify-intent", router.Workbench.ClassifyIntent)
		auth.POST("/visits/:sessionId/consult", router.Workbench.StreamConsultationReply)
		auth.POST("/visits/:sessionId/vitals", router.Workbench.ReportVitals)
		auth.POST("/visits/:sessionId/exit", router.Workbench.ExitVisit)
		auth.POST("/visits/:sessionId/timer", router.Workbench.ToggleTimer)
		auth.POST("/visits/:sessionId/dismiss-emergency", router.Workbench.DismissEmergency)
		auth.POST("/visits/:sessionId/generate-title", router.Workbench.GenerateTitle)
	}

	// Error handler for 404
	engine.NoRoute(func(c *gin.Context) {
		apperrors.WriteNotFound(c, apperrors.CodeNotFound, "endpoint not found")
	})
}

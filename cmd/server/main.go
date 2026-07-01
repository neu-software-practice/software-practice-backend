package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/neuhis/software-practice-backend/internal/config"
	"github.com/neuhis/software-practice-backend/internal/handler"
	"github.com/neuhis/software-practice-backend/internal/llm"
	"github.com/neuhis/software-practice-backend/internal/middleware"
	"github.com/neuhis/software-practice-backend/internal/migrator"
	"github.com/neuhis/software-practice-backend/internal/repository"
	addresssvc "github.com/neuhis/software-practice-backend/internal/service/address"
	adminsvc "github.com/neuhis/software-practice-backend/internal/service/admin"
	authsvc "github.com/neuhis/software-practice-backend/internal/service/auth"
	billingsvc "github.com/neuhis/software-practice-backend/internal/service/billing"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
	medicalordersvc "github.com/neuhis/software-practice-backend/internal/service/medicalorder"
	patientsvc "github.com/neuhis/software-practice-backend/internal/service/patient"
	visitsvc "github.com/neuhis/software-practice-backend/internal/service/visit"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set Gin mode
	switch cfg.ServerMode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := sql.Open("mysql", cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connected")

	// Run database migrations
	if err := migrator.Run(db, "db/migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	patientRepo := repository.NewPatientRepository(db)
	visitRepo := repository.NewVisitRepository(db)
	timelineRepo := repository.NewTimelineRepository(db)
	flowCardRepo := repository.NewFlowCardRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	adminRefreshTokenRepo := repository.NewAdminRefreshTokenRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// Initialize medAgent client
	medAgentClient := medagent.NewClient(cfg.MedAgentBaseURL)
	log.Printf("MedAgent client initialized: %s", cfg.MedAgentBaseURL)

	// Initialize LLM client for title generation (reuses medAgent provider config)
	llmClient := llm.NewFromProvider(cfg.MedAgentProvider, cfg.MedAgentAPIKey, cfg.MedAgentModel, "")
	log.Printf("LLM client initialized: provider=%s model=%s", cfg.MedAgentProvider, cfg.MedAgentModel)

	// Initialize services
	patientSvc := patientsvc.NewService(patientRepo, visitRepo)
	visitSvc := visitsvc.NewService(visitRepo, timelineRepo, patientRepo)
	workbenchSvc := wbsvc.NewService(
		patientRepo, visitRepo, timelineRepo, flowCardRepo, addressRepo,
		visitSvc, medAgentClient, cfg.MedAgentMode, llmClient,
	)
	authSvc := authsvc.NewService(userRepo, refreshTokenRepo, patientRepo, cfg.JWTSecret)
	addressSvc := addresssvc.NewService(addressRepo)
	billingSvc := billingsvc.NewService(visitRepo, flowCardRepo)
	medicalOrderSvc := medicalordersvc.NewService(visitRepo, flowCardRepo)
	adminSvc := adminsvc.NewService(adminRepo, adminRefreshTokenRepo, dashboardRepo, settingsRepo, patientRepo, visitRepo, cfg.AdminJWTSecret)

	// Initialize handlers
	router := handler.NewRouter(patientSvc, visitSvc, workbenchSvc, authSvc, addressSvc, billingSvc, medicalOrderSvc, adminSvc)

	// Create Gin engine
	engine := gin.New()

	// Global middleware
	engine.Use(middleware.RecoveryMiddleware())
	engine.Use(middleware.LoggingMiddleware())
	engine.Use(middleware.CORSMiddleware(middleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
	}))
	engine.Use(middleware.RateLimitMiddleware(float64(cfg.RateLimitRPS), float64(cfg.RateLimitBurst)))

	// Register routes
	handler.SetupRoutes(engine, cfg, router)

	// Start server with graceful shutdown
	addr := cfg.ServerAddr
	srv := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s (mode: %s)", addr, cfg.ServerMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

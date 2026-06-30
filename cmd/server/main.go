package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/neuhis/software-practice-backend/internal/config"
	"github.com/neuhis/software-practice-backend/internal/handler"
	"github.com/neuhis/software-practice-backend/internal/llm"
	"github.com/neuhis/software-practice-backend/internal/middleware"
	"github.com/neuhis/software-practice-backend/internal/repository"
	addresssvc "github.com/neuhis/software-practice-backend/internal/service/address"
	authsvc "github.com/neuhis/software-practice-backend/internal/service/auth"
	billingsvc "github.com/neuhis/software-practice-backend/internal/service/billing"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
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

	// Initialize repositories
	patientRepo := repository.NewPatientRepository(db)
	visitRepo := repository.NewVisitRepository(db)
	timelineRepo := repository.NewTimelineRepository(db)
	flowCardRepo := repository.NewFlowCardRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// Initialize medAgent client
	medAgentClient := medagent.NewClient(cfg.MedAgentBaseURL)
	log.Printf("MedAgent client initialized: %s", cfg.MedAgentBaseURL)

	// Initialize LLM client for title generation (reuses medAgent provider config)
	llmClient := llm.NewFromProvider(cfg.MedAgentProvider, cfg.MedAgentAPIKey, cfg.MedAgentModel, "")
	log.Printf("LLM client initialized: provider=%s model=%s", cfg.MedAgentProvider, cfg.MedAgentModel)

	// Initialize services
	patientSvc := patientsvc.NewService(patientRepo, visitRepo)
	visitSvc := visitsvc.NewService(visitRepo, timelineRepo)
	workbenchSvc := wbsvc.NewService(
		patientRepo, visitRepo, timelineRepo, flowCardRepo, addressRepo,
		medAgentClient, cfg.MedAgentMode, llmClient,
	)
	authSvc := authsvc.NewService(userRepo, refreshTokenRepo, patientRepo, cfg.JWTSecret)
	addressSvc := addresssvc.NewService(addressRepo)
	billingSvc := billingsvc.NewService(visitRepo, flowCardRepo)

	// Initialize handlers
	router := handler.NewRouter(patientSvc, visitSvc, workbenchSvc, authSvc, addressSvc, billingSvc)

	// Create Gin engine
	engine := gin.New()

	// Global middleware
	engine.Use(middleware.RecoveryMiddleware())
	engine.Use(middleware.LoggingMiddleware())
	engine.Use(middleware.CORSMiddleware(middleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		ServerMode:     cfg.ServerMode,
	}))
	engine.Use(middleware.RateLimitMiddleware(10, 20)) // 10 req/s, burst 20

	// Register routes
	handler.SetupRoutes(engine, cfg, router)

	// Start server
	addr := cfg.ServerAddr
	log.Printf("Server starting on %s (mode: %s)", addr, cfg.ServerMode)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
		os.Exit(1)
	}
}

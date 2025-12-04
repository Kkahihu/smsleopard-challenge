package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"smsleopard/internal/config"
	"smsleopard/internal/handler"
	"smsleopard/internal/middleware"
	"smsleopard/internal/repository"
	"smsleopard/internal/service"
)

func main() {
	// Load .env file (ignore error in production)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.GetDatabaseDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database")

	// Initialize repositories
	customerRepo := repository.NewCustomerRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Initialize services
	templateService := service.NewTemplateService()
	campaignService := service.NewCampaignService(
		campaignRepo,
		customerRepo,
		messageRepo,
		templateService,
		db,
	)

	// Initialize handlers
	campaignHandler := handler.NewCampaignHandler(campaignService)
	previewHandler := handler.NewPreviewHandler(campaignService)

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.Recovery)
	router.Use(middleware.Logger)

	// Campaign routes
	router.HandleFunc("/campaigns", campaignHandler.Create).Methods("POST")
	router.HandleFunc("/campaigns", campaignHandler.List).Methods("GET")
	router.HandleFunc("/campaigns/{id:[0-9]+}", campaignHandler.GetByID).Methods("GET")
	router.HandleFunc("/campaigns/{id:[0-9]+}/send", campaignHandler.Send).Methods("POST")

	// Preview route
	router.HandleFunc("/campaigns/{id:[0-9]+}/personalized-preview", previewHandler.Preview).Methods("POST")

	// Health endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"disconnected"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","database":"connected"}`))
	}).Methods("GET")

	// Start server
	port := ":" + cfg.Server.Port
	log.Printf("🚀 API Server starting on port %s", port)
	log.Printf("📍 Health check: http://localhost%s/health", port)
	log.Printf("🌍 Environment: %s", cfg.Env)

	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

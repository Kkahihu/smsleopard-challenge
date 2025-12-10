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
	"smsleopard/internal/queue"
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

	// Connect to RabbitMQ
	rabbitmqURL := cfg.GetRabbitMQURL()

	queueConn, err := queue.NewConnection(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer queueConn.Close()

	// Create publisher
	queueName := "campaign_sends"
	publisher, err := queue.NewPublisher(queueConn, queueName)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}

	log.Println("✅ Connected to RabbitMQ")

	// Initialize repositories
	customerRepo := repository.NewCustomerRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Initialize services
	templateService := service.NewTemplateService()
	healthService := service.NewHealthService(db, rabbitmqURL, "1.0.0")
	campaignService := service.NewCampaignService(
		campaignRepo,
		customerRepo,
		messageRepo,
		templateService,
		publisher,
		db,
	)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(healthService)
	campaignHandler := handler.NewCampaignHandler(campaignService)
	previewHandler := handler.NewPreviewHandler(campaignService)

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.Recovery)
	router.Use(middleware.Logger)

	// Health endpoint (public, no authentication)
	router.HandleFunc("/health", healthHandler.HandleHealth).Methods("GET")

	// Campaign routes
	router.HandleFunc("/campaigns", campaignHandler.Create).Methods("POST")
	router.HandleFunc("/campaigns", campaignHandler.List).Methods("GET")
	router.HandleFunc("/campaigns/{id:[0-9]+}", campaignHandler.GetByID).Methods("GET")
	router.HandleFunc("/campaigns/{id:[0-9]+}/send", campaignHandler.Send).Methods("POST")

	// Preview route
	router.HandleFunc("/campaigns/{id:[0-9]+}/personalized-preview", previewHandler.Preview).Methods("POST")

	// Start server
	port := ":" + cfg.Server.Port
	log.Printf("🚀 API Server starting on port %s", port)
	log.Printf("📍 Health check: http://localhost%s/health", port)
	log.Printf("🌍 Environment: %s", cfg.Env)

	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

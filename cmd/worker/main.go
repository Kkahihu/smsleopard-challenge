package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"smsleopard/internal/config"
	"smsleopard/internal/models"
	"smsleopard/internal/queue"
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

	// Initialize services
	templateSvc := service.NewTemplateService()
	senderSvc := service.NewSenderService(0.95) // 95% success rate
	log.Println("✅ Services initialized")

	// Connect to RabbitMQ
	rabbitmqURL := cfg.GetRabbitMQURL()
	conn, err := queue.NewConnection(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	log.Println("✅ Connected to RabbitMQ")

	// Create message handler
	handler := createMessageHandler(db, templateSvc, senderSvc)

	// Start consumer
	queueName := "campaign_sends"
	consumer, err := queue.NewConsumer(conn, queueName, handler)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	err = consumer.Start()
	if err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
	log.Printf("✅ Worker started, consuming from queue: %s", queueName)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("🛑 Shutting down gracefully...")

	// Stop consumer
	if err := consumer.Stop(); err != nil {
		log.Printf("Error stopping consumer: %v", err)
	}

	// Close connections
	conn.Close()
	db.Close()

	log.Println("✅ Worker stopped")
}

// createMessageHandler creates the message processing handler
func createMessageHandler(db *sql.DB, templateSvc *service.TemplateService, senderSvc *service.SenderService) queue.MessageHandler {
	return func(job *queue.MessageJob) error {
		ctx := context.Background()

		log.Printf("📨 Processing message ID: %d", job.MessageID)

		// Fetch message with campaign and customer
		message, campaign, customer, err := fetchMessageData(ctx, db, job.MessageID)
		if err != nil {
			log.Printf("❌ Failed to fetch message data: %v", err)
			return err
		}

		// Check retry limit
		if message.RetryCount >= 3 {
			log.Printf("⚠️  Message ID %d exceeded retry limit, marking as permanently failed", job.MessageID)
			if err := updateMessagePermanentFailure(ctx, db, job.MessageID); err != nil {
				log.Printf("❌ Failed to update permanent failure: %v", err)
			}
			// Return nil to ACK and remove from queue
			return nil
		}

		// Render template
		rendered, err := templateSvc.Render(campaign.BaseTemplate, customer)
		if err != nil {
			log.Printf("❌ Failed to render template: %v", err)
			updateErr := updateMessageFailure(ctx, db, job.MessageID, err.Error())
			if updateErr != nil {
				log.Printf("❌ Failed to update message failure: %v", updateErr)
			}
			return err
		}

		log.Printf("📝 Rendered message for customer %s: %s", customer.Phone, rendered)

		// Send message
		result := senderSvc.Send(campaign.Channel, customer.Phone, rendered)

		if result.Success {
			// Update as sent
			log.Printf("✅ Message sent successfully to %s (latency: %v)", customer.Phone, result.Latency)
			if err := updateMessageSuccess(ctx, db, job.MessageID); err != nil {
				log.Printf("❌ Failed to update message success: %v", err)
				return err
			}
			return nil
		} else {
			// Update as failed with retry
			errMsg := result.Error.Error()
			log.Printf("❌ Send failed for %s: %s (retry count: %d)", customer.Phone, errMsg, message.RetryCount+1)
			if err := updateMessageFailure(ctx, db, job.MessageID, errMsg); err != nil {
				log.Printf("❌ Failed to update message failure: %v", err)
			}
			return fmt.Errorf("send failed: %s", errMsg)
		}
	}
}

// fetchMessageData fetches message with campaign and customer
func fetchMessageData(ctx context.Context, db *sql.DB, messageID int) (*models.OutboundMessage, *models.Campaign, *models.Customer, error) {
	query := `
		SELECT 
			om.id, om.campaign_id, om.customer_id, om.status, 
			om.rendered_content, om.retry_count, om.created_at, om.updated_at,
			c.id, c.name, c.channel, c.status, c.base_template, c.scheduled_at, c.created_at, c.updated_at,
			cust.id, cust.phone, cust.first_name, cust.last_name, cust.location, cust.preferred_product, cust.created_at
		FROM outbound_messages om
		JOIN campaigns c ON om.campaign_id = c.id
		JOIN customers cust ON om.customer_id = cust.id
		WHERE om.id = $1
	`

	var message models.OutboundMessage
	var campaign models.Campaign
	var customer models.Customer

	err := db.QueryRowContext(ctx, query, messageID).Scan(
		// OutboundMessage fields
		&message.ID,
		&message.CampaignID,
		&message.CustomerID,
		&message.Status,
		&message.RenderedContent,
		&message.RetryCount,
		&message.CreatedAt,
		&message.UpdatedAt,
		// Campaign fields
		&campaign.ID,
		&campaign.Name,
		&campaign.Channel,
		&campaign.Status,
		&campaign.BaseTemplate,
		&campaign.ScheduledAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
		// Customer fields
		&customer.ID,
		&customer.Phone,
		&customer.FirstName,
		&customer.LastName,
		&customer.Location,
		&customer.PreferredProduct,
		&customer.CreatedAt,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch message data: %w", err)
	}

	return &message, &campaign, &customer, nil
}

// updateMessageSuccess updates message as sent
func updateMessageSuccess(ctx context.Context, db *sql.DB, messageID int) error {
	query := `
		UPDATE outbound_messages 
		SET status = 'sent', updated_at = NOW()
		WHERE id = $1
	`

	_, err := db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to update message success: %w", err)
	}

	return nil
}

// updateMessageFailure updates message as failed with retry
func updateMessageFailure(ctx context.Context, db *sql.DB, messageID int, errorMsg string) error {
	query := `
		UPDATE outbound_messages 
		SET status = 'failed', 
			retry_count = retry_count + 1,
			last_error = $2,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := db.ExecContext(ctx, query, messageID, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to update message failure: %w", err)
	}

	return nil
}

// updateMessagePermanentFailure marks message as permanently failed
func updateMessagePermanentFailure(ctx context.Context, db *sql.DB, messageID int) error {
	query := `
		UPDATE outbound_messages 
		SET status = 'failed',
			last_error = 'Exceeded maximum retry attempts (3)',
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to update permanent failure: %w", err)
	}

	return nil
}

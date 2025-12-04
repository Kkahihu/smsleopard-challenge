# Phase 1 Execution Plan - Project Structure & Database Setup

## 🎯 Objective
Set up the foundation: directory structure, database schema, and configuration management.

**Estimated Time:** 1-1.5 hours  
**Status:** Ready to Execute

---

## 📋 Pre-Execution Checklist

- [x] Docker build working
- [x] All fixes applied from FIXES_APPLIED.md
- [x] Go 1.25.5 installed
- [x] Dependencies downloaded (`go mod tidy`)
- [x] Current main.go working as placeholder

---

## 🗂️ Phase 1.1: Create Directory Structure

### Order of Operations
Create directories in this exact order to avoid dependency issues:

```bash
# Step 1: Create main directories
mkdir -p cmd/api
mkdir -p cmd/worker
mkdir -p internal/config
mkdir -p internal/models
mkdir -p internal/repository
mkdir -p internal/service
mkdir -p internal/handler
mkdir -p internal/middleware
mkdir -p internal/queue
mkdir -p migrations
mkdir -p scripts
mkdir -p tests

# Step 2: Verify structure
tree -d -L 2  # or dir /s on Windows
```

### Expected Structure
```
smsleopard/
├── cmd/
│   ├── api/
│   └── worker/
├── internal/
│   ├── config/
│   ├── models/
│   ├── repository/
│   ├── service/
│   ├── handler/
│   ├── middleware/
│   └── queue/
├── migrations/
├── scripts/
└── tests/
```

### Validation
```bash
# All directories should exist
ls -la cmd/api cmd/worker
ls -la internal/config internal/models internal/repository
ls -la migrations scripts tests
```

---

## 📄 Phase 1.2: Create Database Migration Files

### File 1: migrations/001_create_customers.sql

**Content:**
```sql
-- Create customers table
CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    phone VARCHAR(20) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    location VARCHAR(100),
    preferred_product VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for phone lookups
CREATE INDEX IF NOT EXISTS idx_customers_phone ON customers(phone);

-- Add comment for documentation
COMMENT ON TABLE customers IS 'Stores customer information for campaign targeting';
```

**Key Points:**
- ✅ Uses `IF NOT EXISTS` for idempotency
- ✅ All VARCHAR lengths match schema design
- ✅ Index on phone for faster lookups
- ✅ created_at with automatic timestamp

### File 2: migrations/002_create_campaigns.sql

**Content:**
```sql
-- Create campaigns table
CREATE TABLE IF NOT EXISTS campaigns (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('sms', 'whatsapp')),
    status VARCHAR(20) NOT NULL DEFAULT 'draft' 
        CHECK (status IN ('draft', 'scheduled', 'sending', 'sent', 'failed')),
    base_template TEXT NOT NULL,
    scheduled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaigns_channel ON campaigns(channel);

-- Add comments for documentation
COMMENT ON TABLE campaigns IS 'Stores campaign configuration and metadata';
COMMENT ON COLUMN campaigns.base_template IS 'Template with {placeholder} syntax for personalization';
```

**Key Points:**
- ✅ CHECK constraints for channel and status enums
- ✅ Three indexes for filtering and pagination
- ✅ DESC index on created_at for "newest first" ordering
- ✅ NULL allowed for scheduled_at (immediate sends)

### File 3: migrations/003_create_outbound_messages.sql

**Content:**
```sql
-- Create outbound_messages table
CREATE TABLE IF NOT EXISTS outbound_messages (
    id SERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'sent', 'failed')),
    rendered_content TEXT,
    last_error TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_outbound_messages_campaign_id ON outbound_messages(campaign_id);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_status ON outbound_messages(status);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_created_at ON outbound_messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_customer_id ON outbound_messages(customer_id);

-- Add comments for documentation
COMMENT ON TABLE outbound_messages IS 'Tracks individual message deliveries for campaigns';
COMMENT ON COLUMN outbound_messages.retry_count IS 'Number of delivery attempts (max 3)';
COMMENT ON COLUMN outbound_messages.rendered_content IS 'Final personalized message content';
```

**Key Points:**
- ✅ Foreign keys with ON DELETE CASCADE
- ✅ Indexes on campaign_id, customer_id, status for statistics
- ✅ retry_count for retry logic
- ✅ NULL allowed for last_error and rendered_content

### Migration Execution Strategy

**Option A: Docker Init (Recommended for Development)**
- Files in `./migrations/` auto-run via docker-compose volume mount
- Already configured in docker-compose.yml:
  ```yaml
  volumes:
    - ./migrations:/docker-entrypoint-initdb.d
  ```

**Option B: Manual Execution (for Testing)**
```bash
# Connect to database
docker-compose exec db psql -U smsleopard -d smsleopard_db

# Run each migration
\i /docker-entrypoint-initdb.d/001_create_customers.sql
\i /docker-entrypoint-initdb.d/002_create_campaigns.sql
\i /docker-entrypoint-initdb.d/003_create_outbound_messages.sql

# Verify tables
\dt
\d customers
\d campaigns
\d outbound_messages
```

**Option C: Using golang-migrate (Production)**
```bash
# We'll implement this in scripts/migrate.go later
# For now, use Option A or B
```

---

## ⚙️ Phase 1.3: Create Config Management

### File: internal/config/config.go

**Content:**
```go
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Env      string
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "smsleopard"),
			Password: getEnv("POSTGRES_PASSWORD", ""),
			DBName:   getEnv("POSTGRES_DB", "smsleopard_db"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnv("RABBITMQ_PORT", "5672"),
			User:     getEnv("RABBITMQ_DEFAULT_USER", "guest"),
			Password: getEnv("RABBITMQ_DEFAULT_PASS", "guest"),
		},
		Env: getEnv("ENV", "development"),
	}

	// Validate required fields
	if config.Database.Password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
	}

	return config, nil
}

// GetDatabaseDSN returns PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
	)
}

// GetRabbitMQURL returns RabbitMQ connection URL
func (c *Config) GetRabbitMQURL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		c.RabbitMQ.User,
		c.RabbitMQ.Password,
		c.RabbitMQ.Host,
		c.RabbitMQ.Port,
	)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// getEnv gets environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets environment variable as integer or returns default
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
```

**Key Features:**
- ✅ Centralized configuration management
- ✅ Environment variable loading with defaults
- ✅ Helper methods for DSN generation
- ✅ Validation for required fields
- ✅ Type-safe configuration structs

**Testing Config:**
```go
// Quick test in main.go temporarily:
import "smsleopard/internal/config"

cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
log.Printf("Database DSN: %s", cfg.GetDatabaseDSN())
log.Printf("RabbitMQ URL: %s", cfg.GetRabbitMQURL())
```

---

## 🔧 Phase 1.4: Migrate main.go to cmd/ Structure

### File: cmd/api/main.go

**Content:**
```go
package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"smsleopard/internal/config"
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

	// Create router
	router := mux.NewRouter()

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

	// TODO: Add campaign endpoints in Phase 4

	// Start server
	port := ":" + cfg.Server.Port
	log.Printf("🚀 API Server starting on port %s", port)
	log.Printf("📍 Health check: http://localhost%s/health", port)
	log.Printf("🌍 Environment: %s", cfg.Env)

	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

**Key Features:**
- ✅ Uses config package for configuration
- ✅ Database connection with ping verification
- ✅ Working health endpoint with database check
- ✅ Proper error handling and logging
- ✅ Ready for Phase 4 endpoint additions

### File: cmd/worker/main.go

**Content:**
```go
package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"smsleopard/internal/config"
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

	// TODO: Connect to RabbitMQ in Phase 5
	log.Println("⚠️  RabbitMQ connection not implemented yet")

	// TODO: Start message consumer in Phase 5
	log.Println("🔄 Worker started (placeholder mode)")
	log.Println("💤 Waiting for messages...")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("👋 Worker shutting down gracefully")
}
```

**Key Features:**
- ✅ Database connection (for message processing later)
- ✅ Graceful shutdown handling
- ✅ Ready for RabbitMQ connection in Phase 5
- ✅ Placeholder that won't crash

---

## 🐳 Phase 1.5: Update Dockerfile for New Structure

### Updated Dockerfile

**Content:**
```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -o smsleopard-api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -o smsleopard-worker ./cmd/worker

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/smsleopard-api .
COPY --from=builder /app/smsleopard-worker .

EXPOSE 8080

# Default to API (overridden by docker-compose for worker)
CMD ["./smsleopard-api"]
```

**Changes from Previous:**
- ✅ Now builds from `./cmd/api` and `./cmd/worker`
- ✅ No more fallback logic
- ✅ Both binaries properly built
- ✅ Clean separation of concerns

---

## ✅ Phase 1.6: Testing & Validation

### Test 1: Directory Structure
```bash
# Verify all directories exist
ls -R cmd/ internal/ migrations/ scripts/ tests/

# Should show:
# cmd/api, cmd/worker
# internal/config, internal/models, etc.
# migrations/ (with 3 .sql files)
```

### Test 2: Configuration Loading
```bash
# Test config in isolation
cd internal/config
go test -v  # Will need to create config_test.go later

# Quick manual test
go run ../../cmd/api/main.go
# Should see: ✅ Connected to database
```

### Test 3: Database Migrations
```bash
# Option 1: Via Docker (easiest)
docker-compose down -v
docker-compose up db

# Wait for "database system is ready to accept connections"
# Migrations auto-run from ./migrations/

# Verify
docker-compose exec db psql -U smsleopard -d smsleopard_db -c "\dt"
# Should show: customers, campaigns, outbound_messages

# Option 2: Manual verification
docker-compose exec db psql -U smsleopard -d smsleopard_db
\d customers
\d campaigns
\d outbound_messages
\q
```

### Test 4: Docker Build
```bash
# Clean build
docker-compose down
docker-compose build --no-cache

# Should complete without errors
# Both binaries should build successfully
```

### Test 5: Full System Start
```bash
# Start all services
docker-compose up

# Verify logs:
# ✅ PostgreSQL started
# ✅ RabbitMQ started  
# ✅ App: Connected to database
# ✅ Worker: Connected to database

# Test health endpoint
curl http://localhost:8080/health
# Expected: {"status":"healthy","database":"connected"}
```

### Test 6: Service Health Checks
```bash
# Check service status
docker ps

# All services should show:
# STATUS: Up X seconds (healthy)

# If unhealthy, check logs:
docker-compose logs app
docker-compose logs worker
docker-compose logs db
```

---

## 🚨 Common Issues & Solutions

### Issue 1: "directory not found" in Docker build
**Cause:** Directories not created before Docker build  
**Solution:** Create all directories first (Phase 1.1)

### Issue 2: Database connection refused
**Cause:** Database not ready when app starts  
**Solution:** Health checks in docker-compose ensure readiness

### Issue 3: Migration files not found
**Cause:** Volume mount incorrect  
**Solution:** Verify `./migrations:/docker-entrypoint-initdb.d` in docker-compose

### Issue 4: Import cycle detected
**Cause:** Circular imports between packages  
**Solution:** Keep dependencies one-way (config <- repository <- service <- handler)

### Issue 5: "go.mod not found" in Docker
**Cause:** Wrong build context  
**Solution:** Ensure Dockerfile copies go.mod first

---

## 📝 Phase 1 Completion Checklist

- [ ] All directories created (cmd/, internal/, migrations/, scripts/, tests/)
- [ ] Three migration files created and contain correct SQL
- [ ] internal/config/config.go created and tested
- [ ] cmd/api/main.go created with health endpoint
- [ ] cmd/worker/main.go created with graceful shutdown
- [ ] Dockerfile updated to build from cmd/ structure
- [ ] Docker builds successfully without errors
- [ ] All services start and become healthy
- [ ] Health endpoint responds correctly
- [ ] Database tables created (verified with \dt)
- [ ] Old main.go backed up or removed

---

## 🎯 Success Criteria

Phase 1 is complete when:
1. ✅ Directory structure matches plan
2. ✅ Database schema created (3 tables with indexes)
3. ✅ Configuration management working
4. ✅ Both API and Worker build and run
5. ✅ Health checks pass
6. ✅ No compilation errors
7. ✅ Ready for Phase 2 (Models & Repositories)

---

## 📊 Phase 1 Metrics

- **Files Created:** 7 (3 migrations + 1 config + 2 mains + 1 Dockerfile)
- **Directories Created:** 10
- **Database Tables:** 3 with 7 indexes
- **Lines of Code:** ~300
- **Time Estimate:** 1-1.5 hours

---

## 🚀 Next Phase Preview

**Phase 2** will implement:
- Data models (Customer, Campaign, OutboundMessage)
- Repository layer with database operations
- CRUD operations for all entities

**Dependencies:** Phase 1 must be 100% complete and tested.

---

**Ready to execute? Switch to Code mode and let's build Phase 1!** 🛠️
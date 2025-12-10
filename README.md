# SMSLeopard Challenge - Campaign Management System

[![Go Version](https://img.shields.io/badge/Go-1.25.5-blue.svg)](https://golang.org/dl/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-blue.svg)](https://www.postgresql.org/)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-3-orange.svg)](https://www.rabbitmq.com/)
[![Docker](https://img.shields.io/badge/Docker-Compose-blue.svg)](https://docs.docker.com/compose/)

A robust campaign management system built with Go, featuring asynchronous message processing, template personalization, and multi-channel support (SMS/WhatsApp).

## 🎯 Key Features

- ✅ **Async Message Processing** - RabbitMQ-based queue for scalable message delivery
- ✅ **Template Personalization** - Dynamic field replacement with customer data
- ✅ **Multi-Channel Support** - SMS and WhatsApp campaigns
- ✅ **RESTful API** - Clean API design with pagination and filtering
- ✅ **Worker Architecture** - Separate worker service for message processing
- ✅ **PostgreSQL Storage** - Reliable data persistence with 17-alpine
- ✅ **Migration Management** - Go-based migration runner with version tracking
- ✅ **Comprehensive Testing** - Unit tests, integration tests, and mocks
- ✅ **Docker Compose** - Full stack orchestration for development

---

## 📋 Prerequisites

Before you begin, ensure you have the following installed:

- **Docker & Docker Compose** - For containerized deployment
- **Go 1.21+** - For local development (optional)
- **Git** - For version control

The application uses:
- PostgreSQL 17 (via Docker)
- RabbitMQ 3 (via Docker)

---

## 🚀 Quick Start

Get up and running in 3 simple steps:

### 1. Clone and Setup

```bash
# Clone the repository
git clone <repository-url>
cd Challenge

# Copy environment variables
cp .env.example .env

# (Optional) Edit .env file with your settings
```

### 2. Build Services

```bash
# Build all Docker images
docker-compose build
```

### 3. Start Services

```bash
# Start all services (database, RabbitMQ, API, worker)
docker-compose up -d

# Check service status
docker-compose ps

# View logs (optional)
docker-compose logs -f app worker
```

Your services are now running:
- **API Server**: http://localhost:8080
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **PostgreSQL**: localhost:5432

---

## 🗄️ Database Setup

### Running Migrations

The project provides **three methods** for running database migrations:

#### Method 1: Docker Auto-Initialization (Recommended for First Time)

On first container startup, PostgreSQL automatically runs schema migrations from `migrations/*.sql`:

```bash
# Start database (migrations run automatically)
docker-compose up -d db

# Verify tables were created
docker-compose exec db psql -U smsleopard -d smsleopard_db -c "\dt"
```

#### Method 2: Using migrate.go (Recommended for Development)

The [`migrate.go`](scripts/migrate.go) script provides full migration management:

```bash
# Apply all pending migrations
go run scripts/migrate.go up

# Check migration status
go run scripts/migrate.go status

# Rollback last migration
go run scripts/migrate.go down

# Reset all migrations (⚠️ destroys data)
go run scripts/migrate.go reset

# Show help
go run scripts/migrate.go help
```

#### Method 3: Manual psql Execution

```bash
# Connect to database
docker-compose exec db psql -U smsleopard -d smsleopard_db

# Or run migrations manually
docker-compose exec db psql -U smsleopard -d smsleopard_db -f /docker-entrypoint-initdb.d/001_create_customers.sql
```

### Seeding Data

The project provides **two methods** for seeding test data:

#### Method 1: Using seed.go (Programmatic, Flexible)

Generate varied test data programmatically:

```bash
# Seed with defaults (12 customers, 3 campaigns)
go run scripts/seed.go

# Custom counts
go run scripts/seed.go -customers=20 -campaigns=5

# Clear and reseed
go run scripts/seed.go -clear -customers=50

# Show help
go run scripts/seed.go -help
```

**Features:**
- Phone pattern: `+2547000010XXX`
- Realistic Kenyan names and locations
- Varied NULL fields for edge case testing
- Idempotent with `ON CONFLICT DO NOTHING`

#### Method 2: Using SQL Seed Files (Curated Dataset)

Load curated test data from SQL files:

```bash
# Run all seed migrations
go run scripts/migrate.go seed
```

**Seed files:**
- [`migrations/seed/001_customers.sql`](migrations/seed/001_customers.sql) - 15 test customers
- [`migrations/seed/002_campaigns.sql`](migrations/seed/002_campaigns.sql) - 3 test campaigns

#### Verify Seeded Data

```bash
# Check customer count
docker-compose exec db psql -U smsleopard -d smsleopard_db -c "SELECT COUNT(*) FROM customers;"

# Check campaigns
docker-compose exec db psql -U smsleopard -d smsleopard_db -c "SELECT id, name, channel, status FROM campaigns;"

# Check for varied data
docker-compose exec db psql -U smsleopard -d smsleopard_db -c "SELECT phone, first_name, last_name, location FROM customers LIMIT 10;"
```

---

## 🏃 Running the Application

### Docker Compose Method (Recommended)

Run the entire stack with one command:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f app worker

# Stop all services
docker-compose down

# Stop and remove volumes (⚠️ destroys data)
docker-compose down -v
```

### Manual Method (Local Development)

For development with live reloading:

```bash
# Terminal 1: Start infrastructure
docker-compose up -d db rabbitmq

# Terminal 2: Run API server
go run cmd/api/main.go

# Terminal 3: Run worker
go run cmd/worker/main.go
```

### Environment Variables

Key environment variables in [`.env`](.env):

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | API server port | `8080` |
| `POSTGRES_HOST` | PostgreSQL host | `db` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_USER` | Database user | `smsleopard` |
| `POSTGRES_PASSWORD` | Database password | `secret123` |
| `POSTGRES_DB` | Database name | `smsleopard_db` |
| `RABBITMQ_HOST` | RabbitMQ host | `rabbitmq` |
| `RABBITMQ_PORT` | RabbitMQ port | `5672` |
| `RABBITMQ_DEFAULT_USER` | RabbitMQ user | `guest` |
| `RABBITMQ_DEFAULT_PASS` | RabbitMQ password | `guest` |

---

## 🏥 Health Endpoint

The application provides a health check endpoint for monitoring system status and dependencies.

### Endpoint

**GET /health**

Returns the health status of the application and its dependencies (PostgreSQL and RabbitMQ).

### Response Examples

#### Healthy (HTTP 200)

All services are operational:

```json
{
  "status": "healthy",
  "services": {
    "database": "connected",
    "queue": "connected"
  },
  "timestamp": "2025-12-10T16:00:00Z",
  "version": "1.0.0"
}
```

#### Degraded (HTTP 503)

Non-critical services are down, but database is operational:

```json
{
  "status": "degraded",
  "services": {
    "database": "connected",
    "queue": "disconnected"
  },
  "timestamp": "2025-12-10T16:00:00Z",
  "version": "1.0.0"
}
```

#### Unhealthy (HTTP 503)

Critical services (database) are down:

```json
{
  "status": "unhealthy",
  "services": {
    "database": "disconnected",
    "queue": "connected"
  },
  "timestamp": "2025-12-10T16:00:00Z",
  "version": "1.0.0"
}
```

### Status Levels

- **`healthy`**: All services operational (HTTP 200)
- **`degraded`**: Non-critical services (queue) down, database up (HTTP 503)
- **`unhealthy`**: Critical services (database) down (HTTP 503)

### Usage Examples

```bash
# Check health
curl http://localhost:8080/health

# Check with full response including status code
curl -i http://localhost:8080/health
```

### Docker Integration

The health endpoint is used by Docker Compose for container health checks:

```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

---

## 📡 API Endpoints

### Campaigns

```http
# List campaigns (with pagination)
GET /campaigns?page=1&limit=10&status=sent&channel=sms

# Get single campaign
GET /campaigns/:id

# Create campaign
POST /campaigns
Content-Type: application/json

{
  "name": "Weekend Sale",
  "channel": "sms",
  "status": "draft",
  "template": "Hi {{first_name}}, special offer just for you!",
  "scheduled_at": "2024-12-15T10:00:00Z"
}

# Update campaign
PUT /campaigns/:id

# Delete campaign
DELETE /campaigns/:id

# Send campaign
POST /campaigns/:id/send
```

### Preview

```http
# Preview campaign with customer data
POST /preview
Content-Type: application/json

{
  "template": "Hi {{first_name}} from {{location}}!",
  "customer_id": 1
}
```

### Query Parameters

- `page` - Page number (default: 1)
- `limit` - Items per page (default: 10, max: 100)
- `status` - Filter by status (draft, scheduled, sent, failed)
- `channel` - Filter by channel (sms, whatsapp)

For detailed API documentation, see the [API Guide](docs/API_GUIDE.md) (if available).

---

## 🧪 Testing

### Run All Tests

```bash
# Run all tests with coverage
go test ./... -v -cover

# Run tests with race detection
go test ./... -race

# Run specific test file
go test ./tests/template_test.go -v
```

### Test Categories

- **Unit Tests** - [`tests/template_test.go`](tests/template_test.go), [`tests/mocks.go`](tests/mocks.go)
- **Integration Tests** - [`tests/api_test.go`](tests/api_test.go), [`tests/worker_test.go`](tests/worker_test.go)
- **Feature Tests** - [`tests/pagination_test.go`](tests/pagination_test.go), [`tests/preview_test.go`](tests/preview_test.go)

For comprehensive testing guide, see [`docs/TESTING_GUIDE.md`](docs/TESTING_GUIDE.md).

---

## 📂 Project Structure

```
Challenge/
├── cmd/                          # Application entry points
│   ├── api/                      # API server
│   │   └── main.go
│   └── worker/                   # Background worker
│       └── main.go
├── internal/                     # Internal packages
│   ├── config/                   # Configuration management
│   ├── handler/                  # HTTP handlers
│   ├── middleware/               # HTTP middleware
│   ├── models/                   # Data models
│   ├── queue/                    # RabbitMQ integration
│   ├── repository/               # Database layer
│   └── service/                  # Business logic
├── migrations/                   # Database migrations
│   ├── 001_create_customers.sql
│   ├── 002_create_campaigns.sql
│   ├── 003_create_outbound_messages.sql
│   └── seed/                     # Seed data
│       ├── 001_customers.sql
│       └── 002_campaigns.sql
├── scripts/                      # Utility scripts
│   ├── migrate.go                # Migration runner
│   ├── seed.go                   # Data seeder
│   └── README.md                 # Scripts documentation
├── tests/                        # Test files
├── docs/                         # Documentation
├── docker-compose.yml            # Docker orchestration
├── Dockerfile                    # Container definition
├── go.mod                        # Go dependencies
└── README.md                     # This file
```

---

## 🔧 Development Workflow

### Adding New Migrations

1. Create a new migration file in [`migrations/`](migrations/):
   ```bash
   # Create migration file (follow naming convention)
   touch migrations/004_add_user_preferences.sql
   ```

2. Write SQL DDL:
   ```sql
   -- migrations/004_add_user_preferences.sql
   CREATE TABLE user_preferences (
       id SERIAL PRIMARY KEY,
       customer_id INT REFERENCES customers(id),
       preference_key VARCHAR(50),
       preference_value TEXT
   );
   ```

3. Apply migration:
   ```bash
   go run scripts/migrate.go up
   ```

### Seeding Data for Development

```bash
# Quick reseed during development
go run scripts/seed.go -clear -customers=100 -campaigns=10

# Or use SQL seeds for consistent test data
go run scripts/migrate.go seed
```

### Running Tests Locally

```bash
# Before committing, run tests
go test ./... -v

# Check test coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Debugging

```bash
# View API logs
docker-compose logs -f app

# View worker logs
docker-compose logs -f worker

# View database logs
docker-compose logs -f db

# View RabbitMQ logs
docker-compose logs -f rabbitmq

# Connect to database
docker-compose exec db psql -U smsleopard -d smsleopard_db

# Check RabbitMQ queues
# Visit http://localhost:15672 (guest/guest)
```

---

## 🔍 Troubleshooting

### Database Connection Issues

**Problem**: `connection refused` errors

```bash
# Check if database is running
docker-compose ps db

# Restart database
docker-compose restart db

# Check database logs
docker-compose logs db
```

**Problem**: `password authentication failed`

```bash
# Verify .env credentials match docker-compose.yml
cat .env

# Recreate containers with fresh credentials
docker-compose down -v
docker-compose up -d
```

### Port Conflicts

**Problem**: `port is already allocated`

```bash
# Check what's using the port
netstat -ano | findstr :8080  # Windows
lsof -i :8080                 # Linux/Mac

# Change port in .env or docker-compose.yml
# Or stop conflicting service
```

### Migration Errors

**Problem**: Migrations fail to apply

```bash
# Check migration status
go run scripts/migrate.go status

# Rollback problematic migration
go run scripts/migrate.go down

# Reset all migrations (⚠️ destroys data)
go run scripts/migrate.go reset
go run scripts/migrate.go up
```

### RabbitMQ Connection Issues

**Problem**: Worker can't connect to RabbitMQ

```bash
# Check RabbitMQ health
docker-compose ps rabbitmq
docker-compose logs rabbitmq

# Verify RabbitMQ is ready
docker-compose exec rabbitmq rabbitmq-diagnostics ping

# Restart RabbitMQ
docker-compose restart rabbitmq
```

### Worker Not Processing Messages

**Problem**: Messages queued but not sent

```bash
# Check worker logs
docker-compose logs -f worker

# Verify queue exists
# Visit http://localhost:15672 → Queues tab

# Restart worker
docker-compose restart worker
```

### Docker Build Failures

**Problem**: `go: cannot find module` during build

```bash
# Clear Docker build cache
docker-compose build --no-cache

# Ensure go.mod is in sync
go mod tidy
go mod download
```

---

## 📚 Documentation Links

- **System Overview**: [`SYSTEM_OVERVIEW.md`](SYSTEM_OVERVIEW.md) - Comprehensive technical documentation
- **Implementation Plan**: [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md)
- **Testing Guide**: [`docs/TESTING_GUIDE.md`](docs/TESTING_GUIDE.md)
- **Scripts Documentation**: [`scripts/README.md`](scripts/README.md)

---

## 🛠️ Tech Stack

### Backend
- **Go 1.25.5** - Primary language
- **gorilla/mux** - HTTP router
- **lib/pq** - PostgreSQL driver
- **amqp091-go** - RabbitMQ client
- **godotenv** - Environment management

### Infrastructure
- **PostgreSQL 17-alpine** - Database
- **RabbitMQ 3-management-alpine** - Message queue
- **Docker & Docker Compose** - Containerization

### Testing
- **Go testing** - Native test framework
- **go-sqlmock** - Database mocking

---

## 🎯 Design Decisions & Implementation Notes

This project makes several key architectural and implementation decisions. For comprehensive technical documentation including:

- **System Architecture**: Data models, request flows, and component interactions
- **Template Personalization**: NULL field handling strategy and rendering logic
- **Message Queue Design**: RabbitMQ implementation with retry logic and acknowledgments
- **Pagination Strategy**: Offset-based approach with stable ordering
- **Worker Processing**: Asynchronous message handling and retry mechanisms
- **Health Monitoring**: Service health checks and status reporting
- **Testing Approach**: Unit, integration, and end-to-end testing strategies

Please refer to the **[System Overview](SYSTEM_OVERVIEW.md)** document, which provides detailed explanations of:

1. **Data Model & Architecture** - ERD, status enumerations, and indexes
2. **Request Flow** - Complete `/campaigns/{id}/send` processing pipeline
3. **Queue Worker Processing** - Retry logic, acknowledgments, and error handling
4. **Pagination Strategy** - Implementation and performance considerations
5. **Personalization System** - Template syntax and NULL field handling
6. **Technology Choices** - Why PostgreSQL, RabbitMQ, and Go were selected
7. **Quality Assurance** - Testing strategy and observability features

### Quick Reference: Key Decisions

- **NULL Field Handling**: Empty strings for graceful degradation (see [SYSTEM_OVERVIEW.md#52](SYSTEM_OVERVIEW.md#52-null-field-handling-strategy))
- **Queue Choice**: RabbitMQ for durable messaging with built-in retries (see [SYSTEM_OVERVIEW.md#62](SYSTEM_OVERVIEW.md#62-technology-choices))
- **Mock Sender**: 95% success rate simulation for testing (see [`internal/service/sender_service.go`](internal/service/sender_service.go))
- **Pagination**: `ORDER BY id DESC` for stable, predictable results (see [SYSTEM_OVERVIEW.md#4](SYSTEM_OVERVIEW.md#4-pagination-strategy))
- **Health Monitoring**: `/health` endpoint with dependency checks (see [Health Endpoint](#-health-endpoint))

---

## 📄 License

This project is part of the SMSLeopard technical challenge.

---

## 📧 Contact

For questions or issues, please refer to the project documentation or create an issue in the repository.

---

## 🙏 Acknowledgments

Built with ❤️, ☕, and an unhealthy amount of caffeine for the SMSLeopard technical challenge.

**Special Thanks To:**
- 🐹 **Go Gophers** - For making concurrency less scary than JavaScript promises
- 🐰 **RabbitMQ Bunnies** - For hopping messages around reliably
- 🐘 **PostgreSQL Elephants** - For never forgetting my data (unlike me with variable names)
- 🤖 **Claude (Anthropic)** - For being my rubber duck that actually talks back
- ☕ **Coffee** - The real MVP of this challenge
- 🌙 **Night Mode** - For protecting my eyes during those late-night debugging sessions
- 🎵 **Lo-fi Beats** - For keeping the coding vibes immaculate
- 🦁 **The SMSLeopard Team** - For creating such a fun and comprehensive challenge!

> _"In the beginning, there was nothing... then `go run main.go` said 'Let there be logs!'"_ 📜

**P.S.** If you're reading this, SMSLeopard team, I hope you enjoyed this journey as much as I did. No leopards were harmed in the making of this project, but several bugs met their untimely demise. 🐛💀

**Happy Coding! 🚀 May your builds always be green and your queues never blocked!**
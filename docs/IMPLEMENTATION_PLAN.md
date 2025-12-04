# SMSLeopard Challenge - Project Analysis & Implementation Breakdown

## Current Project State

**Existing Setup:**
- Go modules configured with dependencies: `go.mod`
  - gorilla/mux (HTTP routing)
  - godotenv (environment configuration)
  - lib/pq (PostgreSQL driver)
  - rabbitmq/amqp091-go (RabbitMQ client)
- `docker-compose.yml` with PostgreSQL and RabbitMQ services
- `Dockerfile` for containerization
- Basic `main.go` skeleton

**Missing Components:** Database schema, API handlers, business logic, queue worker, tests, migrations, seed data, documentation

---

## Recommended Directory Structure

```
smsleopard/
├── cmd/
│   ├── api/              # HTTP API server
│   │   └── main.go
│   └── worker/           # Queue worker process
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   │   └── config.go
│   ├── models/           # Data models (Customer, Campaign, OutboundMessage)
│   │   ├── customer.go
│   │   ├── campaign.go
│   │   └── message.go
│   ├── repository/       # Database access layer
│   │   ├── customer_repository.go
│   │   ├── campaign_repository.go
│   │   └── message_repository.go
│   ├── service/          # Business logic
│   │   ├── campaign_service.go
│   │   ├── template_service.go
│   │   └── sender_service.go (mock)
│   ├── handler/          # HTTP handlers
│   │   ├── campaign_handler.go
│   │   └── preview_handler.go
│   ├── queue/            # Queue operations
│   │   ├── publisher.go
│   │   └── consumer.go
│   └── middleware/       # HTTP middleware (error handling, etc.)
│       └── error_handler.go
├── migrations/           # Database migrations
│   ├── 001_create_customers.sql
│   ├── 002_create_campaigns.sql
│   └── 003_create_outbound_messages.sql
├── scripts/              # Helper scripts
│   └── seed.go          # Seed data generator
├── tests/               # Integration tests
│   ├── api_test.go
│   ├── worker_test.go
│   └── template_test.go
├── docs/
│   ├── challenge.md
│   ├── IMPLEMENTATION_PLAN.md (this file)
│   ├── SYSTEM_OVERVIEW.md (to create)
│   └── API.md (optional reference)
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── .env.example
├── .env
└── README.md (to update)
```

---

## Detailed Work Breakdown

### **Phase 1: Project Structure & Database Setup**
**Estimated Time: 1-1.5 hours**

**Tasks:**
1. Create directory structure as outlined above
2. Move `main.go` to `cmd/api/main.go` and create `cmd/worker/main.go`
3. Create database migration files:
   - `migrations/001_create_customers.sql` - customers table with indexes
   - `migrations/002_create_campaigns.sql` - campaigns table with indexes
   - `migrations/003_create_outbound_messages.sql` - messages table with foreign keys and indexes
4. Create `internal/config/config.go` for environment variable management
5. Implement migration runner or use a tool (golang-migrate recommended)

**Key Considerations:**
- Add indexes on: `campaigns.status`, `campaigns.created_at`, `outbound_messages.campaign_id`, `outbound_messages.status`, `outbound_messages.created_at`
- Use proper foreign key constraints with `ON DELETE CASCADE`

**Files to Create:**
- `cmd/api/main.go`
- `cmd/worker/main.go`
- `internal/config/config.go`
- `migrations/001_create_customers.sql`
- `migrations/002_create_campaigns.sql`
- `migrations/003_create_outbound_messages.sql`
- `migrations/000_init.sql` (optional - for migration tracking)

---

### **Phase 2: Data Models & Repository Layer**
**Estimated Time: 1.5-2 hours**

**Tasks:**
1. **Models** (`internal/models/`)
   - Define `Customer` struct with JSON tags
   - Define `Campaign` struct with status enum validation
   - Define `OutboundMessage` struct with status transitions
   
2. **Repository Layer** (`internal/repository/`)
   - `customer_repository.go`: CRUD operations for customers
   - `campaign_repository.go`:
     - Create, Get, List with pagination and filtering
     - Update status
     - Get statistics (aggregate message counts)
   - `message_repository.go`:
     - Create batch messages
     - Update message status
     - Get by ID with campaign and customer data

**Key Considerations:**
- Use `database/sql` with prepared statements
- Implement connection pooling properly
- Handle NULL values for optional fields (scheduled_at, last_error)
- Pagination: Use OFFSET/LIMIT with ORDER BY id DESC for consistency

**Files to Create:**
- `internal/models/customer.go`
- `internal/models/campaign.go`
- `internal/models/message.go`
- `internal/repository/customer_repository.go`
- `internal/repository/campaign_repository.go`
- `internal/repository/message_repository.go`
- `internal/repository/repository.go` (base interfaces/structs)

---

### **Phase 3: Core Business Logic & Services**
**Estimated Time: 1.5-2 hours**

**Tasks:**
1. **Template Service** (`internal/service/template_service.go`)
   - Render templates with placeholder substitution `{field_name}`
   - Handle missing/null fields (decide strategy: empty string, keep placeholder, or error)
   - Support all customer fields: first_name, last_name, location, preferred_product
   
2. **Campaign Service** (`internal/service/campaign_service.go`)
   - CreateCampaign: validation, status management
   - SendCampaign: validation, create outbound messages, queue jobs
   - ListCampaigns: apply filters, pagination
   - GetCampaignDetails: fetch with statistics
   - Preview: render template for specific customer
   
3. **Mock Sender Service** (`internal/service/sender_service.go`)
   - Simulate SMS/WhatsApp sending
   - Implement 90-95% success rate (use random number)
   - Return error for failed sends

**Key Considerations:**
- Transaction management for SendCampaign (atomic DB writes + queue publish)
- Template rendering must be tested thoroughly
- Document mock sender behavior clearly

**Files to Create:**
- `internal/service/template_service.go`
- `internal/service/campaign_service.go`
- `internal/service/sender_service.go`
- `internal/service/errors.go` (custom error types)

---

### **Phase 4: API Endpoints Implementation**
**Estimated Time: 2-2.5 hours**

**Tasks:**
1. **HTTP Handlers** (`internal/handler/`)
   - `POST /campaigns` - CreateCampaign handler
   - `POST /campaigns/{id}/send` - SendCampaign handler
   - `GET /campaigns` - ListCampaigns handler with query params
   - `GET /campaigns/{id}` - GetCampaignDetails handler
   - `POST /campaigns/{id}/personalized-preview` - Preview handler

2. **Middleware** (`internal/middleware/`)
   - Error handler for consistent JSON error responses
   - Request logging
   - Panic recovery

3. **Router Setup** (`cmd/api/main.go`)
   - Configure gorilla/mux router
   - Register all routes
   - Add middleware
   - Start HTTP server

**Key Considerations:**
- Proper HTTP status codes (200, 201, 400, 404, 500)
- Consistent error response format:
  ```json
  {
    "error": {
      "code": "CAMPAIGN_NOT_FOUND",
      "message": "Campaign with ID 999 not found"
    }
  }
  ```
- Request validation (required fields, valid enums)
- Return proper JSON responses

**Files to Create:**
- `internal/handler/campaign_handler.go`
- `internal/handler/preview_handler.go`
- `internal/handler/response.go` (response helpers)
- `internal/middleware/error_handler.go`
- `internal/middleware/logger.go`
- `internal/middleware/recovery.go`

---

### **Phase 5: Queue Worker Implementation**
**Estimated Time: 2-2.5 hours**

**Tasks:**
1. **Queue Publisher** (`internal/queue/publisher.go`)
   - Connect to RabbitMQ
   - Publish message jobs to `campaign_sends` queue
   - Handle connection failures
   
2. **Queue Consumer** (`internal/queue/consumer.go`)
   - Connect to RabbitMQ
   - Consume from `campaign_sends` queue
   - Process messages with acknowledgement
   - Implement retry logic (max 3 retries)
   
3. **Worker Main** (`cmd/worker/main.go`)
   - Initialize database connection
   - Initialize queue consumer
   - Process jobs:
     - Fetch outbound_message with campaign and customer
     - Render personalized message
     - Call mock sender
     - Update message status
     - Handle errors and retries

**Key Considerations:**
- Use manual acknowledgement (ack after DB update)
- Implement exponential backoff for retries (optional but recommended)
- Avoid infinite requeue loops (check retry_count)
- Graceful shutdown handling
- Document RabbitMQ choice reasoning (vs Redis)

**Files to Create:**
- `internal/queue/publisher.go`
- `internal/queue/consumer.go`
- `internal/queue/connection.go` (RabbitMQ connection management)
- `cmd/worker/main.go`

---

### **Phase 6: Testing Suite**
**Estimated Time: 2-3 hours**

**Required Tests (as per challenge.md):**

1. **Template Rendering Tests** (`tests/template_test.go`)
   - Test placeholder substitution with all fields
   - Test with missing/null fields
   - Test multiple combinations
   - Edge cases (empty template, no placeholders)
   
2. **Pagination Tests** (`tests/pagination_test.go`)
   - Create >40 campaigns (2+ pages)
   - Test no duplicates across pages
   - Test consistent ordering
   - Test channel and status filters
   
3. **Worker Logic Tests** (`tests/worker_test.go`)
   - Mock queue interface
   - Test successful message processing
   - Test failure scenarios
   - Test retry logic
   - Verify status updates
   
4. **Preview Endpoint Tests** (`tests/preview_test.go`)
   - Test rendering for different customers
   - Test override_template parameter
   - Test with missing customer

**Additional Tests:**
- Integration tests for API endpoints
- Repository layer tests with test database

**Files to Create:**
- `tests/template_test.go`
- `tests/pagination_test.go`
- `tests/worker_test.go`
- `tests/preview_test.go`
- `tests/api_test.go`
- `tests/helpers.go` (test utilities)
- `tests/mocks.go` (mock implementations)

---

### **Phase 7: Seed Data & Migrations**
**Estimated Time: 1 hour**

**Tasks:**
1. Create `scripts/seed.go` to generate:
   - At least 10 customers with varied data
   - 2-3 campaigns (mix of statuses)
   - Some outbound messages
   
2. Document how to run migrations
3. Document how to seed data
4. Update `docker-compose.yml` if needed for init scripts

**Key Considerations:**
- Realistic test data (real-looking names, products, locations)
- Include edge cases (null fields, special characters)
- Make seed data idempotent

**Files to Create:**
- `scripts/seed.go`
- `scripts/migrate.go` (optional migration runner)
- `migrations/seed/001_customers.sql` (optional SQL seed)
- `migrations/seed/002_campaigns.sql` (optional SQL seed)

---

### **Phase 8: Extra Feature Implementation**
**Estimated Time: 1-2 hours**

**Recommended Features (pick ONE):**

1. **Health Endpoint** (Easiest, most practical) ⭐ RECOMMENDED
   - `GET /health` endpoint
   - Check PostgreSQL connectivity
   - Check RabbitMQ connectivity
   - Return JSON with service statuses
   - Example:
     ```json
     {
       "status": "healthy",
       "services": {
         "database": "connected",
         "queue": "connected"
       },
       "timestamp": "2025-06-01T10:00:00Z"
     }
     ```
   
2. **Idempotency for SendCampaign** (Good for reliability)
   - Generate idempotency keys
   - Prevent duplicate message creation
   - Handle concurrent requests
   - Store idempotency keys with expiration
   
3. **Scheduled Dispatch** (Aligns with schema)
   - Background goroutine checks scheduled campaigns
   - Automatically sends when scheduled_at is reached
   - Update status appropriately
   - Use ticker for periodic checks
   
4. **Enhanced Retry with Exponential Backoff**
   - Implement exponential backoff in worker
   - Dead letter queue for permanent failures
   - Configurable retry policy
   - Monitoring/alerting for DLQ

**Files to Create (Health Endpoint example):**
- `internal/handler/health_handler.go`
- `internal/service/health_service.go`

---

### **Phase 9: Documentation**
**Estimated Time: 1.5-2 hours**

**Required Documents:**

1. **`SYSTEM_OVERVIEW.md`** (max 2 pages)
   Must include:
   - Data model diagram with relationships
   - Request flow diagram for `POST /campaigns/{id}/send`
   - Queue worker processing flow with retry logic
   - Pagination strategy explanation
   - Personalization approach and extension points
   
2. **`README.md`** (update existing)
   Must include:
   - Quick start instructions
   - How to run services (docker-compose commands)
   - How to run tests
   - Assumptions made
   - Template handling for null fields
   - Mock sender behavior
   - Queue choice reasoning (RabbitMQ vs Redis)
   - Extra feature description
   - Time spent and AI tool usage
   
3. **API Documentation** (optional but helpful)
   - Request/response examples for each endpoint
   - Error codes reference
   - Postman collection or OpenAPI spec

**Files to Create/Update:**
- `docs/SYSTEM_OVERVIEW.md`
- `README.md` (update)
- `docs/API.md` (optional)

---

### **Phase 10: Final Integration & Testing**
**Estimated Time: 1-1.5 hours**

**Tasks:**
1. End-to-end testing:
   - Start all services with docker-compose
   - Create a campaign via API
   - Send to customers
   - Verify messages processed by worker
   - Check campaign statistics
   
2. Code cleanup:
   - Remove unused imports
   - Format code (`go fmt ./...`)
   - Run linter (`golangci-lint run`)
   - Check error handling everywhere
   
3. Final verification:
   - All required tests pass
   - All API endpoints work correctly
   - Worker processes messages reliably
   - Documentation is complete
   - Extra feature works

**Checklist:**
- [ ] All migrations run successfully
- [ ] Seed data loads without errors
- [ ] API endpoints return expected responses
- [ ] Worker processes messages from queue
- [ ] All tests pass
- [ ] Error handling is comprehensive
- [ ] Documentation is complete and accurate
- [ ] Docker compose starts all services
- [ ] Health endpoint (if implemented) works
- [ ] Code is formatted and linted

---

## Implementation Strategy

**Recommended Order:**
1. Start with database and models (foundation)
2. Build repository layer (data access)
3. Implement services (business logic)
4. Create API handlers (interface)
5. Build queue worker (async processing)
6. Write tests (validation)
7. Add seed data (testing support)
8. Implement extra feature (differentiation)
9. Write documentation (communication)
10. Final testing (quality assurance)

**Time Allocation:**
- Core functionality: 4-5 hours
- Testing: 2-3 hours
- Extra feature + docs: 2-3 hours
- **Total: 8-11 hours** (within recommended 4-8 hours if efficient)

**Tips for Efficiency:**
- Use AI tools for boilerplate code generation
- Focus on correctness over optimization initially
- Write tests alongside implementation (TDD approach)
- Keep documentation updated as you build
- Commit frequently with clear messages

---

## Critical Success Factors

Based on evaluation criteria (from challenge.md):

1. **Code Quality (30%)**: Clean project structure, idiomatic Go, separation of concerns
2. **API Design (25%)**: RESTful patterns, proper status codes, edge case handling
3. **Data Modeling (15%)**: Appropriate schema, relationships, indexes
4. **Queue/Worker (15%)**: Reliability, error handling, no message loss
5. **Tests (10%)**: Coverage of critical paths, quality tests
6. **Documentation (5%)**: Clear explanations, good README

---

## Database Schema Reference

### Customers Table
```sql
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    phone VARCHAR(20) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    location VARCHAR(100),
    preferred_product VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_customers_phone ON customers(phone);
```

### Campaigns Table
```sql
CREATE TABLE campaigns (
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

CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_created_at ON campaigns(created_at DESC);
CREATE INDEX idx_campaigns_channel ON campaigns(channel);
```

### Outbound Messages Table
```sql
CREATE TABLE outbound_messages (
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

CREATE INDEX idx_outbound_messages_campaign_id ON outbound_messages(campaign_id);
CREATE INDEX idx_outbound_messages_status ON outbound_messages(status);
CREATE INDEX idx_outbound_messages_created_at ON outbound_messages(created_at DESC);
```

---

## API Endpoints Quick Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/campaigns` | Create new campaign |
| POST | `/campaigns/{id}/send` | Send campaign to customers |
| GET | `/campaigns` | List campaigns with pagination |
| GET | `/campaigns/{id}` | Get campaign details with stats |
| POST | `/campaigns/{id}/personalized-preview` | Preview personalized message |
| GET | `/health` | Health check (extra feature) |

---

## Testing Checklist

### Unit Tests
- [ ] Template service renders placeholders correctly
- [ ] Template service handles null fields gracefully
- [ ] Campaign service validates input correctly
- [ ] Mock sender returns expected results

### Integration Tests
- [ ] API creates campaigns successfully
- [ ] API sends campaigns and queues messages
- [ ] API lists campaigns with pagination
- [ ] API filters campaigns by status/channel
- [ ] Worker processes messages correctly
- [ ] Worker updates message status
- [ ] Worker handles failures and retries

### End-to-End Tests
- [ ] Complete flow: create → send → process → verify
- [ ] Multiple customers receive personalized messages
- [ ] Failed messages are retried
- [ ] Campaign statistics are accurate

---

## Common Pitfalls to Avoid

1. **Database Connections**: Don't create new connection per request
2. **Queue Acknowledgement**: Always ack after DB update, not before
3. **Pagination**: Use stable sorting (ORDER BY id DESC)
4. **Error Handling**: Return consistent JSON error format
5. **Template Rendering**: Handle null/missing fields gracefully
6. **Worker Retries**: Check retry_count to avoid infinite loops
7. **Transactions**: Use for SendCampaign (messages + queue publish)
8. **SQL Injection**: Use prepared statements always
9. **Graceful Shutdown**: Handle SIGTERM/SIGINT properly
10. **Testing**: Don't test with real RabbitMQ (use mocks)

---

## Environment Variables

Required variables (as per .env.example):
```bash
# Server
PORT=8080

# PostgreSQL
POSTGRES_USER=smsleopard
POSTGRES_PASSWORD=secret123
POSTGRES_DB=smsleopard_db
POSTGRES_HOST=db
POSTGRES_PORT=5432

# RabbitMQ
RABBITMQ_DEFAULT_USER=guest
RABBITMQ_DEFAULT_PASS=guest
RABBITMQ_HOST=rabbitmq
RABBITMQ_PORT=5672

# Other
ENV=development
```

---

## Git Commit Strategy

Suggested commit messages:
```
feat: add database migrations for customers, campaigns, and messages
feat: implement customer repository with CRUD operations
feat: implement campaign service with business logic
feat: add POST /campaigns endpoint
feat: implement queue publisher and consumer
feat: add worker for processing outbound messages
test: add template rendering unit tests
test: add pagination integration tests
feat: implement health check endpoint (extra feature)
docs: add SYSTEM_OVERVIEW.md
docs: update README with setup instructions
```

---

## Next Steps After Completion

1. **Code Review**: Self-review before submission
2. **Security Check**: Ensure no sensitive data in commits
3. **Documentation Review**: Verify all sections are complete
4. **Final Testing**: One more end-to-end test
5. **Repository Cleanup**: Remove unnecessary files
6. **Submission**: Email GitHub repo link to info@smsleopard.com

---

## Resources

- [Go Documentation](https://golang.org/doc/)
- [gorilla/mux Router](https://github.com/gorilla/mux)
- [lib/pq PostgreSQL Driver](https://github.com/lib/pq)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
- [golang-migrate](https://github.com/golang-migrate/migrate)

---

**Last Updated**: 2025-12-04
**Status**: Implementation Ready
**Estimated Total Time**: 8-11 hours
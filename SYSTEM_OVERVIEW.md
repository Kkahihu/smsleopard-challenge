# SMSLeopard Campaign System - Technical Overview

**Version:** 1.0  
**Last Updated:** December 10, 2025

---

## 1. Data Model & Architecture

### 1.1 Entity Relationship Diagram

```
┌─────────────────────┐
│     CUSTOMERS       │
├─────────────────────┤
│ id (PK)            │◄──────┐
│ phone*             │       │
│ first_name         │       │
│ last_name          │       │
│ location           │       │
│ preferred_product  │       │
│ created_at         │       │
└─────────────────────┘       │
                              │
                              │
                              │ Foreign Keys
                              │ (ON DELETE CASCADE)
                              │
┌─────────────────────┐       │
│     CAMPAIGNS       │       │
├─────────────────────┤       │
│ id (PK)            │◄──┐   │
│ name*              │   │   │
│ channel*           │   │   │
│ status*            │   │   │
│ base_template*     │   │   │
│ scheduled_at       │   │   │
│ created_at         │   │   │
│ updated_at         │   │   │
└─────────────────────┘   │   │
                          │   │
                          │   │
                          │   │
┌──────────────────────────┐ │
│  OUTBOUND_MESSAGES       │ │
├──────────────────────────┤ │
│ id (PK)                  │ │
│ campaign_id (FK)         ├─┘
│ customer_id (FK)         ├───┘
│ status*                  │
│ rendered_content         │
│ last_error               │
│ retry_count              │
│ created_at               │
│ updated_at               │
└──────────────────────────┘

* = NOT NULL
```

### 1.2 Status Enumerations

**Campaign Status:**
- `draft` → Initial state, can be edited
- `scheduled` → Scheduled for future dispatch
- `sending` → Currently being processed
- `sent` → All messages processed
- `failed` → Campaign failed

**Message Status:**
- `pending` → Queued, not yet processed
- `sent` → Successfully delivered
- `failed` → Delivery failed (may retry if retry_count < 3)

**Channels:**
- `sms` → SMS messaging
- `whatsapp` → WhatsApp messaging

### 1.3 Key Indexes

**Performance Optimizations:**
```sql
-- Campaigns
CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_created_at ON campaigns(created_at DESC);
CREATE INDEX idx_campaigns_channel ON campaigns(channel);

-- Outbound Messages
CREATE INDEX idx_outbound_messages_campaign_id ON outbound_messages(campaign_id);
CREATE INDEX idx_outbound_messages_status ON outbound_messages(status);
CREATE INDEX idx_outbound_messages_created_at ON outbound_messages(created_at DESC);

-- Customers
CREATE INDEX idx_customers_phone ON customers(phone);
```

---

## 2. Request Flow: POST /campaigns/{id}/send

### 2.1 High-Level Flow Diagram

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       │ POST /campaigns/123/send
       │ {customer_ids: [1,2,3]}
       ▼
┌─────────────────────────────────────────────────┐
│           API Server (Gorilla Mux)              │
│  ┌───────────────────────────────────────────┐  │
│  │  1. CampaignHandler.HandleSendCampaign()  │  │
│  │     - Parse request body                  │  │
│  │     - Validate campaign_id & customer_ids │  │
│  └───────────────┬───────────────────────────┘  │
│                  │                               │
│                  ▼                               │
│  ┌───────────────────────────────────────────┐  │
│  │  2. CampaignService.SendCampaign()        │  │
│  │     - Fetch campaign from DB              │  │
│  │     - Validate campaign.CanSend()         │  │
│  │     - Fetch customers from DB             │  │
│  └───────────────┬───────────────────────────┘  │
└──────────────────┼───────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│              Database Transaction                │
│  ┌───────────────────────────────────────────┐  │
│  │  3. BEGIN TRANSACTION                     │  │
│  │     ├─ Create OutboundMessage records     │  │
│  │     │  (status: pending, for each cust.)  │  │
│  │     │                                      │  │
│  │     ├─ Update campaign.status → "sending" │  │
│  │     │                                      │  │
│  │     └─ COMMIT                              │  │
│  └───────────────┬───────────────────────────┘  │
└──────────────────┼───────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│           RabbitMQ Queue Publisher               │
│  ┌───────────────────────────────────────────┐  │
│  │  4. For each OutboundMessage:             │  │
│  │     - Create MessageJob JSON              │  │
│  │     - Publish to "campaign_sends" queue   │  │
│  │     - Log any publish failures (non-fatal)│  │
│  └───────────────┬───────────────────────────┘  │
└──────────────────┼───────────────────────────────┘
                   │
                   ▼
           ┌──────────────┐
           │  RabbitMQ    │
           │   Queue:     │
           │"campaign_    │
           │  sends"      │
           └──────────────┘
                   │
                   ▼
         (Worker processes jobs)
```

### 2.2 Detailed Step-by-Step Flow

**Step 1: Request Validation**
```go
// Handler validates:
- Campaign ID is valid integer
- Customer IDs array is provided and not empty
- Request body is valid JSON
```

**Step 2: Business Logic Validation**
```go
// Service validates:
- Campaign exists in database
- Campaign status allows sending (draft or scheduled)
- Customer IDs exist in database
- At least one valid customer found
```

**Step 3: Atomic Database Transaction**
```go
// Inside transaction:
1. Create outbound_messages for each customer
   - campaign_id, customer_id set
   - status = 'pending'
   - rendered_content = NULL (set by worker)
   - retry_count = 0

2. Update campaigns.status = 'sending'

3. COMMIT transaction
```

**Step 4: Queue Publishing (Outside Transaction)**
```go
// For each message created:
- Publish MessageJob{message_id, campaign_id, customer_id}
- Use durable queue with persistent messages
- Log errors but don't fail request (worker will retry)
```

**Response:**
```json
{
  "campaign_id": 123,
  "messages_queued": 3,
  "status": "sending"
}
```

### 2.3 Error Handling

| Error Type | HTTP Status | Response |
|------------|-------------|----------|
| Invalid JSON | 400 | `{"error": {"code": "INVALID_REQUEST", "message": "..."}}` |
| Campaign not found | 404 | `{"error": {"code": "CAMPAIGN_NOT_FOUND", "message": "..."}}` |
| Cannot send campaign | 400 | `{"error": {"code": "INVALID_STATUS", "message": "..."}}` |
| Database error | 500 | `{"error": {"code": "INTERNAL_ERROR", "message": "..."}}` |

---

## 3. Queue Worker Processing Flow

### 3.1 Worker Architecture Diagram

```
┌──────────────────────────────────────────────────────────┐
│                    WORKER PROCESS                        │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │  1. Initialize                                  │    │
│  │     - Connect to PostgreSQL                     │    │
│  │     - Connect to RabbitMQ                       │    │
│  │     - Initialize TemplateService                │    │
│  │     - Initialize SenderService (95% success)    │    │
│  └────────────────┬───────────────────────────────┘    │
│                   │                                     │
│                   ▼                                     │
│  ┌────────────────────────────────────────────────┐    │
│  │  2. Start Consumer                              │    │
│  │     - QoS: prefetch_count = 1                   │    │
│  │     - Manual acknowledgment enabled             │    │
│  └────────────────┬───────────────────────────────┘    │
└───────────────────┼──────────────────────────────────────┘
                    │
                    │ Consume from "campaign_sends"
                    ▼
        ┌─────────────────────┐
        │   RabbitMQ Queue    │
        │  "campaign_sends"   │
        └──────────┬──────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────────┐
│              MESSAGE PROCESSING HANDLER                  │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │  3. Parse MessageJob from JSON                  │    │
│  └────────────────┬───────────────────────────────┘    │
│                   │                                     │
│                   ▼                                     │
│  ┌────────────────────────────────────────────────┐    │
│  │  4. Fetch Data (JOIN query)                     │    │
│  │     SELECT om.*, c.*, cust.*                    │    │
│  │     FROM outbound_messages om                   │    │
│  │     JOIN campaigns c ON om.campaign_id = c.id   │    │
│  │     JOIN customers cust ON om.customer_id       │    │
│  │     WHERE om.id = message_id                    │    │
│  └────────────────┬───────────────────────────────┘    │
│                   │                                     │
│                   ▼                                     │
│  ┌────────────────────────────────────────────────┐    │
│  │  5. Check Retry Limit                           │    │
│  │     IF retry_count >= 3:                        │    │
│  │        - Mark as permanently failed             │    │
│  │        - ACK message (remove from queue)        │    │
│  │        - STOP processing                        │    │
│  └────────────────┬───────────────────────────────┘    │
│                   │ retry_count < 3                     │
│                   ▼                                     │
│  ┌────────────────────────────────────────────────┐    │
│  │  6. Render Template                             │    │
│  │     - Replace {first_name}, {last_name}, etc.   │    │
│  │     - NULL fields → empty string                │    │
│  └────────────────┬───────────────────────────────┘    │
│                   │                                     │
│                   ▼                                     │
│  ┌────────────────────────────────────────────────┐    │
│  │  7. Send Message (Mock Sender)                  │    │
│  │     - Simulate latency (50-200ms)               │    │
│  │     - Random success/failure (95% success)      │    │
│  └────────────────┬───────────────────────────────┘    │
│                   │                                     │
│         ┌─────────┴─────────┐                           │
│         │                   │                           │
│         ▼                   ▼                           │
│  ┌──────────┐       ┌──────────────┐                   │
│  │ SUCCESS  │       │   FAILURE    │                   │
│  └────┬─────┘       └──────┬───────┘                   │
│       │                    │                            │
│       ▼                    ▼                            │
│  ┌────────────────┐  ┌────────────────────────┐        │
│  │ 8a. Update DB  │  │ 8b. Update DB          │        │
│  │  status='sent' │  │  status='failed'       │        │
│  │                │  │  retry_count++         │        │
│  │                │  │  last_error=error_msg  │        │
│  └────┬───────────┘  └────┬───────────────────┘        │
│       │                   │                             │
│       ▼                   ▼                             │
│  ┌────────────────┐  ┌────────────────────────┐        │
│  │ 9a. ACK msg    │  │ 9b. NACK msg           │        │
│  │  (remove from  │  │  (requeue for retry)   │        │
│  │   queue)       │  │                        │        │
│  └────────────────┘  └────────────────────────┘        │
└──────────────────────────────────────────────────────────┘
```

### 3.2 Retry Logic Details

**Retry Strategy:**
```
Attempt 1: Initial processing
  ↓ (fails)
Attempt 2: Requeued by NACK (retry_count = 1)
  ↓ (fails)
Attempt 3: Requeued by NACK (retry_count = 2)
  ↓ (fails)
Attempt 4: Retry limit check (retry_count = 3)
  ↓
  Mark as permanently failed, ACK (remove from queue)
```

**Key Implementation Points:**
- **Manual Acknowledgment:** Messages are ACK'd only after database update
- **Requeue on Failure:** NACK with requeue=true for retryable failures
- **Retry Limit Check:** Performed BEFORE processing to avoid infinite loops
- **Permanent Failure:** After 3 retries, message is marked failed and removed from queue
- **Idempotency:** Worker checks message status before processing to handle duplicate delivery

### 3.3 Worker Code Flow

```go
// Pseudocode
func processMessage(job *MessageJob) error {
    // 1. Fetch message with campaign and customer (single JOIN query)
    message, campaign, customer := fetchMessageData(job.MessageID)
    
    // 2. Check retry limit
    if message.RetryCount >= 3 {
        updateMessagePermanentFailure(message.ID)
        return nil // ACK to remove from queue
    }
    
    // 3. Render template
    rendered := templateService.Render(campaign.BaseTemplate, customer)
    
    // 4. Send message
    result := senderService.Send(campaign.Channel, customer.Phone, rendered)
    
    // 5. Update database based on result
    if result.Success {
        updateMessageSuccess(message.ID) // status='sent'
        return nil // ACK
    } else {
        updateMessageFailure(message.ID, result.Error) // status='failed', retry_count++
        return error // NACK (requeue)
    }
}
```

---

## 4. Pagination Strategy

### 4.1 Implementation Approach

**SQL Query Pattern:**
```sql
SELECT id, name, channel, status, base_template, scheduled_at, created_at, updated_at
FROM campaigns
WHERE 1=1
  AND channel = $1     -- Optional filter
  AND status = $2      -- Optional filter
ORDER BY id DESC       -- Stable ordering (newest first)
LIMIT $3               -- Page size (default: 20, max: 100)
OFFSET $4;             -- Skip previous pages
```

**Calculation:**
```go
pageSize := 20 // default
if userPageSize > 0 && userPageSize <= 100 {
    pageSize = userPageSize
}

offset := (page - 1) * pageSize
if offset < 0 {
    offset = 0
}

totalPages := (totalCount + pageSize - 1) / pageSize
```

### 4.2 Why ORDER BY id DESC?

**Stable Ordering Benefits:**
1. **Primary Key Index:** Fast sorting using existing index
2. **Consistent Results:** New records don't shift existing pages
3. **Predictable:** Sequential IDs ensure same order across requests
4. **No Duplicates:** Unique ID prevents overlap between pages

**Alternative Avoided:**
- `ORDER BY created_at DESC` - Timestamp collisions can cause duplicates
- `ORDER BY name` - Insertions can shift page boundaries

### 4.3 Response Format

```json
{
  "campaigns": [
    {"id": 45, "name": "Campaign X", ...},
    {"id": 44, "name": "Campaign Y", ...}
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_count": 42,
    "total_pages": 3
  }
}
```

### 4.4 Query Parameters

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `page` | int | 1 | - | Page number (1-indexed) |
| `page_size` | int | 20 | 100 | Items per page |
| `channel` | string | - | - | Filter: 'sms' or 'whatsapp' |
| `status` | string | - | - | Filter: campaign status |

**Example Request:**
```
GET /campaigns?page=2&page_size=20&status=sent&channel=sms
```

### 4.5 Performance Considerations

**Index Usage:**
- `WHERE` filters use indexes: `idx_campaigns_status`, `idx_campaigns_channel`
- `ORDER BY id DESC` uses primary key index
- `LIMIT/OFFSET` applied after filtering

**Optimization Tips:**
- Keep page size reasonable (≤100 to avoid memory issues)
- Use filters to reduce total count
- Consider cursor-based pagination for very large datasets (future enhancement)

---

## 5. Personalization System

### 5.1 Template Syntax

**Placeholder Format:**
```
{field_name}
```

**Supported Fields:**
- `{first_name}` → Customer.FirstName
- `{last_name}` → Customer.LastName
- `{location}` → Customer.Location
- `{preferred_product}` → Customer.PreferredProduct
- `{phone}` → Customer.Phone

**Example Template:**
```
Hi {first_name} from {location}! 
Check out our {preferred_product} special offers.
```

**Rendered Output:**
```
Hi John from Nairobi! 
Check out our Smartphone special offers.
```

### 5.2 NULL Field Handling Strategy

**Decision:** Replace NULL/empty fields with **empty string**

**Rationale:**
1. **Graceful Degradation:** Message still sends, just less personalized
2. **No Template Errors:** Avoids breaking message delivery
3. **Flexibility:** Allows partial data collection
4. **User Control:** Template design determines context

**Example:**
```
Template: "Hi {first_name}! Your phone is {phone}"
Customer: {first_name: NULL, phone: "+254700123456"}
Result:   "Hi ! Your phone is +254700123456"
```

**Alternative Approaches Considered:**
- ❌ **Keep placeholder:** `"Hi {first_name}!"` - Looks unprofessional
- ❌ **Use default value:** `"Hi Customer!"` - Less authentic
- ❌ **Block sending:** Too restrictive, loses revenue
- ✅ **Empty string:** Balances flexibility and quality

### 5.3 Rendering Process

```go
func Render(template string, customer *Customer) string {
    rendered := template
    
    // Replace each field
    if customer.FirstName != nil && *customer.FirstName != "" {
        rendered = strings.ReplaceAll(rendered, "{first_name}", *customer.FirstName)
    } else {
        rendered = strings.ReplaceAll(rendered, "{first_name}", "")
    }
    
    // ... repeat for all fields ...
    
    return rendered
}
```

### 5.4 Extension Points

**Future Enhancements:**

**1. Dynamic Fields:**
```go
// Add new customer fields without code changes
type CustomerData map[string]interface{}

func RenderDynamic(template string, data CustomerData) string {
    for key, value := range data {
        placeholder := fmt.Sprintf("{%s}", key)
        rendered = strings.ReplaceAll(rendered, placeholder, toString(value))
    }
}
```

**2. Conditional Logic:**
```
Template: "Hi {first_name}! {{if location}}We're in {location} too!{{end}}"
```

**3. Filters/Formatters:**
```
{first_name|uppercase}         → "JOHN"
{location|titlecase}           → "Nairobi"
{preferred_product|pluralize}  → "Smartphones"
```

**4. Fallback Values:**
```
{first_name|default:"Valued Customer"}
```

**5. Multi-language Support:**
```json
{
  "template_en": "Hi {first_name}!",
  "template_sw": "Habari {first_name}!",
  "language": "customer.preferred_language"
}
```

**6. A/B Testing:**
```go
// Randomly select template variant
variants := []string{template_a, template_b}
selectedTemplate := variants[rand.Intn(len(variants))]
```

**7. Rich Content:**
```json
{
  "type": "whatsapp_template",
  "header": {"type": "image", "url": "{product_image}"},
  "body": "Hi {first_name}!",
  "footer": "Reply STOP to unsubscribe"
}
```

### 5.5 Validation

**Template Validation:**
```go
func ValidateTemplate(template string) error {
    // Check balanced braces
    if strings.Count(template, "{") != strings.Count(template, "}") {
        return errors.New("unbalanced braces")
    }
    
    // Warn about unknown fields (non-fatal)
    placeholders := extractPlaceholders(template)
    for _, p := range placeholders {
        if !isValidField(p) {
            log.Warn("Unknown placeholder: %s", p)
        }
    }
    
    return nil
}
```

---

## 6. System Architecture Summary

### 6.1 Component Interaction

```
┌─────────────┐     HTTP      ┌──────────────┐     SQL      ┌────────────┐
│   Client    │ ────────────► │  API Server  │ ───────────► │ PostgreSQL │
└─────────────┘     JSON      │  (Port 8080) │              └────────────┘
                              └───────┬──────┘
                                      │ AMQP
                                      │ Publish
                                      ▼
                              ┌──────────────┐
                              │  RabbitMQ    │
                              │   Queue      │
                              └───────┬──────┘
                                      │ AMQP
                                      │ Consume
                                      ▼
                              ┌──────────────┐     SQL      ┌────────────┐
                              │    Worker    │ ───────────► │ PostgreSQL │
                              │   Process    │              └────────────┘
                              └──────────────┘
```

### 6.2 Technology Choices

**PostgreSQL:**
- ACID transactions for campaign sending
- Complex queries with JOINs
- Strong consistency guarantees

**RabbitMQ:**
- Durable queues with persistent messages
- Manual acknowledgment for reliability
- Prevents message loss during failures
- Built-in retry via requeue mechanism

**Go (Golang):**
- High performance for I/O operations
- Simple concurrency model (goroutines)
- Strong standard library
- Easy deployment (single binary)

---

## 7. Quality Assurance

### 7.1 Testing Strategy

**Unit Tests:**
- Template rendering with various NULL combinations
- Pagination calculations
- Status transitions

**Integration Tests:**
- API endpoints with real database
- Campaign send flow end-to-end
- Worker processing with mocked sender

**Key Test Cases:**
- ✅ Template rendering with NULL fields
- ✅ Pagination without duplicates
- ✅ Worker retry logic (3 attempts)
- ✅ Message preview endpoint

### 7.2 Observability

**Logging:**
- All API requests logged
- Worker processing status logged
- Errors captured with context

**Health Monitoring:**
- `GET /health` endpoint checks:
  - PostgreSQL connectivity
  - RabbitMQ connectivity
  - Returns 503 if any dependency is down

**Metrics (Future):**
- Campaign success rate
- Average processing time
- Queue depth
- Failed message count

---

## Document Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-12-10 | Initial comprehensive documentation |

---

**End of System Overview**
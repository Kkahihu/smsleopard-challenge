# SMSLeopard Engineering Challenge

👋 **Hello!** We're really excited about you potentially joining the team, so we designed this take-home exercise to give you a taste of the challenges you may encounter in the role and to help us better understand what it would be like to work closely together.

Thanks for taking the time!

---

## About SMSLeopard

SMSLeopard is a messaging platform used by businesses to send SMS and WhatsApp campaigns. To learn more about what we do, visit [smsleopard.com](https://smsleopard.com).

> ⚠️ **Important:** Please don't fork this repository or create a pull request against it other applicants may take inspiration from your work. Instead, create a **new repository** for your solution. Once you've completed the challenge, email us at **<info@smsleopard.com>** and share your GitHub repo.

---

## Overview

In this exercise, you will design and implement a backend service in Go that:

- Exposes HTTP endpoints for managing campaigns
- Stores data in a relational database
- Uses a queue to send messages asynchronously
- Supports simple personalized messages

**We are evaluating:**

- How you design and structure a Go backend from scratch
- How you model data and APIs around a realistic use case
- How you use a queue for background work
- How you write tests and explain your design
- How you think about personalization (template + future AI integration)

> **Note:** You may use AI tools (e.g., Copilot, ChatGPT). If you do, please mention where they helped.

**Estimated time:** 4-8 hours. It's okay if not everything is "perfect" or fully complete, prioritize clean design, correctness, and clear reasoning. If you're spending more than 6 hours on core requirements, submit what you have and document what you'd do with more time.

---

## Table of Contents

- [SMSLeopard Engineering Challenge](#smsleopard-engineering-challenge)
  - [About SMSLeopard](#about-smsleopard)
  - [Overview](#overview)
  - [Table of Contents](#table-of-contents)
  - [Functional Requirements](#functional-requirements)
    - [Data Model](#data-model)
      - [`customers`](#customers)
      - [`campaigns`](#campaigns)
      - [`outbound_messages`](#outbound_messages)
    - [API Endpoints](#api-endpoints)
      - [1. Create Campaign](#1-create-campaign)
      - [2. Send Campaign](#2-send-campaign)
      - [3. List Campaigns](#3-list-campaigns)
      - [4. Get Campaign Details](#4-get-campaign-details)
      - [5. Personalized Preview](#5-personalized-preview)
    - [Queue Worker](#queue-worker)
  - [Non-Functional Requirements](#non-functional-requirements)
  - [Required Tests](#required-tests)
    - [1. Template Rendering](#1-template-rendering)
    - [2. Pagination (`GET /campaigns`)](#2-pagination-get-campaigns)
    - [3. Worker Logic](#3-worker-logic)
    - [4. Personalized Preview](#4-personalized-preview)
  - [Open Design Task](#open-design-task)
  - [Bonus: Frontend Dashboard (Optional)](#bonus-frontend-dashboard-optional)
    - [Suggested Scope (pick 1–2)](#suggested-scope-pick-12)
    - [Guidelines](#guidelines)
    - [We'll Appreciate](#well-appreciate)
  - [Deliverables](#deliverables)
    - [1. Code Repository](#1-code-repository)
    - [2. `SYSTEM_OVERVIEW.md` (max 2 pages)](#2-system_overviewmd-max-2-pages)
    - [3. `README.md`](#3-readmemd)
    - [4. Time \& Tools Note](#4-time--tools-note)
  - [Evaluation Criteria](#evaluation-criteria)

---

## Functional Requirements

Build a **Campaign Dispatch Service** with the following responsibilities.

### Data Model

Design a relational schema with at least these entities:

#### `customers`

| Column            | Type    | Notes       |
| ----------------- | ------- | ----------- |
| id                | integer | primary key |
| phone             | string  |             |
| first_name        | string  |             |
| last_name         | string  |             |
| location          | string  |             |
| preferred_product | string  | e.g., "Running Shoes", "Winter Jacket" |

#### `campaigns`

| Column        | Type      | Notes                                                  |
| ------------- | --------- | ------------------------------------------------------ |
| id            | integer   | primary key                                            |
| name          | string    |                                                        |
| channel       | string    | `sms` or `whatsapp`                                    |
| status        | string    | `draft`, `scheduled`, `sending`, `sent`, `failed`      |
| base_template | text      | e.g., `"Hi {first_name}, check out {preferred_product}"` |
| scheduled_at  | timestamp | nullable                                               |
| created_at    | timestamp |                                                        |

#### `outbound_messages`

| Column           | Type      | Notes                           |
| ---------------- | --------- | ------------------------------- |
| id               | integer   | primary key                     |
| campaign_id      | integer   | foreign key → campaigns         |
| customer_id      | integer   | foreign key → customers         |
| status           | string    | `pending`, `sent`, `failed` (see state transitions below) |
| rendered_content | text      | the final personalized message  |
| last_error       | text      | nullable                        |
| retry_count      | integer   | defaults to 0                   |
| created_at       | timestamp |                                 |
| updated_at       | timestamp |                                 |

**Message Status Transitions:**

- `pending` → `sent` (successful delivery)
- `pending` → `failed` (delivery failure after retries)

You may extend the schema if needed (e.g., adding indexes on `campaign_id`, `status`, `created_at` for efficient querying).

**Database choice:** Use **PostgreSQL** (preferred) or MySQL. SQLite is acceptable if time is short, but mention the trade-offs in your documentation.

---

### API Endpoints

Implement an HTTP API with these endpoints:

#### 1. Create Campaign

`POST /campaigns`

Create a new campaign.

**Request body:**

```json
{
  "name": "Summer Sale 2025",
  "channel": "sms",
  "base_template": "Hi {first_name}, check out {preferred_product} in {location}!",
  "scheduled_at": "2025-06-01T10:00:00Z"
}
```

| Field           | Required | Description                                      |
| --------------- | -------- | ------------------------------------------------ |
| `name`          | Yes      | Campaign name                                    |
| `channel`       | Yes      | `sms` or `whatsapp`                              |
| `base_template` | Yes      | Message template with placeholders               |
| `scheduled_at`  | No       | If omitted, campaign is ready to send immediately |

**Response:** Created campaign object with `id`, `status`, `created_at`, etc.

**Behavior:**

- New campaigns start as `draft`
- If `scheduled_at` is provided and is in the future, the campaign can be marked as `scheduled`
- The campaign status determines whether it's ready to be sent via the send endpoint

---

#### 2. Send Campaign

`POST /campaigns/{id}/send`

Initiate sending of a campaign to a set of customers.

**Request body:**

```json
{
  "customer_ids": [1, 2, 3]
}
```

**Required behavior:**

- Validate that the campaign is in `draft` or `scheduled` status
- For each `customer_id`:
  - Create an `outbound_messages` row with status `pending`
  - Publish a job to a queue (e.g., `campaign_sends`) containing the `outbound_message_id`
- Update campaign status to `sending`
- **Return quickly**—no actual sending logic in this endpoint

**Notes:**

- **Customer selection:** For this exercise, you'll manually specify `customer_ids`. In a real system, you might target "all customers" or use segmentation criteria. Feel free to add thoughts on this in your documentation.
- **Scheduled campaigns:** This endpoint allows manual/immediate sending. If you want scheduled campaigns to send automatically at the scheduled time, implement that as part of your extra feature (or core functionality—your choice).

**Response:**

```json
{
  "campaign_id": 1,
  "messages_queued": 3,
  "status": "sending"
}
```

---

#### 3. List Campaigns

`GET /campaigns`

List campaigns with pagination and filtering.

**Query parameters:**

| Parameter   | Default | Description                          |
| ----------- | ------- | ------------------------------------ |
| `page`      | 1       | Page number                          |
| `page_size` | 20      | Items per page (max 100)             |
| `channel`   | —       | Filter by channel (`sms`, `whatsapp`) |
| `status`    | —       | Filter by status                     |

**Response:**

```json
{
  "data": [
    {
      "id": 1,
      "name": "Summer Sale 2025",
      "channel": "sms",
      "status": "sent",
      "created_at": "2025-05-20T08:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_count": 57,
    "total_pages": 3
  }
}
```

**Requirements:**

- Order by newest first (`created_at DESC` or `id DESC`)
- Must be robust: **no duplicates, no missing records** when paging
  - _Hint: Consider what happens if new campaigns are created while someone is paging through results. Use stable sorting (e.g., order by id DESC) to ensure consistency._
- Filters must be correctly applied

---

#### 4. Get Campaign Details

`GET /campaigns/{id}`

Get campaign details with message statistics.

**Response:**

```json
{
  "id": 1,
  "name": "Summer Sale 2025",
  "channel": "sms",
  "status": "sending",
  "base_template": "Hi {first_name}, check out {preferred_product}!",
  "scheduled_at": null,
  "created_at": "2025-05-20T08:00:00Z",
  "stats": {
    "total": 100,
    "pending": 45,
    "sending": 10,
    "sent": 40,
    "failed": 5
  }
}
```

Stats can be computed via SQL aggregates or in Go—your choice.

---

#### 5. Personalized Preview

`POST /campaigns/{id}/personalized-preview`

Preview how a message will render for a specific customer.

**Request body:**

```json
{
  "customer_id": 123,
  "override_template": "Hi {first_name}, special offer just for you!"
}
```

| Field               | Required | Description                                          |
| ------------------- | -------- | ---------------------------------------------------- |
| `customer_id`       | Yes      | The customer to preview for                          |
| `override_template` | No       | Uses campaign's `base_template` if omitted           |

**Response:**

```json
{
  "rendered_message": "Hi Alice, special offer just for you!",
  "used_template": "Hi {first_name}, special offer just for you!",
  "customer": {
    "id": 123,
    "first_name": "Alice"
  }
}
```

> _This endpoint is a stepping stone toward AI-powered personalization—for now, implement template substitution only._

---

### Queue Worker

Build a worker (can be a separate process or a Go command) that processes message jobs.

**Responsibilities:**

1. **Listen** on the queue for message jobs (each job contains an `outbound_message_id`)

2. **For each job:**
   - Fetch the `outbound_messages` row by ID, along with related `customers` and `campaigns` data
   - Render the personalized message using `base_template` and customer fields (replace `{first_name}`, `{preferred_product}`, etc.)
     - **Template behavior:** Replace `{field_name}` placeholders with corresponding customer field values. Handle missing/null fields gracefully (you decide the behavior—document it in your README)
   - Call a **mock sender** function that simulates sending
     - Implement a simple mock that succeeds 90-95% of the time (or always succeeds—your choice)
     - Document your mock sender behavior
   - Update `outbound_messages.status` to `sent` or `failed`
   - Set `last_error` if sending failed
   - Increment `retry_count` if retrying

3. **Reliability requirements:**
   - Use acknowledgements appropriately (ack after successful DB update)
   - Don't lose messages
   - Implement basic retry logic: failed messages should be retried up to 3 times before final failure
   - Avoid infinite requeue loops

**Suggested queue:** RabbitMQ (recommended) or Redis. Document your choice and reasoning.

---

## Non-Functional Requirements

- Use **Go modules**
- Use a clear project structure with separation between:
  - HTTP handlers
  - Business logic / services
  - Data access / repositories
- Provide a `docker-compose.yml` or clear instructions to run:
  - The database
  - The queue service
  - Your application service(s)
- Use **environment variables** or a config file for DB/queue settings
- Return consistent JSON error responses:

```json
{
  "error": {
    "code": "CAMPAIGN_NOT_FOUND",
    "message": "Campaign with ID 999 not found"
  }
}
```

---

## Required Tests

Write tests in Go for at least:

### 1. Template Rendering

- Given a template and a customer, placeholders are correctly substituted
- Handle missing/null customer fields gracefully (you decide the behavior—document it)
- Test with multiple customer field combinations

### 2. Pagination (`GET /campaigns`)

- With more than 2 pages of campaigns:
  - No duplicates between pages
  - Ordering is consistent
  - Filters (`channel`, `status`) are respected

### 3. Worker Logic

- Use an in-memory channel-based queue or mock the queue interface entirely
- Example test: a queued job leads to `outbound_messages.status` updated to `sent`
- Test both success and failure scenarios
- _You don't need to test with real RabbitMQ—demonstrate your testing approach_

### 4. Personalized Preview

- Test that the preview endpoint correctly renders templates for different customers
- Verify that `override_template` parameter works as expected

---

## Open Design Task

Pick **one** additional feature you think is valuable in a real system like this, and implement at least part of it.

**Examples:**

| Feature             | Description                                                                 |
| ------------------- | --------------------------------------------------------------------------- |
| Enhanced retry policy | Failed messages are retried with exponential backoff and dead letter queue |
| Idempotency         | `POST /campaigns/{id}/send` doesn't create duplicates if called twice       |
| Health endpoint     | `/health` that checks DB + queue connectivity                               |
| Stats endpoint      | `/stats` exposing metrics (campaigns sent today, failure rate, etc.)        |
| Rate limiting       | Basic rate limiting on the send endpoint                                    |
| Scheduled dispatch  | Background job that automatically sends campaigns when `scheduled_at` is reached |

**Note:** Since the schema includes a `scheduled_at` field, you may want to implement scheduled dispatch as part of your core solution rather than as the extra feature. Either approach is acceptable—just be clear in your documentation.

Document your choice and reasoning in the README.

---

## Bonus: Frontend Dashboard (Optional)

> ⚡ **This section is entirely optional.** It won't affect your evaluation negatively if skipped, but a clean implementation will certainly stand out.

If you'd like to showcase full-stack skills, build a simple **React** dashboard that interacts with your API. Keep it minimal—we're more interested in clean code and good UX instincts than feature completeness.

### Suggested Scope (pick 1–2)

| Feature         | Description                                                              |
| --------------- | ------------------------------------------------------------------------ |
| Campaign List   | Display paginated campaigns with status badges and basic filtering       |
| Campaign Detail | Show a single campaign with message stats (counts or progress bar)       |
| Create Campaign | A form to create a new campaign with template field                      |
| Live Preview    | Use `/personalized-preview` to show how a message renders for a customer |

### Guidelines

- Use **React** (Vite, Create React App, or Next.js—your choice)
- Keep styling simple but intentional (Tailwind CSS, plain CSS, or a component library)
- Focus on **one or two screens done well**, rather than many screens done poorly
- Include the frontend in your `docker-compose.yml` or provide run instructions

### We'll Appreciate

- Clean component structure
- Thoughtful loading and error states
- Basic responsive design

_If you attempt this, add a brief section in your README describing what you built and any trade-offs you made._

---

## Deliverables

Please provide:

### 1. Code Repository

A GitHub repository (or zip file) containing:

- Source code (backend, and frontend if attempted)
- Tests
- `docker-compose.yml` (if used)
- Database migrations or seed scripts
- Seed data: Include sample data with at least 10 customers and 2-3 campaigns for testing

### 2. `SYSTEM_OVERVIEW.md` (max 2 pages)

Describe:

- Data model and entity relationships (include any indexes you added)
- Request flow for `POST /campaigns/{id}/send`
- How the queue worker processes messages (including retry logic)
- Pagination strategy and how you avoid duplicates/missing records
- Personalization approach: How your template system works and what extension points exist for future enhancements (e.g., AI-driven content, dynamic personalization)

### 3. `README.md`

Include:

- How to run the service(s) and tests
- Any assumptions you made
- **Template handling:** Document how you handle missing/null customer fields
- **Mock sender behavior:** Document your mock sender implementation
- **Queue choice:** Document which queue system you chose (RabbitMQ/Redis) and why
- **EXTRA_FEATURE** section:
  - Which feature you chose and why
  - Implementation details and limitations
- **FRONTEND** section (if attempted):
  - What you built
  - How to run it
  - Any trade-offs or shortcuts taken

### 4. Time & Tools Note

Include a brief note on:

- Approximate time spent
- Whether you used AI tools and where they helped

---

## Evaluation Criteria

| Area                           | Weight   | Notes                                                    |
| ------------------------------ | -------- | -------------------------------------------------------- |
| Code quality & structure       | 30%      | Clean, idiomatic Go; clear separation of concerns        |
| API design & correctness       | 25%      | RESTful patterns, proper status codes, edge cases        |
| Data modeling                  | 15%      | Appropriate schema design, relationships, indexes        |
| Queue/worker implementation    | 15%      | Reliability, error handling, no message loss             |
| Tests                          | 10%      | Coverage of critical paths, test quality                 |
| Documentation                  | 5%       | Clear explanations, good README                          |
| **Bonus** (frontend, extras)   | +10%     | Only positive impact; not attempting won't hurt you      |

---

Good luck! We're excited to see your approach. If you have questions, don't hesitate to reach out.
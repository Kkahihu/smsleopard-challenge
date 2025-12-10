# SMSLeopard Scripts

This directory contains utility scripts for managing the SMSLeopard database.

## Migration Runner (`migrate.go`)

A comprehensive Go-based migration runner that manages database schema migrations with version tracking.

### Features

- **Version Tracking**: Tracks applied migrations in `schema_migrations` table
- **Transaction Safety**: Each migration runs in a transaction
- **Multiple Commands**: up, down, status, reset, seed
- **Colored Output**: Green/red/yellow/cyan console output for clarity
- **Error Handling**: Clear error messages with context

### Commands

#### `up` - Apply Pending Migrations
Applies all migrations that haven't been run yet.

```bash
go run scripts/migrate.go up
```

Example output:
```
=== SMSLeopard Migration Runner ===

Connecting to database...
✓ Connected to database

Running pending migrations...

Applying migration 001_create_customers...
  ✓ Migration 001 applied successfully
Applying migration 002_create_campaigns...
  ✓ Migration 002 applied successfully
Applying migration 003_create_outbound_messages...
  ✓ Migration 003 applied successfully

✓ Successfully applied 3 migration(s)

✨ Operation completed successfully!
```

#### `down` - Rollback Last Migration
Rolls back the most recently applied migration by dropping its tables.

```bash
go run scripts/migrate.go down
```

Example output:
```
Rolling back last migration...

Rolling back migration 003...
  ✓ Migration 003 rolled back
✓ Successfully rolled back migration 003_create_outbound_messages
```

#### `status` - Show Migration Status
Displays a table showing which migrations are applied and which are pending.

```bash
go run scripts/migrate.go status
```

Example output:
```
Migration Status:

VERSION    NAME                              STATUS       APPLIED AT          
-------------------------------------------------------------------------------------
001        create_customers                  applied      2025-12-10 15:20:15
002        create_campaigns                  applied      2025-12-10 15:20:15
003        create_outbound_messages          applied      2025-12-10 15:20:16
-------------------------------------------------------------------------------------

Summary: 3/3 migrations applied
```

#### `reset` - Reset All Migrations
Rolls back all migrations and reapplies them. Useful for testing or resetting to a clean state.

```bash
go run scripts/migrate.go reset
```

⚠️ **Warning**: This will drop all tables and recreate them, deleting all data.

#### `seed` - Run Seed Migrations
Runs only the seed data migrations from `migrations/seed/` directory.

```bash
go run scripts/migrate.go seed
```

Example output:
```
Running seed migrations...

Running seed 001_customers...
  ✓ Seed 001 applied successfully
Running seed 002_campaigns...
  ✓ Seed 002 applied successfully

✓ Successfully ran 2 seed migration(s)
```

### Migration Files

#### Schema Migrations
Located in `migrations/*.sql`:
- `001_create_customers.sql` - Creates customers table
- `002_create_campaigns.sql` - Creates campaigns table
- `003_create_outbound_messages.sql` - Creates outbound_messages table

#### Seed Migrations
Located in `migrations/seed/*.sql`:
- `001_customers.sql` - Seeds 15 test customers
- `002_campaigns.sql` - Seeds 3 test campaigns

### Migration Tracking

Migrations are tracked in the `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Rollback Strategy

The `down` command uses simple DROP TABLE statements in reverse dependency order:

- **Version 003**: `DROP TABLE IF EXISTS outbound_messages;`
- **Version 002**: `DROP TABLE IF EXISTS campaigns;`
- **Version 001**: `DROP TABLE IF EXISTS customers;`

### Usage Workflow

#### Initial Setup
```bash
# 1. Start database (via Docker)
docker-compose up -d postgres

# 2. Apply all migrations
go run scripts/migrate.go up

# 3. Seed test data
go run scripts/migrate.go seed
```

#### Development Workflow
```bash
# Check migration status
go run scripts/migrate.go status

# Apply new migrations
go run scripts/migrate.go up

# Rollback last migration if needed
go run scripts/migrate.go down

# Reset database to clean state
go run scripts/migrate.go reset
```

### Error Handling

The script provides clear error messages for common issues:

- **Database connection errors**: Check Docker is running and credentials are correct
- **File read errors**: Ensure migration files exist and are readable
- **SQL execution errors**: Check SQL syntax in migration files
- **Missing database**: Create database first via Docker

### Notes

- Migrations run in transactions for safety
- Already-applied migrations are skipped automatically
- Seed migrations can be run multiple times (they use `ON CONFLICT DO NOTHING`)
- Each migration file must follow the naming pattern: `NNN_description.sql`
- Version numbers must be unique and sequential (001, 002, 003, etc.)

---

## Database Seeder (`seed.go`)

A Go script that generates and inserts programmatic seed data for testing and development.

### Features

- Generates realistic Kenyan customer data
- Creates diverse campaign templates
- Supports custom counts via flags
- Can clear existing seed data
- Idempotent - safe to run multiple times

### Usage

```bash
# Basic usage (default: 12 customers, 3 campaigns)
go run scripts/seed.go

# Custom counts
go run scripts/seed.go -customers=20 -campaigns=5

# Clear existing seed data first
go run scripts/seed.go -clear

# Clear and reseed with custom counts
go run scripts/seed.go -clear -customers=50

# Show help
go run scripts/seed.go -help
```

### Flags

- `-customers=N` - Number of customers to create (default: 12)
- `-campaigns=N` - Number of campaigns to create (default: 3)
- `-clear` - Clear existing seed data before inserting
- `-help` - Show usage information

### Data Characteristics

**Customers:**
- Phone pattern: `+2547000010XXX` (different from SQL seeds)
- Realistic Kenyan names and locations
- Varied NULL field combinations for testing
- Products: Smartphones, Laptops, Tablets, etc.

**Campaigns:**
- Multiple channel types (SMS, WhatsApp)
- Various statuses (draft, scheduled, sent, failed)
- Template placeholders for personalization
- Some with scheduled_at timestamps

### Notes

- Uses `ON CONFLICT DO NOTHING` for idempotency
- Different phone pattern from SQL seeds to avoid conflicts
- Generates varied data with NULL fields for edge case testing
- Progress reporting with colored output

---

## Comparison: migrate.go vs seed.go

| Feature | migrate.go | seed.go |
|---------|-----------|---------|
| **Purpose** | Schema management | Test data generation |
| **Data Source** | SQL files | Programmatic |
| **Tracking** | Version tracked | Not tracked |
| **Idempotency** | Migrations skip if applied | ON CONFLICT handling |
| **Rollback** | Supported (down/reset) | Manual deletion |
| **Use Case** | Production & Development | Development & Testing |

### When to Use Which

**Use `migrate.go`:**
- Setting up database schema initially
- Deploying schema changes to production
- Rolling back problematic migrations
- Checking migration status
- Loading SQL-based seed data

**Use `seed.go`:**
- Generating large amounts of test data
- Creating varied data with specific patterns
- Quick iteration during development
- Testing with different data sizes

### Typical Workflow

```bash
# 1. Setup database schema
go run scripts/migrate.go up

# 2. Load SQL seed data (small, curated dataset)
go run scripts/migrate.go seed

# 3. Add programmatic seed data (larger, varied dataset)
go run scripts/seed.go -customers=100 -campaigns=10

# 4. During testing, clear and reseed as needed
go run scripts/seed.go -clear -customers=50

# 5. Reset schema if needed
go run scripts/migrate.go reset
go run scripts/migrate.go seed
```

---

## Prerequisites

Both scripts require:
- Go 1.21 or later
- PostgreSQL database running (via Docker or local)
- `.env` file with database configuration
- Project dependencies installed (`go mod download`)

## Environment Variables

Required in `.env` file:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=smsleopard_db
```

## Getting Help

Both scripts provide detailed help:

```bash
go run scripts/migrate.go help
go run scripts/seed.go -help
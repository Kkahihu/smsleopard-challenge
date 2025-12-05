# SMSLeopard Testing Scripts

This directory contains scripts for testing the SMSLeopard asynchronous message processing system.

## Available Scripts

### 1. `test_workflow.ps1` / `test_workflow.sh`

**Purpose**: Complete end-to-end testing workflow automation

**Platforms**:
- `test_workflow.ps1` - Windows PowerShell
- `test_workflow.sh` - Unix/Linux/macOS/WSL (Bash)

**What it does**:
1. ✅ Checks prerequisites (Docker, Go, docker-compose)
2. ✅ Starts Docker services (PostgreSQL, RabbitMQ)
3. ✅ Waits for services to be ready (health checks)
4. ✅ Runs database migrations
5. ✅ Prompts to start API server and Worker
6. ✅ Runs comprehensive API tests
7. ✅ Validates results
8. ✅ Provides cleanup instructions

**Usage**:

```powershell
# Windows PowerShell
.\scripts\test_workflow.ps1

# With options (skip certain steps)
.\scripts\test_workflow.ps1 -SkipPrerequisites -SkipServices
```

```bash
# Unix/Linux/Mac/WSL
chmod +x scripts/test_workflow.sh
./scripts/test_workflow.sh
```

**Options** (PowerShell only):
- `-SkipPrerequisites` - Skip prerequisite checks
- `-SkipServices` - Skip starting Docker services
- `-SkipMigrations` - Skip database migrations
- `-SkipSeeding` - Skip test data seeding

**Expected Duration**: 2-3 minutes (including manual server startup)

---

### 2. `seed_test_data.ps1` / `seed_test_data.sh`

**Purpose**: Populate database with test customers and campaigns

**Platforms**:
- `seed_test_data.ps1` - Windows PowerShell
- `seed_test_data.sh` - Unix/Linux/macOS/WSL (Bash)

**What it does**:
1. ✅ Checks API availability
2. ✅ Creates test customers (default: 5)
3. ✅ Creates test campaigns (default: 2)
4. ✅ Shows preview of first campaign
5. ✅ Provides next steps instructions

**Usage**:

```powershell
# Windows PowerShell - Default (5 customers, 2 campaigns)
.\scripts\seed_test_data.ps1

# Custom counts
.\scripts\seed_test_data.ps1 -CustomerCount 10 -CampaignCount 3

# Custom API URL
.\scripts\seed_test_data.ps1 -ApiUrl "http://localhost:8080"
```

```bash
# Unix/Linux/Mac/WSL - Default
./scripts/seed_test_data.sh

# Custom counts (positional arguments)
./scripts/seed_test_data.sh http://localhost:8080 10 3
# Args: [API_URL] [CUSTOMER_COUNT] [CAMPAIGN_COUNT]
```

**Parameters**:
- `ApiUrl` / `API_URL` - API server URL (default: `http://localhost:8080`)
- `CustomerCount` / `CUSTOMER_COUNT` - Number of customers to create (default: 5)
- `CampaignCount` / `CAMPAIGN_COUNT` - Number of campaigns to create (default: 2)

**Expected Duration**: 10-30 seconds depending on count

**Prerequisites**: API server must be running

---

## Quick Start Guide

### First Time Setup

1. **Run the automated test workflow:**
   ```powershell
   # Windows
   .\scripts\test_workflow.ps1
   
   # Unix/Linux/Mac
   ./scripts/test_workflow.sh
   ```

2. **Follow the prompts** to start API and Worker servers

3. **Tests will run automatically** and validate the complete workflow

### Regular Testing

If you already have services running:

1. **Seed test data:**
   ```powershell
   .\scripts\seed_test_data.ps1
   ```

2. **Send a campaign:**
   ```bash
   curl -X POST http://localhost:8080/api/campaigns/1/send
   ```

3. **Check results:**
   ```bash
   curl http://localhost:8080/api/campaigns/1/messages
   ```

---

## Test Data

### Default Test Customers

The seed script creates customers with these details:

1. **Alice Johnson** - `+254700000001`
2. **Bob Williams** - `+254700000002`
3. **Carol Brown** - `+254700000003`
4. **David Miller** - `+254700000004`
5. **Eve Davis** - `+254700000005`
6. **Frank Garcia** - `+254700000006`
7. **Grace Martinez** - `+254700000007`
8. **Henry Rodriguez** - `+254700000008`

### Default Test Campaigns

1. **Welcome Campaign**
   - Template: `"Hi {{.Name}}, welcome to SMSLeopard! 🎉 Your journey starts here."`

2. **Product Launch**
   - Template: `"Hey {{.Name}}! We have exciting news for you. Check out our new product! Call {{.PhoneNumber}} for details."`

3. **Special Offer**
   - Template: `"Dear {{.Name}}, exclusive offer just for you! Limited time only. Contact: {{.PhoneNumber}}"`

---

## Script Features

### Color-Coded Output

Both scripts use color-coded output for better readability:

- 🟢 **Green** - Success messages
- 🔴 **Red** - Error messages
- 🔵 **Blue** - Information messages
- 🟡 **Yellow** - Section headers / warnings
- ⚫ **Gray** - Details / secondary information

### Error Handling

- Scripts exit on first error (`set -e` in bash, `$ErrorActionPreference = "Stop"` in PowerShell)
- Clear error messages with troubleshooting hints
- Prerequisite checks before running tests
- Service health checks with timeouts

### Validation

**test_workflow** validates:
- ✅ Docker and Docker Compose installed
- ✅ Go installed
- ✅ PostgreSQL accepting connections
- ✅ RabbitMQ running
- ✅ API server responding
- ✅ Worker processing messages
- ✅ Database state correct
- ✅ Messages sent successfully

**seed_test_data** validates:
- ✅ API server is available
- ✅ Customers created successfully
- ✅ Campaigns created successfully
- ✅ Template preview works

---

## Troubleshooting

### Scripts Won't Execute

**PowerShell Execution Policy Issue:**
```powershell
# Check current policy
Get-ExecutionPolicy

# Allow scripts (run as Administrator)
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# Or run with bypass
powershell -ExecutionPolicy Bypass -File .\scripts\test_workflow.ps1
```

**Bash Permission Denied:**
```bash
# Make scripts executable
chmod +x scripts/*.sh

# Verify permissions
ls -l scripts/
```

### API Not Available

**Error**: `API is not available at http://localhost:8080`

**Solution**:
1. Check if API server is running: `go run cmd/api/main.go`
2. Check port is correct: Default is 8080
3. Check no firewall blocking: Test with `curl http://localhost:8080/api/health`

### Docker Services Not Starting

**Error**: `PostgreSQL failed to start within timeout`

**Solution**:
```bash
# Check Docker is running
docker ps

# Check Docker Compose file exists
ls docker-compose.yml

# View logs
docker-compose logs db
docker-compose logs rabbitmq

# Restart services
docker-compose restart
```

### Migration Errors

**Error**: `Migration failed`

**Solution**:
```bash
# Check DATABASE_URL in .env
cat .env | grep DATABASE_URL

# Test database connection
docker-compose exec db psql -U postgres -d smsleopard -c "SELECT 1;"

# Reset database
docker-compose exec db psql -U postgres -c "DROP DATABASE smsleopard; CREATE DATABASE smsleopard;"
go run cmd/migrate/main.go
```

---

## Advanced Usage

### Running Specific Test Phases

**PowerShell:**
```powershell
# Skip prerequisites and services (already running)
.\scripts\test_workflow.ps1 -SkipPrerequisites -SkipServices

# Only run migrations
.\scripts\test_workflow.ps1 -SkipPrerequisites -SkipServices -SkipSeeding
```

### Custom Test Data Volumes

```powershell
# Create 50 customers, 5 campaigns
.\scripts\seed_test_data.ps1 -CustomerCount 50 -CampaignCount 5
```

```bash
# Create 100 customers, 10 campaigns
./scripts/seed_test_data.sh http://localhost:8080 100 10
```

### Automated CI/CD Integration

For CI/CD pipelines, you can use these scripts with minimal interaction:

```yaml
# Example GitHub Actions
- name: Run Tests
  run: |
    docker-compose up -d db rabbitmq
    sleep 10
    go run cmd/migrate/main.go
    go run cmd/api/main.go &
    go run cmd/worker/main.go &
    sleep 5
    ./scripts/seed_test_data.sh
```

---

## Platform Compatibility

### Windows

- ✅ PowerShell 5.1+
- ✅ PowerShell Core 7+
- ✅ Windows 10/11
- ✅ Windows Server 2016+

**Recommended**: Use PowerShell scripts (`.ps1`)

### Unix/Linux

- ✅ Bash 4.0+
- ✅ Ubuntu 18.04+
- ✅ Debian 9+
- ✅ CentOS 7+
- ✅ RHEL 7+

**Recommended**: Use Bash scripts (`.sh`)

### macOS

- ✅ Bash 3.2+ (default)
- ✅ Bash 5.0+ (via Homebrew)
- ✅ Zsh (with bash compatibility)
- ✅ macOS 10.14+

**Recommended**: Use Bash scripts (`.sh`)

### WSL (Windows Subsystem for Linux)

- ✅ WSL 1
- ✅ WSL 2 (recommended)
- ✅ All Linux distributions

**Recommended**: Use Bash scripts (`.sh`)

---

## Script Maintenance

### Adding New Test Scenarios

To add new test scenarios to `test_workflow`:

1. Add a new function in the script
2. Call it from the `Main` function
3. Add validation queries
4. Update the `TESTING_GUIDE.md`

### Modifying Test Data

To change default test data in `seed_test_data`:

1. Edit the `$testCustomers` / `CUSTOMER_NAMES` array
2. Edit the `$testCampaigns` / `CAMPAIGN_NAMES` array
3. Update counts if needed
4. Test the changes

---

## Related Documentation

- [`docs/TESTING_GUIDE.md`](../docs/TESTING_GUIDE.md) - Comprehensive testing guide
- [`docker-compose.yml`](../docker-compose.yml) - Docker service configuration
- [`.env.example`](../.env.example) - Environment configuration template

---

## Support

For issues or questions:

1. Check the [TESTING_GUIDE.md](../docs/TESTING_GUIDE.md) troubleshooting section
2. Review script output for error messages
3. Check Docker logs: `docker-compose logs`
4. Verify service status: `docker-compose ps`

---

## Version History

- **v1.0** (Phase 5.6) - Initial release
  - End-to-end test workflow script
  - Test data seeding script
  - Windows (PowerShell) and Unix (Bash) versions
  - Comprehensive validation and error handling

---

**Last Updated**: Phase 5.6 - End-to-End Queue Workflow Testing
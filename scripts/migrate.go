package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"smsleopard/internal/config"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Migration represents a database migration
type Migration struct {
	Version   int
	Name      string
	FilePath  string
	Applied   bool
	AppliedAt *time.Time
}

func main() {
	// Load .env file (ignore error if not present)
	_ = godotenv.Load()

	printInfo("=== SMSLeopard Migration Runner ===\n")

	// Parse command
	command := "help"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	// Show help for invalid commands
	if command != "up" && command != "down" && command != "status" && command != "reset" && command != "seed" {
		printUsage()
		if command != "help" {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		printError(fmt.Sprintf("Failed to load configuration: %v", err))
		os.Exit(1)
	}

	// Connect to database
	printInfo("Connecting to database...")
	db, err := sql.Open("postgres", cfg.GetDatabaseDSN())
	if err != nil {
		printError(fmt.Sprintf("Failed to open database connection: %v", err))
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		printError(fmt.Sprintf("Failed to ping database: %v", err))
		os.Exit(1)
	}
	printSuccess("✓ Connected to database\n")

	// Create migration tracking table
	if err := createMigrationTable(db); err != nil {
		printError(fmt.Sprintf("Failed to create migration table: %v", err))
		os.Exit(1)
	}

	// Execute command
	switch command {
	case "up":
		if err := runUp(db); err != nil {
			printError(fmt.Sprintf("Migration failed: %v", err))
			os.Exit(1)
		}
	case "down":
		if err := runDown(db); err != nil {
			printError(fmt.Sprintf("Rollback failed: %v", err))
			os.Exit(1)
		}
	case "status":
		if err := showMigrationStatus(db); err != nil {
			printError(fmt.Sprintf("Failed to show status: %v", err))
			os.Exit(1)
		}
	case "reset":
		if err := runReset(db); err != nil {
			printError(fmt.Sprintf("Reset failed: %v", err))
			os.Exit(1)
		}
	case "seed":
		if err := runSeedMigrations(db); err != nil {
			printError(fmt.Sprintf("Seed failed: %v", err))
			os.Exit(1)
		}
	}

	printInfo("\n✨ Operation completed successfully!")
}

// createMigrationTable creates the schema_migrations tracking table
func createMigrationTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// getAppliedMigrations retrieves all applied migrations from database
func getAppliedMigrations(db *sql.DB) (map[int]Migration, error) {
	query := `SELECT version, name, applied_at FROM schema_migrations ORDER BY version`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]Migration)
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.Name, &m.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		m.Applied = true
		applied[m.Version] = m
	}

	return applied, nil
}

// getMigrationFiles scans the migrations directory and returns all migration files
func getMigrationFiles(dir string) ([]Migration, error) {
	var migrations []Migration

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return migrations, nil
	}

	// Read directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Pattern: 001_name.sql
	pattern := regexp.MustCompile(`^(\d{3})_(.+)\.sql$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := pattern.FindStringSubmatch(file.Name())
		if len(matches) != 3 {
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     matches[2],
			FilePath: filepath.Join(dir, file.Name()),
			Applied:  false,
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// runUp applies all pending migrations
func runUp(db *sql.DB) error {
	printInfo("Running pending migrations...\n")

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Get all migration files
	migrations, err := getMigrationFiles("migrations")
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		printWarning("No migration files found in migrations/ directory")
		return nil
	}

	// Filter pending migrations
	var pending []Migration
	for _, m := range migrations {
		if _, exists := applied[m.Version]; !exists {
			pending = append(pending, m)
		}
	}

	if len(pending) == 0 {
		printSuccess("✓ All migrations are up to date")
		return nil
	}

	// Apply pending migrations
	for _, migration := range pending {
		if err := runMigration(db, migration); err != nil {
			return fmt.Errorf("failed to apply migration %03d_%s: %w", migration.Version, migration.Name, err)
		}
	}

	printSuccess(fmt.Sprintf("\n✓ Successfully applied %d migration(s)", len(pending)))
	return nil
}

// runMigration executes a single migration file
func runMigration(db *sql.DB, migration Migration) error {
	printInfo(fmt.Sprintf("Applying migration %03d_%s...", migration.Version, migration.Name))

	// Read migration file
	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration in tracking table
	_, err = tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
		migration.Version,
		migration.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	printSuccess(fmt.Sprintf("  ✓ Migration %03d applied successfully", migration.Version))
	return nil
}

// runDown rolls back the last applied migration
func runDown(db *sql.DB) error {
	printInfo("Rolling back last migration...\n")

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		printWarning("No migrations to rollback")
		return nil
	}

	// Find the highest version
	var lastVersion int
	for version := range applied {
		if version > lastVersion {
			lastVersion = version
		}
	}

	lastMigration := applied[lastVersion]

	// Perform rollback
	if err := rollbackMigration(db, lastMigration.Version); err != nil {
		return fmt.Errorf("failed to rollback migration %03d_%s: %w", lastMigration.Version, lastMigration.Name, err)
	}

	printSuccess(fmt.Sprintf("✓ Successfully rolled back migration %03d_%s", lastMigration.Version, lastMigration.Name))
	return nil
}

// rollbackMigration rolls back a specific migration by dropping its tables
func rollbackMigration(db *sql.DB, version int) error {
	var dropSQL string

	// Define rollback logic for each migration version
	switch version {
	case 1:
		dropSQL = "DROP TABLE IF EXISTS customers CASCADE;"
	case 2:
		dropSQL = "DROP TABLE IF EXISTS campaigns CASCADE;"
	case 3:
		dropSQL = "DROP TABLE IF EXISTS outbound_messages CASCADE;"
	default:
		return fmt.Errorf("no rollback defined for migration version %d", version)
	}

	printInfo(fmt.Sprintf("Rolling back migration %03d...", version))

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback SQL
	if _, err := tx.Exec(dropSQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove from tracking table
	_, err = tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	printSuccess(fmt.Sprintf("  ✓ Migration %03d rolled back", version))
	return nil
}

// runReset rolls back all migrations and reapplies them
func runReset(db *sql.DB) error {
	printWarning("Resetting database (rollback all + reapply all)...\n")

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Rollback all migrations in reverse order
	if len(applied) > 0 {
		printInfo("Rolling back all migrations...")

		// Get versions sorted in descending order
		versions := make([]int, 0, len(applied))
		for version := range applied {
			versions = append(versions, version)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(versions)))

		// Rollback each migration
		for _, version := range versions {
			if err := rollbackMigration(db, version); err != nil {
				return err
			}
		}

		printSuccess("\n✓ All migrations rolled back\n")
	}

	// Reapply all migrations
	printInfo("Reapplying all migrations...")
	if err := runUp(db); err != nil {
		return err
	}

	return nil
}

// showMigrationStatus displays the current migration status
func showMigrationStatus(db *sql.DB) error {
	printInfo("Migration Status:\n")

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Get all migration files
	migrations, err := getMigrationFiles("migrations")
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		printWarning("No migration files found in migrations/ directory")
		return nil
	}

	// Print table header
	fmt.Printf("%s%-10s %-40s %-12s %-20s%s\n",
		colorBold, "VERSION", "NAME", "STATUS", "APPLIED AT", colorReset)
	fmt.Println(strings.Repeat("-", 85))

	// Print each migration
	appliedCount := 0
	for _, migration := range migrations {
		if appliedMig, exists := applied[migration.Version]; exists {
			migration.Applied = true
			migration.AppliedAt = appliedMig.AppliedAt
			appliedCount++
		}

		version := fmt.Sprintf("%03d", migration.Version)
		status := "pending"
		statusColor := colorYellow
		appliedAt := "-"

		if migration.Applied {
			status = "applied"
			statusColor = colorGreen
			if migration.AppliedAt != nil {
				appliedAt = migration.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}

		fmt.Printf("%-10s %-40s %s%-12s%s %-20s\n",
			version, migration.Name, statusColor, status, colorReset, appliedAt)
	}

	// Print summary
	fmt.Println(strings.Repeat("-", 85))
	printInfo(fmt.Sprintf("\nSummary: %d/%d migrations applied", appliedCount, len(migrations)))

	return nil
}

// runSeedMigrations executes seed data migrations
func runSeedMigrations(db *sql.DB) error {
	printInfo("Running seed migrations...\n")

	// Get seed migration files
	seedMigrations, err := getMigrationFiles("migrations/seed")
	if err != nil {
		return err
	}

	if len(seedMigrations) == 0 {
		printWarning("No seed migration files found in migrations/seed/ directory")
		return nil
	}

	// Run each seed migration
	for _, migration := range seedMigrations {
		printInfo(fmt.Sprintf("Running seed %03d_%s...", migration.Version, migration.Name))

		// Read seed file
		content, err := os.ReadFile(migration.FilePath)
		if err != nil {
			return fmt.Errorf("failed to read seed file: %w", err)
		}

		// Execute seed SQL (no transaction tracking needed for seeds)
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute seed SQL: %w", err)
		}

		printSuccess(fmt.Sprintf("  ✓ Seed %03d applied successfully", migration.Version))
	}

	printSuccess(fmt.Sprintf("\n✓ Successfully ran %d seed migration(s)", len(seedMigrations)))
	return nil
}

// Helper functions for colored output

func printSuccess(msg string) {
	fmt.Printf("%s%s%s\n", colorGreen, msg, colorReset)
}

func printError(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", colorRed, msg, colorReset)
}

func printInfo(msg string) {
	fmt.Printf("%s%s%s\n", colorCyan, msg, colorReset)
}

func printWarning(msg string) {
	fmt.Printf("%s%s%s\n", colorYellow, msg, colorReset)
}

func printUsage() {
	printInfo("=== SMSLeopard Migration Runner ===\n")
	fmt.Println("Usage: go run scripts/migrate.go [command]")
	fmt.Println("\nCommands:")
	fmt.Println("  up       - Apply all pending migrations")
	fmt.Println("  down     - Rollback the last applied migration")
	fmt.Println("  status   - Show current migration status")
	fmt.Println("  reset    - Rollback all migrations and reapply them")
	fmt.Println("  seed     - Run seed data migrations only")
	fmt.Println("  help     - Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  go run scripts/migrate.go up")
	fmt.Println("  go run scripts/migrate.go status")
	fmt.Println("  go run scripts/migrate.go down")
	fmt.Println("  go run scripts/migrate.go reset")
	fmt.Println("  go run scripts/migrate.go seed")
	fmt.Println("\nMigration Files:")
	fmt.Println("  Schema:  migrations/*.sql (001_*, 002_*, 003_*)")
	fmt.Println("  Seeds:   migrations/seed/*.sql")
	fmt.Println("\nNotes:")
	fmt.Println("  - Migrations are tracked in the 'schema_migrations' table")
	fmt.Println("  - Each migration runs in a transaction")
	fmt.Println("  - Rollback drops tables in reverse dependency order")
	fmt.Println("  - Seed migrations can be run independently with 'seed' command")
}

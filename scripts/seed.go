package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
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
)

// Command-line flags
var (
	customersCount = flag.Int("customers", 12, "Number of customers to create")
	campaignsCount = flag.Int("campaigns", 3, "Number of campaigns to create")
	clearData      = flag.Bool("clear", false, "Clear existing seed data before inserting")
	showHelp       = flag.Bool("help", false, "Show usage information")
)

func main() {
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	// Load .env file (ignore error if not present)
	_ = godotenv.Load()

	printInfo("=== SMSLeopard Database Seeder ===\n")

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

	// Clear data if requested
	if *clearData {
		if err := clearSeedData(db); err != nil {
			printError(fmt.Sprintf("Failed to clear seed data: %v", err))
			os.Exit(1)
		}
	}

	// Seed customers
	customersCreated, err := seedCustomers(db, *customersCount)
	if err != nil {
		printError(fmt.Sprintf("Failed to seed customers: %v", err))
		os.Exit(1)
	}

	// Seed campaigns
	campaignsCreated, err := seedCampaigns(db, *campaignsCount)
	if err != nil {
		printError(fmt.Sprintf("Failed to seed campaigns: %v", err))
		os.Exit(1)
	}

	// Print summary
	printInfo("\n=== Seeding Summary ===")
	printSuccess(fmt.Sprintf("✓ Customers created: %d", customersCreated))
	printSuccess(fmt.Sprintf("✓ Campaigns created: %d", campaignsCreated))
	printInfo("\nSeeding completed successfully!")
}

// clearSeedData removes existing seed data
func clearSeedData(db *sql.DB) error {
	printWarning("Clearing existing seed data...")

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete campaigns with Go-seeded naming pattern
	_, err = tx.Exec("DELETE FROM campaigns WHERE name LIKE 'Weekend Sale%' OR name LIKE 'New Arrivals%' OR name LIKE 'Customer Appreciation%'")
	if err != nil {
		return fmt.Errorf("failed to delete campaigns: %w", err)
	}

	// Delete customers with Go-seeded phone pattern (+2547000001XX)
	_, err = tx.Exec("DELETE FROM customers WHERE phone LIKE '+254700010%'")
	if err != nil {
		return fmt.Errorf("failed to delete customers: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	printSuccess("✓ Seed data cleared\n")
	return nil
}

// seedCustomers generates and inserts customer data
func seedCustomers(db *sql.DB, count int) (int, error) {
	printInfo(fmt.Sprintf("Seeding %d customers...", count))

	// Realistic Kenyan data
	firstNames := []string{"Michael", "Sophia", "James", "Olivia", "Daniel", "Emma", "Benjamin", "Ava", "Lucas", "Mia", "Noah", "Isabella", "William", "Charlotte", "Alexander"}
	lastNames := []string{"Kamau", "Wanjiku", "Ochieng", "Atieno", "Mwangi", "Akinyi", "Kipchoge", "Chebet", "Kiptoo", "Jepchirchir", "Mutua", "Mumbua", "Omondi", "Adhiambo", "Nzomo"}
	locations := []string{"Nairobi", "Mombasa", "Kisumu", "Eldoret", "Nakuru", "Thika", "Nyeri", "Kitale", "Machakos", "Kakamega", "Malindi", "Garissa"}
	products := []string{"Smartphones", "Laptops", "Tablets", "Cameras", "Headphones", "Watches", "Speakers", "Smartwatches", "Gaming Consoles", "TVs", "Printers", "Routers"}

	created := 0
	for i := 1; i <= count; i++ {
		phone := fmt.Sprintf("+254700010%03d", i)

		// Generate varied data with some NULL fields
		var firstName, lastName, location, product *string

		// Most customers have first name
		if i%10 != 1 { // 90% have first name
			firstName = stringPtr(firstNames[i%len(firstNames)])
		}

		// Some customers have last name
		if i%3 != 0 { // 66% have last name
			lastName = stringPtr(lastNames[i%len(lastNames)])
		}

		// Some customers have location
		if i%4 != 0 { // 75% have location
			location = stringPtr(locations[i%len(locations)])
		}

		// Some customers have preferred product
		if i%5 != 0 { // 80% have preferred product
			product = stringPtr(products[i%len(products)])
		}

		// Insert with ON CONFLICT for idempotency
		query := `
			INSERT INTO customers (phone, first_name, last_name, location, preferred_product)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (phone) DO NOTHING
		`

		result, err := db.Exec(query, phone, firstName, lastName, location, product)
		if err != nil {
			return created, fmt.Errorf("failed to insert customer %s: %w", phone, err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			created++
		}
	}

	printSuccess(fmt.Sprintf("✓ Seeded %d customers (skipped %d existing)", created, count-created))
	return created, nil
}

// seedCampaigns generates and inserts campaign data
func seedCampaigns(db *sql.DB, count int) (int, error) {
	printInfo(fmt.Sprintf("Seeding %d campaigns...", count))

	// Define campaign templates with different variations
	campaigns := []struct {
		name        string
		channel     string
		status      string
		template    string
		scheduledAt *time.Time
	}{
		{
			name:        "Weekend Sale",
			channel:     "sms",
			status:      "scheduled",
			template:    "Hi {first_name}! Special weekend offers in {location}. Visit us today! 🎉",
			scheduledAt: timePtr(time.Now().Add(48 * time.Hour)),
		},
		{
			name:     "New Arrivals Alert",
			channel:  "whatsapp",
			status:   "draft",
			template: "Hello {first_name} {last_name}! New {preferred_product} just arrived in {location}. Check them out now! 🆕",
		},
		{
			name:     "Customer Appreciation",
			channel:  "sms",
			status:   "sent",
			template: "Thank you {first_name} for being a valued customer! 🙏",
		},
		{
			name:     "Flash Sale Alert",
			channel:  "whatsapp",
			status:   "draft",
			template: "⚡ Flash Sale! Hi {first_name}, get 50% off on {preferred_product} today only!",
		},
		{
			name:        "Location Special Offer",
			channel:     "sms",
			status:      "scheduled",
			template:    "Exclusive offer for {location} residents! {first_name}, visit us this week.",
			scheduledAt: timePtr(time.Now().Add(24 * time.Hour)),
		},
	}

	created := 0
	for i := 0; i < count && i < len(campaigns); i++ {
		campaign := campaigns[i]

		query := `
			INSERT INTO campaigns (name, channel, status, base_template, scheduled_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (name) DO NOTHING
		`

		result, err := db.Exec(query, campaign.name, campaign.channel, campaign.status, campaign.template, campaign.scheduledAt)
		if err != nil {
			return created, fmt.Errorf("failed to insert campaign %s: %w", campaign.name, err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			created++
		}
	}

	printSuccess(fmt.Sprintf("✓ Seeded %d campaigns (skipped %d existing)", created, count-created))
	return created, nil
}

// Helper functions

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}

// printSuccess prints a success message in green
func printSuccess(msg string) {
	fmt.Printf("%s%s%s\n", colorGreen, msg, colorReset)
}

// printError prints an error message in red
func printError(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", colorRed, msg, colorReset)
}

// printInfo prints an info message in cyan
func printInfo(msg string) {
	fmt.Printf("%s%s%s\n", colorCyan, msg, colorReset)
}

// printWarning prints a warning message in yellow
func printWarning(msg string) {
	fmt.Printf("%s%s%s\n", colorYellow, msg, colorReset)
}

// printUsage displays usage information
func printUsage() {
	printInfo("=== SMSLeopard Database Seeder ===\n")
	fmt.Println("Usage: go run scripts/seed.go [flags]")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  go run scripts/seed.go")
	fmt.Println("  go run scripts/seed.go -customers=20 -campaigns=5")
	fmt.Println("  go run scripts/seed.go -clear")
	fmt.Println("  go run scripts/seed.go -clear -customers=50")
	fmt.Println("\nNotes:")
	fmt.Println("  - Customers use phone pattern: +2547000010XXX (different from SQL seeds)")
	fmt.Println("  - The script is idempotent - running multiple times won't create duplicates")
	fmt.Println("  - Use -clear to remove existing seed data before inserting new data")
}

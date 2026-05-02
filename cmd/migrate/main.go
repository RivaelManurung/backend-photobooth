package main

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("🔄 Starting Database Migration...")
	fmt.Println("=====================================")

	// Load configuration
	cfg := config.LoadConfig()

	// Display database info
	fmt.Printf("📊 Database: %s\n", cfg.Database.DBName)
	fmt.Printf("🖥️  Host: %s:%s\n", cfg.Database.Host, cfg.Database.Port)
	fmt.Printf("👤 User: %s\n", cfg.Database.User)
	fmt.Println("=====================================")

	// Initialize database connection
	fmt.Println("\n🔌 Connecting to database...")
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	fmt.Println("✅ Database connected successfully")

	// Run migrations
	fmt.Println("\n🔄 Running migrations...")
	if err := database.AutoMigrate(); err != nil {
		log.Fatal("❌ Migration failed:", err)
	}

	fmt.Println("\n✅ Migration completed successfully!")
	fmt.Println("=====================================")
	fmt.Println("\n📋 Tables created:")
	fmt.Println("  1. ✅ users")
	fmt.Println("  2. ✅ templates")
	fmt.Println("  3. ✅ photos")
	fmt.Println("  4. ✅ sessions")
	fmt.Println("  5. ✅ orders")
	fmt.Println("  6. ✅ transactions")
	fmt.Println("  7. ✅ qris_payments (GoPay QRIS)")
	fmt.Println("  8. ✅ promo_codes")
	fmt.Println("  9. ✅ promo_usages")
	fmt.Println(" 10. ✅ analytics")
	fmt.Println(" 11. ✅ daily_stats")
	fmt.Println(" 12. ✅ audit_logs")
	fmt.Println(" 13. ✅ two_factor_auths")
	fmt.Println(" 14. ✅ two_factor_logs")
	fmt.Println("\n🎉 Database is ready to use!")

	os.Exit(0)
}

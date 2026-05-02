package main

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/models"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("🌱 Starting Database Seeding...")
	fmt.Println("=====================================")

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	fmt.Println("🔌 Connecting to database...")
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	fmt.Println("✅ Database connected")

	// Seed admin user
	fmt.Println("\n👤 Creating admin user...")
	if err := seedAdminUser(); err != nil {
		log.Printf("⚠️  Admin user: %v", err)
	} else {
		fmt.Println("✅ Admin user created")
	}

	// Seed templates
	fmt.Println("\n🎨 Creating sample templates...")
	if err := seedTemplates(); err != nil {
		log.Printf("⚠️  Templates: %v", err)
	} else {
		fmt.Println("✅ Sample templates created")
	}

	// Seed promo codes
	fmt.Println("\n🎟️  Creating promo codes...")
	if err := seedPromoCodes(); err != nil {
		log.Printf("⚠️  Promo codes: %v", err)
	} else {
		fmt.Println("✅ Promo codes created")
	}

	fmt.Println("\n=====================================")
	fmt.Println("🎉 Database seeding completed!")
	fmt.Println("\n📋 Created:")
	fmt.Println("  • Admin user: admin@photobooth.com / admin123")
	fmt.Println("  • 8 sample templates")
	fmt.Println("  • 3 promo codes")
	fmt.Println("\n💡 You can now login with admin credentials")
}

func seedAdminUser() error {
	// Check if admin exists
	var existingAdmin models.User
	err := database.DB.Where("email = ?", "admin@photobooth.com").First(&existingAdmin).Error
	if err == nil {
		// Admin exists, delete it first to recreate with correct password
		fmt.Println("⚠️  Existing admin found, deleting to recreate...")
		if err := database.DB.Unscoped().Delete(&existingAdmin).Error; err != nil {
			return fmt.Errorf("failed to delete existing admin: %v", err)
		}
	}

	// Create admin user - let BeforeCreate hook handle password hashing
	admin := models.User{
		Name:             "Admin",
		Email:            "admin@photobooth.com",
		Password:         "admin123", // Plain password - will be hashed by BeforeCreate hook
		Phone:            "081234567890",
		Role:             "admin",
		SubscriptionPlan: "premium",
		IsActive:         true,
	}

	endDate := time.Now().AddDate(1, 0, 0)
	admin.SubscriptionEnd = &endDate

	return database.DB.Create(&admin).Error
}

func seedTemplates() error {
	templates := []models.Template{
		{
			Name:         "Classic Frame",
			Slug:         "classic-frame",
			Description:  "Simple and elegant classic frame",
			Category:     "classic",
			LayoutType:   "single",
			PhotoCount:   1,
			Orientation:  "portrait",
			IsPremium:    false,
			IsActive:     true,
			RequiredPlan: "free",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":400,"height":600,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Birthday Party",
			Slug:         "birthday-party",
			Description:  "Fun birthday celebration template",
			Category:     "birthday",
			LayoutType:   "single",
			PhotoCount:   1,
			Orientation:  "portrait",
			IsPremium:    true,
			IsActive:     true,
			RequiredPlan: "basic",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":400,"height":600,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Wedding Elegance",
			Slug:         "wedding-elegance",
			Description:  "Elegant wedding photo template",
			Category:     "wedding",
			LayoutType:   "single",
			PhotoCount:   1,
			Orientation:  "portrait",
			IsPremium:    true,
			IsActive:     true,
			RequiredPlan: "premium",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":400,"height":600,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Photo Strip 4x",
			Slug:         "photo-strip-4x",
			Description:  "Classic 4-photo strip layout",
			Category:     "strip",
			LayoutType:   "strip",
			PhotoCount:   4,
			Orientation:  "portrait",
			IsPremium:    false,
			IsActive:     true,
			RequiredPlan: "free",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":300,"height":200,"rotation":0,"z_index":1},{"index":1,"x":50,"y":270,"width":300,"height":200,"rotation":0,"z_index":1},{"index":2,"x":50,"y":490,"width":300,"height":200,"rotation":0,"z_index":1},{"index":3,"x":50,"y":710,"width":300,"height":200,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Collage 2x2",
			Slug:         "collage-2x2",
			Description:  "2x2 photo collage",
			Category:     "collage",
			LayoutType:   "grid",
			PhotoCount:   4,
			Orientation:  "square",
			IsPremium:    true,
			IsActive:     true,
			RequiredPlan: "basic",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":300,"height":300,"rotation":0,"z_index":1},{"index":1,"x":370,"y":50,"width":300,"height":300,"rotation":0,"z_index":1},{"index":2,"x":50,"y":370,"width":300,"height":300,"rotation":0,"z_index":1},{"index":3,"x":370,"y":370,"width":300,"height":300,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Vintage Polaroid",
			Slug:         "vintage-polaroid",
			Description:  "Retro polaroid style",
			Category:     "vintage",
			LayoutType:   "single",
			PhotoCount:   1,
			Orientation:  "portrait",
			IsPremium:    true,
			IsActive:     true,
			RequiredPlan: "premium",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":400,"height":500,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Modern Minimal",
			Slug:         "modern-minimal",
			Description:  "Clean and modern design",
			Category:     "modern",
			LayoutType:   "single",
			PhotoCount:   1,
			Orientation:  "portrait",
			IsPremium:    true,
			IsActive:     true,
			RequiredPlan: "basic",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":400,"height":600,"rotation":0,"z_index":1}]`,
		},
		{
			Name:         "Holiday Special",
			Slug:         "holiday-special",
			Description:  "Festive holiday template",
			Category:     "holiday",
			LayoutType:   "single",
			PhotoCount:   1,
			Orientation:  "portrait",
			IsPremium:    true,
			IsActive:     true,
			RequiredPlan: "premium",
			PhotoZones:   `[{"index":0,"x":50,"y":50,"width":400,"height":600,"rotation":0,"z_index":1}]`,
		},
	}

	for _, template := range templates {
		var count int64
		database.DB.Model(&models.Template{}).Where("slug = ?", template.Slug).Count(&count)
		if count == 0 {
			if err := database.DB.Create(&template).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func seedPromoCodes() error {
	now := time.Now()
	promoCodes := []models.PromoCode{
		{
			Code:            "WELCOME10",
			Description:     "Welcome discount 10%",
			Type:            "percentage",
			DiscountPercent: 10,
			MaxDiscount:     50000,
			MinPurchase:     0,
			MaxUses:         100,
			UsedCount:       0,
			MaxUsesPerUser:  1,
			IsActive:        true,
			StartsAt:        now,
			ExpiresAt:       now.AddDate(0, 3, 0),
			ApplicablePlans: "basic,premium",
		},
		{
			Code:            "FIRST50",
			Description:     "First time user discount Rp 50.000",
			Type:            "fixed",
			DiscountAmount:  50000,
			MinPurchase:     100000,
			MaxUses:         50,
			UsedCount:       0,
			MaxUsesPerUser:  1,
			FirstTimeOnly:   true,
			IsActive:        true,
			StartsAt:        now,
			ExpiresAt:       now.AddDate(0, 1, 0),
			ApplicablePlans: "premium",
		},
		{
			Code:            "YEARLY20",
			Description:     "20% off for yearly subscription",
			Type:            "percentage",
			DiscountPercent: 20,
			MaxDiscount:     200000,
			MinPurchase:     0,
			MaxUses:         0, // unlimited
			UsedCount:       0,
			MaxUsesPerUser:  0, // unlimited
			IsActive:        true,
			StartsAt:        now,
			ExpiresAt:       now.AddDate(1, 0, 0),
			ApplicablePlans: "premium",
		},
	}

	for _, promo := range promoCodes {
		var count int64
		database.DB.Model(&models.PromoCode{}).Where("code = ?", promo.Code).Count(&count)
		if count == 0 {
			if err := database.DB.Create(&promo).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

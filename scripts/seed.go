package main

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/models"
	"log"
	"time"
)

func main() {
	// Load config
	cfg := config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	log.Println("🌱 Starting database seeding...")

	// Seed users
	seedUsers()

	// Seed templates
	seedTemplates()

	log.Println("✅ Database seeding completed!")
}

func seedUsers() {
	log.Println("Seeding users...")

	users := []models.User{
		{
			Name:             "Admin User",
			Email:            "admin@photobooth.com",
			Password:         "admin123",
			Role:             "admin",
			IsActive:         true,
			IsEmailVerified:  true,
			SubscriptionPlan: "premium",
		},
		{
			Name:             "John Doe",
			Email:            "john@example.com",
			Password:         "password123",
			Role:             "user",
			IsActive:         true,
			IsEmailVerified:  true,
			SubscriptionPlan: "basic",
		},
		{
			Name:             "Jane Smith",
			Email:            "jane@example.com",
			Password:         "password123",
			Role:             "user",
			IsActive:         true,
			IsEmailVerified:  true,
			SubscriptionPlan: "free",
		},
	}

	for _, user := range users {
		var existing models.User
		if err := database.DB.Where("email = ?", user.Email).First(&existing).Error; err != nil {
			// User doesn't exist, create it
			if err := database.DB.Create(&user).Error; err != nil {
				log.Printf("Failed to create user %s: %v", user.Email, err)
			} else {
				log.Printf("✓ Created user: %s", user.Email)
			}
		} else {
			log.Printf("⊘ User already exists: %s", user.Email)
		}
	}
}

func seedTemplates() {
	log.Println("Seeding templates...")

	templates := []models.Template{
		{
			Name:             "Classic White",
			Slug:             "classic-white",
			Description:      "Timeless white background with elegant borders",
			Category:         "classic",
			BackgroundColor:  "#ffffff",
			TextColor:        "#2d3436",
			BorderStyle:      "1px solid #dfe6e9",
			LayoutType:       "strip",
			PhotoCount:       4,
			Orientation:      "portrait",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    true,
			IsPremium:        false,
			IsActive:         true,
			IsFeatured:       true,
			RequiredPlan:     "free",
			Tags:             "classic,white,elegant",
			PhotoZones:       `[{"index":0,"x":20,"y":100,"width":360,"height":270,"rotation":0,"z_index":1},{"index":1,"x":20,"y":390,"width":360,"height":270,"rotation":0,"z_index":1},{"index":2,"x":20,"y":680,"width":360,"height":270,"rotation":0,"z_index":1},{"index":3,"x":20,"y":970,"width":360,"height":270,"rotation":0,"z_index":1}]`,
		},
		{
			Name:             "Charcoal Matte",
			Slug:             "charcoal-matte",
			Description:      "Modern dark theme with matte finish",
			Category:         "modern",
			BackgroundColor:  "#2d3436",
			TextColor:        "rgba(255,255,255,0.8)",
			LayoutType:       "strip",
			PhotoCount:       4,
			Orientation:      "portrait",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    true,
			IsPremium:        false,
			IsActive:         true,
			IsFeatured:       true,
			RequiredPlan:     "free",
			Tags:             "modern,dark,matte",
			PhotoZones:       `[{"index":0,"x":20,"y":100,"width":360,"height":270,"rotation":0,"z_index":1},{"index":1,"x":20,"y":390,"width":360,"height":270,"rotation":0,"z_index":1},{"index":2,"x":20,"y":680,"width":360,"height":270,"rotation":0,"z_index":1},{"index":3,"x":20,"y":970,"width":360,"height":270,"rotation":0,"z_index":1}]`,
		},
		{
			Name:             "Vintage Film",
			Slug:             "vintage-film",
			Description:      "Classic film strip with sprocket holes",
			Category:         "vintage",
			BackgroundColor:  "#1a1a1a",
			TextColor:        "#ffffff",
			LayoutType:       "strip",
			PhotoCount:       4,
			Orientation:      "portrait",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    false,
			IsPremium:        true,
			IsActive:         true,
			IsFeatured:       true,
			RequiredPlan:     "basic",
			Tags:             "vintage,film,retro",
			PhotoZones:       `[{"index":0,"x":20,"y":100,"width":360,"height":270,"rotation":0,"z_index":1},{"index":1,"x":20,"y":390,"width":360,"height":270,"rotation":0,"z_index":1},{"index":2,"x":20,"y":680,"width":360,"height":270,"rotation":0,"z_index":1},{"index":3,"x":20,"y":970,"width":360,"height":270,"rotation":0,"z_index":1}]`,
		},
		{
			Name:             "Pop Art Love",
			Slug:             "pop-art",
			Description:      "Vibrant pop art style with hearts and colors",
			Category:         "party",
			BackgroundColor:  "#48dbfb",
			TextColor:        "#000",
			LayoutType:       "grid",
			PhotoCount:       4,
			Orientation:      "square",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    true,
			IsPremium:        true,
			IsActive:         true,
			IsFeatured:       false,
			RequiredPlan:     "basic",
			Tags:             "pop,art,colorful,party",
			PhotoZones:       `[{"index":0,"x":10,"y":10,"width":190,"height":190,"rotation":0,"z_index":1},{"index":1,"x":210,"y":10,"width":190,"height":190,"rotation":0,"z_index":1},{"index":2,"x":10,"y":210,"width":190,"height":190,"rotation":0,"z_index":1},{"index":3,"x":210,"y":210,"width":190,"height":190,"rotation":0,"z_index":1}]`,
		},
		{
			Name:             "Green Picnic",
			Slug:             "picnic-green",
			Description:      "Fresh green theme perfect for outdoor events",
			Category:         "nature",
			BackgroundColor:  "#e3f2fd",
			TextColor:        "#2e7d32",
			LayoutType:       "strip",
			PhotoCount:       3,
			Orientation:      "landscape",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    true,
			IsPremium:        true,
			IsActive:         true,
			IsFeatured:       false,
			RequiredPlan:     "premium",
			Tags:             "nature,green,outdoor,picnic",
			PhotoZones:       `[{"index":0,"x":20,"y":20,"width":360,"height":240,"rotation":0,"z_index":1},{"index":1,"x":20,"y":280,"width":360,"height":240,"rotation":0,"z_index":1},{"index":2,"x":20,"y":540,"width":360,"height":240,"rotation":0,"z_index":1}]`,
		},
		{
			Name:             "Birthday Party",
			Slug:             "birthday-party",
			Description:      "Festive birthday theme with balloons and confetti",
			Category:         "party",
			BackgroundColor:  "#fff3cd",
			TextColor:        "#d35400",
			LayoutType:       "collage",
			PhotoCount:       4,
			Orientation:      "portrait",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    true,
			IsPremium:        true,
			IsActive:         true,
			IsFeatured:       true,
			RequiredPlan:     "premium",
			Tags:             "birthday,party,celebration,festive",
			PhotoZones:       `[{"index":0,"x":20,"y":100,"width":180,"height":180,"rotation":-5,"z_index":1},{"index":1,"x":220,"y":80,"width":180,"height":180,"rotation":5,"z_index":2},{"index":2,"x":20,"y":300,"width":180,"height":180,"rotation":3,"z_index":1},{"index":3,"x":220,"y":320,"width":180,"height":180,"rotation":-3,"z_index":2}]`,
		},
		{
			Name:             "SKA Checker",
			Slug:             "checker-ska",
			Description:      "Black and white checkered pattern",
			Category:         "pattern",
			BackgroundColor:  "#fff",
			TextColor:        "#000",
			LayoutType:       "strip",
			PhotoCount:       4,
			Orientation:      "portrait",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    false,
			IsPremium:        true,
			IsActive:         true,
			IsFeatured:       false,
			RequiredPlan:     "premium",
			Tags:             "pattern,checker,ska,retro",
			PhotoZones:       `[{"index":0,"x":20,"y":100,"width":360,"height":270,"rotation":0,"z_index":1},{"index":1,"x":20,"y":390,"width":360,"height":270,"rotation":0,"z_index":1},{"index":2,"x":20,"y":680,"width":360,"height":270,"rotation":0,"z_index":1},{"index":3,"x":20,"y":970,"width":360,"height":270,"rotation":0,"z_index":1}]`,
		},
		{
			Name:             "Neon Nights",
			Slug:             "neon-nights",
			Description:      "Cyberpunk neon glow effect",
			Category:         "modern",
			BackgroundColor:  "#000",
			TextColor:        "#00ff00",
			LayoutType:       "strip",
			PhotoCount:       4,
			Orientation:      "portrait",
			AllowFilters:     true,
			AllowTextEdit:    true,
			AllowStickers:    true,
			IsPremium:        true,
			IsActive:         true,
			IsFeatured:       true,
			RequiredPlan:     "premium",
			Tags:             "neon,cyberpunk,modern,glow",
			PhotoZones:       `[{"index":0,"x":20,"y":100,"width":360,"height":270,"rotation":0,"z_index":1},{"index":1,"x":20,"y":390,"width":360,"height":270,"rotation":0,"z_index":1},{"index":2,"x":20,"y":680,"width":360,"height":270,"rotation":0,"z_index":1},{"index":3,"x":20,"y":970,"width":360,"height":270,"rotation":0,"z_index":1}]`,
		},
	}

	for _, template := range templates {
		var existing models.Template
		if err := database.DB.Where("slug = ?", template.Slug).First(&existing).Error; err != nil {
			// Template doesn't exist, create it
			if err := database.DB.Create(&template).Error; err != nil {
				log.Printf("Failed to create template %s: %v", template.Name, err)
			} else {
				log.Printf("✓ Created template: %s", template.Name)
			}
		} else {
			log.Printf("⊘ Template already exists: %s", template.Name)
		}
	}

	// Seed daily stats
	seedDailyStats()
}

func seedDailyStats() {
	log.Println("Seeding daily stats...")

	// Create stats for last 30 days
	for i := 0; i < 30; i++ {
		date := time.Now().AddDate(0, 0, -i)
		stats := models.DailyStats{
			Date:             date,
			NewUsers:         10 + i,
			ActiveUsers:      50 + i*2,
			PhotosCreated:    100 + i*5,
			PhotosDownloaded: 80 + i*4,
			TemplatesUsed:    60 + i*3,
			Revenue:          float64(500000 + i*10000),
			OrdersCount:      5 + i,
		}

		var existing models.DailyStats
		if err := database.DB.Where("date = ?", date.Format("2006-01-02")).First(&existing).Error; err != nil {
			if err := database.DB.Create(&stats).Error; err != nil {
				log.Printf("Failed to create stats for %s: %v", date.Format("2006-01-02"), err)
			}
		}
	}

	log.Println("✓ Created daily stats")
}

package database

import (
	"backendphotobooth/config"
	"backendphotobooth/models"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase initializes the database connection
func InitDatabase(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")

	// Auto migrate models
	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// AutoMigrate runs auto migration for all models
func AutoMigrate() error {
	// Migrate in correct order to handle foreign key dependencies
	return DB.AutoMigrate(
		// Base models without dependencies
		&models.User{},
		&models.Template{},
		
		// Session (referenced by Photo)
		&models.Session{},
		
		// Photo (depends on User, Template, Session)
		&models.Photo{},
		
		// Order and payment models
		&models.Order{},
		&models.Transaction{},
		&models.QRISPayment{},
		
		// Promo models
		&models.PromoCode{},
		&models.PromoUsage{},
		
		// Analytics
		&models.Analytics{},
		&models.DailyStats{},
		
		// Audit and security
		&models.AuditLog{},
		&models.TwoFactorAuth{},
		&models.TwoFactorLog{},
	)
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

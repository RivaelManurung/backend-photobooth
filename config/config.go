package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Payment  PaymentConfig
	Email    EmailConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Port         string
	Environment  string
	AllowOrigins []string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret           string
	AccessExpiration time.Duration
	RefreshExpiration time.Duration
}

type StorageConfig struct {
	Provider        string // "local", "s3", or "supabase"
	LocalPath       string
	S3Bucket        string
	S3Region        string
	S3AccessKey     string
	S3SecretKey     string
	SupabaseURL     string
	SupabaseKey     string
	SupabaseBucket  string
	MaxUploadSize   int64
	AllowedFormats  []string
}

type PaymentConfig struct {
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	MidtransServerKey    string
	MidtransClientKey    string
	MidtransEnvironment  string // "sandbox" or "production"
	
	// GoPay QRIS Configuration
	GoPayMerchantID   string
	GoPaySecretKey    string
	GoPayBaseURL      string
	GoPayCallbackURL  string
	GoPayTerminalID   string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func LoadConfig() *Config {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Environment: getEnv("GIN_MODE", "debug"),
			AllowOrigins: func() []string {
				origins := strings.Split(getEnv("FRONTEND_URL", "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173"), ",")
				for i := range origins {
					origins[i] = strings.TrimSpace(origins[i])
				}
				return origins
			}(),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "photobooth"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessExpiration:  time.Hour * 24,
			RefreshExpiration: time.Hour * 24 * 7,
		},
		Storage: StorageConfig{
			Provider:       getEnv("STORAGE_PROVIDER", "local"),
			LocalPath:      getEnv("UPLOAD_DIR", "./uploads"),
			S3Bucket:       getEnv("S3_BUCKET", ""),
			S3Region:       getEnv("S3_REGION", "us-east-1"),
			S3AccessKey:    getEnv("S3_ACCESS_KEY", ""),
			S3SecretKey:    getEnv("S3_SECRET_KEY", ""),
			SupabaseURL:    getEnv("SUPABASE_URL", ""),
			SupabaseKey:    getEnv("SUPABASE_KEY", ""),
			SupabaseBucket: getEnv("SUPABASE_BUCKET", "photobooth"),
			MaxUploadSize:  getEnvAsInt64("MAX_UPLOAD_SIZE", 10485760), // 10MB
			AllowedFormats: []string{"image/jpeg", "image/png", "image/jpg", "image/webp"},
		},
		Payment: PaymentConfig{
			StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			MidtransServerKey:    getEnv("MIDTRANS_SERVER_KEY", ""),
			MidtransClientKey:    getEnv("MIDTRANS_CLIENT_KEY", ""),
			MidtransEnvironment:  getEnv("MIDTRANS_ENV", "sandbox"),
			
			// GoPay QRIS
			GoPayMerchantID:  getEnv("GOPAY_MERCHANT_ID", ""),
			GoPaySecretKey:   getEnv("GOPAY_SECRET_KEY", ""),
			GoPayBaseURL:     getEnv("GOPAY_BASE_URL", "https://api.gopay.co.id"),
			GoPayCallbackURL: getEnv("GOPAY_CALLBACK_URL", "http://localhost:8080/api/v1/webhooks/gopay"),
			GoPayTerminalID:  getEnv("GOPAY_TERMINAL_ID", "TERMINAL-001"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@photobooth.com"),
			FromName:     getEnv("FROM_NAME", "Photo Booth"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
	}

	// Security Check: Ensure JWT secret is changed in production
	if cfg.Server.Environment == "release" && cfg.JWT.Secret == "your-secret-key-change-in-production" {
		log.Println("⚠️  WARNING: JWT_SECRET is still using the default value in production!")
		log.Println("⚠️  Please set a secure JWT_SECRET environment variable.")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

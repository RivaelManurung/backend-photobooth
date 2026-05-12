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
	Secret            string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

type StorageConfig struct {
	Provider       string // legacy: "local", "s3", or "supabase"
	Driver         string // "local", "minio", or "r2"
	LocalPath      string
	Bucket         string
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	PublicBaseURL  string
	ForcePathStyle bool
	S3Bucket       string
	S3Region       string
	S3AccessKey    string
	S3SecretKey    string
	SupabaseURL    string
	SupabaseKey    string
	SupabaseBucket string
	MaxUploadSize  int64
	MaxImageWidth  int
	MaxImageHeight int
	AllowedFormats []string
}

type PaymentConfig struct {
	Provider               string
	ManualQRISImageURL     string
	ManualQRISInstructions string
	StripeSecretKey        string
	StripePublishableKey   string
	StripeWebhookSecret    string
	MidtransServerKey      string
	MidtransClientKey      string
	MidtransEnvironment    string // "sandbox" or "production"

	// GoPay QRIS Configuration
	GoPayMerchantID  string
	GoPaySecretKey   string
	GoPayBaseURL     string
	GoPayCallbackURL string
	GoPayTerminalID  string
}

type EmailConfig struct {
	Driver       string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type RedisConfig struct {
	Addr     string
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
			Provider:       getEnv("STORAGE_PROVIDER", getEnv("STORAGE_DRIVER", "local")),
			Driver:         getEnv("STORAGE_DRIVER", getEnv("STORAGE_PROVIDER", "local")),
			LocalPath:      getEnv("STORAGE_PATH", getEnv("UPLOAD_DIR", "./uploads")),
			Bucket:         getEnv("STORAGE_BUCKET", getEnv("S3_BUCKET", "")),
			Endpoint:       getEnv("STORAGE_ENDPOINT", ""),
			Region:         getEnv("STORAGE_REGION", getEnv("S3_REGION", "us-east-1")),
			AccessKey:      getEnv("STORAGE_ACCESS_KEY", getEnv("S3_ACCESS_KEY", "")),
			SecretKey:      getEnv("STORAGE_SECRET_KEY", getEnv("S3_SECRET_KEY", "")),
			PublicBaseURL:  getEnv("STORAGE_PUBLIC_BASE_URL", ""),
			ForcePathStyle: getEnvAsBool("STORAGE_FORCE_PATH_STYLE", true),
			S3Bucket:       getEnv("S3_BUCKET", getEnv("STORAGE_BUCKET", "")),
			S3Region:       getEnv("S3_REGION", getEnv("STORAGE_REGION", "us-east-1")),
			S3AccessKey:    getEnv("S3_ACCESS_KEY", getEnv("STORAGE_ACCESS_KEY", "")),
			S3SecretKey:    getEnv("S3_SECRET_KEY", getEnv("STORAGE_SECRET_KEY", "")),
			SupabaseURL:    getEnv("SUPABASE_URL", ""),
			SupabaseKey:    getEnv("SUPABASE_KEY", ""),
			SupabaseBucket: getEnv("SUPABASE_BUCKET", "photobooth"),
			MaxUploadSize:  getEnvAsInt64("MAX_UPLOAD_SIZE", 10485760), // 10MB
			MaxImageWidth:  getEnvAsInt("MAX_IMAGE_WIDTH", 8000),
			MaxImageHeight: getEnvAsInt("MAX_IMAGE_HEIGHT", 8000),
			AllowedFormats: []string{"image/jpeg", "image/png", "image/webp"},
		},
		Payment: PaymentConfig{
			Provider:               getEnv("PAYMENT_PROVIDER", "manual_qris"),
			ManualQRISImageURL:     getEnv("MANUAL_QRIS_IMAGE_URL", ""),
			ManualQRISInstructions: getEnv("MANUAL_QRIS_INSTRUCTIONS", "Scan QRIS, then wait for admin confirmation."),
			StripeSecretKey:        getEnv("STRIPE_SECRET_KEY", ""),
			StripePublishableKey:   getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			StripeWebhookSecret:    getEnv("STRIPE_WEBHOOK_SECRET", ""),
			MidtransServerKey:      getEnv("MIDTRANS_SERVER_KEY", ""),
			MidtransClientKey:      getEnv("MIDTRANS_CLIENT_KEY", ""),
			MidtransEnvironment:    getEnv("MIDTRANS_ENV", "sandbox"),

			// GoPay QRIS
			GoPayMerchantID:  getEnv("GOPAY_MERCHANT_ID", ""),
			GoPaySecretKey:   getEnv("GOPAY_SECRET_KEY", ""),
			GoPayBaseURL:     getEnv("GOPAY_BASE_URL", "https://api.gopay.co.id"),
			GoPayCallbackURL: getEnv("GOPAY_CALLBACK_URL", "http://localhost:8080/api/v1/webhooks/gopay"),
			GoPayTerminalID:  getEnv("GOPAY_TERMINAL_ID", "TERMINAL-001"),
		},
		Email: EmailConfig{
			Driver:       getEnv("EMAIL_DRIVER", "disabled"),
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@photobooth.com"),
			FromName:     getEnv("FROM_NAME", "Photo Booth"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", ""),
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

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := strings.ToLower(strings.TrimSpace(getEnv(key, "")))
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "1" || valueStr == "true" || valueStr == "yes"
}

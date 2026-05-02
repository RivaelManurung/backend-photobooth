package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	Password          string         `gorm:"not null" json:"-"`
	Name              string         `gorm:"not null" json:"name"`
	Phone             string         `json:"phone"`
	Avatar            string         `json:"avatar"`
	Role              string         `gorm:"default:'user'" json:"role"` // user, admin
	IsEmailVerified   bool           `gorm:"default:false" json:"is_email_verified"`
	EmailVerifiedAt   *time.Time     `json:"email_verified_at"`
	VerificationToken string         `json:"-"`
	ResetToken        string         `json:"-"`
	ResetTokenExpiry  *time.Time     `json:"-"`
	LastLoginAt       *time.Time     `json:"last_login_at"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	
	// Subscription
	SubscriptionPlan  string         `gorm:"default:'free'" json:"subscription_plan"` // free, basic, premium
	SubscriptionEnd   *time.Time     `json:"subscription_end"`
	
	// Relations
	Photos            []Photo        `gorm:"foreignKey:UserID" json:"photos,omitempty"`
	Orders            []Order        `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

// HashPassword hashes the user password
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword checks if the provided password is correct
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// BeforeCreate hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Only hash if password is not empty and not already hashed
	// Bcrypt hashes always start with $2a$, $2b$, or $2y$
	if u.Password != "" && !isBcryptHash(u.Password) {
		return u.HashPassword(u.Password)
	}
	return nil
}

// isBcryptHash checks if a string is already a bcrypt hash
func isBcryptHash(s string) bool {
	return len(s) == 60 && (s[:4] == "$2a$" || s[:4] == "$2b$" || s[:4] == "$2y$")
}

// IsSubscriptionActive checks if user has active subscription
func (u *User) IsSubscriptionActive() bool {
	if u.SubscriptionPlan == "free" {
		return true
	}
	if u.SubscriptionEnd == nil {
		return false
	}
	return u.SubscriptionEnd.After(time.Now())
}

// GetSubscriptionLimits returns limits based on subscription plan
func (u *User) GetSubscriptionLimits() map[string]int {
	limits := map[string]map[string]int{
		"free": {
			"photos_per_month":    10,
			"templates_access":    3,
			"storage_mb":          100,
			"watermark":           1, // has watermark
		},
		"basic": {
			"photos_per_month":    50,
			"templates_access":    10,
			"storage_mb":          500,
			"watermark":           0, // no watermark
		},
		"premium": {
			"photos_per_month":    -1, // unlimited
			"templates_access":    -1, // all templates
			"storage_mb":          5000,
			"watermark":           0,
		},
	}
	
	if plan, exists := limits[u.SubscriptionPlan]; exists {
		return plan
	}
	return limits["free"]
}

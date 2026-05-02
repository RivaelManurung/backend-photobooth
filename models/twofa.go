package models

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

// TwoFactorAuth represents 2FA settings for a user
type TwoFactorAuth struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	UserID      uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	// TOTP Settings
	Secret      string         `gorm:"not null" json:"-"` // Never expose in JSON
	IsEnabled   bool           `gorm:"default:false" json:"is_enabled"`
	VerifiedAt  *time.Time     `json:"verified_at"`
	
	// Backup Codes
	BackupCodes string         `gorm:"type:jsonb" json:"-"` // Encrypted backup codes
	
	// Recovery
	RecoveryEmail string       `json:"recovery_email"`
	
	// Metadata
	LastUsedAt  *time.Time     `json:"last_used_at"`
	FailedAttempts int         `gorm:"default:0" json:"-"`
	LockedUntil *time.Time     `json:"-"`
}

// TwoFactorLog represents 2FA verification attempts
type TwoFactorLog struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	Action      string    `json:"action"` // verify, enable, disable, recovery
	Status      string    `json:"status"` // success, failed
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// GenerateSecret generates a new TOTP secret
func (t *TwoFactorAuth) GenerateSecret() error {
	secret := make([]byte, 20)
	_, err := rand.Read(secret)
	if err != nil {
		return err
	}
	
	t.Secret = base32.StdEncoding.EncodeToString(secret)
	return nil
}

// GenerateQRCode generates QR code URL for TOTP setup
func (t *TwoFactorAuth) GenerateQRCode(issuer, accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Secret:      []byte(t.Secret),
	})
	if err != nil {
		return "", err
	}
	
	return key.URL(), nil
}

// VerifyCode verifies a TOTP code
func (t *TwoFactorAuth) VerifyCode(code string) bool {
	return totp.Validate(code, t.Secret)
}

// GenerateBackupCodes generates backup codes
func (t *TwoFactorAuth) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	
	for i := 0; i < count; i++ {
		code := make([]byte, 8)
		_, err := rand.Read(code)
		if err != nil {
			return nil, err
		}
		
		// Format as XXXX-XXXX
		codes[i] = fmt.Sprintf("%X-%X", code[:4], code[4:])
	}
	
	return codes, nil
}

// IsLocked checks if 2FA is temporarily locked due to failed attempts
func (t *TwoFactorAuth) IsLocked() bool {
	if t.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*t.LockedUntil)
}

// IncrementFailedAttempts increments failed attempts and locks if necessary
func (t *TwoFactorAuth) IncrementFailedAttempts(db *gorm.DB) error {
	t.FailedAttempts++
	
	// Lock after 5 failed attempts for 15 minutes
	if t.FailedAttempts >= 5 {
		lockUntil := time.Now().Add(15 * time.Minute)
		t.LockedUntil = &lockUntil
	}
	
	return db.Save(t).Error
}

// ResetFailedAttempts resets failed attempts counter
func (t *TwoFactorAuth) ResetFailedAttempts(db *gorm.DB) error {
	t.FailedAttempts = 0
	t.LockedUntil = nil
	return db.Save(t).Error
}

// Enable enables 2FA for user
func (t *TwoFactorAuth) Enable(db *gorm.DB) error {
	now := time.Now()
	t.IsEnabled = true
	t.VerifiedAt = &now
	return db.Save(t).Error
}

// Disable disables 2FA for user
func (t *TwoFactorAuth) Disable(db *gorm.DB) error {
	t.IsEnabled = false
	return db.Save(t).Error
}

// UpdateLastUsed updates last used timestamp
func (t *TwoFactorAuth) UpdateLastUsed(db *gorm.DB) error {
	now := time.Now()
	t.LastUsedAt = &now
	return db.Save(t).Error
}

// CreateTwoFactorLog creates a 2FA log entry
func CreateTwoFactorLog(db *gorm.DB, log *TwoFactorLog) error {
	return db.Create(log).Error
}

// GetUserTwoFactorAuth gets 2FA settings for user
func GetUserTwoFactorAuth(db *gorm.DB, userID uint) (*TwoFactorAuth, error) {
	var twoFA TwoFactorAuth
	err := db.Where("user_id = ?", userID).First(&twoFA).Error
	if err != nil {
		return nil, err
	}
	return &twoFA, nil
}

// CreateTwoFactorAuth creates 2FA settings for user
func CreateTwoFactorAuth(db *gorm.DB, twoFA *TwoFactorAuth) error {
	return db.Create(twoFA).Error
}

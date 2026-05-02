package models

import (
	"time"

	"gorm.io/gorm"
)

// Session represents a photo booth session
type Session struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Session Info
	SessionID   string         `gorm:"uniqueIndex;not null" json:"session_id"`
	UserID      *uint          `gorm:"index" json:"user_id"` // Optional, can be guest
	User        *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	// Event Details
	EventName   string         `json:"event_name"`
	EventType   string         `json:"event_type"` // birthday, wedding, corporate, etc
	Location    string         `json:"location"`
	
	// Session Configuration
	TemplateID  uint           `json:"template_id"`
	Template    Template       `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	LayoutCount int            `json:"layout_count"`
	
	// Status
	Status      string         `gorm:"default:'active'" json:"status"` // active, completed, expired
	ExpiresAt   time.Time      `json:"expires_at"`
	
	// Stats
	PhotoCount  int            `gorm:"default:0" json:"photo_count"`
	
	// Relations
	Photos      []Photo        `gorm:"foreignKey:SessionID;references:SessionID" json:"photos,omitempty"`
}

// IsExpired checks if session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CanAddPhotos checks if more photos can be added
func (s *Session) CanAddPhotos(maxPhotos int) bool {
	return !s.IsExpired() && s.Status == "active" && s.PhotoCount < maxPhotos
}

// IncrementPhotoCount increments photo count
func (s *Session) IncrementPhotoCount(db *gorm.DB) error {
	return db.Model(s).Update("photo_count", gorm.Expr("photo_count + ?", 1)).Error
}

package models

import (
	"time"

	"gorm.io/gorm"
)

// Analytics tracks usage statistics
type Analytics struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Date      time.Time `gorm:"index;not null" json:"date"`

	// User Analytics
	UserID *uint `gorm:"index" json:"user_id"`

	// Event Type
	EventType string `gorm:"index;not null" json:"event_type"` // photo_created, photo_downloaded, template_used, etc

	// Related Entities
	PhotoID    *uint `json:"photo_id"`
	TemplateID *uint `json:"template_id"`
	OrderID    *uint `json:"order_id"`

	// Metadata
	Metadata string `gorm:"type:jsonb" json:"metadata"`

	// Device Info
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
	Country   string `json:"country"`
	City      string `json:"city"`
}

// DailyStats aggregated daily statistics
type DailyStats struct {
	ID   uint      `gorm:"primarykey" json:"id"`
	Date time.Time `gorm:"uniqueIndex;not null" json:"date"`

	// User Stats
	NewUsers    int `gorm:"default:0" json:"new_users"`
	ActiveUsers int `gorm:"default:0" json:"active_users"`

	// Photo Stats
	PhotosCreated    int `gorm:"default:0" json:"photos_created"`
	PhotosDownloaded int `gorm:"default:0" json:"photos_downloaded"`

	// Template Stats
	TemplatesUsed int `gorm:"default:0" json:"templates_used"`

	// Revenue Stats
	Revenue     float64 `gorm:"default:0" json:"revenue"`
	OrdersCount int     `gorm:"default:0" json:"orders_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AnalyticsEvent stores raw event activity before aggregation.
type AnalyticsEvent struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	EventName   string    `gorm:"index;not null" json:"event_name"`
	ActorUserID *uint     `gorm:"index" json:"actor_user_id"`
	SessionID   *string   `gorm:"index" json:"session_id"`
	PhotoID     *uint     `gorm:"index" json:"photo_id"`
	TemplateID  *uint     `gorm:"index" json:"template_id"`
	OrderID     *uint     `gorm:"index" json:"order_id"`
	Metadata    string    `gorm:"type:jsonb" json:"metadata"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

// TrackEvent creates an analytics event
func TrackEvent(db *gorm.DB, eventType string, userID *uint, metadata map[string]interface{}) error {
	// Implementation would serialize metadata to JSON
	analytics := Analytics{
		Date:      time.Now(),
		EventType: eventType,
		UserID:    userID,
	}
	return db.Create(&analytics).Error
}

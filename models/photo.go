package models

import (
	"time"

	"gorm.io/gorm"
)

type Photo struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	// User & Template (UserID is nullable — anonymous strips have no user)
	UserID      *uint          `gorm:"index" json:"user_id"`
	User        *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TemplateID  uint           `gorm:"index" json:"template_id"`
	Template    Template       `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	IsAnonymous bool           `gorm:"default:false" json:"is_anonymous"` // true for strip-public uploads
	
	// File Information
	OriginalURL string         `json:"original_url"`
	ProcessedURL string        `json:"processed_url"`
	ThumbnailURL string        `json:"thumbnail_url"`
	FileName    string         `json:"file_name"`
	FileSize    int64          `json:"file_size"`
	MimeType    string         `json:"mime_type"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	
	// Processing
	Status      string         `gorm:"default:'processing'" json:"status"` // processing, completed, failed
	ProcessingError string     `json:"processing_error,omitempty"`
	
	// Customization Applied
	FilterApplied string       `json:"filter_applied"` // none, bw, sepia, vivid, etc
	CustomData    string       `gorm:"type:jsonb" json:"custom_data"` // JSON for custom settings
	
	// Metadata
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Tags        string         `json:"tags"`
	IsPublic    bool           `gorm:"default:false" json:"is_public"`
	IsFavorite  bool           `gorm:"default:false" json:"is_favorite"`
	ViewCount   int            `gorm:"default:0" json:"view_count"`
	DownloadCount int          `gorm:"default:0" json:"download_count"`
	
	// Event/Session
	EventName   string         `json:"event_name"`
	SessionID   string         `gorm:"index" json:"session_id"`
	
	// Storage
	StorageProvider string     `json:"storage_provider"` // local, supabase
	StoragePath     string     `json:"storage_path"`
	
	// Watermark
	HasWatermark bool          `gorm:"default:false" json:"has_watermark"`
}

// CustomPhotoData represents custom settings for a photo
type CustomPhotoData struct {
	LayoutCount     int                    `json:"layout_count"`
	SelectedTemplate string                `json:"selected_template"`
	Filter          string                 `json:"filter"`
	TextOverlays    []TextOverlay          `json:"text_overlays"`
	Stickers        []Sticker              `json:"stickers"`
	Adjustments     map[string]interface{} `json:"adjustments"`
}

type TextOverlay struct {
	Text     string  `json:"text"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	FontSize int     `json:"font_size"`
	Color    string  `json:"color"`
	Font     string  `json:"font"`
}

type Sticker struct {
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
}

// IncrementView increments view count
func (p *Photo) IncrementView(db *gorm.DB) error {
	return db.Model(p).Update("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementDownload increments download count
func (p *Photo) IncrementDownload(db *gorm.DB) error {
	return db.Model(p).Update("download_count", gorm.Expr("download_count + ?", 1)).Error
}

// IsOwnedBy checks if photo belongs to user
func (p *Photo) IsOwnedBy(userID uint) bool {
	return p.UserID == userID
}

// CanBeAccessedBy checks if user can access this photo
func (p *Photo) CanBeAccessedBy(userID uint) bool {
	return p.IsPublic || p.UserID == userID
}

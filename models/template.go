package models

import (
	"time"

	"gorm.io/gorm"
)

type Template struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Basic Info
	Name        string         `gorm:"not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Description string         `json:"description"`
	Category    string         `json:"category"` // classic, vintage, modern, party, etc
	
	// Visual Properties
	PreviewURL  string         `json:"preview_url"`
	ThumbnailURL string        `json:"thumbnail_url"`
	BackgroundColor string     `json:"background_color"`
	TextColor   string         `json:"text_color"`
	BorderStyle string         `json:"border_style"`
	
	// Layout Configuration
	LayoutType  string         `json:"layout_type"` // strip, grid, collage, single
	PhotoCount  int            `json:"photo_count"` // number of photos this template supports
	Orientation string         `json:"orientation"` // portrait, landscape, square
	
	// Template Assets
	BackgroundURL string       `json:"background_url"` // Main background from Canva
	OverlayURL  string         `json:"overlay_url"`  // PNG overlay with transparency
	MaskURL     string         `json:"mask_url"`     // Mask for photo placement
	FrameURL    string         `json:"frame_url"`    // Frame/border image
	
	// Dimensions
	Width       int            `json:"width"`  // Template width in pixels
	Height      int            `json:"height"` // Template height in pixels
	DPI         int            `json:"dpi" gorm:"default:300"` // Print quality
	
	// Photo Zones (JSON array of zones)
	PhotoZones  string         `gorm:"type:text" json:"photo_zones"` // [{x, y, width, height, rotation}]
	
	// Text Elements (JSON array)
	TextElements string        `gorm:"type:text" json:"text_elements"` // [{content, x, y, font, size, color}]
	
	// Customization Options
	AllowFilters     bool      `gorm:"default:true" json:"allow_filters"`
	AllowTextEdit    bool      `gorm:"default:true" json:"allow_text_edit"`
	AllowStickers    bool      `gorm:"default:true" json:"allow_stickers"`
	CustomizableAreas string   `gorm:"type:text" json:"customizable_areas"`
	
	// Access Control
	IsPremium   bool           `gorm:"default:false" json:"is_premium"`
	Price       int            `gorm:"default:0" json:"price"` // Price in Rupiah for premium templates
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	IsFeatured  bool           `gorm:"default:false" json:"is_featured"`
	RequiredPlan string        `json:"required_plan"` // free, basic, premium
	
	// Metadata
	Tags        string         `json:"tags"` // comma-separated
	UsageCount  int            `gorm:"default:0" json:"usage_count"`
	Rating      float64        `gorm:"default:0" json:"rating"`
	
	// Canva Integration
	CanvaDesignID string       `json:"canva_design_id"`
	CanvaExportURL string      `json:"canva_export_url"`
	
	// Relations
	Photos      []Photo        `gorm:"foreignKey:TemplateID" json:"photos,omitempty"`
}

// PhotoZone represents a zone where a photo can be placed
type PhotoZone struct {
	Index    int     `json:"index"`
	X        float64 `json:"x"`        // X position in pixels
	Y        float64 `json:"y"`        // Y position in pixels
	Width    float64 `json:"width"`    // Width in pixels
	Height   float64 `json:"height"`   // Height in pixels
	Rotation float64 `json:"rotation"` // Rotation in degrees
	ZIndex   int     `json:"z_index"`  // Layer order
	Border   Border  `json:"border"`   // Border styling
	Effects  Effects `json:"effects"`  // Visual effects
}

// Border represents border styling
type Border struct {
	Width int    `json:"width"` // Border width in pixels
	Color string `json:"color"` // Border color (hex)
	Style string `json:"style"` // solid, dashed, dotted
}

// Effects represents visual effects
type Effects struct {
	Shadow  bool    `json:"shadow"`  // Drop shadow
	Rounded int     `json:"rounded"` // Border radius in pixels
	Blur    int     `json:"blur"`    // Blur amount
	Opacity float64 `json:"opacity"` // Opacity 0-1
}

// TextElement represents text on template
type TextElement struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`  // Text content or {{variable}}
	X        float64 `json:"x"`        // X position in pixels
	Y        float64 `json:"y"`        // Y position in pixels
	Font     Font    `json:"font"`     // Font styling
	Align    string  `json:"align"`    // left, center, right
	MaxWidth float64 `json:"max_width"` // Max width for wrapping
}

// Font represents font styling
type Font struct {
	Family string `json:"family"` // Font family name
	Size   int    `json:"size"`   // Font size in pixels
	Weight string `json:"weight"` // normal, bold, etc
	Color  string `json:"color"`  // Text color (hex)
	Style  string `json:"style"`  // normal, italic
}

// CustomizableArea represents areas that can be customized
type CustomizableArea struct {
	Type     string  `json:"type"`     // text, sticker, background
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Default  string  `json:"default"`  // Default value
	Options  []string `json:"options"` // Available options
}

// IncrementUsage increments the usage count
func (t *Template) IncrementUsage(db *gorm.DB) error {
	return db.Model(t).Update("usage_count", gorm.Expr("usage_count + ?", 1)).Error
}

// IsAccessibleBy checks if user can access this template
func (t *Template) IsAccessibleBy(user *User) bool {
	if !t.IsActive {
		return false
	}
	
	if !t.IsPremium {
		return true
	}
	
	// Check subscription plan
	switch t.RequiredPlan {
	case "free":
		return true
	case "basic":
		return user.SubscriptionPlan == "basic" || user.SubscriptionPlan == "premium"
	case "premium":
		return user.SubscriptionPlan == "premium"
	default:
		return true
	}
}

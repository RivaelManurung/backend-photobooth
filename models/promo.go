package models

import (
	"time"

	"gorm.io/gorm"
)

type PromoCode struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Code Details
	Code        string         `gorm:"uniqueIndex;not null" json:"code"`
	Description string         `json:"description"`
	Type        string         `gorm:"not null" json:"type"` // percentage, fixed
	
	// Discount
	DiscountPercent float64    `json:"discount_percent"` // For percentage type
	DiscountAmount  float64    `json:"discount_amount"`  // For fixed type
	MaxDiscount     float64    `json:"max_discount"`     // Max discount for percentage
	MinPurchase     float64    `json:"min_purchase"`     // Minimum purchase amount
	
	// Usage Limits
	MaxUses         int        `json:"max_uses"`          // 0 = unlimited
	UsedCount       int        `gorm:"default:0" json:"used_count"`
	MaxUsesPerUser  int        `json:"max_uses_per_user"` // 0 = unlimited
	
	// Validity
	StartsAt        time.Time  `json:"starts_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	
	// Restrictions
	ApplicablePlans string     `json:"applicable_plans"` // Comma-separated: basic,premium
	FirstTimeOnly   bool       `gorm:"default:false" json:"first_time_only"`
	
	// Relations
	UsageHistory    []PromoUsage `gorm:"foreignKey:PromoCodeID" json:"usage_history,omitempty"`
}

type PromoUsage struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	
	PromoCodeID uint      `gorm:"not null;index" json:"promo_code_id"`
	PromoCode   PromoCode `gorm:"foreignKey:PromoCodeID" json:"promo_code,omitempty"`
	
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	OrderID     uint      `gorm:"not null;index" json:"order_id"`
	Order       Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	
	DiscountAmount float64 `json:"discount_amount"`
}

// IsValid checks if promo code is valid
func (p *PromoCode) IsValid() bool {
	now := time.Now()
	
	if !p.IsActive {
		return false
	}
	
	if now.Before(p.StartsAt) || now.After(p.ExpiresAt) {
		return false
	}
	
	if p.MaxUses > 0 && p.UsedCount >= p.MaxUses {
		return false
	}
	
	return true
}

// CanBeUsedBy checks if user can use this promo code
func (p *PromoCode) CanBeUsedBy(db *gorm.DB, userID uint) bool {
	if !p.IsValid() {
		return false
	}
	
	// Check if first time only
	if p.FirstTimeOnly {
		var orderCount int64
		db.Model(&Order{}).Where("user_id = ? AND status = ?", userID, "paid").Count(&orderCount)
		if orderCount > 0 {
			return false
		}
	}
	
	// Check max uses per user
	if p.MaxUsesPerUser > 0 {
		var usageCount int64
		db.Model(&PromoUsage{}).Where("promo_code_id = ? AND user_id = ?", p.ID, userID).Count(&usageCount)
		if int(usageCount) >= p.MaxUsesPerUser {
			return false
		}
	}
	
	return true
}

// CalculateDiscount calculates discount amount
func (p *PromoCode) CalculateDiscount(amount float64) float64 {
	if amount < p.MinPurchase {
		return 0
	}
	
	var discount float64
	
	if p.Type == "percentage" {
		discount = amount * (p.DiscountPercent / 100)
		if p.MaxDiscount > 0 && discount > p.MaxDiscount {
			discount = p.MaxDiscount
		}
	} else {
		discount = p.DiscountAmount
	}
	
	// Discount cannot exceed amount
	if discount > amount {
		discount = amount
	}
	
	return discount
}

// IncrementUsage increments usage count
func (p *PromoCode) IncrementUsage(db *gorm.DB) error {
	return db.Model(p).Update("used_count", gorm.Expr("used_count + ?", 1)).Error
}

// RecordUsage records promo code usage
func (p *PromoCode) RecordUsage(db *gorm.DB, userID, orderID uint, discountAmount float64) error {
	usage := PromoUsage{
		PromoCodeID:    p.ID,
		UserID:         userID,
		OrderID:        orderID,
		DiscountAmount: discountAmount,
	}
	
	if err := db.Create(&usage).Error; err != nil {
		return err
	}
	
	return p.IncrementUsage(db)
}

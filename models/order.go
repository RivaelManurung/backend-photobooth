package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	
	// User
	UserID          uint           `gorm:"not null;index" json:"user_id"`
	User            User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	// Order Details
	OrderNumber     string         `gorm:"uniqueIndex;not null" json:"order_number"`
	Type            string         `gorm:"not null" json:"type"` // subscription, credits, print
	Status          string         `gorm:"default:'pending'" json:"status"` // pending, paid, failed, refunded, cancelled
	
	// Pricing
	Amount          float64        `gorm:"not null" json:"amount"`
	Currency        string         `gorm:"default:'IDR'" json:"currency"`
	Tax             float64        `json:"tax"`
	Discount        float64        `json:"discount"`
	TotalAmount     float64        `gorm:"not null" json:"total_amount"`
	
	// Payment
	PaymentMethod   string         `json:"payment_method"` // stripe, midtrans, manual
	PaymentProvider string         `json:"payment_provider"`
	PaymentID       string         `gorm:"index" json:"payment_id"` // External payment ID
	PaymentURL      string         `json:"payment_url"`
	PaidAt          *time.Time     `json:"paid_at"`
	
	// Subscription Specific
	SubscriptionPlan string        `json:"subscription_plan"` // basic, premium
	BillingPeriod   string         `json:"billing_period"` // monthly, yearly
	StartDate       *time.Time     `json:"start_date"`
	EndDate         *time.Time     `json:"end_date"`
	
	// Credits Specific
	CreditsAmount   int            `json:"credits_amount"`
	
	// Print Specific
	PhotoID         *uint          `json:"photo_id"`
	Photo           *Photo         `gorm:"foreignKey:PhotoID" json:"photo,omitempty"`
	PrintSize       string         `json:"print_size"` // 4x6, 5x7, 8x10
	PrintQuantity   int            `json:"print_quantity"`
	ShippingAddress string         `gorm:"type:text" json:"shipping_address"`
	TrackingNumber  string         `json:"tracking_number"`
	
	// Metadata
	Notes           string         `gorm:"type:text" json:"notes"`
	Metadata        string         `gorm:"type:jsonb" json:"metadata"`
	
	// Refund
	RefundedAt      *time.Time     `json:"refunded_at"`
	RefundAmount    float64        `json:"refund_amount"`
	RefundReason    string         `json:"refund_reason"`
	
	// Relations
	Transactions    []Transaction  `gorm:"foreignKey:OrderID" json:"transactions,omitempty"`
}

type Transaction struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	
	OrderID         uint           `gorm:"not null;index" json:"order_id"`
	Order           Order          `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	
	Type            string         `json:"type"` // payment, refund
	Status          string         `json:"status"` // pending, success, failed
	Amount          float64        `json:"amount"`
	Currency        string         `json:"currency"`
	
	PaymentMethod   string         `json:"payment_method"`
	PaymentProvider string         `json:"payment_provider"`
	ExternalID      string         `gorm:"index" json:"external_id"`
	
	Response        string         `gorm:"type:text" json:"response"`
	ErrorMessage    string         `json:"error_message"`
}

// GenerateOrderNumber generates a unique order number
func GenerateOrderNumber() string {
	return "ORD-" + time.Now().Format("20060102-150405")
}

// MarkAsPaid marks the order as paid
func (o *Order) MarkAsPaid(db *gorm.DB) error {
	now := time.Now()
	o.Status = "paid"
	o.PaidAt = &now
	return db.Save(o).Error
}

// MarkAsFailed marks the order as failed
func (o *Order) MarkAsFailed(db *gorm.DB, reason string) error {
	o.Status = "failed"
	o.Notes = reason
	return db.Save(o).Error
}

// ProcessRefund processes a refund
func (o *Order) ProcessRefund(db *gorm.DB, amount float64, reason string) error {
	now := time.Now()
	o.Status = "refunded"
	o.RefundedAt = &now
	o.RefundAmount = amount
	o.RefundReason = reason
	return db.Save(o).Error
}

// IsSubscriptionOrder checks if this is a subscription order
func (o *Order) IsSubscriptionOrder() bool {
	return o.Type == "subscription"
}

// IsPaid checks if order is paid
func (o *Order) IsPaid() bool {
	return o.Status == "paid"
}

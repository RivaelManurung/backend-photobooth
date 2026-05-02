package models

import (
	"time"

	"gorm.io/gorm"
)

// QRISPayment represents a QRIS payment transaction
type QRISPayment struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Order Reference
	OrderID         uint           `gorm:"not null;index" json:"order_id"`
	Order           Order          `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	
	// QRIS Details
	QRISString      string         `gorm:"type:text;not null" json:"qris_string"` // QRIS code string
	QRISImageURL    string         `json:"qris_image_url"` // URL to QR code image
	
	// GoPay Specific
	GoPayTransactionID string      `gorm:"uniqueIndex" json:"gopay_transaction_id"`
	GoPayMerchantID    string      `json:"gopay_merchant_id"`
	GoPayTerminalID    string      `json:"gopay_terminal_id"`
	
	// Payment Details
	Amount          float64        `gorm:"not null" json:"amount"`
	Currency        string         `gorm:"default:'IDR'" json:"currency"`
	Status          string         `gorm:"default:'pending'" json:"status"` // pending, paid, expired, cancelled
	
	// Timestamps
	ExpiresAt       time.Time      `gorm:"not null" json:"expires_at"`
	PaidAt          *time.Time     `json:"paid_at"`
	
	// Customer Info
	CustomerName    string         `json:"customer_name"`
	CustomerPhone   string         `json:"customer_phone"`
	CustomerEmail   string         `json:"customer_email"`
	
	// Callback & Notification
	CallbackURL     string         `json:"callback_url"`
	NotificationURL string         `json:"notification_url"`
	
	// Response from GoPay
	RawResponse     string         `gorm:"type:text" json:"raw_response"`
	ErrorMessage    string         `json:"error_message"`
	
	// Metadata
	IPAddress       string         `json:"ip_address"`
	UserAgent       string         `json:"user_agent"`
	Metadata        string         `gorm:"type:jsonb" json:"metadata"`
}

// IsExpired checks if QRIS payment is expired
func (q *QRISPayment) IsExpired() bool {
	return time.Now().After(q.ExpiresAt)
}

// IsPaid checks if payment is completed
func (q *QRISPayment) IsPaid() bool {
	return q.Status == "paid"
}

// MarkAsPaid marks the QRIS payment as paid
func (q *QRISPayment) MarkAsPaid(db *gorm.DB, goPayTransactionID string) error {
	now := time.Now()
	q.Status = "paid"
	q.PaidAt = &now
	q.GoPayTransactionID = goPayTransactionID
	return db.Save(q).Error
}

// MarkAsExpired marks the QRIS payment as expired
func (q *QRISPayment) MarkAsExpired(db *gorm.DB) error {
	q.Status = "expired"
	return db.Save(q).Error
}

// MarkAsCancelled marks the QRIS payment as cancelled
func (q *QRISPayment) MarkAsCancelled(db *gorm.DB) error {
	q.Status = "cancelled"
	return db.Save(q).Error
}

// CreateQRISPayment creates a new QRIS payment record
func CreateQRISPayment(db *gorm.DB, payment *QRISPayment) error {
	return db.Create(payment).Error
}

// GetQRISPaymentByOrderID gets QRIS payment by order ID
func GetQRISPaymentByOrderID(db *gorm.DB, orderID uint) (*QRISPayment, error) {
	var payment QRISPayment
	err := db.Where("order_id = ?", orderID).
		Order("created_at DESC").
		First(&payment).Error
	return &payment, err
}

// GetQRISPaymentByGoPayID gets QRIS payment by GoPay transaction ID
func GetQRISPaymentByGoPayID(db *gorm.DB, goPayTransactionID string) (*QRISPayment, error) {
	var payment QRISPayment
	err := db.Where("gopay_transaction_id = ?", goPayTransactionID).First(&payment).Error
	return &payment, err
}

// GetPendingQRISPayments gets all pending QRIS payments
func GetPendingQRISPayments(db *gorm.DB) ([]QRISPayment, error) {
	var payments []QRISPayment
	err := db.Where("status = ?", "pending").
		Where("expires_at > ?", time.Now()).
		Find(&payments).Error
	return payments, err
}

// GetExpiredQRISPayments gets expired but not marked QRIS payments
func GetExpiredQRISPayments(db *gorm.DB) ([]QRISPayment, error) {
	var payments []QRISPayment
	err := db.Where("status = ?", "pending").
		Where("expires_at <= ?", time.Now()).
		Find(&payments).Error
	return payments, err
}

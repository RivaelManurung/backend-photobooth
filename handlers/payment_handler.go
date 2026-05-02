package handlers

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	config *config.Config
}

func NewPaymentHandler(cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{config: cfg}
}

// CreateSubscriptionOrder creates a subscription order
func (h *PaymentHandler) CreateSubscriptionOrder(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Plan          string `json:"plan" binding:"required"` // basic, premium
		BillingPeriod string `json:"billing_period" binding:"required"` // monthly, yearly
		PaymentMethod string `json:"payment_method" binding:"required"` // stripe, midtrans
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate pricing
	pricing := h.calculateSubscriptionPrice(req.Plan, req.BillingPeriod)

	// Create order
	order := models.Order{
		UserID:           user.ID,
		OrderNumber:      models.GenerateOrderNumber(),
		Type:             "subscription",
		Status:           "pending",
		Amount:           pricing["amount"],
		Currency:         "IDR",
		Tax:              pricing["tax"],
		TotalAmount:      pricing["total"],
		PaymentMethod:    req.PaymentMethod,
		SubscriptionPlan: req.Plan,
		BillingPeriod:    req.BillingPeriod,
	}

	// Calculate subscription dates
	startDate := time.Now()
	var endDate time.Time
	if req.BillingPeriod == "monthly" {
		endDate = startDate.AddDate(0, 1, 0)
	} else {
		endDate = startDate.AddDate(1, 0, 0)
	}
	order.StartDate = &startDate
	order.EndDate = &endDate

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Create payment URL based on provider
	paymentURL := h.createPaymentURL(&order)
	order.PaymentURL = paymentURL
	database.DB.Save(&order)

	c.JSON(http.StatusCreated, gin.H{
		"order":       order,
		"payment_url": paymentURL,
	})
}

// GetOrders returns user's orders
func (h *PaymentHandler) GetOrders(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var orders []models.Order
	if err := database.DB.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

// GetOrder returns a single order
func (h *PaymentHandler) GetOrder(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).
		Preload("Transactions").
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

// WebhookStripe handles Stripe webhooks
func (h *PaymentHandler) WebhookStripe(c *gin.Context) {
	// TODO: Implement Stripe webhook verification and processing
	
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process webhook event
	eventType := payload["type"].(string)

	switch eventType {
	case "payment_intent.succeeded":
		// Handle successful payment
		h.handleSuccessfulPayment(payload)
	case "payment_intent.payment_failed":
		// Handle failed payment
		h.handleFailedPayment(payload)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// WebhookMidtrans handles Midtrans webhooks
func (h *PaymentHandler) WebhookMidtrans(c *gin.Context) {
	// TODO: Implement Midtrans webhook verification and processing
	
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID := payload["order_id"].(string)
	transactionStatus := payload["transaction_status"].(string)

	var order models.Order
	if err := database.DB.Where("order_number = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	switch transactionStatus {
	case "capture", "settlement":
		h.processSuccessfulOrder(&order)
	case "deny", "cancel", "expire":
		order.MarkAsFailed(database.DB, "Payment "+transactionStatus)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// CancelOrder cancels an order
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel this order"})
		return
	}

	order.Status = "cancelled"
	database.DB.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully"})
}

// Helper functions

func (h *PaymentHandler) calculateSubscriptionPrice(plan, period string) map[string]float64 {
	prices := map[string]map[string]float64{
		"basic": {
			"monthly": 49000,
			"yearly":  490000,
		},
		"premium": {
			"monthly": 99000,
			"yearly":  990000,
		},
	}

	amount := prices[plan][period]
	tax := amount * 0.11 // 11% PPN
	total := amount + tax

	return map[string]float64{
		"amount": amount,
		"tax":    tax,
		"total":  total,
	}
}

func (h *PaymentHandler) createPaymentURL(order *models.Order) string {
	// In production, this would create actual payment URL with Stripe/Midtrans
	return fmt.Sprintf("https://payment.example.com/pay/%s", order.OrderNumber)
}

func (h *PaymentHandler) handleSuccessfulPayment(payload map[string]interface{}) {
	// Extract order info and mark as paid
	// Update user subscription
}

func (h *PaymentHandler) handleFailedPayment(payload map[string]interface{}) {
	// Mark order as failed
}

func (h *PaymentHandler) processSuccessfulOrder(order *models.Order) {
	// Mark order as paid
	order.MarkAsPaid(database.DB)

	// Update user subscription if it's a subscription order
	if order.IsSubscriptionOrder() {
		var user models.User
		if err := database.DB.First(&user, order.UserID).Error; err == nil {
			user.SubscriptionPlan = order.SubscriptionPlan
			user.SubscriptionEnd = order.EndDate
			database.DB.Save(&user)
		}
	}

	// TODO: Send confirmation email
}

// GetPricingPlans returns available pricing plans
func (h *PaymentHandler) GetPricingPlans(c *gin.Context) {
	plans := []gin.H{
		{
			"id":          "free",
			"name":        "Free",
			"description": "Perfect for trying out",
			"price":       0,
			"features": []string{
				"10 photos per month",
				"3 templates",
				"100MB storage",
				"Watermark on photos",
			},
		},
		{
			"id":          "basic",
			"name":        "Basic",
			"description": "For casual users",
			"price":       49000,
			"yearly_price": 490000,
			"features": []string{
				"50 photos per month",
				"10 templates",
				"500MB storage",
				"No watermark",
				"Email support",
			},
		},
		{
			"id":          "premium",
			"name":        "Premium",
			"description": "For power users",
			"price":       99000,
			"yearly_price": 990000,
			"features": []string{
				"Unlimited photos",
				"All templates",
				"5GB storage",
				"No watermark",
				"Priority support",
				"Custom templates",
				"API access",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

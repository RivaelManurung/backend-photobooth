package handlers

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type GoPayHandler struct {
	config      *config.Config
	qrisService *services.GoPayQRISService
	wsHub       *services.Hub
}

func NewGoPayHandler(cfg *config.Config, qrisService *services.GoPayQRISService, wsHub *services.Hub) *GoPayHandler {
	return &GoPayHandler{
		config:      cfg,
		qrisService: qrisService,
		wsHub:       wsHub,
	}
}

// CreateQRISPayment creates a QRIS payment for an order
func (h *GoPayHandler) CreateQRISPayment(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Plan          string `json:"plan" binding:"required"` // basic, premium
		BillingPeriod string `json:"billing_period" binding:"required"` // monthly, yearly
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate pricing
	pricing := calculateSubscriptionPrice(req.Plan, req.BillingPeriod)

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
		PaymentMethod:    "qris",
		PaymentProvider:  "gopay",
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

	// Save order
	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Create QRIS payment
	qrisPayment, err := h.qrisService.CreateQRIS(&order, user)
	if err != nil {
		order.MarkAsFailed(database.DB, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create QRIS: " + err.Error()})
		return
	}

	// Save QRIS payment
	qrisPayment.IPAddress = c.ClientIP()
	qrisPayment.UserAgent = c.Request.UserAgent()

	if err := models.CreateQRISPayment(database.DB, qrisPayment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save QRIS payment"})
		return
	}

	// Update order with payment info
	order.PaymentID = qrisPayment.GoPayTransactionID
	database.DB.Save(&order)

	c.JSON(http.StatusCreated, gin.H{
		"order":        order,
		"qris_payment": qrisPayment,
		"message":      "Scan QR code to complete payment",
	})
}

// GetQRISPayment gets QRIS payment details
func (h *GoPayHandler) GetQRISPayment(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID := c.Param("order_id")

	// Get order
	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Get QRIS payment
	var oid uint
	c.ShouldBindUri(&oid)
	qrisPayment, err := models.GetQRISPaymentByOrderID(database.DB, oid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QRIS payment not found"})
		return
	}

	// Check if expired
	if qrisPayment.IsExpired() && qrisPayment.Status == "pending" {
		qrisPayment.MarkAsExpired(database.DB)
	}

	c.JSON(http.StatusOK, gin.H{
		"order":        order,
		"qris_payment": qrisPayment,
	})
}

// CheckQRISStatus checks QRIS payment status
func (h *GoPayHandler) CheckQRISStatus(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID := c.Param("order_id")

	// Get order
	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Get QRIS payment
	var oid uint
	c.ShouldBindUri(&oid)
	qrisPayment, err := models.GetQRISPaymentByOrderID(database.DB, oid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QRIS payment not found"})
		return
	}

	// Check status from GoPay
	status, err := h.qrisService.CheckPaymentStatus(qrisPayment.GoPayTransactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check status"})
		return
	}

	// Update status if changed
	if status == "paid" && qrisPayment.Status != "paid" {
		qrisPayment.MarkAsPaid(database.DB, qrisPayment.GoPayTransactionID)
		order.MarkAsPaid(database.DB)

		// Update user subscription
		user.SubscriptionPlan = order.SubscriptionPlan
		user.SubscriptionEnd = order.EndDate
		database.DB.Save(user)

		// Send WebSocket notification
		if h.wsHub != nil {
			h.wsHub.BroadcastToUser(user.ID, "payment_success", map[string]interface{}{
				"order_id":     order.ID,
				"order_number": order.OrderNumber,
				"plan":         order.SubscriptionPlan,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id": order.ID,
		"status":   status,
		"paid":     status == "paid",
	})
}

// GoPayCallback handles callback from GoPay
func (h *GoPayHandler) GoPayCallback(c *gin.Context) {
	var callback services.GoPayCallbackData
	if err := c.ShouldBindJSON(&callback); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify signature
	if !h.qrisService.VerifyCallback(&callback) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Get QRIS payment
	qrisPayment, err := models.GetQRISPaymentByGoPayID(database.DB, callback.TransactionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	// Get order
	var order models.Order
	if err := database.DB.First(&order, qrisPayment.OrderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Process based on status
	switch callback.Status {
	case "success":
		// Mark as paid
		qrisPayment.MarkAsPaid(database.DB, callback.TransactionID)
		order.MarkAsPaid(database.DB)

		// Update user subscription
		var user models.User
		if err := database.DB.First(&user, order.UserID).Error; err == nil {
			user.SubscriptionPlan = order.SubscriptionPlan
			user.SubscriptionEnd = order.EndDate
			database.DB.Save(&user)

			// Send WebSocket notification
			if h.wsHub != nil {
				h.wsHub.BroadcastToUser(user.ID, "payment_success", map[string]interface{}{
					"order_id":     order.ID,
					"order_number": order.OrderNumber,
					"plan":         order.SubscriptionPlan,
				})
			}
		}

	case "failed":
		order.MarkAsFailed(database.DB, "Payment failed")

	case "expired":
		qrisPayment.MarkAsExpired(database.DB)
		order.MarkAsFailed(database.DB, "Payment expired")
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// CancelQRISPayment cancels a QRIS payment
func (h *GoPayHandler) CancelQRISPayment(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID := c.Param("order_id")

	// Get order
	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel this order"})
		return
	}

	// Get QRIS payment
	var oid uint
	c.ShouldBindUri(&oid)
	qrisPayment, err := models.GetQRISPaymentByOrderID(database.DB, oid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QRIS payment not found"})
		return
	}

	// Cancel in GoPay
	if err := h.qrisService.CancelQRIS(qrisPayment.GoPayTransactionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel payment"})
		return
	}

	// Update status
	qrisPayment.MarkAsCancelled(database.DB)
	order.Status = "cancelled"
	database.DB.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Payment cancelled successfully"})
}

// Helper function
func calculateSubscriptionPrice(plan, period string) map[string]float64 {
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

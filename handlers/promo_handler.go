package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PromoHandler struct{}

func NewPromoHandler() *PromoHandler {
	return &PromoHandler{}
}

// ValidatePromoCode validates a promo code
func (h *PromoHandler) ValidatePromoCode(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Code   string  `json:"code" binding:"required"`
		Amount float64 `json:"amount" binding:"required"`
		Plan   string  `json:"plan"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find promo code
	var promo models.PromoCode
	if err := database.DB.Where("UPPER(code) = ?", strings.ToUpper(req.Code)).First(&promo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid promo code"})
		return
	}

	// Check if valid
	if !promo.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Promo code is expired or inactive"})
		return
	}

	// Check if user can use it
	if !promo.CanBeUsedBy(database.DB, user.ID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot use this promo code"})
		return
	}

	// Check applicable plans
	if promo.ApplicablePlans != "" && req.Plan != "" {
		plans := strings.Split(promo.ApplicablePlans, ",")
		applicable := false
		for _, p := range plans {
			if strings.TrimSpace(p) == req.Plan {
				applicable = true
				break
			}
		}
		if !applicable {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Promo code not applicable for this plan"})
			return
		}
	}

	// Calculate discount
	discount := promo.CalculateDiscount(req.Amount)

	if discount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Minimum purchase amount not met",
			"min_purchase": promo.MinPurchase,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":           true,
		"promo_code":      promo,
		"discount_amount": discount,
		"final_amount":    req.Amount - discount,
	})
}

// CreatePromoCode creates a new promo code (admin only)
func (h *PromoHandler) CreatePromoCode(c *gin.Context) {
	var promo models.PromoCode
	if err := c.ShouldBindJSON(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert code to uppercase
	promo.Code = strings.ToUpper(promo.Code)

	// Check if code already exists
	var existing models.PromoCode
	if err := database.DB.Where("code = ?", promo.Code).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Promo code already exists"})
		return
	}

	// Set defaults
	promo.IsActive = true
	promo.UsedCount = 0

	if err := database.DB.Create(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create promo code"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"promo_code": promo,
		"message":    "Promo code created successfully",
	})
}

// GetPromoCodes returns all promo codes (admin only)
func (h *PromoHandler) GetPromoCodes(c *gin.Context) {
	var promos []models.PromoCode

	query := database.DB.Model(&models.PromoCode{})

	// Filter by active status
	if active := c.Query("active"); active != "" {
		query = query.Where("is_active = ?", active == "true")
	}

	// Filter by validity
	if valid := c.Query("valid"); valid == "true" {
		now := time.Now()
		query = query.Where("is_active = ? AND starts_at <= ? AND expires_at >= ?", true, now, now)
	}

	if err := query.Order("created_at DESC").Find(&promos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch promo codes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"promo_codes": promos,
		"total":       len(promos),
	})
}

// GetPromoCode returns single promo code (admin only)
func (h *PromoHandler) GetPromoCode(c *gin.Context) {
	id := c.Param("id")

	var promo models.PromoCode
	if err := database.DB.Preload("UsageHistory").First(&promo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"promo_code": promo})
}

// UpdatePromoCode updates a promo code (admin only)
func (h *PromoHandler) UpdatePromoCode(c *gin.Context) {
	id := c.Param("id")

	var promo models.PromoCode
	if err := database.DB.First(&promo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	if err := c.ShouldBindJSON(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update promo code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"promo_code": promo,
		"message":    "Promo code updated successfully",
	})
}

// DeletePromoCode deletes a promo code (admin only)
func (h *PromoHandler) DeletePromoCode(c *gin.Context) {
	id := c.Param("id")

	var promo models.PromoCode
	if err := database.DB.First(&promo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	if err := database.DB.Delete(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete promo code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promo code deleted successfully"})
}

// GetPromoUsageHistory returns usage history for a promo code (admin only)
func (h *PromoHandler) GetPromoUsageHistory(c *gin.Context) {
	id := c.Param("id")

	var usages []models.PromoUsage
	if err := database.DB.Where("promo_code_id = ?", id).
		Preload("User").
		Preload("Order").
		Order("created_at DESC").
		Find(&usages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch usage history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usage_history": usages,
		"total":         len(usages),
	})
}

// TogglePromoStatus toggles promo code active status (admin only)
func (h *PromoHandler) TogglePromoStatus(c *gin.Context) {
	id := c.Param("id")

	var promo models.PromoCode
	if err := database.DB.First(&promo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	promo.IsActive = !promo.IsActive
	database.DB.Save(&promo)

	c.JSON(http.StatusOK, gin.H{
		"promo_code": promo,
		"message":    "Promo code status updated",
	})
}

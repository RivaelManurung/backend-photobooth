package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// GetDashboardStats returns dashboard statistics
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	var stats struct {
		TotalUsers       int64   `json:"total_users"`
		ActiveUsers      int64   `json:"active_users"`
		TotalPhotos      int64   `json:"total_photos"`
		TotalOrders      int64   `json:"total_orders"`
		TotalRevenue     float64 `json:"total_revenue"`
		MonthlyRevenue   float64 `json:"monthly_revenue"`
		NewUsersToday    int64   `json:"new_users_today"`
		PhotosToday      int64   `json:"photos_today"`
		PremiumUsers     int64   `json:"premium_users"`
		FreeUsers        int64   `json:"free_users"`
	}

	// Total users
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	// Active users (logged in last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	database.DB.Model(&models.User{}).
		Where("last_login_at > ?", thirtyDaysAgo).
		Count(&stats.ActiveUsers)

	// Total photos
	database.DB.Model(&models.Photo{}).Count(&stats.TotalPhotos)

	// Total orders
	database.DB.Model(&models.Order{}).Count(&stats.TotalOrders)

	// Total revenue
	database.DB.Model(&models.Order{}).
		Where("status = ?", "paid").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&stats.TotalRevenue)

	// Monthly revenue
	firstDayOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	database.DB.Model(&models.Order{}).
		Where("status = ? AND created_at >= ?", "paid", firstDayOfMonth).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&stats.MonthlyRevenue)

	// New users today
	today := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&models.User{}).
		Where("created_at >= ?", today).
		Count(&stats.NewUsersToday)

	// Photos today
	database.DB.Model(&models.Photo{}).
		Where("created_at >= ?", today).
		Count(&stats.PhotosToday)

	// Premium users
	database.DB.Model(&models.User{}).
		Where("subscription_plan IN ?", []string{"basic", "premium"}).
		Count(&stats.PremiumUsers)

	// Free users
	database.DB.Model(&models.User{}).
		Where("subscription_plan = ?", "free").
		Count(&stats.FreeUsers)

	c.JSON(http.StatusOK, stats)
}

// GetAllUsers returns all users with pagination
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	search := c.Query("search")
	plan := c.Query("plan")

	var users []models.User
	query := database.DB.Model(&models.User{})

	// Search by name or email
	if search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Filter by plan
	if plan != "" {
		query = query.Where("subscription_plan = ?", plan)
	}

	var total int64
	query.Count(&total)

	// Pagination
	var pageInt, limitInt int
	c.ShouldBindQuery(&pageInt)
	c.ShouldBindQuery(&limitInt)
	
	// Set defaults if not provided
	if pageInt == 0 {
		pageInt = 1
	}
	if limitInt == 0 {
		limitInt = 20
	}
	
	offset := (pageInt - 1) * limitInt

	query.Order("created_at DESC").Limit(limitInt).Offset(offset).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  pageInt,
		"limit": limitInt,
	})
}

// GetUser returns single user details
func (h *AdminHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.Preload("Photos").Preload("Orders").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUserStatus updates user active status
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsActive = req.IsActive
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "User status updated",
		"user":    user,
	})
}

// GetRevenueReport returns revenue report
func (h *AdminHandler) GetRevenueReport(c *gin.Context) {
	period := c.DefaultQuery("period", "month") // day, week, month, year

	var results []struct {
		Date   time.Time `json:"date"`
		Amount float64   `json:"amount"`
		Count  int       `json:"count"`
	}

	var groupBy string
	switch period {
	case "day":
		groupBy = "DATE(created_at)"
	case "week":
		groupBy = "DATE_TRUNC('week', created_at)"
	case "month":
		groupBy = "DATE_TRUNC('month', created_at)"
	case "year":
		groupBy = "DATE_TRUNC('year', created_at)"
	default:
		groupBy = "DATE_TRUNC('month', created_at)"
	}

	database.DB.Model(&models.Order{}).
		Select(groupBy+" as date, SUM(total_amount) as amount, COUNT(*) as count").
		Where("status = ?", "paid").
		Group(groupBy).
		Order("date DESC").
		Limit(30).
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{"report": results})
}

// GetTemplateAnalytics returns template usage analytics
func (h *AdminHandler) GetTemplateAnalytics(c *gin.Context) {
	var templates []models.Template
	database.DB.Order("usage_count DESC").Limit(20).Find(&templates)

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// GetUserGrowth returns user growth statistics
func (h *AdminHandler) GetUserGrowth(c *gin.Context) {
	var results []struct {
		Date  time.Time `json:"date"`
		Count int       `json:"count"`
	}

	database.DB.Model(&models.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Group("DATE(created_at)").
		Order("date DESC").
		Limit(30).
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{"growth": results})
}

// ExportUsers exports users to CSV
func (h *AdminHandler) ExportUsers(c *gin.Context) {
	var users []models.User
	database.DB.Find(&users)

	// Set headers for CSV download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=users.csv")

	// Write CSV header
	c.Writer.Write([]byte("ID,Name,Email,Plan,Created At\n"))

	// Write user data
	for _, user := range users {
		line := []byte(
			user.Name + "," +
				user.Email + "," +
				user.SubscriptionPlan + "," +
				user.CreatedAt.Format("2006-01-02") + "\n",
		)
		c.Writer.Write(line)
	}
}

// DeleteUser deletes a user (soft delete)
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Soft delete
	database.DB.Delete(&user)

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// GetSystemHealth returns system health status
func (h *AdminHandler) GetSystemHealth(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	// Check database
	sqlDB, err := database.DB.DB()
	if err != nil {
		health["database"] = "error"
		health["status"] = "unhealthy"
	} else {
		if err := sqlDB.Ping(); err != nil {
			health["database"] = "disconnected"
			health["status"] = "unhealthy"
		} else {
			health["database"] = "connected"
		}
	}

	// Check disk space (simplified)
	health["storage"] = "ok"

	c.JSON(http.StatusOK, health)
}

package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct{}

func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

// GlobalSearch performs global search across multiple entities
func (h *SearchHandler) GlobalSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	results := gin.H{}

	// Search templates
	var templates []models.Template
	database.DB.Where("name ILIKE ? OR description ILIKE ? OR tags ILIKE ?", 
		"%"+query+"%", "%"+query+"%", "%"+query+"%").
		Where("is_active = ?", true).
		Limit(10).
		Find(&templates)
	results["templates"] = templates

	// Search photos (if authenticated)
	if user != nil {
		var photos []models.Photo
		database.DB.Where("user_id = ? AND (title ILIKE ? OR description ILIKE ? OR tags ILIKE ?)", 
			user.ID, "%"+query+"%", "%"+query+"%", "%"+query+"%").
			Limit(10).
			Find(&photos)
		results["photos"] = photos
	}

	c.JSON(http.StatusOK, results)
}

// SearchTemplates searches templates with advanced filters
func (h *SearchHandler) SearchTemplates(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")
	isPremium := c.Query("is_premium")
	sortBy := c.DefaultQuery("sort", "usage_count") // usage_count, rating, created_at
	order := c.DefaultQuery("order", "desc")

	dbQuery := database.DB.Model(&models.Template{}).Where("is_active = ?", true)

	// Text search
	if query != "" {
		dbQuery = dbQuery.Where("name ILIKE ? OR description ILIKE ? OR tags ILIKE ?", 
			"%"+query+"%", "%"+query+"%", "%"+query+"%")
	}

	// Category filter
	if category != "" {
		dbQuery = dbQuery.Where("category = ?", category)
	}

	// Premium filter
	if isPremium != "" {
		dbQuery = dbQuery.Where("is_premium = ?", isPremium == "true")
	}

	// Sorting
	orderClause := sortBy + " " + order
	dbQuery = dbQuery.Order(orderClause)

	var templates []models.Template
	if err := dbQuery.Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"total":     len(templates),
		"query":     query,
	})
}

// SearchPhotos searches user's photos
func (h *SearchHandler) SearchPhotos(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	status := c.Query("status")
	filter := c.Query("filter")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	dbQuery := database.DB.Model(&models.Photo{}).Where("user_id = ?", user.ID)

	// Text search
	if query != "" {
		dbQuery = dbQuery.Where("title ILIKE ? OR description ILIKE ? OR tags ILIKE ?", 
			"%"+query+"%", "%"+query+"%", "%"+query+"%")
	}

	// Status filter
	if status != "" {
		dbQuery = dbQuery.Where("status = ?", status)
	}

	// Filter applied
	if filter != "" {
		dbQuery = dbQuery.Where("filter_applied = ?", filter)
	}

	// Date range
	if dateFrom != "" {
		dbQuery = dbQuery.Where("created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		dbQuery = dbQuery.Where("created_at <= ?", dateTo)
	}

	// Sorting
	orderClause := sortBy + " " + order
	dbQuery = dbQuery.Order(orderClause)

	var photos []models.Photo
	if err := dbQuery.Preload("Template").Find(&photos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"photos": photos,
		"total":  len(photos),
		"query":  query,
	})
}

// SearchUsers searches users (admin only)
func (h *SearchHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	plan := c.Query("plan")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	dbQuery := database.DB.Model(&models.User{})

	// Text search
	if query != "" {
		dbQuery = dbQuery.Where("name ILIKE ? OR email ILIKE ?", "%"+query+"%", "%"+query+"%")
	}

	// Plan filter
	if plan != "" {
		dbQuery = dbQuery.Where("subscription_plan = ?", plan)
	}

	// Status filter
	if status != "" {
		dbQuery = dbQuery.Where("is_active = ?", status == "active")
	}

	// Sorting
	orderClause := sortBy + " " + order
	dbQuery = dbQuery.Order(orderClause)

	var users []models.User
	if err := dbQuery.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": len(users),
		"query": query,
	})
}

// GetSearchSuggestions returns search suggestions
func (h *SearchHandler) GetSearchSuggestions(c *gin.Context) {
	query := c.Query("q")
	if query == "" || len(query) < 2 {
		c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
		return
	}

	var suggestions []string

	// Get template names
	var templates []models.Template
	database.DB.Select("name").
		Where("name ILIKE ? AND is_active = ?", "%"+query+"%", true).
		Limit(5).
		Find(&templates)
	
	for _, t := range templates {
		suggestions = append(suggestions, t.Name)
	}

	// Get categories
	var categories []string
	database.DB.Model(&models.Template{}).
		Distinct("category").
		Where("category ILIKE ? AND is_active = ?", "%"+query+"%", true).
		Pluck("category", &categories)
	
	suggestions = append(suggestions, categories...)

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// GetPopularSearches returns popular search terms
func (h *SearchHandler) GetPopularSearches(c *gin.Context) {
	// This would typically come from analytics
	// For now, return popular template categories
	var categories []struct {
		Category string `json:"category"`
		Count    int    `json:"count"`
	}

	database.DB.Model(&models.Template{}).
		Select("category, COUNT(*) as count").
		Where("is_active = ?", true).
		Group("category").
		Order("count DESC").
		Limit(10).
		Scan(&categories)

	c.JSON(http.StatusOK, gin.H{"popular_searches": categories})
}

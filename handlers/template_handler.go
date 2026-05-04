package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TemplateHandler struct {
	storageService *services.StorageService
}

func NewTemplateHandler(storage *services.StorageService) *TemplateHandler {
	return &TemplateHandler{storageService: storage}
}

// GetTemplates returns all templates
func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	var templates []models.Template

	query := database.DB.Where("is_active = ?", true)

	// Filter by category
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	// Filter by premium status
	if isPremium := c.Query("is_premium"); isPremium != "" {
		query = query.Where("is_premium = ?", isPremium == "true")
	}

	// Filter by featured
	if isFeatured := c.Query("is_featured"); isFeatured != "" {
		query = query.Where("is_featured = ?", isFeatured == "true")
	}

	// Filter by photo count (layout)
	if photoCount := c.Query("photo_count"); photoCount != "" {
		query = query.Where("photo_count = ?", photoCount)
	}

	// Check user subscription for access control
	user, _ := middleware.GetCurrentUser(c)
	
	if err := query.Order("is_featured DESC, usage_count DESC").Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}

	// Filter templates based on user subscription
	accessibleTemplates := []models.Template{}
	for _, template := range templates {
		if user == nil && !template.IsPremium {
			accessibleTemplates = append(accessibleTemplates, template)
		} else if user != nil && template.IsAccessibleBy(user) {
			accessibleTemplates = append(accessibleTemplates, template)
		}
	}

	// Convert URLs to presigned URLs for all templates
	for i := range accessibleTemplates {
		accessibleTemplates[i].BackgroundURL = h.storageService.GetFileURL(accessibleTemplates[i].BackgroundURL)
		accessibleTemplates[i].ThumbnailURL = h.storageService.GetFileURL(accessibleTemplates[i].ThumbnailURL)
		accessibleTemplates[i].PreviewURL = h.storageService.GetFileURL(accessibleTemplates[i].PreviewURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": accessibleTemplates,
		"total":     len(accessibleTemplates),
	})
}

// GetTemplate returns a single template
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Check access
	user, _ := middleware.GetCurrentUser(c)
	if user == nil && template.IsPremium {
		c.JSON(http.StatusForbidden, gin.H{"error": "Premium template requires authentication"})
		return
	}

	if user != nil && !template.IsAccessibleBy(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Subscription upgrade required"})
		return
	}

	// Convert URLs to presigned URLs
	template.BackgroundURL = h.storageService.GetFileURL(template.BackgroundURL)
	template.ThumbnailURL = h.storageService.GetFileURL(template.ThumbnailURL)
	template.PreviewURL = h.storageService.GetFileURL(template.PreviewURL)

	c.JSON(http.StatusOK, gin.H{"template": template})
}

// CreateTemplate creates a new template (admin only)
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var template models.Template
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template.IsActive = true

	if err := database.DB.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"template": template})
}

// UpdateTemplate updates a template (admin only)
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"template": template})
}

// DeleteTemplate deletes a template (admin only)
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Soft delete
	if err := database.DB.Delete(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

// UploadTemplateAsset uploads template assets (preview, overlay, etc.)
func (h *TemplateHandler) UploadTemplateAsset(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	assetType := c.PostForm("type") // preview, overlay, mask, frame

	// Upload file
	url, err := h.storageService.UploadFile(file, "templates/"+assetType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get presigned URL for response
	presignedURL := h.storageService.GetFileURL(url)

	c.JSON(http.StatusOK, gin.H{
		"url":      presignedURL,
		"filename": file.Filename,
		"size":     file.Size,
	})
}

// GetTemplateCategories returns all template categories
func (h *TemplateHandler) GetTemplateCategories(c *gin.Context) {
	var categories []string
	database.DB.Model(&models.Template{}).
		Distinct("category").
		Where("is_active = ?", true).
		Pluck("category", &categories)

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// IncrementTemplateUsage increments template usage count
func (h *TemplateHandler) IncrementTemplateUsage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	if err := template.IncrementUsage(database.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usage updated"})
}

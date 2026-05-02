package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	
	"backendphotobooth/database"
	"backendphotobooth/models"
	"backendphotobooth/services"
)

type TemplateAdminHandler struct {
	storageService     *services.StorageService
	templateProcessor  *services.TemplateProcessor
}

func NewTemplateAdminHandler(storage *services.StorageService, processor *services.TemplateProcessor) *TemplateAdminHandler {
	return &TemplateAdminHandler{
		storageService:    storage,
		templateProcessor: processor,
	}
}

// CreateTemplate creates a new template with file upload
func (h *TemplateAdminHandler) CreateTemplate(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32 MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	// Get form values
	name := c.PostForm("name")
	description := c.PostForm("description")
	category := c.PostForm("category")
	layoutType := c.PostForm("layout_type")
	photoCountStr := c.PostForm("photo_count")
	widthStr := c.PostForm("width")
	heightStr := c.PostForm("height")
	photoZonesJSON := c.PostForm("photo_zones")
	textElementsJSON := c.PostForm("text_elements")
	isPremiumStr := c.PostForm("is_premium")
	priceStr := c.PostForm("price")

	// Validate required fields
	if name == "" || category == "" || layoutType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	// Parse numeric fields
	photoCount, _ := strconv.Atoi(photoCountStr)
	width, _ := strconv.Atoi(widthStr)
	height, _ := strconv.Atoi(heightStr)
	isPremium := isPremiumStr == "true"
	_, _ = strconv.Atoi(priceStr) // price not used yet but keep for future

	// Set defaults
	if width == 0 {
		width = 1200
	}
	if height == 0 {
		height = 1800
	}
	if photoCount == 0 {
		photoCount = 4
	}

	// Handle background image upload
	backgroundFile, err := c.FormFile("background")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Background image is required"})
		return
	}

	// Upload background to storage (Supabase or local)
	backgroundPath, err := h.storageService.UploadFile(backgroundFile, "templates/backgrounds")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload background: %v", err)})
		return
	}

	// Generate thumbnail (for local storage, for Supabase we'll use the same image)
	thumbnailPath := backgroundPath

	// Generate slug
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())

	// Create template
	template := models.Template{
		Name:          name,
		Slug:          slug,
		Description:   description,
		Category:      category,
		LayoutType:    layoutType,
		PhotoCount:    photoCount,
		Width:         width,
		Height:        height,
		DPI:           300,
		BackgroundURL: backgroundPath,
		ThumbnailURL:  thumbnailPath,
		PreviewURL:    backgroundPath,
		PhotoZones:    photoZonesJSON,
		TextElements:  textElementsJSON,
		IsPremium:     isPremium,
		IsActive:      true,
		IsFeatured:    false,
		UsageCount:    0,
	}

	// Save to database
	if err := database.DB.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	// Convert URLs to presigned URLs for response
	template.BackgroundURL = h.storageService.GetFileURL(template.BackgroundURL)
	template.ThumbnailURL = h.storageService.GetFileURL(template.ThumbnailURL)
	template.PreviewURL = h.storageService.GetFileURL(template.PreviewURL)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Template created successfully",
		"template": template,
	})
}

// UpdateTemplate updates an existing template
func (h *TemplateAdminHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	// Update fields if provided
	if name := c.PostForm("name"); name != "" {
		template.Name = name
		// Update slug
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		template.Slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())
	}
	if description := c.PostForm("description"); description != "" {
		template.Description = description
	}
	if category := c.PostForm("category"); category != "" {
		template.Category = category
	}
	if layoutType := c.PostForm("layout_type"); layoutType != "" {
		template.LayoutType = layoutType
	}
	if photoCountStr := c.PostForm("photo_count"); photoCountStr != "" {
		photoCount, _ := strconv.Atoi(photoCountStr)
		template.PhotoCount = photoCount
	}
	if widthStr := c.PostForm("width"); widthStr != "" {
		width, _ := strconv.Atoi(widthStr)
		template.Width = width
	}
	if heightStr := c.PostForm("height"); heightStr != "" {
		height, _ := strconv.Atoi(heightStr)
		template.Height = height
	}
	if photoZonesJSON := c.PostForm("photo_zones"); photoZonesJSON != "" {
		template.PhotoZones = photoZonesJSON
	}
	if textElementsJSON := c.PostForm("text_elements"); textElementsJSON != "" {
		template.TextElements = textElementsJSON
	}
	if isPremiumStr := c.PostForm("is_premium"); isPremiumStr != "" {
		template.IsPremium = isPremiumStr == "true"
	}
	if priceStr := c.PostForm("price"); priceStr != "" {
		price, _ := strconv.Atoi(priceStr)
		template.Price = price
	}
	if isActiveStr := c.PostForm("is_active"); isActiveStr != "" {
		template.IsActive = isActiveStr == "true"
	}
	if isFeaturedStr := c.PostForm("is_featured"); isFeaturedStr != "" {
		template.IsFeatured = isFeaturedStr == "true"
	}

	// Handle background image update if provided
	if backgroundFile, err := c.FormFile("background"); err == nil {
		// Delete old background if exists
		if template.BackgroundURL != "" {
			h.storageService.DeleteFile(template.BackgroundURL)
		}
		
		// Upload new background
		backgroundPath, err := h.storageService.UploadFile(backgroundFile, "templates/backgrounds")
		if err == nil {
			template.BackgroundURL = backgroundPath
			template.PreviewURL = backgroundPath
			template.ThumbnailURL = backgroundPath
		}
	}

	// Save updates
	if err := database.DB.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	// Convert URLs to presigned URLs for response
	template.BackgroundURL = h.storageService.GetFileURL(template.BackgroundURL)
	template.ThumbnailURL = h.storageService.GetFileURL(template.ThumbnailURL)
	template.PreviewURL = h.storageService.GetFileURL(template.PreviewURL)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Template updated successfully",
		"template": template,
	})
}

// DeleteTemplate deletes a template
func (h *TemplateAdminHandler) DeleteTemplate(c *gin.Context) {
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

// GetAllTemplates returns all templates for admin (including inactive)
func (h *TemplateAdminHandler) GetAllTemplates(c *gin.Context) {
	var templates []models.Template

	query := database.DB.Model(&models.Template{})

	// Filters
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if isPremium := c.Query("is_premium"); isPremium != "" {
		query = query.Where("is_premium = ?", isPremium == "true")
	}
	if isActive := c.Query("is_active"); isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}

	// Convert URLs to presigned URLs for all templates
	for i := range templates {
		templates[i].BackgroundURL = h.storageService.GetFileURL(templates[i].BackgroundURL)
		templates[i].ThumbnailURL = h.storageService.GetFileURL(templates[i].ThumbnailURL)
		templates[i].PreviewURL = h.storageService.GetFileURL(templates[i].PreviewURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"templates":   templates,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetTemplateAnalytics returns template usage analytics
func (h *TemplateAdminHandler) GetTemplateAnalytics(c *gin.Context) {
	var analytics []struct {
		TemplateID   uint   `json:"template_id"`
		TemplateName string `json:"template_name"`
		Category     string `json:"category"`
		UsageCount   int    `json:"usage_count"`
		PhotoCount   int    `json:"photo_count"`
	}

	if err := database.DB.Model(&models.Template{}).
		Select("id as template_id, name as template_name, category, usage_count, (SELECT COUNT(*) FROM photos WHERE template_id = templates.id) as photo_count").
		Order("usage_count DESC").
		Limit(10).
		Find(&analytics).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// ToggleTemplateStatus toggles template active status
func (h *TemplateAdminHandler) ToggleTemplateStatus(c *gin.Context) {
	id := c.Param("id")

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	template.IsActive = !template.IsActive

	if err := database.DB.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Status updated successfully",
		"is_active": template.IsActive,
	})
}

// ToggleTemplateFeatured toggles template featured status
func (h *TemplateAdminHandler) ToggleTemplateFeatured(c *gin.Context) {
	id := c.Param("id")

	var template models.Template
	if err := database.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	template.IsFeatured = !template.IsFeatured

	if err := database.DB.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update featured status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Featured status updated successfully",
		"is_featured": template.IsFeatured,
	})
}

// DuplicateTemplate creates a copy of existing template
func (h *TemplateAdminHandler) DuplicateTemplate(c *gin.Context) {
	id := c.Param("id")

	var original models.Template
	if err := database.DB.First(&original, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Create duplicate
	duplicate := original
	duplicate.ID = 0
	duplicate.Name = original.Name + " (Copy)"
	duplicate.Slug = fmt.Sprintf("%s-copy-%d", original.Slug, time.Now().Unix())
	duplicate.IsActive = false
	duplicate.IsFeatured = false
	duplicate.UsageCount = 0
	duplicate.CreatedAt = time.Now()
	duplicate.UpdatedAt = time.Now()

	if err := database.DB.Create(&duplicate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to duplicate template"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Template duplicated successfully",
		"template": duplicate,
	})
}

// GetTemplateCategories returns all unique categories
func (h *TemplateAdminHandler) GetTemplateCategories(c *gin.Context) {
	var categories []string
	
	if err := database.DB.Model(&models.Template{}).
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type PhotoHandler struct {
	storageService *services.StorageService
	imageProcessor *services.ImageProcessor
}

func NewPhotoHandler(storage *services.StorageService, processor *services.ImageProcessor) *PhotoHandler {
	return &PhotoHandler{
		storageService: storage,
		imageProcessor: processor,
	}
}

// UploadPhoto handles photo upload
func (h *PhotoHandler) UploadPhoto(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get form data
	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	templateID := c.PostForm("template_id")
	filter := c.PostForm("filter")
	sessionID := c.PostForm("session_id")
	customDataStr := c.PostForm("custom_data")

	// Upload file
	url, err := h.storageService.UploadFile(file, "photos")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create photo record
	photo := models.Photo{
		UserID:          user.ID,
		OriginalURL:     url,
		FileName:        file.Filename,
		FileSize:        file.Size,
		MimeType:        file.Header.Get("Content-Type"),
		Status:          "processing",
		FilterApplied:   filter,
		SessionID:       sessionID,
		StorageProvider: "local",
		StoragePath:     url,
		HasWatermark:    user.SubscriptionPlan == "free",
	}

	// Parse template ID
	if templateID != "" {
		tid, _ := strconv.ParseUint(templateID, 10, 32)
		photo.TemplateID = uint(tid)
	}

	// Parse custom data
	if customDataStr != "" {
		photo.CustomData = customDataStr
	}

	if err := database.DB.Create(&photo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create photo record"})
		return
	}

	// Process photo asynchronously
	go func() {
		var template models.Template
		if photo.TemplateID > 0 {
			database.DB.First(&template, photo.TemplateID)
		}

		if err := h.imageProcessor.ProcessPhoto(&photo, &template, filter); err != nil {
			photo.Status = "failed"
			photo.ProcessingError = err.Error()
		} else {
			photo.Status = "completed"
		}

		database.DB.Save(&photo)
	}()

	// Convert URLs to presigned URLs for response
	photo.OriginalURL = h.storageService.GetFileURL(photo.OriginalURL)
	photo.ProcessedURL = h.storageService.GetFileURL(photo.ProcessedURL)
	photo.ThumbnailURL = h.storageService.GetFileURL(photo.ThumbnailURL)

	c.JSON(http.StatusCreated, gin.H{
		"photo":   photo,
		"message": "Photo uploaded and processing",
	})
}

// GetPhotos returns user's photos
func (h *PhotoHandler) GetPhotos(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var photos []models.Photo
	query := database.DB.Where("user_id = ?", user.ID).Preload("Template")

	// Filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if sessionID := c.Query("session_id"); sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	if isFavorite := c.Query("is_favorite"); isFavorite == "true" {
		query = query.Where("is_favorite = ?", true)
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var total int64
	query.Model(&models.Photo{}).Count(&total)

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&photos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch photos"})
		return
	}

	// Convert URLs to presigned URLs for all photos
	for i := range photos {
		photos[i].OriginalURL = h.storageService.GetFileURL(photos[i].OriginalURL)
		photos[i].ProcessedURL = h.storageService.GetFileURL(photos[i].ProcessedURL)
		photos[i].ThumbnailURL = h.storageService.GetFileURL(photos[i].ThumbnailURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"photos": photos,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetPhoto returns a single photo
func (h *PhotoHandler) GetPhoto(c *gin.Context) {
	id := c.Param("id")

	var photo models.Photo
	if err := database.DB.Preload("Template").Preload("User").First(&photo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// Check access
	user, _ := middleware.GetCurrentUser(c)
	if user == nil && !photo.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if user != nil && !photo.CanBeAccessedBy(user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Increment view count
	photo.IncrementView(database.DB)

	// Convert URLs to presigned URLs
	photo.OriginalURL = h.storageService.GetFileURL(photo.OriginalURL)
	photo.ProcessedURL = h.storageService.GetFileURL(photo.ProcessedURL)
	photo.ThumbnailURL = h.storageService.GetFileURL(photo.ThumbnailURL)

	c.JSON(http.StatusOK, gin.H{"photo": photo})
}

// DeletePhoto deletes a photo
func (h *PhotoHandler) DeletePhoto(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	var photo models.Photo
	if err := database.DB.First(&photo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// Check ownership
	if !photo.IsOwnedBy(user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Delete files
	h.storageService.DeleteFile(photo.OriginalURL)
	h.storageService.DeleteFile(photo.ProcessedURL)
	h.storageService.DeleteFile(photo.ThumbnailURL)

	// Delete record
	if err := database.DB.Delete(&photo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete photo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Photo deleted successfully"})
}

// DownloadPhoto handles photo download
func (h *PhotoHandler) DownloadPhoto(c *gin.Context) {
	id := c.Param("id")

	var photo models.Photo
	if err := database.DB.First(&photo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// Check access
	user, _ := middleware.GetCurrentUser(c)
	if user != nil && !photo.CanBeAccessedBy(user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Increment download count
	photo.IncrementDownload(database.DB)

	// Serve file
	c.File("." + photo.ProcessedURL)
}

// ToggleFavorite toggles photo favorite status
func (h *PhotoHandler) ToggleFavorite(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	var photo models.Photo
	if err := database.DB.First(&photo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	if !photo.IsOwnedBy(user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	photo.IsFavorite = !photo.IsFavorite
	database.DB.Save(&photo)

	// Convert URLs to presigned URLs
	photo.OriginalURL = h.storageService.GetFileURL(photo.OriginalURL)
	photo.ProcessedURL = h.storageService.GetFileURL(photo.ProcessedURL)
	photo.ThumbnailURL = h.storageService.GetFileURL(photo.ThumbnailURL)

	c.JSON(http.StatusOK, gin.H{
		"photo":       photo,
		"is_favorite": photo.IsFavorite,
	})
}

// CreatePhotoStrip creates a photo strip from multiple photos
func (h *PhotoHandler) CreatePhotoStrip(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		PhotoIDs   []uint `json:"photo_ids" binding:"required"`
		TemplateID uint   `json:"template_id" binding:"required"`
		Filter     string `json:"filter"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get photos
	var photos []models.Photo
	if err := database.DB.Where("id IN ? AND user_id = ?", req.PhotoIDs, user.ID).Find(&photos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch photos"})
		return
	}

	if len(photos) != len(req.PhotoIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some photos not found or access denied"})
		return
	}

	// Get template
	var template models.Template
	if err := database.DB.First(&template, req.TemplateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Create photo paths
	photoPaths := make([]string, len(photos))
	for i, photo := range photos {
		photoPaths[i] = "." + photo.OriginalURL
	}

	// Create strip
	stripPath, err := h.imageProcessor.CreatePhotoStrip(photoPaths, &template)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create photo strip"})
		return
	}

	// Create photo record for strip
	stripPhoto := models.Photo{
		UserID:          user.ID,
		TemplateID:      req.TemplateID,
		OriginalURL:     stripPath,
		ProcessedURL:    stripPath,
		FileName:        "photo-strip.png",
		Status:          "completed",
		FilterApplied:   req.Filter,
		StorageProvider: "local",
		StoragePath:     stripPath,
		HasWatermark:    user.SubscriptionPlan == "free",
	}

	// Store photo IDs in custom data
	customData := map[string]interface{}{
		"photo_ids": req.PhotoIDs,
		"type":      "strip",
	}
	customDataJSON, _ := json.Marshal(customData)
	stripPhoto.CustomData = string(customDataJSON)

	if err := database.DB.Create(&stripPhoto).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save strip"})
		return
	}

	// Convert URLs to presigned URLs
	stripPhoto.OriginalURL = h.storageService.GetFileURL(stripPhoto.OriginalURL)
	stripPhoto.ProcessedURL = h.storageService.GetFileURL(stripPhoto.ProcessedURL)
	stripPhoto.ThumbnailURL = h.storageService.GetFileURL(stripPhoto.ThumbnailURL)

	c.JSON(http.StatusCreated, gin.H{
		"photo":   stripPhoto,
		"message": "Photo strip created successfully",
	})
}

// UploadPublicStrip accepts a base64-encoded PNG strip from the user (no auth required)
// POST /api/v1/photos/strip-public
func (h *PhotoHandler) UploadPublicStrip(c *gin.Context) {
	var req struct {
		ImageBase64 string `json:"image_base64" binding:"required"` // data:image/png;base64,xxxx
		TemplateID  uint   `json:"template_id"`
		Filter      string `json:"filter"`
		SessionID   string `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_base64 is required"})
		return
	}

	// Strip the data URI prefix
	raw := req.ImageBase64
	contentType := "image/png"
	if idx := strings.Index(raw, ";base64,"); idx != -1 {
		prefix := raw[:idx]
		if strings.Contains(prefix, "image/jpeg") {
			contentType = "image/jpeg"
		}
		raw = raw[idx+len(";base64,"):]
	}

	imgBytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 image"})
		return
	}

	storagePath, err := h.storageService.UploadFromBytes(imgBytes, "strips", contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload strip: " + err.Error()})
		return
	}

	// Build public URL immediately
	publicURL := h.storageService.GetFileURL(storagePath)

	c.JSON(http.StatusCreated, gin.H{
		"url":     publicURL,
		"path":    storagePath,
		"message": "Strip uploaded successfully",
	})
}


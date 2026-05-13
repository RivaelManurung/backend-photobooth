package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"backendphotobooth/utils"
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
	queueService   *services.QueueService
	sessionService *services.SessionService
}

func NewPhotoHandler(storage *services.StorageService, processor *services.ImageProcessor, sessionService *services.SessionService) *PhotoHandler {
	return &PhotoHandler{
		storageService: storage,
		imageProcessor: processor,
		sessionService: sessionService,
	}
}

func (h *PhotoHandler) SetQueueService(queue *services.QueueService) {
	h.queueService = queue
}

// UploadPhoto handles photo upload
func (h *PhotoHandler) UploadPhoto(c *gin.Context) {
	user, _ := middleware.GetCurrentUser(c)

	// Get form data
	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	sessionID := c.PostForm("session_id")
	sessionToken := c.PostForm("session_token")
	templateID := c.PostForm("template_id")
	filter := c.PostForm("filter")
	customDataStr := c.PostForm("custom_data")

	// 1. Fetch and Validate Session
	var session models.Session
	if sessionID != "" {
		if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}

		// Ownership/Token Validation
		if session.UserID != nil {
			if user == nil || *session.UserID != user.ID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to session"})
				return
			}
		} else if session.Token != sessionToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session token"})
			return
		}

		// State Validation
		if err := h.sessionService.ValidateStateAction(&session, "upload"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: session_id or login required"})
		return
	}

	// 2. Validate Image Content
	config := utils.ValidationConfig{
		MaxFileSize: 10 * 1024 * 1024, // 10MB default
		AllowedMimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
	}
	if err := utils.ValidateImageUpload(file, config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image: " + err.Error()})
		return
	}

	// 3. Upload file
	url, err := h.storageService.UploadFile(file, "photos")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: " + err.Error()})
		return
	}

	// Create photo record
	photo := models.Photo{
		OriginalURL:       url,
		FileName:          file.Filename,
		FileSize:          file.Size,
		MimeType:          file.Header.Get("Content-Type"),
		Status:            "processing",
		ProcessingStatus:  "pending",
		OriginalObjectKey: strings.TrimPrefix(url, "/uploads/"),
		FilterApplied:     filter,
		SessionID:         sessionID,
		StorageProvider:   "local",
		StoragePath:       url,
	}

	if user != nil {
		photo.UserID = &user.ID
		photo.HasWatermark = user.SubscriptionPlan == "free"
	} else {
		photo.HasWatermark = true // Anonymous always has watermark
		photo.IsAnonymous = true
	}

	// Parse template ID
	if templateID != "" {
		tid, _ := strconv.ParseUint(templateID, 10, 32)
		photo.TemplateID = uint(tid)
	} else if session.TemplateID > 0 {
		photo.TemplateID = session.TemplateID
	}

	// Parse custom data
	if customDataStr != "" {
		photo.CustomData = customDataStr
	}

	if err := database.DB.Create(&photo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create photo record"})
		return
	}

	// Update session photo count
	if session.ID > 0 {
		session.IncrementPhotoCount(database.DB)
	}

	if h.queueService != nil {
		if err := h.queueService.EnqueuePhotoProcess(c.Request.Context(), photo.ID); err != nil {
			photo.ProcessingStatus = "failed"
			photo.ProcessingError = "failed to enqueue processing job"
			photo.Status = "failed"
			database.DB.Save(&photo)
		}
	} else {
		go func() {
			var template models.Template
			if photo.TemplateID > 0 {
				database.DB.First(&template, photo.TemplateID)
			}

			if err := h.imageProcessor.ProcessPhoto(&photo, &template, filter); err != nil {
				photo.Status = "failed"
				photo.ProcessingStatus = "failed"
				photo.ProcessingError = err.Error()
			} else {
				photo.Status = "completed"
				photo.ProcessingStatus = "completed"
			}

			database.DB.Save(&photo)
		}()
	}

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
	if user == nil && !photo.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
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
		UserID:          &user.ID,
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
func (h *PhotoHandler) UploadPublicStrip(c *gin.Context) {
	var req struct {
		ImageBase64  string `json:"image_base64" binding:"required"`
		TemplateID   uint   `json:"template_id"`
		Filter       string `json:"filter"`
		SessionID    string `json:"session_id"`
		SessionToken string `json:"session_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_base64 is required"})
		return
	}

	// 1. Session Validation if SessionID is provided
	var session models.Session
	if req.SessionID != "" {
		if err := database.DB.Where("session_id = ?", req.SessionID).First(&session).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}

		if session.UserID != nil {
			user, _ := middleware.GetCurrentUser(c)
			if user == nil || *session.UserID != user.ID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to session"})
				return
			}
		} else if session.Token != req.SessionToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session token"})
			return
		}
	}

	// 2. Decode and Upload
	raw := req.ImageBase64
	contentType := "image/png"
	if idx := strings.Index(raw, ";base64,"); idx != -1 {
		if strings.Contains(raw[:idx], "image/jpeg") {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: " + err.Error()})
		return
	}

	publicURL := h.storageService.GetPublicURL(storagePath)

	photo := models.Photo{
		OriginalURL:     publicURL,
		ProcessedURL:    publicURL,
		StoragePath:     storagePath,
		StorageProvider: "supabase",
		MimeType:        contentType,
		FileSize:        int64(len(imgBytes)),
		FilterApplied:   req.Filter,
		SessionID:       req.SessionID,
		TemplateID:      req.TemplateID,
		IsAnonymous:     true,
		IsPublic:        true,
		Status:          "completed",
		Title:           "Photobooth Strip",
	}

	if err := database.DB.Create(&photo).Error; err != nil {
		c.JSON(http.StatusCreated, gin.H{
			"url":     publicURL,
			"path":    storagePath,
			"message": "Strip uploaded (DB record skipped: " + err.Error() + ")",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      photo.ID,
		"url":     publicURL,
		"path":    storagePath,
		"message": "Strip uploaded successfully",
	})
}

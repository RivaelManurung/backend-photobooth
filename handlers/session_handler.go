package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"backendphotobooth/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SessionHandler struct {
	sessionService *services.SessionService
}

func NewSessionHandler(sessionService *services.SessionService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// CreateSession creates a new photo booth session
func (h *SessionHandler) CreateSession(c *gin.Context) {
	user, _ := middleware.GetCurrentUser(c)

	var req struct {
		EventName   string `json:"event_name"`
		EventType   string `json:"event_type"`
		Location    string `json:"location"`
		TemplateID  uint   `json:"template_id" binding:"required"`
		LayoutCount int    `json:"layout_count" binding:"required"`
		Duration    int    `json:"duration"` // Duration in hours
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default duration if not provided
	if req.Duration == 0 {
		req.Duration = 24 // 24 hours default
	}

	// Create session with random token
	token, _ := utils.GenerateRandomToken(32)
	session := models.Session{
		SessionID:   uuid.New().String(),
		Token:       token,
		EventName:   req.EventName,
		EventType:   req.EventType,
		Location:    req.Location,
		TemplateID:  req.TemplateID,
		LayoutCount: req.LayoutCount,
		Status:      models.StatusDraft,
		ExpiresAt:   time.Now().Add(time.Hour * time.Duration(req.Duration)),
	}

	if user != nil {
		session.UserID = &user.ID
	}

	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Transition to CAPTURING immediately if ready
	session.Status = models.StatusCapturing
	database.DB.Save(&session)

	// Load template
	database.DB.Preload("Template").First(&session, session.ID)

	c.JSON(http.StatusCreated, gin.H{
		"session": session,
		"message": "Session created successfully",
	})
}

// GetSession returns session details
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	token := c.Query("token")

	var session models.Session
	if err := database.DB.Where("session_id = ?", sessionID).
		Preload("Template").
		Preload("Photos").
		First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Enforce Ownership/Token
	user, _ := middleware.GetCurrentUser(c)
	if session.UserID != nil {
		if user == nil || *session.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	} else if session.Token != token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session token"})
		return
	}

	// Check if expired
	if session.IsExpired() && session.Status != models.StatusExpired {
		session.Status = models.StatusExpired
		database.DB.Save(&session)
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// GetUserSessions returns all sessions for authenticated user
func (h *SessionHandler) GetUserSessions(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sessions []models.Session
	if err := database.DB.Where("user_id = ?", user.ID).
		Preload("Template").
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// UpdateSession updates session details
func (h *SessionHandler) UpdateSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	var session models.Session
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Ownership check
	user, _ := middleware.GetCurrentUser(c)
	if session.UserID != nil && (user == nil || *session.UserID != user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		EventName string `json:"event_name"`
		Location  string `json:"location"`
		Status    string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle State Transition
	if req.Status != "" && req.Status != session.Status {
		if !h.sessionService.CanTransition(session.Status, req.Status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid status transition from %s to %s", session.Status, req.Status)})
			return
		}
		session.Status = req.Status
	}

	// Update other fields
	if req.EventName != "" {
		session.EventName = req.EventName
	}
	if req.Location != "" {
		session.Location = req.Location
	}

	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"message": "Session updated successfully",
	})
}

// EndSession ends an active session
func (h *SessionHandler) EndSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	var session models.Session
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	if !h.sessionService.CanTransition(session.Status, models.StatusCompleted) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot complete session from current state"})
		return
	}

	session.Status = models.StatusCompleted
	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"message": "Session ended successfully",
	})
}

// GetSessionPhotos returns all photos in a session
func (h *SessionHandler) GetSessionPhotos(c *gin.Context) {
	sessionID := c.Param("session_id")
	token := c.Query("token")

	var session models.Session
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Token/Owner validation
	user, _ := middleware.GetCurrentUser(c)
	if session.UserID != nil {
		if user == nil || *session.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	} else if session.Token != token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	var photos []models.Photo
	if err := database.DB.Where("session_id = ?", sessionID).
		Preload("Template").
		Order("created_at DESC").
		Find(&photos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch photos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"photos": photos,
		"total":  len(photos),
	})
}

// ExtendSession extends session expiration time
func (h *SessionHandler) ExtendSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		Hours int `json:"hours" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var session models.Session
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Extend expiration
	session.ExpiresAt = session.ExpiresAt.Add(time.Hour * time.Duration(req.Hours))
	if session.Status == models.StatusExpired {
		session.Status = models.StatusCapturing
	}

	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"message": "Session extended successfully",
	})
}

// DeleteSession deletes a session
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	var session models.Session
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Check ownership
	user, _ := middleware.GetCurrentUser(c)
	if session.UserID != nil && (user == nil || *session.UserID != user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	database.DB.Delete(&session)

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
}

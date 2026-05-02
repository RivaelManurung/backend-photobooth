package handlers

import (
	"backendphotobooth/middleware"
	"backendphotobooth/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check origin properly
		return true
	},
}

type WebSocketHandler struct {
	hub *services.Hub
}

func NewWebSocketHandler(hub *services.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// HandleWebSocket handles websocket connections
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Get user from context (if authenticated)
	user, err := middleware.GetCurrentUser(c)
	var userID uint
	if err == nil && user != nil {
		userID = user.ID
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := &services.Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      h.hub,
		IsActive: true,
	}

	// Register client
	h.hub.Register <- client

	// Send welcome message
	client.SendMessage("connected", map[string]interface{}{
		"client_id": client.ID,
		"user_id":   userID,
		"message":   "Connected to Photo Booth WebSocket",
	})

	// Start pumps
	go client.WritePump()
	go client.ReadPump()
}

// GetConnectedClients returns connected clients count
func (h *WebSocketHandler) GetConnectedClients(c *gin.Context) {
	count := h.hub.GetConnectedClients()
	c.JSON(http.StatusOK, gin.H{
		"connected_clients": count,
	})
}

// BroadcastMessage broadcasts message to all clients (admin only)
func (h *WebSocketHandler) BroadcastMessage(c *gin.Context) {
	var req struct {
		Event string                 `json:"event" binding:"required"`
		Data  map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.BroadcastToAll(req.Event, req.Data)

	c.JSON(http.StatusOK, gin.H{
		"message": "Broadcast sent",
		"event":   req.Event,
	})
}

// SendMessageToUser sends message to specific user (admin only)
func (h *WebSocketHandler) SendMessageToUser(c *gin.Context) {
	userID := c.Param("user_id")

	var req struct {
		Event string                 `json:"event" binding:"required"`
		Data  map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var uid uint
	c.ShouldBindUri(&uid)

	h.hub.BroadcastToUser(uid, req.Event, req.Data)

	c.JSON(http.StatusOK, gin.H{
		"message": "Message sent to user",
		"user_id": userID,
		"event":   req.Event,
	})
}

package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a websocket message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Client represents a websocket client
type Client struct {
	ID       string
	UserID   uint
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	IsActive bool
	mu       sync.Mutex
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s (User: %d)", client.ID, client.UserID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				log.Printf("Client unregistered: %s", client.ID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToAll sends message to all connected clients
func (h *Hub) BroadcastToAll(event string, data map[string]interface{}) {
	message := WebSocketMessage{
		Type:      "broadcast",
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.broadcast <- jsonMessage
}

// BroadcastToUser sends message to specific user
func (h *Hub) BroadcastToUser(userID uint, event string, data map[string]interface{}) {
	message := WebSocketMessage{
		Type:      "user",
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.UserID == userID && client.IsActive {
			select {
			case client.Send <- jsonMessage:
			default:
				log.Printf("Failed to send message to user %d", userID)
			}
		}
	}
}

// GetConnectedClients returns count of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetUserClients returns all clients for a specific user
func (h *Hub) GetUserClients(userID uint) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var userClients []*Client
	for client := range h.clients {
		if client.UserID == userID {
			userClients = append(userClients, client)
		}
	}
	return userClients
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages
		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Process message based on type
		c.handleMessage(&msg)
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming websocket messages
func (c *Client) handleMessage(msg *WebSocketMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch msg.Event {
	case "ping":
		// Respond with pong
		response := WebSocketMessage{
			Type:      "response",
			Event:     "pong",
			Data:      map[string]interface{}{"client_id": c.ID},
			Timestamp: time.Now(),
		}
		jsonResponse, _ := json.Marshal(response)
		c.Send <- jsonResponse

	case "subscribe":
		// Handle subscription to specific events
		log.Printf("Client %s subscribed to: %v", c.ID, msg.Data)

	case "unsubscribe":
		// Handle unsubscription
		log.Printf("Client %s unsubscribed from: %v", c.ID, msg.Data)

	default:
		log.Printf("Unknown message type: %s", msg.Event)
	}
}

// SendMessage sends a message to this specific client
func (c *Client) SendMessage(event string, data map[string]interface{}) error {
	message := WebSocketMessage{
		Type:      "direct",
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.Send <- jsonMessage:
		return nil
	default:
		return websocket.ErrCloseSent
	}
}

// NotificationService handles real-time notifications
type NotificationService struct {
	hub *Hub
}

// NewNotificationService creates a new notification service
func NewNotificationService(hub *Hub) *NotificationService {
	return &NotificationService{hub: hub}
}

// NotifyPhotoProcessed notifies user when photo is processed
func (ns *NotificationService) NotifyPhotoProcessed(userID uint, photoID uint, photoURL string) {
	ns.hub.BroadcastToUser(userID, "photo_processed", map[string]interface{}{
		"photo_id":  photoID,
		"photo_url": photoURL,
		"status":    "completed",
	})
}

// NotifyOrderPaid notifies user when order is paid
func (ns *NotificationService) NotifyOrderPaid(userID uint, orderID uint, orderNumber string) {
	ns.hub.BroadcastToUser(userID, "order_paid", map[string]interface{}{
		"order_id":     orderID,
		"order_number": orderNumber,
		"status":       "paid",
	})
}

// NotifySubscriptionExpiring notifies user about expiring subscription
func (ns *NotificationService) NotifySubscriptionExpiring(userID uint, daysLeft int) {
	ns.hub.BroadcastToUser(userID, "subscription_expiring", map[string]interface{}{
		"days_left": daysLeft,
		"message":   "Your subscription is expiring soon",
	})
}

// NotifyNewTemplate notifies all users about new template
func (ns *NotificationService) NotifyNewTemplate(templateID uint, templateName string) {
	ns.hub.BroadcastToAll("new_template", map[string]interface{}{
		"template_id":   templateID,
		"template_name": templateName,
		"message":       "New template available!",
	})
}

// NotifySystemMaintenance notifies all users about maintenance
func (ns *NotificationService) NotifySystemMaintenance(message string, scheduledAt time.Time) {
	ns.hub.BroadcastToAll("system_maintenance", map[string]interface{}{
		"message":      message,
		"scheduled_at": scheduledAt,
	})
}

// ServeWs handles websocket requests from clients
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get user ID from query params or context
	userID := uint(0)
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			userID = uint(id)
		}
	}

	client := &Client{
		ID:       generateClientID(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hub,
		IsActive: true,
	}

	client.Hub.Register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

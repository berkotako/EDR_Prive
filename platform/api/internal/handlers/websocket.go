// WebSocket Live Updates Handler

package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// WSHub manages all WebSocket connections
type WSHub struct {
	clients    map[string]*WSClient
	broadcast  chan models.WSMessage
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

// WSClient wraps a WebSocket connection
type WSClient struct {
	id           string
	tenantID     string
	subscription models.WSSubscription
	conn         *websocket.Conn
	send         chan models.WSMessage
	hub          *WSHub
	connectedAt  time.Time
	lastPingAt   time.Time
}

// Global hub instance
var globalHub *WSHub

// InitWebSocketHub initializes the WebSocket hub
func InitWebSocketHub() {
	globalHub = &WSHub{
		clients:    make(map[string]*WSClient),
		broadcast:  make(chan models.WSMessage, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}

	go globalHub.run()
	log.Info("WebSocket hub initialized")
}

// HandleWebSocket handles WebSocket connection requests
func HandleWebSocket(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("Failed to upgrade connection: %v", err)
		return
	}

	// Create new client
	client := &WSClient{
		id:          uuid.New().String(),
		tenantID:    tenantID,
		conn:        conn,
		send:        make(chan models.WSMessage, 256),
		hub:         globalHub,
		connectedAt: time.Now(),
		lastPingAt:  time.Now(),
		subscription: models.WSSubscription{
			TenantID: tenantID,
		},
	}

	// Register client
	globalHub.register <- client

	// Send connected message
	client.send <- models.WSMessage{
		Type:      models.WSTypeConnected,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"client_id": client.id,
			"message":   "Successfully connected to Privé Platform WebSocket",
		},
	}

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	log.Infof("WebSocket client connected: %s (tenant: %s)", client.id, tenantID)
}

// BroadcastEvent broadcasts an event to all subscribed clients
func BroadcastEvent(event models.WSEventNotification) {
	if globalHub != nil {
		globalHub.broadcast <- models.WSMessage{
			Type:      models.WSTypeNewEvent,
			Timestamp: time.Now(),
			Data:      event,
		}
	}
}

// BroadcastAlert broadcasts an alert to all subscribed clients
func BroadcastAlert(alert models.WSAlertNotification) {
	if globalHub != nil {
		globalHub.broadcast <- models.WSMessage{
			Type:      models.WSTypeNewAlert,
			Timestamp: time.Now(),
			Data:      alert,
		}
	}
}

// BroadcastAgentStatus broadcasts agent status change
func BroadcastAgentStatus(status models.WSAgentStatusNotification) {
	if globalHub != nil {
		globalHub.broadcast <- models.WSMessage{
			Type:      models.WSTypeAgentStatus,
			Timestamp: time.Now(),
			Data:      status,
		}
	}
}

// BroadcastStatistics broadcasts real-time statistics
func BroadcastStatistics(stats models.WSStatistics) {
	if globalHub != nil {
		globalHub.broadcast <- models.WSMessage{
			Type:      models.WSTypeSystemNotification,
			Timestamp: time.Now(),
			Data:      stats,
		}
	}
}

// Hub methods

func (h *WSHub) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.id] = client
			h.mu.Unlock()
			log.Infof("Client registered: %s (total: %d)", client.id, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.id]; ok {
				delete(h.clients, client.id)
				close(client.send)
			}
			h.mu.Unlock()
			log.Infof("Client unregistered: %s (remaining: %d)", client.id, len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				// Check if message should be sent to this client
				if h.shouldSendToClient(client, message) {
					select {
					case client.send <- message:
					default:
						// Client send buffer is full, disconnect
						h.mu.RUnlock()
						h.unregister <- client
						h.mu.RLock()
					}
				}
			}
			h.mu.RUnlock()

		case <-ticker.C:
			// Send heartbeat to all clients
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- models.WSMessage{
					Type:      models.WSTypeHeartbeat,
					Timestamp: time.Now(),
				}:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WSHub) shouldSendToClient(client *WSClient, message models.WSMessage) bool {
	// Check tenant isolation
	if message.Type == models.WSTypeNewEvent || message.Type == models.WSTypeNewAlert {
		// For now, send all messages within the same tenant
		// In production, implement subscription filtering
		return true
	}

	// System messages go to all clients
	return true
}

// Client methods

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.lastPingAt = time.Now()
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages
		var incomingMsg models.WSMessage
		if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
			log.Warnf("Failed to parse WebSocket message: %v", err)
			continue
		}

		c.handleMessage(incomingMsg)
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(45 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send message as JSON
			if err := c.conn.WriteJSON(message); err != nil {
				log.Errorf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) handleMessage(msg models.WSMessage) {
	switch msg.Type {
	case models.WSTypeSubscribe:
		// Update subscription preferences
		if data, ok := msg.Data.(map[string]interface{}); ok {
			dataJSON, _ := json.Marshal(data)
			json.Unmarshal(dataJSON, &c.subscription)

			c.send <- models.WSMessage{
				Type:      models.WSTypeSystemNotification,
				Timestamp: time.Now(),
				Data:      map[string]string{"message": "Subscription updated"},
			}
			log.Infof("Client %s updated subscription", c.id)
		}

	case models.WSTypePing:
		// Respond with pong
		c.send <- models.WSMessage{
			Type:      models.WSTypePong,
			Timestamp: time.Now(),
		}

	default:
		log.Warnf("Unknown message type from client %s: %s", c.id, msg.Type)
	}
}

// GetConnectionStats returns WebSocket connection statistics
func GetConnectionStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalHub == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket hub not initialized"})
			return
		}

		globalHub.mu.RLock()
		defer globalHub.mu.RUnlock()

		stats := map[string]interface{}{
			"total_connections": len(globalHub.clients),
			"connections_by_tenant": make(map[string]int),
		}

		// Count connections by tenant
		tenantCounts := make(map[string]int)
		for _, client := range globalHub.clients {
			tenantCounts[client.tenantID]++
		}
		stats["connections_by_tenant"] = tenantCounts

		c.JSON(http.StatusOK, stats)
	}
}

// DisconnectClient disconnects a specific client (admin function)
func DisconnectClient(c *gin.Context) {
	clientID := c.Param("id")

	if globalHub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket hub not initialized"})
		return
	}

	globalHub.mu.RLock()
	client, exists := globalHub.clients[clientID]
	globalHub.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	// Send disconnect message
	client.send <- models.WSMessage{
		Type:      models.WSTypeSystemNotification,
		Timestamp: time.Now(),
		Data:      map[string]string{"message": "Connection closed by administrator"},
	}

	// Close connection
	client.conn.Close()
	globalHub.unregister <- client

	c.JSON(http.StatusOK, gin.H{"message": "Client disconnected successfully"})
}

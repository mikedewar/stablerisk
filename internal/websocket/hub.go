package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/mikedewar/stablerisk/internal/api"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan *api.WebSocketMessage

	// Logger
	logger *zap.Logger

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *api.WebSocketMessage, 256),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start runs the hub's main loop
func (h *Hub) Start() {
	h.wg.Add(1)
	go h.run()
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	h.logger.Info("Shutting down WebSocket hub")
	h.cancel()
	h.wg.Wait()
	h.logger.Info("WebSocket hub shutdown complete")
}

// run is the main event loop for the hub
func (h *Hub) run() {
	defer h.wg.Done()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			h.logger.Info("Client connected",
				zap.String("user_id", client.userID),
				zap.String("username", client.username),
				zap.Int("total_clients", len(h.clients)))

			// Send welcome message
			h.sendToClient(client, &api.WebSocketMessage{
				Type:      "connected",
				Data:      map[string]string{"message": "Connected to StableRisk real-time updates"},
				Timestamp: time.Now(),
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				h.logger.Info("Client disconnected",
					zap.String("user_id", client.userID),
					zap.String("username", client.username),
					zap.Int("total_clients", len(h.clients)))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-h.ctx.Done():
			// Graceful shutdown: close all client connections
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
			}
			h.clients = make(map[*Client]bool)
			h.mu.Unlock()
			return
		}
	}
}

// broadcastMessage sends a message to all connected clients
func (h *Hub) broadcastMessage(message *api.WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	messageJSON, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal WebSocket message",
			zap.Error(err))
		return
	}

	// Extract outlier if this is an outlier message (for filtering)
	var outlier *models.Outlier
	if message.Type == "outlier" {
		if data, ok := message.Data.(models.Outlier); ok {
			outlier = &data
		} else if dataMap, ok := message.Data.(map[string]interface{}); ok {
			// Try to convert map to outlier
			dataJSON, _ := json.Marshal(dataMap)
			var o models.Outlier
			if err := json.Unmarshal(dataJSON, &o); err == nil {
				outlier = &o
			}
		}
	}

	sentCount := 0
	for client := range h.clients {
		// Apply filters if this is an outlier message
		if outlier != nil && !client.matchesFilters(outlier) {
			continue
		}

		select {
		case client.send <- messageJSON:
			sentCount++
		default:
			// Client send buffer is full, close connection
			close(client.send)
			delete(h.clients, client)
			h.logger.Warn("Client send buffer full, closing connection",
				zap.String("user_id", client.userID))
		}
	}

	h.logger.Debug("Broadcast message sent",
		zap.String("type", message.Type),
		zap.Int("recipients", sentCount),
		zap.Int("total_clients", len(h.clients)))
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, message *api.WebSocketMessage) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal WebSocket message",
			zap.Error(err))
		return
	}

	select {
	case client.send <- messageJSON:
	default:
		h.logger.Warn("Failed to send message to client",
			zap.String("user_id", client.userID))
	}
}

// BroadcastOutlier broadcasts an outlier to all connected clients
func (h *Hub) BroadcastOutlier(outlier models.Outlier) {
	h.broadcast <- &api.WebSocketMessage{
		Type:      "outlier",
		Data:      outlier,
		Timestamp: time.Now(),
	}
}

// BroadcastStatistics broadcasts statistics update to all connected clients
func (h *Hub) BroadcastStatistics(stats interface{}) {
	h.broadcast <- &api.WebSocketMessage{
		Type:      "statistics",
		Data:      stats,
		Timestamp: time.Now(),
	}
}

// BroadcastSystemMessage broadcasts a system message to all connected clients
func (h *Hub) BroadcastSystemMessage(message string) {
	h.broadcast <- &api.WebSocketMessage{
		Type:      "system",
		Data:      map[string]string{"message": message},
		Timestamp: time.Now(),
	}
}

// RegisterClient registers a new client with the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientsByUser returns all clients for a specific user
func (h *Hub) GetClientsByUser(userID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var userClients []*Client
	for client := range h.clients {
		if client.userID == userID {
			userClients = append(userClients, client)
		}
	}
	return userClients
}

package websocket

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mikedewar/stablerisk/internal/api"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	username string
	role     models.Role
	filters  *SubscriptionFilters
	logger   *zap.Logger
}

// SubscriptionFilters allows clients to filter which messages they receive
type SubscriptionFilters struct {
	Severities []models.Severity     // Only receive these severities (empty = all)
	Types      []models.OutlierType  // Only receive these types (empty = all)
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID, username string, role models.Role, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		username: username,
		role:     role,
		filters:  &SubscriptionFilters{},
		logger:   logger,
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket read error",
					zap.Error(err),
					zap.String("user_id", c.userID))
			}
			break
		}

		// Handle client messages (subscription filters, etc.)
		c.handleMessage(message)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(message []byte) {
	var msg api.WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Warn("Failed to parse WebSocket message",
			zap.Error(err),
			zap.String("user_id", c.userID))
		return
	}

	switch msg.Type {
	case "subscribe":
		c.handleSubscribe(msg.Data)
	case "pong":
		// Client responded to ping, nothing to do
	default:
		c.logger.Debug("Unknown WebSocket message type",
			zap.String("type", msg.Type),
			zap.String("user_id", c.userID))
	}
}

// handleSubscribe updates client subscription filters
func (c *Client) handleSubscribe(data interface{}) {
	filterData, ok := data.(map[string]interface{})
	if !ok {
		c.logger.Warn("Invalid subscribe message format",
			zap.String("user_id", c.userID))
		return
	}

	// Update severities filter
	if severitiesRaw, ok := filterData["severities"].([]interface{}); ok {
		severities := make([]models.Severity, 0, len(severitiesRaw))
		for _, s := range severitiesRaw {
			if sev, ok := s.(string); ok {
				severities = append(severities, models.Severity(sev))
			}
		}
		c.filters.Severities = severities
	}

	// Update types filter
	if typesRaw, ok := filterData["types"].([]interface{}); ok {
		types := make([]models.OutlierType, 0, len(typesRaw))
		for _, t := range typesRaw {
			if typ, ok := t.(string); ok {
				types = append(types, models.OutlierType(typ))
			}
		}
		c.filters.Types = types
	}

	c.logger.Debug("Updated client subscription filters",
		zap.String("user_id", c.userID),
		zap.Int("severities", len(c.filters.Severities)),
		zap.Int("types", len(c.filters.Types)))
}

// matchesFilters checks if an outlier matches the client's subscription filters
func (c *Client) matchesFilters(outlier *models.Outlier) bool {
	// Check severity filter
	if len(c.filters.Severities) > 0 {
		match := false
		for _, sev := range c.filters.Severities {
			if outlier.Severity == sev {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// Check type filter
	if len(c.filters.Types) > 0 {
		match := false
		for _, typ := range c.filters.Types {
			if outlier.Type == typ {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

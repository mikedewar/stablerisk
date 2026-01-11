package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mikedewar/stablerisk/internal/security"
	ws "github.com/mikedewar/stablerisk/internal/websocket"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check origin properly
		// For now, allow all origins
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub        *ws.Hub
	jwtManager *security.JWTManager
	logger     *zap.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *ws.Hub, jwtManager *security.JWTManager, logger *zap.Logger) *WebSocketHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &WebSocketHandler{
		hub:        hub,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Extract token from query parameter (since WebSocket can't send custom headers)
	token := c.Query("token")
	if token == "" {
		// Try header as fallback
		token = c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication token required",
		})
		return
	}

	// Validate token
	claims, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		h.logger.Warn("WebSocket authentication failed",
			zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid or expired token",
		})
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection",
			zap.Error(err),
			zap.String("user_id", claims.UserID))
		return
	}

	// Create client
	client := ws.NewClient(
		h.hub,
		conn,
		claims.UserID,
		claims.Username,
		claims.Role,
		h.logger,
	)

	// Register client with hub
	h.hub.RegisterClient(client)

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()

	h.logger.Info("WebSocket connection established",
		zap.String("user_id", claims.UserID),
		zap.String("username", claims.Username),
		zap.String("role", string(claims.Role)))
}

// BroadcastOutlier broadcasts an outlier to all connected clients
func (h *WebSocketHandler) BroadcastOutlier(outlier models.Outlier) {
	h.hub.BroadcastOutlier(outlier)
}

// GetConnectionCount returns the number of active WebSocket connections
func (h *WebSocketHandler) GetConnectionCount() int {
	return h.hub.ClientCount()
}

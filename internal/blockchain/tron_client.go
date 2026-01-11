package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// TronClient manages WebSocket connection to TronGrid
type TronClient struct {
	apiKey       string
	wsURL        string
	usdtContract string
	conn         *websocket.Conn
	parser       *TransactionParser
	retryHandler *RetryHandler
	logger       *zap.Logger

	// Channels
	txChannel   chan *models.Transaction
	errChannel  chan error
	closeSignal chan struct{}

	// State
	status     models.ConnectionStatus
	statusLock sync.RWMutex
	connected  bool
	ctx        context.Context
	cancel     context.CancelFunc

	// Configuration
	pingInterval time.Duration
	pongWait     time.Duration
	writeWait    time.Duration
}

// TronClientConfig holds TronGrid client configuration
type TronClientConfig struct {
	APIKey       string
	WebSocketURL string
	USDTContract string
	PingInterval time.Duration
	RetryConfig  RetryConfig
}

// NewTronClient creates a new TronGrid WebSocket client
func NewTronClient(config TronClientConfig, logger *zap.Logger) *TronClient {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &TronClient{
		apiKey:       config.APIKey,
		wsURL:        config.WebSocketURL,
		usdtContract: config.USDTContract,
		parser:       NewTransactionParser(config.USDTContract),
		retryHandler: NewRetryHandler(config.RetryConfig, logger),
		logger:       logger,
		txChannel:    make(chan *models.Transaction, 100),
		errChannel:   make(chan error, 10),
		closeSignal:  make(chan struct{}),
		status:       models.StatusDisconnected,
		connected:    false,
		ctx:          ctx,
		cancel:       cancel,
		pingInterval: config.PingInterval,
		pongWait:     60 * time.Second,
		writeWait:    10 * time.Second,
	}

	return client
}

// Connect establishes WebSocket connection to TronGrid
func (c *TronClient) Connect() error {
	c.setStatus(models.StatusConnecting)
	c.logger.Info("Connecting to TronGrid WebSocket",
		zap.String("url", c.wsURL),
		zap.String("contract", c.usdtContract))

	// Build WebSocket URL with contract subscription
	wsURL := fmt.Sprintf("%s/event/contract/%s", c.wsURL, c.usdtContract)

	// Create HTTP headers with API key
	headers := http.Header{}
	headers.Set("TRON-PRO-API-KEY", c.apiKey)

	// Dial WebSocket
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		c.setStatus(models.StatusError)
		if resp != nil {
			return fmt.Errorf("failed to connect to TronGrid (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("failed to connect to TronGrid: %w", err)
	}

	c.conn = conn
	c.connected = true
	c.setStatus(models.StatusConnected)
	c.retryHandler.Reset()

	c.logger.Info("Successfully connected to TronGrid WebSocket")

	// Start message handlers
	go c.readMessages()
	go c.pingLoop()

	return nil
}

// readMessages reads messages from WebSocket
func (c *TronClient) readMessages() {
	defer func() {
		c.connected = false
		c.setStatus(models.StatusDisconnected)
	}()

	// Set pong handler
	c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Read message
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket read error", zap.Error(err))
			}
			c.errChannel <- err
			return
		}

		// Parse and process message
		if err := c.processMessage(message); err != nil {
			c.logger.Warn("Failed to process message",
				zap.Error(err),
				zap.String("message", string(message)))
		}
	}
}

// processMessage parses and processes a WebSocket message
func (c *TronClient) processMessage(message []byte) error {
	// Parse as TronEvent
	var event models.TronEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Parse into transaction
	tx, err := c.parser.ParseEvent(&event)
	if err != nil {
		// Not all events are valid transactions (e.g., wrong contract, non-Transfer events)
		return err
	}

	// Validate transaction
	if err := ValidateTransaction(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Send to transaction channel
	select {
	case c.txChannel <- tx:
		c.logger.Debug("Transaction received",
			zap.String("tx_hash", tx.TxHash),
			zap.String("from", tx.From),
			zap.String("to", tx.To),
			zap.String("amount", tx.Amount.String()))
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
		c.logger.Warn("Transaction channel full, dropping transaction",
			zap.String("tx_hash", tx.TxHash))
	}

	return nil
}

// pingLoop sends periodic ping messages to keep connection alive
func (c *TronClient) pingLoop() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.sendPing(); err != nil {
				c.logger.Error("Failed to send ping", zap.Error(err))
				c.errChannel <- err
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// sendPing sends a ping message
func (c *TronClient) sendPing() error {
	c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

// Start starts the client with automatic reconnection
func (c *TronClient) Start() error {
	c.logger.Info("Starting TronGrid client")

	// Initial connection
	if err := c.Connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start reconnection loop
	go c.reconnectionLoop()

	return nil
}

// reconnectionLoop handles automatic reconnection on disconnect
func (c *TronClient) reconnectionLoop() {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Reconnection loop stopped")
			return
		case err := <-c.errChannel:
			c.logger.Error("Connection error, attempting to reconnect",
				zap.Error(err))

			// Close existing connection
			if c.conn != nil {
				c.conn.Close()
			}
			c.connected = false
			c.setStatus(models.StatusReconnecting)

			// Retry connection with exponential backoff
			for c.retryHandler.ShouldRetry() {
				// Wait before retry
				if err := c.retryHandler.Wait(c.ctx); err != nil {
					c.logger.Info("Reconnection cancelled", zap.Error(err))
					return
				}

				// Attempt to reconnect
				if err := c.Connect(); err != nil {
					c.logger.Error("Reconnection attempt failed",
						zap.Error(err),
						zap.Int("attempt", c.retryHandler.GetAttempt()))
					continue
				}

				// Success
				c.logger.Info("Successfully reconnected to TronGrid")
				break
			}

			// Check if circuit breaker opened
			if c.retryHandler.IsCircuitOpen() {
				c.logger.Warn("Circuit breaker open, will retry after timeout")
				// Wait for circuit breaker timeout
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(c.retryHandler.config.CircuitTimeout):
					c.retryHandler.Reset()
				}
			}
		}
	}
}

// Transactions returns the transaction channel
func (c *TronClient) Transactions() <-chan *models.Transaction {
	return c.txChannel
}

// Status returns the current connection status
func (c *TronClient) Status() models.ConnectionStatus {
	c.statusLock.RLock()
	defer c.statusLock.RUnlock()
	return c.status
}

// setStatus sets the connection status
func (c *TronClient) setStatus(status models.ConnectionStatus) {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()

	if c.status != status {
		c.logger.Info("Status changed",
			zap.String("from", string(c.status)),
			zap.String("to", string(status)))
		c.status = status
	}
}

// IsConnected returns whether the client is connected
func (c *TronClient) IsConnected() bool {
	return c.connected && c.Status() == models.StatusConnected
}

// Close closes the WebSocket connection and stops the client
func (c *TronClient) Close() error {
	c.logger.Info("Closing TronGrid client")

	// Cancel context to stop all goroutines
	c.cancel()

	// Close WebSocket connection
	if c.conn != nil {
		// Send close message
		err := c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			c.logger.Warn("Failed to send close message", zap.Error(err))
		}

		// Close connection
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	c.connected = false
	c.setStatus(models.StatusDisconnected)

	// Close channels
	close(c.closeSignal)

	c.logger.Info("TronGrid client closed")
	return nil
}

package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// TronClient manages REST API polling to TronGrid
type TronClient struct {
	apiKey       string
	apiURL       string
	usdtContract string
	httpClient   *http.Client
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
	pollingInterval time.Duration
	lastTimestamp   int64 // Track last processed event timestamp to avoid duplicates
	timestampLock   sync.RWMutex
}

// TronClientConfig holds TronGrid client configuration
type TronClientConfig struct {
	APIKey          string
	WebSocketURL    string        // Kept for backwards compatibility, but will use as API URL
	USDTContract    string
	PingInterval    time.Duration // Used as polling interval
	RetryConfig     RetryConfig
}

// NewTronClient creates a new TronGrid REST API client
func NewTronClient(config TronClientConfig, logger *zap.Logger) *TronClient {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Convert WebSocket URL to REST API URL
	apiURL := config.WebSocketURL
	if apiURL == "wss://api.trongrid.io" || apiURL == "ws://api.trongrid.io" {
		apiURL = "https://api.trongrid.io"
	}

	// Use PingInterval as polling interval (default to 10 seconds if not set)
	pollingInterval := config.PingInterval
	if pollingInterval == 0 {
		pollingInterval = 10 * time.Second
	}

	client := &TronClient{
		apiKey:       config.APIKey,
		apiURL:       apiURL,
		usdtContract: config.USDTContract,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		parser:          NewTransactionParser(config.USDTContract),
		retryHandler:    NewRetryHandler(config.RetryConfig, logger),
		logger:          logger,
		txChannel:       make(chan *models.Transaction, 100),
		errChannel:      make(chan error, 10),
		closeSignal:     make(chan struct{}),
		status:          models.StatusDisconnected,
		connected:       false,
		ctx:             ctx,
		cancel:          cancel,
		pollingInterval: pollingInterval,
		lastTimestamp:   0,
	}

	return client
}

// TronEventResponse represents the TronGrid API response
type TronEventResponse struct {
	Success bool              `json:"success"`
	Data    []models.TronEvent `json:"data"`
	Meta    struct {
		At          int64  `json:"at"`
		Fingerprint string `json:"fingerprint"`
	} `json:"meta"`
}

// Connect verifies connection to TronGrid API
func (c *TronClient) Connect() error {
	c.setStatus(models.StatusConnecting)
	c.logger.Info("Connecting to TronGrid REST API",
		zap.String("url", c.apiURL),
		zap.String("contract", c.usdtContract))

	// Test API connectivity with a simple request
	endpoint := fmt.Sprintf("%s/v1/contracts/%s/events", c.apiURL, c.usdtContract)

	req, err := http.NewRequestWithContext(c.ctx, "GET", endpoint, nil)
	if err != nil {
		c.setStatus(models.StatusError)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// Add query parameters for initial test
	q := req.URL.Query()
	q.Add("limit", "1")
	q.Add("only_confirmed", "true")
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setStatus(models.StatusError)
		return fmt.Errorf("failed to connect to TronGrid API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.setStatus(models.StatusError)
		return fmt.Errorf("TronGrid API returned status %d: %s", resp.StatusCode, string(body))
	}

	c.connected = true
	c.setStatus(models.StatusConnected)
	c.retryHandler.Reset()

	c.logger.Info("Successfully connected to TronGrid REST API")

	return nil
}

// pollEvents polls for new events from TronGrid
func (c *TronClient) pollEvents() {
	ticker := time.NewTicker(c.pollingInterval)
	defer ticker.Stop()

	c.logger.Info("Starting TronGrid event polling",
		zap.Duration("interval", c.pollingInterval))

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Event polling stopped")
			return
		case <-ticker.C:
			if err := c.fetchEvents(); err != nil {
				c.logger.Error("Failed to fetch events", zap.Error(err))
				c.errChannel <- err
			}
		}
	}
}

// fetchEvents retrieves events from TronGrid API
func (c *TronClient) fetchEvents() error {
	endpoint := fmt.Sprintf("%s/v1/contracts/%s/events", c.apiURL, c.usdtContract)

	req, err := http.NewRequestWithContext(c.ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// Add query parameters
	q := req.URL.Query()
	q.Add("limit", "200") // Fetch up to 200 events per poll
	q.Add("only_confirmed", "true") // Only get confirmed transactions
	q.Add("order_by", "block_timestamp,asc") // Oldest first

	// Add min timestamp to avoid fetching old events
	c.timestampLock.RLock()
	lastTimestamp := c.lastTimestamp
	c.timestampLock.RUnlock()

	if lastTimestamp > 0 {
		// Add 1ms to avoid getting the same event again
		q.Add("min_block_timestamp", fmt.Sprintf("%d", lastTimestamp+1))
	}

	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TronGrid API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var eventResp TronEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&eventResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !eventResp.Success {
		return fmt.Errorf("TronGrid API returned success=false")
	}

	// Process events
	c.logger.Debug("Fetched events from TronGrid",
		zap.Int("count", len(eventResp.Data)))

	for _, event := range eventResp.Data {
		if err := c.processEvent(&event); err != nil {
			c.logger.Warn("Failed to process event",
				zap.Error(err),
				zap.String("tx_hash", event.TransactionID))
		}

		// Update last timestamp
		if event.BlockTimestamp > 0 {
			c.timestampLock.Lock()
			if event.BlockTimestamp > c.lastTimestamp {
				c.lastTimestamp = event.BlockTimestamp
			}
			c.timestampLock.Unlock()
		}
	}

	return nil
}

// processEvent parses and processes a TronGrid event
func (c *TronClient) processEvent(event *models.TronEvent) error {
	// Parse into transaction
	tx, err := c.parser.ParseEvent(event)
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
		c.logger.Debug("Transaction processed",
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

// Start starts the client with automatic reconnection
func (c *TronClient) Start() error {
	c.logger.Info("Starting TronGrid client")

	// Initial connection test
	if err := c.Connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start polling loop
	go c.pollEvents()

	// Start reconnection handler
	go c.reconnectionLoop()

	return nil
}

// reconnectionLoop handles automatic reconnection on errors
func (c *TronClient) reconnectionLoop() {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Reconnection loop stopped")
			return
		case err := <-c.errChannel:
			c.logger.Error("Connection error, attempting to reconnect",
				zap.Error(err))

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

// Close closes the client and stops polling
func (c *TronClient) Close() error {
	c.logger.Info("Closing TronGrid client")

	// Cancel context to stop all goroutines
	c.cancel()

	c.connected = false
	c.setStatus(models.StatusDisconnected)

	// Close channels
	close(c.closeSignal)

	c.logger.Info("TronGrid client closed")
	return nil
}

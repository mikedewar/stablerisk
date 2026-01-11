package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// RaphtoryClient manages communication with Raphtory service
type RaphtoryClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// RaphtoryConfig holds Raphtory client configuration
type RaphtoryConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// NewRaphtoryClient creates a new Raphtory client
func NewRaphtoryClient(config RaphtoryConfig, logger *zap.Logger) *RaphtoryClient {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &RaphtoryClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// AddTransaction sends a transaction to Raphtory to add to the graph
func (c *RaphtoryClient) AddTransaction(ctx context.Context, tx *models.Transaction) error {
	// Prepare request payload
	payload := map[string]interface{}{
		"tx_hash":      tx.TxHash,
		"block_number": tx.BlockNumber,
		"timestamp":    tx.Timestamp.Unix(),
		"from":         tx.From,
		"to":           tx.To,
		"amount":       tx.Amount.String(),
		"contract":     tx.Contract,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Send HTTP POST request
	url := fmt.Sprintf("%s/graph/transaction", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("raphtory returned status %d", resp.StatusCode)
	}

	c.logger.Debug("Transaction added to Raphtory",
		zap.String("tx_hash", tx.TxHash),
		zap.String("from", tx.From),
		zap.String("to", tx.To))

	return nil
}

// NodeInfo represents node information from Raphtory
type NodeInfo struct {
	Address          string  `json:"address"`
	FirstSeen        int64   `json:"first_seen"`
	LastSeen         int64   `json:"last_seen"`
	TransactionCount int     `json:"transaction_count"`
	SentCount        int     `json:"sent_count"`
	ReceivedCount    int     `json:"received_count"`
	TotalSent        float64 `json:"total_sent"`
	TotalReceived    float64 `json:"total_received"`
}

// TransactionInfo represents a transaction from Raphtory
type TransactionInfo struct {
	TxHash      string `json:"tx_hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	BlockNumber int    `json:"block_number"`
	Timestamp   int64  `json:"timestamp"`
}

// GetNodeInfo gets information about a node from Raphtory
func (c *RaphtoryClient) GetNodeInfo(ctx context.Context, address string) (*NodeInfo, error) {
	url := fmt.Sprintf("%s/graph/node/%s", c.baseURL, address)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("raphtory returned status %d", resp.StatusCode)
	}

	var nodeInfo NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &nodeInfo, nil
}

// GetTransactionsInWindow gets transactions in a time window
func (c *RaphtoryClient) GetTransactionsInWindow(ctx context.Context, startTime, endTime int64, limit int) ([]models.Transaction, error) {
	url := fmt.Sprintf("%s/graph/window?start=%d&end=%d&limit=%d", c.baseURL, startTime, endTime, limit)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("raphtory returned status %d", resp.StatusCode)
	}

	var txInfos []TransactionInfo
	if err := json.NewDecoder(resp.Body).Decode(&txInfos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to models.Transaction
	transactions := make([]models.Transaction, len(txInfos))
	for i, txInfo := range txInfos {
		amount, _ := decimal.NewFromString(txInfo.Amount)
		transactions[i] = models.Transaction{
			TxHash:      txInfo.TxHash,
			From:        txInfo.From,
			To:          txInfo.To,
			Amount:      amount,
			BlockNumber: uint64(txInfo.BlockNumber),
			Timestamp:   time.Unix(txInfo.Timestamp, 0),
		}
	}

	return transactions, nil
}

// Health checks if Raphtory service is healthy
func (c *RaphtoryClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("raphtory health check failed with status %d", resp.StatusCode)
	}

	return nil
}

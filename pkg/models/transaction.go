package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a USDT TRC20 transaction on Tron blockchain
type Transaction struct {
	TxHash      string          `json:"tx_hash"`
	BlockNumber uint64          `json:"block_number"`
	Timestamp   time.Time       `json:"timestamp"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	Amount      decimal.Decimal `json:"amount"`
	Contract    string          `json:"contract"`
	Confirmed   bool            `json:"confirmed"`
}

// TronEvent represents a raw event from TronGrid WebSocket
type TronEvent struct {
	TransactionID string                 `json:"transaction_id"`
	ContractAddress string               `json:"contract_address"`
	CallerAddress string                 `json:"caller_address"`
	OriginAddress string                 `json:"origin_address"`
	EventName     string                 `json:"event_name"`
	EventData     map[string]interface{} `json:"event"`
	BlockNumber   uint64                 `json:"block_number"`
	BlockTimestamp int64                 `json:"block_timestamp"`
	Removed       bool                   `json:"removed"`
}

// TransferEvent represents a decoded Transfer event
type TransferEvent struct {
	From   string          `json:"from"`
	To     string          `json:"to"`
	Value  decimal.Decimal `json:"value"`
}

// ConnectionStatus represents the WebSocket connection status
type ConnectionStatus string

const (
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
	StatusReconnecting ConnectionStatus = "reconnecting"
	StatusError        ConnectionStatus = "error"
)

// TronGridError represents an error response from TronGrid
type TronGridError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *TronGridError) Error() string {
	return e.Message
}

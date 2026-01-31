package blockchain

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/shopspring/decimal"
)

const (
	// TRC20 Transfer event signature: Transfer(address,address,uint256)
	TransferEventSignature = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	// USDT TRC20 has 6 decimals
	USDTDecimals = 6
)

// TransactionParser handles parsing of Tron events into transactions
type TransactionParser struct {
	usdtContract string
}

// NewTransactionParser creates a new transaction parser
func NewTransactionParser(usdtContract string) *TransactionParser {
	return &TransactionParser{
		usdtContract: strings.ToLower(strings.TrimSpace(usdtContract)),
	}
}

// ParseEvent parses a TronEvent into a Transaction
func (p *TransactionParser) ParseEvent(event *models.TronEvent) (*models.Transaction, error) {
	// Validate event
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	// Check if this is a Transfer event
	if event.EventName != "Transfer" {
		return nil, fmt.Errorf("not a Transfer event: %s", event.EventName)
	}

	// Check if this is from the USDT contract
	contractAddr := strings.ToLower(strings.TrimSpace(event.ContractAddress))
	if contractAddr != p.usdtContract {
		return nil, fmt.Errorf("not a USDT contract event: %s", event.ContractAddress)
	}

	// Parse transfer event data from Result field
	transfer, err := p.parseTransferEvent(event.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transfer event: %w", err)
	}

	// Convert timestamp from milliseconds to time.Time
	timestamp := time.Unix(event.BlockTimestamp/1000, (event.BlockTimestamp%1000)*int64(time.Millisecond))

	// Create transaction
	tx := &models.Transaction{
		TxHash:      event.TransactionID,
		BlockNumber: event.BlockNumber,
		Timestamp:   timestamp,
		From:        transfer.From,
		To:          transfer.To,
		Amount:      transfer.Value,
		Contract:    event.ContractAddress,
		Confirmed:   true,
	}

	return tx, nil
}

// parseTransferEvent extracts transfer data from event data
func (p *TransactionParser) parseTransferEvent(eventData map[string]interface{}) (*models.TransferEvent, error) {
	// Extract from address
	fromAddr, err := p.extractAddress(eventData, "from")
	if err != nil {
		return nil, fmt.Errorf("failed to extract from address: %w", err)
	}

	// Extract to address
	toAddr, err := p.extractAddress(eventData, "to")
	if err != nil {
		return nil, fmt.Errorf("failed to extract to address: %w", err)
	}

	// Extract value
	value, err := p.extractValue(eventData, "value")
	if err != nil {
		return nil, fmt.Errorf("failed to extract value: %w", err)
	}

	// Convert value from smallest unit to USDT (6 decimals)
	amount := decimal.NewFromBigInt(value, -USDTDecimals)

	return &models.TransferEvent{
		From:  fromAddr,
		To:    toAddr,
		Value: amount,
	}, nil
}

// extractAddress extracts a Tron address from event data
func (p *TransactionParser) extractAddress(eventData map[string]interface{}, key string) (string, error) {
	val, ok := eventData[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in event data", key)
	}

	// Address can be in different formats
	switch v := val.(type) {
	case string:
		return p.normalizeAddress(v), nil
	case map[string]interface{}:
		// Sometimes addresses come as objects with hex/base58 fields
		if hexAddr, ok := v["hex"].(string); ok {
			return p.hexToBase58(hexAddr)
		}
		if base58Addr, ok := v["base58"].(string); ok {
			return base58Addr, nil
		}
		return "", fmt.Errorf("address object missing hex/base58 fields")
	default:
		return "", fmt.Errorf("unexpected address type: %T", v)
	}
}

// extractValue extracts a numeric value from event data
func (p *TransactionParser) extractValue(eventData map[string]interface{}, key string) (*big.Int, error) {
	val, ok := eventData[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found in event data", key)
	}

	// Value can be string (hex or decimal) or number
	switch v := val.(type) {
	case string:
		return p.parseStringValue(v)
	case float64:
		return big.NewInt(int64(v)), nil
	case int64:
		return big.NewInt(v), nil
	case int:
		return big.NewInt(int64(v)), nil
	default:
		return nil, fmt.Errorf("unexpected value type: %T", v)
	}
}

// parseStringValue parses a string value (hex or decimal)
func (p *TransactionParser) parseStringValue(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)

	// Try hex format first
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		hexStr := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
		value := new(big.Int)
		if _, ok := value.SetString(hexStr, 16); !ok {
			return nil, fmt.Errorf("invalid hex value: %s", s)
		}
		return value, nil
	}

	// Try decimal format
	value := new(big.Int)
	if _, ok := value.SetString(s, 10); !ok {
		return nil, fmt.Errorf("invalid decimal value: %s", s)
	}
	return value, nil
}

// normalizeAddress normalizes a Tron address to base58 format
func (p *TransactionParser) normalizeAddress(addr string) string {
	addr = strings.TrimSpace(addr)

	// If already in base58 format (starts with T)
	if strings.HasPrefix(addr, "T") {
		return addr
	}

	// If in hex format, convert to base58
	if strings.HasPrefix(addr, "0x") || strings.HasPrefix(addr, "41") {
		// For now, return as-is; proper base58 conversion would require crypto library
		// In production, use github.com/fbsobreira/gotron-sdk for proper conversion
		return addr
	}

	return addr
}

// hexToBase58 converts a hex address to base58 (placeholder)
func (p *TransactionParser) hexToBase58(hexAddr string) (string, error) {
	// This is a placeholder. In production, use proper Tron address conversion
	// from github.com/fbsobreira/gotron-sdk/pkg/address
	hexAddr = strings.TrimPrefix(hexAddr, "0x")
	hexAddr = strings.TrimPrefix(hexAddr, "0X")

	// For now, just validate and return
	if len(hexAddr) != 40 && len(hexAddr) != 42 {
		return "", fmt.Errorf("invalid hex address length: %d", len(hexAddr))
	}

	// Validate hex
	if _, err := hex.DecodeString(hexAddr); err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}

	return "0x" + hexAddr, nil
}

// ValidateTransaction performs basic validation on a transaction
func ValidateTransaction(tx *models.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	if tx.TxHash == "" {
		return fmt.Errorf("transaction hash is empty")
	}

	if tx.From == "" {
		return fmt.Errorf("from address is empty")
	}

	if tx.To == "" {
		return fmt.Errorf("to address is empty")
	}

	if tx.Amount.IsNegative() {
		return fmt.Errorf("amount is negative: %s", tx.Amount.String())
	}

	if tx.BlockNumber == 0 {
		return fmt.Errorf("block number is zero")
	}

	if tx.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is zero")
	}

	return nil
}

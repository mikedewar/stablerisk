package blockchain_test

import (
	"testing"
	"time"

	"github.com/mikedewar/stablerisk/internal/blockchain"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUSDTContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	testTxHash       = "0x1234567890abcdef"
	testFromAddress  = "TFromAddress123456789"
	testToAddress    = "TToAddress123456789"
)

func TestTransactionParser_ParseEvent(t *testing.T) {
	parser := blockchain.NewTransactionParser(testUSDTContract)

	tests := []struct {
		name    string
		event   *models.TronEvent
		wantErr bool
		check   func(t *testing.T, tx *models.Transaction)
	}{
		{
			name: "valid transfer event",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"to":    testToAddress,
					"value": "1000000", // 1 USDT (6 decimals)
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
				Removed:        false,
			},
			wantErr: false,
			check: func(t *testing.T, tx *models.Transaction) {
				assert.Equal(t, testTxHash, tx.TxHash)
				assert.Equal(t, testFromAddress, tx.From)
				assert.Equal(t, testToAddress, tx.To)
				assert.Equal(t, "1", tx.Amount.String()) // 1 USDT
				assert.Equal(t, uint64(12345), tx.BlockNumber)
				assert.True(t, tx.Confirmed)
			},
		},
		{
			name: "large amount transfer",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"to":    testToAddress,
					"value": "1000000000000", // 1,000,000 USDT
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
				Removed:        false,
			},
			wantErr: false,
			check: func(t *testing.T, tx *models.Transaction) {
				expected := decimal.NewFromInt(1000000) // 1M USDT
				assert.True(t, expected.Equal(tx.Amount))
			},
		},
		{
			name: "hex value format",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"to":    testToAddress,
					"value": "0xf4240", // 1000000 in hex = 1 USDT
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
				Removed:        false,
			},
			wantErr: false,
			check: func(t *testing.T, tx *models.Transaction) {
				assert.Equal(t, "1", tx.Amount.String())
			},
		},
		{
			name:    "nil event",
			event:   nil,
			wantErr: true,
		},
		{
			name: "wrong event type",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Approval",
				EventData:       map[string]interface{}{},
				BlockNumber:     12345,
				BlockTimestamp:  time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "wrong contract",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: "TWrongContract123456789",
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"to":    testToAddress,
					"value": "1000000",
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "removed event",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"to":    testToAddress,
					"value": "1000000",
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
				Removed:        true, // Removed due to reorg
			},
			wantErr: true,
		},
		{
			name: "missing from address",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"to":    testToAddress,
					"value": "1000000",
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing to address",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"value": "1000000",
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing value",
			event: &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from": testFromAddress,
					"to":   testToAddress,
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := parser.ParseEvent(tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tx)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tx)
				if tt.check != nil {
					tt.check(t, tx)
				}
			}
		})
	}
}

func TestValidateTransaction(t *testing.T) {
	validTx := &models.Transaction{
		TxHash:      testTxHash,
		BlockNumber: 12345,
		Timestamp:   time.Now(),
		From:        testFromAddress,
		To:          testToAddress,
		Amount:      decimal.NewFromInt(100),
		Contract:    testUSDTContract,
		Confirmed:   true,
	}

	tests := []struct {
		name    string
		tx      *models.Transaction
		wantErr bool
	}{
		{
			name:    "valid transaction",
			tx:      validTx,
			wantErr: false,
		},
		{
			name:    "nil transaction",
			tx:      nil,
			wantErr: true,
		},
		{
			name: "empty tx hash",
			tx: &models.Transaction{
				TxHash:      "",
				BlockNumber: 12345,
				Timestamp:   time.Now(),
				From:        testFromAddress,
				To:          testToAddress,
				Amount:      decimal.NewFromInt(100),
			},
			wantErr: true,
		},
		{
			name: "empty from address",
			tx: &models.Transaction{
				TxHash:      testTxHash,
				BlockNumber: 12345,
				Timestamp:   time.Now(),
				From:        "",
				To:          testToAddress,
				Amount:      decimal.NewFromInt(100),
			},
			wantErr: true,
		},
		{
			name: "empty to address",
			tx: &models.Transaction{
				TxHash:      testTxHash,
				BlockNumber: 12345,
				Timestamp:   time.Now(),
				From:        testFromAddress,
				To:          "",
				Amount:      decimal.NewFromInt(100),
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			tx: &models.Transaction{
				TxHash:      testTxHash,
				BlockNumber: 12345,
				Timestamp:   time.Now(),
				From:        testFromAddress,
				To:          testToAddress,
				Amount:      decimal.NewFromInt(-100),
			},
			wantErr: true,
		},
		{
			name: "zero block number",
			tx: &models.Transaction{
				TxHash:      testTxHash,
				BlockNumber: 0,
				Timestamp:   time.Now(),
				From:        testFromAddress,
				To:          testToAddress,
				Amount:      decimal.NewFromInt(100),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			tx: &models.Transaction{
				TxHash:      testTxHash,
				BlockNumber: 12345,
				Timestamp:   time.Time{},
				From:        testFromAddress,
				To:          testToAddress,
				Amount:      decimal.NewFromInt(100),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := blockchain.ValidateTransaction(tt.tx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransactionParser_DecimalConversion(t *testing.T) {
	parser := blockchain.NewTransactionParser(testUSDTContract)

	tests := []struct {
		name          string
		rawValue      string
		expectedUSDT  string
		expectedFloat float64
	}{
		{
			name:          "1 USDT",
			rawValue:      "1000000",
			expectedUSDT:  "1",
			expectedFloat: 1.0,
		},
		{
			name:          "0.5 USDT",
			rawValue:      "500000",
			expectedUSDT:  "0.5",
			expectedFloat: 0.5,
		},
		{
			name:          "1000000 USDT",
			rawValue:      "1000000000000",
			expectedUSDT:  "1000000",
			expectedFloat: 1000000.0,
		},
		{
			name:          "0.000001 USDT (minimum)",
			rawValue:      "1",
			expectedUSDT:  "0.000001",
			expectedFloat: 0.000001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &models.TronEvent{
				TransactionID:   testTxHash,
				ContractAddress: testUSDTContract,
				EventName:       "Transfer",
				EventData: map[string]interface{}{
					"from":  testFromAddress,
					"to":    testToAddress,
					"value": tt.rawValue,
				},
				BlockNumber:    12345,
				BlockTimestamp: time.Now().UnixMilli(),
			}

			tx, err := parser.ParseEvent(event)
			require.NoError(t, err)
			require.NotNil(t, tx)

			assert.Equal(t, tt.expectedUSDT, tx.Amount.String())

			// Check float conversion
			floatVal, _ := tx.Amount.Float64()
			assert.InDelta(t, tt.expectedFloat, floatVal, 0.000001)
		})
	}
}

// Benchmark for parser performance
func BenchmarkTransactionParser_ParseEvent(b *testing.B) {
	parser := blockchain.NewTransactionParser(testUSDTContract)
	event := &models.TronEvent{
		TransactionID:   testTxHash,
		ContractAddress: testUSDTContract,
		EventName:       "Transfer",
		EventData: map[string]interface{}{
			"from":  testFromAddress,
			"to":    testToAddress,
			"value": "1000000",
		},
		BlockNumber:    12345,
		BlockTimestamp: time.Now().UnixMilli(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseEvent(event)
	}
}

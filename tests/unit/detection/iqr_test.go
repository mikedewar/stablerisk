package detection_test

import (
	"testing"
	"time"

	"github.com/mikedewar/stablerisk/internal/detection"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestIQRDetector_Detect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := detection.IQRConfig{
		Multiplier:     1.5,
		WindowDuration: 24 * time.Hour,
		MinDataPoints:  10,
	}
	detector := detection.NewIQRDetector(config, logger)

	t.Run("detect outlier beyond upper bound", func(t *testing.T) {
		// Create transactions: mostly between 90-110, with one outlier at 200
		transactions := make([]models.Transaction, 0, 20)
		for i := 0; i < 19; i++ {
			transactions = append(transactions, createTransaction(
				generateTxHash(i),
				"A", "B",
				"100",
				time.Now(),
			))
		}

		// Add outlier
		outlierTx := createTransaction("outlier", "A", "B", "500", time.Now())
		transactions = append(transactions, outlierTx)

		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Greater(t, len(outliers), 0, "Should detect outlier")

		// Verify outlier was detected
		found := false
		for _, o := range outliers {
			if o.TransactionHash == "outlier" {
				found = true
				assert.Equal(t, models.OutlierTypeIQR, o.Type)
				assert.NotEmpty(t, o.Severity)
				break
			}
		}
		assert.True(t, found, "Outlier should be detected")
	})

	t.Run("detect outlier below lower bound", func(t *testing.T) {
		transactions := make([]models.Transaction, 0, 20)
		for i := 0; i < 19; i++ {
			transactions = append(transactions, createTransaction(
				generateTxHash(i),
				"A", "B",
				"1000",
				time.Now(),
			))
		}

		// Add low outlier
		outlierTx := createTransaction("lowOutlier", "A", "B", "1", time.Now())
		transactions = append(transactions, outlierTx)

		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Greater(t, len(outliers), 0, "Should detect low outlier")
	})

	t.Run("insufficient data points", func(t *testing.T) {
		transactions := generateNormalTransactions(100, 10, 5)
		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Nil(t, outliers, "Should return nil for insufficient data")
	})

	t.Run("no outliers in normal range", func(t *testing.T) {
		// All transactions within IQR bounds
		transactions := make([]models.Transaction, 0, 20)
		for i := 0; i < 20; i++ {
			amount := 100 + i // 100 to 119
			transactions = append(transactions, createTransaction(
				generateTxHash(i),
				"A", "B",
				decimal.NewFromInt(int64(amount)).String(),
				time.Now(),
			))
		}

		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Empty(t, outliers, "Should not detect outliers in normal range")
	})
}

func TestIQRDetector_DetectByAddress(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := detection.IQRConfig{
		Multiplier:     1.5,
		WindowDuration: 24 * time.Hour,
		MinDataPoints:  5,
	}
	detector := detection.NewIQRDetector(config, logger)

	transactions := []models.Transaction{
		createTransaction("tx1", "AddrA", "AddrB", "100", time.Now()),
		createTransaction("tx2", "AddrA", "AddrC", "105", time.Now()),
		createTransaction("tx3", "AddrA", "AddrD", "110", time.Now()),
		createTransaction("tx4", "AddrA", "AddrE", "115", time.Now()),
		createTransaction("tx5", "AddrB", "AddrC", "200", time.Now()), // Different sender
		createTransaction("tx6", "AddrA", "AddrF", "1000", time.Now()), // Outlier
	}

	outliers, err := detector.DetectByAddress("AddrA", transactions)
	require.NoError(t, err)

	// Should detect tx6 as outlier for AddrA
	found := false
	for _, o := range outliers {
		if o.TransactionHash == "tx6" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should detect outlier for specific address")
}

func BenchmarkIQRDetector_Detect(b *testing.B) {
	logger := zap.NewNop()
	config := detection.IQRConfig{
		Multiplier:     1.5,
		WindowDuration: 24 * time.Hour,
		MinDataPoints:  10,
	}
	detector := detection.NewIQRDetector(config, logger)

	// Generate test data
	transactions := generateNormalTransactions(100, 10, 100)
	transactions = append(transactions, createTransaction("outlier", "A", "B", "500", time.Now()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(transactions)
	}
}

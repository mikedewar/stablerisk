package detection_test

import (
	"testing"
	"time"

	"github.com/mikedewar/stablerisk/internal/detection"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestZScoreDetector_Detect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := detection.ZScoreConfig{
		Threshold:      3.0,
		WindowDuration: 24 * time.Hour,
		MinDataPoints:  10,
	}
	detector := detection.NewZScoreDetector(config, logger)

	t.Run("normal distribution with outlier", func(t *testing.T) {
		// Create transactions with narrow range: 95-105
		transactions := make([]models.Transaction, 0, 30)
		for i := 0; i < 30; i++ {
			amount := 100.0 + float64(i%10-5) // 95-105
			transactions = append(transactions, createTransaction(
				generateTxHash(i),
				"A", "B",
				decimal.NewFromFloat(amount).String(),
				time.Now(),
			))
		}

		// Add clear outlier at 500 (way beyond 3σ)
		outlierTx := createTransaction("outlier", "A", "B", "500", time.Now())
		transactions = append(transactions, outlierTx)

		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Greater(t, len(outliers), 0, "Should detect outlier")

		// Check that outlier was detected
		found := false
		for _, o := range outliers {
			if o.TransactionHash == "outlier" {
				found = true
				assert.Equal(t, models.OutlierTypeZScore, o.Type)
				assert.NotEmpty(t, o.Severity)
				break
			}
		}
		assert.True(t, found, "Outlier transaction should be detected")
	})

	t.Run("insufficient data points", func(t *testing.T) {
		transactions := generateNormalTransactions(100, 10, 5) // Only 5 transactions
		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Nil(t, outliers, "Should return nil for insufficient data")
	})

	t.Run("all identical values", func(t *testing.T) {
		transactions := generateIdenticalTransactions("100", 20)
		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.Empty(t, outliers, "Should not detect outliers when stddev=0")
	})

	t.Run("multiple outliers with different severities", func(t *testing.T) {
		// Create very tight distribution (100-102) with MORE normal transactions
		// so that outliers don't skew the statistics as much
		transactions := make([]models.Transaction, 100)
		for i := 0; i < 100; i++ {
			amount := 100.0 + float64(i%3) // Creates range 100-102 (stddev ~0.8)
			transactions[i] = createTransaction(
				generateTxHash(i),
				"AddrA",
				"AddrB",
				decimal.NewFromFloat(amount).String(),
				time.Now().Add(time.Duration(i)*time.Minute),
			)
		}

		// Add clear outliers that are far from the mean
		// With 100 normal transactions, outliers won't skew stats much
		// Expected: mean~101, stddev~0.8
		// outlier1: z = (115-101)/0.8 ≈ 17.5σ (critical)
		// outlier2: z = (130-101)/0.8 ≈ 36σ (critical)
		transactions = append(transactions, createTransaction("out1", "A", "B", "115", time.Now()))
		transactions = append(transactions, createTransaction("out2", "A", "B", "130", time.Now()))

		outliers, err := detector.Detect(transactions)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(outliers), 2, "Should detect multiple outliers")
	})
}

func TestZScoreDetector_DetectByAddress(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := detection.ZScoreConfig{
		Threshold:      3.0,
		WindowDuration: 24 * time.Hour,
		MinDataPoints:  5,
	}
	detector := detection.NewZScoreDetector(config, logger)

	// Create tight distribution of normal transactions for AddrA
	// Use 100 transactions so outlier doesn't skew statistics
	transactions := make([]models.Transaction, 0, 102)
	for i := 0; i < 100; i++ {
		amount := 95.0 + float64(i%10) // Creates range 95-104
		transactions = append(transactions, createTransaction(
			generateTxHash(i),
			"AddrA",
			"AddrB",
			decimal.NewFromFloat(amount).String(),
			time.Now(),
		))
	}

	// Add some transactions from other addresses (should be filtered out)
	transactions = append(transactions, createTransaction("txOther", "AddrB", "AddrC", "200", time.Now()))

	// Add clear outlier for AddrA
	// Expected: mean~99.5, stddev~3
	// z-score: (1000-99.5)/3 ≈ 300σ (clearly critical)
	transactions = append(transactions, createTransaction("txOutlier", "AddrA", "AddrE", "1000", time.Now()))

	outliers, err := detector.DetectByAddress("AddrA", transactions)
	require.NoError(t, err)

	// Should detect the 1000 USDT transaction as outlier
	found := false
	for _, o := range outliers {
		if o.TransactionHash == "txOutlier" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should detect outlier for specific address")
}

func TestCalculateStatistics(t *testing.T) {
	transactions := []models.Transaction{
		createTransaction("tx1", "A", "B", "100", time.Now()),
		createTransaction("tx2", "A", "B", "110", time.Now()),
		createTransaction("tx3", "A", "B", "120", time.Now()),
		createTransaction("tx4", "A", "B", "130", time.Now()),
		createTransaction("tx5", "A", "B", "140", time.Now()),
	}

	stats, err := detection.CalculateStatistics(transactions)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 5, stats.Count)
	assert.InDelta(t, 120.0, stats.Mean, 0.1)
	assert.InDelta(t, 100.0, stats.Min, 0.1)
	assert.InDelta(t, 140.0, stats.Max, 0.1)
	assert.Greater(t, stats.StdDev, 0.0)
}

// Helper functions

func generateNormalTransactions(mean, stddev float64, count int) []models.Transaction {
	transactions := make([]models.Transaction, count)
	for i := 0; i < count; i++ {
		// Simple approximation of normal distribution
		amount := mean + (stddev * float64(i%5-2))
		transactions[i] = createTransaction(
			generateTxHash(i),
			"AddrA",
			"AddrB",
			decimal.NewFromFloat(amount).String(),
			time.Now().Add(time.Duration(i)*time.Minute),
		)
	}
	return transactions
}

func generateIdenticalTransactions(amount string, count int) []models.Transaction {
	transactions := make([]models.Transaction, count)
	for i := 0; i < count; i++ {
		transactions[i] = createTransaction(
			generateTxHash(i),
			"AddrA",
			"AddrB",
			amount,
			time.Now().Add(time.Duration(i)*time.Minute),
		)
	}
	return transactions
}

func createTransaction(txHash, from, to, amount string, timestamp time.Time) models.Transaction {
	amt, _ := decimal.NewFromString(amount)
	return models.Transaction{
		TxHash:      txHash,
		From:        from,
		To:          to,
		Amount:      amt,
		Timestamp:   timestamp,
		BlockNumber: 12345,
		Contract:    "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		Confirmed:   true,
	}
}

func generateTxHash(i int) string {
	return "0x" + string(rune('a'+i%26)) + string(rune('0'+i%10))
}

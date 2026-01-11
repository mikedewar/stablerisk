package detection

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
)

// ZScoreDetector detects outliers using Z-score (standard score) method
type ZScoreDetector struct {
	threshold      float64       // Z-score threshold (typically 3.0)
	windowDuration time.Duration // Time window for calculating statistics
	minDataPoints  int           // Minimum data points required
	logger         *zap.Logger
}

// ZScoreConfig holds configuration for Z-score detector
type ZScoreConfig struct {
	Threshold      float64
	WindowDuration time.Duration
	MinDataPoints  int
}

// NewZScoreDetector creates a new Z-score detector
func NewZScoreDetector(config ZScoreConfig, logger *zap.Logger) *ZScoreDetector {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ZScoreDetector{
		threshold:      config.Threshold,
		windowDuration: config.WindowDuration,
		minDataPoints:  config.MinDataPoints,
		logger:         logger,
	}
}

// Detect finds outliers using Z-score method
func (d *ZScoreDetector) Detect(transactions []models.Transaction) ([]models.Outlier, error) {
	if len(transactions) < d.minDataPoints {
		d.logger.Debug("Insufficient data points for Z-score detection",
			zap.Int("count", len(transactions)),
			zap.Int("min_required", d.minDataPoints))
		return nil, nil
	}

	// Extract amounts as float64 array
	amounts := make([]float64, len(transactions))
	for i, tx := range transactions {
		amt, _ := tx.Amount.Float64()
		amounts[i] = amt
	}

	// Calculate statistics
	mean := stat.Mean(amounts, nil)
	stddev := stat.StdDev(amounts, nil)

	d.logger.Debug("Z-score statistics calculated",
		zap.Float64("mean", mean),
		zap.Float64("stddev", stddev),
		zap.Int("sample_size", len(amounts)))

	// If stddev is 0, all values are the same - no outliers
	if stddev == 0 {
		d.logger.Debug("Standard deviation is zero, no outliers detected")
		return nil, nil
	}

	// Find outliers
	var outliers []models.Outlier
	for i, tx := range transactions {
		amount := amounts[i]
		zScore := (amount - mean) / stddev

		if math.Abs(zScore) > d.threshold {
			severity := d.calculateSeverity(math.Abs(zScore))

			outlier := models.Outlier{
				ID:              uuid.New().String(),
				DetectedAt:      time.Now(),
				Type:            models.OutlierTypeZScore,
				Severity:        severity,
				Address:         tx.From, // Sender as primary address
				TransactionHash: tx.TxHash,
				Amount:          tx.Amount,
				ZScore:          zScore,
				Details: map[string]interface{}{
					"z_score":       zScore,
					"mean":          mean,
					"stddev":        stddev,
					"sample_size":   len(amounts),
					"from":          tx.From,
					"to":            tx.To,
					"block_number":  tx.BlockNumber,
					"timestamp":     tx.Timestamp,
					"threshold":     d.threshold,
				},
				Acknowledged: false,
			}

			outliers = append(outliers, outlier)

			d.logger.Info("Z-score outlier detected",
				zap.String("tx_hash", tx.TxHash),
				zap.Float64("z_score", zScore),
				zap.Float64("amount", amount),
				zap.String("severity", string(severity)))
		}
	}

	d.logger.Info("Z-score detection completed",
		zap.Int("total_transactions", len(transactions)),
		zap.Int("outliers_found", len(outliers)))

	return outliers, nil
}

// DetectByAddress detects outliers for a specific address
func (d *ZScoreDetector) DetectByAddress(address string, transactions []models.Transaction) ([]models.Outlier, error) {
	// Filter transactions involving this address
	var filtered []models.Transaction
	for _, tx := range transactions {
		if tx.From == address || tx.To == address {
			filtered = append(filtered, tx)
		}
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	d.logger.Debug("Detecting outliers for address",
		zap.String("address", address),
		zap.Int("transaction_count", len(filtered)))

	return d.Detect(filtered)
}

// calculateSeverity determines severity based on Z-score magnitude
func (d *ZScoreDetector) calculateSeverity(absZScore float64) models.Severity {
	// Severity thresholds based on standard deviations:
	// 3σ = low (99.7% confidence)
	// 4σ = medium (99.99% confidence)
	// 5σ = high (99.9999% confidence)
	// 6σ+ = critical (extremely rare)

	switch {
	case absZScore >= 6.0:
		return models.SeverityCritical
	case absZScore >= 5.0:
		return models.SeverityHigh
	case absZScore >= 4.0:
		return models.SeverityMedium
	default:
		return models.SeverityLow
	}
}

// CalculateStatistics calculates statistical data for a set of transactions
func CalculateStatistics(transactions []models.Transaction) (*models.StatisticalData, error) {
	if len(transactions) == 0 {
		return nil, fmt.Errorf("no transactions provided")
	}

	// Extract amounts
	amounts := make([]float64, len(transactions))
	for i, tx := range transactions {
		amt, _ := tx.Amount.Float64()
		amounts[i] = amt
	}

	// Calculate mean and standard deviation
	mean := stat.Mean(amounts, nil)
	stddev := stat.StdDev(amounts, nil)

	// Sort for quantile calculations
	sortedAmounts := make([]float64, len(amounts))
	copy(sortedAmounts, amounts)
	sort.Float64s(sortedAmounts)

	// Calculate quantiles for IQR
	q1 := stat.Quantile(0.25, stat.Empirical, sortedAmounts, nil)
	q2 := stat.Quantile(0.50, stat.Empirical, sortedAmounts, nil)
	q3 := stat.Quantile(0.75, stat.Empirical, sortedAmounts, nil)
	iqr := q3 - q1

	// Find min and max
	min := amounts[0]
	max := amounts[0]
	for _, v := range amounts {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return &models.StatisticalData{
		Values: amounts,
		Mean:   mean,
		StdDev: stddev,
		Q1:     q1,
		Q2:     q2,
		Q3:     q3,
		IQR:    iqr,
		Min:    min,
		Max:    max,
		Count:  len(amounts),
	}, nil
}

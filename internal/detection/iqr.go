package detection

import (
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
)

// IQRDetector detects outliers using Interquartile Range (IQR) method
type IQRDetector struct {
	multiplier     float64       // IQR multiplier (typically 1.5)
	windowDuration time.Duration // Time window for calculating statistics
	minDataPoints  int           // Minimum data points required
	logger         *zap.Logger
}

// IQRConfig holds configuration for IQR detector
type IQRConfig struct {
	Multiplier     float64
	WindowDuration time.Duration
	MinDataPoints  int
}

// NewIQRDetector creates a new IQR detector
func NewIQRDetector(config IQRConfig, logger *zap.Logger) *IQRDetector {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &IQRDetector{
		multiplier:     config.Multiplier,
		windowDuration: config.WindowDuration,
		minDataPoints:  config.MinDataPoints,
		logger:         logger,
	}
}

// Detect finds outliers using IQR method
func (d *IQRDetector) Detect(transactions []models.Transaction) ([]models.Outlier, error) {
	if len(transactions) < d.minDataPoints {
		d.logger.Debug("Insufficient data points for IQR detection",
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

	// Sort amounts (required by gonum.stat.Quantile)
	sort.Float64s(amounts)

	// Calculate quartiles
	q1 := stat.Quantile(0.25, stat.Empirical, amounts, nil)
	q3 := stat.Quantile(0.75, stat.Empirical, amounts, nil)
	iqr := q3 - q1

	// Calculate bounds
	lowerBound := q1 - (d.multiplier * iqr)
	upperBound := q3 + (d.multiplier * iqr)

	d.logger.Debug("IQR statistics calculated",
		zap.Float64("q1", q1),
		zap.Float64("q3", q3),
		zap.Float64("iqr", iqr),
		zap.Float64("lower_bound", lowerBound),
		zap.Float64("upper_bound", upperBound),
		zap.Int("sample_size", len(amounts)))

	// Find outliers
	var outliers []models.Outlier
	for i, tx := range transactions {
		amount := amounts[i]

		// Check if outside bounds
		if amount < lowerBound || amount > upperBound {
			// Calculate severity based on how far outside bounds
			deviation := d.calculateDeviation(amount, lowerBound, upperBound, iqr)
			severity := d.calculateSeverity(deviation)

			outlier := models.Outlier{
				ID:              uuid.New().String(),
				DetectedAt:      time.Now(),
				Type:            models.OutlierTypeIQR,
				Severity:        severity,
				Address:         tx.From,
				TransactionHash: tx.TxHash,
				Amount:          tx.Amount,
				Details: map[string]interface{}{
					"q1":            q1,
					"q3":            q3,
					"iqr":           iqr,
					"lower_bound":   lowerBound,
					"upper_bound":   upperBound,
					"deviation":     deviation,
					"sample_size":   len(amounts),
					"from":          tx.From,
					"to":            tx.To,
					"block_number":  tx.BlockNumber,
					"timestamp":     tx.Timestamp,
					"multiplier":    d.multiplier,
					"amount":        amount,
				},
				Acknowledged: false,
			}

			outliers = append(outliers, outlier)

			d.logger.Info("IQR outlier detected",
				zap.String("tx_hash", tx.TxHash),
				zap.Float64("amount", amount),
				zap.Float64("lower_bound", lowerBound),
				zap.Float64("upper_bound", upperBound),
				zap.Float64("deviation", deviation),
				zap.String("severity", string(severity)))
		}
	}

	d.logger.Info("IQR detection completed",
		zap.Int("total_transactions", len(transactions)),
		zap.Int("outliers_found", len(outliers)))

	return outliers, nil
}

// DetectByAddress detects outliers for a specific address
func (d *IQRDetector) DetectByAddress(address string, transactions []models.Transaction) ([]models.Outlier, error) {
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

// calculateDeviation calculates how many IQRs the value is from the bounds
func (d *IQRDetector) calculateDeviation(value, lowerBound, upperBound, iqr float64) float64 {
	if iqr == 0 {
		return 0
	}

	if value < lowerBound {
		return math.Abs((lowerBound - value) / iqr)
	}

	if value > upperBound {
		return math.Abs((value - upperBound) / iqr)
	}

	return 0
}

// calculateSeverity determines severity based on IQR deviation
func (d *IQRDetector) calculateSeverity(deviation float64) models.Severity {
	// Severity thresholds based on IQR deviations:
	// 1.5 IQR = low (standard outlier)
	// 3.0 IQR = medium (far outlier)
	// 5.0 IQR = high (extreme outlier)
	// 10+ IQR = critical (anomalous)

	switch {
	case deviation >= 10.0:
		return models.SeverityCritical
	case deviation >= 5.0:
		return models.SeverityHigh
	case deviation >= 3.0:
		return models.SeverityMedium
	default:
		return models.SeverityLow
	}
}

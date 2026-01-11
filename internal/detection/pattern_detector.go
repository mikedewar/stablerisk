package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mikedewar/stablerisk/internal/graph"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// PatternDetector detects graph-based transaction patterns
type PatternDetector struct {
	raphtoryClient       *graph.RaphtoryClient
	logger               *zap.Logger
	circulationWindow    time.Duration // Time window for detecting circulation
	fanOutThreshold      int           // Number of recipients for fan-out
	fanInThreshold       int           // Number of senders for fan-in
	dormancyPeriod       time.Duration // Period of inactivity before dormant
	velocityWindow       time.Duration // Time window for velocity calculation
	velocityThreshold    int           // Number of transactions in window
}

// PatternDetectorConfig holds configuration for pattern detector
type PatternDetectorConfig struct {
	CirculationWindow time.Duration
	FanOutThreshold   int
	FanInThreshold    int
	DormancyPeriod    time.Duration
	VelocityWindow    time.Duration
	VelocityThreshold int
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(config PatternDetectorConfig, raphtoryClient *graph.RaphtoryClient, logger *zap.Logger) *PatternDetector {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &PatternDetector{
		raphtoryClient:    raphtoryClient,
		logger:            logger,
		circulationWindow: config.CirculationWindow,
		fanOutThreshold:   config.FanOutThreshold,
		fanInThreshold:    config.FanInThreshold,
		dormancyPeriod:    config.DormancyPeriod,
		velocityWindow:    config.VelocityWindow,
		velocityThreshold: config.VelocityThreshold,
	}
}

// DetectAll runs all pattern detection algorithms
func (d *PatternDetector) DetectAll(ctx context.Context) ([]models.Outlier, error) {
	var allOutliers []models.Outlier

	// Detect circulation patterns
	circulation, err := d.DetectCirculation(ctx)
	if err != nil {
		d.logger.Error("Failed to detect circulation patterns", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, circulation...)
	}

	// Detect fan-out patterns
	fanOut, err := d.DetectFanOut(ctx)
	if err != nil {
		d.logger.Error("Failed to detect fan-out patterns", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, fanOut...)
	}

	// Detect fan-in patterns
	fanIn, err := d.DetectFanIn(ctx)
	if err != nil {
		d.logger.Error("Failed to detect fan-in patterns", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, fanIn...)
	}

	// Detect velocity patterns
	velocity, err := d.DetectVelocity(ctx)
	if err != nil {
		d.logger.Error("Failed to detect velocity patterns", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, velocity...)
	}

	d.logger.Info("Pattern detection completed",
		zap.Int("total_outliers", len(allOutliers)))

	return allOutliers, nil
}

// DetectCirculation detects circular transaction patterns (A → B → C → A)
func (d *PatternDetector) DetectCirculation(ctx context.Context) ([]models.Outlier, error) {
	d.logger.Debug("Detecting circulation patterns")

	// This would require implementing path detection in Raphtory client
	// For now, return placeholder

	// TODO: Query Raphtory for paths that form cycles within circulationWindow
	// Example: Find addresses where money flows back to origin within short time

	return nil, nil
}

// DetectFanOut detects fan-out patterns (one sender → many receivers)
func (d *PatternDetector) DetectFanOut(ctx context.Context) ([]models.Outlier, error) {
	d.logger.Debug("Detecting fan-out patterns",
		zap.Int("threshold", d.fanOutThreshold))

	// This would query Raphtory for nodes with high out-degree
	// indicating potential fund distribution

	// Placeholder implementation
	// TODO: Query graph for nodes with out-degree > fanOutThreshold
	// within a time window

	return nil, nil
}

// DetectFanIn detects fan-in patterns (many senders → one receiver)
func (d *PatternDetector) DetectFanIn(ctx context.Context) ([]models.Outlier, error) {
	d.logger.Debug("Detecting fan-in patterns",
		zap.Int("threshold", d.fanInThreshold))

	// This would query Raphtory for nodes with high in-degree
	// indicating potential fund collection

	// Placeholder implementation
	// TODO: Query graph for nodes with in-degree > fanInThreshold
	// within a time window

	return nil, nil
}

// DetectDormantAwakening detects dormant addresses that suddenly become active
func (d *PatternDetector) DetectDormantAwakening(ctx context.Context, address string) (*models.Outlier, error) {
	// Get node info from Raphtory
	nodeInfo, err := d.raphtoryClient.GetNodeInfo(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	if nodeInfo == nil {
		return nil, nil
	}

	// Check if address was dormant
	// Calculate time since first_seen and last_seen
	now := time.Now()
	firstSeen := time.Unix(nodeInfo.FirstSeen, 0)
	lastSeen := time.Unix(nodeInfo.LastSeen, 0)

	dormancyDuration := now.Sub(lastSeen)

	// If dormant for longer than threshold and recently active
	if dormancyDuration > d.dormancyPeriod && now.Sub(lastSeen) < time.Hour {
		outlier := models.Outlier{
			ID:         uuid.New().String(),
			DetectedAt: time.Now(),
			Type:       models.OutlierTypePatternDormant,
			Severity:   d.calculateDormantSeverity(dormancyDuration),
			Address:    address,
			Details: map[string]interface{}{
				"first_seen":        firstSeen,
				"last_seen":         lastSeen,
				"dormancy_duration": dormancyDuration.Hours(),
				"transaction_count": nodeInfo.TransactionCount,
				"pattern":           "dormant_awakening",
			},
			Acknowledged: false,
		}

		d.logger.Info("Dormant awakening detected",
			zap.String("address", address),
			zap.Duration("dormancy", dormancyDuration))

		return &outlier, nil
	}

	return nil, nil
}

// DetectVelocity detects high transaction velocity (many transactions in short time)
func (d *PatternDetector) DetectVelocity(ctx context.Context) ([]models.Outlier, error) {
	d.logger.Debug("Detecting velocity patterns",
		zap.Duration("window", d.velocityWindow),
		zap.Int("threshold", d.velocityThreshold))

	// Query recent transactions from Raphtory
	endTime := time.Now().Unix()
	startTime := time.Now().Add(-d.velocityWindow).Unix()

	transactions, err := d.raphtoryClient.GetTransactionsInWindow(ctx, startTime, endTime, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Group transactions by address
	addressTxCounts := make(map[string]int)
	addressFirstTx := make(map[string]models.Transaction)

	for _, tx := range transactions {
		addressTxCounts[tx.From]++
		addressTxCounts[tx.To]++

		if _, exists := addressFirstTx[tx.From]; !exists {
			addressFirstTx[tx.From] = tx
		}
		if _, exists := addressFirstTx[tx.To]; !exists {
			addressFirstTx[tx.To] = tx
		}
	}

	// Detect addresses with high velocity
	var outliers []models.Outlier
	for address, count := range addressTxCounts {
		if count > d.velocityThreshold {
			tx := addressFirstTx[address]
			severity := d.calculateVelocitySeverity(count, d.velocityThreshold)

			outlier := models.Outlier{
				ID:              uuid.New().String(),
				DetectedAt:      time.Now(),
				Type:            models.OutlierTypePatternVelocity,
				Severity:        severity,
				Address:         address,
				TransactionHash: tx.TxHash,
				Details: map[string]interface{}{
					"transaction_count": count,
					"time_window":       d.velocityWindow.String(),
					"threshold":         d.velocityThreshold,
					"velocity":          float64(count) / d.velocityWindow.Hours(),
					"pattern":           "high_velocity",
				},
				Acknowledged: false,
			}

			outliers = append(outliers, outlier)

			d.logger.Info("High velocity detected",
				zap.String("address", address),
				zap.Int("transaction_count", count),
				zap.Duration("window", d.velocityWindow))
		}
	}

	return outliers, nil
}

// calculateDormantSeverity calculates severity for dormant awakening
func (d *PatternDetector) calculateDormantSeverity(dormancy time.Duration) models.Severity {
	days := dormancy.Hours() / 24

	switch {
	case days >= 365: // 1+ year
		return models.SeverityCritical
	case days >= 180: // 6+ months
		return models.SeverityHigh
	case days >= 90: // 3+ months
		return models.SeverityMedium
	default:
		return models.SeverityLow
	}
}

// calculateVelocitySeverity calculates severity for high velocity
func (d *PatternDetector) calculateVelocitySeverity(count, threshold int) models.Severity {
	ratio := float64(count) / float64(threshold)

	switch {
	case ratio >= 10.0:
		return models.SeverityCritical
	case ratio >= 5.0:
		return models.SeverityHigh
	case ratio >= 2.0:
		return models.SeverityMedium
	default:
		return models.SeverityLow
	}
}

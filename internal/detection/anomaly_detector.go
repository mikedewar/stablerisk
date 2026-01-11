package detection

import (
	"context"
	"sync"
	"time"

	"github.com/mikedewar/stablerisk/internal/graph"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// AnomalyDetector coordinates all anomaly detection methods
type AnomalyDetector struct {
	zscoreDetector  *ZScoreDetector
	iqrDetector     *IQRDetector
	patternDetector *PatternDetector
	raphtoryClient  *graph.RaphtoryClient
	logger          *zap.Logger

	interval time.Duration
	running  bool
	stopChan chan struct{}
	mu       sync.RWMutex

	// Channels
	outlierChan chan models.Outlier
}

// AnomalyDetectorConfig holds configuration for anomaly detector
type AnomalyDetectorConfig struct {
	Interval              time.Duration
	ZScoreConfig          ZScoreConfig
	IQRConfig             IQRConfig
	PatternDetectorConfig PatternDetectorConfig
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(config AnomalyDetectorConfig, raphtoryClient *graph.RaphtoryClient, logger *zap.Logger) *AnomalyDetector {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AnomalyDetector{
		zscoreDetector:  NewZScoreDetector(config.ZScoreConfig, logger),
		iqrDetector:     NewIQRDetector(config.IQRConfig, logger),
		patternDetector: NewPatternDetector(config.PatternDetectorConfig, raphtoryClient, logger),
		raphtoryClient:  raphtoryClient,
		logger:          logger,
		interval:        config.Interval,
		running:         false,
		stopChan:        make(chan struct{}),
		outlierChan:     make(chan models.Outlier, 100),
	}
}

// Start starts the anomaly detection loop
func (d *AnomalyDetector) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	d.mu.Unlock()

	d.logger.Info("Starting anomaly detector",
		zap.Duration("interval", d.interval))

	go d.detectionLoop(ctx)

	return nil
}

// Stop stops the anomaly detection loop
func (d *AnomalyDetector) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return
	}

	d.logger.Info("Stopping anomaly detector")
	close(d.stopChan)
	d.running = false
}

// IsRunning returns whether the detector is running
func (d *AnomalyDetector) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// Outliers returns the outlier channel
func (d *AnomalyDetector) Outliers() <-chan models.Outlier {
	return d.outlierChan
}

// detectionLoop runs detection periodically
func (d *AnomalyDetector) detectionLoop(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Run detection immediately on start
	d.runDetection(ctx)

	for {
		select {
		case <-ticker.C:
			d.runDetection(ctx)
		case <-d.stopChan:
			d.logger.Info("Detection loop stopped")
			return
		case <-ctx.Done():
			d.logger.Info("Detection loop cancelled")
			return
		}
	}
}

// runDetection executes all detection methods
func (d *AnomalyDetector) runDetection(ctx context.Context) {
	d.logger.Info("Running anomaly detection cycle")
	startTime := time.Now()

	// Get recent transactions from Raphtory
	endTime := time.Now().Unix()
	startTimeQuery := time.Now().Add(-d.interval * 2).Unix() // Look back 2 intervals

	transactions, err := d.raphtoryClient.GetTransactionsInWindow(ctx, startTimeQuery, endTime, 10000)
	if err != nil {
		d.logger.Error("Failed to get transactions from Raphtory", zap.Error(err))
		return
	}

	if len(transactions) == 0 {
		d.logger.Debug("No transactions in window, skipping detection")
		return
	}

	d.logger.Info("Retrieved transactions for analysis",
		zap.Int("count", len(transactions)))

	var allOutliers []models.Outlier
	var wg sync.WaitGroup
	outliersLock := sync.Mutex{}

	// Run Z-score detection
	wg.Add(1)
	go func() {
		defer wg.Done()
		outliers, err := d.zscoreDetector.Detect(transactions)
		if err != nil {
			d.logger.Error("Z-score detection failed", zap.Error(err))
			return
		}
		outliersLock.Lock()
		allOutliers = append(allOutliers, outliers...)
		outliersLock.Unlock()
	}()

	// Run IQR detection
	wg.Add(1)
	go func() {
		defer wg.Done()
		outliers, err := d.iqrDetector.Detect(transactions)
		if err != nil {
			d.logger.Error("IQR detection failed", zap.Error(err))
			return
		}
		outliersLock.Lock()
		allOutliers = append(allOutliers, outliers...)
		outliersLock.Unlock()
	}()

	// Run pattern detection
	wg.Add(1)
	go func() {
		defer wg.Done()
		outliers, err := d.patternDetector.DetectAll(ctx)
		if err != nil {
			d.logger.Error("Pattern detection failed", zap.Error(err))
			return
		}
		outliersLock.Lock()
		allOutliers = append(allOutliers, outliers...)
		outliersLock.Unlock()
	}()

	// Wait for all detections to complete
	wg.Wait()

	// Deduplicate outliers (same transaction detected by multiple methods)
	deduped := d.deduplicateOutliers(allOutliers)

	// Publish outliers
	d.publishOutliers(deduped)

	duration := time.Since(startTime)
	d.logger.Info("Detection cycle completed",
		zap.Int("transactions_analyzed", len(transactions)),
		zap.Int("outliers_found", len(deduped)),
		zap.Duration("duration", duration))
}

// deduplicateOutliers removes duplicate outliers
func (d *AnomalyDetector) deduplicateOutliers(outliers []models.Outlier) []models.Outlier {
	// Use map to track unique outliers by transaction hash
	seen := make(map[string]*models.Outlier)

	for i := range outliers {
		outlier := &outliers[i]
		key := outlier.TransactionHash

		// If no transaction hash, use address
		if key == "" {
			key = outlier.Address
		}

		existing, exists := seen[key]
		if !exists {
			seen[key] = outlier
			continue
		}

		// If exists, keep the one with higher severity
		if d.compareSeverity(outlier.Severity, existing.Severity) > 0 {
			seen[key] = outlier
		}
	}

	// Convert map back to slice
	deduped := make([]models.Outlier, 0, len(seen))
	for _, outlier := range seen {
		deduped = append(deduped, *outlier)
	}

	if len(deduped) < len(outliers) {
		d.logger.Debug("Deduplicated outliers",
			zap.Int("original", len(outliers)),
			zap.Int("deduped", len(deduped)))
	}

	return deduped
}

// compareSeverity compares two severity levels
// Returns: >0 if s1 > s2, 0 if equal, <0 if s1 < s2
func (d *AnomalyDetector) compareSeverity(s1, s2 models.Severity) int {
	severityValue := map[models.Severity]int{
		models.SeverityLow:      1,
		models.SeverityMedium:   2,
		models.SeverityHigh:     3,
		models.SeverityCritical: 4,
	}

	return severityValue[s1] - severityValue[s2]
}

// publishOutliers sends outliers to the channel
func (d *AnomalyDetector) publishOutliers(outliers []models.Outlier) {
	for _, outlier := range outliers {
		select {
		case d.outlierChan <- outlier:
			d.logger.Debug("Outlier published",
				zap.String("id", outlier.ID),
				zap.String("type", string(outlier.Type)),
				zap.String("severity", string(outlier.Severity)))
		default:
			d.logger.Warn("Outlier channel full, dropping outlier",
				zap.String("id", outlier.ID))
		}
	}
}

// DetectOnce runs detection once and returns outliers
func (d *AnomalyDetector) DetectOnce(ctx context.Context) ([]models.Outlier, error) {
	// Get recent transactions
	endTime := time.Now().Unix()
	startTime := time.Now().Add(-24 * time.Hour).Unix()

	transactions, err := d.raphtoryClient.GetTransactionsInWindow(ctx, startTime, endTime, 10000)
	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return nil, nil
	}

	var allOutliers []models.Outlier

	// Run Z-score detection
	zscoreOutliers, err := d.zscoreDetector.Detect(transactions)
	if err != nil {
		d.logger.Error("Z-score detection failed", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, zscoreOutliers...)
	}

	// Run IQR detection
	iqrOutliers, err := d.iqrDetector.Detect(transactions)
	if err != nil {
		d.logger.Error("IQR detection failed", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, iqrOutliers...)
	}

	// Run pattern detection
	patternOutliers, err := d.patternDetector.DetectAll(ctx)
	if err != nil {
		d.logger.Error("Pattern detection failed", zap.Error(err))
	} else {
		allOutliers = append(allOutliers, patternOutliers...)
	}

	// Deduplicate
	return d.deduplicateOutliers(allOutliers), nil
}

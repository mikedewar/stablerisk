package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/api"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// StatisticsHandler handles statistics requests
type StatisticsHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewStatisticsHandler creates a new statistics handler
func NewStatisticsHandler(db *sql.DB, logger *zap.Logger) *StatisticsHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &StatisticsHandler{
		db:     db,
		logger: logger,
	}
}

// GetStatistics returns overall statistics
func (h *StatisticsHandler) GetStatistics(c *gin.Context) {
	stats := api.StatisticsResponse{
		OutliersBySeverity: make(map[models.Severity]int64),
		OutliersByType:     make(map[models.OutlierType]int64),
	}

	// Note: In a real implementation, we would query a transactions table
	// For now, we'll return placeholder values or query outliers

	// Total outliers
	err := h.db.QueryRow(`SELECT COUNT(*) FROM outliers`).Scan(&stats.TotalOutliers)
	if err != nil && err != sql.ErrNoRows {
		h.logger.Error("Failed to count outliers",
			zap.Error(err))
	}

	// Outliers by severity
	rows, err := h.db.Query(`
		SELECT severity, COUNT(*)
		FROM outliers
		GROUP BY severity
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var severity models.Severity
			var count int64
			if err := rows.Scan(&severity, &count); err == nil {
				stats.OutliersBySeverity[severity] = count
			}
		}
	}

	// Outliers by type
	rows, err = h.db.Query(`
		SELECT type, COUNT(*)
		FROM outliers
		GROUP BY type
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var outlierType models.OutlierType
			var count int64
			if err := rows.Scan(&outlierType, &count); err == nil {
				stats.OutliersByType[outlierType] = count
			}
		}
	}

	// Last detection run
	// This would typically come from a detection_runs table
	// For now, use the most recent outlier timestamp
	var lastDetection sql.NullTime
	err = h.db.QueryRow(`
		SELECT MAX(detected_at) FROM outliers
	`).Scan(&lastDetection)
	if err == nil && lastDetection.Valid {
		stats.LastDetectionRun = lastDetection.Time
	} else {
		stats.LastDetectionRun = time.Now()
	}

	// Detection running status
	// This would typically come from a monitoring system
	stats.DetectionRunning = true

	// Total transactions (placeholder - would come from transactions table)
	stats.TotalTransactions = 0

	c.JSON(http.StatusOK, stats)
}

// GetOutlierTrends returns outlier trends over time
func (h *StatisticsHandler) GetOutlierTrends(c *gin.Context) {
	// Query parameters for time range
	daysStr := c.DefaultQuery("days", "7")

	var days int
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil || days < 1 || days > 90 {
		days = 7
	}

	startTime := time.Now().AddDate(0, 0, -days)

	// Query outliers grouped by day
	rows, err := h.db.Query(`
		SELECT
			DATE(detected_at) as date,
			severity,
			COUNT(*) as count
		FROM outliers
		WHERE detected_at >= $1
		GROUP BY DATE(detected_at), severity
		ORDER BY date DESC
	`, startTime)

	if err != nil {
		h.logger.Error("Failed to query outlier trends",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch trends",
		})
		return
	}
	defer rows.Close()

	type DailyStats struct {
		Date     string                       `json:"date"`
		Severity map[models.Severity]int64    `json:"severity"`
	}

	statsMap := make(map[string]*DailyStats)

	for rows.Next() {
		var date time.Time
		var severity models.Severity
		var count int64

		if err := rows.Scan(&date, &severity, &count); err != nil {
			continue
		}

		dateStr := date.Format("2006-01-02")
		if _, ok := statsMap[dateStr]; !ok {
			statsMap[dateStr] = &DailyStats{
				Date:     dateStr,
				Severity: make(map[models.Severity]int64),
			}
		}
		statsMap[dateStr].Severity[severity] = count
	}

	// Convert map to slice
	trends := make([]DailyStats, 0, len(statsMap))
	for _, stats := range statsMap {
		trends = append(trends, *stats)
	}

	c.JSON(http.StatusOK, gin.H{
		"trends": trends,
		"period": gin.H{
			"start": startTime.Format(time.RFC3339),
			"end":   time.Now().Format(time.RFC3339),
			"days":  days,
		},
	})
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/api"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// OutlierHandler handles outlier-related requests
type OutlierHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewOutlierHandler creates a new outlier handler
func NewOutlierHandler(db *sql.DB, logger *zap.Logger) *OutlierHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OutlierHandler{
		db:     db,
		logger: logger,
	}
}

// ListOutliers returns a paginated list of outliers
func (h *OutlierHandler) ListOutliers(c *gin.Context) {
	var req api.OutlierListRequest

	// Set defaults
	req.Page = 1
	req.Limit = 50

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid query parameters",
		})
		return
	}

	// Validate pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 50
	}

	// Build query
	query := `
		SELECT id, detected_at, type, severity, address, transaction_hash,
		       amount, z_score, details, acknowledged, acknowledged_by, acknowledged_at, notes
		FROM outliers
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Apply filters
	if req.Type != "" {
		query += ` AND type = $` + string(rune('0'+argCount))
		args = append(args, req.Type)
		argCount++
	}

	if req.Severity != "" {
		query += ` AND severity = $` + string(rune('0'+argCount))
		args = append(args, req.Severity)
		argCount++
	}

	if req.Address != "" {
		query += ` AND address = $` + string(rune('0'+argCount))
		args = append(args, req.Address)
		argCount++
	}

	if req.Acknowledged != nil {
		query += ` AND acknowledged = $` + string(rune('0'+argCount))
		args = append(args, *req.Acknowledged)
		argCount++
	}

	if req.FromTimestamp != nil {
		query += ` AND detected_at >= $` + string(rune('0'+argCount))
		args = append(args, *req.FromTimestamp)
		argCount++
	}

	if req.ToTimestamp != nil {
		query += ` AND detected_at <= $` + string(rune('0'+argCount))
		args = append(args, *req.ToTimestamp)
		argCount++
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM (` + query + `) AS filtered`
	var total int
	err := h.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		h.logger.Error("Failed to count outliers",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch outliers",
		})
		return
	}

	// Add ordering and pagination
	query += ` ORDER BY detected_at DESC LIMIT $` + string(rune('0'+argCount)) + ` OFFSET $` + string(rune('0'+argCount+1))
	args = append(args, req.Limit, (req.Page-1)*req.Limit)

	// Query outliers
	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.logger.Error("Failed to query outliers",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch outliers",
		})
		return
	}
	defer rows.Close()

	outliers := []models.Outlier{}
	for rows.Next() {
		var outlier models.Outlier
		var amountStr string
		var detailsJSON []byte
		var acknowledgedBy, notes sql.NullString
		var acknowledgedAt sql.NullTime
		var zScore sql.NullFloat64

		err := rows.Scan(
			&outlier.ID,
			&outlier.DetectedAt,
			&outlier.Type,
			&outlier.Severity,
			&outlier.Address,
			&outlier.TransactionHash,
			&amountStr,
			&zScore,
			&detailsJSON,
			&outlier.Acknowledged,
			&acknowledgedBy,
			&acknowledgedAt,
			&notes,
		)
		if err != nil {
			h.logger.Error("Failed to scan outlier row",
				zap.Error(err))
			continue
		}

		// Parse amount
		outlier.Amount, _ = decimal.NewFromString(amountStr)

		// Parse z-score
		if zScore.Valid {
			outlier.ZScore = zScore.Float64
		}

		// Parse details
		if err := json.Unmarshal(detailsJSON, &outlier.Details); err != nil {
			h.logger.Error("Failed to unmarshal outlier details",
				zap.Error(err))
		}

		// Parse nullable fields
		if acknowledgedBy.Valid {
			outlier.AcknowledgedBy = acknowledgedBy.String
		}
		if acknowledgedAt.Valid {
			outlier.AcknowledgedAt = acknowledgedAt.Time
		}
		if notes.Valid {
			outlier.Notes = notes.String
		}

		outliers = append(outliers, outlier)
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))

	c.JSON(http.StatusOK, api.OutlierListResponse{
		Outliers:   outliers,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	})
}

// GetOutlier returns a single outlier by ID
func (h *OutlierHandler) GetOutlier(c *gin.Context) {
	id := c.Param("id")

	var outlier models.Outlier
	var amountStr string
	var detailsJSON []byte
	var acknowledgedBy, notes sql.NullString
	var acknowledgedAt sql.NullTime
	var zScore sql.NullFloat64

	err := h.db.QueryRow(`
		SELECT id, detected_at, type, severity, address, transaction_hash,
		       amount, z_score, details, acknowledged, acknowledged_by, acknowledged_at, notes
		FROM outliers
		WHERE id = $1
	`, id).Scan(
		&outlier.ID,
		&outlier.DetectedAt,
		&outlier.Type,
		&outlier.Severity,
		&outlier.Address,
		&outlier.TransactionHash,
		&amountStr,
		&zScore,
		&detailsJSON,
		&outlier.Acknowledged,
		&acknowledgedBy,
		&acknowledgedAt,
		&notes,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Outlier not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to query outlier",
			zap.Error(err),
			zap.String("outlier_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch outlier",
		})
		return
	}

	// Parse amount
	outlier.Amount, _ = decimal.NewFromString(amountStr)

	// Parse z-score
	if zScore.Valid {
		outlier.ZScore = zScore.Float64
	}

	// Parse details
	if err := json.Unmarshal(detailsJSON, &outlier.Details); err != nil {
		h.logger.Error("Failed to unmarshal outlier details",
			zap.Error(err))
	}

	// Parse nullable fields
	if acknowledgedBy.Valid {
		outlier.AcknowledgedBy = acknowledgedBy.String
	}
	if acknowledgedAt.Valid {
		outlier.AcknowledgedAt = acknowledgedAt.Time
	}
	if notes.Valid {
		outlier.Notes = notes.String
	}

	c.JSON(http.StatusOK, outlier)
}

// AcknowledgeOutlier marks an outlier as acknowledged
func (h *OutlierHandler) AcknowledgeOutlier(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var req api.AcknowledgeOutlierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
		})
		return
	}

	// Update outlier
	result, err := h.db.Exec(`
		UPDATE outliers
		SET acknowledged = true,
		    acknowledged_by = $1,
		    acknowledged_at = $2,
		    notes = $3
		WHERE id = $4
	`, userID, time.Now(), req.Notes, id)

	if err != nil {
		h.logger.Error("Failed to acknowledge outlier",
			zap.Error(err),
			zap.String("outlier_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to acknowledge outlier",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Outlier not found",
		})
		return
	}

	h.logger.Info("Outlier acknowledged",
		zap.String("outlier_id", id),
		zap.String("user_id", userID))

	c.JSON(http.StatusOK, api.SuccessResponse{
		Success: true,
		Message: "Outlier acknowledged successfully",
	})
}

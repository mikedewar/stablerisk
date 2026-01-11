package api

import (
	"time"

	"github.com/mikedewar/stablerisk/pkg/models"
)

// OutlierListRequest represents query parameters for listing outliers
type OutlierListRequest struct {
	Page          int                 `form:"page" binding:"omitempty,min=1"`
	Limit         int                 `form:"limit" binding:"omitempty,min=1,max=100"`
	Type          models.OutlierType  `form:"type" binding:"omitempty"`
	Severity      models.Severity     `form:"severity" binding:"omitempty"`
	Address       string              `form:"address" binding:"omitempty"`
	Acknowledged  *bool               `form:"acknowledged" binding:"omitempty"`
	FromTimestamp *time.Time          `form:"from" binding:"omitempty"`
	ToTimestamp   *time.Time          `form:"to" binding:"omitempty"`
}

// OutlierListResponse represents a paginated list of outliers
type OutlierListResponse struct {
	Outliers   []models.Outlier `json:"outliers"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

// AcknowledgeOutlierRequest represents a request to acknowledge an outlier
type AcknowledgeOutlierRequest struct {
	Notes string `json:"notes"`
}

// StatisticsResponse represents overall statistics
type StatisticsResponse struct {
	TotalTransactions int64                      `json:"total_transactions"`
	TotalOutliers     int64                      `json:"total_outliers"`
	OutliersBySeverity map[models.Severity]int64 `json:"outliers_by_severity"`
	OutliersByType    map[models.OutlierType]int64 `json:"outliers_by_type"`
	LastDetectionRun  time.Time                  `json:"last_detection_run"`
	DetectionRunning  bool                       `json:"detection_running"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]ServiceStatus `json:"services"`
	Version   string                 `json:"version"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"` // "outlier", "ping", "pong"
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

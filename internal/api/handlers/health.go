package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/api"
	"github.com/mikedewar/stablerisk/internal/graph"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db             *sql.DB
	raphtoryClient *graph.RaphtoryClient
	version        string
	logger         *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB, raphtoryClient *graph.RaphtoryClient, version string, logger *zap.Logger) *HealthHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	if version == "" {
		version = "dev"
	}

	return &HealthHandler{
		db:             db,
		raphtoryClient: raphtoryClient,
		version:        version,
		logger:         logger,
	}
}

// GetHealth returns the health status of the service and its dependencies
func (h *HealthHandler) GetHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := api.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   h.version,
		Services:  make(map[string]api.ServiceStatus),
	}

	// Check database
	dbHealthy := true
	dbMessage := "ok"
	if err := h.db.PingContext(ctx); err != nil {
		dbHealthy = false
		dbMessage = err.Error()
		response.Status = "unhealthy"
		h.logger.Error("Database health check failed", zap.Error(err))
	}
	response.Services["database"] = api.ServiceStatus{
		Healthy: dbHealthy,
		Message: dbMessage,
	}

	// Check Raphtory
	raphtoryHealthy := true
	raphtoryMessage := "ok"
	if err := h.raphtoryClient.Health(ctx); err != nil {
		raphtoryHealthy = false
		raphtoryMessage = err.Error()
		response.Status = "degraded"
		h.logger.Warn("Raphtory health check failed", zap.Error(err))
	}
	response.Services["raphtory"] = api.ServiceStatus{
		Healthy: raphtoryHealthy,
		Message: raphtoryMessage,
	}

	// Determine HTTP status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if response.Status == "degraded" {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	c.JSON(statusCode, response)
}

// GetReadiness returns the readiness status (for Kubernetes readiness probes)
func (h *HealthHandler) GetReadiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// Check database connectivity
	if err := h.db.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"message": "Database not ready",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

// GetLiveness returns the liveness status (for Kubernetes liveness probes)
func (h *HealthHandler) GetLiveness(c *gin.Context) {
	// Simple check that the service is running
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

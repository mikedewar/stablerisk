package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/mikedewar/stablerisk/internal/api/handlers"
	"github.com/mikedewar/stablerisk/internal/api/middleware"
	"github.com/mikedewar/stablerisk/internal/config"
	"github.com/mikedewar/stablerisk/internal/graph"
	"github.com/mikedewar/stablerisk/internal/security"
	"github.com/mikedewar/stablerisk/internal/websocket"
	"github.com/mikedewar/stablerisk/pkg/utils"
	"go.uber.org/zap"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := utils.NewLogger(utils.LoggerConfig{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		OutputPath: cfg.Logging.OutputPath,
		ErrorPath:  cfg.Logging.ErrorPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting StableRisk API Server",
		zap.String("version", version))

	// Connect to database
	db, err := connectDatabase(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Raphtory client
	raphtoryClient := graph.NewRaphtoryClient(graph.RaphtoryConfig{
		BaseURL:    cfg.Raphtory.BaseURL,
		Timeout:    cfg.Raphtory.Timeout,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}, logger)

	// Initialize JWT manager
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          cfg.Security.JWTSecret,
		Issuer:             "stablerisk",
		Audience:           "stablerisk-api",
		AccessTokenExpiry:  cfg.Security.JWTExpiry,
		RefreshTokenExpiry: cfg.Security.RefreshTokenExpiry,
	})

	// Initialize audit logger
	auditLogger := security.NewAuditLogger(db, security.AuditLoggerConfig{
		SecretKey:     cfg.Security.HMACKey,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}, logger)
	defer auditLogger.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, jwtManager, logger)
	outlierHandler := handlers.NewOutlierHandler(db, logger)
	statisticsHandler := handlers.NewStatisticsHandler(db, raphtoryClient, logger)
	healthHandler := handlers.NewHealthHandler(db, raphtoryClient, version, logger)
	wsHandler := handlers.NewWebSocketHandler(hub, jwtManager, logger)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, logger)
	rbacMiddleware := middleware.NewRBACMiddleware(logger)
	auditMiddleware := middleware.NewAuditMiddleware(auditLogger, logger)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode) // Production mode

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Public routes
	public := router.Group("/api/v1")
	{
		// Health checks (no auth required)
		router.GET("/health", healthHandler.GetHealth)
		router.GET("/readiness", healthHandler.GetReadiness)
		router.GET("/liveness", healthHandler.GetLiveness)

		// Authentication
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.RefreshToken)
	}

	// Protected routes (require authentication)
	protected := router.Group("/api/v1")
	protected.Use(auditMiddleware.Log())
	protected.Use(authMiddleware.Authenticate())
	{
		// User profile
		protected.GET("/auth/profile", authHandler.GetProfile)

		// Outliers (all authenticated users can read)
		protected.GET("/outliers", rbacMiddleware.RequireViewer(), outlierHandler.ListOutliers)
		protected.GET("/outliers/:id", rbacMiddleware.RequireViewer(), outlierHandler.GetOutlier)

		// Acknowledge outliers (analysts and admins only)
		protected.POST("/outliers/:id/acknowledge", rbacMiddleware.RequireAnalyst(), outlierHandler.AcknowledgeOutlier)

		// Statistics
		protected.GET("/statistics", rbacMiddleware.RequireViewer(), statisticsHandler.GetStatistics)
		protected.GET("/statistics/trends", rbacMiddleware.RequireViewer(), statisticsHandler.GetOutlierTrends)

		// WebSocket (authenticated)
		router.GET("/api/v1/ws", wsHandler.HandleWebSocket)
	}

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.APIPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("API server listening",
			zap.Int("port", cfg.Server.APIPort))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}

// connectDatabase establishes database connection with retry logic
func connectDatabase(cfg config.DatabaseConfig, logger *zap.Logger) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	var db *sql.DB
	var err error

	// Retry connection up to 5 times
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			logger.Warn("Failed to open database connection",
				zap.Error(err),
				zap.Int("attempt", i+1))
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			// Connection successful
			db.SetMaxOpenConns(cfg.MaxOpenConns)
			db.SetMaxIdleConns(cfg.MaxIdleConns)
			db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

			logger.Info("Database connection established",
				zap.String("host", cfg.Host),
				zap.Int("port", cfg.Port),
				zap.String("database", cfg.Database))

			return db, nil
		}

		logger.Warn("Failed to ping database",
			zap.Error(err),
			zap.Int("attempt", i+1))
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to database after 5 attempts: %w", err)
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

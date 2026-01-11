package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mikedewar/stablerisk/internal/blockchain"
	"github.com/mikedewar/stablerisk/internal/config"
	"github.com/mikedewar/stablerisk/internal/graph"
	"github.com/mikedewar/stablerisk/pkg/utils"
	"go.uber.org/zap"
)

const (
	serviceName = "stablerisk-monitor"
	version     = "0.1.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := utils.LoggerFromConfig(
		cfg.Logging.Level,
		cfg.Logging.Format,
		cfg.Logging.OutputPath,
		cfg.Logging.ErrorPath,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting monitor service",
		zap.String("service", serviceName),
		zap.String("version", version),
		zap.String("trongrid_url", cfg.TronGrid.WebSocketURL),
		zap.String("usdt_contract", cfg.TronGrid.USDTContract),
		zap.String("raphtory_url", cfg.Raphtory.BaseURL))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Raphtory client
	raphtoryClient := graph.NewRaphtoryClient(graph.RaphtoryConfig{
		BaseURL:    cfg.Raphtory.BaseURL,
		Timeout:    cfg.Raphtory.Timeout,
		MaxRetries: cfg.Raphtory.MaxRetries,
		RetryDelay: cfg.Raphtory.RetryDelay,
	}, logger)

	// Check Raphtory health
	logger.Info("Checking Raphtory health...")
	healthCtx, healthCancel := context.WithTimeout(ctx, 10*time.Second)
	defer healthCancel()

	if err := raphtoryClient.Health(healthCtx); err != nil {
		logger.Warn("Raphtory health check failed, will continue anyway",
			zap.Error(err))
	} else {
		logger.Info("Raphtory service is healthy")
	}

	// Initialize TronGrid client
	tronClient := blockchain.NewTronClient(blockchain.TronClientConfig{
		APIKey:       cfg.TronGrid.APIKey,
		WebSocketURL: cfg.TronGrid.WebSocketURL,
		USDTContract: cfg.TronGrid.USDTContract,
		PingInterval: cfg.TronGrid.PingInterval,
		RetryConfig: blockchain.RetryConfig{
			InitialDelay:   cfg.TronGrid.ReconnectDelay,
			MaxDelay:       30 * time.Second,
			MaxRetries:     cfg.TronGrid.MaxReconnects,
			Multiplier:     2.0,
			Jitter:         true,
			CircuitTimeout: 5 * time.Minute,
		},
	}, logger)

	// Start TronGrid client
	if err := tronClient.Start(); err != nil {
		logger.Fatal("Failed to start TronGrid client", zap.Error(err))
	}
	defer tronClient.Close()

	logger.Info("TronGrid client started, listening for USDT transactions...")

	// Start transaction processor
	go processTransactions(ctx, tronClient, raphtoryClient, logger)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("Shutting down gracefully...")

	// Close TronGrid client
	if err := tronClient.Close(); err != nil {
		logger.Error("Error closing TronGrid client", zap.Error(err))
	}

	// Wait for shutdown to complete or timeout
	<-shutdownCtx.Done()
	if shutdownCtx.Err() == context.DeadlineExceeded {
		logger.Warn("Shutdown timed out")
	}

	logger.Info("Monitor service stopped")
}

// processTransactions processes transactions from TronGrid and forwards them to Raphtory
func processTransactions(ctx context.Context, tronClient *blockchain.TronClient,
	raphtoryClient *graph.RaphtoryClient, logger *zap.Logger) {

	txCount := uint64(0)
	errorCount := uint64(0)
	startTime := time.Now()

	// Log statistics periodically
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Transaction processor stopped")
			return

		case tx := <-tronClient.Transactions():
			txCount++

			// Log transaction
			logger.Info("Transaction received",
				zap.Uint64("count", txCount),
				zap.String("tx_hash", tx.TxHash),
				zap.String("from", tx.From),
				zap.String("to", tx.To),
				zap.String("amount", tx.Amount.String()),
				zap.Uint64("block", tx.BlockNumber),
				zap.Time("timestamp", tx.Timestamp))

			// Forward to Raphtory
			forwardCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if err := raphtoryClient.AddTransaction(forwardCtx, tx); err != nil {
				errorCount++
				logger.Error("Failed to add transaction to Raphtory",
					zap.Error(err),
					zap.String("tx_hash", tx.TxHash))
			}
			cancel()

		case <-ticker.C:
			// Log statistics
			elapsed := time.Since(startTime)
			rate := float64(txCount) / elapsed.Seconds()

			logger.Info("Transaction processing statistics",
				zap.Uint64("total_transactions", txCount),
				zap.Uint64("errors", errorCount),
				zap.Duration("uptime", elapsed),
				zap.Float64("rate_per_second", rate),
				zap.String("status", string(tronClient.Status())),
				zap.Bool("connected", tronClient.IsConnected()))
		}
	}
}

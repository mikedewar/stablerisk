package blockchain

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	MaxRetries     int
	Multiplier     float64
	Jitter         bool
	CircuitTimeout time.Duration // How long to wait after max retries before trying again
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		MaxRetries:     10,
		Multiplier:     2.0,
		Jitter:         true,
		CircuitTimeout: 5 * time.Minute,
	}
}

// RetryHandler manages reconnection logic with exponential backoff
type RetryHandler struct {
	config        RetryConfig
	logger        *zap.Logger
	attempt       int
	circuitOpen   bool
	circuitOpened time.Time
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(config RetryConfig, logger *zap.Logger) *RetryHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &RetryHandler{
		config:      config,
		logger:      logger,
		attempt:     0,
		circuitOpen: false,
	}
}

// ShouldRetry determines if another retry attempt should be made
func (r *RetryHandler) ShouldRetry() bool {
	// If circuit breaker is open, check if timeout has elapsed
	if r.circuitOpen {
		if time.Since(r.circuitOpened) >= r.config.CircuitTimeout {
			r.logger.Info("Circuit breaker timeout elapsed, attempting to reconnect",
				zap.Duration("elapsed", time.Since(r.circuitOpened)))
			r.Reset()
			return true
		}
		return false
	}

	// Check if we've exceeded max retries
	if r.attempt >= r.config.MaxRetries {
		r.OpenCircuit()
		return false
	}

	return true
}

// NextDelay calculates the next retry delay using exponential backoff
func (r *RetryHandler) NextDelay() time.Duration {
	if r.attempt == 0 {
		return r.config.InitialDelay
	}

	// Calculate exponential delay: initialDelay * (multiplier ^ attempt)
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(r.attempt))

	// Apply max delay cap
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	if r.config.Jitter {
		jitter := float64(time.Now().UnixNano()%1000) / 1000.0 // 0-1 random
		delay = delay * (0.5 + jitter*0.5)                     // 50%-100% of calculated delay
	}

	return time.Duration(delay)
}

// Wait waits for the calculated retry delay
func (r *RetryHandler) Wait(ctx context.Context) error {
	delay := r.NextDelay()

	r.logger.Info("Waiting before retry",
		zap.Int("attempt", r.attempt+1),
		zap.Int("max_attempts", r.config.MaxRetries),
		zap.Duration("delay", delay))

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		r.attempt++
		return nil
	}
}

// Reset resets the retry counter (call on successful connection)
func (r *RetryHandler) Reset() {
	if r.attempt > 0 || r.circuitOpen {
		r.logger.Info("Resetting retry handler",
			zap.Int("previous_attempts", r.attempt),
			zap.Bool("circuit_was_open", r.circuitOpen))
	}
	r.attempt = 0
	r.circuitOpen = false
	r.circuitOpened = time.Time{}
}

// OpenCircuit opens the circuit breaker
func (r *RetryHandler) OpenCircuit() {
	r.circuitOpen = true
	r.circuitOpened = time.Now()
	r.logger.Warn("Circuit breaker opened, stopping retry attempts",
		zap.Int("failed_attempts", r.attempt),
		zap.Duration("timeout", r.config.CircuitTimeout))
}

// GetAttempt returns the current attempt number
func (r *RetryHandler) GetAttempt() int {
	return r.attempt
}

// IsCircuitOpen returns whether the circuit breaker is open
func (r *RetryHandler) IsCircuitOpen() bool {
	return r.circuitOpen
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config RetryConfig, logger *zap.Logger, fn func() error) error {
	handler := NewRetryHandler(config, logger)

	for {
		// Try the operation
		err := fn()
		if err == nil {
			handler.Reset()
			return nil
		}

		// Log the error
		logger.Error("Operation failed, will retry",
			zap.Error(err),
			zap.Int("attempt", handler.GetAttempt()+1))

		// Check if we should retry
		if !handler.ShouldRetry() {
			return fmt.Errorf("max retries exceeded: %w", err)
		}

		// Wait before retrying
		if err := handler.Wait(ctx); err != nil {
			return fmt.Errorf("retry cancelled: %w", err)
		}
	}
}

// RetryWithBackoffAsync executes a function with retry logic, calling a callback on each attempt
func RetryWithBackoffAsync(ctx context.Context, config RetryConfig, logger *zap.Logger,
	fn func() error, onRetry func(attempt int, err error)) {

	handler := NewRetryHandler(config, logger)

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Retry loop cancelled")
				return
			default:
			}

			// Try the operation
			err := fn()
			if err == nil {
				handler.Reset()
				// Wait a bit before trying again (for continuous operations)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}

			// Call retry callback
			if onRetry != nil {
				onRetry(handler.GetAttempt()+1, err)
			}

			// Check if we should retry
			if !handler.ShouldRetry() {
				logger.Error("Max retries exceeded, circuit breaker open",
					zap.Error(err))

				// Wait for circuit breaker timeout, then continue
				select {
				case <-ctx.Done():
					return
				case <-time.After(config.CircuitTimeout):
					handler.Reset()
					continue
				}
			}

			// Wait before retrying
			if err := handler.Wait(ctx); err != nil {
				logger.Info("Retry wait cancelled", zap.Error(err))
				return
			}
		}
	}()
}

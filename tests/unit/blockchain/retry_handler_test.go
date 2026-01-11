package blockchain_test

import (
	"context"
	"testing"
	"time"

	"github.com/mikedewar/stablerisk/internal/blockchain"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestRetryHandler_ShouldRetry(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   1 * time.Millisecond,
		MaxDelay:       10 * time.Millisecond,
		MaxRetries:     3,
		Multiplier:     2.0,
		Jitter:         false,
		CircuitTimeout: 100 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	handler := blockchain.NewRetryHandler(config, logger)

	// Should retry initially
	assert.True(t, handler.ShouldRetry())
	assert.Equal(t, 0, handler.GetAttempt())

	// Simulate attempts
	for i := 0; i < 3; i++ {
		assert.True(t, handler.ShouldRetry())
		handler.Wait(context.Background())
	}

	// After max retries, circuit should open
	assert.False(t, handler.ShouldRetry())
	assert.True(t, handler.IsCircuitOpen())
}

func TestRetryHandler_NextDelay(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		MaxRetries:     10,
		Multiplier:     2.0,
		Jitter:         false,
		CircuitTimeout: 5 * time.Minute,
	}

	logger := zaptest.NewLogger(t)
	handler := blockchain.NewRetryHandler(config, logger)

	tests := []struct {
		attempt      int
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{0, 1 * time.Second, 1 * time.Second},       // Initial delay
		{1, 2 * time.Second, 2 * time.Second},       // 1 * 2^1
		{2, 4 * time.Second, 4 * time.Second},       // 1 * 2^2
		{3, 8 * time.Second, 8 * time.Second},       // 1 * 2^3
		{4, 16 * time.Second, 16 * time.Second},     // 1 * 2^4
		{5, 30 * time.Second, 30 * time.Second},     // Capped at max
		{6, 30 * time.Second, 30 * time.Second},     // Still capped
	}

	for _, tt := range tests {
		t.Run(time.Duration(tt.attempt).String(), func(t *testing.T) {
			// Simulate reaching this attempt
			for i := 0; i < tt.attempt; i++ {
				handler.Wait(context.Background())
			}

			delay := handler.NextDelay()
			assert.GreaterOrEqual(t, delay, tt.expectedMin)
			assert.LessOrEqual(t, delay, tt.expectedMax)
		})

		// Reset for next test
		handler.Reset()
	}
}

func TestRetryHandler_Jitter(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		MaxRetries:     10,
		Multiplier:     2.0,
		Jitter:         true,
		CircuitTimeout: 5 * time.Minute,
	}

	logger := zaptest.NewLogger(t)
	handler := blockchain.NewRetryHandler(config, logger)

	// Simulate first attempt
	handler.Wait(context.Background())

	// With jitter, delay should vary between 50-100% of calculated value
	delay := handler.NextDelay()
	expectedBase := 2 * time.Second

	// Delay should be between 1s (50%) and 2s (100%)
	assert.GreaterOrEqual(t, delay, expectedBase/2)
	assert.LessOrEqual(t, delay, expectedBase)
}

func TestRetryHandler_Reset(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   1 * time.Millisecond,
		MaxDelay:       10 * time.Millisecond,
		MaxRetries:     3,
		Multiplier:     2.0,
		Jitter:         false,
		CircuitTimeout: 100 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	handler := blockchain.NewRetryHandler(config, logger)

	// Make some attempts
	for i := 0; i < 2; i++ {
		handler.Wait(context.Background())
	}
	assert.Equal(t, 2, handler.GetAttempt())

	// Reset
	handler.Reset()
	assert.Equal(t, 0, handler.GetAttempt())
	assert.False(t, handler.IsCircuitOpen())
	assert.Equal(t, config.InitialDelay, handler.NextDelay())
}

func TestRetryHandler_CircuitBreaker(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   1 * time.Millisecond,
		MaxDelay:       10 * time.Millisecond,
		MaxRetries:     2,
		Multiplier:     2.0,
		Jitter:         false,
		CircuitTimeout: 50 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	handler := blockchain.NewRetryHandler(config, logger)

	// Exhaust retries
	for i := 0; i < 3; i++ {
		if handler.ShouldRetry() {
			handler.Wait(context.Background())
		}
	}

	// Circuit should be open
	assert.True(t, handler.IsCircuitOpen())
	assert.False(t, handler.ShouldRetry())

	// Wait for circuit timeout
	time.Sleep(60 * time.Millisecond)

	// Circuit should allow retry after timeout
	assert.True(t, handler.ShouldRetry())
}

func TestRetryHandler_ContextCancellation(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		MaxRetries:     5,
		Multiplier:     2.0,
		Jitter:         false,
		CircuitTimeout: 1 * time.Minute,
	}

	logger := zaptest.NewLogger(t)
	handler := blockchain.NewRetryHandler(config, logger)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Wait should return context error
	err := handler.Wait(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestRetryWithBackoff(t *testing.T) {
	config := blockchain.RetryConfig{
		InitialDelay:   1 * time.Millisecond,
		MaxDelay:       10 * time.Millisecond,
		MaxRetries:     3,
		Multiplier:     2.0,
		Jitter:         false,
		CircuitTimeout: 100 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)

	t.Run("successful after retries", func(t *testing.T) {
		attempts := 0
		fn := func() error {
			attempts++
			if attempts < 3 {
				return assert.AnError
			}
			return nil
		}

		ctx := context.Background()
		err := blockchain.RetryWithBackoff(ctx, config, logger, fn)
		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		attempts := 0
		fn := func() error {
			attempts++
			return assert.AnError
		}

		ctx := context.Background()
		err := blockchain.RetryWithBackoff(ctx, config, logger, fn)
		assert.Error(t, err)
		assert.Greater(t, attempts, 3) // Should try at least MaxRetries times
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()

		fn := func() error {
			return assert.AnError
		}

		err := blockchain.RetryWithBackoff(ctx, config, logger, fn)
		assert.Error(t, err)
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := blockchain.DefaultRetryConfig()

	assert.Equal(t, 1*time.Second, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 10, config.MaxRetries)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.True(t, config.Jitter)
	assert.Equal(t, 5*time.Minute, config.CircuitTimeout)
}

// Benchmark retry delay calculation
func BenchmarkRetryHandler_NextDelay(b *testing.B) {
	config := blockchain.DefaultRetryConfig()
	logger := zaptest.NewLogger(b)
	handler := blockchain.NewRetryHandler(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.NextDelay()
	}
}

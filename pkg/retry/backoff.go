package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// Config defines retry behavior
type Config struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	Multiplier    float64
	JitterEnabled bool
}

// DefaultConfig returns production-ready retry settings
func DefaultConfig() Config {
	return Config{
		MaxRetries:    10,
		InitialDelay:  2 * time.Second,
		MaxDelay:      60 * time.Second,
		Multiplier:    2.0,
		JitterEnabled: true,
	}
}

// WithBackoff executes fn with exponential backoff and optional jitter
func WithBackoff(ctx context.Context, cfg Config, logger *zap.Logger, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		lastErr = fn()
		if lastErr == nil {
			if attempt > 1 {
				logger.Info("Operation succeeded after retries",
					zap.String("operation", operation),
					zap.Int("attempts", attempt))
			}
			return nil
		}

		if attempt == cfg.MaxRetries {
			return fmt.Errorf("%s failed after %d attempts: %w", operation, cfg.MaxRetries, lastErr)
		}

		delay := calculateBackoff(cfg, attempt)

		logger.Warn("Operation failed, retrying",
			zap.String("operation", operation),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", cfg.MaxRetries),
			zap.Duration("retry_in", delay),
			zap.Error(lastErr))

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return lastErr
}

func calculateBackoff(cfg Config, attempt int) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))

	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	if cfg.JitterEnabled {
		jitter := rand.Float64() * 0.3 * delay
		delay = delay + jitter - (0.15 * delay)
	}

	return time.Duration(delay)
}

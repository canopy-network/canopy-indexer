package backfill

import (
	"os"
	"strconv"
	"time"
)

// Config holds backfill-specific configuration.
type Config struct {
	// BatchSize is the number of blocks to process in each batch.
	BatchSize int

	// Concurrency is the number of concurrent block indexing operations.
	Concurrency int

	// StartHeight overrides the start of the range (default: 1).
	StartHeight uint64

	// EndHeight overrides the end of the range (default: current chain height).
	// Use 0 to fetch from RPC.
	EndHeight uint64

	// DryRun only reports gaps without indexing.
	DryRun bool

	// ProgressInterval is how often to log progress.
	ProgressInterval time.Duration

	// UseBlobs enables blob-based fetching (single RPC call per block).
	UseBlobs bool
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		BatchSize:        1000,
		Concurrency:      10,
		StartHeight:      1,
		EndHeight:        0,
		DryRun:           false,
		ProgressInterval: 10 * time.Second,
	}
}

// LoadConfig loads backfill configuration from environment variables.
func LoadConfig() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("BACKFILL_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BatchSize = n
		}
	}

	if v := os.Getenv("BACKFILL_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Concurrency = n
		}
	}

	if v := os.Getenv("BACKFILL_START_HEIGHT"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			cfg.StartHeight = n
		}
	}

	if v := os.Getenv("BACKFILL_END_HEIGHT"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			cfg.EndHeight = n
		}
	}

	if v := os.Getenv("BACKFILL_DRY_RUN"); v == "true" || v == "1" {
		cfg.DryRun = true
	}

	if v := os.Getenv("BACKFILL_PROGRESS_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ProgressInterval = d
		}
	}

	if v := os.Getenv("BACKFILL_USE_BLOBS"); v == "true" || v == "1" {
		cfg.UseBlobs = true
	}

	return cfg
}

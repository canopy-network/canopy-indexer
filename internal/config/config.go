package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ChainConfig holds the RPC endpoint for a specific chain.
type ChainConfig struct {
	ChainID uint64
	RPCURL  string
}

// Config holds all configuration for the indexer.
type Config struct {
	// Chains to index (hardcoded mock nodes)
	Chains []ChainConfig

	// RPC rate limiting
	RPCRPS   int
	RPCBurst int

	// PostgreSQL
	PostgresURL string

	// Redis
	RedisURL      string
	BlocksTopic   string
	ConsumerGroup string

	// Worker
	WorkerConcurrency int

	// WebSocket
	WSEnabled        bool
	WSMaxRetries     int
	WSReconnectDelay time.Duration

	// Logging
	LogLevel string

	// Backfill
	BackfillCheckInterval time.Duration // Periodic gap check interval (0 = disabled)
}

// MockChains returns the hardcoded list of 100 mock chain configurations.
// Chain IDs 1000-1099 on ports 60000-60099.
func MockChains() []ChainConfig {
	chains := make([]ChainConfig, 100)
	for i := 0; i < 100; i++ {
		chains[i] = ChainConfig{
			ChainID: uint64(1000 + i),
			RPCURL:  fmt.Sprintf("http://rpc-mock:%d", 60000+i),
		}
	}
	return chains
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		// Hardcoded mock chains
		Chains: MockChains(),

		// Defaults
		RPCRPS:            500,
		RPCBurst:          1000,
		BlocksTopic:       "blocks-to-index",
		ConsumerGroup:     "indexer-workers",
		WorkerConcurrency: 1,
		WSEnabled:         true,
		WSMaxRetries:      25,
		WSReconnectDelay:  time.Second,
		LogLevel:          "info",
	}

	// Required
	cfg.PostgresURL = os.Getenv("POSTGRES_URL")
	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("POSTGRES_URL is required")
	}

	cfg.RedisURL = os.Getenv("REDIS_URL")
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}

	// Optional overrides
	if v := os.Getenv("RPC_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RPCRPS = n
		}
	}

	if v := os.Getenv("RPC_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RPCBurst = n
		}
	}

	if v := os.Getenv("BLOCKS_TOPIC"); v != "" {
		cfg.BlocksTopic = v
	}

	if v := os.Getenv("CONSUMER_GROUP"); v != "" {
		cfg.ConsumerGroup = v
	}

	if v := os.Getenv("WORKER_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerConcurrency = n
		}
	}

	if v := os.Getenv("WS_ENABLED"); v != "" {
		cfg.WSEnabled = v == "true" || v == "1"
	}

	if v := os.Getenv("WS_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WSMaxRetries = n
		}
	}

	if v := os.Getenv("WS_RECONNECT_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.WSReconnectDelay = d
		}
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("BACKFILL_CHECK_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.BackfillCheckInterval = d
		}
	}

	return cfg, nil
}

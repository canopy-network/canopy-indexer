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

	// WebSocket (height notification mode)
	WSEnabled        bool
	WSMaxRetries     int
	WSReconnectDelay time.Duration

	// WebSocket Blob Mode (receives IndexerBlob instead of just height)
	WSBlobEnabled       bool
	WSBlobURL           string
	WSBlobChainID       uint64
	WSBlobAutoSubscribe bool // Enable auto-subscription for discovered chains (default: true)

	// Logging
	LogLevel string

	// Backfill
	BackfillCheckInterval time.Duration // Periodic gap check interval (0 = disabled)

	// Chain Rediscovery
	ChainRediscoveryInterval time.Duration // Periodic chain rediscovery interval

	// HTTP API
	HTTPEnabled bool
	HTTPAddr    string
	AdminToken  string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
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

	// WebSocket Blob Mode
	if v := os.Getenv("WS_BLOB_ENABLED"); v != "" {
		cfg.WSBlobEnabled = v == "true" || v == "1"
	}

	cfg.WSBlobURL = os.Getenv("WS_BLOB_URL")

	if v := os.Getenv("WS_BLOB_CHAIN_ID"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			cfg.WSBlobChainID = n
		}
	}

	if v := os.Getenv("WS_BLOB_AUTO_SUBSCRIBE"); v != "" {
		cfg.WSBlobAutoSubscribe = v == "true" || v == "1"
	} else {
		cfg.WSBlobAutoSubscribe = true // Default to true for auto-subscription
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("BACKFILL_CHECK_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.BackfillCheckInterval = d
		}
	}

	if v := os.Getenv("CHAIN_REDISCOVERY_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ChainRediscoveryInterval = d
		}
	} else {
		cfg.ChainRediscoveryInterval = 5 * time.Second // Default 5 seconds
	}

	// Default Canopy node for blob mode (can be overridden by WS_BLOB_URL)
	if cfg.WSBlobURL == "" && cfg.WSBlobEnabled {
		cfg.WSBlobURL = "http://host.docker.internal:50002"
	}
	if cfg.WSBlobChainID == 0 && cfg.WSBlobEnabled {
		cfg.WSBlobChainID = 1 // Default chain ID
	}

	// HTTP API Configuration
	if v := os.Getenv("HTTP_ENABLED"); v != "" {
		cfg.HTTPEnabled = v == "true" || v == "1"
	}

	cfg.HTTPAddr = os.Getenv("HTTP_ADDR")
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080" // Default port
	}

	cfg.AdminToken = os.Getenv("ADMIN_TOKEN")
	if cfg.AdminToken == "" {
		cfg.AdminToken = "devtoken" // Default token for development
	}

	return cfg, nil
}

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/api"
	"github.com/canopy-network/canopy-indexer/internal/config"
	"github.com/canopy-network/canopy-indexer/internal/indexer"
	"github.com/canopy-network/canopy-indexer/internal/listener"
	"github.com/canopy-network/canopy-indexer/internal/publisher"
	"github.com/canopy-network/canopy-indexer/internal/worker"
	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres/admin"
	"github.com/canopy-network/canopy/fsm"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Create logger for database connections
	logger, err := zap.NewProduction()
	if err != nil {
		slog.Error("failed to create logger", "err", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Connect to admin database for chain discovery
	adminPoolConfig := postgres.GetPoolConfigForComponent("admin")
	adminDB, err := admin.NewWithPoolConfig(ctx, logger, "admin", *adminPoolConfig)
	if err != nil {
		slog.Error("failed to connect to admin database", "err", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg.LogLevel)

	slog.Info("starting canopy-indexer",
		"ws_enabled", cfg.WSEnabled,
	)

	// Connect to PostgreSQL using postgres package

	// Use default pool config for indexer
	poolConfig := postgres.GetPoolConfigForComponent("indexer_chain")
	client, err := postgres.New(ctx, logger, "indexer", poolConfig)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	// Connect to Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to parse redis url", "err", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// Create publisher
	pub, err := publisher.New(redisClient, cfg.BlocksTopic)
	if err != nil {
		slog.Error("failed to create publisher", "err", err)
		os.Exit(1)
	}
	defer pub.Close()

	// Create indexer
	idx, err := indexer.New(ctx, logger, cfg, &client)
	if err != nil {
		slog.Error("failed to create indexer", "err", err)
		os.Exit(1)
	}
	defer idx.Close()

	// Create worker
	wrk, err := worker.New(worker.Config{
		RedisClient:   redisClient,
		Indexer:       idx,
		Topic:         cfg.BlocksTopic,
		ConsumerGroup: cfg.ConsumerGroup,
		Concurrency:   cfg.WorkerConcurrency,
	})
	if err != nil {
		slog.Error("failed to create worker", "err", err)
		os.Exit(1)
	}
	defer wrk.Close()

	// Run all components
	g, ctx := errgroup.WithContext(ctx)

	// Start HTTP API server if enabled
	if cfg.HTTPEnabled {
		apiServer, err := api.NewServer(adminDB, logger, cfg.HTTPAddr, cfg.AdminToken)
		if err != nil {
			slog.Error("failed to create API server", "err", err)
			os.Exit(1)
		}

		g.Go(func() error {
			slog.Info("starting HTTP API server", "addr", cfg.HTTPAddr)
			return apiServer.Run(ctx)
		})
	}

	// WebSocket Blob Mode - receives IndexerBlob instead of just height
	// Priority: Auto-subscribe takes precedence if enabled
	if cfg.WSBlobAutoSubscribe {
		// Use dynamic per-chain subscriptions (ignore WSBlobEnabled/WSBlobURL/WSBlobChainID)
		slog.Info("blob auto-subscription enabled - will subscribe to discovered chains")
	} else if cfg.WSBlobEnabled && cfg.WSBlobURL != "" {
		// Fall back to legacy single-chain mode
		chainID := cfg.WSBlobChainID
		blobListener := listener.NewBlobListener(listener.BlobConfig{
			URL:            cfg.WSBlobURL,
			ChainID:        chainID,
			MaxRetries:     cfg.WSMaxRetries,
			ReconnectDelay: cfg.WSReconnectDelay,
		}, func(b *fsm.IndexerBlob) error {
			// Wrap single blob in IndexerBlobs for decoder
			blobs := &fsm.IndexerBlobs{Current: b}
			data, err := blob.Decode(blobs, chainID)
			if err != nil {
				slog.Error("failed to decode blob", "err", err)
				return err
			}
			if err := idx.IndexBlockWithData(ctx, data); err != nil {
				slog.Error("failed to index blob", "height", data.Height, "err", err)
				return err
			}
			return nil
		})

		g.Go(func() error {
			slog.Info("starting blob websocket listener",
				"chain_id", chainID,
				"url", cfg.WSBlobURL,
			)
			return blobListener.Run(ctx)
		})
	} else if cfg.WSEnabled {
		// Legacy WebSocket mode - only receives height, requires HTTP fetch
		slog.Warn("WebSocket mode requires NODE_WS_URL environment variable to be set")
	}

	g.Go(func() error {
		slog.Info("starting worker")
		return wrk.Run(ctx)
	})

	g.Go(func() error {
		return runChainRediscovery(ctx, idx, cfg.ChainRediscoveryInterval)
	})

	if err := g.Wait(); err != nil && ctx.Err() == nil {
		slog.Error("indexer error", "err", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}

func runChainRediscovery(ctx context.Context, idx *indexer.Indexer, interval time.Duration) error {
	// Run immediately on startup
	if err := idx.Rediscover(ctx); err != nil {
		slog.Warn("initial chain discovery failed", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := idx.Rediscover(ctx); err != nil {
				slog.Error("chain rediscovery failed", "error", err)
			}
		}
	}
}

func setupLogging(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}

// startWSListener starts a WebSocket listener for a single chain (when WS is enabled).
func startWSListener(ctx context.Context, cfg *config.Config, pub *publisher.Publisher, wsURL string, chainID uint64) error {
	lst := listener.New(listener.Config{
		URL:            wsURL,
		ChainID:        chainID,
		MaxRetries:     cfg.WSMaxRetries,
		ReconnectDelay: cfg.WSReconnectDelay,
	}, func(chainID, height uint64) {
		if err := pub.PublishBlock(ctx, chainID, height); err != nil {
			slog.Error("failed to publish block", "chain_id", chainID, "height", height, "err", err)
		}
	})

	slog.Info("starting websocket listener", "chain_id", chainID, "url", wsURL)
	return lst.Run(ctx)
}

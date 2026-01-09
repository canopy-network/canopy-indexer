package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/backfill"
	"github.com/canopy-network/canopy-indexer/internal/config"
	"github.com/canopy-network/canopy-indexer/internal/indexer"
	"github.com/canopy-network/canopy-indexer/internal/listener"
	"github.com/canopy-network/canopy-indexer/internal/publisher"
	"github.com/canopy-network/canopy-indexer/internal/worker"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"github.com/canopy-network/canopy-indexer/pkg/snapshot"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopyx/pkg/db/postgres"
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

	// Setup logging
	setupLogging(cfg.LogLevel)

	slog.Info("starting canopy-indexer",
		"chains", len(cfg.Chains),
		"ws_enabled", cfg.WSEnabled,
	)

	// Connect to PostgreSQL using postgres package
	logger, err := zap.NewProduction()
	if err != nil {
		slog.Error("failed to create logger", "err", err)
		os.Exit(1)
	}
	defer logger.Sync()

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

	// Create RPC clients for all chains
	rpcClients := make(map[uint64]*rpc.HTTPClient)
	for _, chain := range cfg.Chains {
		rpcClients[chain.ChainID] = rpc.NewHTTPWithOpts(rpc.Opts{
			Endpoints: []string{chain.RPCURL},
			RPS:       cfg.RPCRPS,
			Burst:     cfg.RPCBurst,
		})
	}

	// Create publisher
	pub, err := publisher.New(redisClient, cfg.BlocksTopic)
	if err != nil {
		slog.Error("failed to create publisher", "err", err)
		os.Exit(1)
	}
	defer pub.Close()

	// Create indexer
	idx := indexer.New(rpcClients, &client)

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

	// WebSocket Snapshot Mode - receives IndexerBlob instead of just height
	if cfg.WSSnapshotEnabled && cfg.WSSnapshotURL != "" {
		chainID := cfg.WSSnapshotChainID
		snapshotListener := listener.NewSnapshotListener(listener.SnapshotConfig{
			URL:            cfg.WSSnapshotURL,
			ChainID:        chainID,
			MaxRetries:     cfg.WSMaxRetries,
			ReconnectDelay: cfg.WSReconnectDelay,
		}, func(blob *fsm.IndexerBlob) error {
			// Wrap single blob in IndexerBlobs for decoder
			blobs := &fsm.IndexerBlobs{Current: blob}
			data, err := snapshot.Decode(blobs, chainID)
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
			slog.Info("starting snapshot websocket listener",
				"chain_id", chainID,
				"url", cfg.WSSnapshotURL,
			)
			return snapshotListener.Run(ctx)
		})
	} else if cfg.WSEnabled {
		// Legacy WebSocket mode - only receives height, requires HTTP fetch
		slog.Warn("WebSocket mode requires NODE_WS_URL environment variable to be set")
	}

	g.Go(func() error {
		slog.Info("starting worker")
		return wrk.Run(ctx)
	})

	// Optional: Periodic gap health check for all chains
	if cfg.BackfillCheckInterval > 0 {
		for _, chain := range cfg.Chains {
			chainID := chain.ChainID
			rpcClient := rpcClients[chainID]
			bf := backfill.New(rpcClient, &client, idx, chainID, nil)
			g.Go(func() error {
				return runPeriodicHealthCheck(ctx, bf, chainID, cfg.BackfillCheckInterval)
			})
		}
	}

	if err := g.Wait(); err != nil && ctx.Err() == nil {
		slog.Error("indexer error", "err", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
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

// runPeriodicHealthCheck runs a periodic gap health check for a specific chain.
func runPeriodicHealthCheck(ctx context.Context, bf *backfill.Backfiller, chainID uint64, interval time.Duration) error {
	slog.Info("starting periodic gap health check", "chain_id", chainID, "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stats, err := bf.CheckHealth(ctx)
			if err != nil {
				slog.Warn("gap health check failed", "chain_id", chainID, "err", err)
				continue
			}

			if stats.TotalMissing > 0 {
				slog.Warn("gaps detected during health check",
					"chain_id", chainID,
					"missing_blocks", stats.TotalMissing,
					"first_missing", stats.FirstMissing,
					"last_missing", stats.LastMissing,
				)
			} else {
				slog.Debug("gap health check passed, no missing blocks", "chain_id", chainID)
			}
		}
	}
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

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/api"
	"github.com/canopy-network/canopy-indexer/internal/backfill"
	"github.com/canopy-network/canopy-indexer/internal/config"
	"github.com/canopy-network/canopy-indexer/internal/indexer"
	"github.com/canopy-network/canopy-indexer/internal/listener"
	"github.com/canopy-network/canopy-indexer/internal/publisher"
	"github.com/canopy-network/canopy-indexer/internal/worker"
	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres/admin"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
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
	defer adminDB.Close()

	// Discover chains dynamically
	adminChains, err := adminDB.ListChain(ctx, false)
	if err != nil {
		slog.Error("failed to list chains from admin database", "err", err)
		os.Exit(1)
	}

	// Filter and convert chains
	var chains []config.ChainConfig
	for _, chain := range adminChains {
		if chain.Deleted == 1 || chain.Paused == 1 || len(chain.RPCEndpoints) == 0 {
			continue
		}
		chains = append(chains, config.ChainConfig{
			ChainID: chain.ChainID,
			RPCURL:  chain.RPCEndpoints[0],
		})
	}

	if len(chains) == 0 {
		slog.Error("no active chains found in admin database")
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg.LogLevel)

	slog.Info("starting canopy-indexer",
		"chains", len(chains),
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

	// Create RPC clients for all chains
	rpcClients := make(map[uint64]*rpc.HTTPClient)
	for _, chain := range chains {
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
	idx, err := indexer.New(ctx, logger, chains[0].ChainID)
	if err != nil {
		slog.Error("failed to create indexer", "err", err)
		os.Exit(1)
	}

	// Set RPC clients for all chains
	for _, chain := range chains {
		idx.SetRPCClient(chain.ChainID, rpcClients[chain.ChainID])
	}

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
	if cfg.WSBlobEnabled && cfg.WSBlobURL != "" {
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

	// Optional: Periodic gap health check for all chains
	if cfg.BackfillCheckInterval > 0 {
		for _, chain := range chains {
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

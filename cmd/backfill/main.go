package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/canopy-network/pgindexer/internal/backfill"
	"github.com/canopy-network/pgindexer/internal/config"
	"github.com/canopy-network/pgindexer/internal/db"
	"github.com/canopy-network/pgindexer/internal/indexer"
	"github.com/canopy-network/pgindexer/pkg/rpc"
)

func main() {
	// Parse flags
	dryRun := flag.Bool("dry-run", false, "Only report gaps, don't index")
	startHeight := flag.Uint64("start", 0, "Start height (default: 1)")
	endHeight := flag.Uint64("end", 0, "End height (default: current chain height)")
	batchSize := flag.Int("batch", 0, "Batch size (default: 1000)")
	concurrency := flag.Int("concurrency", 0, "Number of concurrent workers (default: 10)")
	statsOnly := flag.Bool("stats", false, "Only show gap statistics")
	chainID := flag.Uint64("chain", 0, "Specific chain ID to backfill (default: all chains)")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load base configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg.LogLevel)

	// Determine which chains to backfill
	var chainsToBackfill []config.ChainConfig
	if *chainID > 0 {
		// Find the specific chain
		found := false
		for _, chain := range cfg.Chains {
			if chain.ChainID == *chainID {
				chainsToBackfill = append(chainsToBackfill, chain)
				found = true
				break
			}
		}
		if !found {
			slog.Error("chain not found in configuration", "chain_id", *chainID)
			os.Exit(1)
		}
	} else {
		chainsToBackfill = cfg.Chains
	}

	slog.Info("pgindexer backfill starting",
		"chains", len(chainsToBackfill),
	)

	// Connect to PostgreSQL
	pool, err := db.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Create RPC clients for all chains
	rpcClients := make(map[uint64]*rpc.HTTPClient)
	for _, chain := range chainsToBackfill {
		rpcClients[chain.ChainID] = rpc.NewHTTPWithOpts(rpc.Opts{
			Endpoints: []string{chain.RPCURL},
			RPS:       cfg.RPCRPS,
			Burst:     cfg.RPCBurst,
		})
	}

	// Create indexer
	idx := indexer.New(rpcClients, pool)

	// Build backfill config
	backfillCfg := backfill.LoadConfig()

	// Override with flags if provided
	if *dryRun {
		backfillCfg.DryRun = true
	}
	if *startHeight > 0 {
		backfillCfg.StartHeight = *startHeight
	}
	if *endHeight > 0 {
		backfillCfg.EndHeight = *endHeight
	}
	if *batchSize > 0 {
		backfillCfg.BatchSize = *batchSize
	}
	if *concurrency > 0 {
		backfillCfg.Concurrency = *concurrency
	}

	// Stats only mode
	if *statsOnly {
		for _, chain := range chainsToBackfill {
			rpcClient := rpcClients[chain.ChainID]
			bf := backfill.New(rpcClient, pool, idx, chain.ChainID, backfillCfg)
			stats, err := bf.CheckHealth(ctx)
			if err != nil {
				slog.Error("failed to check health", "chain_id", chain.ChainID, "err", err)
				continue
			}

			fmt.Printf("Gap Statistics for chain %d:\n", chain.ChainID)
			fmt.Printf("  Total Expected: %d\n", stats.TotalExpected)
			fmt.Printf("  Total Indexed:  %d\n", stats.TotalIndexed)
			fmt.Printf("  Total Missing:  %d\n", stats.TotalMissing)
			if stats.TotalMissing > 0 {
				fmt.Printf("  First Missing:  %d\n", stats.FirstMissing)
				fmt.Printf("  Last Missing:   %d\n", stats.LastMissing)
				completionPct := float64(stats.TotalIndexed) / float64(stats.TotalExpected) * 100
				fmt.Printf("  Completion:     %.2f%%\n", completionPct)
			} else {
				fmt.Printf("  Completion:     100%%\n")
			}
			fmt.Println()
		}
		os.Exit(0)
	}

	// Run backfill for all chains concurrently
	var wg sync.WaitGroup
	results := make(map[uint64]*backfill.Result)
	var mu sync.Mutex

	for _, chain := range chainsToBackfill {
		wg.Add(1)
		go func(chain config.ChainConfig) {
			defer wg.Done()

			rpcClient := rpcClients[chain.ChainID]
			bf := backfill.New(rpcClient, pool, idx, chain.ChainID, backfillCfg)

			result, err := bf.Run(ctx)
			if err != nil && ctx.Err() == nil {
				slog.Error("backfill failed", "chain_id", chain.ChainID, "err", err)
				return
			}

			mu.Lock()
			results[chain.ChainID] = result
			mu.Unlock()
		}(chain)
	}

	wg.Wait()

	// Print summary
	fmt.Printf("\nBackfill Summary:\n")
	var totalMissing, totalProcessed, totalSucceeded, totalFailed uint64
	for chainID, result := range results {
		if result == nil {
			continue
		}
		fmt.Printf("\nChain %d:\n", chainID)
		fmt.Printf("  Total Missing:   %d\n", result.TotalMissing)
		fmt.Printf("  Total Processed: %d\n", result.TotalProcessed)
		fmt.Printf("  Total Succeeded: %d\n", result.TotalSucceeded)
		fmt.Printf("  Total Failed:    %d\n", result.TotalFailed)
		fmt.Printf("  Duration:        %s\n", result.Duration)

		totalMissing += result.TotalMissing
		totalProcessed += result.TotalProcessed
		totalSucceeded += result.TotalSucceeded
		totalFailed += result.TotalFailed

		if result.TotalFailed > 0 {
			fmt.Printf("\n  Failed blocks (%d):\n", len(result.Errors))
			for i, err := range result.Errors {
				if i >= 5 {
					fmt.Printf("    ... and %d more\n", len(result.Errors)-5)
					break
				}
				fmt.Printf("    - %v\n", err)
			}
		}
	}

	fmt.Printf("\nOverall Totals:\n")
	fmt.Printf("  Total Missing:   %d\n", totalMissing)
	fmt.Printf("  Total Processed: %d\n", totalProcessed)
	fmt.Printf("  Total Succeeded: %d\n", totalSucceeded)
	fmt.Printf("  Total Failed:    %d\n", totalFailed)

	if totalFailed > 0 {
		os.Exit(1)
	}

	slog.Info("backfill complete")
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

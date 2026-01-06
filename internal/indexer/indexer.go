package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/pgindexer/pkg/rpc"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// Indexer handles the core indexing logic.
type Indexer struct {
	rpcClients map[uint64]*rpc.HTTPClient // chainID -> RPC client
	db         *pgxpool.Pool
}

// New creates a new Indexer with a map of RPC clients keyed by chain ID.
func New(rpcClients map[uint64]*rpc.HTTPClient, db *pgxpool.Pool) *Indexer {
	return &Indexer{
		rpcClients: rpcClients,
		db:         db,
	}
}

// rpcForChain returns the RPC client for the given chain ID.
func (idx *Indexer) rpcForChain(chainID uint64) (*rpc.HTTPClient, error) {
	client, ok := idx.rpcClients[chainID]
	if !ok {
		return nil, fmt.Errorf("no RPC client configured for chain %d", chainID)
	}
	return client, nil
}

// IndexBlock indexes a single block at the given height.
func (idx *Indexer) IndexBlock(ctx context.Context, chainID, height uint64) error {
	start := time.Now()

	// Get RPC client for this chain
	rpcClient, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}

	// 1. Fetch block from RPC with retry (block may not be ready immediately after WS notification)
	var block *lib.BlockResult
	for attempt := 0; attempt < 10; attempt++ {
		block, err = rpcClient.BlockByHeight(ctx, height)
		if err == nil {
			break
		}
		// Wait before retry - block might not be committed yet (exponential backoff, max 5s)
		delay := time.Duration(1<<attempt) * 500 * time.Millisecond
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
		slog.Debug("block not ready, retrying", "height", height, "attempt", attempt+1, "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	if err != nil {
		return fmt.Errorf("fetch block: %w", err)
	}
	if block == nil || block.BlockHeader == nil {
		return fmt.Errorf("block or block header is nil for height %d", height)
	}
	blockTime := time.UnixMicro(int64(block.BlockHeader.Time))

	// 2. Run all indexers in parallel
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error { return idx.runIndexer(gCtx, "block", chainID, height, blockTime, block) })
	g.Go(func() error { return idx.runIndexer(gCtx, "transactions", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "events", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "accounts", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "validators", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "pools", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "orders", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "dex_prices", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "dex_batch", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "params", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "supply", chainID, height, blockTime, nil) })
	g.Go(func() error { return idx.runIndexer(gCtx, "committees", chainID, height, blockTime, nil) })

	if err := g.Wait(); err != nil {
		return err
	}

	// 3. Build block summary from indexed data
	if err := idx.buildBlockSummary(ctx, chainID, height, blockTime); err != nil {
		return fmt.Errorf("build block summary: %w", err)
	}

	// 4. Update index progress
	if err := idx.updateProgress(ctx, chainID, height); err != nil {
		return fmt.Errorf("update progress: %w", err)
	}

	slog.Debug("indexed block",
		"chain_id", chainID,
		"height", height,
		"duration", time.Since(start),
	)

	return nil
}

// runIndexer wraps an indexer function with logging to identify failures.
func (idx *Indexer) runIndexer(ctx context.Context, name string, chainID, height uint64, blockTime time.Time, block *lib.BlockResult) error {
	var err error
	switch name {
	case "block":
		err = idx.saveBlock(ctx, chainID, height, blockTime, block)
	case "transactions":
		err = idx.indexTransactions(ctx, chainID, height, blockTime)
	case "events":
		err = idx.indexEvents(ctx, chainID, height, blockTime)
	case "accounts":
		err = idx.indexAccounts(ctx, chainID, height, blockTime)
	case "validators":
		err = idx.indexValidators(ctx, chainID, height, blockTime)
	case "pools":
		err = idx.indexPools(ctx, chainID, height, blockTime)
	case "orders":
		err = idx.indexOrders(ctx, chainID, height, blockTime)
	case "dex_prices":
		err = idx.indexDexPrices(ctx, chainID, height, blockTime)
	case "dex_batch":
		err = idx.indexDexBatch(ctx, chainID, height, blockTime)
	case "params":
		err = idx.indexParams(ctx, chainID, height, blockTime)
	case "supply":
		err = idx.indexSupply(ctx, chainID, height, blockTime)
	case "committees":
		err = idx.indexCommittees(ctx, chainID, height, blockTime)
	default:
		return fmt.Errorf("unknown indexer: %s", name)
	}
	if err != nil {
		slog.Error("indexer failed", "indexer", name, "height", height, "err", err)
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// updateProgress updates the indexing progress for a chain.
func (idx *Indexer) updateProgress(ctx context.Context, chainID, height uint64) error {
	_, err := idx.db.Exec(ctx, `SELECT update_index_progress($1, $2)`, chainID, height)
	return err
}

// buildBlockSummary aggregates counts from all indexed tables into block_summaries.
func (idx *Indexer) buildBlockSummary(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	_, err := idx.db.Exec(ctx, `SELECT build_block_summary($1, $2, $3)`, chainID, height, blockTime)
	return err
}

package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopyx/pkg/db/postgres"
)

// Indexer handles the core indexing logic.
type Indexer struct {
	rpcClients map[uint64]*rpc.HTTPClient // chainID -> RPC client
	db         *postgres.Client
}

// New creates a new Indexer with a map of RPC clients keyed by chain ID.
func New(rpcClients map[uint64]*rpc.HTTPClient, db *postgres.Client) *Indexer {
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

// IndexBlock indexes a single block at the given height using two-phase architecture:
// Phase 1 (Fetch): All RPC calls in parallel - any failure returns error (NACK)
// Phase 2 (Write): All DB writes in single transaction - any failure rolls back (NACK)
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

	// 2. Phase 1: Fetch all data in parallel (any RPC failure → return error → NACK)
	// On retry, cached RPC calls return immediately from the 10-block rolling cache
	data, err := idx.fetchAllData(ctx, rpcClient, chainID, height, block, blockTime)
	if err != nil {
		return fmt.Errorf("fetch data: %w", err)
	}

	// 3. Phase 2: Write all data atomically (any DB failure → rollback → NACK)
	if err := idx.writeAllData(ctx, data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// 4. Build block summary from indexed data
	if err := idx.buildBlockSummary(ctx, chainID, height, blockTime); err != nil {
		return fmt.Errorf("build block summary: %w", err)
	}

	// 5. Update index progress
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

// IndexBlockWithData indexes a block using pre-fetched data (from WebSocket blob).
// This bypasses the fetch phase entirely and goes directly to the write phase.
func (idx *Indexer) IndexBlockWithData(ctx context.Context, data *BlockData) error {
	start := time.Now()

	if data == nil {
		return fmt.Errorf("block data is nil")
	}

	// Phase 2: Write all data atomically (any DB failure → rollback)
	if err := idx.writeAllData(ctx, data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// Build block summary from indexed data
	if err := idx.buildBlockSummary(ctx, data.ChainID, data.Height, data.BlockTime); err != nil {
		return fmt.Errorf("build block summary: %w", err)
	}

	// Update index progress
	if err := idx.updateProgress(ctx, data.ChainID, data.Height); err != nil {
		return fmt.Errorf("update progress: %w", err)
	}

	slog.Debug("indexed block (from blob)",
		"chain_id", data.ChainID,
		"height", data.Height,
		"duration", time.Since(start),
	)

	return nil
}

// updateProgress updates the indexing progress for a chain.
func (idx *Indexer) updateProgress(ctx context.Context, chainID, height uint64) error {
	return idx.db.Exec(ctx, `SELECT update_index_progress($1, $2)`, chainID, height)
}

// buildBlockSummary aggregates counts from all indexed tables into block_summaries.
func (idx *Indexer) buildBlockSummary(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	return idx.db.Exec(ctx, `SELECT build_block_summary($1, $2, $3)`, chainID, height, blockTime)
}

package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
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

// IndexBlock indexes a single block at the given height using blob-based fetching.
func (idx *Indexer) IndexBlock(ctx context.Context, chainID, height uint64) error {
	rpcClient, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}

	// Fetch blob (single HTTP call)
	blobs, err := rpcClient.Blob(ctx, height)
	if err != nil {
		slog.Error("failed to fetch indexer blob",
			"chain_id", chainID,
			"height", height,
			"err", err,
		)
		return fmt.Errorf("fetch blob: %w", err)
	}

	// Decode blob to BlockData
	data, err := blob.Decode(blobs, chainID)
	if err != nil {
		return fmt.Errorf("decode blob: %w", err)
	}

	return idx.IndexBlockWithData(ctx, data)
}

// IndexBlockWithData indexes a block using pre-fetched data (from WebSocket blob).
// This bypasses the fetch phase entirely and goes directly to the write phase.
func (idx *Indexer) IndexBlockWithData(ctx context.Context, data *blob.BlockData) error {
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

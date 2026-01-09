package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	indexermodels "github.com/canopy-network/canopyx/pkg/db/models/indexer"
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

	// Convert BlockData to canopyx models (includes change detection)
	canopyxData := idx.convertToCanopyxModels(data)

	// Transactional canopyx writes
	if err := idx.writeWithCanopyx(ctx, canopyxData); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// Build block summary (computed from the data we just wrote)
	if err := idx.buildBlockSummary(ctx, canopyxData); err != nil {
		return fmt.Errorf("build block summary: %w", err)
	}

	// Record index progress
	if err := idx.updateProgress(ctx, data.ChainID, data.Height); err != nil {
		slog.Warn("failed to record index progress", "err", err)
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

// writeWithCanopyx implements transactional writes using canopyx database operations
func (idx *Indexer) writeWithCanopyx(ctx context.Context, data *CanopyxBlockData) error {
	// Placeholder: in canopyx, this would use idx.db.Client.BeginFunc with inserts
	return nil
}

// buildBlockSummary computes and inserts the block summary from converted data
func (idx *Indexer) buildBlockSummary(ctx context.Context, data *CanopyxBlockData) error {
	summary := &indexermodels.BlockSummary{
		Height:            data.Height,
		HeightTime:        data.BlockTime,
		TotalTransactions: data.Block.TotalTxs,
		NumTxs:            uint32(len(data.Transactions)),
		NumAccounts:       uint32(len(data.Accounts)),
		NumEvents:         uint32(len(data.Events)),
		NumValidators:     uint32(len(data.Validators)),
		NumPools:          uint32(len(data.Pools)),
		NumOrders:         uint32(len(data.Orders)),
		NumDexPrices:      uint32(len(data.DexPrices)),
		NumDexOrders:      uint32(len(data.DexOrders)),
		NumDexDeposits:    uint32(len(data.DexDeposits)),
		NumDexWithdrawals: uint32(len(data.DexWithdrawals)),
		NumCommittees:     uint32(len(data.Committees)),
		// ... compute other summary fields from data
	}

	// Count transactions by type
	for _, tx := range data.Transactions {
		switch tx.MessageType {
		case "send":
			summary.NumTxsSend++
		case "stake":
			summary.NumTxsStake++
			// ... other transaction types
		}
	}

	// Count events by type
	for _, event := range data.Events {
		switch event.EventType {
		case "reward":
			summary.NumEventsReward++
		case "slash":
			summary.NumEventsSlash++
		case "dex-swap":
			summary.NumEventsDexSwap++
			// ... other event types
		}
	}

	// Count DEX states
	for _, order := range data.DexOrders {
		switch order.State {
		case "future":
			summary.NumDexOrdersFuture++
		case "locked":
			summary.NumDexOrdersLocked++
		case "complete":
			summary.NumDexOrdersComplete++
			if order.Success {
				summary.NumDexOrdersSuccess++
			} else {
				summary.NumDexOrdersFailed++
			}
		}
	}

	// Supply info
	if data.Supply != nil {
		summary.SupplyChanged = true
		summary.SupplyTotal = data.Supply.Total
		summary.SupplyStaked = data.Supply.Staked
		summary.SupplyDelegatedOnly = data.Supply.DelegatedOnly
	}

	// Placeholder: in canopyx, this would be idx.db.InsertBlockSummariesStaging(ctx, summary)
	return nil
}

// convertToCanopyxModels applies change detection and converts to canopyx types
func (idx *Indexer) convertToCanopyxModels(data *blob.BlockData) *CanopyxBlockData {
	result := &CanopyxBlockData{
		ChainID:   data.ChainID,
		Height:    data.Height,
		BlockTime: data.BlockTime,
	}

	// Simple conversions
	result.Block = convertBlock(data.Block, data.ChainID, data.BlockTime)
	result.Transactions = convertTransactions(data.Transactions, data.ChainID, data.Height, data.BlockTime)
	result.Events = convertEvents(data.Events, data.ChainID, data.Height, data.BlockTime)
	result.Accounts = convertAccounts(data.Accounts, data.Height, data.BlockTime)
	result.Pools = convertPools(data.PoolsCurrent, data.Height, data.BlockTime)
	result.Orders = convertOrders(data.Orders, data.Height, data.BlockTime)
	result.DexPrices = convertDexPrices(data.DexPrices, data.Height, data.BlockTime)
	result.Params = convertParams(data.Params, data.Height, data.BlockTime)
	result.Supply = convertSupply(data.Supply, data.Height, data.BlockTime)
	result.Committees = convertCommittees(data.Committees, data.Height, data.BlockTime)

	// Change detection conversions
	result.Validators, result.CommitteeValidators = ConvertValidatorsWithChangeDetection(
		data.ValidatorsCurrent, data.ValidatorsPrevious,
		data.Height, data.BlockTime,
	)
	result.ValidatorNonSigningInfo = ConvertNonSignersWithChangeDetection(
		data.NonSignersCurrent, data.NonSignersPrevious,
		data.Height, data.BlockTime,
	)
	result.ValidatorDoubleSigningInfo = ConvertDoubleSignersWithChangeDetection(
		data.DoubleSignersCurrent, data.DoubleSignersPrevious,
		data.Height, data.BlockTime,
	)
	result.PoolPointsByHolder = ConvertPoolPointsWithChangeDetection(
		data.PoolsCurrent, data.PoolsPrevious,
		data.Height, data.BlockTime,
	)

	// DEX state machine conversions
	result.DexOrders = ConvertDexOrdersWithStateMachine(data)
	result.DexDeposits = ConvertDexDepositsWithStateMachine(data)
	result.DexWithdrawals = ConvertDexWithdrawalsWithStateMachine(data)

	return result
}

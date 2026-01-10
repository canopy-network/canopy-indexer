package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/blob"
	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres/admin"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres/chain"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Indexer handles the core indexing logic.
type Indexer struct {
	rpcClients map[uint64]*rpc.HTTPClient
	db         *chain.DB // Chain database for entity inserts
	adminDB    *admin.DB // Admin database for index progress
}

// New creates a new Indexer with initialized databases.
func New(ctx context.Context, logger *zap.Logger, chainID uint64) (*Indexer, error) {
	// Initialize chain DB
	chainDB, err := chain.NewWithPoolConfig(ctx, logger, chainID,
		*postgres.GetPoolConfigForComponent("indexer_chain"))
	if err != nil {
		return nil, fmt.Errorf("create chain db: %w", err)
	}

	// Initialize admin DB
	adminDB, err := admin.NewWithPoolConfig(ctx, logger, "indexer_admin",
		*postgres.GetPoolConfigForComponent("indexer_admin"))
	if err != nil {
		return nil, fmt.Errorf("create admin db: %w", err)
	}

	return &Indexer{
		rpcClients: make(map[uint64]*rpc.HTTPClient),
		db:         chainDB,
		adminDB:    adminDB,
	}, nil
}

// SetRPCClient sets the RPC client for a specific chain.
func (idx *Indexer) SetRPCClient(chainID uint64, client *rpc.HTTPClient) {
	idx.rpcClients[chainID] = client
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
	if err := idx.updateProgress(ctx, data.ChainID, data.Height, time.Since(start)); err != nil {
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
func (idx *Indexer) updateProgress(ctx context.Context, chainID, height uint64, duration time.Duration) error {
	indexingTimeMs := float64(duration.Milliseconds())
	return idx.adminDB.RecordIndexed(ctx, chainID, height, indexingTimeMs, "indexed")
}

// writeWithCanopyx implements transactional writes using canopyx database operations
func (idx *Indexer) writeWithCanopyx(ctx context.Context, data *CanopyxBlockData) error {
	return idx.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		// Embed transaction in context for all insert methods
		txCtx := idx.db.WithTx(ctx, tx)

		// Insert in dependency order
		if err := idx.db.InsertBlocksStaging(txCtx, data.Block); err != nil {
			return fmt.Errorf("insert block: %w", err)
		}
		if err := idx.db.InsertTransactionsStaging(txCtx, data.Transactions); err != nil {
			return fmt.Errorf("insert transactions: %w", err)
		}
		if err := idx.db.InsertEventsStaging(txCtx, data.Events); err != nil {
			return fmt.Errorf("insert events: %w", err)
		}
		if err := idx.db.InsertAccountsStaging(txCtx, data.Accounts); err != nil {
			return fmt.Errorf("insert accounts: %w", err)
		}
		if err := idx.db.InsertValidatorsStaging(txCtx, data.Validators); err != nil {
			return fmt.Errorf("insert validators: %w", err)
		}
		if err := idx.db.InsertValidatorNonSigningInfoStaging(txCtx, data.ValidatorNonSigningInfo); err != nil {
			return fmt.Errorf("insert validator non-signing info: %w", err)
		}
		if err := idx.db.InsertValidatorDoubleSigningInfoStaging(txCtx, data.ValidatorDoubleSigningInfo); err != nil {
			return fmt.Errorf("insert validator double-signing info: %w", err)
		}
		if err := idx.db.InsertPoolsStaging(txCtx, data.Pools); err != nil {
			return fmt.Errorf("insert pools: %w", err)
		}
		if err := idx.db.InsertOrdersStaging(txCtx, data.Orders); err != nil {
			return fmt.Errorf("insert orders: %w", err)
		}
		if err := idx.db.InsertDexPricesStaging(txCtx, data.DexPrices); err != nil {
			return fmt.Errorf("insert dex prices: %w", err)
		}
		if err := idx.db.InsertDexOrdersStaging(txCtx, data.DexOrders); err != nil {
			return fmt.Errorf("insert dex orders: %w", err)
		}
		if err := idx.db.InsertDexDepositsStaging(txCtx, data.DexDeposits); err != nil {
			return fmt.Errorf("insert dex deposits: %w", err)
		}
		if err := idx.db.InsertDexWithdrawalsStaging(txCtx, data.DexWithdrawals); err != nil {
			return fmt.Errorf("insert dex withdrawals: %w", err)
		}
		if err := idx.db.InsertPoolPointsByHolderStaging(txCtx, data.PoolPointsByHolder); err != nil {
			return fmt.Errorf("insert pool points: %w", err)
		}
		if data.Params != nil {
			if err := idx.db.InsertParamsStaging(txCtx, data.Params); err != nil {
				return fmt.Errorf("insert params: %w", err)
			}
		}
		if data.Supply != nil {
			if err := idx.db.InsertSupplyStaging(txCtx, []*indexermodels.Supply{data.Supply}); err != nil {
				return fmt.Errorf("insert supply: %w", err)
			}
		}
		if err := idx.db.InsertCommitteesStaging(txCtx, data.Committees); err != nil {
			return fmt.Errorf("insert committees: %w", err)
		}
		if err := idx.db.InsertCommitteeValidatorsStaging(txCtx, data.CommitteeValidators); err != nil {
			return fmt.Errorf("insert committee validators: %w", err)
		}
		if err := idx.db.InsertCommitteePaymentsStaging(txCtx, data.CommitteePayments); err != nil {
			return fmt.Errorf("insert committee payments: %w", err)
		}

		return nil
	})
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

	// Insert block summary using chain database
	return idx.db.InsertBlockSummariesStaging(ctx, summary)
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

package indexer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/config"
	"github.com/canopy-network/canopy-indexer/pkg/blob"
	adminmodels "github.com/canopy-network/canopy-indexer/pkg/db/models/admin"
	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres/admin"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres/chain"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// ErrChainNotConfigured is returned when attempting to index a chain that hasn't been configured.
var ErrChainNotConfigured = errors.New("chain not configured")

// Indexer handles the core indexing logic.
type Indexer struct {
	mu             sync.RWMutex
	rpcClients     map[uint64]*rpc.HTTPClient
	chains         map[uint64]*chain.DB // Per-chain databases
	adminDB        *admin.DB            // Admin database for index progress
	logger         *zap.Logger
	cfg            *config.Config
	postgresClient *postgres.Client
}

// New creates a new Indexer with initialized databases.
func New(ctx context.Context, logger *zap.Logger, cfg *config.Config, postgresClient *postgres.Client) (*Indexer, error) {
	// Initialize admin DB - use "admin" database to share chains with API
	adminDB, err := admin.NewWithPoolConfig(ctx, logger, "admin",
		*postgres.GetPoolConfigForComponent("indexer_admin"))
	if err != nil {
		return nil, fmt.Errorf("create admin db: %w", err)
	}

	return &Indexer{
		rpcClients:     make(map[uint64]*rpc.HTTPClient),
		chains:         make(map[uint64]*chain.DB),
		adminDB:        adminDB,
		logger:         logger,
		cfg:            cfg,
		postgresClient: postgresClient,
	}, nil
}

// SetRPCClient sets the RPC client for a specific chain.
func (idx *Indexer) SetRPCClient(chainID uint64, client *rpc.HTTPClient) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.rpcClients[chainID] = client
}

// rpcForChain returns the RPC client for the given chain ID.
func (idx *Indexer) rpcForChain(chainID uint64) (*rpc.HTTPClient, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
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

	idx.mu.RLock()
	db, exists := idx.chains[data.ChainID]
	idx.mu.RUnlock()

	if !exists {
		return ErrChainNotConfigured
	}

	// Convert BlockData to canopyx models (includes change detection)
	canopyxData := idx.convertToCanopyxModels(data)

	// Transactional canopyx writes
	if err := idx.writeWithCanopyx(ctx, db, canopyxData); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// Build block summary (computed from the data we just wrote)
	if err := idx.buildBlockSummary(ctx, db, canopyxData); err != nil {
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

// AddChain initializes RPC client and chain DB for a new chain
func (idx *Indexer) AddChain(ctx context.Context, chainID uint64, rpcURL string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.chains[chainID]; exists {
		return fmt.Errorf("chain %d already exists", chainID)
	}

	chainDB, err := chain.NewWithPoolConfig(ctx, idx.logger, chainID,
		*postgres.GetPoolConfigForComponent("indexer_chain"))
	if err != nil {
		return fmt.Errorf("create chain db: %w", err)
	}

	client := rpc.NewHTTPWithOpts(rpc.Opts{
		Endpoints: []string{rpcURL},
	})
	if client == nil {
		return fmt.Errorf("create rpc client: invalid URL")
	}

	idx.chains[chainID] = chainDB
	idx.rpcClients[chainID] = client

	return nil
}

// RemoveChain closes connections and removes a chain
func (idx *Indexer) RemoveChain(chainID uint64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if db, exists := idx.chains[chainID]; exists {
		db.Close()
		delete(idx.chains, chainID)
	}

	delete(idx.rpcClients, chainID)
}

// HasChain checks if a chain is currently configured
func (idx *Indexer) HasChain(chainID uint64) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	_, exists := idx.chains[chainID]
	return exists
}

// ChainIDs returns all currently configured chain IDs
func (idx *Indexer) ChainIDs() []uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ids := make([]uint64, 0, len(idx.chains))
	for id := range idx.chains {
		ids = append(ids, id)
	}
	return ids
}

// Rediscover fetches chains from admin DB and syncs state
func (idx *Indexer) Rediscover(ctx context.Context) error {
	// Fetch active chains from admin DB
	adminChains, err := idx.adminDB.ListChain(ctx, false)
	if err != nil {
		return fmt.Errorf("list chains: %w", err)
	}

	// Log all chains found
	idx.logger.Info("chain rediscovery started",
		zap.Int("total_chains_in_db", len(adminChains)),
	)
	for _, c := range adminChains {
		idx.logger.Debug("found chain in admin DB",
			zap.Uint64("chain_id", c.ChainID),
			zap.String("chain_name", c.ChainName),
			zap.Strings("rpc_endpoints", c.RPCEndpoints),
			zap.Uint8("paused", c.Paused),
			zap.Uint8("deleted", c.Deleted),
		)
	}

	// Build set of active chain IDs
	activeSet := make(map[uint64]adminmodels.Chain)
	for _, c := range adminChains {
		if c.Deleted == 0 && c.Paused == 0 && len(c.RPCEndpoints) > 0 {
			activeSet[c.ChainID] = c
		}
	}

	idx.logger.Info("active chains after filtering",
		zap.Int("active_count", len(activeSet)),
	)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove chains no longer active
	for chainID := range idx.chains {
		if _, active := activeSet[chainID]; !active {
			idx.removeChainLocked(chainID)
			idx.logger.Info("removed chain", zap.Uint64("chain_id", chainID))
		}
	}

	// Add new chains
	for chainID, c := range activeSet {
		if _, exists := idx.chains[chainID]; !exists {
			if err := idx.addChainLocked(ctx, chainID, c.RPCEndpoints[0]); err != nil {
				idx.logger.Error("failed to add chain", zap.Uint64("chain_id", chainID), zap.Error(err))
				continue
			}
			idx.logger.Info("added chain", zap.Uint64("chain_id", chainID))
		}
	}

	return nil
}

// addChainLocked adds a chain without locking (caller must hold lock)
func (idx *Indexer) addChainLocked(ctx context.Context, chainID uint64, rpcURL string) error {
	chainDB, err := chain.NewWithPoolConfig(ctx, idx.logger, chainID,
		*postgres.GetPoolConfigForComponent("indexer_chain"))
	if err != nil {
		return fmt.Errorf("create chain db: %w", err)
	}

	client := rpc.NewHTTPWithOpts(rpc.Opts{
		Endpoints: []string{rpcURL},
	})
	if client == nil {
		return fmt.Errorf("create rpc client: invalid URL")
	}

	idx.chains[chainID] = chainDB
	idx.rpcClients[chainID] = client

	return nil
}

// removeChainLocked removes a chain without locking (caller must hold lock)
func (idx *Indexer) removeChainLocked(chainID uint64) {
	if db, exists := idx.chains[chainID]; exists {
		db.Close()
		delete(idx.chains, chainID)
	}

	delete(idx.rpcClients, chainID)
}

// Close closes all connections and resources.
func (idx *Indexer) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var errs []error

	// Close all chain databases
	for chainID, db := range idx.chains {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("chain %d: %w", chainID, err))
		}
		delete(idx.chains, chainID)
	}

	// Close admin DB
	if err := idx.adminDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("admin db: %w", err))
	}

	return errors.Join(errs...)
}

// writeWithCanopyx implements transactional writes using canopyx database operations
func (idx *Indexer) writeWithCanopyx(ctx context.Context, db *chain.DB, data *CanopyxBlockData) error {
	return db.BeginFunc(ctx, func(tx pgx.Tx) error {
		// Embed transaction in context for all insert methods
		txCtx := db.WithTx(ctx, tx)

		// Insert in dependency order
		if err := db.InsertBlocksStaging(txCtx, data.Block); err != nil {
			return fmt.Errorf("insert block: %w", err)
		}
		if err := db.InsertTransactionsStaging(txCtx, data.Transactions); err != nil {
			return fmt.Errorf("insert transactions: %w", err)
		}
		if err := db.InsertEventsStaging(txCtx, data.Events); err != nil {
			return fmt.Errorf("insert events: %w", err)
		}
		if err := db.InsertAccountsStaging(txCtx, data.Accounts); err != nil {
			return fmt.Errorf("insert accounts: %w", err)
		}
		if err := db.InsertValidatorsStaging(txCtx, data.Validators); err != nil {
			return fmt.Errorf("insert validators: %w", err)
		}
		if err := db.InsertValidatorNonSigningInfoStaging(txCtx, data.ValidatorNonSigningInfo); err != nil {
			return fmt.Errorf("insert validator non-signing info: %w", err)
		}
		if err := db.InsertValidatorDoubleSigningInfoStaging(txCtx, data.ValidatorDoubleSigningInfo); err != nil {
			return fmt.Errorf("insert validator double-signing info: %w", err)
		}
		if err := db.InsertPoolsStaging(txCtx, data.Pools); err != nil {
			return fmt.Errorf("insert pools: %w", err)
		}
		if err := db.InsertOrdersStaging(txCtx, data.Orders); err != nil {
			return fmt.Errorf("insert orders: %w", err)
		}
		if err := db.InsertDexPricesStaging(txCtx, data.DexPrices); err != nil {
			return fmt.Errorf("insert dex prices: %w", err)
		}
		if err := db.InsertDexOrdersStaging(txCtx, data.DexOrders); err != nil {
			return fmt.Errorf("insert dex orders: %w", err)
		}
		if err := db.InsertDexDepositsStaging(txCtx, data.DexDeposits); err != nil {
			return fmt.Errorf("insert dex deposits: %w", err)
		}
		if err := db.InsertDexWithdrawalsStaging(txCtx, data.DexWithdrawals); err != nil {
			return fmt.Errorf("insert dex withdrawals: %w", err)
		}
		if err := db.InsertPoolPointsByHolderStaging(txCtx, data.PoolPointsByHolder); err != nil {
			return fmt.Errorf("insert pool points: %w", err)
		}
		if data.Params != nil {
			if err := db.InsertParamsStaging(txCtx, data.Params); err != nil {
				return fmt.Errorf("insert params: %w", err)
			}
		}
		if data.Supply != nil {
			if err := db.InsertSupplyStaging(txCtx, []*indexermodels.Supply{data.Supply}); err != nil {
				return fmt.Errorf("insert supply: %w", err)
			}
		}
		if err := db.InsertCommitteesStaging(txCtx, data.Committees); err != nil {
			return fmt.Errorf("insert committees: %w", err)
		}
		if err := db.InsertCommitteeValidatorsStaging(txCtx, data.CommitteeValidators); err != nil {
			return fmt.Errorf("insert committee validators: %w", err)
		}
		if err := db.InsertCommitteePaymentsStaging(txCtx, data.CommitteePayments); err != nil {
			return fmt.Errorf("insert committee payments: %w", err)
		}

		return nil
	})
}

// buildBlockSummary computes and inserts the block summary from converted data
func (idx *Indexer) buildBlockSummary(ctx context.Context, db *chain.DB, data *CanopyxBlockData) error {
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
	return db.InsertBlockSummariesStaging(ctx, summary)
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

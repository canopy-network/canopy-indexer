# Blob Migration Plan: Replace internal/indexer with canopyx

## Overview
This document outlines the comprehensive plan to migrate from the custom `internal/indexer` implementation to canopyx database functionality, utilizing the blob RPC endpoint for dramatically simplified data fetching.

## Key Changes
- **Data Fetching**: Replace 20+ parallel RPC calls with single blob fetch
- **Database Operations**: Replace custom SQL logic with canopyx transactional inserts
- **Change Detection**: Preserve snapshot-on-change logic in conversion layer
- **Code Reduction**: Eliminate ~10 files (~1000+ lines) of legacy SQL implementation

## Confirmed Requirements
- ✅ Blob endpoint available: `POST /v1/query/indexer-blobs` with `{"height": 0}` or specific height
- ✅ BlockData needs conversion to canopyx models
- ✅ Conversion functions in `internal/indexer`
- ✅ Keep `pkg/transform` for data transformation utilities
- ✅ canopyx provides all `Insert*Staging` methods
- ✅ canopyx `indexermodels` package exists with all required types
- ❌ No legacy RPC fallback

## Critical: Change Detection Requirements

The indexer implements **snapshot-on-change** patterns that only write records when data changes between heights H-1 and H. This prevents database bloat and maintains query performance.

### Entities Requiring Change Detection

| Entity | H-1 Data Required | Change Logic |
|--------|-------------------|--------------|
| Validators | `ValidatorsPrevious` | Field comparison (9 fields) |
| CommitteeValidators | (derived) | Only for changed validators |
| NonSigners | `NonSignersPrevious` | Counter changes + reset detection |
| DoubleSigners | `DoubleSignersPrevious` | Evidence count + clear detection |
| PoolPointsByHolder | `PoolsPrevious` | Points or TotalPoolPoints changed |
| DexOrders | `DexBatchesPrevious*` | State machine + event correlation |
| DexDeposits | `DexBatchesPrevious*` | State machine + event correlation |
| DexWithdrawals | `DexBatchesPrevious*` | State machine + event correlation |

### Validator Change Detection
Compare each validator at H against H-1:
- Fields: `StakedAmount`, `PublicKey`, `NetAddress`, `MaxPausedHeight`, `UnstakingHeight`, `Output`, `Delegate`, `Compound`, `Committees`
- Write only if: new validator OR any field changed

### Non-Signer Reset Detection (Two-Phase)
1. **Updates**: Write if counter changed or new entry
2. **Resets**: If validator was at H-1 but NOT at H → write `missed_blocks_count = 0`

### Double-Signer Evidence Tracking (Two-Phase)
1. **Updates**: Write if evidence count changed
2. **Clears**: If validator was at H-1 but NOT at H → write zeros (evidence cleared)

### Pool Points Snapshot Triggers
Write if ANY of:
- Holder is NEW (not in H-1)
- Holder's `Points` changed
- Pool's `TotalPoolPoints` changed

### DEX State Machine
States: `FUTURE` → `LOCKED` → `COMPLETE` (orders) or `PENDING` → `LOCKED` → `COMPLETE` (deposits/withdrawals)

Requires correlation of:
- Current batch data (H)
- Previous batch data (H-1)
- Completion events at H

---

## Implementation Phases

### Phase 1: Create Conversion Functions with Change Detection ✅ COMPLETED
**Location:** `internal/indexer/conversions.go` (768 lines)

Implemented change detection conversion functions that output canopyx `indexermodels` types:

#### Change Detection Conversions (Implemented)

| Function | Description | Lines |
|----------|-------------|-------|
| `ConvertValidatorsWithChangeDetection()` | Compares 9 fields at H vs H-1, returns changed validators + committee assignments | 18-101 |
| `ConvertNonSignersWithChangeDetection()` | Two-phase: counter changes + reset detection (H-1 present, H absent → reset to 0) | 117-175 |
| `ConvertDoubleSignersWithChangeDetection()` | Two-phase: evidence count changes + clear detection | 181-239 |
| `ConvertPoolPointsWithChangeDetection()` | Snapshots when points or TotalPoolPoints change | 245-295 |
| `ConvertDexOrdersWithStateMachine()` | FUTURE → LOCKED → COMPLETE with event correlation | 318-404 |
| `ConvertDexDepositsWithStateMachine()` | PENDING → LOCKED → COMPLETE with event correlation | 407-473 |
| `ConvertDexWithdrawalsWithStateMachine()` | PENDING → LOCKED → COMPLETE with event correlation | 476-545 |

#### Helper Functions (Implemented)

| Function | Description |
|----------|-------------|
| `equalCommittees()` | Order-independent committee slice comparison |
| `parseDexEventsFromSlice()` | Extracts DEX events for completion correlation |
| `parseDexEvent()` | Parses event JSON into dexEvent struct |
| `buildH1Maps()` | Builds H-1 lookup maps for DEX change detection |
| `poolHolderKey()` | Creates unique key for pool holder lookups |

#### Types Defined

```go
// dexEvent holds parsed event data for DEX completion correlation
type dexEvent struct {
    OrderID, EventType string
    SoldAmount, BoughtAmount, PointsReceived, LocalAmount, RemoteAmount, PointsBurned uint64
    LocalOrigin, Success bool
}

// h1Maps holds H-1 data for DEX change detection
type h1Maps struct {
    OrdersLocked, OrdersPending map[string]*lib.DexLimitOrder
    DepositsLocked, DepositsPending map[string]*lib.DexLiquidityDeposit
    WithdrawalsLocked, WithdrawalsPending map[string]*lib.DexLiquidityWithdraw
}
```

#### Files Modified
- `internal/indexer/validators.go` - Removed `equalCommittees()` (now in conversions.go)
- `internal/indexer/pools.go` - Removed `poolHolderKey()` (now in conversions.go)
- `internal/indexer/dex.go` - Removed `dexEvent` and `h1Maps` types (now in conversions.go)

#### Simple Conversions (TODO - Phase 1b)
These still need to be implemented to convert BlockData fields to canopyx models:
- `convertBlock()`, `convertTransactions()`, `convertEvents()`, `convertAccounts()`
- `convertPools()`, `convertOrders()`, `convertDexPrices()`
- `convertParams()`, `convertSupply()`, `convertCommittees()`

### Phase 2: Update IndexBlock() Method
**File:** `internal/indexer/indexer.go`

Replace the current implementation with blob-based fetching:

```go
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
```

### Phase 3: Update IndexBlockWithData() Method
**File:** `internal/indexer/indexer.go`

```go
func (idx *Indexer) IndexBlockWithData(ctx context.Context, data *BlockData) error {
    start := time.Now()

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
    if err := idx.adminDB.RecordIndexed(ctx, data.ChainID, data.Height, 0, ""); err != nil {
        slog.Warn("failed to record index progress", "err", err)
    }

    slog.Debug("indexed block (from blob)",
        "chain_id", data.ChainID,
        "height", data.Height,
        "duration", time.Since(start),
    )

    return nil
}

// convertToCanopyxModels applies change detection and converts to canopyx types
func (idx *Indexer) convertToCanopyxModels(data *BlockData) *CanopyxBlockData {
    result := &CanopyxBlockData{
        ChainID:   data.ChainID,
        Height:    data.Height,
        BlockTime: data.BlockTime,
    }

    // Simple conversions
    result.Block = convertBlock(data.Block, data.ChainID, data.BlockTime)
    result.Transactions = convertTransactions(data.Transactions, data.ChainID, data.Height, data.BlockTime)
    result.Events = convertEvents(data.Events, data.ChainID, data.Height, data.BlockTime)
    result.Accounts = convertAccounts(data.AccountsCurrent, data.Height, data.BlockTime)
    result.Pools = convertPools(data.PoolsCurrent, data.Height, data.BlockTime)
    result.Orders = convertOrders(data.Orders, data.Height, data.BlockTime)
    result.DexPrices = convertDexPrices(data.DexPrices, data.Height, data.BlockTime)
    result.Params = convertParams(data.Params, data.Height, data.BlockTime)
    result.Supply = convertSupply(data.Supply, data.Height, data.BlockTime)
    result.Committees = convertCommittees(data.Committees, data.Height, data.BlockTime)

    // Change detection conversions
    result.Validators, result.CommitteeValidators = convertValidatorsWithChangeDetection(
        data.ValidatorsCurrent, data.ValidatorsPrevious,
        data.Height, data.BlockTime,
    )
    result.ValidatorNonSigningInfo = convertNonSignersWithChangeDetection(
        data.NonSignersCurrent, data.NonSignersPrevious,
        data.Height, data.BlockTime,
    )
    result.ValidatorDoubleSigningInfo = convertDoubleSignersWithChangeDetection(
        data.DoubleSignersCurrent, data.DoubleSignersPrevious,
        data.Height, data.BlockTime,
    )
    result.PoolPointsByHolder = convertPoolPointsWithChangeDetection(
        data.PoolsCurrent, data.PoolsPrevious,
        data.Height, data.BlockTime,
    )

    // DEX state machine conversions
    result.DexOrders = convertDexOrdersWithStateMachine(data, data.Events)
    result.DexDeposits = convertDexDepositsWithStateMachine(data, data.Events)
    result.DexWithdrawals = convertDexWithdrawalsWithStateMachine(data, data.Events)

    return result
}
```

### Phase 4: Add canopyx Transactional Write Method
**File:** `internal/indexer/indexer.go`

Correct transaction handling using context-propagated transactions:

```go
func (idx *Indexer) writeWithCanopyx(ctx context.Context, data *CanopyxBlockData) error {
    return idx.db.Client.BeginFunc(ctx, func(tx pgx.Tx) error {
        // Embed transaction in context for all insert methods
        txCtx := idx.db.Client.WithTx(ctx, tx)

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
```

### Phase 5: Update Indexer Struct
**File:** `internal/indexer/indexer.go`

```go
type Indexer struct {
    rpcClients map[uint64]*rpc.HTTPClient
    db         *chain.DB       // Chain database for entity inserts
    adminDB    *admin.DB       // Admin database for index progress
}

func New(ctx context.Context, logger *zap.Logger, chainID uint64) (*Indexer, error) {
    // Initialize chain DB
    chainDB, err := chain.NewWithPoolConfig(ctx, logger, chainID,
        postgres.GetPoolConfigForComponent("indexer_chain"))
    if err != nil {
        return nil, fmt.Errorf("create chain db: %w", err)
    }

    // Initialize admin DB
    adminDB, err := admin.NewWithPoolConfig(ctx, logger,
        postgres.GetPoolConfigForComponent("indexer_admin"))
    if err != nil {
        return nil, fmt.Errorf("create admin db: %w", err)
    }

    return &Indexer{
        rpcClients: make(map[uint64]*rpc.HTTPClient),
        db:         chainDB,
        adminDB:    adminDB,
    }, nil
}
```

### Phase 6: Update CanopyxBlockData Structure
**File:** `internal/indexer/types.go`

```go
type CanopyxBlockData struct {
    // Metadata
    ChainID   uint64
    Height    uint64
    BlockTime time.Time

    // Core entities
    Block        *indexermodels.Block
    Transactions []*indexermodels.Transaction
    Events       []*indexermodels.Event
    Accounts     []*indexermodels.Account
    Params       *indexermodels.Params
    Supply       *indexermodels.Supply

    // Validators (change-detected)
    Validators                  []*indexermodels.Validator
    ValidatorNonSigningInfo     []*indexermodels.ValidatorNonSigningInfo
    ValidatorDoubleSigningInfo  []*indexermodels.ValidatorDoubleSigningInfo

    // Pools (change-detected for holders)
    Pools              []*indexermodels.Pool
    PoolPointsByHolder []*indexermodels.PoolPointsByHolder

    // Orders
    Orders []*indexermodels.Order

    // DEX (state machine)
    DexPrices      []*indexermodels.DexPrice
    DexOrders      []*indexermodels.DexOrder
    DexDeposits    []*indexermodels.DexDeposit
    DexWithdrawals []*indexermodels.DexWithdrawal

    // Committees
    Committees          []*indexermodels.Committee
    CommitteeValidators []*indexermodels.CommitteeValidator
    CommitteePayments   []*indexermodels.CommitteePayment

    // Block summary (computed after conversion)
    BlockSummary *indexermodels.BlockSummary
}
```

### Phase 7: Add Block Summary Builder
**File:** `internal/indexer/summary.go`

```go
// buildBlockSummary computes and inserts the block summary from converted data
func (idx *Indexer) buildBlockSummary(ctx context.Context, data *CanopyxBlockData) error {
    summary := &indexermodels.BlockSummary{
        Height:            data.Height,
        HeightTime:        data.BlockTime,
        TotalTransactions: data.Block.TotalTxs,
        NumTxs:            int64(len(data.Transactions)),
        NumAccounts:       int64(len(data.Accounts)),
        NumEvents:         int64(len(data.Events)),
        NumValidators:     int64(len(data.Validators)),
        NumPools:          int64(len(data.Pools)),
        NumOrders:         int64(len(data.Orders)),
        NumDexPrices:      int64(len(data.DexPrices)),
        NumDexOrders:      int64(len(data.DexOrders)),
        NumDexDeposits:    int64(len(data.DexDeposits)),
        NumDexWithdrawals: int64(len(data.DexWithdrawals)),
        NumCommittees:     int64(len(data.Committees)),
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

    return idx.db.InsertBlockSummariesStaging(ctx, summary)
}
```

### Phase 8: Remove Legacy Files
Delete these files containing legacy SQL implementation:

```
internal/indexer/fetch.go           # Parallel RPC orchestration
internal/indexer/write.go           # Transaction wrapper
internal/indexer/accounts.go        # Account SQL inserts
internal/indexer/validators.go      # Validator SQL inserts (KEEP change detection logic)
internal/indexer/pools.go           # Pool SQL inserts (KEEP change detection logic)
internal/indexer/transactions.go    # Transaction SQL inserts
internal/indexer/events.go          # Event SQL inserts
internal/indexer/dex.go             # DEX SQL inserts (KEEP state machine logic)
internal/indexer/committees.go      # Committee SQL inserts
internal/indexer/orders.go          # Order SQL inserts
internal/indexer/params.go          # Params SQL inserts
internal/indexer/supply.go          # Supply SQL inserts
internal/indexer/block.go           # Block SQL inserts
```

**KEEP:** `pkg/transform/` - Continue using for data transformation utilities

### Phase 9: Update Imports
**Files to update:**
- `cmd/indexer/main.go`
- `cmd/backfill/main.go`
- `internal/backfill/backfill.go`

**Import changes:**
```go
import (
    "github.com/canopy-network/canopyx/pkg/db/postgres"
    "github.com/canopy-network/canopyx/pkg/db/postgres/chain"
    "github.com/canopy-network/canopyx/pkg/db/postgres/admin"
    indexermodels "github.com/canopy-network/canopyx/pkg/db/models/indexer"
)
```

---

## Benefits

### Performance Improvements
- **20x fewer HTTP requests**: 1 blob call vs 20+ individual RPC calls
- **Reduced network latency**: Single round trip instead of parallel orchestration
- **Better compression**: Protobuf encoding of complete dataset
- **Eliminated coordination overhead**: No errgroup parallel fetch management

### Reliability Improvements
- **Atomic data fetching**: All-or-nothing blob retrieval
- **ACID transactions**: All writes in single transaction with proper rollback
- **Simplified error handling**: Single point of failure instead of 20+ RPC calls

### Maintenance Improvements
- **Simpler code**: Remove ~1000 lines of custom SQL logic
- **Preserved logic**: Change detection and DEX state machine retained
- **Battle-tested inserts**: canopyx provides production-ready database operations

---

## Error Handling

- **Blob fetch failure**: Log error and return (stops indexing for that height)
- **Conversion failure**: Return error (data format compatibility issue)
- **Database failure**: Transaction rollback ensures no partial writes
- **Progress tracking**: Uses `admin.DB.RecordIndexed()` with `GREATEST` to prevent regression

---

## Files Changed

### Modified Files
- `internal/indexer/indexer.go` - Core indexing logic with retry
- `internal/indexer/types.go` - Add CanopyxBlockData structure

### New Files
- `internal/indexer/conversions.go` - Data conversion with change detection
- `internal/indexer/summary.go` - Block summary builder

### Deleted Files
- `internal/indexer/fetch.go`
- `internal/indexer/write.go`
- `internal/indexer/accounts.go`
- `internal/indexer/transactions.go`
- `internal/indexer/events.go`
- `internal/indexer/committees.go`
- `internal/indexer/orders.go`
- `internal/indexer/params.go`
- `internal/indexer/supply.go`
- `internal/indexer/block.go`

### Refactored Files (logic moved to conversions.go)
- `internal/indexer/validators.go` → `convertValidatorsWithChangeDetection()`
- `internal/indexer/pools.go` → `convertPoolPointsWithChangeDetection()`
- `internal/indexer/dex.go` → `convertDex*WithStateMachine()`

### Preserved
- `pkg/transform/` - Data transformation utilities (unchanged)

---

## Risk Assessment

### Identified Risks
- **Blob endpoint dependency**: If unavailable, indexing stops
- **Data format changes**: canopyx model changes could break conversion layer

### Mitigation
- **Comprehensive conversion tests**: Ensure all data types handled correctly
- **Staged deployment**: Test on development chains before production
- **Monitoring**: Implement alerts for blob fetch failures

---

## Success Criteria

- ✅ **Functionality preserved**: Same indexing behavior and data consistency
- ✅ **Change detection preserved**: Snapshot-on-change logic retained
- ✅ **DEX state machine preserved**: State transitions correctly tracked
- ✅ **Performance improved**: Fewer RPC calls, simpler error handling
- ✅ **Code simplified**: Remove ~1000 lines of custom SQL logic
- ✅ **ACID compliance**: Proper transaction handling with context propagation

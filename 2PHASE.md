# Plan: Two-Phase Indexer Refactoring

## Goal
Refactor `IndexBlock` from 12 parallel indexers (each doing RPC + DB interleaved) to a two-phase architecture:
1. **Phase 1 (Fetch)**: All RPC calls run in parallel; any failure → return error (NACK)
2. **Phase 2 (Write)**: All DB writes in a single PostgreSQL transaction; any failure → rollback + return error (NACK)

## Key Decision
- DEX batch currently reads from `events` table mid-processing
- **Solution**: Pass events in-memory from RPC fetch (no DB query)

---

## Files to Create

### 1. `internal/indexer/types.go` (new)
Define `BlockData` struct to hold all RPC-fetched data:
```go
type BlockData struct {
    ChainID, Height uint64
    BlockTime       time.Time
    Block           *lib.BlockResult

    // Simple fetches
    Txs, Events, Accounts, Orders, DexPrices, Params, Supply, Committees

    // Shared (validators + committees)
    Subsidized, Retired []uint64

    // Change detection pairs (current + H-1)
    ValidatorsCurrent, ValidatorsPrevious
    PoolsCurrent, PoolsPrevious
    DexBatches (4 variants: current, next, H-1 current, H-1 next)
    NonSigners, DoubleSigners (current + H-1)
}
```

### 2. `internal/indexer/fetch.go` (new)
`fetchAllData()` - runs ~20 RPC calls in parallel via errgroup:
- Single calls: Txs, Events, Accounts, Orders, DexPrices, Params, Supply, Committees
- Shared: SubsidizedCommittees, RetiredCommittees (used by both validators + committees)
- Pairs: Validators, Pools, NonSigners, DoubleSigners (current + H-1)
- DEX: AllDexBatches, AllNextDexBatches (current + H-1)

### 3. `internal/indexer/write.go` (new)
`writeAllData()` - single transaction:
```go
func (idx *Indexer) writeAllData(ctx context.Context, data *BlockData) error {
    return pgx.BeginFunc(ctx, idx.db, func(tx pgx.Tx) error {
        batch := &pgx.Batch{}
        idx.writeBlock(batch, data)
        idx.writeTransactions(batch, data)
        // ... all 12 write functions
        br := tx.SendBatch(ctx, batch)
        // exec all, rollback on any error
    })
}
```

---

## Files to Modify

### 4. `internal/indexer/indexer.go`
Replace current implementation:
```go
// OLD: errgroup running 12 indexers in parallel
// NEW:
func (idx *Indexer) IndexBlock(ctx, chainID, height) error {
    block := idx.fetchBlockWithRetry(ctx, height)
    data := idx.fetchAllData(ctx, chainID, height, block, blockTime)  // Phase 1
    idx.writeAllData(ctx, data)                                        // Phase 2
    idx.buildBlockSummary(...)
    idx.updateProgress(...)
}
```
- Remove `runIndexer()` helper function

### 5. `internal/indexer/dex.go` (most complex)
- Remove `queryDexEvents()` (DB query)
- Add `parseDexEventsFromSlice(events []*lib.Event)` - parse in-memory
- Rename `indexDexPrices` → `writeDexPrices(batch, data)`
- Rename `indexDexBatch` → `writeDexBatch(batch, data)` - use `data.Events` instead of DB

### 6. `internal/indexer/validators.go` (complex)
- Remove 8 RPC calls (ValidatorsByHeight x2, NonSigners x2, DoubleSigners x2, Subsidized, Retired)
- Use data from `BlockData` (Subsidized/Retired shared with committees)
- Consolidate `indexNonSigners`, `indexDoubleSigners` into `writeValidators(batch, data)`

### 7. Simple refactors (same pattern for each):
| File | Change |
|------|--------|
| `block.go` | `saveBlock` → `writeBlock(batch, data)` |
| `transactions.go` | Remove RPC, `writeTransactions(batch, data)` |
| `events.go` | Remove RPC, `writeEvents(batch, data)` |
| `accounts.go` | Remove RPC, `writeAccounts(batch, data)` |
| `pools.go` | Remove 2 RPC, `writePools(batch, data)` |
| `orders.go` | Remove RPC, `writeOrders(batch, data)` |
| `params.go` | Remove RPC, `writeParams(batch, data)` |
| `supply.go` | Remove RPC, `writeSupply(batch, data)` |
| `committees.go` | Remove 3 RPC, use shared Subsidized/Retired, `writeCommittees(batch, data)` |

---

## Implementation Order

1. Create `types.go` with `BlockData` struct
2. Create `fetch.go` with `fetchAllData()`
3. Create `write.go` with `writeAllData()` skeleton
4. Refactor simple indexers first (block, transactions, events, accounts, orders, params, supply)
5. Refactor pools (has H-1 comparison)
6. Refactor committees (uses shared Subsidized/Retired)
7. Refactor validators (complex: 8 RPC calls, nested functions)
8. Refactor dex.go (complex: in-memory event parsing)
9. Update `indexer.go` to use two-phase approach
10. Test

---

## Benefits
- **Atomic writes**: All-or-nothing DB operations
- **Clear error semantics**: RPC fail OR DB fail → NACK → redelivery
- **No DB dependency**: DEX batch uses in-memory events
- **RPC deduplication**: Subsidized/Retired called once (was twice)

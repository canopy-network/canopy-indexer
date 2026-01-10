# Postgres Implementation Status

## Overview

The Postgres backend implementation has been initiated with schema creation and interface stubs. However, **the table schemas need to be updated to match the actual ClickHouse model structures** before query methods can be properly implemented.

## Current Status

### ✅ Completed

1. **Admin Store** (`pkg/db/postgres/admin/`)
   - ✅ Schema: chains, index_progress, rpc_endpoints, reindex_requests
   - ✅ All interface methods implemented
   - ✅ Fixed index_progress from multi-row to single-row-per-chain design
   - **Status**: Production ready

2. **Connection Infrastructure** (`pkg/db/postgres/`)
   - ✅ Client wrapper for pgxpool
   - ✅ Connection pooling configuration
   - ✅ Transaction support (BeginFunc, SendBatch)
   - **Status**: Production ready

3. **CrossChain Store - Schema** (`pkg/db/postgres/crosschain/`)
   - ✅ 22 tables created (matching 001_initial.sql)
   - ✅ 11 views for latest state (using DISTINCT ON)
   - ✅ 2 PL/pgSQL functions (update_index_progress, build_block_summary)
   - ✅ 5 enum types
   - **Status**: Schema complete, query methods need implementation

### ⚠️ Incomplete - Requires Schema Updates

4. **Chain Store** (`pkg/db/postgres/chain/`)
   - ⚠️ **Problem**: Simplified schemas don't match ClickHouse models
   - ⚠️ **Blocker**: Cannot implement query methods until schemas are updated
   - **Status**: Schemas need rework

## Schema Mismatch Examples

### Transaction Table

**Current Postgres Schema** (simplified):
```sql
CREATE TABLE txs (
    height BIGINT,
    tx_hash TEXT,
    tx_index INTEGER,
    tx_type TEXT,
    sender TEXT,
    recipient TEXT,
    amount BIGINT,
    fee BIGINT,
    memo TEXT,
    time TIMESTAMP WITH TIME ZONE,
    msg JSONB
);
```

**Actual ClickHouse Model** (`indexer.Transaction`):
- 30+ fields including:
  - TxHash, TxIndex, Time, HeightTime, CreatedHeight, NetworkID
  - MessageType, Signer, Amount, Fee, Memo
  - ValidatorAddress, Commission
  - ChainID, SellAmount, BuyAmount, LiquidityAmt, LiquidityPercent
  - OrderID, Price
  - ParamKey, ParamValue
  - CommitteeID, Recipient, PollHash
  - BuyerReceiveAddress, BuyerSendAddress, BuyerChainDeadline
  - Msg, PublicKey, Signature

### Event Table

**Current Postgres Schema** (simplified):
```sql
CREATE TABLE events (
    height BIGINT,
    event_type TEXT,
    event_index INTEGER,
    address TEXT,
    amount BIGINT,
    data TEXT,
    time TIMESTAMP WITH TIME ZONE,
    msg JSONB
);
```

**Actual ClickHouse Model** (`indexer.Event`):
- 20+ fields including:
  - Height, ChainID, Address, Reference, EventType, BlockHeight
  - Amount, SoldAmount, BoughtAmount, LocalAmount, RemoteAmount
  - Success, LocalOrigin, OrderID
  - PointsReceived, PointsBurned, Data
  - SellerReceiveAddress, BuyerSendAddress, SellersSendAddress
  - Msg, HeightTime

### Block Summary Table

**Current Postgres Schema** (minimal):
```sql
CREATE TABLE block_summaries (
    height BIGINT PRIMARY KEY,
    time TIMESTAMP WITH TIME ZONE,
    num_txs INTEGER,
    num_events INTEGER,
    num_accounts INTEGER,
    num_validators INTEGER
);
```

**Actual ClickHouse Model** (`indexer.BlockSummary`):
- **90+ fields** including:
  - Transaction counters (20 fields): NumTxsSend, NumTxsStake, NumTxsUnstake, etc.
  - Account counters (2 fields): NumAccounts, NumAccountsNew
  - Event counters (10 fields): NumEventsReward, NumEventsSlash, etc.
  - Order counters (5 fields): NumOrders, NumOrdersOpen, etc.
  - Pool counters (2 fields): NumPools, NumPoolsNew
  - DEX counters (15 fields): NumDexOrders, NumDexDeposits, etc.
  - Validator counters (8 fields): NumValidators, NumValidatorsActive, etc.
  - Committee counters (6 fields): NumCommittees, NumCommitteeValidators, etc.
  - Supply metrics (4 fields): SupplyTotal, SupplyStaked, etc.

## Required Work

### Phase 1: Update Chain Table Schemas

For each table in `pkg/db/postgres/chain/`, update to match ClickHouse models:

1. **blocks** → Match `indexer.Block` (14 fields)
2. **txs** → Match `indexer.Transaction` (30+ fields)
3. **events** → Match `indexer.Event` (20+ fields)
4. **accounts** → Match `indexer.Account`
5. **block_summaries** → Match `indexer.BlockSummary` (90+ fields)
6. **validators** → Match `indexer.Validator`
7. **validator_non_signing_info** → Match `indexer.ValidatorNonSigningInfo`
8. **validator_double_signing_info** → Match `indexer.ValidatorDoubleSigningInfo`
9. **pools** → Match `indexer.Pool`
10. **pool_points_by_holder** → Match `indexer.PoolPointsByHolder`
11. **orders** → Match `indexer.Order`
12. **dex_orders** → Match `indexer.DexOrder`
13. **dex_deposits** → Match `indexer.DexDeposit`
14. **dex_withdrawals** → Match `indexer.DexWithdrawal`
15. **dex_prices** → Match `indexer.DexPrice`
16. **committees** → Match `indexer.Committee`
17. **committee_validators** → Match `indexer.CommitteeValidator`
18. **committee_payments** → Match `indexer.CommitteePayment`
19. **params** → Match `indexer.Params`
20. **supply** → Match `indexer.Supply`
21. **poll_snapshots** → Match `indexer.PollSnapshot`
22. **proposal_snapshots** → Match `indexer.ProposalSnapshot`

### Phase 2: Implement Insert Methods

After schemas match models, implement insert methods for each entity:
- `insertBlock`, `insertTransactions`, `insertEvents`, `insertAccounts`, etc.
- Use batch operations (`pgx.Batch`) for arrays
- Use `ON CONFLICT DO UPDATE` for upserts (idempotent writes)

### Phase 3: Implement Query Methods

Implement all `chain.Store` interface methods:
- `GetBlock`, `GetBlockTime`, `GetBlockSummary`, `HasBlock`
- `GetEventsByTypeAndHeight`, `GetRewardSlashEvents`, etc.
- `GetHighestBlockBeforeTime`
- Event query methods (GetValidatorLifecycleEvents, GetOrderBookSwapEvents, etc.)

### Phase 4: Implement Delete Methods

- `DeleteBlock`, `DeleteTransactions`
- Use transactions to ensure atomicity

### Phase 5: Implement CrossChain Query Methods

Implement all `crosschain.Store` interface methods:
- `QueryLatestAccounts`, `QueryLatestValidators`, `QueryLatestPools`, etc.
- `QueryLPPositionSnapshots`, `GetLatestLPPositionSnapshot`
- `SetupChainSync`, `GetSyncStatus`, `ResyncTable`, `ResyncChain`

## Recommended Approach

### Option 1: Automated Schema Generation

Create a tool to auto-generate Postgres table schemas from the ClickHouse model definitions:

```go
// Read model struct tags and field types
// Generate CREATE TABLE statements with proper type mappings
// Preserve the exact field structure
```

### Option 2: Manual Schema Update

Manually update each table file to match the model structures:

1. Read the model file (e.g., `pkg/db/models/indexer/tx.go`)
2. Update the corresponding Postgres file (e.g., `pkg/db/postgres/chain/transaction.go`)
3. Add all fields with correct Postgres types
4. Maintain indexes for queryable fields

### Option 3: Incremental Implementation

Focus on the most critical tables first:
1. blocks, txs, events (core blockchain data)
2. accounts, validators (essential state)
3. block_summaries (analytics)
4. Everything else (DEX, committees, pools, etc.)

## Type Mapping Reference

| ClickHouse | Postgres | Notes |
|-----------|----------|-------|
| UInt64 | BIGINT | |
| UInt32 | INTEGER | |
| UInt16 | SMALLINT | |
| UInt8 | SMALLINT | Postgres has no TINYINT |
| String | TEXT | |
| LowCardinality(String) | TEXT | Use ENUM if limited values |
| DateTime64(6) | TIMESTAMP WITH TIME ZONE | Microsecond precision |
| DateTime | TIMESTAMP | |
| Bool | BOOLEAN | |
| Float64 | DOUBLE PRECISION | |
| Array(String) | TEXT[] | |

## Files to Update

### Chain Store
- `pkg/db/postgres/chain/block.go` - Update blocks + block_summaries schemas
- `pkg/db/postgres/chain/transaction.go` - Update txs schema
- `pkg/db/postgres/chain/event.go` - Update events schema
- `pkg/db/postgres/chain/account.go` - Update accounts schema
- `pkg/db/postgres/chain/validator.go` - Update validator schemas
- `pkg/db/postgres/chain/pool.go` - Update pool schemas
- `pkg/db/postgres/chain/order.go` - Update orders schema
- `pkg/db/postgres/chain/dex.go` - Update DEX schemas
- `pkg/db/postgres/chain/committee.go` - Update committee schemas
- `pkg/db/postgres/chain/param.go` - Update params schema
- `pkg/db/postgres/chain/supply.go` - Update supply schema
- `pkg/db/postgres/chain/poll.go` - Update poll/proposal schemas

### Then Create
- `pkg/db/postgres/chain/inserts.go` - All insert methods
- `pkg/db/postgres/chain/queries.go` - All query methods
- `pkg/db/postgres/chain/deletes.go` - All delete methods

### CrossChain Store
- `pkg/db/postgres/crosschain/queries.go` - Implement all query methods
- `pkg/db/postgres/crosschain/sync.go` - Implement sync methods

## Estimate

- **Schema updates**: 2-3 days (22 tables, ensuring accuracy)
- **Insert methods**: 2-3 days (batch operations, upserts)
- **Query methods**: 3-4 days (30+ methods, some complex)
- **CrossChain queries**: 2-3 days (entity queries + sync logic)
- **Testing**: 2-3 days (integration tests)

**Total**: ~2 weeks of focused development

## Current Workaround

For immediate use, the admin store is fully functional. The chain and crosschain stores can be used with:
- Schema creation (tables/views/functions are created correctly)
- Direct SQL queries via the `Exec` and `Select` methods
- Manual batch operations via `SendBatch`

The stub methods return `ErrNotImplemented` which allows the code to compile and provides clear error messages when unimplemented methods are called.

## Next Steps

1. **Decision**: Choose implementation approach (automated, manual, or incremental)
2. **Schema Update**: Start with core tables (blocks, txs, events, accounts)
3. **Implement Methods**: Add insert/query/delete methods incrementally
4. **Test**: Add integration tests for each completed table
5. **Document**: Update this file as tables are completed

## Questions?

- Should we prioritize certain tables over others?
- Would automated schema generation from models be valuable?
- Is there a subset of functionality needed urgently?

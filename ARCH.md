# Architecture

## WebSocket Listener

Subscribes to Canopy node for real-time block notifications:

```
Canopy Node ──WebSocket──→ Listener ──→ Publisher ──→ Redis Stream
```

- Connects to `/v1/subscribe-rc-info?chainId=N`
- Receives protobuf `RootChainInfo` messages with new block heights
- Auto-reconnects with linear backoff (configurable max retries)
- On new block: publishes `(chainID, height)` to Redis Stream

## Watermill / Redis Streams

Decouples block detection from indexing via message queue:

- **Publisher**: Encodes `chainID + height` as 16-byte payload, publishes to Redis Stream
- **Worker**: Subscribes via Watermill consumer group, calls `IndexBlock()` for each message
- **Consumer Groups**: Multiple workers share load; failed messages are redelivered (NACK)
- **At-least-once**: Messages retry until ACK'd after successful indexing

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Listener   │────▶│ Redis Stream │────▶│   Worker    │
│ (WebSocket) │     │  (Watermill) │     │ (IndexBlock)│
└─────────────┘     └──────────────┘     └─────────────┘
```

## Two-Phase Indexing

Each block is indexed in two distinct phases:

1. **Fetch Phase** - All RPC calls execute in parallel via `errgroup`. Any failure causes immediate error return (NACK), triggering retry.

2. **Write Phase** - All database writes execute in a single atomic transaction. Any failure causes full rollback (NACK).

This separation ensures retries are efficient: cached RPC responses avoid redundant network calls on retry.

## Concurrency

- **RPC Fetching**: ~20 goroutines fetch block data, transactions, validators, pools, DEX state, etc. in parallel
- **Workers**: Redis Streams consumer group enables multiple worker instances to share load
- **Backfill**: Configurable concurrency limit via `errgroup.SetLimit()`

## PostgreSQL Transactions

All writes for a single block are batched into one transaction using `pgx.BeginFunc`:

```
BEGIN
  → write block
  → write transactions
  → write events
  → write accounts
  → write validators
  → write pools
  → write orders
  → write dex prices
  → write dex batches
  → write params
  → write supply
  → write committees
COMMIT (or ROLLBACK on any error)
```

Uses `pgx.Batch` for efficient pipelining of INSERT statements.

## Caching

The RPC client maintains a rolling cache for recent heights:

- Caches responses keyed by `path:height`
- Primarily benefits H-1 lookups (previous height for change detection)
- On retry, cached data returns immediately without network call
- Also includes: token-bucket rate limiting + circuit breaker per endpoint

## Backfill

Fills gaps in indexed data:

1. Query DB to find missing block heights in range
2. Process missing heights in batches with bounded concurrency
3. Each block uses the same two-phase `IndexBlock()` path
4. Failures are logged but don't halt the backfill
5. Progress reported at configurable intervals

## Transform Layer

The `pkg/transform` package converts RPC responses (protobuf) to database models:

- **Validators**: Derives status (`active`/`paused`/`unstaking`) from height fields
- **Pools**: Extracts chain ID from pool ID encoding, categorizes pool type (reward/holding/liquidity/escrow)
- **DEX**: Parses event JSON to correlate swap results with orders
- **Common**: Hex encoding for addresses/keys, timestamp conversion

## Change Detection (Snapshot-on-Change)

To avoid writing redundant rows, the indexer compares state at height H vs H-1:

```
Fetch H-1 data ──┬──▶ Build lookup maps (address → state)
Fetch H data ────┴──▶ Compare each item → only INSERT if changed
```

**Validators**: Snapshot if stake, status, committees, output, or flags changed
**Pools**: Snapshot if amount, total points, or holder points changed
**Non-Signers**: Snapshot if counter changed; detect resets (existed at H-1, gone at H)
**Double-Signers**: Snapshot if evidence count changed

This reduces write volume significantly for stable state.

## DEX State Machine

DEX orders, deposits, and withdrawals follow a lifecycle tracked via state transitions:

```
┌─────────┐     ┌─────────┐     ┌──────────┐
│ FUTURE  │────▶│ LOCKED  │────▶│ COMPLETE │
│ (next)  │     │ (curr)  │     │ (event)  │
└─────────┘     └─────────┘     └──────────┘
```

- **FUTURE/PENDING**: Item appears in "next batch" (queued for next block)
- **LOCKED**: Item moves to "current batch" (being processed)
- **COMPLETE**: Completion event emitted with results (success, amounts)

Processing correlates DEX events with batch data:
1. Parse `dex-swap`, `dex-liquidity-deposit`, `dex-liquidity-withdraw` events
2. Build H-1 maps from previous block's current/next batches
3. Match events to orders → write COMPLETE state with results
4. Apply change detection for LOCKED/PENDING items

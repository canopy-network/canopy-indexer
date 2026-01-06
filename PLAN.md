# pgindexer - Canopy Blockchain Indexer

A simplified blockchain indexer for Canopy, replacing Temporal workflows with Watermill + Redis Streams.

## Goals

- Remove Temporal dependency (complex, requires separate server)
- Use WebSocket subscription for real-time block notifications
- Use Watermill + Redis Streams for job queue
- Use PostgreSQL for indexed data storage
- Keep existing RPC client code from canopyx

## Architecture

```
┌─────────────────┐     WebSocket      ┌──────────────────┐
│  Canopy Node    │ ──────────────────▶│  Block Listener  │
└─────────────────┘  (RootChainInfo)   └────────┬─────────┘
                                                │
                                                │ Publish (height)
                                                ▼
                                       ┌──────────────────┐
                                       │  Redis Streams   │
                                       │ "blocks-to-index"│
                                       └────────┬─────────┘
                                                │
                              ┌─────────────────┼─────────────────┐
                              │                 │                 │
                              ▼                 ▼                 ▼
                       ┌──────────┐      ┌──────────┐      ┌──────────┐
                       │ Worker 1 │      │ Worker 2 │      │ Worker N │
                       └────┬─────┘      └────┬─────┘      └────┬─────┘
                            │                 │                 │
                            │ RPC calls (parallel via errgroup) │
                            ▼                 ▼                 ▼
                       ┌─────────────────────────────────────────────┐
                       │                  PostgreSQL                 │
                       └─────────────────────────────────────────────┘
```

## Components

### 1. WebSocket Listener

Subscribes to Canopy node WebSocket for new block notifications.

- **Endpoint**: `wss://<node>/rpc/v1/subscribe/root-chain-info?chainId=<id>`
- **Message format**: Protobuf `lib.RootChainInfo` with `RootChainId` and `Height`
- **Reference**: `/home/enielson/canopy/launchpad/internal/websocket/canopy_consumer.go`

Features:
- Auto-reconnect with linear backoff
- Configurable max retries (default: 25)
- Publishes heights to Redis on new block

### 2. Publisher

Publishes block heights to Redis Streams using Watermill.

- **Topic**: `blocks-to-index`
- **Payload**: uint64 height (8 bytes, big-endian)

### 3. Worker

Consumes block heights from Redis Streams and indexes them.

- **Consumer group**: `indexer-workers` (allows multiple workers)
- **Retry**: Automatic redelivery on failure (no `msg.Ack()`)
- **Parallelism**: Uses `errgroup` for concurrent RPC calls within each block

### 4. Indexer

Core indexing logic - makes RPC calls and writes to PostgreSQL.

All RPC calls run in parallel (no dependencies between them):
- `BlockByHeight()` → blocks table
- `TxsByHeight()` → txs table
- `EventsByHeight()` → events table
- `AccountsByHeight()` → accounts table
- `ValidatorsByHeight()` → validators table
- `PoolsByHeight()` → pools table
- `OrdersByHeight()` → orders table
- `DexPricesByHeight()` → dex_prices table
- `AllDexBatchesByHeight()` → dex_orders, dex_deposits, dex_withdrawals tables
- `SupplyByHeight()` → supply table
- `AllParamsByHeight()` → params table
- `CommitteesDataByHeight()` → committees table
- `NonSignersByHeight()` → validator_non_signing_info table
- `DoubleSignersByHeight()` → validator_double_signing_info table

## RPC Client

Reuse from canopyx (`pkg/rpc/`):
- `HTTPClient` with circuit breaker and token-bucket rate limiting
- Default: 500 RPS, 1000 burst
- Endpoint failover with health tracking

## Dependencies

```
github.com/ThreeDotsLabs/watermill
github.com/ThreeDotsLabs/watermill-redisstream
github.com/redis/go-redis/v9
github.com/jackc/pgx/v5
github.com/gorilla/websocket
github.com/canopy-network/canopy/lib  (protobuf types)
golang.org/x/sync/errgroup
```

## Project Structure

```
pgindexer/
├── cmd/
│   └── indexer/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration
│   ├── listener/
│   │   └── websocket.go      # WebSocket subscription
│   ├── publisher/
│   │   └── publisher.go      # Redis Streams publisher
│   ├── worker/
│   │   └── worker.go         # Watermill consumer
│   └── indexer/
│       ├── indexer.go        # Core indexing orchestration
│       ├── block.go          # Block indexing
│       ├── transactions.go   # Transaction indexing
│       ├── events.go         # Event indexing
│       ├── accounts.go       # Account indexing
│       ├── validators.go     # Validator indexing
│       ├── pools.go          # Pool indexing
│       ├── orders.go         # Order indexing
│       ├── dex.go            # DEX order/deposit/withdrawal indexing
│       ├── params.go         # Params indexing
│       ├── supply.go         # Supply indexing
│       └── committees.go     # Committee indexing
├── pkg/
│   └── rpc/                  # Copy from canopyx/pkg/rpc
│       ├── interfaces.go
│       ├── httpclient.go
│       └── paths.go
├── migrations/               # PostgreSQL migrations
│   └── 001_initial.sql
├── go.mod
├── go.sum
├── Makefile
└── PLAN.md
```

## Database Schema

Target tables (based on canopyx cross-chain schema):

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `blocks` | Block headers | height, hash, time, proposer |
| `txs` | Transactions | height, tx_hash, message_type, signer |
| `events` | All events | height, event_type, address, amount |
| `accounts` | Account balances | address, amount, height |
| `validators` | Validator state | address, staked_amount, status |
| `committees` | Committee state | committee_id, subsidized, retired |
| `pools` | Liquidity pools | pool_id, amount, total_points |
| `orders` | Order book | order_id, status, amount_for_sale |
| `dex_orders` | DEX limit orders | order_id, state, success |
| `dex_deposits` | Liquidity deposits | order_id, state, amount |
| `dex_withdrawals` | Liquidity withdrawals | order_id, state, percent |
| `dex_prices` | Price snapshots | height, price_e6 |
| `params` | Governance params | height, all fee columns |
| `supply` | Token supply | height, total, staked |
| `index_progress` | Indexing progress | chain_id, last_height |

## Key Decisions

### Why not index events first?

The original canopyx code had a comment "save events first so the other indexing process can list and filter by type" but no indexer actually queries the events table. This was dead code - all indexers can run in parallel.

### Why Watermill + Redis?

- Simpler than Temporal (no separate server)
- Persistent queue survives restarts
- Consumer groups enable multiple workers
- Automatic redelivery on failure

### Why PostgreSQL over ClickHouse?

- Simpler operational model
- Better for transactional writes
- Standard SQL tooling
- Can add ClickHouse later for analytics if needed

## TODO

- [ ] Initialize Go module
- [ ] Copy RPC client from canopyx
- [ ] Create PostgreSQL migrations
- [ ] Implement WebSocket listener
- [ ] Implement Watermill publisher/worker
- [ ] Implement indexer with parallel RPC calls
- [ ] Add backfill command for historical blocks
- [ ] Add health check endpoint
- [ ] Add metrics (Prometheus)
- [ ] Docker/Kubernetes deployment

## References

- canopyx indexer: `/home/enielson/canopy/canopyx/app/indexer/`
- WebSocket consumer: `/home/enielson/canopy/launchpad/internal/websocket/canopy_consumer.go`
- RPC client: `/home/enielson/canopy/canopyx/pkg/rpc/`
- ClickHouse schema: `/home/enielson/canopy/launchpad/.claude/skills/clickhouse/references/clickhouse_tables.md`

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Build
make build              # Build indexer binary
make build-backfill     # Build backfill binary
make build-all          # Build both

# Run
make run                # Build and run indexer

# Test
make test               # Run all tests
go test -v ./internal/indexer/...  # Run tests for specific package

# Code Quality
make fmt                # Format code (go fmt + goimports)
make lint               # Run golangci-lint

# Database
make migrate            # Run migrations (local postgres on 5434)
make db-reset           # Drop and recreate schema
make show-progress      # View index_progress table
make show-stats         # View row counts per table

# Backfill
make backfill           # Run backfill for all chains
make backfill-stats     # Show gap statistics only
./bin/backfill -chain 1000 -stats  # Stats for specific chain

# Local Development
make tilt               # Start Tilt environment (port 10370)
make tilt-down          # Stop Tilt
```

## Architecture

### Binaries

- **indexer** (`cmd/indexer`): Main daemon for real-time block indexing via WebSocket
- **backfill** (`cmd/backfill`): CLI for filling gaps in indexed history via HTTP RPC

### Data Flow

```
WebSocket Blob Mode (primary):
  Canopy Node → BlobListener → IndexBlockWithData() → PostgreSQL

Height Notification Mode (legacy):
  Canopy Node → Listener → Publisher → Redis → Worker → IndexBlock() → PostgreSQL
```

### Package Structure

**internal/**
| Package | Purpose |
|---------|---------|
| `indexer` | Core indexing: chain management, two-phase indexing (fetch + write) |
| `listener` | WebSocket clients: `Listener` (height-only) and `BlobListener` (full data) |
| `publisher` | Redis Streams publisher for (chainID, height) tuples |
| `worker` | Redis consumer that dequeues and indexes blocks |
| `backfill` | Gap detection and concurrent block backfilling |
| `config` | Environment-based configuration |
| `api` | HTTP REST API for chain management |

**pkg/**
| Package | Purpose |
|---------|---------|
| `pkg/rpc` | HTTP client with rate limiting, circuit breaker, response caching |
| `pkg/blob` | Decode protobuf IndexerBlobs to typed BlockData |
| `pkg/transform` | Convert RPC responses to database models |
| `pkg/db/postgres` | Connection pool management |
| `pkg/db/postgres/admin` | Admin DB: chains, RPC endpoints, index progress |
| `pkg/db/postgres/chain` | Per-chain DB: 22 entity insert methods |
| `pkg/db/entities` | Type-safe entity constants |

### Database Design

- **Admin database**: Shared metadata (chains, rpc_endpoints, index_progress)
- **Per-chain databases**: Named `chain_{id}`, contains blocks, transactions, events, accounts, validators, pools, orders, etc.

### Two-Phase Indexing

1. **Fetch Phase**: Parallel RPC calls (~20 goroutines) for block data + H-1 data for change detection
2. **Write Phase**: Single atomic transaction inserting all entities in dependency order

Benefits: RPC cache on retry, atomic consistency per block.

### Key Patterns

- **Change Detection**: Compare H vs H-1 state, only write changed entities (validators, pools, non-signers)
- **Chain Rediscovery**: Periodic admin DB poll adds/removes chains dynamically
- **WebSocket Auto-Subscribe**: When `WSBlobAutoSubscribe=true`, creates per-chain BlobListener on discovery

### Key Files

| File | Purpose |
|------|---------|
| `internal/indexer/indexer.go` | Core: AddChain, IndexBlock, IndexBlockWithData, Rediscover |
| `internal/listener/blob.go` | WebSocket client receiving full IndexerBlobs |
| `pkg/blob/decoder.go` | Converts protobuf to BlockData struct |
| `pkg/db/postgres/chain/db.go` | Per-chain entity insert methods |
| `pkg/db/postgres/admin/db.go` | Admin DB operations |

## Environment Variables

Required: `POSTGRES_URL`, `REDIS_URL`

Key optional:
- `WS_BLOB_AUTO_SUBSCRIBE=true` - Enable per-chain WebSocket blob listeners
- `CHAIN_REDISCOVERY_INTERVAL=5s` - How often to poll for new chains
- `RPC_RPS=500`, `RPC_BURST=1000` - Rate limiting
- `WORKER_CONCURRENCY=1` - Redis consumer workers

## Local Testing

Mock RPC runs 100 chains (IDs 1000-1099) with 1000 blocks each. No WebSocket support, use backfill:

```bash
tilt up          # Start mock environment
make backfill    # Index mock blocks
```

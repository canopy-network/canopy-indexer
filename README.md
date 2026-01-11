# canopy-indexer

PostgreSQL indexer for the Canopy blockchain.

## Quick Start

```bash
# Start dependencies
docker-compose up -d postgres redis

# Build
make build

# Initialize database
make migrate

# Run (requires env vars)
make run
```

## Requirements

- Go 1.21+
- PostgreSQL
- Redis

## Database Setup

The indexer uses PostgreSQL with schema-based organization. The database must exist before running the indexer.

### Docker Compose (Recommended)

```bash
# Start PostgreSQL
docker-compose up -d postgres

# The database and initial schema will be created automatically
```

### Local PostgreSQL

```bash
# Create the indexer database
createdb -U canopy-indexer indexer

# Run initial migrations
make migrate
```

### Tilt Development

```bash
# Start Tilt (includes database setup)
tilt up

# Or run database initialization manually
tilt trigger db-init
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_URL` | Yes | - | PostgreSQL connection string |
| `REDIS_URL` | Yes | - | Redis connection string |
| `WS_ENABLED` | No | false | Enable WebSocket for real-time notifications |
| `WS_MAX_RETRIES` | No | 25 | Max WebSocket reconnection attempts |
| `WS_RECONNECT_DELAY` | No | 1s | Delay between reconnection attempts |
| `RPC_RPS` | No | 500 | RPC requests per second limit |
| `RPC_BURST` | No | 1000 | RPC burst size |
| `WORKER_CONCURRENCY` | No | 1 | Number of concurrent block workers |
| `LOG_LEVEL` | No | info | Log level (debug, info, warn, error) |

## Multi-Chain Configuration

Chain endpoints are hardcoded in `internal/config/config.go` via the `MockChains()` function. By default, 100 mock chains are configured:
- Chain IDs: 1000-1099
- Ports: 61000-61099
- URL format: `http://rpc-mock:{port}`

To modify the chain configuration, edit `MockChains()` in `internal/config/config.go`.

## Backfill

The backfill tool detects and fills gaps in indexed blocks.

```bash
# Check for gaps (all chains)
make backfill-stats

# Check specific chain
./bin/backfill -chain 1000 -stats

# Dry run (report only)
make backfill-dry-run

# Run backfill (all chains)
make backfill

# Run backfill for specific chain
./bin/backfill -chain 1000

# High-throughput mode
make backfill-fast
```

Enable periodic gap detection in the main indexer:
```bash
BACKFILL_CHECK_INTERVAL=5m ./bin/indexer
```

See [docs/backfill.md](docs/backfill.md) for full documentation.

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Lint
make lint

# Database setup
make migrate      # Run migrations (creates database if needed)
make psql         # Connect to database
make db-reset     # Reset all data

# Show indexing progress
make show-progress
```

## Mock RPC Nodes

For local testing, `canopy-rpc-mock` provides deterministic blockchain data without connecting to a real network.

```bash
# Start with Tilt (includes mock)
tilt up
```

The mock runs 100 chains (IDs 1000-1099) with 1000 pre-built blocks each on ports 61000-61099.

```bash
# Verify mock is running
curl http://localhost:61000/v1/query/height
# Returns: {"height":1000}

# Query different chains
curl http://localhost:60001/v1/query/height  # Chain 1001
curl http://localhost:60099/v1/query/height  # Chain 1099
```

The mock only supports HTTP (no WebSocket), so set `WS_ENABLED=false` and use the backfill worker for indexing:

```bash
# Run backfill for all mock chains
./bin/backfill
```

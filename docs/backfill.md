# Backfill

The backfill tool detects and fills gaps in indexed blocks. It uses efficient SQL queries to find missing block heights, then indexes them using the same logic as the main indexer.

## Quick Start

```bash
# Check for gaps (no indexing)
make backfill-stats

# Run backfill
make backfill
```

## CLI Usage

```bash
# Build the backfill tool
make build-backfill

# Show gap statistics
./bin/backfill -stats

# Dry run (report gaps without indexing)
./bin/backfill -dry-run

# Backfill all missing blocks
./bin/backfill

# Backfill a specific range
./bin/backfill -start 1000 -end 5000

# High-throughput mode
./bin/backfill -concurrency 50 -batch 5000
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-stats` | false | Only show gap statistics, don't index |
| `-dry-run` | false | Report gaps without indexing |
| `-start` | 1 | Start height for range |
| `-end` | (current) | End height (fetches from RPC if not set) |
| `-batch` | 1000 | Number of blocks per batch |
| `-concurrency` | 10 | Number of concurrent workers |

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build-backfill` | Build the backfill binary |
| `make backfill` | Run backfill (index all missing blocks) |
| `make backfill-stats` | Show gap statistics only |
| `make backfill-dry-run` | Dry run (report gaps without indexing) |
| `make backfill-range START=1000 END=5000` | Backfill a specific range |
| `make backfill-fast` | High-throughput backfill (50 workers) |
| `make show-gaps` | Show gaps via SQL query |

## Environment Variables

Environment variables can override CLI defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKFILL_BATCH_SIZE` | 1000 | Blocks per batch |
| `BACKFILL_CONCURRENCY` | 10 | Concurrent workers |
| `BACKFILL_START_HEIGHT` | 1 | Start height |
| `BACKFILL_END_HEIGHT` | (current) | End height (0 = fetch from RPC) |
| `BACKFILL_DRY_RUN` | false | Dry run mode |
| `BACKFILL_PROGRESS_INTERVAL` | 10s | Progress logging interval |

Example:
```bash
BACKFILL_CONCURRENCY=50 BACKFILL_BATCH_SIZE=5000 ./bin/backfill
```

## Periodic Health Check

The main indexer can run periodic gap checks. Enable with:

```bash
BACKFILL_CHECK_INTERVAL=5m ./bin/indexer
```

This logs warnings when gaps are detected but does not auto-backfill.

## How It Works

1. **Gap Detection**: Uses PostgreSQL `generate_series` with anti-join for O(n) performance:
   ```sql
   SELECT gs.height
   FROM generate_series(start, end) AS gs(height)
   WHERE NOT EXISTS (
       SELECT 1 FROM blocks b
       WHERE b.chain_id = ? AND b.height = gs.height
   )
   ```

2. **Batch Processing**: Gaps are processed in configurable batches to manage memory.

3. **Direct Indexing**: Calls `IndexBlock()` directly, bypassing the Redis queue to avoid backpressure during bulk operations.

4. **Concurrency**: Uses `errgroup` with configurable worker limit. The RPC client has built-in rate limiting (default 500 RPS).

5. **Error Handling**: Individual block failures are logged and counted but don't stop the backfill.

## Performance Tuning

| Scenario | Concurrency | Batch Size | Notes |
|----------|-------------|------------|-------|
| Default | 10 | 1000 | Safe for most cases |
| High-throughput | 50 | 5000 | For catching up large gaps |
| Low-resource | 5 | 500 | For constrained environments |

The RPC client rate limits at 500 RPS by default, so concurrency can safely exceed this without overwhelming the node.

## Example Output

```
$ ./bin/backfill -stats
Gap Statistics for chain 1:
  Total Expected: 50000
  Total Indexed:  49850
  Total Missing:  150
  First Missing:  1234
  Last Missing:   48901
  Completion:     99.70%
```

```
$ ./bin/backfill
time=... level=INFO msg="fetched chain head from RPC" height=50000
time=... level=INFO msg="starting backfill" chain_id=1 start_height=1 end_height=50000 batch_size=1000 concurrency=10 dry_run=false
time=... level=INFO msg="gap analysis complete" total_expected=50000 total_indexed=49850 total_missing=150 first_missing=1234 last_missing=48901
time=... level=INFO msg="backfill progress" processed=50 total=150 progress_pct=33.3% succeeded=50 failed=0 rate_per_sec=5.0 eta=20s
time=... level=INFO msg="backfill complete" total_missing=150 total_processed=150 total_succeeded=150 total_failed=0 duration=30s

Backfill Summary:
  Total Missing:   150
  Total Processed: 150
  Total Succeeded: 150
  Total Failed:    0
  Duration:        30s
```

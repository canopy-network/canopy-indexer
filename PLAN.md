## Finalized Plan

With your clarifications incorporated, here's the complete plan for updating the backfill tool into a default worker mode that continuously gap-checks and backfills all active chains:

### 1. Modify `cmd/backfill/main.go` for Worker Mode (Default)
- Add a `-once` flag for legacy one-time backfill behavior.
- In worker mode:
  - Perform initial chain discovery on startup (fetch active chains from admin DB using a new `DiscoverActiveChains` helper).
  - Maintain a map of active backfill goroutines per chain.
  - Run a polling loop every `ChainRediscoveryInterval` to rediscover active chains.
  - For new active chains, start a continuous backfill goroutine.
  - For removed chains, cancel their backfill.
- Each chain's backfill goroutine runs a loop: every 5 minutes (configurable `GapCheckInterval`), check gaps with `CheckHealth`; if gaps exist, run `Run`.
- Handle `-chain` flag in worker mode to restrict to that chain only.
- Ensure graceful shutdown (cancel all backfills on signal).

### 2. Add Helper Function in `internal/backfill`
- `DiscoverActiveChains(ctx, adminDB)`: Fetch and filter active chains (mirroring indexer's logic).

### 3. Update `internal/backfill/backfill.go`
- Add `RunContinuous` method: Loop with gap checks and backfill runs, respecting context.

### 4. Update `internal/backfill/config.go`
- Add `GapCheckInterval` (default 5 minutes).

### 5. Adjust Docker and Makefile
- Update `docker-compose.yml` for long-running service.
- Update `Makefile` (`backfill` implies `-once`).

This keeps the codebase minimal while making backfill a robust worker.
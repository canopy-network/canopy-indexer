# WebSocket Blob Auto-Subscription Plan

## Overview

This document outlines the plan to implement automatic WebSocket blob subscription for each discovered blockchain chain in the canopy-indexer system, while maintaining Redis publishing for external service awareness.

## Current State

- **Chain Discovery**: Working ✅ - rediscovers chains every 5 seconds from admin database
- **WebSocket Blob Listeners**: Implemented ✅ - auto-subscribe creates listeners per chain
- **Redis Publishing**: Working ✅ - publishes block heights for external consumption
- **Gap**: Worker provides HTTP polling fallback, but indexer should be WebSocket-only

## Goal

Enable the indexer to automatically subscribe to blob WebSockets for **every discovered chain**, providing real-time indexing with no HTTP fallback in the indexer binary. Block heights are published to Redis for external service awareness.

## Architecture

### New Hybrid Architecture
```
WebSocket Blob Listeners → IndexBlockWithData → Publish Height to Redis
```

- **WebSocket Reception**: Indexer receives IndexerBlob data directly via WebSocket (no HTTP)
- **Direct Indexing**: Blobs are processed and indexed immediately
- **Redis Publishing**: Block heights published to Redis streams for external services
- **HTTP Isolation**: Only backfill binary makes HTTP requests for gap filling

### Components
- **Blob Listeners**: Per-chain WebSocket connections receiving IndexerBlob data
- **Indexer Core**: Processes blobs directly via `IndexBlockWithData()`
- **Publisher**: Sends block heights to Redis streams after successful indexing
- **Backfill**: Separate binary for HTTP-based gap filling

## Implementation Plan

### Phase 1: Indexer Publisher Integration
**File**: `internal/indexer/indexer.go`

Add publisher to Indexer struct and integrate publishing:
```go
type Indexer struct {
    // ... existing fields ...
    publisher *publisher.Publisher  // For Redis height publishing
}

func New(ctx context.Context, logger *zap.Logger, cfg *config.Config, postgresClient *postgres.Client, publisher *publisher.Publisher) (*Indexer, error) {
    return &Indexer{
        // ... existing fields ...
        publisher: publisher,
    }, nil
}

func (idx *Indexer) IndexBlockWithData(ctx context.Context, data *blob.BlockData) error {
    // ... existing indexing logic ...

    // Publish height to Redis after successful indexing
    if err := idx.publisher.PublishBlock(ctx, data.ChainID, data.Height); err != nil {
        idx.logger.Error("failed to publish block height",
            zap.Uint64("chain_id", data.ChainID),
            zap.Uint64("height", data.Height),
            zap.Error(err))
        // Don't fail indexing due to publish error
    }

    return nil
}
```

### Phase 2: Remove Worker from Indexer Binary
**File**: `cmd/indexer/main.go`

Eliminate worker HTTP consumption:
- Remove worker creation and startup
- Keep publisher for Redis publishing
- Pass publisher to indexer constructor

```go
// Remove these imports
"github.com/canopy-network/canopy-indexer/internal/worker"

// Remove worker creation
// wrk, err := worker.New(worker.Config{...})

// Remove worker goroutine
// g.Go(func() error { return wrk.Run(ctx) })

// Pass publisher to indexer
idx, err := indexer.New(ctx, logger, cfg, &client, pub)
```

### Phase 3: Blob Handler Integration
**Files**: `internal/indexer/indexer.go`, `cmd/indexer/main.go`

Blob handlers already call `IndexBlockWithData()`, so publishing will happen automatically:
- Auto-subscription mode: `addChainLocked()` handlers call `IndexBlockWithData()`
- Single-chain mode: Main goroutine handler calls `IndexBlockWithData()`

No additional changes needed - publishing is integrated at the indexing level.

### Phase 4: Configuration Updates
**File**: `internal/config/config.go`

Optional cleanup:
- Remove worker-related configs if not needed elsewhere
- Ensure `WSBlobAutoSubscribe=true` default
- Keep Redis configs for publishing

### Phase 5: Testing and Validation
- Verify WebSocket blob reception works without HTTP fallback
- Confirm Redis streams receive block heights
- Test backfill HTTP functionality remains intact
- Validate chain rediscovery creates proper listeners

## Technical Details

### Redis Message Format
Each Redis stream message contains:
- 16 bytes total: 8 bytes chain ID + 8 bytes height (big-endian)
- Topic: `cfg.BlocksTopic` (default: "blocks-to-index")
- Consumer group: `cfg.ConsumerGroup` (default: "indexer-workers")

### Error Handling
- **WebSocket failures**: Log errors, auto-retry, continue with other chains
- **Publishing failures**: Log warnings, don't block indexing
- **Chain removal**: Gracefully close WebSocket, stop publishing for that chain

### Performance Characteristics
- **Real-time indexing**: Immediate processing of WebSocket-delivered blobs
- **Redis publishing**: Minimal overhead after successful indexing
- **No HTTP polling**: Eliminates network round-trips for block data
- **Memory efficient**: One listener per active chain

## Migration Benefits

1. **WebSocket-Only Reception**: Indexer never makes HTTP requests for blobs
2. **Real-Time Indexing**: Immediate indexing as blocks are produced
3. **External Awareness**: Redis streams provide block height notifications
4. **Simplified Architecture**: Direct blob → index → publish flow
5. **HTTP Isolation**: Backfill handles gap filling via HTTP separately

## Testing Strategy

### Unit Tests
- Publisher integration in Indexer
- Blob handler publishing behavior
- Error handling for publish failures

### Integration Tests
- WebSocket blob reception and indexing
- Redis stream publishing verification
- Chain lifecycle management
- Backfill HTTP functionality preservation

### Monitoring
- WebSocket connection health
- Redis publish success rates
- Indexing performance metrics
- Chain discovery and listener management

## Rollout Plan

1. **Deploy with auto-subscribe enabled**
2. **Monitor WebSocket connections and Redis publishing**
3. **Verify backfill continues working**
4. **Gradually disable legacy modes if needed**
5. **Scale to additional chains**

## Backward Compatibility

- **Existing deployments**: Can enable auto-subscribe alongside legacy modes
- **Redis consumers**: Existing services consuming height streams unaffected
- **Backfill**: HTTP-based gap filling continues to work
- **Configuration**: Legacy configs remain functional during transition

## Answered Questions

### Q: Should indexer publish to Redis if WebSocket delivers data directly?
**A**: Yes, for external service awareness. Indexer is the primary real-time indexer, but other services may want block notifications without full blob data.

### Q: What happens if WebSocket fails?
**A**: Indexer stops real-time indexing for that chain until connection recovers. No HTTP fallback - gaps filled by backfill binary.

### Q: Why keep Redis if indexer doesn't consume from it?
**A**: Redis streams serve as a notification mechanism for external services, not internal consumption.

### Q: Does this affect backfill?
**A**: No, backfill continues using HTTP RPC calls for gap filling as designed.

## Implementation Status

- [x] WebSocket blob auto-subscription framework
- [x] Per-chain listener management
- [x] Configuration integration
- [x] Publisher integration (Phase 1) - COMPLETED
- [x] Worker removal (Phase 2) - COMPLETED
- [x] Blob handler integration (Phase 3) - COMPLETED (automatic)
- [ ] Configuration validation (Phase 4) - Optional
- [ ] Testing and validation (Phase 5)</content>
<parameter name="filePath">/home/enielson/canopy/canopy-indexer/PLAN.md
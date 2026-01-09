# Blob Functionality Port Plan

Port blob functionality from `canopyx-blob` to `canopy-indexer`, replacing `lib.IndexerSnapshot` with `fsm.IndexerBlob`.

## Summary

Replace the current multi-RPC fetch pattern and `lib.IndexerSnapshot` WebSocket with a unified blob-based approach using `fsm.IndexerBlobs` (contains `Current` + `Previous` blobs for change detection in a single payload).

## Files to Modify/Create

### New Files
| File | Purpose |
|------|---------|
| `pkg/rpc/blob.go` | `HTTPClient.Blob()` method + `doProtobuf()` helper |
| `pkg/snapshot/blob_decoder.go` | Decode `fsm.IndexerBlob` to `indexer.BlockData` |

### Files to Modify
| File | Changes |
|------|---------|
| `pkg/rpc/paths.go` | Add `indexerBlobsPath = "/v1/query/indexer-blobs"` |
| `pkg/rpc/interfaces.go` | Add `Blob(ctx, height) (*fsm.IndexerBlobs, error)` to interface |
| `internal/listener/snapshot.go` | Replace `lib.IndexerSnapshot` with `fsm.IndexerBlob`, update path to `/v1/subscribe-indexer-blobs` |
| `pkg/snapshot/decoder.go` | Update `Decode()` to accept `fsm.IndexerBlob` instead of `lib.IndexerSnapshot` |
| `internal/backfill/backfill.go` | Add blob-based fetching option |
| `cmd/indexer/main.go` | Update handler to use new decoder |

## Implementation Steps

### Step 1: Add Blob RPC Method

**`pkg/rpc/paths.go`** - Add constant:
```go
indexerBlobsPath = "/v1/query/indexer-blobs"
```

**`pkg/rpc/blob.go`** - Create new file:
```go
package rpc

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/canopy-network/canopy/fsm"
    "github.com/canopy-network/canopy/lib"
)

// Blob fetches indexer blobs (current + previous) for a height.
func (c *HTTPClient) Blob(ctx context.Context, height uint64) (*fsm.IndexerBlobs, error) {
    var resp fsm.IndexerBlobs
    if err := c.doProtobuf(ctx, http.MethodPost, indexerBlobsPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

// doProtobuf makes an HTTP request with JSON payload and binary protobuf response.
func (c *HTTPClient) doProtobuf(ctx context.Context, method, path string, payload any, out lib.ProtoMessage) error {
    // Similar to doJSON but uses lib.Unmarshal for binary protobuf response
}
```

**`pkg/rpc/interfaces.go`** - Add to Client interface:
```go
Blob(ctx context.Context, height uint64) (*fsm.IndexerBlobs, error)
```

### Step 2: Update Snapshot Decoder

**`pkg/snapshot/decoder.go`** - Replace `lib.IndexerSnapshot` with `fsm.IndexerBlob`:

Key mapping from `fsm.IndexerBlob` fields:
- `Block` ([]byte) → decode to `lib.BlockResult`, extract `Transactions` and `Events`
- `Accounts` ([][]byte) → decode each to `fsm.Account`
- `Validators` ([][]byte) → `ValidatorsCurrent` (from Current blob) / `ValidatorsPrevious` (from Previous blob)
- `Pools` ([][]byte) → same pattern as Validators
- `NonSigners` ([][]byte) → same pattern
- `DoubleSigners` ([][]byte) → same pattern
- `DexBatches` ([][]byte) → `DexBatchesCurrent` / `DexBatchesPreviousCurr`
- `NextDexBatches` ([][]byte) → `DexBatchesNext` / `DexBatchesPreviousNext`
- `Orders` ([]byte) → decode `lib.OrderBooks`, flatten to `[]*lib.SellOrder`
- `Params`, `Supply`, `CommitteesData` → single decode each
- `SubsidizedCommittees`, `RetiredCommittees` → direct copy (already `[]uint64`)

**Signature change:**
```go
// Before
func Decode(snapshot *lib.IndexerSnapshot, chainID uint64) (*indexer.BlockData, error)

// After
func Decode(blobs *fsm.IndexerBlobs, chainID uint64) (*indexer.BlockData, error)
```

### Step 3: Update WebSocket Listener

**`internal/listener/snapshot.go`** - Update:

1. Change handler type:
```go
// Before
type SnapshotHandler func(snapshot *lib.IndexerSnapshot) error

// After
type SnapshotHandler func(blob *fsm.IndexerBlob) error
```

2. Update `buildURL()` path:
```go
// Before
fullPath := basePath + "/v1/subscribe-block-data"

// After
fullPath := basePath + "/v1/subscribe-indexer-blobs"
```

3. Update `listen()` unmarshaling:
```go
// Before
snapshot := new(lib.IndexerSnapshot)
if err := lib.Unmarshal(data, snapshot); err != nil { ... }

// After
blob := new(fsm.IndexerBlob)
if err := lib.Unmarshal(data, blob); err != nil { ... }
```

### Step 4: Update Backfill

**`internal/backfill/backfill.go`** - Add blob-based indexing:

```go
func (b *Backfiller) indexBlockWithBlob(ctx context.Context, rpc *rpc.HTTPClient, height uint64) error {
    blobs, err := rpc.Blob(ctx, height)
    if err != nil {
        return fmt.Errorf("fetch blob at height %d: %w", height, err)
    }

    data, err := snapshot.Decode(blobs, b.chainID)
    if err != nil {
        return fmt.Errorf("decode blob at height %d: %w", height, err)
    }

    return b.indexer.IndexBlockWithData(ctx, data)
}
```

### Step 5: Update Main

**`cmd/indexer/main.go`** - Update handler:
```go
handler := func(blob *fsm.IndexerBlob) error {
    blobs := &fsm.IndexerBlobs{Current: blob}
    data, err := snapshot.Decode(blobs, chainID)
    if err != nil {
        return err
    }
    return idx.IndexBlockWithData(ctx, data)
}
```

## Key Type Mapping

### fsm.IndexerBlob Structure
```go
type IndexerBlob struct {
    Block                []byte    // Serialized lib.BlockResult (contains Txs, Events)
    Accounts             [][]byte  // Each: serialized fsm.Account
    Validators           [][]byte  // Each: serialized fsm.Validator
    Pools                [][]byte  // Each: serialized fsm.Pool
    DexPrices            [][]byte  // Each: serialized lib.DexPrice
    NonSigners           [][]byte  // Each: serialized fsm.NonSigner
    DoubleSigners        [][]byte  // Each: serialized lib.DoubleSigner
    Orders               []byte    // Serialized lib.OrderBooks
    Params               []byte    // Serialized fsm.Params
    Supply               []byte    // Serialized fsm.Supply
    CommitteesData       []byte    // Serialized lib.CommitteesData
    DexBatches           [][]byte  // Each: serialized lib.DexBatch
    NextDexBatches       [][]byte  // Each: serialized lib.DexBatch
    SubsidizedCommittees []uint64
    RetiredCommittees    []uint64
}

type IndexerBlobs struct {
    Current  *IndexerBlob  // Data at height H
    Previous *IndexerBlob  // Data at height H-1 (for change detection)
}
```

## Notes

- `fsm.IndexerBlobs` contains both Current and Previous blobs - eliminates need for separate H-1 RPC calls
- Decoder must handle `Previous` being nil (height 1)
- Block contains embedded Transactions and Events (unlike lib.IndexerSnapshot which had them separate)

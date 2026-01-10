package indexer

import (
	"time"
)

const SupplyProductionTableName = "supply"
const SupplyStagingTableName = "supply_staging"

// SupplyColumns defines the schema for the supply table.
var SupplyColumns = []ColumnDef{
	{Name: "total", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "staked", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "delegated_only", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "height_time", Type: "DateTime64(3)", Codec: "DoubleDelta, ZSTD(1)"},
}

// Supply represents a versioned snapshot of token supply metrics.
// Snapshots are created using the snapshot-on-change pattern: a new row is created
// only when the supply metrics change from the previous height.
//
// This enables temporal queries like "What was the total supply at height 5000?"
// while using significantly less storage than storing supply at every height.
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - We compare RPC(height H) vs RPC(height H-1) to detect changes
// - We don't rely on database state which may be incomplete during parallel indexing
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
type Supply struct {
	// Supply metrics
	Total         uint64 `ch:"total" json:"total"`                   // Total tokens in the system (minted - burned)
	Staked        uint64 `ch:"staked" json:"staked"`                 // Total locked tokens (includes delegated)
	DelegatedOnly uint64 `ch:"delegated_only" json:"delegated_only"` // Total delegated tokens only

	// Version tracking - every change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries
}

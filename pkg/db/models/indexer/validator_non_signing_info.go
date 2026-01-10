package indexer

import (
	"time"
)

const ValidatorNonSigningInfoProductionTableName = "validator_non_signing_info"
const ValidatorNonSigningInfoStagingTableName = "validator_non_signing_info_staging"

// ValidatorNonSigningInfoColumns defines the schema for the validator_non_signing_info table.
// Removed trash properties: start_height, missed_blocks_window (per issues.md)
var ValidatorNonSigningInfoColumns = []ColumnDef{
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "missed_blocks_count", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "last_signed_height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "height_time", Type: "DateTime64(3)", Codec: "DoubleDelta, ZSTD(1)"},
}

// ValidatorNonSigningInfo represents a versioned snapshot of a validator's non-signing performance.
// This tracks non-signing counters for validators (missed blocks).
// Snapshots are created using the snapshot-on-change pattern: a new row is created
// only when the non-signing info changes (counter increments).
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - We compare RPC(height H) vs RPC(height H-1) to detect changes
// - We don't rely on database state which may be incomplete during parallel indexing
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Data Source: Canopy /v1/query/non-signers endpoint returns RpcNonSigner with:
// - Address: validator address
// - Counter: number of missed blocks in the current window
//
// This data is correlated with validator events like:
// - EventAutoPause: when a validator is automatically paused for missing too many blocks
// - EventAutoBeginUnstaking: when a validator is automatically unstaked
// - EventFinishUnstaking: when unstaking completes
type ValidatorNonSigningInfo struct {
	// Identity
	Address string `ch:"address" json:"address"` // Hex string representation of validator address

	// Non-signing performance tracking (from Canopy RpcNonSigner)
	MissedBlocksCount uint64 `ch:"missed_blocks_count" json:"missed_blocks_count"` // Maps from RpcNonSigner.Counter - count of missed blocks in current window

	// Additional tracking fields (if available from other sources)
	LastSignedHeight uint64 `ch:"last_signed_height" json:"last_signed_height"` // Last height at which validator signed (if tracked)

	// Version tracking - every change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries
}

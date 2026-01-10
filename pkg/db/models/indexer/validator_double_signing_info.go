package indexer

import (
	"time"
)

const ValidatorDoubleSigningInfoProductionTableName = "validator_double_signing_info"
const ValidatorDoubleSigningInfoStagingTableName = "validator_double_signing_info_staging"

// ValidatorDoubleSigningInfoColumns defines the schema for the validator_double_signing_info table.
var ValidatorDoubleSigningInfoColumns = []ColumnDef{
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "evidence_count", Type: "UInt64", Codec: "Delta, ZSTD(1)"},
	{Name: "first_evidence_height", Type: "UInt64", Codec: "Delta, ZSTD(1)"},
	{Name: "last_evidence_height", Type: "UInt64", Codec: "Delta, ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "Delta, ZSTD(1)"},
	{Name: "height_time", Type: "DateTime64(3)", Codec: "DoubleDelta, ZSTD(1)"},
}

// ValidatorDoubleSigningInfo represents a versioned snapshot of a validator's double-signing evidence.
// This tracks Byzantine behavior where validators sign multiple conflicting blocks at the same height.
// Snapshots are created using the snapshot-on-change pattern: a new row is created
// only when the evidence count changes (new infraction detected).
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - We compare RPC(height H) vs RPC(height H-1) to detect changes
// - We don't rely on database state which may be incomplete during parallel indexing
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Data Source: Canopy /v1/query/double-signers endpoint returns RpcDoubleSigner with:
// - Address: validator address
// - InfractionHeights: array of heights where double-signing occurred
//
// This data is correlated with Byzantine behavior tracking and validator slashing events.
type ValidatorDoubleSigningInfo struct {
	// Identity
	Address string `ch:"address" json:"address"` // Hex string representation of validator address

	// Double-signing evidence tracking (from Canopy RpcDoubleSigner)
	EvidenceCount uint64 `ch:"evidence_count" json:"evidence_count"` // Total count of double-signing infractions

	// Height range of infractions (derived from InfractionHeights array)
	FirstEvidenceHeight uint64 `ch:"first_evidence_height" json:"first_evidence_height"` // Height of first double-signing infraction (0 if no evidence)
	LastEvidenceHeight  uint64 `ch:"last_evidence_height" json:"last_evidence_height"`   // Height of most recent double-signing infraction (0 if no evidence)

	// Version tracking - every change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries
}

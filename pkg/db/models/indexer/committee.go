package indexer

import "time"

const (
	// CommitteeProductionTableName is the production table name for committees
	CommitteeProductionTableName = "committees"
	// CommitteeStagingTableName is the staging table name for committees
	CommitteeStagingTableName = "committees_staging"
)

// CommitteeColumns defines the schema for the committees table.
// Boolean fields are stored as UInt8 (0=false, 1=true).
// Codecs are optimized for 15x compression ratio:
// - DoubleDelta,LZ4 for sequential/monotonic values (height, timestamps)
// - Delta,ZSTD(3) for gradually changing counts and chain_id
// - ZSTD(1) for boolean flags
var CommitteeColumns = []ColumnDef{
	{Name: "chain_id", Type: "UInt16", Codec: "Delta, ZSTD(1)", CrossChainRename: "committee_chain_id"},
	{Name: "last_root_height_updated", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "last_chain_height_updated", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "number_of_samples", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "subsidized", Type: "UInt8", Codec: "ZSTD(1)"},
	{Name: "retired", Type: "UInt8", Codec: "ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "height_time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
}

// Committee represents committee data at a specific height.
// A committee is a group of validators responsible for a specific chain.
// Committees are only inserted when their data changes to maintain a sparse historical record.
type Committee struct {
	// Primary identifier
	ChainID uint16 `ch:"chain_id" json:"chain_id"` // Unique identifier of the chain/committee

	// Committee state tracking
	LastRootHeightUpdated  uint64 `ch:"last_root_height_updated" json:"last_root_height_updated"`   // Canopy height of most recent Certificate Results tx
	LastChainHeightUpdated uint64 `ch:"last_chain_height_updated" json:"last_chain_height_updated"` // 3rd party chain height of most recent Certificate Results
	NumberOfSamples        uint64 `ch:"number_of_samples" json:"number_of_samples"`                 // Count of processed Certificate Result Transactions

	// Status counts (aggregate across all committees at this height)
	Subsidized uint8 `ch:"subsidized" json:"subsidized"` // Total number of subsidized committees at this height
	Retired    uint8 `ch:"retired" json:"retired"`       // Total number of retired committees at this height

	// Block context
	Height     uint64    `ch:"height" json:"height"`           // Block height when this committee state became effective
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Timestamp of the block
}

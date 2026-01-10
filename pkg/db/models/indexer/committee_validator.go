package indexer

import (
	"time"
)

const CommitteeValidatorProductionTableName = "committee_validators"
const CommitteeValidatorStagingTableName = "committee_validators_staging"

// CommitteeValidatorColumns defines the schema for the committee_validators table.
// Codecs are optimized for 15x compression ratio:
// - DoubleDelta,LZ4 for sequential/monotonic values (height, timestamps)
// - ZSTD(1) for strings and booleans
// - Delta,ZSTD(3) for gradually changing amounts
//
// Denormalized fields from Committee entity enable queries like
// "show all validators in subsidized committees" without JOIN.
var CommitteeValidatorColumns = []ColumnDef{
	{Name: "committee_id", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validator_address", Type: "String", Codec: "ZSTD(1)", CrossChainRename: "address"},
	{Name: "staked_amount", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "status", Type: "LowCardinality(String)", Codec: "ZSTD(1)"},
	{Name: "delegate", Type: "Bool", Codec: "ZSTD(1)"},
	{Name: "compound", Type: "Bool", Codec: "ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "height_time", Type: "DateTime64(3)", Codec: "DoubleDelta, LZ4"},
	// Denormalized from Committee - enables filtering by committee status without JOIN
	{Name: "subsidized", Type: "Bool", Codec: "ZSTD(1)"},
	{Name: "retired", Type: "Bool", Codec: "ZSTD(1)"},
}

// CommitteeValidator represents the junction table between committees and validators.
// This denormalized table enables efficient queries like:
// - "Get all validators in committee X at height Y"
// - "Get all committees that validator X belongs to at height Y"
//
// While the Validator entity already contains a Committees array, this junction table
// provides O(1) lookups when querying by committee_id, which is critical for:
// - Committee dashboard views showing all members
// - Historical committee composition analysis
// - Validator set queries for specific chains
//
// Snapshots are created using the snapshot-on-change pattern: a new row is created
// only when a validator's committee membership changes.
//
// Note: This table is derived from the Validators.Committees array field.
// It should be populated during IndexValidators activity when committee membership changes.
type CommitteeValidator struct {
	// Relationship
	CommitteeID      uint64 `ch:"committee_id" json:"committee_id"`           // Committee (chain) ID
	ValidatorAddress string `ch:"validator_address" json:"validator_address"` // Hex string representation of validator address

	// Validator metadata (denormalized for query efficiency)
	StakedAmount uint64 `ch:"staked_amount" json:"staked_amount"` // Amount staked by validator in uCNPY
	Status       string `ch:"status" json:"status"`               // Validator status: active/paused/unstaking (derived from MaxPausedHeight/UnstakingHeight)
	Delegate     bool   `ch:"delegate" json:"delegate"`           // Whether validator accepts delegations
	Compound     bool   `ch:"compound" json:"compound"`           // Whether validator auto-compounds rewards

	// Version tracking - every membership change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries

	// Committee metadata (denormalized for query efficiency)
	// Enables queries like "show all validators in subsidized committees" without JOIN
	Subsidized bool `ch:"subsidized" json:"subsidized"` // Whether committee is subsidized (from Committee entity)
	Retired    bool `ch:"retired" json:"retired"`       // Whether committee is retired (from Committee entity)
}

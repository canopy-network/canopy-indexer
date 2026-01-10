package indexer

import (
	"time"
)

const ValidatorsProductionTableName = "validators"
const ValidatorsStagingTableName = "validators_staging"

// ValidatorColumns defines the schema for the validators table.
var ValidatorColumns = []ColumnDef{
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "public_key", Type: "String", Codec: "ZSTD(1)"},
	{Name: "net_address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "staked_amount", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "max_paused_height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "unstaking_height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "output", Type: "String", Codec: "ZSTD(1)"},
	{Name: "delegate", Type: "UInt8 DEFAULT 0"},
	{Name: "compound", Type: "UInt8 DEFAULT 0"},
	{Name: "status", Type: "String", Codec: "ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "height_time", Type: "DateTime64(3)", Codec: "DoubleDelta, ZSTD(1)"},
}

// Validator represents a versioned snapshot of a validator's state.
// Snapshots are created using the snapshot-on-change pattern: a new row is created
// only when the validator state changes (status, stake, committees, pause/unstaking state, etc.).
// This enables temporal queries like "What was validator X's state at height 5000?"
// while using significantly less storage than storing all validators at every height.
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - We compare RPC(height H) vs RPC(height H-1) to detect changes
// - We don't rely on database state which may be incomplete during parallel indexing
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Note: Validator creation height is tracked via the validator_created_height materialized view,
// which calculates MIN(height) for each address. Consumers should JOIN with this view
// if they need to know when a validator was created.
type Validator struct {
	// Identity
	Address    string `ch:"address" json:"address"`         // Hex string representation of validator address
	PublicKey  string `ch:"public_key" json:"public_key"`   // Hex string representation of public key
	NetAddress string `ch:"net_address" json:"net_address"` // P2P network address for validator communication

	// Stake and economics
	StakedAmount uint64 `ch:"staked_amount" json:"staked_amount"` // Amount staked by validator in uCNPY
	Output       string `ch:"output" json:"output"`               // Reward recipient address (empty = self)

	// Committee assignments (NOT stored in database - query from committee_validators table instead)
	Committees []uint64 `ch:"-" json:"committees"` // Array of committee IDs validator is assigned to (ephemeral field for activity processing)

	// State management
	MaxPausedHeight uint64 `ch:"max_paused_height" json:"max_paused_height"` // Height when pause expires (0 = not paused)
	UnstakingHeight uint64 `ch:"unstaking_height" json:"unstaking_height"`   // Height when unstaking completes (0 = not unstaking)

	// Delegation settings (stored as UInt8 in ClickHouse: 0=false, 1=true)
	Delegate bool `ch:"delegate" json:"delegate"` // Whether validator accepts delegations
	Compound bool `ch:"compound" json:"compound"` // Whether rewards are auto-compounded

	// Derived status (computed from MaxPausedHeight and UnstakingHeight)
	Status string `ch:"status" json:"status"` // Validator status: active/paused/unstaking

	// Version tracking - every state change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries
}

// DeriveStatus computes the validator status from protocol state fields.
// This implements the status derivation logic based on Canopy protocol rules:
//
//	unstaking: UnstakingHeight > 0 (validator is in unstaking period)
//	paused:    MaxPausedHeight > 0 (validator is paused until this height)
//	active:    default state (validator is active and signing)
//
// Note: Status is NOT returned by Canopy RPC - it must be derived from the
// MaxPausedHeight and UnstakingHeight fields. The derivation follows this priority:
// 1. If unstaking, always return "unstaking" (highest priority)
// 2. If paused, return "paused"
// 3. Otherwise return "active"
func (v *Validator) DeriveStatus() string {
	if v.UnstakingHeight > 0 {
		return "unstaking"
	}
	if v.MaxPausedHeight > 0 {
		return "paused"
	}
	return "active"
}

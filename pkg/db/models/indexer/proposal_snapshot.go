package indexer

import (
	"time"
)

const ProposalSnapshotsProductionTableName = "proposal_snapshots"

// ProposalSnapshotColumns defines the schema for the proposal_snapshots table.
// NOTE: Proposal snapshots are NOT height-based since /v1/gov/proposals endpoint doesn't support height queries.
// Snapshots are time-based and inserted via scheduled workflow every 5 minutes.
// Uses non-Nullable types with defaults to avoid UInt8 null-mask overhead per column.
// Codecs are optimized for compression:
// - DoubleDelta,LZ4 for monotonic timestamps
// - ZSTD(1) for strings (proposal hash and JSON data)
// - ZSTD(1) for boolean (approval flag)
// - LowCardinality(String) for type field (limited set of proposal types)
var ProposalSnapshotColumns = []ColumnDef{
	{Name: "proposal_hash", Type: "String", Codec: "ZSTD(1)"},
	{Name: "proposal", Type: "String", Codec: "ZSTD(1)"}, // JSON proposal data (full message)
	{Name: "approve", Type: "Bool", Codec: "ZSTD(1)"},    // Node approval flag
	{Name: "snapshot_time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},

	// Exploded proposal fields for efficient querying
	{Name: "proposal_type", Type: "LowCardinality(String)", Codec: "ZSTD(1)"},        // Message type (changeParameter, daoTransfer)
	{Name: "signer", Type: "String", Codec: "ZSTD(1)"},                               // Proposal signer address
	{Name: "start_height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},                // Proposal start height
	{Name: "end_height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},                  // Proposal end height
	{Name: "parameter_space", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},           // For changeParameter: fee, val, cons, gov
	{Name: "parameter_key", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},             // For changeParameter: parameter name
	{Name: "parameter_value", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},           // For changeParameter: parameter value as string
	{Name: "dao_transfer_address", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},      // For daoTransfer: recipient address
	{Name: "dao_transfer_amount", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"}, // For daoTransfer: amount
}

// ProposalSnapshot stores time-series snapshots of governance proposals.
// Unlike other entities, proposal snapshots are NOT height-based because the /v1/gov/proposals
// endpoint does not support height parameters - it always returns current state.
//
// Snapshots are captured via a scheduled workflow that runs every 5 minutes.
// ReplacingMergeTree deduplicates by (proposal_hash, snapshot_time), keeping the latest state.
// Uses non-Nullable types with defaults (0, ”) to avoid UInt8 null-mask overhead.
//
// Query patterns:
//   - Latest proposal state: SELECT * FROM proposal_snapshots FINAL WHERE proposal_hash = ? ORDER BY snapshot_time DESC LIMIT 1
//   - Historical state: SELECT * FROM proposal_snapshots FINAL WHERE proposal_hash = ? AND snapshot_time <= ? ORDER BY snapshot_time DESC LIMIT 1
//   - All proposals at time: SELECT DISTINCT proposal_hash FROM proposal_snapshots FINAL WHERE snapshot_time = ?
//   - By type: SELECT * FROM proposal_snapshots FINAL WHERE proposal_type = 'changeParameter'
//   - By parameter: SELECT * FROM proposal_snapshots FINAL WHERE parameter_space = 'fee' AND parameter_key = 'sendFee'
type ProposalSnapshot struct {
	// Primary key - composite key for deduplication
	ProposalHash string `ch:"proposal_hash" json:"proposal_hash"` // Hex hash of proposal (transaction hash)

	// Proposal data
	Proposal string `ch:"proposal" json:"proposal"` // JSON-encoded proposal (from fsm.GovProposalWithVote.Proposal)
	Approve  bool   `ch:"approve" json:"approve"`   // Whether this node approves the proposal

	// Time field for snapshot tracking
	SnapshotTime time.Time `ch:"snapshot_time" json:"snapshot_time"` // When this snapshot was captured

	// Exploded proposal fields for efficient querying (extracted from the Proposal JSON)
	ProposalType       string `ch:"proposal_type" json:"proposal_type"`                         // Message type (changeParameter, daoTransfer)
	Signer             string `ch:"signer" json:"signer"`                                       // Proposal signer address (hex)
	StartHeight        uint64 `ch:"start_height" json:"start_height"`                           // Proposal start height
	EndHeight          uint64 `ch:"end_height" json:"end_height"`                               // Proposal end height
	ParameterSpace     string `ch:"parameter_space" json:"parameter_space,omitempty"`           // For changeParameter: fee, val, cons, gov
	ParameterKey       string `ch:"parameter_key" json:"parameter_key,omitempty"`               // For changeParameter: parameter name
	ParameterValue     string `ch:"parameter_value" json:"parameter_value,omitempty"`           // For changeParameter: parameter value as string
	DaoTransferAddress string `ch:"dao_transfer_address" json:"dao_transfer_address,omitempty"` // For daoTransfer: recipient address
	DaoTransferAmount  uint64 `ch:"dao_transfer_amount" json:"dao_transfer_amount,omitempty"`   // For daoTransfer: amount
}

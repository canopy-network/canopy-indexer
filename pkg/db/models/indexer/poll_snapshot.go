package indexer

import (
	"time"
)

const PollSnapshotsProductionTableName = "poll_snapshots"

// PollSnapshotColumns defines the schema for the poll_snapshots table.
// NOTE: Poll snapshots are NOT height-based since /v1/gov/poll endpoint doesn't support height queries.
// Snapshots are time-based and inserted via scheduled workflow every 20 seconds.
// Codecs are optimized for 15x compression ratio:
// - DoubleDelta,LZ4 for monotonic timestamps
// - ZSTD(1) for strings
// - Delta,ZSTD(3) for gradually changing token amounts and percentages
var PollSnapshotColumns = []ColumnDef{
	{Name: "proposal_hash", Type: "String", Codec: "ZSTD(1)"},
	{Name: "proposal_url", Type: "String", Codec: "ZSTD(1)"},
	{Name: "accounts_approve_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "accounts_reject_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "accounts_total_voted_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "accounts_total_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "accounts_approve_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "accounts_reject_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "accounts_voted_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_approve_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_reject_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_total_voted_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_total_tokens", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_approve_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_reject_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "validators_voted_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "snapshot_time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
}

// PollSnapshot stores time-series snapshots of governance poll results.
// Unlike other entities, poll snapshots are NOT height-based because the /v1/gov/poll
// endpoint does not support height parameters - it always returns current state.
//
// Snapshots are captured via a scheduled workflow that runs every 20 seconds.
// ReplacingMergeTree deduplicates by (proposal_hash, snapshot_time), keeping the latest state.
//
// Query patterns:
//   - Latest poll state: SELECT * FROM poll_snapshots FINAL WHERE proposal_hash = ? ORDER BY snapshot_time DESC LIMIT 1
//   - Historical state: SELECT * FROM poll_snapshots FINAL WHERE proposal_hash = ? AND snapshot_time <= ? ORDER BY snapshot_time DESC LIMIT 1
//   - All proposals at time: SELECT DISTINCT proposal_hash FROM poll_snapshots FINAL WHERE snapshot_time = ?
type PollSnapshot struct {
	// Primary key - composite key for deduplication
	ProposalHash string `ch:"proposal_hash" json:"proposal_hash"` // Hex hash of proposal

	// Proposal metadata
	ProposalURL string `ch:"proposal_url" json:"proposal_url"` // URL to proposal document

	// Account voting statistics
	AccountsApproveTokens     uint64 `ch:"accounts_approve_tokens" json:"accounts_approve_tokens"`
	AccountsRejectTokens      uint64 `ch:"accounts_reject_tokens" json:"accounts_reject_tokens"`
	AccountsTotalVotedTokens  uint64 `ch:"accounts_total_voted_tokens" json:"accounts_total_voted_tokens"`
	AccountsTotalTokens       uint64 `ch:"accounts_total_tokens" json:"accounts_total_tokens"`
	AccountsApprovePercentage uint64 `ch:"accounts_approve_percentage" json:"accounts_approve_percentage"`
	AccountsRejectPercentage  uint64 `ch:"accounts_reject_percentage" json:"accounts_reject_percentage"`
	AccountsVotedPercentage   uint64 `ch:"accounts_voted_percentage" json:"accounts_voted_percentage"`

	// Validator voting statistics
	ValidatorsApproveTokens     uint64 `ch:"validators_approve_tokens" json:"validators_approve_tokens"`
	ValidatorsRejectTokens      uint64 `ch:"validators_reject_tokens" json:"validators_reject_tokens"`
	ValidatorsTotalVotedTokens  uint64 `ch:"validators_total_voted_tokens" json:"validators_total_voted_tokens"`
	ValidatorsTotalTokens       uint64 `ch:"validators_total_tokens" json:"validators_total_tokens"`
	ValidatorsApprovePercentage uint64 `ch:"validators_approve_percentage" json:"validators_approve_percentage"`
	ValidatorsRejectPercentage  uint64 `ch:"validators_reject_percentage" json:"validators_reject_percentage"`
	ValidatorsVotedPercentage   uint64 `ch:"validators_voted_percentage" json:"validators_voted_percentage"`

	// Time field for snapshot tracking
	SnapshotTime time.Time `ch:"snapshot_time" json:"snapshot_time"` // When this snapshot was captured
}

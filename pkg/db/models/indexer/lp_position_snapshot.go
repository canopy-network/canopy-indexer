package indexer

import (
	"time"
)

const LPPositionSnapshotsProductionTableName = "lp_position_snapshots"
const LPPositionSnapshotsUpdateAtColumnName = "updated_at"

// LPPositionSnapshotColumns defines the schema for the lp_position_snapshots table.
// Stores daily snapshots of liquidity provider positions across all chains.
// Snapshots capture calendar-day based (UTC) states of LP positions, tracking both
// liquidity amounts and pool share percentages with 6 decimal precision.
//
// Codecs are optimized for compression:
// - Delta,ZSTD(1) for chain IDs and pool IDs
// - ZSTD(1) for addresses
// - DoubleDelta,ZSTD(1) for monotonically increasing heights
// - Delta,ZSTD(3) for gradually changing balances and percentages
// - DoubleDelta,ZSTD(1) for timestamps
var LPPositionSnapshotColumns = []ColumnDef{
	// Position Identity
	{Name: "source_chain_id", Type: "UInt16", Codec: "Delta, ZSTD(1)"},
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "pool_id", Type: "UInt32", Codec: "Delta, ZSTD(1)"},

	// Snapshot Date & State
	{Name: "snapshot_date", Type: "Date", Codec: "Delta, ZSTD(1)"},
	{Name: "snapshot_height", Type: "UInt64", Codec: "DoubleDelta, ZSTD(1)"},
	{Name: "snapshot_balance", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "pool_share_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"}, // 6 decimal precision: 25.46421% = 25464210

	// Position Lifecycle
	{Name: "position_created_date", Type: "Date", Codec: "Delta, ZSTD(1)"},
	{Name: "position_closed_date", Type: "Date DEFAULT '1970-01-01'", Codec: "ZSTD(1)"}, // Default to epoch if not closed
	{Name: "is_position_active", Type: "UInt8", Codec: "ZSTD(1)"},

	// Metadata
	{Name: "computed_at", Type: "DateTime64(6)", Codec: "DoubleDelta, ZSTD(1)"},
	{Name: LPPositionSnapshotsUpdateAtColumnName, Type: "DateTime64(6)", Codec: "DoubleDelta, ZSTD(1)"},
}

// LPPositionSnapshot stores daily snapshots of liquidity provider positions.
// Each row represents the state of an (address, pool_id) position on a specific date.
//
// ARCHITECTURAL NOTES:
// - Calendar-day based: Snapshots at highest block with block_time ≤ 23:59:59 UTC
// - Cross-chain storage: All chains stored in canopyx_cross_chain.lp_position_snapshots_global
// - Data source: Per-chain pool_points_by_holder tables (NOT cross-chain materialized views)
// - Position identity: (address, pool_id) - one active position per pair at any time
// - Position lifecycle: Created when points > 0, closed when points = 0
// - ReplacingMergeTree: Deduplicates by updated_at (allows recomputing "today" multiple times)
// - Scheduled workflow: Runs hourly per chain, recomputes "today", finalizes "yesterday"
// - Manual backfill: Admin API triggers Temporal's built-in schedule backfill
//
// Query patterns:
//   - Latest position: SELECT * FROM lp_position_snapshots_global FINAL WHERE source_chain_id = ? AND address = ? AND pool_id = ? ORDER BY snapshot_date DESC LIMIT 1
//   - Historical state: SELECT * FROM lp_position_snapshots_global FINAL WHERE source_chain_id = ? AND address = ? AND pool_id = ? AND snapshot_date = ?
//   - All positions for address: SELECT * FROM lp_position_snapshots_global FINAL WHERE source_chain_id = ? AND address = ? ORDER BY snapshot_date DESC
//   - Active positions: SELECT * FROM lp_position_snapshots_global FINAL WHERE source_chain_id = ? AND is_position_active = 1
type LPPositionSnapshot struct {
	// Position Identity - composite key for deduplication
	SourceChainID uint16 `ch:"source_chain_id" json:"source_chain_id"` // Chain where LP staked
	Address       string `ch:"address" json:"address"`                 // LP holder address
	PoolID        uint32 `ch:"pool_id" json:"pool_id"`                 // Liquidity pool ID

	// Snapshot Date & State
	SnapshotDate        time.Time `ch:"snapshot_date" json:"snapshot_date"`                 // Calendar date (UTC) of snapshot
	SnapshotHeight      uint64    `ch:"snapshot_height" json:"snapshot_height"`             // Block height at end of day
	SnapshotBalance     uint64    `ch:"snapshot_balance" json:"snapshot_balance"`           // LP balance in pool (from liquidity_pool_points)
	PoolSharePercentage uint64    `ch:"pool_share_percentage" json:"pool_share_percentage"` // Pool share with 6 decimals (25.46421% = 25464210)

	// Position Lifecycle
	PositionCreatedDate time.Time `ch:"position_created_date" json:"position_created_date"` // Date when position first created (points > 0)
	PositionClosedDate  time.Time `ch:"position_closed_date" json:"position_closed_date"`   // Date when position closed (points = 0), epoch default if active
	IsPositionActive    uint8     `ch:"is_position_active" json:"is_position_active"`       // 1 if position currently active, 0 if closed

	// Metadata
	ComputedAt time.Time `ch:"computed_at" json:"computed_at"` // When snapshot was computed
	UpdatedAt  time.Time `ch:"updated_at" json:"updated_at"`   // When snapshot was last updated (for ReplacingMergeTree)
}

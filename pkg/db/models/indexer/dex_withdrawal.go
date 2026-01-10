package indexer

import (
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
)

const DexWithdrawalsProductionTableName = "dex_withdrawals"
const DexWithdrawalsStagingTableName = DexWithdrawalsProductionTableName + entities.StagingSuffix

// DexWithdrawalColumns defines the schema for the dex_withdrawals table.
var DexWithdrawalColumns = []ColumnDef{
	{Name: "order_id", Type: "String", Codec: "ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "height_time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
	{Name: "committee", Type: "UInt16", Codec: "Delta, ZSTD(1)"},
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "percent", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "state", Type: "LowCardinality(String)"},
	{Name: "local_amount", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "remote_amount", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "points_burned", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
}

// DexWithdrawal represents a versioned snapshot of a DEX liquidity withdrawal.
// Liquidity providers withdraw their tokens based on their pool points percentage.
// Snapshots are created at each state transition (pending -> complete).
//
// Lifecycle States:
// - pending: Withdrawal appears in dex-batch or next-dex-batch (not yet processed)
// - complete: EventDexLiquidityWithdrawal fired, withdrawal processed with final amounts
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - Initial state comes from RPC (DexBatch or NextDexBatch at height H)
// - Final state comes from events at height H describing withdrawals from H-1
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Note: Withdrawal creation time is tracked via the dex_withdrawal_created_height materialized view,
// which calculates MIN(height) for each order_id. Consumers should JOIN with this view
// if they need to know when a withdrawal was created.
type DexWithdrawal struct {
	// Identity
	OrderID string `ch:"order_id" json:"order_id"` // Hex string representation of order ID

	// Version tracking - every state change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries

	// Chain context
	Committee uint16 `ch:"committee" json:"committee"` // Committee ID (pool's chain ID)

	// Withdrawal details (from DexLiquidityWithdraw)
	Address string `ch:"address" json:"address"` // Hex string representation of address
	Percent uint64 `ch:"percent" json:"percent"` // Percentage of pool points being withdrawn (0-100)

	// Lifecycle state
	State string `ch:"state" json:"state"` // "pending" or "complete"

	// Execution results (populated by EventDexLiquidityWithdrawal when state=complete)
	LocalAmount  uint64 `ch:"local_amount" json:"local_amount"`   // Amount received on this chain
	RemoteAmount uint64 `ch:"remote_amount" json:"remote_amount"` // Amount received on counter chain
	PointsBurned uint64 `ch:"points_burned" json:"points_burned"` // Pool points burned in withdrawal
}

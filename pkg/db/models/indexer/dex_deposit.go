package indexer

import (
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
)

const DexDepositsProductionTableName = "dex_deposits"
const DexDepositsStagingTableName = DexDepositsProductionTableName + entities.StagingSuffix

// DexDepositColumns defines the schema for the dex_deposits table.
var DexDepositColumns = []ColumnDef{
	{Name: "order_id", Type: "String", Codec: "ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "height_time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
	{Name: "committee", Type: "UInt16", Codec: "Delta, ZSTD(1)"},
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "amount", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "state", Type: "LowCardinality(String)"},
	{Name: "local_origin", Type: "Bool"},
	{Name: "points_received", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
}

// DexDeposit represents a versioned snapshot of a DEX liquidity deposit.
// Liquidity providers deposit tokens to earn a share of trading fees.
// Snapshots are created at each state transition (pending -> complete).
//
// Lifecycle States:
// - pending: Deposit appears in dex-batch or next-dex-batch (not yet processed)
// - complete: EventDexLiquidityDeposit fired, deposit processed with final amounts
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - Initial state comes from RPC (DexBatch or NextDexBatch at height H)
// - Final state comes from events at height H describing deposits from H-1
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Note: Deposit creation time is tracked via the dex_deposit_created_height materialized view,
// which calculates MIN(height) for each order_id. Consumers should JOIN with this view
// if they need to know when a deposit was created.
type DexDeposit struct {
	// Identity
	OrderID string `ch:"order_id" json:"order_id"` // Hex string representation of order ID

	// Version tracking - every state change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries

	// Chain context
	Committee uint16 `ch:"committee" json:"committee"` // Committee ID (pool's chain ID)

	// Deposit details (from DexLiquidityDeposit)
	Address string `ch:"address" json:"address"` // Hex string representation of address
	Amount  uint64 `ch:"amount" json:"amount"`   // Amount being deposited

	// Lifecycle state
	State string `ch:"state" json:"state"` // "pending" or "complete"

	// Execution results (populated by EventDexLiquidityDeposit when state=complete)
	LocalOrigin    bool   `ch:"local_origin" json:"local_origin"`       // Was deposit made on this chain or counter chain
	PointsReceived uint64 `ch:"points_received" json:"points_received"` // Pool points received for this deposit
}

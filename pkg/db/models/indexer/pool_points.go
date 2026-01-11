package indexer

import (
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
)

const PoolPointsByHolderProductionTableName = "pool_points_by_holder"
const PoolPointsByHolderStagingTableName = PoolPointsByHolderProductionTableName + entities.StagingSuffix

// PoolPointsByHolderColumns defines the schema for the pool_points_by_holder table.
// Denormalized fields from Pool entity enable value calculations without JOIN.
var PoolPointsByHolderColumns = []ColumnDef{
	{Name: "address"},
	{Name: "pool_id"},
	{Name: "height"},
	{Name: "height_time"},
	{Name: "committee"},
	{Name: "points"},
	{Name: "liquidity_pool_points"},
	{Name: "liquidity_pool_id"},
	// Denormalized from Pool - enables holder value calculation without JOIN
	{Name: "pool_amount"},
}

// PoolPointsByHolder represents a versioned snapshot of a liquidity provider's pool points.
// Pool points represent ownership shares in a DEX liquidity pool, determining the proportion
// of trading fees earned and the amount received when withdrawing liquidity.
//
// Snapshots are created when:
// - EventDexLiquidityDeposit: Points increase after successful deposit
// - EventDexLiquidityWithdrawal: Points decrease after successful withdrawal
//
// Uses snapshot-on-change pattern: a new row is created only when a holder's points change.
// This enables temporal queries like "What were Alice's pool points at height 5000?" while
// using significantly less storage than storing all holders at every height.
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - We query RPC PoolPoints at height H (after event execution)
// - We don't rely on database state which may be incomplete during parallel indexing
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Note: Pool points holder creation time is tracked via the pool_points_created_height
// materialized view, which calculates MIN(height) for each (address, pool_id) pair.
type PoolPointsByHolder struct {
	// Identity - composite key
	Address string `ch:"address" json:"address"` // Hex string representation of holder address
	PoolID  uint32 `ch:"pool_id" json:"pool_id"` // Calculated pool ID (committee + LiquidityPoolAddend)

	// Version tracking - every points change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries

	// Chain context
	Committee uint16 `ch:"committee" json:"committee"` // Committee ID (pool's chain ID)

	// Pool points ownership
	Points uint64 `ch:"points" json:"points"` // Number of pool points owned by this holder

	// Derived values for convenience (calculated from pool state at height H)
	LiquidityPoolPoints uint64 `ch:"liquidity_pool_points" json:"liquidity_pool_points"` // Pool's total points (TotalPoolPoints from the pool)
	LiquidityPoolID     uint32 `ch:"liquidity_pool_id" json:"liquidity_pool_id"`         // Calculated liquidity pool ID (committee + LiquidityPoolAddend)

	// Denormalized from Pool - enables holder value calculation without JOIN
	// Value calculation: holder_value = (points / liquidity_pool_points) * pool_amount
	PoolAmount uint64 `ch:"pool_amount" json:"pool_amount"` // Total token amount in pool at snapshot height
}

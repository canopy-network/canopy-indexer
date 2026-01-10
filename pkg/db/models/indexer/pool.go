package indexer

import (
	"math"
	"time"
)

const PoolsProductionTableName = "pools"
const PoolsStagingTableName = "pools_staging"

// PoolColumns defines the schema for the pools table.
// NOTE: chain_id is renamed to pool_chain_id in cross-chain tables to avoid conflict
// with the cross-chain table's chain_id column (which indicates the source database).
var PoolColumns = []ColumnDef{
	{Name: "pool_id", Type: "UInt32"},
	{Name: "height", Type: "UInt64"},
	{Name: "chain_id", Type: "UInt16", Codec: "Delta, ZSTD(1)", CrossChainRename: "pool_chain_id"},
	{Name: "amount", Type: "UInt64"},
	{Name: "total_points", Type: "UInt64"},
	{Name: "lp_count", Type: "UInt16"},
	{Name: "height_time", Type: "DateTime64(6)"},
	{Name: "liquidity_pool_id", Type: "UInt32"},
	{Name: "holding_pool_id", Type: "UInt32"},
	{Name: "escrow_pool_id", Type: "UInt32"},
	{Name: "reward_pool_id", Type: "UInt32"},
	{Name: "amount_delta", Type: "Int64", Codec: "Delta, ZSTD(3)"},
	{Name: "total_points_delta", Type: "Int64", Codec: "Delta, ZSTD(3)"},
	{Name: "lp_count_delta", Type: "Int16", Codec: "Delta, ZSTD(3)"},
}

// Pool ID calculation constants (from Canopy blockchain logic)
const (
	// TODO: use this from canopy as pkg (fsm/key.go)
	MaxChainID          = uint64(math.MaxUint16 / 4)
	HoldingPoolAddend   = uint64(1 * math.MaxUint16 / 4)
	LiquidityPoolAddend = uint64(2 * math.MaxUint16 / 4)
	EscrowPoolAddend    = uint64(4 * math.MaxUint16 / 4)
)

// Pool stores pool state snapshots at each height.
// Uses snapshot-on-change pattern: a new row is created only when pool state changes.
// ReplacingMergeTree deduplicates by (pool_id, height), keeping the latest state.
//
// Query patterns:
//   - Latest state: SELECT * FROM pools FINAL WHERE pool_id = ? ORDER BY height DESC LIMIT 1
//   - Historical state: SELECT * FROM pools FINAL WHERE pool_id = ? AND height <= ? ORDER BY height DESC LIMIT 1
//   - All pools at height: SELECT * FROM pools FINAL WHERE height <= ? GROUP BY pool_id HAVING height = max(height)
type Pool struct {
	// Primary key - composite key for deduplication
	PoolID uint32 `ch:"pool_id" json:"pool_id"`

	// Chain identifier for multi-chain support
	ChainID uint16 `ch:"chain_id" json:"chain_id"`

	// Pool state fields
	Amount      uint64 `ch:"amount" json:"amount"`             // Total amount in pool
	TotalPoints uint64 `ch:"total_points" json:"total_points"` // Total pool points
	LPCount     uint16 `ch:"lp_count" json:"lp_count"`         // Number of liquidity providers

	// H-1 delta fields (change from previous height)
	AmountDelta      int64 `ch:"amount_delta" json:"amount_delta"`             // Change in amount from H-1 to H
	TotalPointsDelta int64 `ch:"total_points_delta" json:"total_points_delta"` // Change in total points from H-1 to H
	LPCountDelta     int16 `ch:"lp_count_delta" json:"lp_count_delta"`         // Change in LP count from H-1 to H

	Height uint64 `ch:"height" json:"height"`
	// Time fields for range queries
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp

	// Calculated pool IDs for different pool types
	// These IDs are calculated using: Addend + ChainID
	LiquidityPoolID uint32 `ch:"liquidity_pool_id" json:"liquidity_pool_id"` // LiquidityPoolAddend + ChainID
	HoldingPoolID   uint32 `ch:"holding_pool_id" json:"holding_pool_id"`     // HoldingPoolAddend + ChainID
	EscrowPoolID    uint32 `ch:"escrow_pool_id" json:"escrow_pool_id"`       // EscrowPoolAddend + ChainID
	RewardPoolID    uint32 `ch:"reward_pool_id" json:"reward_pool_id"`       // Same as ChainID
}

// CalculatePoolIDs populates the calculated pool ID fields based on the ChainID.
// This should be called after setting the ChainID field.
func (p *Pool) CalculatePoolIDs() {
	chainID := uint64(p.ChainID)
	p.LiquidityPoolID = uint32(LiquidityPoolAddend + chainID)
	p.HoldingPoolID = uint32(HoldingPoolAddend + chainID)
	p.EscrowPoolID = uint32(EscrowPoolAddend + chainID)
	p.RewardPoolID = uint32(chainID)
}

// ExtractChainIDFromPoolID extracts the chain ID from a pool ID.
// Pool IDs are encoded as: TypeAddend + ChainID
// where TypeAddend is 0 (reward), 16384 (holding), 32768 (liquidity), or 65536 (escrow)
func ExtractChainIDFromPoolID(poolID uint64) uint64 {
	switch {
	case poolID >= EscrowPoolAddend:
		return poolID - EscrowPoolAddend
	case poolID >= LiquidityPoolAddend:
		return poolID - LiquidityPoolAddend
	case poolID >= HoldingPoolAddend:
		return poolID - HoldingPoolAddend
	default:
		return poolID // Reward pool: ID = ChainID
	}
}

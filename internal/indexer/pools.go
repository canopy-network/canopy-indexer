package indexer

import (
	"strconv"

	"github.com/canopy-network/canopy-indexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writePools(batch *pgx.Batch, data *BlockData) {
	if len(data.PoolsCurrent) == 0 {
		return
	}

	// Build previous state maps for change detection
	type prevHolder struct {
		points          uint64
		totalPoolPoints uint64
	}
	prevHolderMap := make(map[string]prevHolder) // key: "poolID:address"

	for _, p := range data.PoolsPrevious {
		for _, pp := range p.Points {
			key := poolHolderKey(p.Id, transform.BytesToHex(pp.Address))
			prevHolderMap[key] = prevHolder{
				points:          pp.Points,
				totalPoolPoints: p.TotalPoolPoints,
			}
		}
	}

	for _, pool := range data.PoolsCurrent {
		batch.Queue(`
			INSERT INTO pools (
				chain_id, pool_id, pool_chain_id, amount, total_points, lp_count,
				height, height_time, liquidity_pool_id, holding_pool_id, escrow_pool_id, reward_pool_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (chain_id, pool_id, height) DO UPDATE SET
				amount = EXCLUDED.amount,
				total_points = EXCLUDED.total_points,
				lp_count = EXCLUDED.lp_count
		`,
			data.ChainID,
			pool.Id,
			transform.ExtractChainIDFromPoolID(pool.Id),
			pool.Amount,
			pool.TotalPoolPoints,
			len(pool.Points),
			data.Height,
			data.BlockTime,
			0, 0, 0, 0, // pool IDs - derive based on pool type
		)

		// Extract pool points holders with change detection
		holders := transform.PoolPointsHoldersFromFSM(pool)
		for _, h := range holders {
			key := poolHolderKey(pool.Id, h.Address)
			prev, existed := prevHolderMap[key]

			// Snapshot if: new holder, points changed, or pool's TotalPoolPoints changed
			shouldSnapshot := !existed ||
				h.Points != prev.points ||
				h.LiquidityPoolPoints != prev.totalPoolPoints

			if shouldSnapshot {
				batch.Queue(`
					INSERT INTO pool_points_by_holder (
						chain_id, address, pool_id, committee, points,
						liquidity_pool_points, liquidity_pool_id, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
					ON CONFLICT (chain_id, address, pool_id, height) DO NOTHING
				`,
					data.ChainID,
					h.Address,
					h.PoolID,
					h.Committee,
					h.Points,
					h.LiquidityPoolPoints,
					h.LiquidityPoolID,
					data.Height,
					data.BlockTime,
				)
			}
		}
	}
}

// poolHolderKey creates a unique key for a pool holder.
func poolHolderKey(poolID uint64, address string) string {
	return strconv.FormatUint(poolID, 10) + ":" + address
}

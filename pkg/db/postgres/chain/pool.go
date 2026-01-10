package chain

import (
	"context"
)

// initPools creates the pools table matching indexer.Pool
// This matches pkg/db/models/indexer/pool.go:48-75 (14 fields)
func (db *DB) initPools(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pools (
			pool_id INTEGER NOT NULL,                      -- UInt32 -> INTEGER
			height BIGINT NOT NULL,
			chain_id SMALLINT NOT NULL,                    -- UInt16 -> SMALLINT (renamed from pool_chain_id)
			amount BIGINT NOT NULL DEFAULT 0,
			total_points BIGINT NOT NULL DEFAULT 0,
			lp_count SMALLINT NOT NULL DEFAULT 0,          -- UInt16 -> SMALLINT
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			liquidity_pool_id INTEGER NOT NULL DEFAULT 0,  -- UInt32 -> INTEGER
			holding_pool_id INTEGER NOT NULL DEFAULT 0,    -- UInt32 -> INTEGER
			escrow_pool_id INTEGER NOT NULL DEFAULT 0,     -- UInt32 -> INTEGER
			reward_pool_id INTEGER NOT NULL DEFAULT 0,     -- UInt32 -> INTEGER
			amount_delta BIGINT NOT NULL DEFAULT 0,        -- Int64 -> BIGINT
			total_points_delta BIGINT NOT NULL DEFAULT 0,  -- Int64 -> BIGINT
			lp_count_delta SMALLINT NOT NULL DEFAULT 0,    -- Int16 -> SMALLINT
			PRIMARY KEY (pool_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_pools_height ON pools(height);
		CREATE INDEX IF NOT EXISTS idx_pools_chain ON pools(chain_id);
	`

	return db.Exec(ctx, query)
}

// initPoolPointsByHolder creates the pool_points_by_holder table matching indexer.PoolPointsByHolder
// This matches pkg/db/models/indexer/pool_points.go:46-68 (9 fields)
func (db *DB) initPoolPointsByHolder(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pool_points_by_holder (
			address TEXT NOT NULL,
			pool_id INTEGER NOT NULL,                      -- UInt32 -> INTEGER
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			committee SMALLINT NOT NULL,                   -- UInt16 -> SMALLINT
			points BIGINT NOT NULL DEFAULT 0,
			liquidity_pool_points BIGINT NOT NULL DEFAULT 0,
			liquidity_pool_id INTEGER NOT NULL DEFAULT 0,  -- UInt32 -> INTEGER
			pool_amount BIGINT NOT NULL DEFAULT 0,         -- NEW: Denormalized from Pool
			PRIMARY KEY (address, pool_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_pool_points_by_holder_height ON pool_points_by_holder(height);
		CREATE INDEX IF NOT EXISTS idx_pool_points_by_holder_pool ON pool_points_by_holder(pool_id);
		CREATE INDEX IF NOT EXISTS idx_pool_points_by_holder_address ON pool_points_by_holder(address);
	`

	return db.Exec(ctx, query)
}

package chain

import (
	"context"
)

// initDexOrders creates the dex_orders table matching indexer.DexOrder
// This matches pkg/db/models/indexer/dex_order.go:53-80 (13 fields)
func (db *DB) initDexOrders(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_orders (
			order_id TEXT NOT NULL,
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			committee SMALLINT NOT NULL,                 -- UInt16 -> SMALLINT
			address TEXT NOT NULL,
			amount_for_sale BIGINT NOT NULL DEFAULT 0,
			requested_amount BIGINT NOT NULL DEFAULT 0,
			state TEXT NOT NULL DEFAULT 'future',        -- LowCardinality(String): future, locked, complete
			success BOOLEAN NOT NULL DEFAULT false,
			sold_amount BIGINT NOT NULL DEFAULT 0,
			bought_amount BIGINT NOT NULL DEFAULT 0,
			local_origin BOOLEAN NOT NULL DEFAULT false,
			locked_height BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_orders_height ON dex_orders(height);
		CREATE INDEX IF NOT EXISTS idx_dex_orders_address ON dex_orders(address);
		CREATE INDEX IF NOT EXISTS idx_dex_orders_state ON dex_orders(state);
		CREATE INDEX IF NOT EXISTS idx_dex_orders_committee ON dex_orders(committee);
	`

	return db.Exec(ctx, query)
}

// initDexDeposits creates the dex_deposits table matching indexer.DexDeposit
// This matches pkg/db/models/indexer/dex_deposit.go:41-62 (9 fields)
func (db *DB) initDexDeposits(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_deposits (
			order_id TEXT NOT NULL,
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			committee SMALLINT NOT NULL,                 -- UInt16 -> SMALLINT
			address TEXT NOT NULL,
			amount BIGINT NOT NULL DEFAULT 0,
			state TEXT NOT NULL DEFAULT 'pending',       -- LowCardinality(String): pending, complete
			local_origin BOOLEAN NOT NULL DEFAULT false,
			points_received BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_deposits_height ON dex_deposits(height);
		CREATE INDEX IF NOT EXISTS idx_dex_deposits_address ON dex_deposits(address);
		CREATE INDEX IF NOT EXISTS idx_dex_deposits_state ON dex_deposits(state);
	`

	return db.Exec(ctx, query)
}

// initDexWithdrawals creates the dex_withdrawals table matching indexer.DexWithdrawal
// This matches pkg/db/models/indexer/dex_withdrawal.go:42-64 (10 fields)
func (db *DB) initDexWithdrawals(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_withdrawals (
			order_id TEXT NOT NULL,
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			committee SMALLINT NOT NULL,                 -- UInt16 -> SMALLINT
			address TEXT NOT NULL,
			percent BIGINT NOT NULL DEFAULT 0,
			state TEXT NOT NULL DEFAULT 'pending',       -- LowCardinality(String): pending, complete
			local_amount BIGINT NOT NULL DEFAULT 0,
			remote_amount BIGINT NOT NULL DEFAULT 0,
			points_burned BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_withdrawals_height ON dex_withdrawals(height);
		CREATE INDEX IF NOT EXISTS idx_dex_withdrawals_address ON dex_withdrawals(address);
		CREATE INDEX IF NOT EXISTS idx_dex_withdrawals_state ON dex_withdrawals(state);
	`

	return db.Exec(ctx, query)
}

// initDexPrices creates the dex_prices table matching indexer.DexPrice
// This matches pkg/db/models/indexer/dexprice.go:30-50 (10 fields)
func (db *DB) initDexPrices(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_prices (
			local_chain_id SMALLINT NOT NULL,             -- UInt16 -> SMALLINT
			remote_chain_id SMALLINT NOT NULL,            -- UInt16 -> SMALLINT
			height BIGINT NOT NULL,
			local_pool BIGINT NOT NULL DEFAULT 0,
			remote_pool BIGINT NOT NULL DEFAULT 0,
			price_e6 BIGINT NOT NULL DEFAULT 0,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			price_delta BIGINT NOT NULL DEFAULT 0,        -- Int64 -> BIGINT
			local_pool_delta BIGINT NOT NULL DEFAULT 0,   -- Int64 -> BIGINT
			remote_pool_delta BIGINT NOT NULL DEFAULT 0,  -- Int64 -> BIGINT
			PRIMARY KEY (local_chain_id, remote_chain_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_prices_height ON dex_prices(height);
		CREATE INDEX IF NOT EXISTS idx_dex_prices_time ON dex_prices(height_time);
	`

	return db.Exec(ctx, query)
}

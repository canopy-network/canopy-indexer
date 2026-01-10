package crosschain

import (
	"context"
	"fmt"
)

// initViews creates all views
func (db *DB) initViews(ctx context.Context) error {
	views := []func(context.Context) error{
		db.createAccountsLatestView,
		db.createValidatorsLatestView,
		db.createCommitteesLatestView,
		db.createPoolsLatestView,
		db.createPoolPointsByHolderLatestView,
		db.createOrdersLatestView,
		db.createDexOrdersLatestView,
		db.createDexDepositsLatestView,
		db.createDexWithdrawalsLatestView,
		db.createParamsLatestView,
		db.createSupplyLatestView,
		db.createLpPositionSnapshotsView,
	}

	for _, createView := range views {
		if err := createView(ctx); err != nil {
			return err
		}
	}

	return nil
}

// createAccountsLatestView creates the accounts_latest view
func (db *DB) createAccountsLatestView(ctx context.Context) error {
	accountsTable := db.SchemaTable("accounts")
	accountsLatestView := db.SchemaTable("accounts_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, address) *
		FROM %s
		ORDER BY chain_id, address, height DESC
	`, accountsLatestView, accountsTable)

	return db.Exec(ctx, query)
}

// createValidatorsLatestView creates the validators_latest view
func (db *DB) createValidatorsLatestView(ctx context.Context) error {
	validatorsTable := db.SchemaTable("validators")
	validatorsLatestView := db.SchemaTable("validators_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, address) *
		FROM %s
		ORDER BY chain_id, address, height DESC
	`, validatorsLatestView, validatorsTable)

	return db.Exec(ctx, query)
}

// createCommitteesLatestView creates the committees_latest view
func (db *DB) createCommitteesLatestView(ctx context.Context) error {
	committeesTable := db.SchemaTable("committees")
	committeesLatestView := db.SchemaTable("committees_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, committee_chain_id) *
		FROM %s
		ORDER BY chain_id, committee_chain_id, height DESC
	`, committeesLatestView, committeesTable)

	return db.Exec(ctx, query)
}

// createPoolsLatestView creates the pools_latest view
func (db *DB) createPoolsLatestView(ctx context.Context) error {
	poolsTable := db.SchemaTable("pools")
	poolsLatestView := db.SchemaTable("pools_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, pool_id) *
		FROM %s
		ORDER BY chain_id, pool_id, height DESC
	`, poolsLatestView, poolsTable)

	return db.Exec(ctx, query)
}

// createPoolPointsByHolderLatestView creates the pool_points_by_holder_latest view
func (db *DB) createPoolPointsByHolderLatestView(ctx context.Context) error {
	poolPointsByHolderTable := db.SchemaTable("pool_points_by_holder")
	poolPointsByHolderLatestView := db.SchemaTable("pool_points_by_holder_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, address, pool_id) *
		FROM %s
		ORDER BY chain_id, address, pool_id, height DESC
	`, poolPointsByHolderLatestView, poolPointsByHolderTable)

	return db.Exec(ctx, query)
}

// createOrdersLatestView creates the orders_latest view
func (db *DB) createOrdersLatestView(ctx context.Context) error {
	ordersTable := db.SchemaTable("orders")
	ordersLatestView := db.SchemaTable("orders_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM %s
		ORDER BY chain_id, order_id, height DESC
	`, ordersLatestView, ordersTable)

	return db.Exec(ctx, query)
}

// createDexOrdersLatestView creates the dex_orders_latest view
func (db *DB) createDexOrdersLatestView(ctx context.Context) error {
	dexOrdersTable := db.SchemaTable("dex_orders")
	dexOrdersLatestView := db.SchemaTable("dex_orders_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM %s
		ORDER BY chain_id, order_id, height DESC
	`, dexOrdersLatestView, dexOrdersTable)

	return db.Exec(ctx, query)
}

// createDexDepositsLatestView creates the dex_deposits_latest view
func (db *DB) createDexDepositsLatestView(ctx context.Context) error {
	dexDepositsTable := db.SchemaTable("dex_deposits")
	dexDepositsLatestView := db.SchemaTable("dex_deposits_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM %s
		ORDER BY chain_id, order_id, height DESC
	`, dexDepositsLatestView, dexDepositsTable)

	return db.Exec(ctx, query)
}

// createDexWithdrawalsLatestView creates the dex_withdrawals_latest view
func (db *DB) createDexWithdrawalsLatestView(ctx context.Context) error {
	dexWithdrawalsTable := db.SchemaTable("dex_withdrawals")
	dexWithdrawalsLatestView := db.SchemaTable("dex_withdrawals_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM %s
		ORDER BY chain_id, order_id, height DESC
	`, dexWithdrawalsLatestView, dexWithdrawalsTable)

	return db.Exec(ctx, query)
}

// createParamsLatestView creates the params_latest view
func (db *DB) createParamsLatestView(ctx context.Context) error {
	paramsTable := db.SchemaTable("params")
	paramsLatestView := db.SchemaTable("params_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id) *
		FROM %s
		ORDER BY chain_id, height DESC
	`, paramsLatestView, paramsTable)

	return db.Exec(ctx, query)
}

// createSupplyLatestView creates the supply_latest view
func (db *DB) createSupplyLatestView(ctx context.Context) error {
	supplyTable := db.SchemaTable("supply")
	supplyLatestView := db.SchemaTable("supply_latest")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		SELECT DISTINCT ON (chain_id) *
		FROM %s
		ORDER BY chain_id, height DESC
	`, supplyLatestView, supplyTable)

	return db.Exec(ctx, query)
}

// createLpPositionSnapshotsView creates the lp_position_snapshots view
func (db *DB) createLpPositionSnapshotsView(ctx context.Context) error {
	poolPointsByHolderTable := db.SchemaTable("pool_points_by_holder")
	poolsTable := db.SchemaTable("pools")
	lpPositionSnapshotsView := db.SchemaTable("lp_position_snapshots")
	query := fmt.Sprintf(`
		CREATE OR REPLACE VIEW %s AS
		WITH latest_positions AS (
			-- Most recent record per holder/pool
			SELECT DISTINCT ON (chain_id, address, pool_id)
				chain_id, address, pool_id, committee,
				points, liquidity_pool_points, liquidity_pool_id,
				height, height_time
			FROM %s
			ORDER BY chain_id, address, pool_id, height DESC
		),
		first_seen AS (
			-- Position creation date
			SELECT chain_id, address, pool_id, MIN(height_time) as created_date
			FROM %s
			GROUP BY chain_id, address, pool_id
		),
		pool_totals AS (
			-- Current pool total_points
			SELECT DISTINCT ON (chain_id, pool_id)
				chain_id, pool_id, total_points
			FROM %s
			ORDER BY chain_id, pool_id, height DESC
		)
		SELECT
			lp.chain_id,
			lp.address,
			lp.pool_id,
			lp.committee,
			lp.points,
			lp.liquidity_pool_points,
			lp.liquidity_pool_id,
			CASE WHEN pt.total_points > 0
				 THEN (lp.points::numeric / pt.total_points * 100 * 1e6)::bigint
				 ELSE 0
			END as pool_share_ppm,  -- parts per million (divide by 1e6 for percentage)
			fs.created_date as position_created_date,
			lp.points > 0 as is_position_active,
			lp.height,
			lp.height_time
		FROM latest_positions lp
		JOIN first_seen fs USING (chain_id, address, pool_id)
		LEFT JOIN pool_totals pt USING (chain_id, pool_id);

		COMMENT ON VIEW %s IS 'Real-time LP position data derived from pool_points_by_holder';
	`, lpPositionSnapshotsView, poolPointsByHolderTable, poolPointsByHolderTable, poolsTable, lpPositionSnapshotsView)

	return db.Exec(ctx, query)
}

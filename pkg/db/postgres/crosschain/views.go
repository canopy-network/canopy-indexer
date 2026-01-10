package crosschain

import (
	"context"
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
	query := `
		CREATE OR REPLACE VIEW accounts_latest AS
		SELECT DISTINCT ON (chain_id, address) *
		FROM accounts
		ORDER BY chain_id, address, height DESC
	`

	return db.Exec(ctx, query)
}

// createValidatorsLatestView creates the validators_latest view
func (db *DB) createValidatorsLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW validators_latest AS
		SELECT DISTINCT ON (chain_id, address) *
		FROM validators
		ORDER BY chain_id, address, height DESC
	`

	return db.Exec(ctx, query)
}

// createCommitteesLatestView creates the committees_latest view
func (db *DB) createCommitteesLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW committees_latest AS
		SELECT DISTINCT ON (chain_id, committee_chain_id) *
		FROM committees
		ORDER BY chain_id, committee_chain_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createPoolsLatestView creates the pools_latest view
func (db *DB) createPoolsLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW pools_latest AS
		SELECT DISTINCT ON (chain_id, pool_id) *
		FROM pools
		ORDER BY chain_id, pool_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createPoolPointsByHolderLatestView creates the pool_points_by_holder_latest view
func (db *DB) createPoolPointsByHolderLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW pool_points_by_holder_latest AS
		SELECT DISTINCT ON (chain_id, address, pool_id) *
		FROM pool_points_by_holder
		ORDER BY chain_id, address, pool_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createOrdersLatestView creates the orders_latest view
func (db *DB) createOrdersLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW orders_latest AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM orders
		ORDER BY chain_id, order_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createDexOrdersLatestView creates the dex_orders_latest view
func (db *DB) createDexOrdersLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW dex_orders_latest AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM dex_orders
		ORDER BY chain_id, order_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createDexDepositsLatestView creates the dex_deposits_latest view
func (db *DB) createDexDepositsLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW dex_deposits_latest AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM dex_deposits
		ORDER BY chain_id, order_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createDexWithdrawalsLatestView creates the dex_withdrawals_latest view
func (db *DB) createDexWithdrawalsLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW dex_withdrawals_latest AS
		SELECT DISTINCT ON (chain_id, order_id) *
		FROM dex_withdrawals
		ORDER BY chain_id, order_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createParamsLatestView creates the params_latest view
func (db *DB) createParamsLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW params_latest AS
		SELECT DISTINCT ON (chain_id) *
		FROM params
		ORDER BY chain_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createSupplyLatestView creates the supply_latest view
func (db *DB) createSupplyLatestView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW supply_latest AS
		SELECT DISTINCT ON (chain_id) *
		FROM supply
		ORDER BY chain_id, height DESC
	`

	return db.Exec(ctx, query)
}

// createLpPositionSnapshotsView creates the lp_position_snapshots view
func (db *DB) createLpPositionSnapshotsView(ctx context.Context) error {
	query := `
		CREATE OR REPLACE VIEW lp_position_snapshots AS
		WITH latest_positions AS (
			-- Most recent record per holder/pool
			SELECT DISTINCT ON (chain_id, address, pool_id)
				chain_id, address, pool_id, committee,
				points, liquidity_pool_points, liquidity_pool_id,
				height, height_time
			FROM pool_points_by_holder
			ORDER BY chain_id, address, pool_id, height DESC
		),
		first_seen AS (
			-- Position creation date
			SELECT chain_id, address, pool_id, MIN(height_time) as created_date
			FROM pool_points_by_holder
			GROUP BY chain_id, address, pool_id
		),
		pool_totals AS (
			-- Current pool total_points
			SELECT DISTINCT ON (chain_id, pool_id)
				chain_id, pool_id, total_points
			FROM pools
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

		COMMENT ON VIEW lp_position_snapshots IS 'Real-time LP position data derived from pool_points_by_holder';
	`

	return db.Exec(ctx, query)
}

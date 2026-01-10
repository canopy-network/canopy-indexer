package chain

import (
	"context"
)

// initBlocks creates the blocks table
func (db *DB) initBlocks(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS blocks (
			height BIGINT PRIMARY KEY,
			hash TEXT NOT NULL,
			time TIMESTAMP WITH TIME ZONE NOT NULL,
			network_id INTEGER NOT NULL,
			parent_hash TEXT,
			proposer_address TEXT,
			size INTEGER,
			num_txs BIGINT NOT NULL DEFAULT 0,
			total_txs BIGINT NOT NULL DEFAULT 0,
			total_vdf_iterations INTEGER NOT NULL DEFAULT 0,
			state_root TEXT,
			transaction_root TEXT,
			validator_root TEXT,
			next_validator_root TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_blocks_time ON blocks(time);
		CREATE INDEX IF NOT EXISTS idx_blocks_network_id ON blocks(network_id);
		CREATE INDEX IF NOT EXISTS idx_blocks_proposer ON blocks(proposer_address);
	`

	return db.Exec(ctx, query)
}

// initBlockSummaries creates the block_summaries table matching indexer.BlockSummary
// This matches pkg/db/models/indexer/block_summary.go:119-241 (90+ fields)
func (db *DB) initBlockSummaries(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS block_summaries (
			-- Block metadata
			height BIGINT PRIMARY KEY,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			total_transactions BIGINT NOT NULL DEFAULT 0,

			-- ========== Transactions (20 fields) ==========
			num_txs INTEGER NOT NULL DEFAULT 0,
			num_txs_send INTEGER NOT NULL DEFAULT 0,
			num_txs_stake INTEGER NOT NULL DEFAULT 0,
			num_txs_unstake INTEGER NOT NULL DEFAULT 0,
			num_txs_edit_stake INTEGER NOT NULL DEFAULT 0,
			num_txs_start_poll INTEGER NOT NULL DEFAULT 0,
			num_txs_vote_poll INTEGER NOT NULL DEFAULT 0,
			num_txs_lock_order INTEGER NOT NULL DEFAULT 0,
			num_txs_close_order INTEGER NOT NULL DEFAULT 0,
			num_txs_unknown INTEGER NOT NULL DEFAULT 0,
			num_txs_pause INTEGER NOT NULL DEFAULT 0,
			num_txs_unpause INTEGER NOT NULL DEFAULT 0,
			num_txs_change_parameter INTEGER NOT NULL DEFAULT 0,
			num_txs_dao_transfer INTEGER NOT NULL DEFAULT 0,
			num_txs_certificate_results INTEGER NOT NULL DEFAULT 0,
			num_txs_subsidy INTEGER NOT NULL DEFAULT 0,
			num_txs_create_order INTEGER NOT NULL DEFAULT 0,
			num_txs_edit_order INTEGER NOT NULL DEFAULT 0,
			num_txs_delete_order INTEGER NOT NULL DEFAULT 0,
			num_txs_dex_limit_order INTEGER NOT NULL DEFAULT 0,
			num_txs_dex_liquidity_deposit INTEGER NOT NULL DEFAULT 0,
			num_txs_dex_liquidity_withdraw INTEGER NOT NULL DEFAULT 0,

			-- ========== Accounts (2 fields) ==========
			num_accounts INTEGER NOT NULL DEFAULT 0,
			num_accounts_new INTEGER NOT NULL DEFAULT 0,

			-- ========== Events (10 fields) ==========
			num_events INTEGER NOT NULL DEFAULT 0,
			num_events_reward INTEGER NOT NULL DEFAULT 0,
			num_events_slash INTEGER NOT NULL DEFAULT 0,
			num_events_dex_liquidity_deposit INTEGER NOT NULL DEFAULT 0,
			num_events_dex_liquidity_withdraw INTEGER NOT NULL DEFAULT 0,
			num_events_dex_swap INTEGER NOT NULL DEFAULT 0,
			num_events_order_book_swap INTEGER NOT NULL DEFAULT 0,
			num_events_automatic_pause INTEGER NOT NULL DEFAULT 0,
			num_events_automatic_begin_unstaking INTEGER NOT NULL DEFAULT 0,
			num_events_automatic_finish_unstaking INTEGER NOT NULL DEFAULT 0,

			-- ========== Orders (5 fields) ==========
			num_orders INTEGER NOT NULL DEFAULT 0,
			num_orders_new INTEGER NOT NULL DEFAULT 0,
			num_orders_open INTEGER NOT NULL DEFAULT 0,
			num_orders_filled INTEGER NOT NULL DEFAULT 0,
			num_orders_cancelled INTEGER NOT NULL DEFAULT 0,

			-- ========== Pools (2 fields) ==========
			num_pools INTEGER NOT NULL DEFAULT 0,
			num_pools_new INTEGER NOT NULL DEFAULT 0,

			-- ========== DexPrices (1 field) ==========
			num_dex_prices INTEGER NOT NULL DEFAULT 0,

			-- ========== DexOrders (6 fields) ==========
			num_dex_orders INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_future INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_locked INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_complete INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_success INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_failed INTEGER NOT NULL DEFAULT 0,

			-- ========== DexDeposits (4 fields) ==========
			num_dex_deposits INTEGER NOT NULL DEFAULT 0,
			num_dex_deposits_pending INTEGER NOT NULL DEFAULT 0,
			num_dex_deposits_locked INTEGER NOT NULL DEFAULT 0,
			num_dex_deposits_complete INTEGER NOT NULL DEFAULT 0,

			-- ========== DexWithdrawals (4 fields) ==========
			num_dex_withdrawals INTEGER NOT NULL DEFAULT 0,
			num_dex_withdrawals_pending INTEGER NOT NULL DEFAULT 0,
			num_dex_withdrawals_locked INTEGER NOT NULL DEFAULT 0,
			num_dex_withdrawals_complete INTEGER NOT NULL DEFAULT 0,

			-- ========== PoolPointsByHolder (2 fields) ==========
			num_pool_points_holders INTEGER NOT NULL DEFAULT 0,
			num_pool_points_holders_new INTEGER NOT NULL DEFAULT 0,

			-- ========== Params (1 field) ==========
			params_changed BOOLEAN NOT NULL DEFAULT false,

			-- ========== Validators (5 fields) ==========
			num_validators INTEGER NOT NULL DEFAULT 0,
			num_validators_new INTEGER NOT NULL DEFAULT 0,
			num_validators_active INTEGER NOT NULL DEFAULT 0,
			num_validators_paused INTEGER NOT NULL DEFAULT 0,
			num_validators_unstaking INTEGER NOT NULL DEFAULT 0,

			-- ========== ValidatorNonSigningInfo (2 fields) ==========
			num_validator_non_signing_info INTEGER NOT NULL DEFAULT 0,
			num_validator_non_signing_info_new INTEGER NOT NULL DEFAULT 0,

			-- ========== ValidatorDoubleSigningInfo (1 field) ==========
			num_validator_double_signing_info INTEGER NOT NULL DEFAULT 0,

			-- ========== Committees (4 fields) ==========
			num_committees INTEGER NOT NULL DEFAULT 0,
			num_committees_new INTEGER NOT NULL DEFAULT 0,
			num_committees_subsidized INTEGER NOT NULL DEFAULT 0,
			num_committees_retired INTEGER NOT NULL DEFAULT 0,

			-- ========== CommitteeValidators (1 field) ==========
			num_committee_validators INTEGER NOT NULL DEFAULT 0,

			-- ========== CommitteePayments (1 field) ==========
			num_committee_payments INTEGER NOT NULL DEFAULT 0,

			-- ========== Supply (4 fields) ==========
			supply_changed BOOLEAN NOT NULL DEFAULT false,
			supply_total BIGINT NOT NULL DEFAULT 0,
			supply_staked BIGINT NOT NULL DEFAULT 0,
			supply_delegated_only BIGINT NOT NULL DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_block_summaries_time ON block_summaries(height_time);
	`

	return db.Exec(ctx, query)
}

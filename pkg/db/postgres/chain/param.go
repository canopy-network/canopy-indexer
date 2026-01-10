package chain

import (
	"context"
)

// initParams creates the params table matching indexer.Params
// This matches pkg/db/models/indexer/params.go:66-118 (42 fields)
// Only inserted when parameters change
func (db *DB) initParams(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS params (
			height BIGINT PRIMARY KEY,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			block_size BIGINT NOT NULL DEFAULT 0,
			protocol_version TEXT DEFAULT '',
			root_chain_id SMALLINT NOT NULL DEFAULT 0,           -- UInt16 -> SMALLINT
			retired BIGINT NOT NULL DEFAULT 0,
			unstaking_blocks BIGINT NOT NULL DEFAULT 0,
			max_pause_blocks BIGINT NOT NULL DEFAULT 0,
			double_sign_slash_percentage BIGINT NOT NULL DEFAULT 0,
			non_sign_slash_percentage BIGINT NOT NULL DEFAULT 0,
			max_non_sign BIGINT NOT NULL DEFAULT 0,
			non_sign_window BIGINT NOT NULL DEFAULT 0,
			max_committees BIGINT NOT NULL DEFAULT 0,
			max_committee_size BIGINT NOT NULL DEFAULT 0,
			early_withdrawal_penalty BIGINT NOT NULL DEFAULT 0,
			delegate_unstaking_blocks BIGINT NOT NULL DEFAULT 0,
			minimum_order_size BIGINT NOT NULL DEFAULT 0,
			stake_percent_for_subsidized BIGINT NOT NULL DEFAULT 0,
			max_slash_per_committee BIGINT NOT NULL DEFAULT 0,
			delegate_reward_percentage BIGINT NOT NULL DEFAULT 0,
			buy_deadline_blocks BIGINT NOT NULL DEFAULT 0,
			lock_order_fee_multiplier BIGINT NOT NULL DEFAULT 0,
			minimum_stake_for_validators BIGINT NOT NULL DEFAULT 0,
			minimum_stake_for_delegates BIGINT NOT NULL DEFAULT 0,
			maximum_delegates_per_committee BIGINT NOT NULL DEFAULT 0,
			send_fee BIGINT NOT NULL DEFAULT 0,
			stake_fee BIGINT NOT NULL DEFAULT 0,
			edit_stake_fee BIGINT NOT NULL DEFAULT 0,
			unstake_fee BIGINT NOT NULL DEFAULT 0,
			pause_fee BIGINT NOT NULL DEFAULT 0,
			unpause_fee BIGINT NOT NULL DEFAULT 0,
			change_parameter_fee BIGINT NOT NULL DEFAULT 0,
			dao_transfer_fee BIGINT NOT NULL DEFAULT 0,
			certificate_results_fee BIGINT NOT NULL DEFAULT 0,
			subsidy_fee BIGINT NOT NULL DEFAULT 0,
			create_order_fee BIGINT NOT NULL DEFAULT 0,
			edit_order_fee BIGINT NOT NULL DEFAULT 0,
			delete_order_fee BIGINT NOT NULL DEFAULT 0,
			dex_limit_order_fee BIGINT NOT NULL DEFAULT 0,
			dex_liquidity_deposit_fee BIGINT NOT NULL DEFAULT 0,
			dex_liquidity_withdraw_fee BIGINT NOT NULL DEFAULT 0,
			dao_reward_percentage BIGINT NOT NULL DEFAULT 0
		);
	`

	return db.Exec(ctx, query)
}

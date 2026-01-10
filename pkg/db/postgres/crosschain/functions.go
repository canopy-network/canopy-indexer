package crosschain

import (
	"context"
)

// initFunctions creates all PL/pgSQL functions
func (db *DB) initFunctions(ctx context.Context) error {
	functions := []func(context.Context) error{
		db.createUpdateIndexProgressFunction,
		db.createBuildBlockSummaryFunction,
	}

	for _, createFunc := range functions {
		if err := createFunc(ctx); err != nil {
			return err
		}
	}

	return nil
}

// createUpdateIndexProgressFunction creates the update_index_progress function
func (db *DB) createUpdateIndexProgressFunction(ctx context.Context) error {
	query := `
		CREATE OR REPLACE FUNCTION update_index_progress(
			p_chain_id BIGINT,
			p_height BIGINT
		) RETURNS VOID AS $$
		BEGIN
			INSERT INTO index_progress (chain_id, last_height, last_indexed_at, updated_at)
			VALUES (p_chain_id, p_height, NOW(), NOW())
			ON CONFLICT (chain_id) DO UPDATE SET
				last_height = GREATEST(index_progress.last_height, EXCLUDED.last_height),
				last_indexed_at = NOW(),
				updated_at = NOW();
		END;
		$$ LANGUAGE plpgsql;
	`

	return db.Exec(ctx, query)
}

// createBuildBlockSummaryFunction creates the build_block_summary function
// This function aggregates data from all tables to build comprehensive block summaries
func (db *DB) createBuildBlockSummaryFunction(ctx context.Context) error {
	query := `
		CREATE OR REPLACE FUNCTION build_block_summary(
			p_chain_id BIGINT,
			p_height BIGINT,
			p_height_time TIMESTAMPTZ
		) RETURNS VOID AS $$
		DECLARE
			-- Transaction counts
			v_total_transactions BIGINT;
			v_num_txs INT;
			v_num_txs_send INT;
			v_num_txs_stake INT;
			v_num_txs_unstake INT;
			v_num_txs_edit_stake INT;
			v_num_txs_start_poll INT;
			v_num_txs_vote_poll INT;
			v_num_txs_lock_order INT;
			v_num_txs_close_order INT;
			v_num_txs_unknown INT;
			v_num_txs_pause INT;
			v_num_txs_unpause INT;
			v_num_txs_change_parameter INT;
			v_num_txs_dao_transfer INT;
			v_num_txs_certificate_results INT;
			v_num_txs_subsidy INT;
			v_num_txs_create_order INT;
			v_num_txs_edit_order INT;
			v_num_txs_delete_order INT;
			v_num_txs_dex_limit_order INT;
			v_num_txs_dex_liquidity_deposit INT;
			v_num_txs_dex_liquidity_withdraw INT;

			-- Account counts
			v_num_accounts INT;
			v_num_accounts_new INT;

			-- Event counts
			v_num_events INT;
			v_num_events_reward INT;
			v_num_events_slash INT;
			v_num_events_dex_liquidity_deposit INT;
			v_num_events_dex_liquidity_withdraw INT;
			v_num_events_dex_swap INT;
			v_num_events_order_book_swap INT;
			v_num_events_automatic_pause INT;
			v_num_events_automatic_begin_unstaking INT;
			v_num_events_automatic_finish_unstaking INT;

			-- Validator counts
			v_num_validators INT;
			v_num_validators_new INT;
			v_num_validators_active INT;
			v_num_validators_paused INT;
			v_num_validators_unstaking INT;
			v_num_validator_non_signing_info INT;
			v_num_validator_non_signing_info_new INT;
			v_num_validator_double_signing_info INT;

			-- Committee counts
			v_num_committees INT;
			v_num_committees_new INT;
			v_num_committees_subsidized INT;
			v_num_committees_retired INT;
			v_num_committee_validators INT;
			v_num_committee_payments INT;

			-- Pool counts
			v_num_pools INT;
			v_num_pools_new INT;
			v_num_pool_points_holders INT;
			v_num_pool_points_holders_new INT;

			-- Order counts
			v_num_orders INT;
			v_num_orders_new INT;
			v_num_orders_open INT;
			v_num_orders_filled INT;
			v_num_orders_cancelled INT;

			-- DEX counts
			v_num_dex_prices INT;
			v_num_dex_orders INT;
			v_num_dex_orders_future INT;
			v_num_dex_orders_locked INT;
			v_num_dex_orders_complete INT;
			v_num_dex_orders_success INT;
			v_num_dex_orders_failed INT;
			v_num_dex_deposits INT;
			v_num_dex_deposits_pending INT;
			v_num_dex_deposits_locked INT;
			v_num_dex_deposits_complete INT;
			v_num_dex_withdrawals INT;
			v_num_dex_withdrawals_pending INT;
			v_num_dex_withdrawals_locked INT;
			v_num_dex_withdrawals_complete INT;

			-- Params and supply
			v_params_changed BOOLEAN;
			v_supply_changed BOOLEAN;
			v_supply_total BIGINT;
			v_supply_staked BIGINT;
			v_supply_delegated_only BIGINT;
		BEGIN
			-- ========================================================================
			-- TRANSACTIONS (from blocks table for total, txs table for counts by type)
			-- ========================================================================
			SELECT COALESCE(total_txs, 0) INTO v_total_transactions
			FROM blocks WHERE chain_id = p_chain_id AND height = p_height;

			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE message_type = 'send'),
				COUNT(*) FILTER (WHERE message_type = 'stake'),
				COUNT(*) FILTER (WHERE message_type = 'unstake'),
				COUNT(*) FILTER (WHERE message_type = 'editStake'),
				COUNT(*) FILTER (WHERE message_type = 'startPoll'),
				COUNT(*) FILTER (WHERE message_type = 'votePoll'),
				COUNT(*) FILTER (WHERE message_type = 'lockOrder'),
				COUNT(*) FILTER (WHERE message_type = 'closeOrder'),
				COUNT(*) FILTER (WHERE message_type NOT IN (
					'send', 'stake', 'unstake', 'editStake', 'startPoll', 'votePoll',
					'lockOrder', 'closeOrder', 'pause', 'unpause', 'changeParameter',
					'daoTransfer', 'certificateResults', 'subsidy', 'createOrder',
					'editOrder', 'deleteOrder', 'dexLimitOrder', 'dexLiquidityDeposit',
					'dexLiquidityWithdraw'
				)),
				COUNT(*) FILTER (WHERE message_type = 'pause'),
				COUNT(*) FILTER (WHERE message_type = 'unpause'),
				COUNT(*) FILTER (WHERE message_type = 'changeParameter'),
				COUNT(*) FILTER (WHERE message_type = 'daoTransfer'),
				COUNT(*) FILTER (WHERE message_type = 'certificateResults'),
				COUNT(*) FILTER (WHERE message_type = 'subsidy'),
				COUNT(*) FILTER (WHERE message_type = 'createOrder'),
				COUNT(*) FILTER (WHERE message_type = 'editOrder'),
				COUNT(*) FILTER (WHERE message_type = 'deleteOrder'),
				COUNT(*) FILTER (WHERE message_type = 'dexLimitOrder'),
				COUNT(*) FILTER (WHERE message_type = 'dexLiquidityDeposit'),
				COUNT(*) FILTER (WHERE message_type = 'dexLiquidityWithdraw')
			INTO
				v_num_txs, v_num_txs_send, v_num_txs_stake, v_num_txs_unstake,
				v_num_txs_edit_stake, v_num_txs_start_poll, v_num_txs_vote_poll,
				v_num_txs_lock_order, v_num_txs_close_order, v_num_txs_unknown,
				v_num_txs_pause, v_num_txs_unpause, v_num_txs_change_parameter,
				v_num_txs_dao_transfer, v_num_txs_certificate_results, v_num_txs_subsidy,
				v_num_txs_create_order, v_num_txs_edit_order, v_num_txs_delete_order,
				v_num_txs_dex_limit_order, v_num_txs_dex_liquidity_deposit,
				v_num_txs_dex_liquidity_withdraw
			FROM txs WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- ACCOUNTS (count at H, new = at H but not at H-1)
			-- ========================================================================
			SELECT COUNT(*) INTO v_num_accounts
			FROM accounts WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_accounts_new
			FROM accounts a
			WHERE a.chain_id = p_chain_id AND a.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM accounts prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.address = a.address
			  );

			-- ========================================================================
			-- EVENTS
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE event_type = 'reward'),
				COUNT(*) FILTER (WHERE event_type = 'slash'),
				COUNT(*) FILTER (WHERE event_type = 'dex-liquidity-deposit'),
				COUNT(*) FILTER (WHERE event_type = 'dex-liquidity-withdraw'),
				COUNT(*) FILTER (WHERE event_type = 'dex-swap'),
				COUNT(*) FILTER (WHERE event_type = 'order-book-swap'),
				COUNT(*) FILTER (WHERE event_type = 'automatic-pause'),
				COUNT(*) FILTER (WHERE event_type = 'automatic-begin-unstaking'),
				COUNT(*) FILTER (WHERE event_type = 'automatic-finish-unstaking')
			INTO
				v_num_events, v_num_events_reward, v_num_events_slash,
				v_num_events_dex_liquidity_deposit, v_num_events_dex_liquidity_withdraw,
				v_num_events_dex_swap, v_num_events_order_book_swap,
				v_num_events_automatic_pause, v_num_events_automatic_begin_unstaking,
				v_num_events_automatic_finish_unstaking
			FROM events WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- VALIDATORS
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE status = 'active'),
				COUNT(*) FILTER (WHERE status = 'paused'),
				COUNT(*) FILTER (WHERE status = 'unstaking')
			INTO v_num_validators, v_num_validators_active, v_num_validators_paused, v_num_validators_unstaking
			FROM validators WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_validators_new
			FROM validators v
			WHERE v.chain_id = p_chain_id AND v.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM validators prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.address = v.address
			  );

			-- Validator non-signing info
			SELECT COUNT(*) INTO v_num_validator_non_signing_info
			FROM validator_non_signing_info WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_validator_non_signing_info_new
			FROM validator_non_signing_info v
			WHERE v.chain_id = p_chain_id AND v.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM validator_non_signing_info prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.address = v.address
			  );

			-- Validator double-signing info
			SELECT COUNT(*) INTO v_num_validator_double_signing_info
			FROM validator_double_signing_info WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- COMMITTEES
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE subsidized = true),
				COUNT(*) FILTER (WHERE retired = true)
			INTO v_num_committees, v_num_committees_subsidized, v_num_committees_retired
			FROM committees WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_committees_new
			FROM committees c
			WHERE c.chain_id = p_chain_id AND c.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM committees prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.committee_chain_id = c.committee_chain_id
			  );

			-- Committee validators
			SELECT COUNT(*) INTO v_num_committee_validators
			FROM committee_validators WHERE chain_id = p_chain_id AND height = p_height;

			-- Committee payments
			SELECT COUNT(*) INTO v_num_committee_payments
			FROM committee_payments WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- POOLS
			-- ========================================================================
			SELECT COUNT(*) INTO v_num_pools
			FROM pools WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_pools_new
			FROM pools p
			WHERE p.chain_id = p_chain_id AND p.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM pools prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.pool_id = p.pool_id
			  );

			-- Pool points holders
			SELECT COUNT(*) INTO v_num_pool_points_holders
			FROM pool_points_by_holder WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_pool_points_holders_new
			FROM pool_points_by_holder h
			WHERE h.chain_id = p_chain_id AND h.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM pool_points_by_holder prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.address = h.address
					AND prev.pool_id = h.pool_id
			  );

			-- ========================================================================
			-- ORDERS (order book) - enum values: open, complete, canceled
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE status = 'open'),
				COUNT(*) FILTER (WHERE status = 'complete'),
				COUNT(*) FILTER (WHERE status = 'canceled')
			INTO v_num_orders, v_num_orders_open, v_num_orders_filled, v_num_orders_cancelled
			FROM orders WHERE chain_id = p_chain_id AND height = p_height;

			SELECT COUNT(*) INTO v_num_orders_new
			FROM orders o
			WHERE o.chain_id = p_chain_id AND o.height = p_height
			  AND NOT EXISTS (
				  SELECT 1 FROM orders prev
				  WHERE prev.chain_id = p_chain_id
					AND prev.height = p_height - 1
					AND prev.order_id = o.order_id
			  );

			-- ========================================================================
			-- DEX PRICES
			-- ========================================================================
			SELECT COUNT(*) INTO v_num_dex_prices
			FROM dex_prices WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- DEX ORDERS
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE state = 'future'),
				COUNT(*) FILTER (WHERE state = 'locked'),
				COUNT(*) FILTER (WHERE state = 'complete'),
				COUNT(*) FILTER (WHERE state = 'complete' AND success = true),
				COUNT(*) FILTER (WHERE state = 'complete' AND success = false)
			INTO
				v_num_dex_orders, v_num_dex_orders_future, v_num_dex_orders_locked,
				v_num_dex_orders_complete, v_num_dex_orders_success, v_num_dex_orders_failed
			FROM dex_orders WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- DEX DEPOSITS
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE state = 'pending'),
				COUNT(*) FILTER (WHERE state = 'locked'),
				COUNT(*) FILTER (WHERE state = 'complete')
			INTO v_num_dex_deposits, v_num_dex_deposits_pending, v_num_dex_deposits_locked, v_num_dex_deposits_complete
			FROM dex_deposits WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- DEX WITHDRAWALS
			-- ========================================================================
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE state = 'pending'),
				COUNT(*) FILTER (WHERE state = 'locked'),
				COUNT(*) FILTER (WHERE state = 'complete')
			INTO v_num_dex_withdrawals, v_num_dex_withdrawals_pending, v_num_dex_withdrawals_locked, v_num_dex_withdrawals_complete
			FROM dex_withdrawals WHERE chain_id = p_chain_id AND height = p_height;

			-- ========================================================================
			-- PARAMS (check if row exists at H - means params changed)
			-- ========================================================================
			SELECT EXISTS(
				SELECT 1 FROM params WHERE chain_id = p_chain_id AND height = p_height
			) INTO v_params_changed;

			-- ========================================================================
			-- SUPPLY (check if row exists at H, get values)
			-- ========================================================================
			SELECT
				true, total, staked, delegated_only
			INTO v_supply_changed, v_supply_total, v_supply_staked, v_supply_delegated_only
			FROM supply WHERE chain_id = p_chain_id AND height = p_height;

			IF NOT FOUND THEN
				v_supply_changed := false;
				v_supply_total := 0;
				v_supply_staked := 0;
				v_supply_delegated_only := 0;
			END IF;

			-- ========================================================================
			-- INSERT BLOCK SUMMARY
			-- ========================================================================
			INSERT INTO block_summaries (
				chain_id, height, height_time, total_transactions,
				-- Transactions
				num_txs, num_txs_send, num_txs_stake, num_txs_unstake, num_txs_edit_stake,
				num_txs_start_poll, num_txs_vote_poll, num_txs_lock_order, num_txs_close_order,
				num_txs_unknown, num_txs_pause, num_txs_unpause, num_txs_change_parameter,
				num_txs_dao_transfer, num_txs_certificate_results, num_txs_subsidy,
				num_txs_create_order, num_txs_edit_order, num_txs_delete_order,
				num_txs_dex_limit_order, num_txs_dex_liquidity_deposit, num_txs_dex_liquidity_withdraw,
				-- Accounts
				num_accounts, num_accounts_new,
				-- Events
				num_events, num_events_reward, num_events_slash,
				num_events_dex_liquidity_deposit, num_events_dex_liquidity_withdraw,
				num_events_dex_swap, num_events_order_book_swap, num_events_automatic_pause,
				num_events_automatic_begin_unstaking, num_events_automatic_finish_unstaking,
				-- Orders
				num_orders, num_orders_new, num_orders_open, num_orders_filled, num_orders_cancelled,
				-- Pools
				num_pools, num_pools_new,
				-- DEX prices
				num_dex_prices,
				-- DEX orders
				num_dex_orders, num_dex_orders_future, num_dex_orders_locked,
				num_dex_orders_complete, num_dex_orders_success, num_dex_orders_failed,
				-- DEX deposits
				num_dex_deposits, num_dex_deposits_pending, num_dex_deposits_locked, num_dex_deposits_complete,
				-- DEX withdrawals
				num_dex_withdrawals, num_dex_withdrawals_pending, num_dex_withdrawals_locked, num_dex_withdrawals_complete,
				-- Pool points holders
				num_pool_points_holders, num_pool_points_holders_new,
				-- Params
				params_changed,
				-- Validators
				num_validators, num_validators_new, num_validators_active, num_validators_paused, num_validators_unstaking,
				num_validator_non_signing_info, num_validator_non_signing_info_new, num_validator_double_signing_info,
				-- Committees
				num_committees, num_committees_new, num_committees_subsidized, num_committees_retired,
				num_committee_validators, num_committee_payments,
				-- Supply
				supply_changed, supply_total, supply_staked, supply_delegated_only
			) VALUES (
				p_chain_id, p_height, p_height_time, COALESCE(v_total_transactions, 0),
				-- Transactions
				COALESCE(v_num_txs, 0), COALESCE(v_num_txs_send, 0), COALESCE(v_num_txs_stake, 0),
				COALESCE(v_num_txs_unstake, 0), COALESCE(v_num_txs_edit_stake, 0),
				COALESCE(v_num_txs_start_poll, 0), COALESCE(v_num_txs_vote_poll, 0),
				COALESCE(v_num_txs_lock_order, 0), COALESCE(v_num_txs_close_order, 0),
				COALESCE(v_num_txs_unknown, 0), COALESCE(v_num_txs_pause, 0), COALESCE(v_num_txs_unpause, 0),
				COALESCE(v_num_txs_change_parameter, 0), COALESCE(v_num_txs_dao_transfer, 0),
				COALESCE(v_num_txs_certificate_results, 0), COALESCE(v_num_txs_subsidy, 0),
				COALESCE(v_num_txs_create_order, 0), COALESCE(v_num_txs_edit_order, 0),
				COALESCE(v_num_txs_delete_order, 0), COALESCE(v_num_txs_dex_limit_order, 0),
				COALESCE(v_num_txs_dex_liquidity_deposit, 0), COALESCE(v_num_txs_dex_liquidity_withdraw, 0),
				-- Accounts
				COALESCE(v_num_accounts, 0), COALESCE(v_num_accounts_new, 0),
				-- Events
				COALESCE(v_num_events, 0), COALESCE(v_num_events_reward, 0), COALESCE(v_num_events_slash, 0),
				COALESCE(v_num_events_dex_liquidity_deposit, 0), COALESCE(v_num_events_dex_liquidity_withdraw, 0),
				COALESCE(v_num_events_dex_swap, 0), COALESCE(v_num_events_order_book_swap, 0),
				COALESCE(v_num_events_automatic_pause, 0), COALESCE(v_num_events_automatic_begin_unstaking, 0),
				COALESCE(v_num_events_automatic_finish_unstaking, 0),
				-- Orders
				COALESCE(v_num_orders, 0), COALESCE(v_num_orders_new, 0), COALESCE(v_num_orders_open, 0),
				COALESCE(v_num_orders_filled, 0), COALESCE(v_num_orders_cancelled, 0),
				-- Pools
				COALESCE(v_num_pools, 0), COALESCE(v_num_pools_new, 0),
				-- DEX prices
				COALESCE(v_num_dex_prices, 0),
				-- DEX orders
				COALESCE(v_num_dex_orders, 0), COALESCE(v_num_dex_orders_future, 0),
				COALESCE(v_num_dex_orders_locked, 0), COALESCE(v_num_dex_orders_complete, 0),
				COALESCE(v_num_dex_orders_success, 0), COALESCE(v_num_dex_orders_failed, 0),
				-- DEX deposits
				COALESCE(v_num_dex_deposits, 0), COALESCE(v_num_dex_deposits_pending, 0),
				COALESCE(v_num_dex_deposits_locked, 0), COALESCE(v_num_dex_deposits_complete, 0),
				-- DEX withdrawals
				COALESCE(v_num_dex_withdrawals, 0), COALESCE(v_num_dex_withdrawals_pending, 0),
				COALESCE(v_num_dex_withdrawals_locked, 0), COALESCE(v_num_dex_withdrawals_complete, 0),
				-- Pool points holders
				COALESCE(v_num_pool_points_holders, 0), COALESCE(v_num_pool_points_holders_new, 0),
				-- Params
				COALESCE(v_params_changed, false),
				-- Validators
				COALESCE(v_num_validators, 0), COALESCE(v_num_validators_new, 0),
				COALESCE(v_num_validators_active, 0), COALESCE(v_num_validators_paused, 0),
				COALESCE(v_num_validators_unstaking, 0), COALESCE(v_num_validator_non_signing_info, 0),
				COALESCE(v_num_validator_non_signing_info_new, 0), COALESCE(v_num_validator_double_signing_info, 0),
				-- Committees
				COALESCE(v_num_committees, 0), COALESCE(v_num_committees_new, 0),
				COALESCE(v_num_committees_subsidized, 0), COALESCE(v_num_committees_retired, 0),
				COALESCE(v_num_committee_validators, 0), COALESCE(v_num_committee_payments, 0),
				-- Supply
				COALESCE(v_supply_changed, false), COALESCE(v_supply_total, 0),
				COALESCE(v_supply_staked, 0), COALESCE(v_supply_delegated_only, 0)
			)
			ON CONFLICT (chain_id, height) DO UPDATE SET
				total_transactions = EXCLUDED.total_transactions,
				num_txs = EXCLUDED.num_txs,
				num_accounts = EXCLUDED.num_accounts,
				num_events = EXCLUDED.num_events,
				num_validators = EXCLUDED.num_validators,
				num_committees = EXCLUDED.num_committees,
				num_pools = EXCLUDED.num_pools,
				supply_total = EXCLUDED.supply_total;
		END;
		$$ LANGUAGE plpgsql;
	`

	return db.Exec(ctx, query)
}

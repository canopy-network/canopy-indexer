package chain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
)

// GetBlock retrieves a block by height
func (db *DB) GetBlock(ctx context.Context, height uint64) (*indexermodels.Block, error) {
	query := fmt.Sprintf(`
		SELECT height, hash, timestamp, network_id, parent_hash, proposer_address, size,
		       num_txs, total_txs, total_vdf_iterations, state_root, transaction_root,
		       validator_root, next_validator_root
		FROM %s
		WHERE height = $1
	`, db.SchemaTable("blocks"))

	var block indexermodels.Block
	err := db.Client.Pool.QueryRow(ctx, query, height).Scan(
		&block.Height, &block.Hash, &block.Time, &block.NetworkID, &block.LastBlockHash, &block.ProposerAddress,
		&block.Size, &block.NumTxs, &block.TotalTxs, &block.TotalVDFIterations,
		&block.StateRoot, &block.TransactionRoot, &block.ValidatorRoot, &block.NextValidatorRoot,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("block not found at height %d", height)
		}
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return &block, nil
}

// GetBlockTime retrieves the timestamp of a block
func (db *DB) GetBlockTime(ctx context.Context, height uint64) (time.Time, error) {
	query := fmt.Sprintf(`SELECT timestamp FROM %s WHERE height = $1`, db.SchemaTable("blocks"))

	var blockTime time.Time
	err := db.Client.Pool.QueryRow(ctx, query, height).Scan(&blockTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, fmt.Errorf("block not found at height %d", height)
		}
		return time.Time{}, fmt.Errorf("failed to get block time: %w", err)
	}

	return blockTime, nil
}

// GetBlockSummary retrieves a block summary (staging parameter ignored for Postgres)
func (db *DB) GetBlockSummary(ctx context.Context, height uint64, staging bool) (*indexermodels.BlockSummary, error) {
	query := fmt.Sprintf(`
		SELECT height, height_time, total_transactions,
		       num_txs, num_txs_send, num_txs_stake, num_txs_unstake, num_txs_edit_stake,
		       num_txs_start_poll, num_txs_vote_poll, num_txs_lock_order, num_txs_close_order,
		       num_txs_unknown, num_txs_pause, num_txs_unpause, num_txs_change_parameter,
		       num_txs_dao_transfer, num_txs_certificate_results, num_txs_subsidy,
		       num_txs_create_order, num_txs_edit_order, num_txs_delete_order,
		       num_txs_dex_limit_order, num_txs_dex_liquidity_deposit, num_txs_dex_liquidity_withdraw,
		       num_accounts, num_accounts_new,
		       num_events, num_events_reward, num_events_slash,
		       num_events_dex_liquidity_deposit, num_events_dex_liquidity_withdraw, num_events_dex_swap,
		       num_events_order_book_swap, num_events_automatic_pause,
		       num_events_automatic_begin_unstaking, num_events_automatic_finish_unstaking,
		       num_orders, num_orders_new, num_orders_open, num_orders_filled, num_orders_cancelled,
		       num_pools, num_pools_new,
		       num_dex_prices,
		       num_dex_orders, num_dex_orders_future, num_dex_orders_locked, num_dex_orders_complete,
		       num_dex_orders_success, num_dex_orders_failed,
		       num_dex_deposits, num_dex_deposits_pending, num_dex_deposits_locked, num_dex_deposits_complete,
		       num_dex_withdrawals, num_dex_withdrawals_pending, num_dex_withdrawals_locked, num_dex_withdrawals_complete,
		       num_pool_points_holders, num_pool_points_holders_new,
		       params_changed,
		       num_validators, num_validators_new, num_validators_active, num_validators_paused, num_validators_unstaking,
		       num_validator_non_signing_info, num_validator_non_signing_info_new,
		       num_validator_double_signing_info,
		       num_committees, num_committees_new, num_committees_subsidized, num_committees_retired,
		       num_committee_validators,
		       num_committee_payments,
		       supply_changed, supply_total, supply_staked, supply_delegated_only
		FROM %s
		WHERE height = $1
	`, db.SchemaTable("block_summaries"))

	var summary indexermodels.BlockSummary
	err := db.Client.Pool.QueryRow(ctx, query, height).Scan(
		&summary.Height, &summary.HeightTime, &summary.TotalTransactions,
		&summary.NumTxs, &summary.NumTxsSend, &summary.NumTxsStake, &summary.NumTxsUnstake, &summary.NumTxsEditStake,
		&summary.NumTxsStartPoll, &summary.NumTxsVotePoll, &summary.NumTxsLockOrder, &summary.NumTxsCloseOrder,
		&summary.NumTxsUnknown, &summary.NumTxsPause, &summary.NumTxsUnpause, &summary.NumTxsChangeParameter,
		&summary.NumTxsDaoTransfer, &summary.NumTxsCertificateResults, &summary.NumTxsSubsidy,
		&summary.NumTxsCreateOrder, &summary.NumTxsEditOrder, &summary.NumTxsDeleteOrder,
		&summary.NumTxsDexLimitOrder, &summary.NumTxsDexLiquidityDeposit, &summary.NumTxsDexLiquidityWithdraw,
		&summary.NumAccounts, &summary.NumAccountsNew,
		&summary.NumEvents, &summary.NumEventsReward, &summary.NumEventsSlash,
		&summary.NumEventsDexLiquidityDeposit, &summary.NumEventsDexLiquidityWithdraw, &summary.NumEventsDexSwap,
		&summary.NumEventsOrderBookSwap, &summary.NumEventsAutomaticPause,
		&summary.NumEventsAutomaticBeginUnstaking, &summary.NumEventsAutomaticFinishUnstaking,
		&summary.NumOrders, &summary.NumOrdersNew, &summary.NumOrdersOpen, &summary.NumOrdersFilled, &summary.NumOrdersCancelled,
		&summary.NumPools, &summary.NumPoolsNew,
		&summary.NumDexPrices,
		&summary.NumDexOrders, &summary.NumDexOrdersFuture, &summary.NumDexOrdersLocked, &summary.NumDexOrdersComplete,
		&summary.NumDexOrdersSuccess, &summary.NumDexOrdersFailed,
		&summary.NumDexDeposits, &summary.NumDexDepositsPending, &summary.NumDexDepositsLocked, &summary.NumDexDepositsComplete,
		&summary.NumDexWithdrawals, &summary.NumDexWithdrawalsPending, &summary.NumDexWithdrawalsLocked, &summary.NumDexWithdrawalsComplete,
		&summary.NumDexPoolPointsHolders, &summary.NumDexPoolPointsHoldersNew,
		&summary.ParamsChanged,
		&summary.NumValidators, &summary.NumValidatorsNew, &summary.NumValidatorsActive, &summary.NumValidatorsPaused, &summary.NumValidatorsUnstaking,
		&summary.NumValidatorNonSigningInfo, &summary.NumValidatorNonSigningInfoNew,
		&summary.NumValidatorDoubleSigningInfo,
		&summary.NumCommittees, &summary.NumCommitteesNew, &summary.NumCommitteesSubsidized, &summary.NumCommitteesRetired,
		&summary.NumCommitteeValidators,
		&summary.NumCommitteePayments,
		&summary.SupplyChanged, &summary.SupplyTotal, &summary.SupplyStaked, &summary.SupplyDelegatedOnly,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("block summary not found at height %d", height)
		}
		return nil, fmt.Errorf("failed to get block summary: %w", err)
	}

	return &summary, nil
}

// HasBlock checks if a block exists at a given height
func (db *DB) HasBlock(ctx context.Context, height uint64) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE height = $1)`, db.SchemaTable("blocks"))

	var exists bool
	err := db.Client.Pool.QueryRow(ctx, query, height).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check block existence: %w", err)
	}

	return exists, nil
}

// GetHighestBlockBeforeTime retrieves the highest block before a target time
func (db *DB) GetHighestBlockBeforeTime(ctx context.Context, targetTime time.Time) (*indexermodels.Block, error) {
	query := fmt.Sprintf(`
		SELECT height, hash, time, network_id, parent_hash, proposer_address, size,
		       num_txs, total_txs, total_vdf_iterations, state_root, transaction_root,
		       validator_root, next_validator_root
		FROM %s
		WHERE time <= $1
		ORDER BY height DESC
		LIMIT 1
	`, db.SchemaTable("blocks"))

	var block indexermodels.Block
	err := db.Client.Pool.QueryRow(ctx, query, targetTime).Scan(
		&block.Height, &block.Hash, &block.Time, &block.NetworkID, &block.LastBlockHash, &block.ProposerAddress,
		&block.Size, &block.NumTxs, &block.TotalTxs, &block.TotalVDFIterations,
		&block.StateRoot, &block.TransactionRoot, &block.ValidatorRoot, &block.NextValidatorRoot,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no block found before time %v", targetTime)
		}
		return nil, fmt.Errorf("failed to get highest block before time: %w", err)
	}

	return &block, nil
}

// GetEventsByTypeAndHeight retrieves events by type and height (staging parameter ignored)
func (db *DB) GetEventsByTypeAndHeight(ctx context.Context, height uint64, staging bool, eventTypes ...string) ([]*indexermodels.Event, error) {
	if len(eventTypes) == 0 {
		return nil, fmt.Errorf("at least one event type must be specified")
	}

	query := fmt.Sprintf(`
		SELECT height, chain_id, address, reference, event_type, block_height,
		       amount, sold_amount, bought_amount, local_amount, remote_amount,
		       success, local_origin, order_id, points_received, points_burned,
		       data, seller_receive_address, buyer_send_address, sellers_send_address,
		       msg, height_time
		FROM %s
		WHERE height = $1 AND event_type = ANY($2)
		ORDER BY event_type, address
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height, eventTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*indexermodels.Event
	for rows.Next() {
		var event indexermodels.Event
		err := rows.Scan(
			&event.Height, &event.ChainID, &event.Address, &event.Reference, &event.EventType, &event.BlockHeight,
			&event.Amount, &event.SoldAmount, &event.BoughtAmount, &event.LocalAmount, &event.RemoteAmount,
			&event.Success, &event.LocalOrigin, &event.OrderID, &event.PointsReceived, &event.PointsBurned,
			&event.Data, &event.SellerReceiveAddress, &event.BuyerSendAddress, &event.SellersSendAddress,
			&event.Msg, &event.HeightTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	return events, rows.Err()
}

// GetRewardSlashEvents retrieves reward and slash events (optimized query)
func (db *DB) GetRewardSlashEvents(ctx context.Context, height uint64, staging bool) ([]EventRewardSlash, error) {
	query := fmt.Sprintf(`
		SELECT address, amount, event_type
		FROM %s
		WHERE height = $1 AND event_type IN ('EventReward', 'EventSlash')
		ORDER BY address
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height)
	if err != nil {
		return nil, fmt.Errorf("failed to query reward/slash events: %w", err)
	}
	defer rows.Close()

	var events []EventRewardSlash
	for rows.Next() {
		var event EventRewardSlash
		var amount uint64
		err := rows.Scan(&event.Address, &amount, &event.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		event.Amount = int64(amount)
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetValidatorLifecycleEvents retrieves validator lifecycle events
func (db *DB) GetValidatorLifecycleEvents(ctx context.Context, height uint64, staging bool) ([]EventLifecycle, error) {
	query := fmt.Sprintf(`
		SELECT address, event_type
		FROM %s
		WHERE height = $1 AND event_type IN (
			'EventAutoPause', 'EventAutoBeginUnstaking', 'EventAutoFinishUnstaking'
		)
		ORDER BY address
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height)
	if err != nil {
		return nil, fmt.Errorf("failed to query lifecycle events: %w", err)
	}
	defer rows.Close()

	var events []EventLifecycle
	for rows.Next() {
		var event EventLifecycle
		err := rows.Scan(&event.Address, &event.EventType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetOrderBookSwapEvents retrieves order book swap events
func (db *DB) GetOrderBookSwapEvents(ctx context.Context, height uint64, staging bool) ([]EventWithOrderID, error) {
	query := fmt.Sprintf(`
		SELECT order_id, event_type
		FROM %s
		WHERE height = $1 AND event_type = 'EventOrderBookSwap' AND order_id != ''
		ORDER BY order_id
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height)
	if err != nil {
		return nil, fmt.Errorf("failed to query order book swap events: %w", err)
	}
	defer rows.Close()

	var events []EventWithOrderID
	for rows.Next() {
		var event EventWithOrderID
		err := rows.Scan(&event.OrderID, &event.EventType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetDexOrderEvents retrieves DEX order events
func (db *DB) GetDexOrderEvents(ctx context.Context, height uint64, staging bool) ([]EventWithOrderID, error) {
	query := fmt.Sprintf(`
		SELECT order_id, event_type
		FROM %s
		WHERE height = $1 AND event_type = 'EventDexSwap' AND order_id != ''
		ORDER BY order_id
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height)
	if err != nil {
		return nil, fmt.Errorf("failed to query DEX order events: %w", err)
	}
	defer rows.Close()

	var events []EventWithOrderID
	for rows.Next() {
		var event EventWithOrderID
		err := rows.Scan(&event.OrderID, &event.EventType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetEventCountsByType counts events by type at a given height
func (db *DB) GetEventCountsByType(ctx context.Context, height uint64, staging bool, eventTypes ...string) (map[string]uint64, error) {
	if len(eventTypes) == 0 {
		return nil, fmt.Errorf("at least one event type must be specified")
	}

	query := fmt.Sprintf(`
		SELECT event_type, COUNT(*) as count
		FROM %s
		WHERE height = $1 AND event_type = ANY($2)
		GROUP BY event_type
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height, eventTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to query event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]uint64)
	for rows.Next() {
		var eventType string
		var count uint64
		err := rows.Scan(&eventType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event count: %w", err)
		}
		counts[eventType] = count
	}

	return counts, rows.Err()
}

// GetDexBatchEvents retrieves DEX batch events
func (db *DB) GetDexBatchEvents(ctx context.Context, height uint64, staging bool) ([]EventDexBatch, error) {
	query := fmt.Sprintf(`
		SELECT order_id, event_type, amount
		FROM %s
		WHERE height = $1 AND event_type IN (
			'EventDexLiquidityDeposit', 'EventDexLiquidityWithdraw', 'EventDexSwap'
		) AND order_id != ''
		ORDER BY order_id
	`, db.SchemaTable("events"))

	rows, err := db.Client.Pool.Query(ctx, query, height)
	if err != nil {
		return nil, fmt.Errorf("failed to query DEX batch events: %w", err)
	}
	defer rows.Close()

	var events []EventDexBatch
	for rows.Next() {
		var event EventDexBatch
		var amount uint64
		err := rows.Scan(&event.OrderID, &event.EventType, &amount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		event.Amount = int64(amount)
		events = append(events, event)
	}

	return events, rows.Err()
}

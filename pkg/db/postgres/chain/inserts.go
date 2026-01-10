package chain

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
)

// insertBlock inserts a block into the blocks table
// Accepts an Executor which can be either a transaction (pgx.Tx) or connection pool
func (db *DB) insertBlock(ctx context.Context, exec postgres.Executor, block *indexermodels.Block) error {
	query := `
		INSERT INTO blocks (
			height, hash, timestamp, network_id, parent_hash, proposer_address, size,
			num_txs, total_txs, total_vdf_iterations,
			state_root, transaction_root, validator_root, next_validator_root
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (height) DO UPDATE SET
			hash = EXCLUDED.hash,
			timestamp = EXCLUDED.timestamp,
			network_id = EXCLUDED.network_id,
			parent_hash = EXCLUDED.parent_hash,
			proposer_address = EXCLUDED.proposer_address,
			size = EXCLUDED.size,
			num_txs = EXCLUDED.num_txs,
			total_txs = EXCLUDED.total_txs,
			total_vdf_iterations = EXCLUDED.total_vdf_iterations,
			state_root = EXCLUDED.state_root,
			transaction_root = EXCLUDED.transaction_root,
			validator_root = EXCLUDED.validator_root,
			next_validator_root = EXCLUDED.next_validator_root
	`

	_, err := exec.Exec(ctx, query,
		block.Height, block.Hash, block.Time, block.NetworkID, block.LastBlockHash, block.ProposerAddress,
		block.Size, block.NumTxs, block.TotalTxs, block.TotalVDFIterations,
		block.StateRoot, block.TransactionRoot, block.ValidatorRoot, block.NextValidatorRoot,
	)
	return err
}

// insertTransactions inserts transactions into the txs table
func (db *DB) insertTransactions(ctx context.Context, exec postgres.Executor, txs []*indexermodels.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO txs (
			height, tx_hash, tx_index, timestamp, height_time, created_height, network_id,
			message_type, signer, amount, fee, memo,
			validator_address, commission,
			chain_id, sell_amount, buy_amount, liquidity_amount, liquidity_percent,
			order_id, price,
			param_key, param_value,
			committee_id, recipient, poll_hash,
			buyer_receive_address, buyer_send_address, buyer_chain_deadline,
			msg, public_key, signature
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32
		)
		ON CONFLICT (height, tx_hash) DO UPDATE SET
			tx_index = EXCLUDED.tx_index,
			timestamp = EXCLUDED.timestamp,
			height_time = EXCLUDED.height_time,
			created_height = EXCLUDED.created_height,
			network_id = EXCLUDED.network_id,
			message_type = EXCLUDED.message_type,
			signer = EXCLUDED.signer,
			amount = EXCLUDED.amount,
			fee = EXCLUDED.fee,
			memo = EXCLUDED.memo,
			validator_address = EXCLUDED.validator_address,
			commission = EXCLUDED.commission,
			chain_id = EXCLUDED.chain_id,
			sell_amount = EXCLUDED.sell_amount,
			buy_amount = EXCLUDED.buy_amount,
			liquidity_amount = EXCLUDED.liquidity_amount,
			liquidity_percent = EXCLUDED.liquidity_percent,
			order_id = EXCLUDED.order_id,
			price = EXCLUDED.price,
			param_key = EXCLUDED.param_key,
			param_value = EXCLUDED.param_value,
			committee_id = EXCLUDED.committee_id,
			recipient = EXCLUDED.recipient,
			poll_hash = EXCLUDED.poll_hash,
			buyer_receive_address = EXCLUDED.buyer_receive_address,
			buyer_send_address = EXCLUDED.buyer_send_address,
			buyer_chain_deadline = EXCLUDED.buyer_chain_deadline,
			msg = EXCLUDED.msg,
			public_key = EXCLUDED.public_key,
			signature = EXCLUDED.signature
	`

	for _, tx := range txs {
		batch.Queue(query,
			tx.Height, tx.TxHash, tx.TxIndex, tx.Timestamp, tx.HeightTime, tx.CreatedHeight, tx.NetworkID,
			tx.MessageType, tx.Signer, tx.Amount, tx.Fee, tx.Memo,
			tx.ValidatorAddress, tx.Commission,
			tx.ChainID, tx.SellAmount, tx.BuyAmount, tx.LiquidityAmt, tx.LiquidityPercent,
			tx.OrderID, tx.Price,
			tx.ParamKey, tx.ParamValue,
			tx.CommitteeID, tx.Recipient, tx.PollHash,
			tx.BuyerReceiveAddress, tx.BuyerSendAddress, tx.BuyerChainDeadline,
			tx.Msg, tx.PublicKey, tx.Signature,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertBlockSummary inserts a block summary into the block_summaries table
func (db *DB) insertBlockSummary(ctx context.Context, exec postgres.Executor, summary *indexermodels.BlockSummary) error {
	query := `
		INSERT INTO block_summaries (
			height, height_time, total_transactions,
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
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38,
			$39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56,
			$57, $58, $59, $60, $61, $62, $63, $64, $65, $66, $67, $68, $69, $70, $71, $72, $73, $74,
			$75, $76, $77, $78, $79, $80, $81
		)
		ON CONFLICT (height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			total_transactions = EXCLUDED.total_transactions,
			num_txs = EXCLUDED.num_txs,
			num_txs_send = EXCLUDED.num_txs_send,
			num_txs_stake = EXCLUDED.num_txs_stake,
			num_txs_unstake = EXCLUDED.num_txs_unstake,
			num_txs_edit_stake = EXCLUDED.num_txs_edit_stake,
			num_txs_start_poll = EXCLUDED.num_txs_start_poll,
			num_txs_vote_poll = EXCLUDED.num_txs_vote_poll,
			num_txs_lock_order = EXCLUDED.num_txs_lock_order,
			num_txs_close_order = EXCLUDED.num_txs_close_order,
			num_txs_unknown = EXCLUDED.num_txs_unknown,
			num_txs_pause = EXCLUDED.num_txs_pause,
			num_txs_unpause = EXCLUDED.num_txs_unpause,
			num_txs_change_parameter = EXCLUDED.num_txs_change_parameter,
			num_txs_dao_transfer = EXCLUDED.num_txs_dao_transfer,
			num_txs_certificate_results = EXCLUDED.num_txs_certificate_results,
			num_txs_subsidy = EXCLUDED.num_txs_subsidy,
			num_txs_create_order = EXCLUDED.num_txs_create_order,
			num_txs_edit_order = EXCLUDED.num_txs_edit_order,
			num_txs_delete_order = EXCLUDED.num_txs_delete_order,
			num_txs_dex_limit_order = EXCLUDED.num_txs_dex_limit_order,
			num_txs_dex_liquidity_deposit = EXCLUDED.num_txs_dex_liquidity_deposit,
			num_txs_dex_liquidity_withdraw = EXCLUDED.num_txs_dex_liquidity_withdraw,
			num_accounts = EXCLUDED.num_accounts,
			num_accounts_new = EXCLUDED.num_accounts_new,
			num_events = EXCLUDED.num_events,
			num_events_reward = EXCLUDED.num_events_reward,
			num_events_slash = EXCLUDED.num_events_slash,
			num_events_dex_liquidity_deposit = EXCLUDED.num_events_dex_liquidity_deposit,
			num_events_dex_liquidity_withdraw = EXCLUDED.num_events_dex_liquidity_withdraw,
			num_events_dex_swap = EXCLUDED.num_events_dex_swap,
			num_events_order_book_swap = EXCLUDED.num_events_order_book_swap,
			num_events_automatic_pause = EXCLUDED.num_events_automatic_pause,
			num_events_automatic_begin_unstaking = EXCLUDED.num_events_automatic_begin_unstaking,
			num_events_automatic_finish_unstaking = EXCLUDED.num_events_automatic_finish_unstaking,
			num_orders = EXCLUDED.num_orders,
			num_orders_new = EXCLUDED.num_orders_new,
			num_orders_open = EXCLUDED.num_orders_open,
			num_orders_filled = EXCLUDED.num_orders_filled,
			num_orders_cancelled = EXCLUDED.num_orders_cancelled,
			num_pools = EXCLUDED.num_pools,
			num_pools_new = EXCLUDED.num_pools_new,
			num_dex_prices = EXCLUDED.num_dex_prices,
			num_dex_orders = EXCLUDED.num_dex_orders,
			num_dex_orders_future = EXCLUDED.num_dex_orders_future,
			num_dex_orders_locked = EXCLUDED.num_dex_orders_locked,
			num_dex_orders_complete = EXCLUDED.num_dex_orders_complete,
			num_dex_orders_success = EXCLUDED.num_dex_orders_success,
			num_dex_orders_failed = EXCLUDED.num_dex_orders_failed,
			num_dex_deposits = EXCLUDED.num_dex_deposits,
			num_dex_deposits_pending = EXCLUDED.num_dex_deposits_pending,
			num_dex_deposits_locked = EXCLUDED.num_dex_deposits_locked,
			num_dex_deposits_complete = EXCLUDED.num_dex_deposits_complete,
			num_dex_withdrawals = EXCLUDED.num_dex_withdrawals,
			num_dex_withdrawals_pending = EXCLUDED.num_dex_withdrawals_pending,
			num_dex_withdrawals_locked = EXCLUDED.num_dex_withdrawals_locked,
			num_dex_withdrawals_complete = EXCLUDED.num_dex_withdrawals_complete,
			num_pool_points_holders = EXCLUDED.num_pool_points_holders,
			num_pool_points_holders_new = EXCLUDED.num_pool_points_holders_new,
			params_changed = EXCLUDED.params_changed,
			num_validators = EXCLUDED.num_validators,
			num_validators_new = EXCLUDED.num_validators_new,
			num_validators_active = EXCLUDED.num_validators_active,
			num_validators_paused = EXCLUDED.num_validators_paused,
			num_validators_unstaking = EXCLUDED.num_validators_unstaking,
			num_validator_non_signing_info = EXCLUDED.num_validator_non_signing_info,
			num_validator_non_signing_info_new = EXCLUDED.num_validator_non_signing_info_new,
			num_validator_double_signing_info = EXCLUDED.num_validator_double_signing_info,
			num_committees = EXCLUDED.num_committees,
			num_committees_new = EXCLUDED.num_committees_new,
			num_committees_subsidized = EXCLUDED.num_committees_subsidized,
			num_committees_retired = EXCLUDED.num_committees_retired,
			num_committee_validators = EXCLUDED.num_committee_validators,
			num_committee_payments = EXCLUDED.num_committee_payments,
			supply_changed = EXCLUDED.supply_changed,
			supply_total = EXCLUDED.supply_total,
			supply_staked = EXCLUDED.supply_staked,
			supply_delegated_only = EXCLUDED.supply_delegated_only
	`

	_, err := exec.Exec(ctx, query,
		summary.Height, summary.HeightTime, summary.TotalTransactions,
		summary.NumTxs, summary.NumTxsSend, summary.NumTxsStake, summary.NumTxsUnstake, summary.NumTxsEditStake,
		summary.NumTxsStartPoll, summary.NumTxsVotePoll, summary.NumTxsLockOrder, summary.NumTxsCloseOrder,
		summary.NumTxsUnknown, summary.NumTxsPause, summary.NumTxsUnpause, summary.NumTxsChangeParameter,
		summary.NumTxsDaoTransfer, summary.NumTxsCertificateResults, summary.NumTxsSubsidy,
		summary.NumTxsCreateOrder, summary.NumTxsEditOrder, summary.NumTxsDeleteOrder,
		summary.NumTxsDexLimitOrder, summary.NumTxsDexLiquidityDeposit, summary.NumTxsDexLiquidityWithdraw,
		summary.NumAccounts, summary.NumAccountsNew,
		summary.NumEvents, summary.NumEventsReward, summary.NumEventsSlash,
		summary.NumEventsDexLiquidityDeposit, summary.NumEventsDexLiquidityWithdraw, summary.NumEventsDexSwap,
		summary.NumEventsOrderBookSwap, summary.NumEventsAutomaticPause,
		summary.NumEventsAutomaticBeginUnstaking, summary.NumEventsAutomaticFinishUnstaking,
		summary.NumOrders, summary.NumOrdersNew, summary.NumOrdersOpen, summary.NumOrdersFilled, summary.NumOrdersCancelled,
		summary.NumPools, summary.NumPoolsNew,
		summary.NumDexPrices,
		summary.NumDexOrders, summary.NumDexOrdersFuture, summary.NumDexOrdersLocked, summary.NumDexOrdersComplete,
		summary.NumDexOrdersSuccess, summary.NumDexOrdersFailed,
		summary.NumDexDeposits, summary.NumDexDepositsPending, summary.NumDexDepositsLocked, summary.NumDexDepositsComplete,
		summary.NumDexWithdrawals, summary.NumDexWithdrawalsPending, summary.NumDexWithdrawalsLocked, summary.NumDexWithdrawalsComplete,
		summary.NumDexPoolPointsHolders, summary.NumDexPoolPointsHoldersNew,
		summary.ParamsChanged,
		summary.NumValidators, summary.NumValidatorsNew, summary.NumValidatorsActive, summary.NumValidatorsPaused, summary.NumValidatorsUnstaking,
		summary.NumValidatorNonSigningInfo, summary.NumValidatorNonSigningInfoNew,
		summary.NumValidatorDoubleSigningInfo,
		summary.NumCommittees, summary.NumCommitteesNew, summary.NumCommitteesSubsidized, summary.NumCommitteesRetired,
		summary.NumCommitteeValidators,
		summary.NumCommitteePayments,
		summary.SupplyChanged, summary.SupplyTotal, summary.SupplyStaked, summary.SupplyDelegatedOnly,
	)
	return err
}

// insertEvents inserts events into the events table
func (db *DB) insertEvents(ctx context.Context, exec postgres.Executor, events []*indexermodels.Event) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO events (
			height, chain_id, address, reference, event_type, block_height,
			amount, sold_amount, bought_amount, local_amount, remote_amount,
			success, local_origin, order_id, points_received, points_burned,
			data, seller_receive_address, buyer_send_address, sellers_send_address,
			msg, height_time
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
		ON CONFLICT (height, event_type, chain_id, address, reference) DO UPDATE SET
			block_height = EXCLUDED.block_height,
			amount = EXCLUDED.amount,
			sold_amount = EXCLUDED.sold_amount,
			bought_amount = EXCLUDED.bought_amount,
			local_amount = EXCLUDED.local_amount,
			remote_amount = EXCLUDED.remote_amount,
			success = EXCLUDED.success,
			local_origin = EXCLUDED.local_origin,
			order_id = EXCLUDED.order_id,
			points_received = EXCLUDED.points_received,
			points_burned = EXCLUDED.points_burned,
			data = EXCLUDED.data,
			seller_receive_address = EXCLUDED.seller_receive_address,
			buyer_send_address = EXCLUDED.buyer_send_address,
			sellers_send_address = EXCLUDED.sellers_send_address,
			msg = EXCLUDED.msg,
			height_time = EXCLUDED.height_time
	`

	for _, event := range events {
		batch.Queue(query,
			event.Height, event.ChainID, event.Address, event.Reference, event.EventType, event.BlockHeight,
			event.Amount, event.SoldAmount, event.BoughtAmount, event.LocalAmount, event.RemoteAmount,
			event.Success, event.LocalOrigin, event.OrderID, event.PointsReceived, event.PointsBurned,
			event.Data, event.SellerReceiveAddress, event.BuyerSendAddress, event.SellersSendAddress,
			event.Msg, event.HeightTime,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertAccounts inserts accounts into the accounts table
func (db *DB) insertAccounts(ctx context.Context, exec postgres.Executor, accounts []*indexermodels.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO accounts (address, height, amount, rewards, slashes, height_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (address, height) DO UPDATE SET
			amount = EXCLUDED.amount,
			rewards = EXCLUDED.rewards,
			slashes = EXCLUDED.slashes,
			height_time = EXCLUDED.height_time
	`

	for _, account := range accounts {
		batch.Queue(query, account.Address, account.Height, account.Amount, account.Rewards, account.Slashes, account.HeightTime)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertValidators inserts validators into the validators table
func (db *DB) insertValidators(ctx context.Context, exec postgres.Executor, validators []*indexermodels.Validator) error {
	if len(validators) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO validators (
			address, height, public_key, net_address, staked_amount,
			max_paused_height, unstaking_height, output, delegate, compound,
			status, height_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (address, height) DO UPDATE SET
			public_key = EXCLUDED.public_key,
			net_address = EXCLUDED.net_address,
			staked_amount = EXCLUDED.staked_amount,
			max_paused_height = EXCLUDED.max_paused_height,
			unstaking_height = EXCLUDED.unstaking_height,
			output = EXCLUDED.output,
			delegate = EXCLUDED.delegate,
			compound = EXCLUDED.compound,
			status = EXCLUDED.status,
			height_time = EXCLUDED.height_time
	`

	for _, validator := range validators {
		batch.Queue(query,
			validator.Address, validator.Height, validator.PublicKey, validator.NetAddress, validator.StakedAmount,
			validator.MaxPausedHeight, validator.UnstakingHeight, validator.Output, validator.Delegate, validator.Compound,
			validator.Status, validator.HeightTime,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertValidatorNonSigningInfo inserts non-signing info
func (db *DB) insertValidatorNonSigningInfo(ctx context.Context, exec postgres.Executor, infos []*indexermodels.ValidatorNonSigningInfo) error {
	if len(infos) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO validator_non_signing_info (address, height, missed_blocks_count, last_signed_height, height_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (address, height) DO UPDATE SET
			missed_blocks_count = EXCLUDED.missed_blocks_count,
			last_signed_height = EXCLUDED.last_signed_height,
			height_time = EXCLUDED.height_time
	`

	for _, info := range infos {
		batch.Queue(query, info.Address, info.Height, info.MissedBlocksCount, info.LastSignedHeight, info.HeightTime)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertValidatorDoubleSigningInfo inserts double signing info
func (db *DB) insertValidatorDoubleSigningInfo(ctx context.Context, exec postgres.Executor, infos []*indexermodels.ValidatorDoubleSigningInfo) error {
	if len(infos) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO validator_double_signing_info (
			address, height, evidence_count, first_evidence_height, last_evidence_height, height_time
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (address, height) DO UPDATE SET
			evidence_count = EXCLUDED.evidence_count,
			first_evidence_height = EXCLUDED.first_evidence_height,
			last_evidence_height = EXCLUDED.last_evidence_height,
			height_time = EXCLUDED.height_time
	`

	for _, info := range infos {
		batch.Queue(query, info.Address, info.Height, info.EvidenceCount, info.FirstEvidenceHeight, info.LastEvidenceHeight, info.HeightTime)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertPools inserts pools
func (db *DB) insertPools(ctx context.Context, exec postgres.Executor, pools []*indexermodels.Pool) error {
	if len(pools) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO pools (
			pool_id, height, chain_id, amount, total_points, lp_count, height_time,
			liquidity_pool_id, holding_pool_id, escrow_pool_id, reward_pool_id,
			amount_delta, total_points_delta, lp_count_delta
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (pool_id, height) DO UPDATE SET
			chain_id = EXCLUDED.chain_id,
			amount = EXCLUDED.amount,
			total_points = EXCLUDED.total_points,
			lp_count = EXCLUDED.lp_count,
			height_time = EXCLUDED.height_time,
			liquidity_pool_id = EXCLUDED.liquidity_pool_id,
			holding_pool_id = EXCLUDED.holding_pool_id,
			escrow_pool_id = EXCLUDED.escrow_pool_id,
			reward_pool_id = EXCLUDED.reward_pool_id,
			amount_delta = EXCLUDED.amount_delta,
			total_points_delta = EXCLUDED.total_points_delta,
			lp_count_delta = EXCLUDED.lp_count_delta
	`

	for _, pool := range pools {
		batch.Queue(query,
			pool.PoolID, pool.Height, pool.ChainID, pool.Amount, pool.TotalPoints, pool.LPCount, pool.HeightTime,
			pool.LiquidityPoolID, pool.HoldingPoolID, pool.EscrowPoolID, pool.RewardPoolID,
			pool.AmountDelta, pool.TotalPointsDelta, pool.LPCountDelta,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertPoolPointsByHolder inserts pool points by holder
func (db *DB) insertPoolPointsByHolder(ctx context.Context, exec postgres.Executor, holders []*indexermodels.PoolPointsByHolder) error {
	if len(holders) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO pool_points_by_holder (
			address, pool_id, height, height_time, committee, points,
			liquidity_pool_points, liquidity_pool_id, pool_amount
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (address, pool_id, height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			committee = EXCLUDED.committee,
			points = EXCLUDED.points,
			liquidity_pool_points = EXCLUDED.liquidity_pool_points,
			liquidity_pool_id = EXCLUDED.liquidity_pool_id,
			pool_amount = EXCLUDED.pool_amount
	`

	for _, holder := range holders {
		batch.Queue(query,
			holder.Address, holder.PoolID, holder.Height, holder.HeightTime, holder.Committee, holder.Points,
			holder.LiquidityPoolPoints, holder.LiquidityPoolID, holder.PoolAmount,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertOrders inserts orders
func (db *DB) insertOrders(ctx context.Context, exec postgres.Executor, orders []*indexermodels.Order) error {
	if len(orders) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO orders (
			order_id, height, height_time, committee, data, amount_for_sale, requested_amount,
			seller_receive_address, buyer_send_address, buyer_receive_address,
			buyer_chain_deadline, sellers_send_address, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (order_id, height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			committee = EXCLUDED.committee,
			data = EXCLUDED.data,
			amount_for_sale = EXCLUDED.amount_for_sale,
			requested_amount = EXCLUDED.requested_amount,
			seller_receive_address = EXCLUDED.seller_receive_address,
			buyer_send_address = EXCLUDED.buyer_send_address,
			buyer_receive_address = EXCLUDED.buyer_receive_address,
			buyer_chain_deadline = EXCLUDED.buyer_chain_deadline,
			sellers_send_address = EXCLUDED.sellers_send_address,
			status = EXCLUDED.status
	`

	for _, order := range orders {
		batch.Queue(query,
			order.OrderID, order.Height, order.HeightTime, order.Committee, order.Data, order.AmountForSale, order.RequestedAmount,
			order.SellerReceiveAddress, order.BuyerSendAddress, order.BuyerReceiveAddress,
			order.BuyerChainDeadline, order.SellersSendAddress, order.Status,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertDexOrders inserts DEX orders
func (db *DB) insertDexOrders(ctx context.Context, exec postgres.Executor, orders []*indexermodels.DexOrder) error {
	if len(orders) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO dex_orders (
			order_id, height, height_time, committee, address, amount_for_sale, requested_amount,
			state, success, sold_amount, bought_amount, local_origin, locked_height
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (order_id, height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			committee = EXCLUDED.committee,
			address = EXCLUDED.address,
			amount_for_sale = EXCLUDED.amount_for_sale,
			requested_amount = EXCLUDED.requested_amount,
			state = EXCLUDED.state,
			success = EXCLUDED.success,
			sold_amount = EXCLUDED.sold_amount,
			bought_amount = EXCLUDED.bought_amount,
			local_origin = EXCLUDED.local_origin,
			locked_height = EXCLUDED.locked_height
	`

	for _, order := range orders {
		batch.Queue(query,
			order.OrderID, order.Height, order.HeightTime, order.Committee, order.Address, order.AmountForSale, order.RequestedAmount,
			order.State, order.Success, order.SoldAmount, order.BoughtAmount, order.LocalOrigin, order.LockedHeight,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertDexDeposits inserts DEX deposits
func (db *DB) insertDexDeposits(ctx context.Context, exec postgres.Executor, deposits []*indexermodels.DexDeposit) error {
	if len(deposits) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO dex_deposits (
			order_id, height, height_time, committee, address, amount, state, local_origin, points_received
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (order_id, height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			committee = EXCLUDED.committee,
			address = EXCLUDED.address,
			amount = EXCLUDED.amount,
			state = EXCLUDED.state,
			local_origin = EXCLUDED.local_origin,
			points_received = EXCLUDED.points_received
	`

	for _, deposit := range deposits {
		batch.Queue(query,
			deposit.OrderID, deposit.Height, deposit.HeightTime, deposit.Committee, deposit.Address,
			deposit.Amount, deposit.State, deposit.LocalOrigin, deposit.PointsReceived,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertDexWithdrawals inserts DEX withdrawals
func (db *DB) insertDexWithdrawals(ctx context.Context, exec postgres.Executor, withdrawals []*indexermodels.DexWithdrawal) error {
	if len(withdrawals) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO dex_withdrawals (
			order_id, height, height_time, committee, address, percent, state,
			local_amount, remote_amount, points_burned
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (order_id, height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			committee = EXCLUDED.committee,
			address = EXCLUDED.address,
			percent = EXCLUDED.percent,
			state = EXCLUDED.state,
			local_amount = EXCLUDED.local_amount,
			remote_amount = EXCLUDED.remote_amount,
			points_burned = EXCLUDED.points_burned
	`

	for _, withdrawal := range withdrawals {
		batch.Queue(query,
			withdrawal.OrderID, withdrawal.Height, withdrawal.HeightTime, withdrawal.Committee, withdrawal.Address,
			withdrawal.Percent, withdrawal.State, withdrawal.LocalAmount, withdrawal.RemoteAmount, withdrawal.PointsBurned,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertDexPrices inserts DEX prices
func (db *DB) insertDexPrices(ctx context.Context, exec postgres.Executor, prices []*indexermodels.DexPrice) error {
	if len(prices) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO dex_prices (
			local_chain_id, remote_chain_id, height, local_pool, remote_pool, price_e6,
			height_time, price_delta, local_pool_delta, remote_pool_delta
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (local_chain_id, remote_chain_id, height) DO UPDATE SET
			local_pool = EXCLUDED.local_pool,
			remote_pool = EXCLUDED.remote_pool,
			price_e6 = EXCLUDED.price_e6,
			height_time = EXCLUDED.height_time,
			price_delta = EXCLUDED.price_delta,
			local_pool_delta = EXCLUDED.local_pool_delta,
			remote_pool_delta = EXCLUDED.remote_pool_delta
	`

	for _, price := range prices {
		batch.Queue(query,
			price.LocalChainID, price.RemoteChainID, price.Height, price.LocalPool, price.RemotePool, price.PriceE6,
			price.HeightTime, price.PriceDelta, price.LocalPoolDelta, price.RemotePoolDelta,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertCommittees inserts committees
func (db *DB) insertCommittees(ctx context.Context, exec postgres.Executor, committees []*indexermodels.Committee) error {
	if len(committees) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO committees (
			chain_id, last_root_height_updated, last_chain_height_updated, number_of_samples,
			subsidized, retired, height, height_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (chain_id, height) DO UPDATE SET
			last_root_height_updated = EXCLUDED.last_root_height_updated,
			last_chain_height_updated = EXCLUDED.last_chain_height_updated,
			number_of_samples = EXCLUDED.number_of_samples,
			subsidized = EXCLUDED.subsidized,
			retired = EXCLUDED.retired,
			height_time = EXCLUDED.height_time
	`

	for _, committee := range committees {
		batch.Queue(query,
			committee.ChainID, committee.LastRootHeightUpdated, committee.LastChainHeightUpdated, committee.NumberOfSamples,
			committee.Subsidized > 0, committee.Retired > 0, committee.Height, committee.HeightTime,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertCommitteeValidators inserts committee validators
func (db *DB) insertCommitteeValidators(ctx context.Context, exec postgres.Executor, cvs []*indexermodels.CommitteeValidator) error {
	if len(cvs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO committee_validators (
			committee_id, validator_address, staked_amount, status, delegate, compound,
			height, height_time, subsidized, retired
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (committee_id, validator_address, height) DO UPDATE SET
			staked_amount = EXCLUDED.staked_amount,
			status = EXCLUDED.status,
			delegate = EXCLUDED.delegate,
			compound = EXCLUDED.compound,
			height_time = EXCLUDED.height_time,
			subsidized = EXCLUDED.subsidized,
			retired = EXCLUDED.retired
	`

	for _, cv := range cvs {
		batch.Queue(query,
			cv.CommitteeID, cv.ValidatorAddress, cv.StakedAmount, cv.Status, cv.Delegate, cv.Compound,
			cv.Height, cv.HeightTime, cv.Subsidized, cv.Retired,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertCommitteePayments inserts committee payments
func (db *DB) insertCommitteePayments(ctx context.Context, exec postgres.Executor, payments []*indexermodels.CommitteePayment) error {
	if len(payments) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO committee_payments (committee_id, address, percent, height, height_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (committee_id, address, height) DO UPDATE SET
			percent = EXCLUDED.percent,
			height_time = EXCLUDED.height_time
	`

	for _, payment := range payments {
		batch.Queue(query, payment.CommitteeID, payment.Address, payment.Percent, payment.Height, payment.HeightTime)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertParams inserts params
func (db *DB) insertParams(ctx context.Context, exec postgres.Executor, params *indexermodels.Params) error {
	query := `
		INSERT INTO params (
			height, height_time, block_size, protocol_version, root_chain_id, retired,
			unstaking_blocks, max_pause_blocks, double_sign_slash_percentage, non_sign_slash_percentage,
			max_non_sign, non_sign_window, max_committees, max_committee_size, early_withdrawal_penalty,
			delegate_unstaking_blocks, minimum_order_size, stake_percent_for_subsidized, max_slash_per_committee,
			delegate_reward_percentage, buy_deadline_blocks, lock_order_fee_multiplier,
			minimum_stake_for_validators, minimum_stake_for_delegates, maximum_delegates_per_committee,
			send_fee, stake_fee, edit_stake_fee, unstake_fee, pause_fee, unpause_fee,
			change_parameter_fee, dao_transfer_fee, certificate_results_fee, subsidy_fee,
			create_order_fee, edit_order_fee, delete_order_fee, dex_limit_order_fee,
			dex_liquidity_deposit_fee, dex_liquidity_withdraw_fee, dao_reward_percentage
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42
		)
		ON CONFLICT (height) DO UPDATE SET
			height_time = EXCLUDED.height_time,
			block_size = EXCLUDED.block_size,
			protocol_version = EXCLUDED.protocol_version,
			root_chain_id = EXCLUDED.root_chain_id,
			retired = EXCLUDED.retired,
			unstaking_blocks = EXCLUDED.unstaking_blocks,
			max_pause_blocks = EXCLUDED.max_pause_blocks,
			double_sign_slash_percentage = EXCLUDED.double_sign_slash_percentage,
			non_sign_slash_percentage = EXCLUDED.non_sign_slash_percentage,
			max_non_sign = EXCLUDED.max_non_sign,
			non_sign_window = EXCLUDED.non_sign_window,
			max_committees = EXCLUDED.max_committees,
			max_committee_size = EXCLUDED.max_committee_size,
			early_withdrawal_penalty = EXCLUDED.early_withdrawal_penalty,
			delegate_unstaking_blocks = EXCLUDED.delegate_unstaking_blocks,
			minimum_order_size = EXCLUDED.minimum_order_size,
			stake_percent_for_subsidized = EXCLUDED.stake_percent_for_subsidized,
			max_slash_per_committee = EXCLUDED.max_slash_per_committee,
			delegate_reward_percentage = EXCLUDED.delegate_reward_percentage,
			buy_deadline_blocks = EXCLUDED.buy_deadline_blocks,
			lock_order_fee_multiplier = EXCLUDED.lock_order_fee_multiplier,
			minimum_stake_for_validators = EXCLUDED.minimum_stake_for_validators,
			minimum_stake_for_delegates = EXCLUDED.minimum_stake_for_delegates,
			maximum_delegates_per_committee = EXCLUDED.maximum_delegates_per_committee,
			send_fee = EXCLUDED.send_fee,
			stake_fee = EXCLUDED.stake_fee,
			edit_stake_fee = EXCLUDED.edit_stake_fee,
			unstake_fee = EXCLUDED.unstake_fee,
			pause_fee = EXCLUDED.pause_fee,
			unpause_fee = EXCLUDED.unpause_fee,
			change_parameter_fee = EXCLUDED.change_parameter_fee,
			dao_transfer_fee = EXCLUDED.dao_transfer_fee,
			certificate_results_fee = EXCLUDED.certificate_results_fee,
			subsidy_fee = EXCLUDED.subsidy_fee,
			create_order_fee = EXCLUDED.create_order_fee,
			edit_order_fee = EXCLUDED.edit_order_fee,
			delete_order_fee = EXCLUDED.delete_order_fee,
			dex_limit_order_fee = EXCLUDED.dex_limit_order_fee,
			dex_liquidity_deposit_fee = EXCLUDED.dex_liquidity_deposit_fee,
			dex_liquidity_withdraw_fee = EXCLUDED.dex_liquidity_withdraw_fee,
			dao_reward_percentage = EXCLUDED.dao_reward_percentage
	`

	_, err := exec.Exec(ctx, query,
		params.Height, params.HeightTime, params.BlockSize, params.ProtocolVersion, params.RootChainID, params.Retired,
		params.UnstakingBlocks, params.MaxPauseBlocks, params.DoubleSignSlashPercentage, params.NonSignSlashPercentage,
		params.MaxNonSign, params.NonSignWindow, params.MaxCommittees, params.MaxCommitteeSize, params.EarlyWithdrawalPenalty,
		params.DelegateUnstakingBlocks, params.MinimumOrderSize, params.StakePercentForSubsidized, params.MaxSlashPerCommittee,
		params.DelegateRewardPercentage, params.BuyDeadlineBlocks, params.LockOrderFeeMultiplier,
		params.MinimumStakeForValidators, params.MinimumStakeForDelegates, params.MaximumDelegatesPerCommittee,
		params.SendFee, params.StakeFee, params.EditStakeFee, params.UnstakeFee, params.PauseFee, params.UnpauseFee,
		params.ChangeParameterFee, params.DaoTransferFee, params.CertificateResultsFee, params.SubsidyFee,
		params.CreateOrderFee, params.EditOrderFee, params.DeleteOrderFee, params.DexLimitOrderFee,
		params.DexLiquidityDepositFee, params.DexLiquidityWithdrawFee, params.DaoRewardPercentage,
	)
	return err
}

// insertSupply inserts supply
func (db *DB) insertSupply(ctx context.Context, exec postgres.Executor, supply *indexermodels.Supply) error {
	query := `
		INSERT INTO supply (total, staked, delegated_only, height, height_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (height) DO UPDATE SET
			total = EXCLUDED.total,
			staked = EXCLUDED.staked,
			delegated_only = EXCLUDED.delegated_only,
			height_time = EXCLUDED.height_time
	`

	_, err := exec.Exec(ctx, query, supply.Total, supply.Staked, supply.DelegatedOnly, supply.Height, supply.HeightTime)
	return err
}

// insertPollSnapshots inserts poll snapshots
func (db *DB) insertPollSnapshots(ctx context.Context, exec postgres.Executor, snapshots []*indexermodels.PollSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO poll_snapshots (
			proposal_hash, proposal_url, accounts_approve_tokens, accounts_reject_tokens,
			accounts_total_voted_tokens, accounts_total_tokens, accounts_approve_percentage,
			accounts_reject_percentage, accounts_voted_percentage, validators_approve_tokens,
			validators_reject_tokens, validators_total_voted_tokens, validators_total_tokens,
			validators_approve_percentage, validators_reject_percentage, validators_voted_percentage,
			snapshot_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (proposal_hash, snapshot_time) DO UPDATE SET
			proposal_url = EXCLUDED.proposal_url,
			accounts_approve_tokens = EXCLUDED.accounts_approve_tokens,
			accounts_reject_tokens = EXCLUDED.accounts_reject_tokens,
			accounts_total_voted_tokens = EXCLUDED.accounts_total_voted_tokens,
			accounts_total_tokens = EXCLUDED.accounts_total_tokens,
			accounts_approve_percentage = EXCLUDED.accounts_approve_percentage,
			accounts_reject_percentage = EXCLUDED.accounts_reject_percentage,
			accounts_voted_percentage = EXCLUDED.accounts_voted_percentage,
			validators_approve_tokens = EXCLUDED.validators_approve_tokens,
			validators_reject_tokens = EXCLUDED.validators_reject_tokens,
			validators_total_voted_tokens = EXCLUDED.validators_total_voted_tokens,
			validators_total_tokens = EXCLUDED.validators_total_tokens,
			validators_approve_percentage = EXCLUDED.validators_approve_percentage,
			validators_reject_percentage = EXCLUDED.validators_reject_percentage,
			validators_voted_percentage = EXCLUDED.validators_voted_percentage
	`

	for _, snapshot := range snapshots {
		batch.Queue(query,
			snapshot.ProposalHash, snapshot.ProposalURL, snapshot.AccountsApproveTokens, snapshot.AccountsRejectTokens,
			snapshot.AccountsTotalVotedTokens, snapshot.AccountsTotalTokens, snapshot.AccountsApprovePercentage,
			snapshot.AccountsRejectPercentage, snapshot.AccountsVotedPercentage, snapshot.ValidatorsApproveTokens,
			snapshot.ValidatorsRejectTokens, snapshot.ValidatorsTotalVotedTokens, snapshot.ValidatorsTotalTokens,
			snapshot.ValidatorsApprovePercentage, snapshot.ValidatorsRejectPercentage, snapshot.ValidatorsVotedPercentage,
			snapshot.SnapshotTime,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertProposalSnapshots inserts proposal snapshots
func (db *DB) insertProposalSnapshots(ctx context.Context, exec postgres.Executor, snapshots []*indexermodels.ProposalSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO proposal_snapshots (
			proposal_hash, proposal, approve, snapshot_time, proposal_type, signer,
			start_height, end_height, parameter_space, parameter_key, parameter_value,
			dao_transfer_address, dao_transfer_amount
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (proposal_hash, snapshot_time) DO UPDATE SET
			proposal = EXCLUDED.proposal,
			approve = EXCLUDED.approve,
			proposal_type = EXCLUDED.proposal_type,
			signer = EXCLUDED.signer,
			start_height = EXCLUDED.start_height,
			end_height = EXCLUDED.end_height,
			parameter_space = EXCLUDED.parameter_space,
			parameter_key = EXCLUDED.parameter_key,
			parameter_value = EXCLUDED.parameter_value,
			dao_transfer_address = EXCLUDED.dao_transfer_address,
			dao_transfer_amount = EXCLUDED.dao_transfer_amount
	`

	for _, snapshot := range snapshots {
		batch.Queue(query,
			snapshot.ProposalHash, snapshot.Proposal, snapshot.Approve, snapshot.SnapshotTime, snapshot.ProposalType, snapshot.Signer,
			snapshot.StartHeight, snapshot.EndHeight, snapshot.ParameterSpace, snapshot.ParameterKey, snapshot.ParameterValue,
			snapshot.DaoTransferAddress, snapshot.DaoTransferAmount,
		)
	}

	return db.executeBatch(ctx, exec, batch)
}

// insertSupplies inserts multiple supply records (batch)
func (db *DB) insertSupplies(ctx context.Context, exec postgres.Executor, supplies []*indexermodels.Supply) error {
	if len(supplies) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO supply (total, staked, delegated_only, height, height_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (height) DO UPDATE SET
			total = EXCLUDED.total,
			staked = EXCLUDED.staked,
			delegated_only = EXCLUDED.delegated_only,
			height_time = EXCLUDED.height_time
	`

	for _, supply := range supplies {
		batch.Queue(query, supply.Total, supply.Staked, supply.DelegatedOnly, supply.Height, supply.HeightTime)
	}

	return db.executeBatch(ctx, exec, batch)
}

// Helper function to execute batch and handle results
func (db *DB) executeBatch(ctx context.Context, exec postgres.Executor, batch *pgx.Batch) error {
	br := exec.SendBatch(ctx, batch)
	defer br.Close()

	// Execute all statements in batch and check for errors
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch statement %d failed: %w", i, err)
		}
	}

	return nil
}

// Helper function to format error messages
func fmtInsertError(entity string, err error) error {
	if err != nil {
		return fmt.Errorf("failed to insert %s: %w", entity, err)
	}
	return nil
}

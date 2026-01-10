package crosschain

import (
	"context"
)

// initIndexProgress creates the index_progress table
func (db *DB) initIndexProgress(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS index_progress (
			chain_id        BIGINT PRIMARY KEY,
			last_height     BIGINT NOT NULL DEFAULT 0,
			last_indexed_at TIMESTAMP WITH TIME ZONE,
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`

	return db.Exec(ctx, query)
}

// initBlocks creates the blocks table
func (db *DB) initBlocks(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS blocks (
			chain_id            BIGINT NOT NULL,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			block_hash          TEXT NOT NULL,
			proposer_address    TEXT,
			total_txs           BIGINT NOT NULL DEFAULT 0,
			num_txs             INTEGER NOT NULL DEFAULT 0,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_blocks_time ON blocks(height_time);
	`

	return db.Exec(ctx, query)
}

// initBlockSummaries creates the block_summaries table
func (db *DB) initBlockSummaries(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS block_summaries (
			chain_id                                BIGINT NOT NULL,
			height                                  BIGINT NOT NULL,
			height_time                             TIMESTAMP WITH TIME ZONE NOT NULL,
			total_transactions                      BIGINT NOT NULL DEFAULT 0,

			-- Transaction counts by type
			num_txs                                 INTEGER NOT NULL DEFAULT 0,
			num_txs_send                            INTEGER NOT NULL DEFAULT 0,
			num_txs_stake                           INTEGER NOT NULL DEFAULT 0,
			num_txs_unstake                         INTEGER NOT NULL DEFAULT 0,
			num_txs_edit_stake                      INTEGER NOT NULL DEFAULT 0,
			num_txs_start_poll                      INTEGER NOT NULL DEFAULT 0,
			num_txs_vote_poll                       INTEGER NOT NULL DEFAULT 0,
			num_txs_lock_order                      INTEGER NOT NULL DEFAULT 0,
			num_txs_close_order                     INTEGER NOT NULL DEFAULT 0,
			num_txs_unknown                         INTEGER NOT NULL DEFAULT 0,
			num_txs_pause                           INTEGER NOT NULL DEFAULT 0,
			num_txs_unpause                         INTEGER NOT NULL DEFAULT 0,
			num_txs_change_parameter                INTEGER NOT NULL DEFAULT 0,
			num_txs_dao_transfer                    INTEGER NOT NULL DEFAULT 0,
			num_txs_certificate_results             INTEGER NOT NULL DEFAULT 0,
			num_txs_subsidy                         INTEGER NOT NULL DEFAULT 0,
			num_txs_create_order                    INTEGER NOT NULL DEFAULT 0,
			num_txs_edit_order                      INTEGER NOT NULL DEFAULT 0,
			num_txs_delete_order                    INTEGER NOT NULL DEFAULT 0,
			num_txs_dex_limit_order                 INTEGER NOT NULL DEFAULT 0,
			num_txs_dex_liquidity_deposit           INTEGER NOT NULL DEFAULT 0,
			num_txs_dex_liquidity_withdraw          INTEGER NOT NULL DEFAULT 0,

			-- Account counts
			num_accounts                            INTEGER NOT NULL DEFAULT 0,
			num_accounts_new                        INTEGER NOT NULL DEFAULT 0,

			-- Event counts by type
			num_events                              INTEGER NOT NULL DEFAULT 0,
			num_events_reward                       INTEGER NOT NULL DEFAULT 0,
			num_events_slash                        INTEGER NOT NULL DEFAULT 0,
			num_events_dex_liquidity_deposit        INTEGER NOT NULL DEFAULT 0,
			num_events_dex_liquidity_withdraw       INTEGER NOT NULL DEFAULT 0,
			num_events_dex_swap                     INTEGER NOT NULL DEFAULT 0,
			num_events_order_book_swap              INTEGER NOT NULL DEFAULT 0,
			num_events_automatic_pause              INTEGER NOT NULL DEFAULT 0,
			num_events_automatic_begin_unstaking    INTEGER NOT NULL DEFAULT 0,
			num_events_automatic_finish_unstaking   INTEGER NOT NULL DEFAULT 0,

			-- Order counts
			num_orders                              INTEGER NOT NULL DEFAULT 0,
			num_orders_new                          INTEGER NOT NULL DEFAULT 0,
			num_orders_open                         INTEGER NOT NULL DEFAULT 0,
			num_orders_filled                       INTEGER NOT NULL DEFAULT 0,
			num_orders_cancelled                    INTEGER NOT NULL DEFAULT 0,

			-- Pool counts
			num_pools                               INTEGER NOT NULL DEFAULT 0,
			num_pools_new                           INTEGER NOT NULL DEFAULT 0,

			-- DEX price counts
			num_dex_prices                          INTEGER NOT NULL DEFAULT 0,

			-- DEX order counts
			num_dex_orders                          INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_future                   INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_locked                   INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_complete                 INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_success                  INTEGER NOT NULL DEFAULT 0,
			num_dex_orders_failed                   INTEGER NOT NULL DEFAULT 0,

			-- DEX deposit counts
			num_dex_deposits                        INTEGER NOT NULL DEFAULT 0,
			num_dex_deposits_pending                INTEGER NOT NULL DEFAULT 0,
			num_dex_deposits_locked                 INTEGER NOT NULL DEFAULT 0,
			num_dex_deposits_complete               INTEGER NOT NULL DEFAULT 0,

			-- DEX withdrawal counts
			num_dex_withdrawals                     INTEGER NOT NULL DEFAULT 0,
			num_dex_withdrawals_pending             INTEGER NOT NULL DEFAULT 0,
			num_dex_withdrawals_locked              INTEGER NOT NULL DEFAULT 0,
			num_dex_withdrawals_complete            INTEGER NOT NULL DEFAULT 0,

			-- Pool points holders
			num_pool_points_holders                 INTEGER NOT NULL DEFAULT 0,
			num_pool_points_holders_new             INTEGER NOT NULL DEFAULT 0,

			-- Params
			params_changed                          BOOLEAN NOT NULL DEFAULT FALSE,

			-- Validator counts
			num_validators                          INTEGER NOT NULL DEFAULT 0,
			num_validators_new                      INTEGER NOT NULL DEFAULT 0,
			num_validators_active                   INTEGER NOT NULL DEFAULT 0,
			num_validators_paused                   INTEGER NOT NULL DEFAULT 0,
			num_validators_unstaking                INTEGER NOT NULL DEFAULT 0,
			num_validator_non_signing_info          INTEGER NOT NULL DEFAULT 0,
			num_validator_non_signing_info_new      INTEGER NOT NULL DEFAULT 0,
			num_validator_double_signing_info       INTEGER NOT NULL DEFAULT 0,

			-- Committee counts
			num_committees                          INTEGER NOT NULL DEFAULT 0,
			num_committees_new                      INTEGER NOT NULL DEFAULT 0,
			num_committees_subsidized               INTEGER NOT NULL DEFAULT 0,
			num_committees_retired                  INTEGER NOT NULL DEFAULT 0,
			num_committee_validators                INTEGER NOT NULL DEFAULT 0,
			num_committee_payments                  INTEGER NOT NULL DEFAULT 0,

			-- Supply metrics
			supply_changed                          BOOLEAN NOT NULL DEFAULT FALSE,
			supply_total                            BIGINT NOT NULL DEFAULT 0,
			supply_staked                           BIGINT NOT NULL DEFAULT 0,
			supply_delegated_only                   BIGINT NOT NULL DEFAULT 0,

			created_at                              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height)
		)
	`

	return db.Exec(ctx, query)
}

// initTransactions creates the txs table
func (db *DB) initTransactions(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS txs (
			chain_id                BIGINT NOT NULL,
			height                  BIGINT NOT NULL,
			height_time             TIMESTAMP WITH TIME ZONE NOT NULL,
			tx_hash                 TEXT NOT NULL,
			tx_index                INTEGER NOT NULL,
			tx_time                 TIMESTAMP WITH TIME ZONE NOT NULL,
			created_height          BIGINT NOT NULL,
			network_id              BIGINT NOT NULL,
			message_type            TEXT NOT NULL,
			signer                  TEXT NOT NULL,
			amount                  BIGINT,
			fee                     BIGINT NOT NULL DEFAULT 0,
			memo                    TEXT,
			validator_address       TEXT,
			commission              DOUBLE PRECISION,
			tx_chain_id             BIGINT,
			sell_amount             BIGINT,
			buy_amount              BIGINT,
			liquidity_amount        BIGINT,
			liquidity_percent       BIGINT,
			order_id                TEXT,
			price                   DOUBLE PRECISION,
			param_key               TEXT,
			param_value             TEXT,
			committee_id            BIGINT,
			recipient               TEXT,
			poll_hash               TEXT,
			buyer_receive_address   TEXT,
			buyer_send_address      TEXT,
			buyer_chain_deadline    BIGINT,
			msg                     JSONB NOT NULL,
			public_key              TEXT,
			signature               TEXT,
			created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height, tx_hash)
		);

		CREATE INDEX IF NOT EXISTS idx_txs_signer ON txs(chain_id, signer);
		CREATE INDEX IF NOT EXISTS idx_txs_message_type ON txs(chain_id, message_type);
		CREATE INDEX IF NOT EXISTS idx_txs_time ON txs(height_time);
	`

	return db.Exec(ctx, query)
}

// initEvents creates the events table
func (db *DB) initEvents(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS events (
			chain_id                BIGINT NOT NULL,
			height                  BIGINT NOT NULL,
			height_time             TIMESTAMP WITH TIME ZONE NOT NULL,
			event_chain_id          BIGINT NOT NULL,
			address                 TEXT NOT NULL,
			reference               TEXT NOT NULL,
			event_type              TEXT NOT NULL,
			block_height            BIGINT NOT NULL,
			amount                  BIGINT,
			sold_amount             BIGINT,
			bought_amount           BIGINT,
			local_amount            BIGINT,
			remote_amount           BIGINT,
			success                 BOOLEAN,
			local_origin            BOOLEAN,
			order_id                TEXT,
			points_received         BIGINT,
			points_burned           BIGINT,
			data                    TEXT,
			seller_receive_address  TEXT,
			buyer_send_address      TEXT,
			sellers_send_address    TEXT,
			msg                     JSONB NOT NULL,
			created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_events_pk ON events(chain_id, height, event_type, address);
		CREATE INDEX IF NOT EXISTS idx_events_address ON events(chain_id, address);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(chain_id, event_type);
		CREATE INDEX IF NOT EXISTS idx_events_time ON events(height_time);
	`

	return db.Exec(ctx, query)
}

// initAccounts creates the accounts table
func (db *DB) initAccounts(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS accounts (
			chain_id        BIGINT NOT NULL,
			address         TEXT NOT NULL,
			amount          BIGINT NOT NULL DEFAULT 0,
			rewards         BIGINT NOT NULL DEFAULT 0,
			slashes         BIGINT NOT NULL DEFAULT 0,
			height          BIGINT NOT NULL,
			height_time     TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_accounts_address ON accounts(chain_id, address);
	`

	return db.Exec(ctx, query)
}

// initValidators creates the validators table
func (db *DB) initValidators(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS validators (
			chain_id            BIGINT NOT NULL,
			address             TEXT NOT NULL,
			public_key          TEXT NOT NULL,
			net_address         TEXT NOT NULL,
			staked_amount       BIGINT NOT NULL DEFAULT 0,
			max_paused_height   BIGINT NOT NULL DEFAULT 0,
			unstaking_height    BIGINT NOT NULL DEFAULT 0,
			output              TEXT NOT NULL,
			delegate            BOOLEAN NOT NULL DEFAULT FALSE,
			compound            BOOLEAN NOT NULL DEFAULT FALSE,
			status              validator_status NOT NULL DEFAULT 'active',
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_validators_address ON validators(chain_id, address);
		CREATE INDEX IF NOT EXISTS idx_validators_status ON validators(chain_id, status);
	`

	return db.Exec(ctx, query)
}

// initValidatorNonSigningInfo creates the validator_non_signing_info table
func (db *DB) initValidatorNonSigningInfo(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS validator_non_signing_info (
			chain_id            BIGINT NOT NULL,
			address             TEXT NOT NULL,
			missed_blocks_count BIGINT NOT NULL DEFAULT 0,
			last_signed_height  BIGINT NOT NULL DEFAULT 0,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, address, height)
		)
	`

	return db.Exec(ctx, query)
}

// initValidatorDoubleSigningInfo creates the validator_double_signing_info table
func (db *DB) initValidatorDoubleSigningInfo(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS validator_double_signing_info (
			chain_id                BIGINT NOT NULL,
			address                 TEXT NOT NULL,
			evidence_count          BIGINT NOT NULL DEFAULT 0,
			first_evidence_height   BIGINT NOT NULL DEFAULT 0,
			last_evidence_height    BIGINT NOT NULL DEFAULT 0,
			height                  BIGINT NOT NULL,
			height_time             TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, address, height)
		)
	`

	return db.Exec(ctx, query)
}

// initCommittees creates the committees table
func (db *DB) initCommittees(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS committees (
			chain_id                    BIGINT NOT NULL,
			committee_chain_id          BIGINT NOT NULL,
			last_root_height_updated    BIGINT NOT NULL DEFAULT 0,
			last_chain_height_updated   BIGINT NOT NULL DEFAULT 0,
			number_of_samples           BIGINT NOT NULL DEFAULT 0,
			subsidized                  BOOLEAN NOT NULL DEFAULT FALSE,
			retired                     BOOLEAN NOT NULL DEFAULT FALSE,
			height                      BIGINT NOT NULL,
			height_time                 TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, committee_chain_id, height)
		)
	`

	return db.Exec(ctx, query)
}

// initCommitteeValidators creates the committee_validators table
func (db *DB) initCommitteeValidators(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS committee_validators (
			chain_id        BIGINT NOT NULL,
			committee_id    BIGINT NOT NULL,
			address         TEXT NOT NULL,
			staked_amount   BIGINT NOT NULL DEFAULT 0,
			status          validator_status NOT NULL DEFAULT 'active',
			delegate        BOOLEAN NOT NULL DEFAULT FALSE,
			compound        BOOLEAN NOT NULL DEFAULT FALSE,
			height          BIGINT NOT NULL,
			height_time     TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, committee_id, address, height)
		)
	`

	return db.Exec(ctx, query)
}

// initCommitteePayments creates the committee_payments table
func (db *DB) initCommitteePayments(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS committee_payments (
			chain_id        BIGINT NOT NULL,
			committee_id    BIGINT NOT NULL,
			address         TEXT NOT NULL,
			percent         BIGINT NOT NULL DEFAULT 0,
			height          BIGINT NOT NULL,
			height_time     TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, committee_id, address, height)
		)
	`

	return db.Exec(ctx, query)
}

// initPools creates the pools table
func (db *DB) initPools(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pools (
			chain_id            BIGINT NOT NULL,
			pool_id             BIGINT NOT NULL,
			pool_chain_id       BIGINT NOT NULL,
			amount              BIGINT NOT NULL DEFAULT 0,
			total_points        BIGINT NOT NULL DEFAULT 0,
			lp_count            INTEGER NOT NULL DEFAULT 0,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			liquidity_pool_id   BIGINT NOT NULL DEFAULT 0,
			holding_pool_id     BIGINT NOT NULL DEFAULT 0,
			escrow_pool_id      BIGINT NOT NULL DEFAULT 0,
			reward_pool_id      BIGINT NOT NULL DEFAULT 0,
			amount_delta        BIGINT NOT NULL DEFAULT 0,
			total_points_delta  BIGINT NOT NULL DEFAULT 0,
			lp_count_delta      INTEGER NOT NULL DEFAULT 0,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, pool_id, height)
		)
	`

	return db.Exec(ctx, query)
}

// initPoolPointsByHolder creates the pool_points_by_holder table
func (db *DB) initPoolPointsByHolder(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pool_points_by_holder (
			chain_id                BIGINT NOT NULL,
			address                 TEXT NOT NULL,
			pool_id                 BIGINT NOT NULL,
			committee               BIGINT NOT NULL,
			points                  BIGINT NOT NULL DEFAULT 0,
			liquidity_pool_points   BIGINT NOT NULL DEFAULT 0,
			liquidity_pool_id       BIGINT NOT NULL DEFAULT 0,
			height                  BIGINT NOT NULL,
			height_time             TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, address, pool_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_pph_latest ON pool_points_by_holder(chain_id, address, pool_id, height DESC);
		CREATE INDEX IF NOT EXISTS idx_pph_first_seen ON pool_points_by_holder(chain_id, address, pool_id, height_time);
	`

	return db.Exec(ctx, query)
}

// initOrders creates the orders table
func (db *DB) initOrders(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS orders (
			chain_id                BIGINT NOT NULL,
			order_id                TEXT NOT NULL,
			committee               BIGINT NOT NULL,
			data                    TEXT,
			amount_for_sale         BIGINT NOT NULL DEFAULT 0,
			requested_amount        BIGINT NOT NULL DEFAULT 0,
			seller_receive_address  TEXT NOT NULL,
			buyer_send_address      TEXT,
			buyer_receive_address   TEXT,
			buyer_chain_deadline    BIGINT NOT NULL DEFAULT 0,
			sellers_send_address    TEXT,
			status                  order_status NOT NULL DEFAULT 'open',
			height                  BIGINT NOT NULL,
			height_time             TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, order_id, height)
		)
	`

	return db.Exec(ctx, query)
}

// initDexOrders creates the dex_orders table
func (db *DB) initDexOrders(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_orders (
			chain_id            BIGINT NOT NULL,
			order_id            TEXT NOT NULL,
			committee           BIGINT NOT NULL,
			address             TEXT NOT NULL,
			amount_for_sale     BIGINT NOT NULL DEFAULT 0,
			requested_amount    BIGINT NOT NULL DEFAULT 0,
			state               dex_order_state NOT NULL DEFAULT 'future',
			success             BOOLEAN NOT NULL DEFAULT FALSE,
			sold_amount         BIGINT NOT NULL DEFAULT 0,
			bought_amount       BIGINT NOT NULL DEFAULT 0,
			local_origin        BOOLEAN NOT NULL DEFAULT FALSE,
			locked_height       BIGINT NOT NULL DEFAULT 0,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_orders_address ON dex_orders(chain_id, address);
		CREATE INDEX IF NOT EXISTS idx_dex_orders_state ON dex_orders(chain_id, state);
	`

	return db.Exec(ctx, query)
}

// initDexDeposits creates the dex_deposits table
func (db *DB) initDexDeposits(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_deposits (
			chain_id            BIGINT NOT NULL,
			order_id            TEXT NOT NULL,
			committee           BIGINT NOT NULL,
			address             TEXT NOT NULL,
			amount              BIGINT NOT NULL DEFAULT 0,
			state               dex_deposit_state NOT NULL DEFAULT 'pending',
			local_origin        BOOLEAN NOT NULL DEFAULT FALSE,
			points_received     BIGINT NOT NULL DEFAULT 0,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_deposits_address ON dex_deposits(chain_id, address);
	`

	return db.Exec(ctx, query)
}

// initDexWithdrawals creates the dex_withdrawals table
func (db *DB) initDexWithdrawals(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_withdrawals (
			chain_id            BIGINT NOT NULL,
			order_id            TEXT NOT NULL,
			committee           BIGINT NOT NULL,
			address             TEXT NOT NULL,
			percent             BIGINT NOT NULL DEFAULT 0,
			state               dex_withdrawal_state NOT NULL DEFAULT 'pending',
			local_amount        BIGINT NOT NULL DEFAULT 0,
			remote_amount       BIGINT NOT NULL DEFAULT 0,
			points_burned       BIGINT NOT NULL DEFAULT 0,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_withdrawals_address ON dex_withdrawals(chain_id, address);
	`

	return db.Exec(ctx, query)
}

// initDexPrices creates the dex_prices table
func (db *DB) initDexPrices(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS dex_prices (
			chain_id            BIGINT NOT NULL,
			local_chain_id      BIGINT NOT NULL,
			remote_chain_id     BIGINT NOT NULL,
			height              BIGINT NOT NULL,
			height_time         TIMESTAMP WITH TIME ZONE NOT NULL,
			local_pool          BIGINT NOT NULL DEFAULT 0,
			remote_pool         BIGINT NOT NULL DEFAULT 0,
			price_e6            BIGINT NOT NULL DEFAULT 0,
			price_delta         BIGINT NOT NULL DEFAULT 0,
			local_pool_delta    BIGINT NOT NULL DEFAULT 0,
			remote_pool_delta   BIGINT NOT NULL DEFAULT 0,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, local_chain_id, remote_chain_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_dex_prices_time ON dex_prices(height_time);
	`

	return db.Exec(ctx, query)
}

// initParams creates the params table
func (db *DB) initParams(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS params (
			chain_id                        BIGINT NOT NULL,
			height                          BIGINT NOT NULL,
			height_time                     TIMESTAMP WITH TIME ZONE NOT NULL,
			block_size                      BIGINT NOT NULL DEFAULT 0,
			protocol_version                TEXT,
			root_chain_id                   BIGINT NOT NULL DEFAULT 0,
			retired                         BIGINT NOT NULL DEFAULT 0,
			unstaking_blocks                BIGINT NOT NULL DEFAULT 0,
			max_pause_blocks                BIGINT NOT NULL DEFAULT 0,
			double_sign_slash_percentage    BIGINT NOT NULL DEFAULT 0,
			non_sign_slash_percentage       BIGINT NOT NULL DEFAULT 0,
			max_non_sign                    BIGINT NOT NULL DEFAULT 0,
			non_sign_window                 BIGINT NOT NULL DEFAULT 0,
			max_committees                  BIGINT NOT NULL DEFAULT 0,
			max_committee_size              BIGINT NOT NULL DEFAULT 0,
			early_withdrawal_penalty        BIGINT NOT NULL DEFAULT 0,
			delegate_unstaking_blocks       BIGINT NOT NULL DEFAULT 0,
			minimum_order_size              BIGINT NOT NULL DEFAULT 0,
			stake_percent_for_subsidized    BIGINT NOT NULL DEFAULT 0,
			max_slash_per_committee         BIGINT NOT NULL DEFAULT 0,
			delegate_reward_percentage      BIGINT NOT NULL DEFAULT 0,
			buy_deadline_blocks             BIGINT NOT NULL DEFAULT 0,
			lock_order_fee_multiplier       BIGINT NOT NULL DEFAULT 0,
			minimum_stake_for_validators    BIGINT NOT NULL DEFAULT 0,
			minimum_stake_for_delegates     BIGINT NOT NULL DEFAULT 0,
			maximum_delegates_per_committee BIGINT NOT NULL DEFAULT 0,
			send_fee                        BIGINT NOT NULL DEFAULT 0,
			stake_fee                       BIGINT NOT NULL DEFAULT 0,
			edit_stake_fee                  BIGINT NOT NULL DEFAULT 0,
			unstake_fee                     BIGINT NOT NULL DEFAULT 0,
			pause_fee                       BIGINT NOT NULL DEFAULT 0,
			unpause_fee                     BIGINT NOT NULL DEFAULT 0,
			change_parameter_fee            BIGINT NOT NULL DEFAULT 0,
			dao_transfer_fee                BIGINT NOT NULL DEFAULT 0,
			certificate_results_fee         BIGINT NOT NULL DEFAULT 0,
			subsidy_fee                     BIGINT NOT NULL DEFAULT 0,
			create_order_fee                BIGINT NOT NULL DEFAULT 0,
			edit_order_fee                  BIGINT NOT NULL DEFAULT 0,
			delete_order_fee                BIGINT NOT NULL DEFAULT 0,
			dex_limit_order_fee             BIGINT NOT NULL DEFAULT 0,
			dex_liquidity_deposit_fee       BIGINT NOT NULL DEFAULT 0,
			dex_liquidity_withdraw_fee      BIGINT NOT NULL DEFAULT 0,
			dao_reward_percentage           BIGINT NOT NULL DEFAULT 0,
			created_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height)
		)
	`

	return db.Exec(ctx, query)
}

// initSupply creates the supply table
func (db *DB) initSupply(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS supply (
			chain_id        BIGINT NOT NULL,
			total           BIGINT NOT NULL DEFAULT 0,
			staked          BIGINT NOT NULL DEFAULT 0,
			delegated_only  BIGINT NOT NULL DEFAULT 0,
			height          BIGINT NOT NULL,
			height_time     TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height)
		)
	`

	return db.Exec(ctx, query)
}

// initTvlSnapshots creates the tvl_snapshots table
func (db *DB) initTvlSnapshots(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS tvl_snapshots (
			chain_id        BIGINT NOT NULL,
			snapshot_hour   TIMESTAMP WITH TIME ZONE NOT NULL,
			snapshot_height BIGINT NOT NULL,
			total_staked    BIGINT NOT NULL DEFAULT 0,
			total_pooled    BIGINT NOT NULL DEFAULT 0,
			total_tvl       BIGINT NOT NULL DEFAULT 0,
			computed_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, snapshot_hour)
		)
	`

	return db.Exec(ctx, query)
}

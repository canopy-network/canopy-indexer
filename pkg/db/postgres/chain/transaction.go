package chain

import (
	"context"
)

// initTransactions creates the txs table matching the ClickHouse Transaction model
// This matches pkg/db/models/indexer/tx.go:52-113 (33 fields)
func (db *DB) initTransactions(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS txs (
			-- Primary key fields (composite key)
			height BIGINT NOT NULL,
			tx_hash TEXT NOT NULL,
			tx_index SMALLINT NOT NULL,  -- UInt16 -> SMALLINT

			-- Time fields for queries
			time TIMESTAMP WITH TIME ZONE NOT NULL,       -- Transaction timestamp
			height_time TIMESTAMP WITH TIME ZONE NOT NULL, -- Block timestamp for range queries
			created_height BIGINT NOT NULL,                -- Height when tx was created
			network_id INTEGER NOT NULL,                   -- UInt32 -> INTEGER

			-- Common filterable fields
			message_type TEXT NOT NULL,  -- LowCardinality(String) in ClickHouse
			signer TEXT NOT NULL,        -- Transaction signer address
			amount BIGINT DEFAULT 0,     -- Amount transferred/staked/delegated (0 for votes, etc.)
			fee BIGINT NOT NULL,         -- Transaction fee
			memo TEXT DEFAULT '',        -- Transaction memo (poll/order operations)

			-- ===== EXTRACTED FIELDS (for efficient querying) =====

			-- Validator-related (stake, unstake, editStake)
			validator_address TEXT DEFAULT '',
			commission DOUBLE PRECISION DEFAULT 0,

			-- DEX-related (dexLimitOrder, dexLiquidityDeposit, dexLiquidityWithdraw)
			chain_id SMALLINT DEFAULT 0,       -- UInt16 -> SMALLINT (renamed from 'chain_id' to avoid conflict)
			sell_amount BIGINT DEFAULT 0,
			buy_amount BIGINT DEFAULT 0,
			liquidity_amount BIGINT DEFAULT 0,   -- For dexLiquidityDeposit
			liquidity_percent BIGINT DEFAULT 0,  -- For dexLiquidityWithdraw

			-- Order-related (createOrder, editOrder, deleteOrder)
			order_id TEXT DEFAULT '',
			price DOUBLE PRECISION DEFAULT 0,

			-- Governance-related (changeParameter, startPoll, votePoll)
			param_key TEXT DEFAULT '',
			param_value TEXT DEFAULT '',

			-- Other
			committee_id SMALLINT DEFAULT 0,  -- UInt16 -> SMALLINT (for: subsidy)
			recipient TEXT DEFAULT '',        -- For: daoTransfer
			poll_hash TEXT DEFAULT '',        -- For: startPoll, votePoll

			-- LockOrder-related (lockOrder memo transactions)
			buyer_receive_address TEXT DEFAULT '',  -- Canopy address to receive tokens
			buyer_send_address TEXT DEFAULT '',     -- External chain address sending payment
			buyer_chain_deadline BIGINT DEFAULT 0,  -- Deadline height on buyer's chain

			-- Full message as JSON (ALL type-specific fields)
			msg TEXT NOT NULL,  -- Complete message data (stored as JSON)

			-- Signature fields (permanent audit trail)
			public_key TEXT DEFAULT '',
			signature TEXT DEFAULT '',

			PRIMARY KEY (height, tx_hash)
		);

		-- Indexes for efficient querying
		CREATE INDEX IF NOT EXISTS idx_txs_signer ON txs(signer);
		CREATE INDEX IF NOT EXISTS idx_txs_recipient ON txs(recipient) WHERE recipient != '';
		CREATE INDEX IF NOT EXISTS idx_txs_message_type ON txs(message_type);
		CREATE INDEX IF NOT EXISTS idx_txs_time ON txs(height_time);
		CREATE INDEX IF NOT EXISTS idx_txs_validator ON txs(validator_address) WHERE validator_address != '';
		CREATE INDEX IF NOT EXISTS idx_txs_order ON txs(order_id) WHERE order_id != '';
		CREATE INDEX IF NOT EXISTS idx_txs_poll ON txs(poll_hash) WHERE poll_hash != '';
	`

	return db.Exec(ctx, query)
}

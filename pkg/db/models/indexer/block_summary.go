package indexer

import (
	"time"
)

const BlockSummariesProductionTableName = "block_summaries"
const BlockSummariesStagingTableName = "block_summaries_staging"

// BlockSummaryColumns defines the schema for the block_summaries table.
// This table has 70+ counter fields tracking all indexed entities per block.
var BlockSummaryColumns = []ColumnDef{
	{Name: "height", Type: "UInt64"},
	{Name: "height_time", Type: "DateTime64(6)"},
	{Name: "total_transactions", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Transaction counters (20 fields)
	{Name: "num_txs", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_send", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_stake", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_unstake", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_edit_stake", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_start_poll", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_vote_poll", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_lock_order", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_close_order", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_unknown", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_pause", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_unpause", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_change_parameter", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_dao_transfer", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_certificate_results", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_subsidy", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_create_order", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_edit_order", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_delete_order", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_dex_limit_order", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_dex_liquidity_deposit", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_txs_dex_liquidity_withdraw", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Account counters (2 fields)
	{Name: "num_accounts", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_accounts_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Event counters (10 fields)
	{Name: "num_events", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_reward", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_slash", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_dex_liquidity_deposit", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_dex_liquidity_withdraw", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_dex_swap", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_order_book_swap", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_automatic_pause", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_automatic_begin_unstaking", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_events_automatic_finish_unstaking", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Order counters (5 fields)
	{Name: "num_orders", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_orders_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_orders_open", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_orders_filled", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_orders_cancelled", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Pool counters (2 fields)
	{Name: "num_pools", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_pools_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Dex price counters (1 field)
	{Name: "num_dex_prices", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Dex order counters (6 fields)
	{Name: "num_dex_orders", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_orders_future", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_orders_locked", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_orders_complete", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_orders_success", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_orders_failed", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Dex deposit counters (4 fields)
	{Name: "num_dex_deposits", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_deposits_pending", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_deposits_locked", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_deposits_complete", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Dex withdrawal counters (4 fields)
	{Name: "num_dex_withdrawals", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_withdrawals_pending", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_withdrawals_locked", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_dex_withdrawals_complete", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Pool points counters (2 fields)
	{Name: "num_pool_points_holders", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_pool_points_holders_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Params counters (1 field)
	{Name: "params_changed", Type: "Bool DEFAULT false", Codec: "ZSTD"},
	// Validator counters (5 fields)
	{Name: "num_validators", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_validators_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_validators_active", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_validators_paused", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_validators_unstaking", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Validator non-signing info counters (2 fields)
	{Name: "num_validator_non_signing_info", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_validator_non_signing_info_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Validator double signing info counters (1 field)
	{Name: "num_validator_double_signing_info", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Committee counters (4 fields)
	{Name: "num_committees", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_committees_new", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_committees_subsidized", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "num_committees_retired", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Committee validator counters (1 field)
	{Name: "num_committee_validators", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Committee payment counters (1 field)
	{Name: "num_committee_payments", Type: "UInt32 DEFAULT 0", Codec: "Delta, ZSTD"},
	// Supply metrics (4 fields)
	{Name: "supply_changed", Type: "Bool DEFAULT false", Codec: "ZSTD"},
	{Name: "supply_total", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "supply_staked", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD"},
	{Name: "supply_delegated_only", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD"},
}

// BlockSummary stores aggregated entity counts for each indexed block.
// This table is separate from blocks to keep the blocks table immutable
// and avoid updating it with summary data after entity indexing completes.
//
// This model tracks all 16 indexed entities with comprehensive field coverage (90+ fields).
// All fields use individual typed columns (no maps) following the pattern from params.go.
type BlockSummary struct {
	// Block metadata
	Height            uint64    `ch:"height" json:"height"`
	HeightTime        time.Time `ch:"height_time" json:"height_time"`               // Block timestamp for time-range queries
	TotalTransactions uint64    `ch:"total_transactions" json:"total_transactions"` // Lifetime number of transactions across all blocks

	// ========== Transactions (20 fields) ==========
	NumTxs uint32 `ch:"num_txs" json:"num_txs"` // Total number of transactions

	// Transaction counts by message type (19 types)
	NumTxsSend                 uint32 `ch:"num_txs_send" json:"num_txs_send"`
	NumTxsStake                uint32 `ch:"num_txs_stake" json:"num_txs_stake"`
	NumTxsUnstake              uint32 `ch:"num_txs_unstake" json:"num_txs_unstake"`
	NumTxsEditStake            uint32 `ch:"num_txs_edit_stake" json:"num_txs_edit_stake"`
	NumTxsStartPoll            uint32 `ch:"num_txs_start_poll" json:"num_txs_start_poll"`   // Send tx with startPoll memo
	NumTxsVotePoll             uint32 `ch:"num_txs_vote_poll" json:"num_txs_vote_poll"`     // Send tx with votePoll memo
	NumTxsLockOrder            uint32 `ch:"num_txs_lock_order" json:"num_txs_lock_order"`   // Send tx with lockOrder memo
	NumTxsCloseOrder           uint32 `ch:"num_txs_close_order" json:"num_txs_close_order"` // Send tx with closeOrder memo
	NumTxsUnknown              uint32 `ch:"num_txs_unknown" json:"num_txs_unknown"`
	NumTxsPause                uint32 `ch:"num_txs_pause" json:"num_txs_pause"`
	NumTxsUnpause              uint32 `ch:"num_txs_unpause" json:"num_txs_unpause"`
	NumTxsChangeParameter      uint32 `ch:"num_txs_change_parameter" json:"num_txs_change_parameter"`
	NumTxsDaoTransfer          uint32 `ch:"num_txs_dao_transfer" json:"num_txs_dao_transfer"`
	NumTxsCertificateResults   uint32 `ch:"num_txs_certificate_results" json:"num_txs_certificate_results"`
	NumTxsSubsidy              uint32 `ch:"num_txs_subsidy" json:"num_txs_subsidy"`
	NumTxsCreateOrder          uint32 `ch:"num_txs_create_order" json:"num_txs_create_order"`
	NumTxsEditOrder            uint32 `ch:"num_txs_edit_order" json:"num_txs_edit_order"`
	NumTxsDeleteOrder          uint32 `ch:"num_txs_delete_order" json:"num_txs_delete_order"`
	NumTxsDexLimitOrder        uint32 `ch:"num_txs_dex_limit_order" json:"num_txs_dex_limit_order"`
	NumTxsDexLiquidityDeposit  uint32 `ch:"num_txs_dex_liquidity_deposit" json:"num_txs_dex_liquidity_deposit"`
	NumTxsDexLiquidityWithdraw uint32 `ch:"num_txs_dex_liquidity_withdraw" json:"num_txs_dex_liquidity_withdraw"`

	// ========== Accounts (2 fields) ==========
	NumAccounts    uint32 `ch:"num_accounts" json:"num_accounts"`         // Number of accounts that changed
	NumAccountsNew uint32 `ch:"num_accounts_new" json:"num_accounts_new"` // Number of new accounts created

	// ========== Events (10 fields) ==========
	NumEvents uint32 `ch:"num_events" json:"num_events"` // Total number of events

	// Event counts by type (9 types)
	NumEventsReward                   uint32 `ch:"num_events_reward" json:"num_events_reward"`
	NumEventsSlash                    uint32 `ch:"num_events_slash" json:"num_events_slash"`
	NumEventsDexLiquidityDeposit      uint32 `ch:"num_events_dex_liquidity_deposit" json:"num_events_dex_liquidity_deposit"`
	NumEventsDexLiquidityWithdraw     uint32 `ch:"num_events_dex_liquidity_withdraw" json:"num_events_dex_liquidity_withdraw"`
	NumEventsDexSwap                  uint32 `ch:"num_events_dex_swap" json:"num_events_dex_swap"`
	NumEventsOrderBookSwap            uint32 `ch:"num_events_order_book_swap" json:"num_events_order_book_swap"`
	NumEventsAutomaticPause           uint32 `ch:"num_events_automatic_pause" json:"num_events_automatic_pause"`
	NumEventsAutomaticBeginUnstaking  uint32 `ch:"num_events_automatic_begin_unstaking" json:"num_events_automatic_begin_unstaking"`
	NumEventsAutomaticFinishUnstaking uint32 `ch:"num_events_automatic_finish_unstaking" json:"num_events_automatic_finish_unstaking"`

	// ========== Orders (5 fields) ==========
	NumOrders          uint32 `ch:"num_orders" json:"num_orders"`                     // Total number of orders
	NumOrdersNew       uint32 `ch:"num_orders_new" json:"num_orders_new"`             // Number of new orders
	NumOrdersOpen      uint32 `ch:"num_orders_open" json:"num_orders_open"`           // Number of open orders
	NumOrdersFilled    uint32 `ch:"num_orders_filled" json:"num_orders_filled"`       // Number of filled orders
	NumOrdersCancelled uint32 `ch:"num_orders_cancelled" json:"num_orders_cancelled"` // Number of cancelled orders

	// ========== PoolsByHeight (2 fields) ==========
	NumPools    uint32 `ch:"num_pools" json:"num_pools"`         // Total number of pools
	NumPoolsNew uint32 `ch:"num_pools_new" json:"num_pools_new"` // Number of new pools created

	// ========== DexPricesByHeight (1 field) ==========
	NumDexPrices uint32 `ch:"num_dex_prices" json:"num_dex_prices"` // Number of DEX price records

	// ========== DexOrders (6 fields) ==========
	NumDexOrders         uint32 `ch:"num_dex_orders" json:"num_dex_orders"`                   // Total number of DEX orders
	NumDexOrdersFuture   uint32 `ch:"num_dex_orders_future" json:"num_dex_orders_future"`     // Number of future DEX orders
	NumDexOrdersLocked   uint32 `ch:"num_dex_orders_locked" json:"num_dex_orders_locked"`     // Number of locked DEX orders
	NumDexOrdersComplete uint32 `ch:"num_dex_orders_complete" json:"num_dex_orders_complete"` // Number of complete DEX orders
	NumDexOrdersSuccess  uint32 `ch:"num_dex_orders_success" json:"num_dex_orders_success"`   // Number of successful DEX orders
	NumDexOrdersFailed   uint32 `ch:"num_dex_orders_failed" json:"num_dex_orders_failed"`     // Number of failed DEX orders

	// ========== DexDeposits (4 fields) ==========
	NumDexDeposits         uint32 `ch:"num_dex_deposits" json:"num_dex_deposits"`                   // Total number of DEX deposits
	NumDexDepositsPending  uint32 `ch:"num_dex_deposits_pending" json:"num_dex_deposits_pending"`   // Number of pending DEX deposits
	NumDexDepositsLocked   uint32 `ch:"num_dex_deposits_locked" json:"num_dex_deposits_locked"`     // Number of locked DEX deposits
	NumDexDepositsComplete uint32 `ch:"num_dex_deposits_complete" json:"num_dex_deposits_complete"` // Number of complete DEX deposits

	// ========== DexWithdrawals (4 fields) ==========
	NumDexWithdrawals         uint32 `ch:"num_dex_withdrawals" json:"num_dex_withdrawals"`                   // Total number of DEX withdrawals
	NumDexWithdrawalsPending  uint32 `ch:"num_dex_withdrawals_pending" json:"num_dex_withdrawals_pending"`   // Number of pending DEX withdrawals
	NumDexWithdrawalsLocked   uint32 `ch:"num_dex_withdrawals_locked" json:"num_dex_withdrawals_locked"`     // Number of locked DEX withdrawals
	NumDexWithdrawalsComplete uint32 `ch:"num_dex_withdrawals_complete" json:"num_dex_withdrawals_complete"` // Number of complete DEX withdrawals

	// ========== PoolPointsByHolder (2 fields) ==========
	NumDexPoolPointsHolders    uint32 `ch:"num_pool_points_holders" json:"num_pool_points_holders"`         // Total number of pool point holders
	NumDexPoolPointsHoldersNew uint32 `ch:"num_pool_points_holders_new" json:"num_pool_points_holders_new"` // Number of new pool point holders

	// ========== Params (1 field) ==========
	ParamsChanged bool `ch:"params_changed" json:"params_changed"` // Whether chain parameters changed at this height

	// ========== Validators (5 fields) ==========
	NumValidators          uint32 `ch:"num_validators" json:"num_validators"`                     // Total number of validators
	NumValidatorsNew       uint32 `ch:"num_validators_new" json:"num_validators_new"`             // Number of new validators
	NumValidatorsActive    uint32 `ch:"num_validators_active" json:"num_validators_active"`       // Number of active validators
	NumValidatorsPaused    uint32 `ch:"num_validators_paused" json:"num_validators_paused"`       // Number of paused validators
	NumValidatorsUnstaking uint32 `ch:"num_validators_unstaking" json:"num_validators_unstaking"` // Number of unstaking validators

	// ========== ValidatorNonSigningInfo (2 fields) ==========
	NumValidatorNonSigningInfo    uint32 `ch:"num_validator_non_signing_info" json:"num_validator_non_signing_info"`         // Total number of non-signing info records
	NumValidatorNonSigningInfoNew uint32 `ch:"num_validator_non_signing_info_new" json:"num_validator_non_signing_info_new"` // Number of new non-signing info records

	// ========== ValidatorDoubleSigningInfo (1 field) ==========
	NumValidatorDoubleSigningInfo uint32 `ch:"num_validator_double_signing_info" json:"num_validator_double_signing_info"` // Total number of double signing info records

	// ========== Committees (4 fields) ==========
	NumCommittees           uint32 `ch:"num_committees" json:"num_committees"`                       // Total number of committees
	NumCommitteesNew        uint32 `ch:"num_committees_new" json:"num_committees_new"`               // Number of new committees
	NumCommitteesSubsidized uint32 `ch:"num_committees_subsidized" json:"num_committees_subsidized"` // Number of subsidized committees
	NumCommitteesRetired    uint32 `ch:"num_committees_retired" json:"num_committees_retired"`       // Number of retired committees

	// ========== CommitteeValidators (1 field) ==========
	NumCommitteeValidators uint32 `ch:"num_committee_validators" json:"num_committee_validators"` // Number of committee-validator relationships

	// ========== CommitteePayments (1 field) ==========
	NumCommitteePayments uint32 `ch:"num_committee_payments" json:"num_committee_payments"` // Number of committee payment records

	// ========== Supply (4 fields) ==========
	SupplyChanged       bool   `ch:"supply_changed" json:"supply_changed"`               // Whether supply changed at this height
	SupplyTotal         uint64 `ch:"supply_total" json:"supply_total"`                   // Total token supply
	SupplyStaked        uint64 `ch:"supply_staked" json:"supply_staked"`                 // Total staked tokens (validators + delegators)
	SupplyDelegatedOnly uint64 `ch:"supply_delegated_only" json:"supply_delegated_only"` // Delegated-only tokens (excluding validator stake)
}

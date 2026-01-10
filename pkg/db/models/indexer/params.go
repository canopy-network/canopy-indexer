package indexer

import "time"

const (
	// ParamsProductionTableName is the production table name for params
	ParamsProductionTableName = "params"
	// ParamsStagingTableName is the staging table name for params
	ParamsStagingTableName = "params_staging"
)

// ParamsColumns defines the schema for the params table.
// All numeric columns use Delta+ZSTD compression for optimal storage.
var ParamsColumns = []ColumnDef{
	{Name: "height", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "height_time", Type: "DateTime64(6)"},
	// Consensus parameters
	{Name: "block_size", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "protocol_version", Type: "String", Codec: "ZSTD(3)"},
	{Name: "root_chain_id", Type: "UInt16", Codec: "Delta, ZSTD(1)"},
	{Name: "retired", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	// Validator parameters
	{Name: "unstaking_blocks", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "max_pause_blocks", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "double_sign_slash_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "non_sign_slash_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "max_non_sign", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "non_sign_window", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "max_committees", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "max_committee_size", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "early_withdrawal_penalty", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "delegate_unstaking_blocks", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "minimum_order_size", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "stake_percent_for_subsidized", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "max_slash_per_committee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "delegate_reward_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "buy_deadline_blocks", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "lock_order_fee_multiplier", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "minimum_stake_for_validators", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "minimum_stake_for_delegates", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "maximum_delegates_per_committee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	// Fee parameters
	{Name: "send_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "stake_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "edit_stake_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "unstake_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "pause_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "unpause_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "change_parameter_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "dao_transfer_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "certificate_results_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "subsidy_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "create_order_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "edit_order_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "delete_order_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "dex_limit_order_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "dex_liquidity_deposit_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "dex_liquidity_withdraw_fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	// Governance parameters
	{Name: "dao_reward_percentage", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
}

// Params represents chain parameters at a specific height.
// This entity stores all governance parameters including consensus, validator, fee, and gov settings.
// Parameters are only inserted when they change to maintain a sparse historical record.
type Params struct {
	// Height and timestamp
	Height     uint64    `ch:"height" json:"height"`           // Block height when these params became effective
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Timestamp of the block

	// Consensus parameters (4 fields)
	BlockSize       uint64 `ch:"block_size" json:"block_size"`             // Maximum allowed block size in bytes
	ProtocolVersion string `ch:"protocol_version" json:"protocol_version"` // Minimum protocol version required
	RootChainID     uint16 `ch:"root_chain_id" json:"root_chain_id"`       // Root chain ID (parent chain)
	Retired         uint64 `ch:"retired" json:"retired"`                   // Whether chain is retired (0=active, 1=retired)

	// Validator parameters (19 fields)
	UnstakingBlocks              uint64 `ch:"unstaking_blocks" json:"unstaking_blocks"`                               // Number of blocks before unstaking completes
	MaxPauseBlocks               uint64 `ch:"max_pause_blocks" json:"max_pause_blocks"`                               // Maximum blocks a validator can be paused
	DoubleSignSlashPercentage    uint64 `ch:"double_sign_slash_percentage" json:"double_sign_slash_percentage"`       // Slash percentage for double signing
	NonSignSlashPercentage       uint64 `ch:"non_sign_slash_percentage" json:"non_sign_slash_percentage"`             // Slash percentage for not signing
	MaxNonSign                   uint64 `ch:"max_non_sign" json:"max_non_sign"`                                       // Maximum allowed non-sign events
	NonSignWindow                uint64 `ch:"non_sign_window" json:"non_sign_window"`                                 // Window size for counting non-signs
	MaxCommittees                uint64 `ch:"max_committees" json:"max_committees"`                                   // Maximum number of committees
	MaxCommitteeSize             uint64 `ch:"max_committee_size" json:"max_committee_size"`                           // Maximum size of a committee
	EarlyWithdrawalPenalty       uint64 `ch:"early_withdrawal_penalty" json:"early_withdrawal_penalty"`               // Penalty for early withdrawal
	DelegateUnstakingBlocks      uint64 `ch:"delegate_unstaking_blocks" json:"delegate_unstaking_blocks"`             // Blocks before delegate unstaking completes
	MinimumOrderSize             uint64 `ch:"minimum_order_size" json:"minimum_order_size"`                           // Minimum size for an order
	StakePercentForSubsidized    uint64 `ch:"stake_percent_for_subsidized" json:"stake_percent_for_subsidized"`       // Stake percentage required for subsidized committee
	MaxSlashPerCommittee         uint64 `ch:"max_slash_per_committee" json:"max_slash_per_committee"`                 // Maximum slash amount per committee
	DelegateRewardPercentage     uint64 `ch:"delegate_reward_percentage" json:"delegate_reward_percentage"`           // Percentage of rewards that go to delegates
	BuyDeadlineBlocks            uint64 `ch:"buy_deadline_blocks" json:"buy_deadline_blocks"`                         // Deadline in blocks for buy orders
	LockOrderFeeMultiplier       uint64 `ch:"lock_order_fee_multiplier" json:"lock_order_fee_multiplier"`             // Fee multiplier for locked orders
	MinimumStakeForValidators    uint64 `ch:"minimum_stake_for_validators" json:"minimum_stake_for_validators"`       // Minimum stake required to become a validator (from PR #261)
	MinimumStakeForDelegates     uint64 `ch:"minimum_stake_for_delegates" json:"minimum_stake_for_delegates"`         // Minimum stake required to delegate (from PR #261)
	MaximumDelegatesPerCommittee uint64 `ch:"maximum_delegates_per_committee" json:"maximum_delegates_per_committee"` // Maximum number of delegates allowed per committee (from PR #261)

	// Fee parameters (16 fields)
	SendFee                 uint64 `ch:"send_fee" json:"send_fee"`                                     // Fee for send transactions
	StakeFee                uint64 `ch:"stake_fee" json:"stake_fee"`                                   // Fee for staking
	EditStakeFee            uint64 `ch:"edit_stake_fee" json:"edit_stake_fee"`                         // Fee for editing stake
	UnstakeFee              uint64 `ch:"unstake_fee" json:"unstake_fee"`                               // Fee for unstaking
	PauseFee                uint64 `ch:"pause_fee" json:"pause_fee"`                                   // Fee for pausing validator
	UnpauseFee              uint64 `ch:"unpause_fee" json:"unpause_fee"`                               // Fee for unpausing validator
	ChangeParameterFee      uint64 `ch:"change_parameter_fee" json:"change_parameter_fee"`             // Fee for changing parameters
	DaoTransferFee          uint64 `ch:"dao_transfer_fee" json:"dao_transfer_fee"`                     // Fee for DAO transfers
	CertificateResultsFee   uint64 `ch:"certificate_results_fee" json:"certificate_results_fee"`       // Fee for certificate results
	SubsidyFee              uint64 `ch:"subsidy_fee" json:"subsidy_fee"`                               // Fee for subsidy transactions
	CreateOrderFee          uint64 `ch:"create_order_fee" json:"create_order_fee"`                     // Fee for creating orders
	EditOrderFee            uint64 `ch:"edit_order_fee" json:"edit_order_fee"`                         // Fee for editing orders
	DeleteOrderFee          uint64 `ch:"delete_order_fee" json:"delete_order_fee"`                     // Fee for deleting orders
	DexLimitOrderFee        uint64 `ch:"dex_limit_order_fee" json:"dex_limit_order_fee"`               // Fee for DEX limit orders
	DexLiquidityDepositFee  uint64 `ch:"dex_liquidity_deposit_fee" json:"dex_liquidity_deposit_fee"`   // Fee for DEX liquidity deposits
	DexLiquidityWithdrawFee uint64 `ch:"dex_liquidity_withdraw_fee" json:"dex_liquidity_withdraw_fee"` // Fee for DEX liquidity withdrawals

	// Governance parameters (1 field)
	DaoRewardPercentage uint64 `ch:"dao_reward_percentage" json:"dao_reward_percentage"` // Percentage of rewards that go to DAO
}

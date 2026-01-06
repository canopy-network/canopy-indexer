package indexer

import (
	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeParams(batch *pgx.Batch, data *BlockData) {
	p := transform.ParamsFromFSM(data.Params)
	if p == nil {
		return
	}
	p.Height = data.Height
	p.HeightTime = data.BlockTime

	batch.Queue(`
		INSERT INTO params (
			chain_id, height, height_time,
			block_size, protocol_version, root_chain_id, retired,
			unstaking_blocks, max_pause_blocks,
			double_sign_slash_percentage, non_sign_slash_percentage,
			max_non_sign, non_sign_window,
			max_committees, max_committee_size,
			early_withdrawal_penalty, delegate_unstaking_blocks,
			minimum_order_size, stake_percent_for_subsidized,
			max_slash_per_committee, delegate_reward_percentage,
			buy_deadline_blocks, lock_order_fee_multiplier,
			minimum_stake_for_validators, minimum_stake_for_delegates,
			maximum_delegates_per_committee,
			send_fee, stake_fee, edit_stake_fee, unstake_fee,
			pause_fee, unpause_fee, change_parameter_fee,
			dao_transfer_fee, certificate_results_fee, subsidy_fee,
			create_order_fee, edit_order_fee, delete_order_fee,
			dex_limit_order_fee, dex_liquidity_deposit_fee, dex_liquidity_withdraw_fee,
			dao_reward_percentage
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13,
			$14, $15,
			$16, $17,
			$18, $19,
			$20, $21,
			$22, $23,
			$24, $25,
			$26,
			$27, $28, $29, $30,
			$31, $32, $33,
			$34, $35, $36,
			$37, $38, $39,
			$40, $41, $42,
			$43
		)
		ON CONFLICT (chain_id, height) DO NOTHING
	`,
		data.ChainID, p.Height, p.HeightTime,
		p.BlockSize,
		p.ProtocolVersion,
		p.RootChainID,
		0, // retired
		p.UnstakingBlocks,
		p.MaxPauseBlocks,
		p.DoubleSignSlashPercentage,
		p.NonSignSlashPercentage,
		p.MaxNonSign,
		p.NonSignWindow,
		p.MaxCommittees,
		p.MaxCommitteeSize,
		p.EarlyWithdrawalPenalty,
		p.DelegateUnstakingBlocks,
		p.MinimumOrderSize,
		p.StakePercentForSubsidizedCommittee,
		p.MaxSlashPerCommittee,
		p.DelegateRewardPercentage,
		p.BuyDeadlineBlocks,
		p.LockOrderFeeMultiplier,
		p.MinimumStakeForValidators,
		p.MinimumStakeForDelegates,
		p.MaximumDelegatesPerCommittee,
		p.SendFee,
		p.StakeFee,
		p.EditStakeFee,
		p.UnstakeFee,
		p.PauseFee,
		p.UnpauseFee,
		p.ChangeParameterFee,
		p.DaoTransferFee,
		p.CertificateResultsFee,
		p.SubsidyFee,
		p.CreateOrderFee,
		p.EditOrderFee,
		p.DeleteOrderFee,
		p.DexLimitOrderFee,
		p.DexLiquidityDepositFee,
		p.DexLiquidityWithdrawFee,
		p.DaoRewardPercentage,
	)
}

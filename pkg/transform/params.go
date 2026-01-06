package transform

import (
	"time"

	"github.com/canopy-network/canopy/fsm"
)

// Params holds the transformed params data for database insertion.
type Params struct {
	Height     uint64
	HeightTime time.Time

	// Consensus params
	BlockSize       uint64
	ProtocolVersion string
	RootChainID     uint64

	// Validator params
	UnstakingBlocks                    uint64
	MaxPauseBlocks                     uint64
	DoubleSignSlashPercentage          uint64
	NonSignSlashPercentage             uint64
	MaxNonSign                         uint64
	NonSignWindow                      uint64
	MaxCommittees                      uint64
	MaxCommitteeSize                   uint64
	EarlyWithdrawalPenalty             uint64
	DelegateUnstakingBlocks            uint64
	MinimumOrderSize                   uint64
	StakePercentForSubsidizedCommittee uint64
	MaxSlashPerCommittee               uint64
	DelegateRewardPercentage           uint64
	BuyDeadlineBlocks                  uint64
	LockOrderFeeMultiplier             uint64
	MinimumStakeForValidators          uint64
	MinimumStakeForDelegates           uint64
	MaximumDelegatesPerCommittee       uint64

	// Fee params
	SendFee                 uint64
	StakeFee                uint64
	EditStakeFee            uint64
	UnstakeFee              uint64
	PauseFee                uint64
	UnpauseFee              uint64
	ChangeParameterFee      uint64
	DaoTransferFee          uint64
	CertificateResultsFee   uint64
	SubsidyFee              uint64
	CreateOrderFee          uint64
	EditOrderFee            uint64
	DeleteOrderFee          uint64
	DexLimitOrderFee        uint64
	DexLiquidityDepositFee  uint64
	DexLiquidityWithdrawFee uint64

	// Gov params
	DaoRewardPercentage uint64
}

// ParamsFromFSM converts fsm.Params to the database Params model.
// Height and HeightTime must be set by the caller.
// Returns nil if the input is nil or has no valid data.
func ParamsFromFSM(p *fsm.Params) *Params {
	if p == nil {
		return nil
	}

	result := &Params{}

	// Consensus params
	if p.Consensus != nil {
		result.BlockSize = p.Consensus.BlockSize
		result.ProtocolVersion = p.Consensus.ProtocolVersion
		result.RootChainID = p.Consensus.RootChainId
	}

	// Validator params
	if p.Validator != nil {
		result.UnstakingBlocks = p.Validator.UnstakingBlocks
		result.MaxPauseBlocks = p.Validator.MaxPauseBlocks
		result.DoubleSignSlashPercentage = p.Validator.DoubleSignSlashPercentage
		result.NonSignSlashPercentage = p.Validator.NonSignSlashPercentage
		result.MaxNonSign = p.Validator.MaxNonSign
		result.NonSignWindow = p.Validator.NonSignWindow
		result.MaxCommittees = p.Validator.MaxCommittees
		result.MaxCommitteeSize = p.Validator.MaxCommitteeSize
		result.EarlyWithdrawalPenalty = p.Validator.EarlyWithdrawalPenalty
		result.DelegateUnstakingBlocks = p.Validator.DelegateUnstakingBlocks
		result.MinimumOrderSize = p.Validator.MinimumOrderSize
		result.StakePercentForSubsidizedCommittee = p.Validator.StakePercentForSubsidizedCommittee
		result.MaxSlashPerCommittee = p.Validator.MaxSlashPerCommittee
		result.DelegateRewardPercentage = p.Validator.DelegateRewardPercentage
		result.BuyDeadlineBlocks = p.Validator.BuyDeadlineBlocks
		result.LockOrderFeeMultiplier = p.Validator.LockOrderFeeMultiplier
		result.MinimumStakeForValidators = p.Validator.MinimumStakeForValidators
		result.MinimumStakeForDelegates = p.Validator.MinimumStakeForDelegates
		result.MaximumDelegatesPerCommittee = p.Validator.MaximumDelegatesPerCommittee
	}

	// Fee params
	if p.Fee != nil {
		result.SendFee = p.Fee.SendFee
		result.StakeFee = p.Fee.StakeFee
		result.EditStakeFee = p.Fee.EditStakeFee
		result.UnstakeFee = p.Fee.UnstakeFee
		result.PauseFee = p.Fee.PauseFee
		result.UnpauseFee = p.Fee.UnpauseFee
		result.ChangeParameterFee = p.Fee.ChangeParameterFee
		result.DaoTransferFee = p.Fee.DaoTransferFee
		result.CertificateResultsFee = p.Fee.CertificateResultsFee
		result.SubsidyFee = p.Fee.SubsidyFee
		result.CreateOrderFee = p.Fee.CreateOrderFee
		result.EditOrderFee = p.Fee.EditOrderFee
		result.DeleteOrderFee = p.Fee.DeleteOrderFee
		result.DexLimitOrderFee = p.Fee.DexLimitOrderFee
		result.DexLiquidityDepositFee = p.Fee.DexLiquidityDepositFee
		result.DexLiquidityWithdrawFee = p.Fee.DexLiquidityWithdrawFee
	}

	// Governance params
	if p.Governance != nil {
		result.DaoRewardPercentage = p.Governance.DaoRewardPercentage
	}

	return result
}

package indexer

import (
	"time"

	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
)

// CanopyxBlockData holds all converted data ready for canopyx database insertion.
// This structure contains all the fields needed for transactional writes.
type CanopyxBlockData struct {
	// Metadata
	ChainID   uint64
	Height    uint64
	BlockTime time.Time

	// Core entities
	Block        *indexermodels.Block
	Transactions []*indexermodels.Transaction
	Events       []*indexermodels.Event
	Accounts     []*indexermodels.Account
	Params       *indexermodels.Params
	Supply       *indexermodels.Supply

	// Validators (change-detected)
	Validators                 []*indexermodels.Validator
	ValidatorNonSigningInfo    []*indexermodels.ValidatorNonSigningInfo
	ValidatorDoubleSigningInfo []*indexermodels.ValidatorDoubleSigningInfo

	// Pools (change-detected for holders)
	Pools              []*indexermodels.Pool
	PoolPointsByHolder []*indexermodels.PoolPointsByHolder

	// Orders
	Orders []*indexermodels.Order

	// DEX (state machine)
	DexPrices      []*indexermodels.DexPrice
	DexOrders      []*indexermodels.DexOrder
	DexDeposits    []*indexermodels.DexDeposit
	DexWithdrawals []*indexermodels.DexWithdrawal

	// Committees
	Committees          []*indexermodels.Committee
	CommitteeValidators []*indexermodels.CommitteeValidator
	CommitteePayments   []*indexermodels.CommitteePayment

	// Block summary (computed after conversion)
	BlockSummary *indexermodels.BlockSummary
}

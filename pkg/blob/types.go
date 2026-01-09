package blob

import (
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// BlockData represents all data for a block, populated during fetch phase and consumed during write phase.
type BlockData struct {
	// Core identifiers
	ChainID   uint64
	Height    uint64
	BlockTime time.Time

	// Block data (fetched with retry logic)
	Block *lib.BlockResult

	// Simple fetches (single RPC call each)
	Transactions []*lib.TxResult
	Events       []*lib.Event
	Accounts     []*fsm.Account
	Orders       []*lib.SellOrder
	DexPrices    []*lib.DexPrice
	Params       *fsm.Params
	Supply       *fsm.Supply
	Committees   []*lib.CommitteeData

	// Shared data (used by both validators and committees)
	// Consolidates duplicate RPC calls
	SubsidizedCommittees []uint64
	RetiredCommittees    []uint64

	// Change detection pairs: current + previous (H-1)
	ValidatorsCurrent  []*fsm.Validator
	ValidatorsPrevious []*fsm.Validator

	PoolsCurrent  []*fsm.Pool
	PoolsPrevious []*fsm.Pool

	NonSignersCurrent  []*fsm.NonSigner
	NonSignersPrevious []*fsm.NonSigner

	DoubleSignersCurrent  []*lib.DoubleSigner
	DoubleSignersPrevious []*lib.DoubleSigner

	// DEX batch data (4 variants for completion detection)
	DexBatchesCurrent      []*lib.DexBatch
	DexBatchesNext         []*lib.DexBatch
	DexBatchesPreviousCurr []*lib.DexBatch
	DexBatchesPreviousNext []*lib.DexBatch
}

package rpc

import (
	"context"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// Client captures the RPC calls used by activities when indexing blocks and transactions.
type Client interface {
	ChainHead(ctx context.Context) (uint64, error)
	BlockByHeight(ctx context.Context, height uint64) (*lib.BlockResult, error)
	TxsByHeight(ctx context.Context, height uint64) ([]*lib.TxResult, error)
	EventsByHeight(ctx context.Context, height uint64) ([]*lib.Event, error)
	AccountsByHeight(ctx context.Context, height uint64) ([]*fsm.Account, error)
	ValidatorsByHeight(ctx context.Context, height uint64) ([]*fsm.Validator, error)
	OrdersByHeight(ctx context.Context, height uint64) ([]*lib.SellOrder, error)
	DexPricesByHeight(ctx context.Context, height uint64) ([]*lib.DexPrice, error)
	PoolsByHeight(ctx context.Context, height uint64) ([]*fsm.Pool, error)
	AllDexBatchesByHeight(ctx context.Context, height uint64) ([]*lib.DexBatch, error)
	AllNextDexBatchesByHeight(ctx context.Context, height uint64) ([]*lib.DexBatch, error)
	AllParamsByHeight(ctx context.Context, height uint64) (*fsm.Params, error)
	ValParamsByHeight(ctx context.Context, height uint64) (*fsm.ValidatorParams, error)
	NonSignersByHeight(ctx context.Context, height uint64) ([]*fsm.NonSigner, error)
	DoubleSignersByHeight(ctx context.Context, height uint64) ([]*lib.DoubleSigner, error)
	CommitteesDataByHeight(ctx context.Context, height uint64) ([]*lib.CommitteeData, error)
	SubsidizedCommitteesByHeight(ctx context.Context, height uint64) ([]uint64, error)
	RetiredCommitteesByHeight(ctx context.Context, height uint64) ([]uint64, error)
	SupplyByHeight(ctx context.Context, height uint64) (*fsm.Supply, error)
	Poll(ctx context.Context) (fsm.Poll, error)
	Proposals(ctx context.Context) (fsm.GovProposals, error)
}

// Factory produces RPC clients for a given set of endpoints.
type Factory interface {
	NewClient(endpoints []string) Client
}

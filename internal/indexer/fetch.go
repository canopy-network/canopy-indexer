package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"golang.org/x/sync/errgroup"
)

// fetchAllData fetches all data needed for indexing in parallel.
// Any RPC failure causes immediate return with error (NACK semantics).
// On retry, cached RPC calls return immediately from the 10-block rolling cache.
func (idx *Indexer) fetchAllData(ctx context.Context, rpc *rpc.HTTPClient, chainID, height uint64, block *lib.BlockResult, blockTime time.Time) (*blob.BlockData, error) {
	data := &blob.BlockData{
		ChainID:   chainID,
		Height:    height,
		BlockTime: blockTime,
		Block:     block,
	}

	g, gCtx := errgroup.WithContext(ctx)

	// ==================== Simple single fetches ====================

	g.Go(func() error {
		var err error
		data.Transactions, err = rpc.TxsByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.Events, err = rpc.EventsByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.Accounts, err = rpc.AccountsByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.Orders, err = rpc.OrdersByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.DexPrices, err = rpc.DexPricesByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.Params, err = rpc.AllParamsByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.Supply, err = rpc.SupplyByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.Committees, err = rpc.CommitteesDataByHeight(gCtx, height)
		return err
	})

	// ==================== Shared data (validators + committees) ====================

	g.Go(func() error {
		var err error
		data.SubsidizedCommittees, err = rpc.SubsidizedCommitteesByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.RetiredCommittees, err = rpc.RetiredCommitteesByHeight(gCtx, height)
		return err
	})

	// ==================== Change detection pairs ====================

	// Validators: current + H-1
	g.Go(func() error {
		var err error
		data.ValidatorsCurrent, err = rpc.ValidatorsByHeight(gCtx, height)
		return err
	})
	g.Go(func() error {
		if height <= 1 {
			data.ValidatorsPrevious = make([]*fsm.Validator, 0)
			return nil
		}
		var err error
		data.ValidatorsPrevious, err = rpc.ValidatorsByHeight(gCtx, height-1)
		return err
	})

	// Pools: current + H-1
	g.Go(func() error {
		var err error
		data.PoolsCurrent, err = rpc.PoolsByHeight(gCtx, height)
		return err
	})
	g.Go(func() error {
		if height <= 1 {
			data.PoolsPrevious = make([]*fsm.Pool, 0)
			return nil
		}
		var err error
		data.PoolsPrevious, err = rpc.PoolsByHeight(gCtx, height-1)
		return err
	})

	// NonSigners: current + H-1
	g.Go(func() error {
		var err error
		data.NonSignersCurrent, err = rpc.NonSignersByHeight(gCtx, height)
		return err
	})
	g.Go(func() error {
		if height <= 1 {
			data.NonSignersPrevious = make([]*fsm.NonSigner, 0)
			return nil
		}
		var err error
		data.NonSignersPrevious, err = rpc.NonSignersByHeight(gCtx, height-1)
		return err
	})

	// DoubleSigners: current + H-1
	g.Go(func() error {
		var err error
		data.DoubleSignersCurrent, err = rpc.DoubleSignersByHeight(gCtx, height)
		return err
	})
	g.Go(func() error {
		if height <= 1 {
			data.DoubleSignersPrevious = make([]*lib.DoubleSigner, 0)
			return nil
		}
		var err error
		data.DoubleSignersPrevious, err = rpc.DoubleSignersByHeight(gCtx, height-1)
		return err
	})

	// ==================== DEX batch data (4 variants) ====================

	g.Go(func() error {
		var err error
		data.DexBatchesCurrent, err = rpc.AllDexBatchesByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		data.DexBatchesNext, err = rpc.AllNextDexBatchesByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		if height <= 1 {
			data.DexBatchesPreviousCurr = make([]*lib.DexBatch, 0)
			return nil
		}
		var err error
		data.DexBatchesPreviousCurr, err = rpc.AllDexBatchesByHeight(gCtx, height-1)
		return err
	})

	g.Go(func() error {
		if height <= 1 {
			data.DexBatchesPreviousNext = make([]*lib.DexBatch, 0)
			return nil
		}
		var err error
		data.DexBatchesPreviousNext, err = rpc.AllNextDexBatchesByHeight(gCtx, height-1)
		return err
	})

	// Wait for all fetches - any error fails the entire phase
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return data, nil
}

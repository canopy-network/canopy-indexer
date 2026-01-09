package blob

import (
	"fmt"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/indexer"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// Decode converts fsm.IndexerBlobs to an indexer.BlockData struct.
// The blobs contain both current and previous height data for change detection.
func Decode(blobs *fsm.IndexerBlobs, chainID uint64) (*indexer.BlockData, error) {
	if blobs == nil || blobs.Current == nil {
		return nil, fmt.Errorf("blobs or current blob is nil")
	}

	current, err := decodeBlob(blobs.Current)
	if err != nil {
		return nil, fmt.Errorf("decode current blob: %w", err)
	}

	var previous *decodedBlob
	if blobs.Previous != nil {
		previous, err = decodeBlob(blobs.Previous)
		if err != nil {
			return nil, fmt.Errorf("decode previous blob: %w", err)
		}
	}

	return buildBlockData(chainID, current, previous)
}

// decodedBlob holds the decoded contents of a single IndexerBlob.
type decodedBlob struct {
	block                *lib.BlockResult
	accounts             []*fsm.Account
	pools                []*fsm.Pool
	validators           []*fsm.Validator
	dexPrices            []*lib.DexPrice
	nonSigners           []*fsm.NonSigner
	doubleSigners        []*lib.DoubleSigner
	orders               []*lib.SellOrder
	params               *fsm.Params
	dexBatches           []*lib.DexBatch
	nextDexBatches       []*lib.DexBatch
	committeesData       []*lib.CommitteeData
	subsidizedCommittees []uint64
	retiredCommittees    []uint64
	supply               *fsm.Supply
}

// decodeBlob deserializes a single fsm.IndexerBlob into decoded structures.
func decodeBlob(blob *fsm.IndexerBlob) (*decodedBlob, error) {
	if blob == nil {
		return nil, fmt.Errorf("blob is nil")
	}

	out := &decodedBlob{
		subsidizedCommittees: blob.SubsidizedCommittees,
		retiredCommittees:    blob.RetiredCommittees,
	}

	// Decode block (contains transactions and events)
	if len(blob.Block) > 0 {
		var block lib.BlockResult
		if err := lib.Unmarshal(blob.Block, &block); err != nil {
			return nil, fmt.Errorf("unmarshal block: %w", err)
		}
		out.block = &block
	}

	// Decode params
	if len(blob.Params) > 0 {
		var params fsm.Params
		if err := lib.Unmarshal(blob.Params, &params); err != nil {
			return nil, fmt.Errorf("unmarshal params: %w", err)
		}
		out.params = &params
	}

	// Decode supply
	if len(blob.Supply) > 0 {
		var supply fsm.Supply
		if err := lib.Unmarshal(blob.Supply, &supply); err != nil {
			return nil, fmt.Errorf("unmarshal supply: %w", err)
		}
		out.supply = &supply
	}

	// Decode committees data
	if len(blob.CommitteesData) > 0 {
		var committees lib.CommitteesData
		if err := lib.Unmarshal(blob.CommitteesData, &committees); err != nil {
			return nil, fmt.Errorf("unmarshal committees data: %w", err)
		}
		out.committeesData = committees.List
	}

	// Decode accounts
	if blob.Accounts != nil {
		out.accounts = make([]*fsm.Account, 0, len(blob.Accounts))
		for _, raw := range blob.Accounts {
			var acc fsm.Account
			if err := lib.Unmarshal(raw, &acc); err != nil {
				return nil, fmt.Errorf("unmarshal account: %w", err)
			}
			out.accounts = append(out.accounts, &acc)
		}
	}

	// Decode pools
	if blob.Pools != nil {
		out.pools = make([]*fsm.Pool, 0, len(blob.Pools))
		for _, raw := range blob.Pools {
			var pool fsm.Pool
			if err := lib.Unmarshal(raw, &pool); err != nil {
				return nil, fmt.Errorf("unmarshal pool: %w", err)
			}
			out.pools = append(out.pools, &pool)
		}
	}

	// Decode validators
	if blob.Validators != nil {
		out.validators = make([]*fsm.Validator, 0, len(blob.Validators))
		for _, raw := range blob.Validators {
			var validator fsm.Validator
			if err := lib.Unmarshal(raw, &validator); err != nil {
				return nil, fmt.Errorf("unmarshal validator: %w", err)
			}
			out.validators = append(out.validators, &validator)
		}
	}

	// Decode dex prices
	if blob.DexPrices != nil {
		out.dexPrices = make([]*lib.DexPrice, 0, len(blob.DexPrices))
		for _, raw := range blob.DexPrices {
			var price lib.DexPrice
			if err := lib.Unmarshal(raw, &price); err != nil {
				return nil, fmt.Errorf("unmarshal dex price: %w", err)
			}
			out.dexPrices = append(out.dexPrices, &price)
		}
	}

	// Decode non-signers
	if blob.NonSigners != nil {
		out.nonSigners = make([]*fsm.NonSigner, 0, len(blob.NonSigners))
		for _, raw := range blob.NonSigners {
			var nonSigner fsm.NonSigner
			if err := lib.Unmarshal(raw, &nonSigner); err != nil {
				return nil, fmt.Errorf("unmarshal non-signer: %w", err)
			}
			out.nonSigners = append(out.nonSigners, &nonSigner)
		}
	}

	// Decode double-signers
	if blob.DoubleSigners != nil {
		out.doubleSigners = make([]*lib.DoubleSigner, 0, len(blob.DoubleSigners))
		for _, raw := range blob.DoubleSigners {
			var doubleSigner lib.DoubleSigner
			if err := lib.Unmarshal(raw, &doubleSigner); err != nil {
				return nil, fmt.Errorf("unmarshal double-signer: %w", err)
			}
			out.doubleSigners = append(out.doubleSigners, &doubleSigner)
		}
	}

	// Decode orders (flatten from OrderBooks)
	if len(blob.Orders) > 0 {
		var orderBooks lib.OrderBooks
		if err := lib.Unmarshal(blob.Orders, &orderBooks); err != nil {
			return nil, fmt.Errorf("unmarshal orders: %w", err)
		}
		for _, book := range orderBooks.OrderBooks {
			out.orders = append(out.orders, book.Orders...)
		}
	}

	// Decode dex batches
	if blob.DexBatches != nil {
		out.dexBatches = make([]*lib.DexBatch, 0, len(blob.DexBatches))
		for _, raw := range blob.DexBatches {
			var batch lib.DexBatch
			if err := lib.Unmarshal(raw, &batch); err != nil {
				return nil, fmt.Errorf("unmarshal dex batch: %w", err)
			}
			out.dexBatches = append(out.dexBatches, &batch)
		}
	}

	// Decode next dex batches
	if blob.NextDexBatches != nil {
		out.nextDexBatches = make([]*lib.DexBatch, 0, len(blob.NextDexBatches))
		for _, raw := range blob.NextDexBatches {
			var batch lib.DexBatch
			if err := lib.Unmarshal(raw, &batch); err != nil {
				return nil, fmt.Errorf("unmarshal next dex batch: %w", err)
			}
			out.nextDexBatches = append(out.nextDexBatches, &batch)
		}
	}

	return out, nil
}

// buildBlockData maps decoded current and previous blobs to indexer.BlockData.
func buildBlockData(chainID uint64, current, previous *decodedBlob) (*indexer.BlockData, error) {
	if current == nil || current.block == nil {
		return nil, fmt.Errorf("current blob has no block data")
	}

	data := &indexer.BlockData{
		ChainID: chainID,
		Height:  current.block.BlockHeader.Height,
	}

	// Extract block time
	if current.block.BlockHeader != nil {
		data.BlockTime = time.UnixMicro(int64(current.block.BlockHeader.Time))
	}

	// Block data
	data.Block = current.block
	data.Transactions = current.block.Transactions
	data.Events = current.block.Events

	// Simple fields from current blob
	data.Accounts = current.accounts
	data.Orders = current.orders
	data.DexPrices = current.dexPrices
	data.Params = current.params
	data.Supply = current.supply
	data.Committees = current.committeesData
	data.SubsidizedCommittees = current.subsidizedCommittees
	data.RetiredCommittees = current.retiredCommittees

	// Current height data for change detection
	data.ValidatorsCurrent = current.validators
	data.PoolsCurrent = current.pools
	data.NonSignersCurrent = current.nonSigners
	data.DoubleSignersCurrent = current.doubleSigners
	data.DexBatchesCurrent = current.dexBatches
	data.DexBatchesNext = current.nextDexBatches

	// Previous height data for change detection (H-1)
	if previous != nil {
		data.ValidatorsPrevious = previous.validators
		data.PoolsPrevious = previous.pools
		data.NonSignersPrevious = previous.nonSigners
		data.DoubleSignersPrevious = previous.doubleSigners
		data.DexBatchesPreviousCurr = previous.dexBatches
		data.DexBatchesPreviousNext = previous.nextDexBatches
	}

	return data, nil
}

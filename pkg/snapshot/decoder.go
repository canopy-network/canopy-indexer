package snapshot

import (
	"fmt"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/indexer"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// Decode converts a lib.IndexerSnapshot to an indexer.BlockData struct.
// This allows the indexer to bypass HTTP RPC fetching and write directly from WebSocket data.
func Decode(snapshot *lib.IndexerSnapshot, chainID uint64) (*indexer.BlockData, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot is nil")
	}

	data := &indexer.BlockData{
		ChainID: chainID,
		Height:  snapshot.Height,
	}

	// Extract block time from block header
	if snapshot.Block != nil && snapshot.Block.BlockHeader != nil {
		data.BlockTime = time.UnixMicro(int64(snapshot.Block.BlockHeader.Time))
	}

	// Direct fields (no conversion needed)
	data.Block = snapshot.Block
	data.Transactions = snapshot.Transactions
	data.Events = snapshot.Events
	data.SubsidizedCommittees = snapshot.SubsidizedCommittees
	data.RetiredCommittees = snapshot.RetiredCommittees
	data.DoubleSignersCurrent = snapshot.DoubleSignersCurrent
	data.DexBatchesCurrent = snapshot.DexBatchesCurrent
	data.DexBatchesPreviousCurr = snapshot.DexBatchesPrevious
	data.DexBatchesNext = snapshot.NextDexBatchesCurrent
	data.DexBatchesPreviousNext = snapshot.NextDexBatchesPrevious

	// Flatten OrderBooks -> []SellOrder
	data.Orders = flattenOrders(snapshot.Orders)

	// Extract CommitteesData list
	if snapshot.CommitteesData != nil {
		data.Committees = snapshot.CommitteesData.GetList()
	}

	// Decode [][]byte fields to their respective types
	var err error

	data.Accounts, err = decodeAccounts(snapshot.Accounts)
	if err != nil {
		return nil, fmt.Errorf("decode accounts: %w", err)
	}

	data.DexPrices, err = decodeDexPrices(snapshot.DexPrices)
	if err != nil {
		return nil, fmt.Errorf("decode dex prices: %w", err)
	}

	data.Params, err = decodeParams(snapshot.Params)
	if err != nil {
		return nil, fmt.Errorf("decode params: %w", err)
	}

	data.Supply, err = decodeSupply(snapshot.Supply)
	if err != nil {
		return nil, fmt.Errorf("decode supply: %w", err)
	}

	data.ValidatorsCurrent, err = decodeValidators(snapshot.ValidatorsCurrent)
	if err != nil {
		return nil, fmt.Errorf("decode validators current: %w", err)
	}

	data.ValidatorsPrevious, err = decodeValidators(snapshot.ValidatorsPrevious)
	if err != nil {
		return nil, fmt.Errorf("decode validators previous: %w", err)
	}

	data.PoolsCurrent, err = decodePools(snapshot.PoolsCurrent)
	if err != nil {
		return nil, fmt.Errorf("decode pools current: %w", err)
	}

	data.PoolsPrevious, err = decodePools(snapshot.PoolsPrevious)
	if err != nil {
		return nil, fmt.Errorf("decode pools previous: %w", err)
	}

	data.NonSignersCurrent, err = decodeNonSigners(snapshot.NonSignersCurrent)
	if err != nil {
		return nil, fmt.Errorf("decode non-signers current: %w", err)
	}

	data.NonSignersPrevious, err = decodeNonSigners(snapshot.NonSignersPrevious)
	if err != nil {
		return nil, fmt.Errorf("decode non-signers previous: %w", err)
	}

	return data, nil
}

// flattenOrders extracts all SellOrders from OrderBooks into a flat slice.
func flattenOrders(books *lib.OrderBooks) []*lib.SellOrder {
	if books == nil {
		return nil
	}
	var orders []*lib.SellOrder
	for _, book := range books.OrderBooks {
		if book != nil {
			orders = append(orders, book.Orders...)
		}
	}
	return orders
}

// decodeAccounts deserializes [][]byte to []*fsm.Account.
func decodeAccounts(raw [][]byte) ([]*fsm.Account, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	accounts := make([]*fsm.Account, 0, len(raw))
	for i, b := range raw {
		acc := new(fsm.Account)
		if err := lib.Unmarshal(b, acc); err != nil {
			return nil, fmt.Errorf("account[%d]: %w", i, err)
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// decodeDexPrices deserializes [][]byte to []*lib.DexPrice.
func decodeDexPrices(raw [][]byte) ([]*lib.DexPrice, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	prices := make([]*lib.DexPrice, 0, len(raw))
	for i, b := range raw {
		price := new(lib.DexPrice)
		if err := lib.Unmarshal(b, price); err != nil {
			return nil, fmt.Errorf("dex_price[%d]: %w", i, err)
		}
		prices = append(prices, price)
	}
	return prices, nil
}

// decodeParams deserializes []byte to *fsm.Params.
func decodeParams(raw []byte) (*fsm.Params, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	params := new(fsm.Params)
	if err := lib.Unmarshal(raw, params); err != nil {
		return nil, err
	}
	return params, nil
}

// decodeSupply deserializes []byte to *fsm.Supply.
func decodeSupply(raw []byte) (*fsm.Supply, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	supply := new(fsm.Supply)
	if err := lib.Unmarshal(raw, supply); err != nil {
		return nil, err
	}
	return supply, nil
}

// decodeValidators deserializes [][]byte to []*fsm.Validator.
func decodeValidators(raw [][]byte) ([]*fsm.Validator, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	validators := make([]*fsm.Validator, 0, len(raw))
	for i, b := range raw {
		val := new(fsm.Validator)
		if err := lib.Unmarshal(b, val); err != nil {
			return nil, fmt.Errorf("validator[%d]: %w", i, err)
		}
		validators = append(validators, val)
	}
	return validators, nil
}

// decodePools deserializes [][]byte to []*fsm.Pool.
func decodePools(raw [][]byte) ([]*fsm.Pool, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	pools := make([]*fsm.Pool, 0, len(raw))
	for i, b := range raw {
		pool := new(fsm.Pool)
		if err := lib.Unmarshal(b, pool); err != nil {
			return nil, fmt.Errorf("pool[%d]: %w", i, err)
		}
		pools = append(pools, pool)
	}
	return pools, nil
}

// decodeNonSigners deserializes []byte (NonSignerList proto) to []*fsm.NonSigner.
func decodeNonSigners(raw []byte) ([]*fsm.NonSigner, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	list := new(fsm.NonSignerList)
	if err := lib.Unmarshal(raw, list); err != nil {
		return nil, err
	}
	return list.GetList(), nil
}

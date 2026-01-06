package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

// DEX states
const (
	DexStateFuture   = "future"  // dex_order_state initial state
	DexStatePending  = "pending" // dex_deposit_state/dex_withdrawal_state initial state
	DexStateLocked   = "locked"
	DexStateComplete = "complete"
)

func (idx *Indexer) indexDexPrices(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	prices, err := rpc.DexPricesByHeight(ctx, height)
	if err != nil {
		return err
	}

	if len(prices) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, price := range prices {
		p := transform.DexPriceFromLib(price)
		batch.Queue(`
			INSERT INTO dex_prices (
				chain_id, local_chain_id, remote_chain_id, height, height_time,
				local_pool, remote_pool, price_e6
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (chain_id, local_chain_id, remote_chain_id, height) DO UPDATE SET
				local_pool = EXCLUDED.local_pool,
				remote_pool = EXCLUDED.remote_pool,
				price_e6 = EXCLUDED.price_e6
		`,
			chainID,
			p.LocalChainID,
			p.RemoteChainID,
			height,
			blockTime,
			p.LocalPool,
			p.RemotePool,
			p.PriceE6,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range prices {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

// dexEvent holds parsed event data for correlation
type dexEvent struct {
	OrderID      string
	EventType    string
	SoldAmount   uint64
	BoughtAmount uint64
	LocalOrigin  bool
	Success      bool
	// Deposit fields
	PointsReceived uint64
	// Withdrawal fields
	LocalAmount  uint64
	RemoteAmount uint64
	PointsBurned uint64
}

// h1Maps holds H-1 data for change detection
type h1Maps struct {
	OrdersLocked       map[string]*lib.DexLimitOrder
	OrdersPending      map[string]*lib.DexLimitOrder
	DepositsLocked     map[string]*lib.DexLiquidityDeposit
	DepositsPending    map[string]*lib.DexLiquidityDeposit
	WithdrawalsLocked  map[string]*lib.DexLiquidityWithdraw
	WithdrawalsPending map[string]*lib.DexLiquidityWithdraw
}

func (idx *Indexer) indexDexBatch(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}

	// Fetch current batch (locked items)
	currentBatches, err := rpc.AllDexBatchesByHeight(ctx, height)
	if err != nil {
		return err
	}

	// Fetch next batch (pending items)
	nextBatches, err := rpc.AllNextDexBatchesByHeight(ctx, height)
	if err != nil {
		return err
	}

	// Fetch H-1 batches for completion checking and change detection
	var currentBatchesH1, nextBatchesH1 []*lib.DexBatch
	if height > 1 {
		currentBatchesH1, err = rpc.AllDexBatchesByHeight(ctx, height-1)
		if err != nil {
			return err
		}
		nextBatchesH1, err = rpc.AllNextDexBatchesByHeight(ctx, height-1)
		if err != nil {
			return err
		}
	}

	// Query events from database for completion correlation
	swapEvents, depositEvents, withdrawalEvents, err := idx.queryDexEvents(ctx, chainID, height)
	if err != nil {
		return err
	}

	// Build H-1 comparison maps for change detection
	h1 := idx.buildH1Maps(currentBatchesH1, nextBatchesH1)

	// Process DEX orders
	if err := idx.processDexOrders(ctx, chainID, height, blockTime,
		currentBatches, nextBatches, currentBatchesH1, h1, swapEvents); err != nil {
		return err
	}

	// Process DEX deposits
	if err := idx.processDexDeposits(ctx, chainID, height, blockTime,
		currentBatches, nextBatches, currentBatchesH1, h1, depositEvents); err != nil {
		return err
	}

	// Process DEX withdrawals
	return idx.processDexWithdrawals(ctx, chainID, height, blockTime,
		currentBatches, nextBatches, currentBatchesH1, h1, withdrawalEvents)
}

// queryDexEvents queries events table for DEX-related events at height H
func (idx *Indexer) queryDexEvents(ctx context.Context, chainID, height uint64) (
	swapEvents, depositEvents, withdrawalEvents map[string]*dexEvent, err error,
) {
	swapEvents = make(map[string]*dexEvent)
	depositEvents = make(map[string]*dexEvent)
	withdrawalEvents = make(map[string]*dexEvent)

	// Query events from database
	rows, err := idx.db.Query(ctx, `
		SELECT event_type, msg
		FROM events
		WHERE chain_id = $1 AND height = $2
		AND event_type IN ('dex-swap', 'dex-liquidity-deposit', 'dex-liquidity-withdraw')
	`, chainID, height)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var msg []byte
		if err := rows.Scan(&eventType, &msg); err != nil {
			return nil, nil, nil, err
		}

		event, err := idx.parseDexEvent(eventType, msg)
		if err != nil || event == nil || event.OrderID == "" {
			continue
		}

		switch eventType {
		case "dex-swap":
			swapEvents[event.OrderID] = event
		case "dex-liquidity-deposit":
			depositEvents[event.OrderID] = event
		case "dex-liquidity-withdraw":
			withdrawalEvents[event.OrderID] = event
		}
	}

	return swapEvents, depositEvents, withdrawalEvents, rows.Err()
}

// parseDexEvent parses the event message JSON
func (idx *Indexer) parseDexEvent(eventType string, msg []byte) (*dexEvent, error) {
	event := &dexEvent{EventType: eventType}

	// Parse based on event type using lib types
	switch eventType {
	case "dex-swap":
		var e lib.EventDexSwap
		if err := e.UnmarshalJSON(msg); err != nil {
			return nil, err
		}
		event.OrderID = transform.BytesToHex(e.OrderId)
		event.SoldAmount = e.SoldAmount
		event.BoughtAmount = e.BoughtAmount
		event.LocalOrigin = e.LocalOrigin
		event.Success = e.Success

	case "dex-liquidity-deposit":
		var e lib.EventDexLiquidityDeposit
		if err := e.UnmarshalJSON(msg); err != nil {
			return nil, err
		}
		event.OrderID = transform.BytesToHex(e.OrderId)
		event.LocalOrigin = e.LocalOrigin
		event.PointsReceived = e.Points

	case "dex-liquidity-withdraw":
		var e lib.EventDexLiquidityWithdrawal
		if err := e.UnmarshalJSON(msg); err != nil {
			return nil, err
		}
		event.OrderID = transform.BytesToHex(e.OrderId)
		event.LocalAmount = e.LocalAmount
		event.RemoteAmount = e.RemoteAmount
		event.PointsBurned = e.PointsBurned
	}

	return event, nil
}

// buildH1Maps builds lookup maps from H-1 batches for change detection
func (idx *Indexer) buildH1Maps(currentBatchesH1, nextBatchesH1 []*lib.DexBatch) *h1Maps {
	h1 := &h1Maps{
		OrdersLocked:       make(map[string]*lib.DexLimitOrder),
		OrdersPending:      make(map[string]*lib.DexLimitOrder),
		DepositsLocked:     make(map[string]*lib.DexLiquidityDeposit),
		DepositsPending:    make(map[string]*lib.DexLiquidityDeposit),
		WithdrawalsLocked:  make(map[string]*lib.DexLiquidityWithdraw),
		WithdrawalsPending: make(map[string]*lib.DexLiquidityWithdraw),
	}

	// Build maps from current batches at H-1 (locked items)
	for _, batch := range currentBatchesH1 {
		for _, order := range batch.Orders {
			h1.OrdersLocked[transform.BytesToHex(order.OrderId)] = order
		}
		for _, dep := range batch.Deposits {
			h1.DepositsLocked[transform.BytesToHex(dep.OrderId)] = dep
		}
		for _, w := range batch.Withdrawals {
			h1.WithdrawalsLocked[transform.BytesToHex(w.OrderId)] = w
		}
	}

	// Build maps from next batches at H-1 (pending items)
	for _, batch := range nextBatchesH1 {
		for _, order := range batch.Orders {
			h1.OrdersPending[transform.BytesToHex(order.OrderId)] = order
		}
		for _, dep := range batch.Deposits {
			h1.DepositsPending[transform.BytesToHex(dep.OrderId)] = dep
		}
		for _, w := range batch.Withdrawals {
			h1.WithdrawalsPending[transform.BytesToHex(w.OrderId)] = w
		}
	}

	return h1
}

func (idx *Indexer) processDexOrders(ctx context.Context, chainID, height uint64, blockTime time.Time,
	currentBatches, nextBatches, currentBatchesH1 []*lib.DexBatch, h1 *h1Maps, swapEvents map[string]*dexEvent) error {

	batch := &pgx.Batch{}

	// 1. Process COMPLETE orders: items from H-1 batch that have completion events at H
	for _, b := range currentBatchesH1 {
		for _, order := range b.Orders {
			orderID := transform.BytesToHex(order.OrderId)
			if event, exists := swapEvents[orderID]; exists {
				batch.Queue(`
					INSERT INTO dex_orders (
						chain_id, order_id, committee, address, amount_for_sale, requested_amount,
						state, success, sold_amount, bought_amount, local_origin, locked_height, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
					ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
						state = EXCLUDED.state,
						success = EXCLUDED.success,
						sold_amount = EXCLUDED.sold_amount,
						bought_amount = EXCLUDED.bought_amount,
						local_origin = EXCLUDED.local_origin
				`,
					chainID,
					orderID,
					b.Committee,
					transform.BytesToHex(order.Address),
					order.AmountForSale,
					order.RequestedAmount,
					DexStateComplete,
					event.Success,
					event.SoldAmount,
					event.BoughtAmount,
					event.LocalOrigin,
					b.LockedHeight,
					height,
					blockTime,
				)
			}
		}
	}

	// 2. Process LOCKED orders: items in current batch (with change detection)
	for _, b := range currentBatches {
		for _, order := range b.Orders {
			orderID := transform.BytesToHex(order.OrderId)
			address := transform.BytesToHex(order.Address)

			// Check if changed from H-1
			changed := true
			if orderH1, exists := h1.OrdersLocked[orderID]; exists {
				if orderH1.AmountForSale == order.AmountForSale &&
					orderH1.RequestedAmount == order.RequestedAmount &&
					transform.BytesToHex(orderH1.Address) == address {
					changed = false
				}
			}

			if changed {
				batch.Queue(`
					INSERT INTO dex_orders (
						chain_id, order_id, committee, address, amount_for_sale, requested_amount,
						state, success, sold_amount, bought_amount, local_origin, locked_height, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, $7, false, 0, 0, false, $8, $9, $10)
					ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
						state = EXCLUDED.state,
						locked_height = EXCLUDED.locked_height
				`,
					chainID,
					orderID,
					b.Committee,
					address,
					order.AmountForSale,
					order.RequestedAmount,
					DexStateLocked,
					b.LockedHeight,
					height,
					blockTime,
				)
			}
		}
	}

	// 3. Process PENDING orders: items in next batch (with change detection)
	for _, b := range nextBatches {
		for _, order := range b.Orders {
			orderID := transform.BytesToHex(order.OrderId)
			address := transform.BytesToHex(order.Address)

			// Check if changed from H-1
			changed := true
			if orderH1, exists := h1.OrdersPending[orderID]; exists {
				if orderH1.AmountForSale == order.AmountForSale &&
					orderH1.RequestedAmount == order.RequestedAmount &&
					transform.BytesToHex(orderH1.Address) == address {
					changed = false
				}
			}

			if changed {
				batch.Queue(`
					INSERT INTO dex_orders (
						chain_id, order_id, committee, address, amount_for_sale, requested_amount,
						state, success, sold_amount, bought_amount, local_origin, locked_height, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, $7, false, 0, 0, false, 0, $8, $9)
					ON CONFLICT (chain_id, order_id, height) DO NOTHING
				`,
					chainID,
					orderID,
					b.Committee,
					address,
					order.AmountForSale,
					order.RequestedAmount,
					DexStateFuture,
					height,
					blockTime,
				)
			}
		}
	}

	if batch.Len() == 0 {
		return nil
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (idx *Indexer) processDexDeposits(ctx context.Context, chainID, height uint64, blockTime time.Time,
	currentBatches, nextBatches, currentBatchesH1 []*lib.DexBatch, h1 *h1Maps, depositEvents map[string]*dexEvent) error {

	batch := &pgx.Batch{}

	// 1. Process COMPLETE deposits: items from H-1 batch that have completion events at H
	for _, b := range currentBatchesH1 {
		for _, dep := range b.Deposits {
			orderID := transform.BytesToHex(dep.OrderId)
			if event, exists := depositEvents[orderID]; exists {
				batch.Queue(`
					INSERT INTO dex_deposits (
						chain_id, order_id, committee, address, amount, state,
						local_origin, points_received, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
					ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
						state = EXCLUDED.state,
						local_origin = EXCLUDED.local_origin,
						points_received = EXCLUDED.points_received
				`,
					chainID,
					orderID,
					b.Committee,
					transform.BytesToHex(dep.Address),
					dep.Amount,
					DexStateComplete,
					event.LocalOrigin,
					event.PointsReceived,
					height,
					blockTime,
				)
			}
		}
	}

	// 2. Process LOCKED deposits
	for _, b := range currentBatches {
		for _, dep := range b.Deposits {
			orderID := transform.BytesToHex(dep.OrderId)
			address := transform.BytesToHex(dep.Address)

			changed := true
			if depH1, exists := h1.DepositsLocked[orderID]; exists {
				if depH1.Amount == dep.Amount && transform.BytesToHex(depH1.Address) == address {
					changed = false
				}
			}

			if changed {
				batch.Queue(`
					INSERT INTO dex_deposits (
						chain_id, order_id, committee, address, amount, state,
						local_origin, points_received, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, false, 0, $7, $8)
					ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
						state = EXCLUDED.state,
						amount = EXCLUDED.amount
				`,
					chainID,
					orderID,
					b.Committee,
					address,
					dep.Amount,
					DexStateLocked,
					height,
					blockTime,
				)
			}
		}
	}

	// 3. Process PENDING deposits
	for _, b := range nextBatches {
		for _, dep := range b.Deposits {
			orderID := transform.BytesToHex(dep.OrderId)
			address := transform.BytesToHex(dep.Address)

			changed := true
			if depH1, exists := h1.DepositsPending[orderID]; exists {
				if depH1.Amount == dep.Amount && transform.BytesToHex(depH1.Address) == address {
					changed = false
				}
			}

			if changed {
				batch.Queue(`
					INSERT INTO dex_deposits (
						chain_id, order_id, committee, address, amount, state,
						local_origin, points_received, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, false, 0, $7, $8)
					ON CONFLICT (chain_id, order_id, height) DO NOTHING
				`,
					chainID,
					orderID,
					b.Committee,
					address,
					dep.Amount,
					DexStatePending,
					height,
					blockTime,
				)
			}
		}
	}

	if batch.Len() == 0 {
		return nil
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (idx *Indexer) processDexWithdrawals(ctx context.Context, chainID, height uint64, blockTime time.Time,
	currentBatches, nextBatches, currentBatchesH1 []*lib.DexBatch, h1 *h1Maps, withdrawalEvents map[string]*dexEvent) error {

	batch := &pgx.Batch{}

	// 1. Process COMPLETE withdrawals: items from H-1 batch that have completion events at H
	for _, b := range currentBatchesH1 {
		for _, w := range b.Withdrawals {
			orderID := transform.BytesToHex(w.OrderId)
			if event, exists := withdrawalEvents[orderID]; exists {
				batch.Queue(`
					INSERT INTO dex_withdrawals (
						chain_id, order_id, committee, address, percent, state,
						local_amount, remote_amount, points_burned, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
					ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
						state = EXCLUDED.state,
						local_amount = EXCLUDED.local_amount,
						remote_amount = EXCLUDED.remote_amount,
						points_burned = EXCLUDED.points_burned
				`,
					chainID,
					orderID,
					b.Committee,
					transform.BytesToHex(w.Address),
					w.Percent,
					DexStateComplete,
					event.LocalAmount,
					event.RemoteAmount,
					event.PointsBurned,
					height,
					blockTime,
				)
			}
		}
	}

	// 2. Process LOCKED withdrawals
	for _, b := range currentBatches {
		for _, w := range b.Withdrawals {
			orderID := transform.BytesToHex(w.OrderId)
			address := transform.BytesToHex(w.Address)

			changed := true
			if wH1, exists := h1.WithdrawalsLocked[orderID]; exists {
				if wH1.Percent == w.Percent && transform.BytesToHex(wH1.Address) == address {
					changed = false
				}
			}

			if changed {
				batch.Queue(`
					INSERT INTO dex_withdrawals (
						chain_id, order_id, committee, address, percent, state,
						local_amount, remote_amount, points_burned, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, 0, 0, 0, $7, $8)
					ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
						state = EXCLUDED.state,
						percent = EXCLUDED.percent
				`,
					chainID,
					orderID,
					b.Committee,
					address,
					w.Percent,
					DexStateLocked,
					height,
					blockTime,
				)
			}
		}
	}

	// 3. Process PENDING withdrawals
	for _, b := range nextBatches {
		for _, w := range b.Withdrawals {
			orderID := transform.BytesToHex(w.OrderId)
			address := transform.BytesToHex(w.Address)

			changed := true
			if wH1, exists := h1.WithdrawalsPending[orderID]; exists {
				if wH1.Percent == w.Percent && transform.BytesToHex(wH1.Address) == address {
					changed = false
				}
			}

			if changed {
				batch.Queue(`
					INSERT INTO dex_withdrawals (
						chain_id, order_id, committee, address, percent, state,
						local_amount, remote_amount, points_burned, height, height_time
					) VALUES ($1, $2, $3, $4, $5, $6, 0, 0, 0, $7, $8)
					ON CONFLICT (chain_id, order_id, height) DO NOTHING
				`,
					chainID,
					orderID,
					b.Committee,
					address,
					w.Percent,
					DexStatePending,
					height,
					blockTime,
				)
			}
		}
	}

	if batch.Len() == 0 {
		return nil
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

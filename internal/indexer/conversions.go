package indexer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/blob"
	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
	"github.com/canopy-network/canopy-indexer/pkg/transform"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// =============================================================================
// Validator Change Detection Conversions
// =============================================================================

// ConvertValidatorsWithChangeDetection compares validators at height H against H-1
// and returns only changed validators plus their committee assignments.
// This implements the snapshot-on-change pattern to minimize database writes.
// Event correlation detects state transitions even when RPC fields haven't updated yet.
func ConvertValidatorsWithChangeDetection(
	current, previous []*fsm.Validator,
	events []*lib.Event,
	subsidizedCommittees, retiredCommittees []uint64,
	height uint64,
	blockTime time.Time,
) ([]*indexermodels.Validator, []*indexermodels.CommitteeValidator) {
	// Build previous validator map for O(1) lookup
	prevMap := make(map[string]*fsm.Validator, len(previous))
	for _, v := range previous {
		prevMap[transform.BytesToHex(v.Address)] = v
	}

	// Parse lifecycle events for event correlation
	pauseEvents, beginUnstakingEvents, slashEvents, rewardEvents := parseValidatorLifecycleEvents(events)

	// Build subsidized/retired committee maps for O(1) lookup
	subsidizedMap := make(map[uint64]bool, len(subsidizedCommittees))
	for _, id := range subsidizedCommittees {
		subsidizedMap[id] = true
	}
	retiredMap := make(map[uint64]bool, len(retiredCommittees))
	for _, id := range retiredCommittees {
		retiredMap[id] = true
	}

	var validators []*indexermodels.Validator
	var committeeValidators []*indexermodels.CommitteeValidator

	for _, curr := range current {
		addrHex := transform.BytesToHex(curr.Address)
		prev := prevMap[addrHex]

		// Check if changed
		changed := false
		hasEvent := false

		// PHASE 1: Check for lifecycle events (state transitions)
		// Events indicate state changes even if RPC fields haven't updated yet
		if pauseEvents[addrHex] {
			changed = true
			hasEvent = true
		}
		if beginUnstakingEvents[addrHex] {
			changed = true
			hasEvent = true
		}
		if slashEvents[addrHex] {
			changed = true
			hasEvent = true
		}
		if rewardEvents[addrHex] {
			changed = true
			hasEvent = true
		}

		// PHASE 2: Check for new validator
		if prev == nil {
			changed = true
		}

		// PHASE 3: Compare RPC fields (only if no event occurred)
		// If an event occurred, we already marked changed=true above
		if !hasEvent && prev != nil {
			// Compare all fields that affect validator state
			if curr.StakedAmount != prev.StakedAmount ||
				transform.BytesToHex(curr.PublicKey) != transform.BytesToHex(prev.PublicKey) ||
				curr.NetAddress != prev.NetAddress ||
				curr.MaxPausedHeight != prev.MaxPausedHeight ||
				curr.UnstakingHeight != prev.UnstakingHeight ||
				transform.BytesToHex(curr.Output) != transform.BytesToHex(prev.Output) ||
				curr.Delegate != prev.Delegate ||
				curr.Compound != prev.Compound ||
				!equalCommittees(curr.Committees, prev.Committees) {
				changed = true
			}
		}

		if changed {
			// Derive status
			status := "active"
			if curr.UnstakingHeight > 0 {
				status = "unstaking"
			} else if curr.MaxPausedHeight > 0 {
				status = "paused"
			}

			validators = append(validators, &indexermodels.Validator{
				Address:         addrHex,
				PublicKey:       transform.BytesToHex(curr.PublicKey),
				NetAddress:      curr.NetAddress,
				StakedAmount:    curr.StakedAmount,
				MaxPausedHeight: curr.MaxPausedHeight,
				UnstakingHeight: curr.UnstakingHeight,
				Output:          transform.BytesToHex(curr.Output),
				Delegate:        curr.Delegate,
				Compound:        curr.Compound,
				Status:          status,
				Height:          height,
				HeightTime:      blockTime,
			})

			// Create committee_validator entries for changed validators
			for _, committeeID := range curr.Committees {
				committeeValidators = append(committeeValidators, &indexermodels.CommitteeValidator{
					CommitteeID:      committeeID,
					ValidatorAddress: addrHex,
					StakedAmount:     curr.StakedAmount,
					Status:           status,
					Delegate:         curr.Delegate,
					Compound:         curr.Compound,
					Subsidized:       subsidizedMap[committeeID],
					Retired:          retiredMap[committeeID],
					Height:           height,
					HeightTime:       blockTime,
				})
			}
		}
	}

	return validators, committeeValidators
}

// equalCommittees compares two committee slices for equality (order-independent).
func equalCommittees(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[uint64]bool, len(a))
	for _, c := range a {
		aMap[c] = true
	}
	for _, c := range b {
		if !aMap[c] {
			return false
		}
	}
	return true
}

// parseValidatorLifecycleEvents extracts validator lifecycle events and maps them by address.
// Returns maps for O(1) lookup during change detection.
func parseValidatorLifecycleEvents(events []*lib.Event) (
	pauseEvents, beginUnstakingEvents, slashEvents, rewardEvents map[string]bool,
) {
	pauseEvents = make(map[string]bool)
	beginUnstakingEvents = make(map[string]bool)
	slashEvents = make(map[string]bool)
	rewardEvents = make(map[string]bool)

	for _, e := range events {
		// Skip non-validator events
		if e.EventType != string(lib.EventTypeAutoPause) &&
			e.EventType != string(lib.EventTypeAutoBeginUnstaking) &&
			e.EventType != string(lib.EventTypeSlash) &&
			e.EventType != string(lib.EventTypeReward) {
			continue
		}

		// Extract validator address from event
		if len(e.Address) == 0 {
			continue // Skip events without address
		}
		addrHex := transform.BytesToHex(e.Address)

		// Map event to address for O(1) lookup
		switch e.EventType {
		case string(lib.EventTypeAutoPause):
			pauseEvents[addrHex] = true
		case string(lib.EventTypeAutoBeginUnstaking):
			beginUnstakingEvents[addrHex] = true
		case string(lib.EventTypeSlash):
			slashEvents[addrHex] = true
		case string(lib.EventTypeReward):
			rewardEvents[addrHex] = true
		}
	}

	return pauseEvents, beginUnstakingEvents, slashEvents, rewardEvents
}

// =============================================================================
// Non-Signer Change Detection (Two-Phase)
// =============================================================================

// ConvertNonSignersWithChangeDetection implements two-phase change detection:
// Phase 1: Detect new entries and counter changes
// Phase 2: Detect resets (existed at H-1 but not at H)
func ConvertNonSignersWithChangeDetection(
	current, previous []*fsm.NonSigner,
	height uint64,
	blockTime time.Time,
) []*indexermodels.ValidatorNonSigningInfo {
	// Build maps for O(1) lookup
	prevMap := make(map[string]*fsm.NonSigner, len(previous))
	for _, ns := range previous {
		prevMap[transform.BytesToHex(ns.Address)] = ns
	}

	currMap := make(map[string]*fsm.NonSigner, len(current))
	for _, ns := range current {
		currMap[transform.BytesToHex(ns.Address)] = ns
	}

	var results []*indexermodels.ValidatorNonSigningInfo

	// Phase 1: Check current non-signers for new entries and updates
	for _, curr := range current {
		addrHex := transform.BytesToHex(curr.Address)
		prev := prevMap[addrHex]

		changed := false
		if prev == nil {
			// New non-signer entry
			changed = true
		} else if curr.Counter != prev.Counter {
			// Counter changed
			changed = true
		}

		if changed {
			results = append(results, &indexermodels.ValidatorNonSigningInfo{
				Address:           addrHex,
				MissedBlocksCount: curr.Counter,
				LastSignedHeight:  height,
				Height:            height,
				HeightTime:        blockTime,
			})
		}
	}

	// Phase 2: Detect resets - existed at H-1 but not at H
	for _, prev := range previous {
		addrHex := transform.BytesToHex(prev.Address)
		if _, exists := currMap[addrHex]; !exists {
			// Validator was a non-signer previously but not anymore (counter reset)
			results = append(results, &indexermodels.ValidatorNonSigningInfo{
				Address:           addrHex,
				MissedBlocksCount: 0, // Reset to zero
				LastSignedHeight:  height,
				Height:            height,
				HeightTime:        blockTime,
			})
		}
	}

	return results
}

// =============================================================================
// Double-Signer Change Detection (Two-Phase)
// =============================================================================

// ConvertDoubleSignersWithChangeDetection implements two-phase change detection:
// Phase 1: Detect evidence count changes
// Phase 2: Detect clears (existed at H-1 but not at H)
func ConvertDoubleSignersWithChangeDetection(
	current, previous []*lib.DoubleSigner,
	height uint64,
	blockTime time.Time,
) []*indexermodels.ValidatorDoubleSigningInfo {
	// Build maps for O(1) lookup (stores evidence count)
	prevCountMap := make(map[string]uint64, len(previous))
	for _, ds := range previous {
		prevCountMap[transform.BytesToHex(ds.Id)] = uint64(len(ds.Heights))
	}

	currMap := make(map[string]*lib.DoubleSigner, len(current))
	for _, ds := range current {
		currMap[transform.BytesToHex(ds.Id)] = ds
	}

	var results []*indexermodels.ValidatorDoubleSigningInfo

	// Phase 1: Check current double-signers for new entries and updates
	for _, curr := range current {
		addrHex := transform.BytesToHex(curr.Id)
		currentCount := uint64(len(curr.Heights))
		prevCount := prevCountMap[addrHex]

		// Snapshot-on-change: only insert if evidence count changed
		if currentCount != prevCount {
			var firstHeight, lastHeight uint64
			if len(curr.Heights) > 0 {
				firstHeight = curr.Heights[0]
				lastHeight = curr.Heights[len(curr.Heights)-1]
			}

			results = append(results, &indexermodels.ValidatorDoubleSigningInfo{
				Address:             addrHex,
				EvidenceCount:       currentCount,
				FirstEvidenceHeight: firstHeight,
				LastEvidenceHeight:  lastHeight,
				Height:              height,
				HeightTime:          blockTime,
			})
		}
	}

	// Phase 2: Detect clears - existed at H-1 but not at H
	for _, prev := range previous {
		addrHex := transform.BytesToHex(prev.Id)
		if _, exists := currMap[addrHex]; !exists {
			// Evidence was cleared/reset
			results = append(results, &indexermodels.ValidatorDoubleSigningInfo{
				Address:             addrHex,
				EvidenceCount:       0,
				FirstEvidenceHeight: 0,
				LastEvidenceHeight:  0,
				Height:              height,
				HeightTime:          blockTime,
			})
		}
	}

	return results
}

// =============================================================================
// Pool Points Change Detection
// =============================================================================

// ConvertPoolPointsWithChangeDetection implements snapshot-on-change for pool points holders.
// Writes if: new holder, points changed, or pool's TotalPoolPoints changed.
func ConvertPoolPointsWithChangeDetection(
	currentPools, previousPools []*fsm.Pool,
	height uint64,
	blockTime time.Time,
) []*indexermodels.PoolPointsByHolder {
	// Build previous state map for change detection
	type prevHolder struct {
		points          uint64
		totalPoolPoints uint64
	}
	prevMap := make(map[string]prevHolder) // key: "poolID:address"

	for _, p := range previousPools {
		for _, pp := range p.Points {
			key := poolHolderKey(p.Id, transform.BytesToHex(pp.Address))
			prevMap[key] = prevHolder{
				points:          pp.Points,
				totalPoolPoints: p.TotalPoolPoints,
			}
		}
	}

	var results []*indexermodels.PoolPointsByHolder

	for _, pool := range currentPools {
		committee := uint16(transform.ExtractChainIDFromPoolID(pool.Id))

		for _, pp := range pool.Points {
			addrHex := transform.BytesToHex(pp.Address)
			key := poolHolderKey(pool.Id, addrHex)
			prev, existed := prevMap[key]

			// Snapshot if: new holder, points changed, or pool's TotalPoolPoints changed
			shouldSnapshot := !existed ||
				pp.Points != prev.points ||
				pool.TotalPoolPoints != prev.totalPoolPoints

			if shouldSnapshot {
				results = append(results, &indexermodels.PoolPointsByHolder{
					Address:             addrHex,
					PoolID:              uint32(pool.Id),
					Height:              height,
					HeightTime:          blockTime,
					Committee:           committee,
					Points:              pp.Points,
					LiquidityPoolPoints: pool.TotalPoolPoints,
					LiquidityPoolID:     uint32(transform.LiquidityPoolAddend + uint64(committee)),
					PoolAmount:          pool.Amount,
				})
			}
		}
	}

	return results
}

// =============================================================================
// DEX State Machine Conversions
// =============================================================================

// dexEvent holds parsed event data for correlation
type dexEvent struct {
	OrderID        string
	EventType      string
	SoldAmount     uint64
	BoughtAmount   uint64
	LocalOrigin    bool
	Success        bool
	PointsReceived uint64
	LocalAmount    uint64
	RemoteAmount   uint64
	PointsBurned   uint64
}

// h1Maps holds H-1 data for DEX change detection
type h1Maps struct {
	OrdersLocked       map[string]*lib.DexLimitOrder
	OrdersPending      map[string]*lib.DexLimitOrder
	DepositsLocked     map[string]*lib.DexLiquidityDeposit
	DepositsPending    map[string]*lib.DexLiquidityDeposit
	WithdrawalsLocked  map[string]*lib.DexLiquidityWithdraw
	WithdrawalsPending map[string]*lib.DexLiquidityWithdraw
}

// ConvertDexOrdersWithStateMachine converts DEX orders using the state machine pattern.
// States: FUTURE -> LOCKED -> COMPLETE
func ConvertDexOrdersWithStateMachine(data *blob.BlockData) []*indexermodels.DexOrder {
	// Parse DEX events for completion detection
	swapEvents, _, _ := parseDexEventsFromSlice(data.Events)

	// Build H-1 comparison maps
	h1 := buildH1Maps(data.DexBatchesPreviousCurr, data.DexBatchesPreviousNext)

	var results []*indexermodels.DexOrder

	// 1. COMPLETE orders: items from H-1 batch that have completion events at H
	for _, b := range data.DexBatchesPreviousCurr {
		for _, order := range b.Orders {
			orderID := transform.BytesToHex(order.OrderId)
			if event, exists := swapEvents[orderID]; exists {
				results = append(results, &indexermodels.DexOrder{
					OrderID:         orderID,
					Height:          data.Height,
					HeightTime:      data.BlockTime,
					Committee:       uint16(b.Committee),
					Address:         transform.BytesToHex(order.Address),
					AmountForSale:   order.AmountForSale,
					RequestedAmount: order.RequestedAmount,
					State:           indexermodels.DexCompleteState,
					Success:         event.Success,
					SoldAmount:      event.SoldAmount,
					BoughtAmount:    event.BoughtAmount,
					LocalOrigin:     event.LocalOrigin,
					LockedHeight:    b.LockedHeight,
				})
			}
		}
	}

	// 2. LOCKED orders: items in current batch (with change detection)
	for _, b := range data.DexBatchesCurrent {
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
				results = append(results, &indexermodels.DexOrder{
					OrderID:         orderID,
					Height:          data.Height,
					HeightTime:      data.BlockTime,
					Committee:       uint16(b.Committee),
					Address:         address,
					AmountForSale:   order.AmountForSale,
					RequestedAmount: order.RequestedAmount,
					State:           indexermodels.DexLockedState,
					Success:         false,
					SoldAmount:      0,
					BoughtAmount:    0,
					LocalOrigin:     false,
					LockedHeight:    b.LockedHeight,
				})
			}
		}
	}

	// 3. FUTURE orders: items in next batch (with change detection)
	for _, b := range data.DexBatchesNext {
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
				results = append(results, &indexermodels.DexOrder{
					OrderID:         orderID,
					Height:          data.Height,
					HeightTime:      data.BlockTime,
					Committee:       uint16(b.Committee),
					Address:         address,
					AmountForSale:   order.AmountForSale,
					RequestedAmount: order.RequestedAmount,
					State:           indexermodels.DexPendingState, // "future" state uses pending constant
					Success:         false,
					SoldAmount:      0,
					BoughtAmount:    0,
					LocalOrigin:     false,
					LockedHeight:    0,
				})
			}
		}
	}

	return results
}

// ConvertDexDepositsWithStateMachine converts DEX deposits using the state machine pattern.
// States: PENDING -> LOCKED -> COMPLETE
func ConvertDexDepositsWithStateMachine(data *blob.BlockData) []*indexermodels.DexDeposit {
	// Parse DEX events for completion detection
	_, depositEvents, _ := parseDexEventsFromSlice(data.Events)

	// Build H-1 comparison maps
	h1 := buildH1Maps(data.DexBatchesPreviousCurr, data.DexBatchesPreviousNext)

	var results []*indexermodels.DexDeposit

	// 1. COMPLETE deposits: items from H-1 batch that have completion events at H
	for _, b := range data.DexBatchesPreviousCurr {
		for _, dep := range b.Deposits {
			orderID := transform.BytesToHex(dep.OrderId)
			if event, exists := depositEvents[orderID]; exists {
				results = append(results, &indexermodels.DexDeposit{
					OrderID:        orderID,
					Height:         data.Height,
					HeightTime:     data.BlockTime,
					Committee:      uint16(b.Committee),
					Address:        transform.BytesToHex(dep.Address),
					Amount:         dep.Amount,
					State:          indexermodels.DexCompleteState,
					LocalOrigin:    event.LocalOrigin,
					PointsReceived: event.PointsReceived,
				})
			}
		}
	}

	// 2. LOCKED deposits
	for _, b := range data.DexBatchesCurrent {
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
				results = append(results, &indexermodels.DexDeposit{
					OrderID:        orderID,
					Height:         data.Height,
					HeightTime:     data.BlockTime,
					Committee:      uint16(b.Committee),
					Address:        address,
					Amount:         dep.Amount,
					State:          indexermodels.DexLockedState,
					LocalOrigin:    false,
					PointsReceived: 0,
				})
			}
		}
	}

	// 3. PENDING deposits
	for _, b := range data.DexBatchesNext {
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
				results = append(results, &indexermodels.DexDeposit{
					OrderID:        orderID,
					Height:         data.Height,
					HeightTime:     data.BlockTime,
					Committee:      uint16(b.Committee),
					Address:        address,
					Amount:         dep.Amount,
					State:          indexermodels.DexPendingState,
					LocalOrigin:    false,
					PointsReceived: 0,
				})
			}
		}
	}

	return results
}

// ConvertDexWithdrawalsWithStateMachine converts DEX withdrawals using the state machine pattern.
// States: PENDING -> LOCKED -> COMPLETE
func ConvertDexWithdrawalsWithStateMachine(data *blob.BlockData) []*indexermodels.DexWithdrawal {
	// Parse DEX events for completion detection
	_, _, withdrawalEvents := parseDexEventsFromSlice(data.Events)

	// Build H-1 comparison maps
	h1 := buildH1Maps(data.DexBatchesPreviousCurr, data.DexBatchesPreviousNext)

	var results []*indexermodels.DexWithdrawal

	// 1. COMPLETE withdrawals: items from H-1 batch that have completion events at H
	for _, b := range data.DexBatchesPreviousCurr {
		for _, w := range b.Withdrawals {
			orderID := transform.BytesToHex(w.OrderId)
			if event, exists := withdrawalEvents[orderID]; exists {
				results = append(results, &indexermodels.DexWithdrawal{
					OrderID:      orderID,
					Height:       data.Height,
					HeightTime:   data.BlockTime,
					Committee:    uint16(b.Committee),
					Address:      transform.BytesToHex(w.Address),
					Percent:      w.Percent,
					State:        indexermodels.DexCompleteState,
					LocalAmount:  event.LocalAmount,
					RemoteAmount: event.RemoteAmount,
					PointsBurned: event.PointsBurned,
				})
			}
		}
	}

	// 2. LOCKED withdrawals
	for _, b := range data.DexBatchesCurrent {
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
				results = append(results, &indexermodels.DexWithdrawal{
					OrderID:      orderID,
					Height:       data.Height,
					HeightTime:   data.BlockTime,
					Committee:    uint16(b.Committee),
					Address:      address,
					Percent:      w.Percent,
					State:        indexermodels.DexLockedState,
					LocalAmount:  0,
					RemoteAmount: 0,
					PointsBurned: 0,
				})
			}
		}
	}

	// 3. PENDING withdrawals
	for _, b := range data.DexBatchesNext {
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
				results = append(results, &indexermodels.DexWithdrawal{
					OrderID:      orderID,
					Height:       data.Height,
					HeightTime:   data.BlockTime,
					Committee:    uint16(b.Committee),
					Address:      address,
					Percent:      w.Percent,
					State:        indexermodels.DexPendingState,
					LocalAmount:  0,
					RemoteAmount: 0,
					PointsBurned: 0,
				})
			}
		}
	}

	return results
}

// =============================================================================
// DEX Helper Functions
// =============================================================================

// parseDexEventsFromSlice parses DEX events from the in-memory Events slice.
func parseDexEventsFromSlice(events []*lib.Event) (
	swapEvents, depositEvents, withdrawalEvents map[string]*dexEvent,
) {
	swapEvents = make(map[string]*dexEvent)
	depositEvents = make(map[string]*dexEvent)
	withdrawalEvents = make(map[string]*dexEvent)

	for _, e := range events {
		if e.EventType != "dex-swap" &&
			e.EventType != "dex-liquidity-deposit" &&
			e.EventType != "dex-liquidity-withdraw" {
			continue
		}

		// Marshal event message to JSON for parsing
		msgJSON, err := json.Marshal(e.Msg)
		if err != nil {
			continue
		}

		event, err := parseDexEvent(e.EventType, msgJSON)
		if err != nil || event == nil || event.OrderID == "" {
			continue
		}

		switch e.EventType {
		case "dex-swap":
			swapEvents[event.OrderID] = event
		case "dex-liquidity-deposit":
			depositEvents[event.OrderID] = event
		case "dex-liquidity-withdraw":
			withdrawalEvents[event.OrderID] = event
		}
	}

	return swapEvents, depositEvents, withdrawalEvents
}

// parseDexEvent parses the event message JSON
func parseDexEvent(eventType string, msg []byte) (*dexEvent, error) {
	event := &dexEvent{EventType: eventType}

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
func buildH1Maps(currentBatchesH1, nextBatchesH1 []*lib.DexBatch) *h1Maps {
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

// poolHolderKey creates a unique key for a pool holder.
func poolHolderKey(poolID uint64, address string) string {
	return strconv.FormatUint(poolID, 10) + ":" + address
}

// =============================================================================
// Simple Conversions (Direct mappings without change detection)
// =============================================================================

// convertBlock converts lib.BlockResult to indexermodels.Block
func convertBlock(block *lib.BlockResult, chainID uint64, blockTime time.Time) *indexermodels.Block {
	h := block.BlockHeader
	return &indexermodels.Block{
		Height:             h.Height,
		Hash:               transform.BytesToHex(h.Hash),
		Time:               blockTime,
		NetworkID:          uint32(chainID),
		LastBlockHash:      transform.BytesToHex(h.LastBlockHash),
		ProposerAddress:    transform.BytesToHex(h.ProposerAddress),
		Size:               int32(block.Meta.Size),
		NumTxs:             h.NumTxs,
		TotalTxs:           h.TotalTxs,
		TotalVDFIterations: int32(h.TotalVdfIterations),
		StateRoot:          transform.BytesToHex(h.StateRoot),
		TransactionRoot:    transform.BytesToHex(h.TransactionRoot),
		ValidatorRoot:      transform.BytesToHex(h.ValidatorRoot),
		NextValidatorRoot:  transform.BytesToHex(h.NextValidatorRoot),
	}
}

// convertTransactions converts []*lib.TxResult to []*indexermodels.Transaction
func convertTransactions(txs []*lib.TxResult, chainID, height uint64, blockTime time.Time) []*indexermodels.Transaction {
	result := make([]*indexermodels.Transaction, len(txs))
	for i, tx := range txs {
		result[i] = &indexermodels.Transaction{
			Height:      height,
			HeightTime:  blockTime,
			MessageType: tx.MessageType,
			// Add other fields as needed
		}
	}
	return result
}

// convertEvents converts []*lib.Event to []*indexermodels.Event
func convertEvents(events []*lib.Event, chainID, height uint64, blockTime time.Time) []*indexermodels.Event {
	result := make([]*indexermodels.Event, len(events))
	for i, event := range events {
		result[i] = &indexermodels.Event{
			Height:     height,
			HeightTime: blockTime,
			EventType:  event.EventType,
			// Add other fields
		}
	}
	return result
}

// convertAccounts converts []*fsm.Account to []*indexermodels.Account
func convertAccounts(accounts []*fsm.Account, height uint64, blockTime time.Time) []*indexermodels.Account {
	result := make([]*indexermodels.Account, len(accounts))
	for i, acc := range accounts {
		result[i] = &indexermodels.Account{
			Address:    transform.BytesToHex(acc.Address),
			Amount:     acc.Amount,
			Rewards:    0, // TODO: aggregate from reward events
			Slashes:    0, // TODO: aggregate from slash events
			Height:     height,
			HeightTime: blockTime,
		}
	}
	return result
}

// ConvertAccountsWithChangeDetection compares accounts at height H against H-1
// and returns only changed accounts. This implements snapshot-on-change to minimize writes.
func ConvertAccountsWithChangeDetection(
	current, previous []*fsm.Account,
	height uint64,
	blockTime time.Time,
) []*indexermodels.Account {
	// Build previous account map for O(1) lookup
	prevMap := make(map[string]*fsm.Account, len(previous))
	for _, acc := range previous {
		prevMap[transform.BytesToHex(acc.Address)] = acc
	}

	var results []*indexermodels.Account

	for _, curr := range current {
		addrHex := transform.BytesToHex(curr.Address)
		prev := prevMap[addrHex]

		// Check if changed
		changed := false

		// New account
		if prev == nil {
			changed = true
		} else {
			// Compare amount (rewards and slashes are TODO)
			if curr.Amount != prev.Amount {
				changed = true
			}
		}

		if changed {
			results = append(results, &indexermodels.Account{
				Address:    addrHex,
				Amount:     curr.Amount,
				Rewards:    0, // TODO: aggregate from reward events
				Slashes:    0, // TODO: aggregate from slash events
				Height:     height,
				HeightTime: blockTime,
			})
		}
	}

	return results
}

// convertPools converts []*fsm.Pool to []*indexermodels.Pool with delta calculations
func convertPools(current []*fsm.Pool, previous []*fsm.Pool, height uint64, blockTime time.Time) []*indexermodels.Pool {
	// Build previous pools map for O(1) delta calculation
	prevMap := make(map[uint64]*fsm.Pool, len(previous))
	for _, p := range previous {
		prevMap[p.Id] = p
	}

	var result []*indexermodels.Pool

	for _, pool := range current {
		poolModel := &indexermodels.Pool{
			PoolID:      uint32(pool.Id),
			Height:      height,
			HeightTime:  blockTime,
			Amount:      pool.Amount,
			TotalPoints: pool.TotalPoolPoints,
		}

		// Calculate H-1 deltas by comparing with previous pool state
		changed := false
		if prevPool, exists := prevMap[pool.Id]; exists {
			// Pool existed at H-1, calculate deltas
			poolModel.AmountDelta = int64(pool.Amount) - int64(prevPool.Amount)
			poolModel.TotalPointsDelta = int64(pool.TotalPoolPoints) - int64(prevPool.TotalPoolPoints)

			// Calculate LP count delta (need to count Points array)
			currentLPCount := uint16(len(pool.Points))
			prevLPCount := uint16(len(prevPool.Points))
			poolModel.LPCountDelta = int16(currentLPCount) - int16(prevLPCount)
			poolModel.LPCount = currentLPCount

			// Changed if any delta != 0
			if poolModel.AmountDelta != 0 || poolModel.TotalPointsDelta != 0 || poolModel.LPCountDelta != 0 {
				changed = true
			}
		} else {
			// New pool at height H
			poolModel.AmountDelta = 0
			poolModel.TotalPointsDelta = 0
			poolModel.LPCountDelta = 0
			poolModel.LPCount = uint16(len(pool.Points))
			changed = true
		}

		if changed {
			// Calculate derived pool IDs (committee-based addressing)
			chainID := transform.ExtractChainIDFromPoolID(pool.Id)
			poolModel.ChainID = uint16(chainID)
			poolModel.LiquidityPoolID = uint32(transform.LiquidityPoolAddend + chainID)
			result = append(result, poolModel)
		}
	}

	return result
}

// determineOrderStatus determines the order status based on order state.
// Orders with buyer addresses set are "locked", otherwise "open".
// Note: "complete" and "canceled" statuses are determined by order removal from state.
func determineOrderStatus(order *lib.SellOrder) string {
	// If buyer addresses are set, the order is locked
	if len(order.BuyerSendAddress) > 0 && len(order.BuyerReceiveAddress) > 0 {
		return "locked"
	}
	return "open"
}

// convertOrders converts []*lib.SellOrder to []*indexermodels.Order
func convertOrders(orders []*lib.SellOrder, height uint64, blockTime time.Time) []*indexermodels.Order {
	result := make([]*indexermodels.Order, len(orders))
	for i, order := range orders {
		result[i] = &indexermodels.Order{
			Height:          height,
			HeightTime:      blockTime,
			AmountForSale:   order.AmountForSale,
			RequestedAmount: order.RequestedAmount,
			// Add other fields
		}
	}
	return result
}

// ConvertOrdersWithChangeDetection compares orders at height H against H-1
// and returns only changed orders. This implements snapshot-on-change to minimize writes.
func ConvertOrdersWithChangeDetection(
	current, previous []*lib.SellOrder,
	height uint64,
	blockTime time.Time,
) []*indexermodels.Order {
	// Build previous order map for O(1) lookup (key by Id)
	prevMap := make(map[string]*lib.SellOrder, len(previous))
	for _, order := range previous {
		prevMap[transform.BytesToHex(order.Id)] = order
	}

	var results []*indexermodels.Order

	for _, curr := range current {
		idHex := transform.BytesToHex(curr.Id)
		prev := prevMap[idHex]

		// Check if changed
		changed := false

		// New order
		if prev == nil {
			changed = true
		} else {
			// Compare key fields that can change
			if curr.AmountForSale != prev.AmountForSale ||
				curr.RequestedAmount != prev.RequestedAmount ||
				curr.BuyerChainDeadline != prev.BuyerChainDeadline ||
				transform.BytesToHex(curr.BuyerSendAddress) != transform.BytesToHex(prev.BuyerSendAddress) ||
				transform.BytesToHex(curr.BuyerReceiveAddress) != transform.BytesToHex(prev.BuyerReceiveAddress) {
				changed = true
			}
		}

		if changed {
			results = append(results, &indexermodels.Order{
				OrderID:              idHex,
				Committee:            uint16(curr.Committee),
				Data:                 transform.BytesToHex(curr.Data),
				AmountForSale:        curr.AmountForSale,
				RequestedAmount:      curr.RequestedAmount,
				SellerReceiveAddress: transform.BytesToHex(curr.SellerReceiveAddress),
				BuyerSendAddress:     transform.BytesToHex(curr.BuyerSendAddress),
				BuyerReceiveAddress:  transform.BytesToHex(curr.BuyerReceiveAddress),
				BuyerChainDeadline:   curr.BuyerChainDeadline,
				SellersSendAddress:   transform.BytesToHex(curr.SellersSendAddress),
				Status:               determineOrderStatus(curr),
				Height:               height,
				HeightTime:           blockTime,
			})
		}
	}

	return results
}

// convertDexPrices converts []*lib.DexPrice to []*indexermodels.DexPrice
func convertDexPrices(prices []*lib.DexPrice, height uint64, blockTime time.Time) []*indexermodels.DexPrice {
	result := make([]*indexermodels.DexPrice, len(prices))
	for i := range prices {
		result[i] = &indexermodels.DexPrice{
			Height:     height,
			HeightTime: blockTime,
			// Add other fields like pair
		}
	}
	return result
}

// ConvertDexPricesWithChangeDetection compares DEX prices at height H against H-1
// and returns only changed prices. This implements snapshot-on-change to minimize writes.
func ConvertDexPricesWithChangeDetection(
	current, previous []*lib.DexPrice,
	height uint64,
	blockTime time.Time,
) []*indexermodels.DexPrice {
	// Build previous price map for O(1) lookup (key by "local-remote")
	prevMap := make(map[string]*lib.DexPrice, len(previous))
	for _, price := range previous {
		key := fmt.Sprintf("%d-%d", price.LocalChainId, price.RemoteChainId)
		prevMap[key] = price
	}

	var results []*indexermodels.DexPrice

	for _, curr := range current {
		key := fmt.Sprintf("%d-%d", curr.LocalChainId, curr.RemoteChainId)
		prev := prevMap[key]

		// Check if changed
		changed := false

		// New price
		if prev == nil {
			changed = true
		} else {
			// Compare price and pool amounts
			if curr.E6ScaledPrice != prev.E6ScaledPrice ||
				curr.LocalPool != prev.LocalPool ||
				curr.RemotePool != prev.RemotePool {
				changed = true
			}
		}

		if changed {
			// Calculate deltas
			var priceDelta, localPoolDelta, remotePoolDelta int64
			if prev != nil {
				priceDelta = int64(curr.E6ScaledPrice) - int64(prev.E6ScaledPrice)
				localPoolDelta = int64(curr.LocalPool) - int64(prev.LocalPool)
				remotePoolDelta = int64(curr.RemotePool) - int64(prev.RemotePool)
			}

			results = append(results, &indexermodels.DexPrice{
				LocalChainID:    uint16(curr.LocalChainId),
				RemoteChainID:   uint16(curr.RemoteChainId),
				LocalPool:       curr.LocalPool,
				RemotePool:      curr.RemotePool,
				PriceE6:         curr.E6ScaledPrice,
				PriceDelta:      priceDelta,
				LocalPoolDelta:  localPoolDelta,
				RemotePoolDelta: remotePoolDelta,
				Height:          height,
				HeightTime:      blockTime,
			})
		}
	}

	return results
}

// convertParams converts *fsm.Params to *indexermodels.Params
func convertParams(params *fsm.Params, height uint64, blockTime time.Time) *indexermodels.Params {
	if params == nil {
		return nil
	}
	return &indexermodels.Params{
		Height:     height,
		HeightTime: blockTime,
		// Add param fields
	}
}

// ConvertParamsWithChangeDetection compares params at height H against H-1
// and returns params only if changed. Since params rarely change, this minimizes writes.
func ConvertParamsWithChangeDetection(
	current, previous *fsm.Params,
	height uint64,
	blockTime time.Time,
) *indexermodels.Params {
	if current == nil {
		return nil
	}

	// Convert current params using transform helper
	currentConverted := transform.ParamsFromFSM(current)
	if currentConverted == nil {
		return nil
	}

	// If no previous, always snapshot
	if previous == nil {
		result := fsmParamsToIndexerParams(currentConverted, height, blockTime)
		return result
	}

	// Compare with previous params
	previousConverted := transform.ParamsFromFSM(previous)
	if previousConverted == nil || paramsChanged(currentConverted, previousConverted) {
		return fsmParamsToIndexerParams(currentConverted, height, blockTime)
	}

	// No change
	return nil
}

// fsmParamsToIndexerParams converts transform.Params to indexermodels.Params
func fsmParamsToIndexerParams(p *transform.Params, height uint64, blockTime time.Time) *indexermodels.Params {
	return &indexermodels.Params{
		Height:                       height,
		HeightTime:                   blockTime,
		BlockSize:                    p.BlockSize,
		ProtocolVersion:              p.ProtocolVersion,
		RootChainID:                  uint16(p.RootChainID),
		Retired:                      0, // Retired status handled separately
		UnstakingBlocks:              p.UnstakingBlocks,
		MaxPauseBlocks:               p.MaxPauseBlocks,
		DoubleSignSlashPercentage:    p.DoubleSignSlashPercentage,
		NonSignSlashPercentage:       p.NonSignSlashPercentage,
		MaxNonSign:                   p.MaxNonSign,
		NonSignWindow:                p.NonSignWindow,
		MaxCommittees:                p.MaxCommittees,
		MaxCommitteeSize:             p.MaxCommitteeSize,
		EarlyWithdrawalPenalty:       p.EarlyWithdrawalPenalty,
		DelegateUnstakingBlocks:      p.DelegateUnstakingBlocks,
		MinimumOrderSize:             p.MinimumOrderSize,
		StakePercentForSubsidized:    p.StakePercentForSubsidizedCommittee,
		MaxSlashPerCommittee:         p.MaxSlashPerCommittee,
		DelegateRewardPercentage:     p.DelegateRewardPercentage,
		BuyDeadlineBlocks:            p.BuyDeadlineBlocks,
		LockOrderFeeMultiplier:       p.LockOrderFeeMultiplier,
		MinimumStakeForValidators:    p.MinimumStakeForValidators,
		MinimumStakeForDelegates:     p.MinimumStakeForDelegates,
		MaximumDelegatesPerCommittee: p.MaximumDelegatesPerCommittee,
		SendFee:                      p.SendFee,
		StakeFee:                     p.StakeFee,
		EditStakeFee:                 p.EditStakeFee,
		UnstakeFee:                   p.UnstakeFee,
		PauseFee:                     p.PauseFee,
		UnpauseFee:                   p.UnpauseFee,
		ChangeParameterFee:           p.ChangeParameterFee,
		DaoTransferFee:               p.DaoTransferFee,
		CertificateResultsFee:        p.CertificateResultsFee,
		SubsidyFee:                   p.SubsidyFee,
		CreateOrderFee:               p.CreateOrderFee,
		EditOrderFee:                 p.EditOrderFee,
		DeleteOrderFee:               p.DeleteOrderFee,
		DexLimitOrderFee:             p.DexLimitOrderFee,
		DexLiquidityDepositFee:       p.DexLiquidityDepositFee,
		DexLiquidityWithdrawFee:      p.DexLiquidityWithdrawFee,
		DaoRewardPercentage:          p.DaoRewardPercentage,
	}
}

// paramsChanged compares two Params structs and returns true if any field differs
func paramsChanged(current, previous *transform.Params) bool {
	// Compare consensus params
	if current.BlockSize != previous.BlockSize ||
		current.ProtocolVersion != previous.ProtocolVersion ||
		current.RootChainID != previous.RootChainID {
		return true
	}

	// Compare validator params
	if current.UnstakingBlocks != previous.UnstakingBlocks ||
		current.MaxPauseBlocks != previous.MaxPauseBlocks ||
		current.DoubleSignSlashPercentage != previous.DoubleSignSlashPercentage ||
		current.NonSignSlashPercentage != previous.NonSignSlashPercentage ||
		current.MaxNonSign != previous.MaxNonSign ||
		current.NonSignWindow != previous.NonSignWindow ||
		current.MaxCommittees != previous.MaxCommittees ||
		current.MaxCommitteeSize != previous.MaxCommitteeSize ||
		current.EarlyWithdrawalPenalty != previous.EarlyWithdrawalPenalty ||
		current.DelegateUnstakingBlocks != previous.DelegateUnstakingBlocks ||
		current.MinimumOrderSize != previous.MinimumOrderSize ||
		current.StakePercentForSubsidizedCommittee != previous.StakePercentForSubsidizedCommittee ||
		current.MaxSlashPerCommittee != previous.MaxSlashPerCommittee ||
		current.DelegateRewardPercentage != previous.DelegateRewardPercentage ||
		current.BuyDeadlineBlocks != previous.BuyDeadlineBlocks ||
		current.LockOrderFeeMultiplier != previous.LockOrderFeeMultiplier ||
		current.MinimumStakeForValidators != previous.MinimumStakeForValidators ||
		current.MinimumStakeForDelegates != previous.MinimumStakeForDelegates ||
		current.MaximumDelegatesPerCommittee != previous.MaximumDelegatesPerCommittee {
		return true
	}

	// Compare fee params
	if current.SendFee != previous.SendFee ||
		current.StakeFee != previous.StakeFee ||
		current.EditStakeFee != previous.EditStakeFee ||
		current.UnstakeFee != previous.UnstakeFee ||
		current.PauseFee != previous.PauseFee ||
		current.UnpauseFee != previous.UnpauseFee ||
		current.ChangeParameterFee != previous.ChangeParameterFee ||
		current.DaoTransferFee != previous.DaoTransferFee ||
		current.CertificateResultsFee != previous.CertificateResultsFee ||
		current.SubsidyFee != previous.SubsidyFee ||
		current.CreateOrderFee != previous.CreateOrderFee ||
		current.EditOrderFee != previous.EditOrderFee ||
		current.DeleteOrderFee != previous.DeleteOrderFee ||
		current.DexLimitOrderFee != previous.DexLimitOrderFee ||
		current.DexLiquidityDepositFee != previous.DexLiquidityDepositFee ||
		current.DexLiquidityWithdrawFee != previous.DexLiquidityWithdrawFee {
		return true
	}

	// Compare governance params
	if current.DaoRewardPercentage != previous.DaoRewardPercentage {
		return true
	}

	return false
}

// convertSupply converts *fsm.Supply to *indexermodels.Supply
func convertSupply(supply *fsm.Supply, height uint64, blockTime time.Time) *indexermodels.Supply {
	if supply == nil {
		return nil
	}
	return &indexermodels.Supply{
		Height:        height,
		HeightTime:    blockTime,
		Total:         supply.Total,
		Staked:        supply.Staked,
		DelegatedOnly: supply.DelegatedOnly,
	}
}

// ConvertSupplyWithChangeDetection compares supply at height H against H-1
// and returns supply only if changed. Since supply changes with staking activity, this minimizes writes.
func ConvertSupplyWithChangeDetection(
	current, previous *fsm.Supply,
	height uint64,
	blockTime time.Time,
) *indexermodels.Supply {
	if current == nil {
		return nil
	}

	// If no previous, always snapshot
	if previous == nil {
		return &indexermodels.Supply{
			Height:        height,
			HeightTime:    blockTime,
			Total:         current.Total,
			Staked:        current.Staked,
			DelegatedOnly: current.DelegatedOnly,
		}
	}

	// Compare supply fields
	if current.Total != previous.Total ||
		current.Staked != previous.Staked ||
		current.DelegatedOnly != previous.DelegatedOnly {
		return &indexermodels.Supply{
			Height:        height,
			HeightTime:    blockTime,
			Total:         current.Total,
			Staked:        current.Staked,
			DelegatedOnly: current.DelegatedOnly,
		}
	}

	// No change
	return nil
}

// convertCommittees converts []*lib.CommitteeData to []*indexermodels.Committee
func convertCommittees(committees []*lib.CommitteeData, height uint64, blockTime time.Time) []*indexermodels.Committee {
	result := make([]*indexermodels.Committee, len(committees))
	for i := range committees {
		result[i] = &indexermodels.Committee{
			Height:     height,
			HeightTime: blockTime,
			// Add other fields
		}
	}
	return result
}

// ConvertCommitteesWithChangeDetection compares committees at height H against H-1
// and returns only changed committees. This implements snapshot-on-change to minimize writes.
func ConvertCommitteesWithChangeDetection(
	current, previous []*lib.CommitteeData,
	height uint64,
	blockTime time.Time,
) []*indexermodels.Committee {
	// Build previous committee map for O(1) lookup (key by ChainId)
	prevMap := make(map[uint64]*lib.CommitteeData, len(previous))
	for _, committee := range previous {
		prevMap[committee.ChainId] = committee
	}

	var results []*indexermodels.Committee

	for _, curr := range current {
		prev := prevMap[curr.ChainId]

		// Check if changed
		changed := false

		// New committee
		if prev == nil {
			changed = true
		} else {
			// Compare key fields
			if curr.LastRootHeightUpdated != prev.LastRootHeightUpdated ||
				curr.LastChainHeightUpdated != prev.LastChainHeightUpdated ||
				curr.NumberOfSamples != prev.NumberOfSamples {
				changed = true
			}
			// TODO: Compare PaymentPercents if needed
		}

		if changed {
			results = append(results, &indexermodels.Committee{
				ChainID:                uint16(curr.ChainId),
				LastRootHeightUpdated:  curr.LastRootHeightUpdated,
				LastChainHeightUpdated: curr.LastChainHeightUpdated,
				NumberOfSamples:        curr.NumberOfSamples,
				Subsidized:             0, // Aggregate computed at block summary level
				Retired:                0, // Aggregate computed at block summary level
				Height:                 height,
				HeightTime:             blockTime,
			})
		}
	}

	return results
}

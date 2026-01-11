package indexer

import (
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
)

const DexOrdersProductionTableName = "dex_orders"
const DexOrdersStagingTableName = DexOrdersProductionTableName + entities.StagingSuffix

// State is used not only by order but also by Deposit and Withdrawal.
const (
	DexPendingState  string = "pending"
	DexLockedState   string = "locked"
	DexCompleteState string = "complete"
)

// DexOrderColumns defines the schema for the dex_orders table.
var DexOrderColumns = []ColumnDef{
	{Name: "order_id"},
	{Name: "height"},
	{Name: "height_time"},
	{Name: "committee"},
	{Name: "address"},
	{Name: "amount_for_sale"},
	{Name: "requested_amount"},
	{Name: "state"},
	{Name: "success"},
	{Name: "sold_amount"},
	{Name: "bought_amount"},
	{Name: "local_origin"},
	{Name: "locked_height"},
}

// DexOrder represents a versioned snapshot of a DEX limit order.
// DEX orders are AMM-based cross-chain swaps that go through a batch lifecycle.
// Snapshots are created at each state transition (future -> locked -> complete).
//
// Lifecycle States:
// - future: Order appears in next-dex-batch (not yet locked for execution)
// - locked: Order appears in dex-batch (locked for execution at this height)
// - complete: EventDexSwap fired, order executed with final amounts and success status
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - Initial state comes from RPC (DexBatch or NextDexBatch at height H)
// - Final state comes from events at height H describing orders from H-1
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Note: Order creation time is tracked via the dex_order_created_height materialized view,
// which calculates MIN(height) for each order_id. Consumers should JOIN with this view
// if they need to know when an order was created.
type DexOrder struct {
	// Identity
	OrderID string `ch:"order_id" json:"order_id"` // Hex string representation of order ID

	// Version tracking - every state change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries

	// Chain context
	Committee uint16 `ch:"committee" json:"committee"` // Committee ID (counter-asset chain ID)

	// Order details (from DexLimitOrder)
	Address         string `ch:"address" json:"address"`                   // Hex string representation of address
	AmountForSale   uint64 `ch:"amount_for_sale" json:"amount_for_sale"`   // Amount of asset being sold
	RequestedAmount uint64 `ch:"requested_amount" json:"requested_amount"` // Minimum amount of counter-asset to receive

	// Lifecycle state
	State string `ch:"state" json:"state"` // "future", "locked", or "complete"

	// Execution results (populated by EventDexSwap when state=complete)
	Success      bool   `ch:"success" json:"success"`             // Whether swap succeeded
	SoldAmount   uint64 `ch:"sold_amount" json:"sold_amount"`     // Actual amount sold (from event)
	BoughtAmount uint64 `ch:"bought_amount" json:"bought_amount"` // Actual amount bought (from event)
	LocalOrigin  bool   `ch:"local_origin" json:"local_origin"`   // Did user sell on this chain or counter chain

	// Batch tracking
	LockedHeight uint64 `ch:"locked_height" json:"locked_height"` // Height when order was locked (moved from next -> current batch)
}

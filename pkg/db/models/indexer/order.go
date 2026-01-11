package indexer

import (
	"time"
)

const OrdersProductionTableName = "orders"
const OrdersStagingTableName = "orders_staging"

// Order status constants
const (
	OrderStatusOpen     = "open"
	OrderStatusComplete = "complete"
	OrderStatusCanceled = "canceled"
)

// OrderColumns defines the schema for the orders table.
var OrderColumns = []ColumnDef{
	{Name: "order_id"},
	{Name: "height"},
	{Name: "height_time"},
	{Name: "committee"},
	{Name: "data"},
	{Name: "amount_for_sale"},
	{Name: "requested_amount"},
	{Name: "seller_receive_address"},
	{Name: "buyer_send_address"},
	{Name: "buyer_receive_address"},
	{Name: "buyer_chain_deadline"},
	{Name: "sellers_send_address"},
	{Name: "status"},
}

// Order represents a versioned snapshot of an order's state.
// Snapshots are created ONLY when order state changes (not every block).
// This enables temporal queries like "What was order X's state at height 5000?"
// while using significantly less storage than storing all orders at every height.
//
// The snapshot-on-change pattern works correctly with parallel/unordered indexing because:
// - We compare RPC(height H) vs RPC(height H-1) to detect changes
// - We don't rely on database state which may be incomplete during parallel indexing
// - ReplacingMergeTree handles deduplication if the same height is indexed multiple times
//
// Note: Order creation time is tracked via the order_created_height materialized view,
// which calculates MIN(height) for each order_id. Consumers should JOIN with this view
// if they need to know when an order was created.
type Order struct {
	// Identity - OrderID is the unique identifier for this order
	OrderID string `ch:"order_id" json:"order_id"` // Unique order identifier

	// Version tracking - every state change creates a new snapshot
	Height     uint64    `ch:"height" json:"height"`           // Height at which this snapshot was created
	HeightTime time.Time `ch:"height_time" json:"height_time"` // Block timestamp for time-range queries

	// Order Details
	Committee            uint16 `ch:"committee" json:"committee"`                           // Committee ID for the order
	Data                 string `ch:"data" json:"data"`                                     // Generic data field for committee-specific functionality
	AmountForSale        uint64 `ch:"amount_for_sale" json:"amount_for_sale"`               // Amount being sold
	RequestedAmount      uint64 `ch:"requested_amount" json:"requested_amount"`             // Amount requested in return
	SellerReceiveAddress string `ch:"seller_receive_address" json:"seller_receive_address"` // External chain address to receive counter-asset
	BuyerSendAddress     string `ch:"buyer_send_address" json:"buyer_send_address"`         // Address buyer transfers funds from (empty if not locked)
	BuyerReceiveAddress  string `ch:"buyer_receive_address" json:"buyer_receive_address"`   // Buyer Canopy address to receive CNPY (empty if not locked)
	BuyerChainDeadline   uint64 `ch:"buyer_chain_deadline" json:"buyer_chain_deadline"`     // External chain height deadline (0 if not locked)
	SellersSendAddress   string `ch:"sellers_send_address" json:"sellers_send_address"`     // Signing address of seller selling CNPY

	// Status tracking (LowCardinality for efficient filtering)
	// Possible values: "open", "complete", "canceled"
	Status string `ch:"status" json:"status"` // LowCardinality(String) for efficient filtering
}

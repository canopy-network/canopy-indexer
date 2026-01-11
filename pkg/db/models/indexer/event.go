package indexer

import (
	"time"
)

const EventsProductionTableName = "events"
const EventsStagingTableName = "events_staging"

// EventColumns defines the schema for the events table.
// Uses non-Nullable types with defaults (0 for numbers, "" for strings, false for bools)
// for efficient storage and querying.
// Codecs are optimized for 15x compression ratio:
// - DoubleDelta,LZ4 for sequential/monotonic values (height, timestamps)
// - ZSTD(1) for strings (addresses, hashes, event_type)
// - Delta,ZSTD(3) for gradually changing amounts
var EventColumns = []ColumnDef{
	{Name: "height"},
	{Name: "chain_id", CrossChainRename: "event_chain_id"},
	{Name: "address"},
	{Name: "reference"},
	{Name: "event_type"},
	{Name: "block_height"},
	{Name: "amount"},
	{Name: "sold_amount"},
	{Name: "bought_amount"},
	{Name: "local_amount"},
	{Name: "remote_amount"},
	{Name: "success"},
	{Name: "local_origin"},
	{Name: "order_id"},
	{Name: "points_received"},
	{Name: "points_burned"},
	{Name: "data"},
	{Name: "seller_receive_address"},
	{Name: "buyer_send_address"},
	{Name: "sellers_send_address"},
	{Name: "msg"},
	{Name: "height_time"},
}

// Event stores ALL event data emitted during block processing.
// Events can be emitted in three contexts:
//   - begin_block: Events occurring at the start of block processing
//   - tx_hash: Events occurring during transaction execution (referenced by transaction hash)
//   - end_block: Events occurring at the end of block processing
//
// Common queryable fields are typed columns.
// Type-specific fields are stored in the compressed 'msg' JSON field.
// Queries only read the columns they need.
//
// Non-Nullable with defaults: Uses value types instead of pointers to avoid
// UInt8 null-mask overhead. Default values (0, false, ”) indicate "not applicable".
type Event struct {
	// Primary key (composite)
	Height      uint64 `ch:"height" json:"height"`
	ChainID     uint16 `ch:"chain_id" json:"chain_id"` // Which nested chain (committee) this event is for
	Address     string `ch:"address" json:"address"`
	Reference   string `ch:"reference" json:"reference"`       // "begin_block", tx hash, or "end_block"
	EventType   string `ch:"event_type" json:"event_type"`     // LowCardinality for efficient filtering
	BlockHeight uint64 `ch:"block_height" json:"block_height"` // Block number where event occurred

	// Extracted queryable fields (default 0/false/'' when not applicable)
	Amount         uint64 `ch:"amount" json:"amount,omitempty"`
	SoldAmount     uint64 `ch:"sold_amount" json:"sold_amount,omitempty"`
	BoughtAmount   uint64 `ch:"bought_amount" json:"bought_amount,omitempty"`
	LocalAmount    uint64 `ch:"local_amount" json:"local_amount,omitempty"`
	RemoteAmount   uint64 `ch:"remote_amount" json:"remote_amount,omitempty"`
	Success        bool   `ch:"success" json:"success,omitempty"`
	LocalOrigin    bool   `ch:"local_origin" json:"local_origin,omitempty"`
	OrderID        string `ch:"order_id" json:"order_id,omitempty"`
	PointsReceived uint64 `ch:"points_received" json:"points_received,omitempty"`
	PointsBurned   uint64 `ch:"points_burned" json:"points_burned,omitempty"`

	// OrderBookSwap-specific fields
	Data                 string `ch:"data" json:"data,omitempty"`
	SellerReceiveAddress string `ch:"seller_receive_address" json:"seller_receive_address,omitempty"`
	BuyerSendAddress     string `ch:"buyer_send_address" json:"buyer_send_address,omitempty"`
	SellersSendAddress   string `ch:"sellers_send_address" json:"sellers_send_address,omitempty"`

	// Full message (compressed)
	Msg string `ch:"msg" json:"msg"`

	// Time-range query field
	HeightTime time.Time `ch:"height_time" json:"height_time"`
}

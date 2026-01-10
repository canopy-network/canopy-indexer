package indexer

import (
	"time"
)

const TxsProductionTableName = "txs"
const TxsStagingTableName = "txs_staging"

// TransactionColumns defines the schema for the txs table.
// Uses non-Nullable types with defaults to avoid UInt8 null-mask overhead per column.
// Default values: 0 for integers, ” for strings, 0.0 for floats.
// Codecs are optimized for 15x compression ratio:
// - DoubleDelta,LZ4 for sequential/monotonic values (height, timestamps)
// - ZSTD(1) for strings (addresses, hashes, message_type)
// - Delta,ZSTD(3) for gradually changing amounts
var TransactionColumns = []ColumnDef{
	{Name: "height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "tx_hash", Type: "String", Codec: "ZSTD(1)"},
	{Name: "tx_index", Type: "UInt16", Codec: "Delta, ZSTD(3)"},
	{Name: "time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
	{Name: "height_time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
	{Name: "created_height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "network_id", Type: "UInt32", Codec: "Delta, ZSTD(3)"},
	{Name: "message_type", Type: "LowCardinality(String)", Codec: "ZSTD(1)"},
	{Name: "signer", Type: "String", Codec: "ZSTD(1)"},
	{Name: "amount", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"},
	{Name: "fee", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "memo", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "validator_address", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "commission", Type: "Float64 DEFAULT 0", Codec: "ZSTD(3)"},
	{Name: "chain_id", Type: "UInt16 DEFAULT 0", Codec: "Delta, ZSTD(1)", CrossChainRename: "tx_chain_id"},
	{Name: "sell_amount", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"},
	{Name: "buy_amount", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"},
	{Name: "liquidity_amount", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"},
	{Name: "liquidity_percent", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"},
	{Name: "order_id", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "price", Type: "Float64 DEFAULT 0", Codec: "ZSTD(3)"},
	{Name: "param_key", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "param_value", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "committee_id", Type: "UInt16 DEFAULT 0", Codec: "Delta, ZSTD(1)"},
	{Name: "recipient", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "poll_hash", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "buyer_receive_address", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "buyer_send_address", Type: "String DEFAULT ''", Codec: "ZSTD(1)"},
	{Name: "buyer_chain_deadline", Type: "UInt64 DEFAULT 0", Codec: "Delta, ZSTD(3)"},
	{Name: "msg", Type: "String", Codec: "ZSTD(3)"},
	{Name: "public_key", Type: "String DEFAULT ''", Codec: "ZSTD(3)"},
	{Name: "signature", Type: "String DEFAULT ''", Codec: "ZSTD(3)"},
}

// Transaction stores ALL transaction data in a single table.
// Common queryable fields are typed columns.
// Type-specific fields are stored in the compressed 'msg' JSON field.
// ClickHouse's columnar storage ensures list queries only read the columns they need.
// Uses non-Nullable types with defaults (0, ”, 0.0) to avoid UInt8 null-mask overhead.
type Transaction struct {
	// Primary key
	Height  uint64 `ch:"height" json:"height"`
	TxHash  string `ch:"tx_hash" json:"tx_hash"`
	TxIndex uint16 `ch:"tx_index" json:"tx_index"` // Transaction index within block

	// Time fields for queries
	Time          time.Time `ch:"time" json:"time"`                     // Transaction timestamp
	HeightTime    time.Time `ch:"height_time" json:"height_time"`       // Block timestamp for range queries
	CreatedHeight uint64    `ch:"created_height" json:"created_height"` // Height when tx was created
	NetworkID     uint32    `ch:"network_id" json:"network_id"`         // Network identifier

	// Common filterable fields
	MessageType string `ch:"message_type" json:"message_type"` // LowCardinality(String) for efficient filtering
	Signer      string `ch:"signer" json:"signer"`             // Transaction signer address
	Amount      uint64 `ch:"amount" json:"amount,omitempty"`   // Amount transferred/staked/delegated (0 for votes, etc.)
	Fee         uint64 `ch:"fee" json:"fee"`                   // Transaction fee
	Memo        string `ch:"memo" json:"memo,omitempty"`       // Transaction memo (poll/order operations)

	// ===== EXTRACTED FIELDS (for efficient querying) =====

	// Validator-related (stake, unstake, editStake)
	ValidatorAddress string  `ch:"validator_address" json:"validator_address,omitempty"`
	Commission       float64 `ch:"commission" json:"commission,omitempty"`

	// DEX-related (dexLimitOrder, dexLiquidityDeposit, dexLiquidityWithdraw)
	ChainID          uint16 `ch:"chain_id" json:"chain_id,omitempty"`
	SellAmount       uint64 `ch:"sell_amount" json:"sell_amount,omitempty"`
	BuyAmount        uint64 `ch:"buy_amount" json:"buy_amount,omitempty"`
	LiquidityAmt     uint64 `ch:"liquidity_amount" json:"liquidity_amount,omitempty"`   // For dexLiquidityDeposit
	LiquidityPercent uint64 `ch:"liquidity_percent" json:"liquidity_percent,omitempty"` // For dexLiquidityWithdraw

	// Order-related (createOrder, editOrder, deleteOrder)
	OrderID string  `ch:"order_id" json:"order_id,omitempty"`
	Price   float64 `ch:"price" json:"price,omitempty"`

	// Governance-related (changeParameter, startPoll, votePoll)
	ParamKey   string `ch:"param_key" json:"param_key,omitempty"`
	ParamValue string `ch:"param_value" json:"param_value,omitempty"`

	// Other
	CommitteeID uint16 `ch:"committee_id" json:"committee_id,omitempty"` // For: subsidy
	Recipient   string `ch:"recipient" json:"recipient,omitempty"`       // For: daoTransfer
	PollHash    string `ch:"poll_hash" json:"poll_hash,omitempty"`       // For: startPoll, votePoll

	// LockOrder-related (lockOrder memo transactions)
	BuyerReceiveAddress string `ch:"buyer_receive_address" json:"buyer_receive_address,omitempty"` // Canopy address to receive tokens
	BuyerSendAddress    string `ch:"buyer_send_address" json:"buyer_send_address,omitempty"`       // External chain address sending payment
	BuyerChainDeadline  uint64 `ch:"buyer_chain_deadline" json:"buyer_chain_deadline,omitempty"`   // Deadline height on buyer's chain

	// Full message as compressed JSON (ALL type-specific fields)
	Msg string `ch:"msg" json:"msg"` // Complete message data with ZSTD compression

	// Signature fields (compressed, permanent audit trail)
	PublicKey string `ch:"public_key" json:"public_key,omitempty"` // Compressed with ZSTD
	Signature string `ch:"signature" json:"signature,omitempty"`   // Compressed with ZSTD
}

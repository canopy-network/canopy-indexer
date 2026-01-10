package indexer

import (
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
)

const BlocksProductionTableName = "blocks"
const BlocksStagingTableName = BlocksProductionTableName + entities.StagingSuffix

// BlockColumns defines the schema for the blocks table.
// Codecs are optimized for 15x compression ratio:
// - DoubleDelta,LZ4 for sequential/monotonic values (height, timestamps)
// - ZSTD(1) for strings (hashes, addresses)
// - Delta,ZSTD(3) for gradually changing counts and metrics
var BlockColumns = []ColumnDef{
	{Name: "height", Type: "UInt64", Codec: "DoubleDelta, LZ4"},
	{Name: "hash", Type: "String", Codec: "ZSTD(1)"},
	{Name: "time", Type: "DateTime64(6)", Codec: "DoubleDelta, LZ4"},
	{Name: "network_id", Type: "UInt32", Codec: "Delta, ZSTD(3)"},
	{Name: "parent_hash", Type: "String", Codec: "ZSTD(1)"},
	{Name: "proposer_address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "size", Type: "Int32", Codec: "Delta, ZSTD(3)"},
	// Block metrics and verification fields
	{Name: "num_txs", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "total_txs", Type: "UInt64", Codec: "Delta, ZSTD(3)"},
	{Name: "total_vdf_iterations", Type: "Int32", Codec: "Delta, ZSTD(3)"},
	// Merkle roots for verification
	{Name: "state_root", Type: "String", Codec: "ZSTD(1)"},
	{Name: "transaction_root", Type: "String", Codec: "ZSTD(1)"},
	{Name: "validator_root", Type: "String", Codec: "ZSTD(1)"},
	{Name: "next_validator_root", Type: "String", Codec: "ZSTD(1)"},
}

type Block struct {
	Height          uint64    `ch:"height" json:"height"`
	Hash            string    `ch:"hash" json:"hash"`
	Time            time.Time `ch:"time" json:"time"` // stored as DateTime64(6)
	NetworkID       uint32    `ch:"network_id" json:"network_id"`
	LastBlockHash   string    `ch:"parent_hash" json:"parent_hash"`
	ProposerAddress string    `ch:"proposer_address" json:"proposer_address"`
	Size            int32     `ch:"size" json:"size"`
	// Block metrics
	NumTxs             uint64 `ch:"num_txs" json:"num_txs"`
	TotalTxs           uint64 `ch:"total_txs" json:"total_txs"`
	TotalVDFIterations int32  `ch:"total_vdf_iterations" json:"total_vdf_iterations"`
	// Merkle roots for verification
	StateRoot         string `ch:"state_root" json:"state_root"`
	TransactionRoot   string `ch:"transaction_root" json:"transaction_root"`
	ValidatorRoot     string `ch:"validator_root" json:"validator_root"`
	NextValidatorRoot string `ch:"next_validator_root" json:"next_validator_root"`
}

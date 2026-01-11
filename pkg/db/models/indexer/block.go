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
	{Name: "height"},
	{Name: "block_hash"},
	{Name: "height_time"},
	{Name: "network_id"},
	{Name: "parent_hash"},
	{Name: "proposer_address"},
	{Name: "size"},
	// Block metrics and verification fields
	{Name: "num_txs"},
	{Name: "total_txs"},
	{Name: "total_vdf_iterations", CrossChainSkip: true},
	// Merkle roots for verification
	{Name: "state_root", CrossChainSkip: true},
	{Name: "transaction_root", CrossChainSkip: true},
	{Name: "validator_root", CrossChainSkip: true},
	{Name: "next_validator_root", CrossChainSkip: true},
}

type Block struct {
	Height          uint64    `ch:"height" json:"height"`
	Hash            string    `ch:"block_hash" json:"hash"`
	Time            time.Time `ch:"height_time" json:"time"` // stored as DateTime64(6)
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

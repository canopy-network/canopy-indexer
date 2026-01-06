package indexer

import (
	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeBlock(batch *pgx.Batch, data *BlockData) {
	b := transform.BlockFromResult(data.Block)

	batch.Queue(`
		INSERT INTO blocks (chain_id, height, height_time, block_hash, proposer_address, total_txs, num_txs)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chain_id, height) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			proposer_address = EXCLUDED.proposer_address,
			total_txs = EXCLUDED.total_txs,
			num_txs = EXCLUDED.num_txs
	`,
		data.ChainID,
		data.Height,
		data.BlockTime,
		b.Hash,
		b.ProposerAddress,
		b.TotalTxs,
		b.NumTxs,
	)
}

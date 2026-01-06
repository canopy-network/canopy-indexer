package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/pgindexer/pkg/transform"
)

func (idx *Indexer) saveBlock(ctx context.Context, chainID, height uint64, blockTime time.Time, block *lib.BlockResult) error {
	b := transform.BlockFromResult(block)

	_, err := idx.db.Exec(ctx, `
		INSERT INTO blocks (chain_id, height, height_time, block_hash, proposer_address, total_txs, num_txs)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chain_id, height) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			proposer_address = EXCLUDED.proposer_address,
			total_txs = EXCLUDED.total_txs,
			num_txs = EXCLUDED.num_txs
	`,
		chainID,
		height,
		blockTime,
		b.Hash,
		b.ProposerAddress,
		b.TotalTxs,
		b.NumTxs,
	)
	return err
}

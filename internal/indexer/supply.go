package indexer

import (
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeSupply(batch *pgx.Batch, data *BlockData) {
	if data.Supply == nil {
		return
	}

	batch.Queue(`
		INSERT INTO supply (chain_id, total, staked, delegated_only, height, height_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chain_id, height) DO UPDATE SET
			total = EXCLUDED.total,
			staked = EXCLUDED.staked,
			delegated_only = EXCLUDED.delegated_only
	`,
		data.ChainID,
		data.Supply.Total,
		data.Supply.Staked,
		data.Supply.DelegatedOnly,
		data.Height,
		data.BlockTime,
	)
}

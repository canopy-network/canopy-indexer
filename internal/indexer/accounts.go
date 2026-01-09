package indexer

import (
	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeAccounts(batch *pgx.Batch, data *blob.BlockData) {
	for _, acc := range data.Accounts {
		batch.Queue(`
			INSERT INTO accounts (chain_id, address, amount, rewards, slashes, height, height_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (chain_id, address, height) DO UPDATE SET
				amount = EXCLUDED.amount,
				rewards = EXCLUDED.rewards,
				slashes = EXCLUDED.slashes
		`,
			data.ChainID,
			transform.BytesToHex(acc.Address),
			acc.Amount,
			0, // rewards - extract if available
			0, // slashes - extract if available
			data.Height,
			data.BlockTime,
		)
	}
}

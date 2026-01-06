package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) indexAccounts(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	accounts, err := rpc.AccountsByHeight(ctx, height)
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, acc := range accounts {
		batch.Queue(`
			INSERT INTO accounts (chain_id, address, amount, rewards, slashes, height, height_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (chain_id, address, height) DO UPDATE SET
				amount = EXCLUDED.amount,
				rewards = EXCLUDED.rewards,
				slashes = EXCLUDED.slashes
		`,
			chainID,
			transform.BytesToHex(acc.Address),
			acc.Amount,
			0, // rewards - extract if available
			0, // slashes - extract if available
			height,
			blockTime,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range accounts {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

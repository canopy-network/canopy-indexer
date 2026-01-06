package indexer

import (
	"context"
	"time"
)

func (idx *Indexer) indexSupply(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	supply, err := rpc.SupplyByHeight(ctx, height)
	if err != nil {
		return err
	}

	if supply == nil {
		return nil
	}

	_, err = idx.db.Exec(ctx, `
		INSERT INTO supply (chain_id, total, staked, delegated_only, height, height_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chain_id, height) DO UPDATE SET
			total = EXCLUDED.total,
			staked = EXCLUDED.staked,
			delegated_only = EXCLUDED.delegated_only
	`,
		chainID,
		supply.Total,
		supply.Staked,
		supply.DelegatedOnly,
		height,
		blockTime,
	)

	return err
}

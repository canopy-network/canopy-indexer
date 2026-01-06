package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) indexCommittees(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	committees, err := rpc.CommitteesDataByHeight(ctx, height)
	if err != nil {
		return err
	}

	subsidized, err := rpc.SubsidizedCommitteesByHeight(ctx, height)
	if err != nil {
		return err
	}

	retired, err := rpc.RetiredCommitteesByHeight(ctx, height)
	if err != nil {
		return err
	}

	// Build lookup maps
	subsidizedMap := make(map[uint64]bool)
	for _, id := range subsidized {
		subsidizedMap[id] = true
	}

	retiredMap := make(map[uint64]bool)
	for _, id := range retired {
		retiredMap[id] = true
	}

	if len(committees) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	numQueries := 0

	for _, comm := range committees {
		batch.Queue(`
			INSERT INTO committees (
				chain_id, committee_chain_id, last_root_height_updated, last_chain_height_updated,
				number_of_samples, subsidized, retired, height, height_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (chain_id, committee_chain_id, height) DO UPDATE SET
				last_root_height_updated = EXCLUDED.last_root_height_updated,
				last_chain_height_updated = EXCLUDED.last_chain_height_updated,
				number_of_samples = EXCLUDED.number_of_samples,
				subsidized = EXCLUDED.subsidized,
				retired = EXCLUDED.retired
		`,
			chainID,
			comm.ChainId,
			comm.LastRootHeightUpdated,
			comm.LastChainHeightUpdated,
			comm.NumberOfSamples,
			subsidizedMap[comm.ChainId],
			retiredMap[comm.ChainId],
			height,
			blockTime,
		)
		numQueries++

		// Extract and insert committee payments
		payments := transform.CommitteePaymentsFromLib(comm)
		for _, p := range payments {
			batch.Queue(`
				INSERT INTO committee_payments (
					chain_id, committee_id, address, percent, height, height_time
				) VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (chain_id, committee_id, address, height) DO NOTHING
			`,
				chainID,
				p.CommitteeID,
				p.Address,
				p.Percent,
				height,
				blockTime,
			)
			numQueries++
		}
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < numQueries; i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

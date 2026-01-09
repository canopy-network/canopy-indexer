package indexer

import (
	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeCommittees(batch *pgx.Batch, data *blob.BlockData) {
	if len(data.Committees) == 0 {
		return
	}

	// Build lookup maps from shared data
	subsidizedMap := make(map[uint64]bool)
	for _, id := range data.SubsidizedCommittees {
		subsidizedMap[id] = true
	}

	retiredMap := make(map[uint64]bool)
	for _, id := range data.RetiredCommittees {
		retiredMap[id] = true
	}

	for _, comm := range data.Committees {
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
			data.ChainID,
			comm.ChainId,
			comm.LastRootHeightUpdated,
			comm.LastChainHeightUpdated,
			comm.NumberOfSamples,
			subsidizedMap[comm.ChainId],
			retiredMap[comm.ChainId],
			data.Height,
			data.BlockTime,
		)

		// Extract and insert committee payments
		payments := transform.CommitteePaymentsFromLib(comm)
		for _, p := range payments {
			batch.Queue(`
				INSERT INTO committee_payments (
					chain_id, committee_id, address, percent, height, height_time
				) VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (chain_id, committee_id, address, height) DO NOTHING
			`,
				data.ChainID,
				p.CommitteeID,
				p.Address,
				p.Percent,
				data.Height,
				data.BlockTime,
			)
		}
	}
}

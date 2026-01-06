package indexer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// writeAllData writes all indexed data in a single atomic transaction.
// Any write failure causes transaction rollback and returns error (NACK).
func (idx *Indexer) writeAllData(ctx context.Context, data *BlockData) error {
	return pgx.BeginFunc(ctx, idx.db, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}

		// Order matters for foreign key constraints (if any)
		// Block must be written first as other tables may reference it
		idx.writeBlock(batch, data)
		idx.writeTransactions(batch, data)
		idx.writeEvents(batch, data)
		idx.writeAccounts(batch, data)
		idx.writeValidators(batch, data)
		idx.writePools(batch, data)
		idx.writeOrders(batch, data)
		idx.writeDexPrices(batch, data)
		idx.writeDexBatch(batch, data)
		idx.writeParams(batch, data)
		idx.writeSupply(batch, data)
		idx.writeCommittees(batch, data)

		// Execute all queued statements
		if batch.Len() == 0 {
			return nil // Nothing to write
		}

		br := tx.SendBatch(ctx, batch)
		defer br.Close()

		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				return fmt.Errorf("batch statement %d: %w", i, err)
			}
		}

		return nil
	})
}

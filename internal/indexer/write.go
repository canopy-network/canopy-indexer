package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/jackc/pgx/v5"
)

// writeAllData writes all indexed data in a single atomic transaction.
// Any write failure causes transaction rollback and returns error (NACK).
func (idx *Indexer) writeAllData(ctx context.Context, data *blob.BlockData) error {
	start := time.Now()
	slog.Debug("pg transaction: BEGIN", "height", data.Height, "chain_id", data.ChainID)

	err := idx.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		batch := idx.db.PrepareBatch(ctx)

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
			slog.Debug("pg transaction: COMMIT (empty batch)", "height", data.Height)
			return nil // Nothing to write
		}

		slog.Debug("pg transaction: executing batch", "height", data.Height, "statements", batch.Len())

		br := idx.db.SendBatch(ctx, batch)
		defer br.Close()

		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				slog.Error("pg transaction: ROLLBACK", "height", data.Height, "statement", i, "err", err)
				return fmt.Errorf("batch statement %d: %w", i, err)
			}
		}

		return nil
	})

	if err != nil {
		slog.Debug("pg transaction: ROLLBACK", "height", data.Height, "duration", time.Since(start), "err", err)
		return err
	}

	slog.Debug("pg transaction: COMMIT", "height", data.Height, "duration", time.Since(start))
	return nil
}

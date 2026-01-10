package chain

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
)

// deleteBlocksAtHeight deletes all blocks at the specified height
func (db *DB) deleteBlocksAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM blocks WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteBlockSummariesAtHeight deletes block summaries at the specified height
func (db *DB) deleteBlockSummariesAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM block_summaries WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteTransactionsAtHeight deletes all transactions at the specified height
func (db *DB) deleteTransactionsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM txs WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteEventsAtHeight deletes all events at the specified height
func (db *DB) deleteEventsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM events WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteAccountsAtHeight deletes all account snapshots at the specified height
func (db *DB) deleteAccountsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM accounts WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteValidatorsAtHeight deletes all validator snapshots at the specified height
func (db *DB) deleteValidatorsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM validators WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteValidatorNonSigningInfoAtHeight deletes all non-signing info at the specified height
func (db *DB) deleteValidatorNonSigningInfoAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM validator_non_signing_info WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteValidatorDoubleSigningInfoAtHeight deletes all double signing info at the specified height
func (db *DB) deleteValidatorDoubleSigningInfoAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM validator_double_signing_info WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deletePoolsAtHeight deletes all pool snapshots at the specified height
func (db *DB) deletePoolsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM pools WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deletePoolPointsByHolderAtHeight deletes all pool points by holder at the specified height
func (db *DB) deletePoolPointsByHolderAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM pool_points_by_holder WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteOrdersAtHeight deletes all order snapshots at the specified height
func (db *DB) deleteOrdersAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM orders WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteDexOrdersAtHeight deletes all DEX orders at the specified height
func (db *DB) deleteDexOrdersAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM dex_orders WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteDexDepositsAtHeight deletes all DEX deposits at the specified height
func (db *DB) deleteDexDepositsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM dex_deposits WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteDexWithdrawalsAtHeight deletes all DEX withdrawals at the specified height
func (db *DB) deleteDexWithdrawalsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM dex_withdrawals WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteDexPricesAtHeight deletes all DEX prices at the specified height
func (db *DB) deleteDexPricesAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM dex_prices WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteCommitteesAtHeight deletes all committee snapshots at the specified height
func (db *DB) deleteCommitteesAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM committees WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteCommitteeValidatorsAtHeight deletes all committee validator relationships at the specified height
func (db *DB) deleteCommitteeValidatorsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM committee_validators WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteCommitteePaymentsAtHeight deletes all committee payments at the specified height
func (db *DB) deleteCommitteePaymentsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM committee_payments WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteParamsAtHeight deletes all parameter snapshots at the specified height
func (db *DB) deleteParamsAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM params WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// deleteSupplyAtHeight deletes all supply snapshots at the specified height
func (db *DB) deleteSupplyAtHeight(ctx context.Context, exec postgres.Executor, height uint64) error {
	query := `DELETE FROM supply WHERE height = $1`
	_, err := exec.Exec(ctx, query, height)
	return err
}

// DeleteAllAtHeight deletes all data at the specified height across all tables
// This is used for reorg handling or reindexing
func (db *DB) DeleteAllAtHeight(ctx context.Context, height uint64) error {
	return db.BeginFunc(ctx, func(tx pgx.Tx) error {
		// Order matters: delete dependent data first
		deleteFuncs := []struct {
			name string
			fn   func(context.Context, postgres.Executor, uint64) error
		}{
			{"events", db.deleteEventsAtHeight},
			{"transactions", db.deleteTransactionsAtHeight},
			{"accounts", db.deleteAccountsAtHeight},
			{"validators", db.deleteValidatorsAtHeight},
			{"validator_non_signing_info", db.deleteValidatorNonSigningInfoAtHeight},
			{"validator_double_signing_info", db.deleteValidatorDoubleSigningInfoAtHeight},
			{"pools", db.deletePoolsAtHeight},
			{"pool_points_by_holder", db.deletePoolPointsByHolderAtHeight},
			{"orders", db.deleteOrdersAtHeight},
			{"dex_orders", db.deleteDexOrdersAtHeight},
			{"dex_deposits", db.deleteDexDepositsAtHeight},
			{"dex_withdrawals", db.deleteDexWithdrawalsAtHeight},
			{"dex_prices", db.deleteDexPricesAtHeight},
			{"committees", db.deleteCommitteesAtHeight},
			{"committee_validators", db.deleteCommitteeValidatorsAtHeight},
			{"committee_payments", db.deleteCommitteePaymentsAtHeight},
			{"params", db.deleteParamsAtHeight},
			{"supply", db.deleteSupplyAtHeight},
			{"block_summaries", db.deleteBlockSummariesAtHeight},
			{"blocks", db.deleteBlocksAtHeight},
		}

		for _, df := range deleteFuncs {
			if err := df.fn(ctx, tx, height); err != nil {
				return fmt.Errorf("failed to delete %s at height %d: %w", df.name, height, err)
			}
		}

		return nil
	})
}

// DeleteBlockRange deletes all data in the specified height range [fromHeight, toHeight]
// This is useful for cleanup or reindexing operations
func (db *DB) DeleteBlockRange(ctx context.Context, fromHeight, toHeight uint64) error {
	if fromHeight > toHeight {
		return fmt.Errorf("invalid range: fromHeight (%d) > toHeight (%d)", fromHeight, toHeight)
	}

	return db.BeginFunc(ctx, func(tx pgx.Tx) error {
		tables := []string{
			"events",
			"txs",
			"accounts",
			"validators",
			"validator_non_signing_info",
			"validator_double_signing_info",
			"pools",
			"pool_points_by_holder",
			"orders",
			"dex_orders",
			"dex_deposits",
			"dex_withdrawals",
			"dex_prices",
			"committees",
			"committee_validators",
			"committee_payments",
			"params",
			"supply",
			"block_summaries",
			"blocks",
		}

		for _, table := range tables {
			query := fmt.Sprintf("DELETE FROM %s WHERE height >= $1 AND height <= $2", table)
			_, err := tx.Exec(ctx, query, fromHeight, toHeight)
			if err != nil {
				return fmt.Errorf("failed to delete from %s: %w", table, err)
			}
		}

		return nil
	})
}

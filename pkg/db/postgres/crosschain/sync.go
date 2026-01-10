package crosschain

import (
	"context"
	"fmt"
	"strings"

	dbtypes "github.com/canopy-network/canopy-indexer/pkg/db"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// SetupChainSync creates triggers on all chain tables to automatically sync to crosschain tables.
// This sets up real-time data replication from chain-specific tables to global crosschain tables.
func (db *DB) SetupChainSync(ctx context.Context, chainID uint64) error {
	db.Logger.Info("Setting up cross-chain sync triggers for chain",
		zap.Uint64("chain_id", chainID),
		zap.String("target_db", db.DatabaseName()))

	chainDBName := fmt.Sprintf("chain_%d", chainID)

	// Get all table configurations
	configs := dbtypes.GetTableConfigs()

	return db.Client.BeginFunc(ctx, func(tx pgx.Tx) error {
		for _, cfg := range configs {
			if err := db.createSyncTrigger(ctx, tx, chainID, chainDBName, &cfg); err != nil {
				db.Logger.Error("Failed to create sync trigger",
					zap.Uint64("chain_id", chainID),
					zap.String("table", cfg.TableName),
					zap.Error(err))
				return fmt.Errorf("failed to create sync trigger for %s: %w", cfg.TableName, err)
			}
		}

		db.Logger.Info("Cross-chain sync triggers created successfully",
			zap.Uint64("chain_id", chainID),
			zap.Int("tables", len(configs)))

		return nil
	})
}

// createSyncTrigger creates a trigger on a chain table that inserts into the crosschain table
func (db *DB) createSyncTrigger(ctx context.Context, tx pgx.Tx, chainID uint64, chainDBName string, cfg *dbtypes.TableConfig) error {
	triggerName := fmt.Sprintf("sync_to_crosschain_%d", chainID)
	functionName := fmt.Sprintf("sync_%s_to_crosschain_%d", cfg.TableName, chainID)

	// Build column list for INSERT (excluding chain_id and updated_at which we add)
	columnList := strings.Join(cfg.ColumnNames, ", ")

	// Build NEW.column_name list for SELECT
	newColumns := make([]string, len(cfg.ColumnNames))
	for i, col := range cfg.ColumnNames {
		newColumns[i] = "NEW." + col
	}
	newColumnList := strings.Join(newColumns, ", ")

	// Create trigger function
	createFunctionSQL := fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION %s()
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO %s.%s (chain_id, %s, updated_at)
			VALUES (%d, %s, NOW())
			ON CONFLICT ON CONSTRAINT %s_pkey DO UPDATE SET
				%s,
				updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
	`, functionName, db.DatabaseName(), cfg.TableName, columnList, chainID, newColumnList, cfg.TableName, buildUpdateSet(cfg.ColumnNames))

	if _, err := tx.Exec(ctx, createFunctionSQL); err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Create trigger on chain table
	createTriggerSQL := fmt.Sprintf(`
		DROP TRIGGER IF EXISTS %s ON %s.%s;
		CREATE TRIGGER %s
		AFTER INSERT ON %s.%s
		FOR EACH ROW
		EXECUTE FUNCTION %s();
	`, triggerName, chainDBName, cfg.TableName, triggerName, chainDBName, cfg.TableName, functionName)

	if _, err := tx.Exec(ctx, createTriggerSQL); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	db.Logger.Debug("Sync trigger created",
		zap.String("table", cfg.TableName),
		zap.String("trigger", triggerName),
		zap.Uint64("chain_id", chainID))

	return nil
}

// buildUpdateSet creates the SET clause for ON CONFLICT DO UPDATE
// Example: "amount = EXCLUDED.amount, rewards = EXCLUDED.rewards"
func buildUpdateSet(columnNames []string) string {
	updates := make([]string, len(columnNames))
	for i, col := range columnNames {
		updates[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
	}
	return strings.Join(updates, ", ")
}

// ResyncTable copies all existing data from a chain table to the crosschain table.
// This is needed because triggers only sync NEW data after creation.
func (db *DB) ResyncTable(ctx context.Context, chainID uint64, tableName string) error {
	db.Logger.Info("Resyncing table for chain",
		zap.Uint64("chain_id", chainID),
		zap.String("table", tableName))

	// Validate and get table config
	cfg, err := db.getTableConfig(tableName)
	if err != nil {
		return err
	}

	chainDBName := fmt.Sprintf("chain_%d", chainID)
	columnList := strings.Join(cfg.ColumnNames, ", ")

	return db.Client.BeginFunc(ctx, func(tx pgx.Tx) error {
		// Delete existing data for this chain to avoid duplicates
		deleteSQL := fmt.Sprintf(`DELETE FROM %s.%s WHERE chain_id = $1`, db.DatabaseName(), tableName)
		if _, err := tx.Exec(ctx, deleteSQL, chainID); err != nil {
			return fmt.Errorf("failed to delete existing data: %w", err)
		}

		// Copy all data from chain table
		insertSQL := fmt.Sprintf(`
			INSERT INTO %s.%s (chain_id, %s, updated_at)
			SELECT $1, %s, NOW()
			FROM %s.%s
		`, db.DatabaseName(), tableName, columnList, columnList, chainDBName, tableName)

		if _, err := tx.Exec(ctx, insertSQL, chainID); err != nil {
			return fmt.Errorf("failed to insert data: %w", err)
		}

		db.Logger.Info("Table resync complete",
			zap.Uint64("chain_id", chainID),
			zap.String("table", tableName))

		return nil
	})
}

// ResyncChain resyncs all tables for a chain.
// This should be called after SetupChainSync to populate historical data.
func (db *DB) ResyncChain(ctx context.Context, chainID uint64) error {
	db.Logger.Info("Resyncing all tables for chain",
		zap.Uint64("chain_id", chainID))

	configs := dbtypes.GetTableConfigs()
	successCount := 0
	failCount := 0

	for _, cfg := range configs {
		if err := db.ResyncTable(ctx, chainID, cfg.TableName); err != nil {
			db.Logger.Error("Failed to resync table",
				zap.Uint64("chain_id", chainID),
				zap.String("table", cfg.TableName),
				zap.Error(err))
			failCount++
			// Continue with other tables
		} else {
			successCount++
		}
	}

	db.Logger.Info("Chain resync complete",
		zap.Uint64("chain_id", chainID),
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	if failCount > 0 {
		return fmt.Errorf("resync completed with %d failures", failCount)
	}

	return nil
}

// GetSyncStatus returns sync status for a specific chain and table.
// Shows row counts and lag to help monitor trigger health.
func (db *DB) GetSyncStatus(ctx context.Context, chainID uint64, tableName string) (*SyncStatus, error) {
	// Validate table name
	if err := db.validateTableName(tableName); err != nil {
		return nil, err
	}

	chainDBName := fmt.Sprintf("chain_%d", chainID)

	status := &SyncStatus{
		ChainID:   chainID,
		TableName: tableName,
	}

	// Get source table row count
	sourceCountSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, chainDBName, tableName)
	var sourceCount uint64
	if err := db.Client.Pool.QueryRow(ctx, sourceCountSQL).Scan(&sourceCount); err != nil {
		// Table might not exist yet
		sourceCount = 0
	}
	status.SourceRowCount = sourceCount

	// Get crosschain table row count for this chain
	globalCountSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s WHERE chain_id = $1`, db.DatabaseName(), tableName)
	var globalCount uint64
	if err := db.Client.Pool.QueryRow(ctx, globalCountSQL, chainID).Scan(&globalCount); err != nil {
		globalCount = 0
	}
	status.GlobalRowCount = globalCount

	// Calculate lag
	status.Lag = int64(sourceCount) - int64(globalCount)

	// Check if trigger exists
	triggerCheckSQL := `
		SELECT COUNT(*) FROM pg_trigger
		WHERE tgname = $1
		AND tgrelid = (SELECT oid FROM pg_class WHERE relname = $2 AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = $3))
	`
	triggerName := fmt.Sprintf("sync_to_crosschain_%d", chainID)
	var triggerCount int
	if err := db.Client.Pool.QueryRow(ctx, triggerCheckSQL, triggerName, tableName, chainDBName).Scan(&triggerCount); err == nil {
		status.TriggerExists = triggerCount > 0
	}

	return status, nil
}

// RemoveChainSync removes all triggers and deletes chain data from crosschain tables.
// This is called when a chain is removed from the system.
func (db *DB) RemoveChainSync(ctx context.Context, chainID uint64) error {
	db.Logger.Info("Removing cross-chain sync for chain",
		zap.Uint64("chain_id", chainID))

	chainDBName := fmt.Sprintf("chain_%d", chainID)
	configs := dbtypes.GetTableConfigs()

	return db.Client.BeginFunc(ctx, func(tx pgx.Tx) error {
		// Drop triggers and functions
		for _, cfg := range configs {
			triggerName := fmt.Sprintf("sync_to_crosschain_%d", chainID)
			functionName := fmt.Sprintf("sync_%s_to_crosschain_%d", cfg.TableName, chainID)

			// Drop trigger
			dropTriggerSQL := fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s.%s`, triggerName, chainDBName, cfg.TableName)
			if _, err := tx.Exec(ctx, dropTriggerSQL); err != nil {
				db.Logger.Warn("Failed to drop trigger",
					zap.String("trigger", triggerName),
					zap.Uint64("chain_id", chainID),
					zap.Error(err))
			}

			// Drop function
			dropFunctionSQL := fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName)
			if _, err := tx.Exec(ctx, dropFunctionSQL); err != nil {
				db.Logger.Warn("Failed to drop trigger function",
					zap.String("function", functionName),
					zap.Uint64("chain_id", chainID),
					zap.Error(err))
			}
		}

		// Delete all data for this chain from crosschain tables
		for _, cfg := range configs {
			deleteSQL := fmt.Sprintf(`DELETE FROM %s.%s WHERE chain_id = $1`, db.DatabaseName(), cfg.TableName)
			if _, err := tx.Exec(ctx, deleteSQL, chainID); err != nil {
				db.Logger.Warn("Failed to delete chain data",
					zap.String("table", cfg.TableName),
					zap.Uint64("chain_id", chainID),
					zap.Error(err))
			} else {
				db.Logger.Debug("Chain data deleted",
					zap.String("table", cfg.TableName),
					zap.Uint64("chain_id", chainID))
			}
		}

		db.Logger.Info("Cross-chain sync removed for chain",
			zap.Uint64("chain_id", chainID))

		return nil
	})
}

// validateTableName checks if a table name is in the allowed whitelist
func (db *DB) validateTableName(tableName string) error {
	validTables := make(map[string]bool)
	for _, cfg := range dbtypes.GetTableConfigs() {
		validTables[cfg.TableName] = true
	}

	if !validTables[tableName] {
		return fmt.Errorf("invalid table name: %s (not in whitelist)", tableName)
	}

	return nil
}

// getTableConfig retrieves the configuration for a specific table
func (db *DB) getTableConfig(tableName string) (*dbtypes.TableConfig, error) {
	if err := db.validateTableName(tableName); err != nil {
		return nil, err
	}

	for _, cfg := range dbtypes.GetTableConfigs() {
		if cfg.TableName == tableName {
			return &cfg, nil
		}
	}

	return nil, fmt.Errorf("table config not found: %s", tableName)
}

// SyncStatus represents the sync status for a specific chain and table
type SyncStatus struct {
	ChainID        uint64
	TableName      string
	SourceRowCount uint64
	GlobalRowCount uint64
	Lag            int64 // sourceCount - globalCount
	TriggerExists  bool
}

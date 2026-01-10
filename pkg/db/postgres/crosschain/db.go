package crosschain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"go.uber.org/zap"
)

// DB represents a PostgreSQL database connection for handling cross-chain operations
type DB struct {
	postgres.Client
	Name   string // Database name (e.g., "indexer")
	Schema string // Schema name (e.g., "crosschain")
}

// NewWithPoolConfig creates and initializes a crosschain database instance with custom pool configuration
func NewWithPoolConfig(ctx context.Context, logger *zap.Logger, databaseName string, poolConfig postgres.PoolConfig) (*DB, error) {
	schemaName := "crosschain"

	client, err := postgres.New(ctx, logger.With(
		zap.String("db", databaseName),
		zap.String("schema", schemaName),
		zap.String("component", poolConfig.Component),
	), databaseName, &poolConfig)
	if err != nil {
		return nil, err
	}

	crosschainDB := &DB{
		Client: client,
		Name:   databaseName,
		Schema: schemaName,
	}

	if err := crosschainDB.InitializeDB(ctx); err != nil {
		return nil, err
	}

	return crosschainDB, nil
}

// Close terminates the underlying PostgreSQL connection
func (db *DB) Close() error {
	db.Pool.Close()
	return nil
}

// GetConnection returns the underlying connection pool
func (db *DB) GetConnection() interface{} {
	return db.Pool
}

// DatabaseName returns the name of the crosschain database
func (db *DB) DatabaseName() string {
	return db.Name
}

// SchemaTable returns a schema-qualified table name
func (db *DB) SchemaTable(tableName string) string {
	return fmt.Sprintf("%s.%s", db.Schema, tableName)
}

// InitializeDB ensures the required database and tables exist
// Creates all tables, enums, views, and functions in parallel where possible
func (db *DB) InitializeDB(ctx context.Context) error {
	initStart := time.Now()

	db.Logger.Info("Initializing crosschain database",
		zap.String("database", db.Name),
		zap.String("schema", db.Schema))

	if err := db.CreateSchemaIfNotExists(ctx, db.Schema); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", db.Schema, err)
	}
	db.Logger.Info("Schema created successfully", zap.String("schema", db.Schema))

	// Step 1: Create enums (must be done before tables that use them)
	db.Logger.Info("Creating enums")
	if err := db.initEnums(ctx); err != nil {
		return fmt.Errorf("failed to create enums: %w", err)
	}

	// Step 2: Create all tables in parallel
	db.Logger.Info("Creating tables")
	if err := db.initTables(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Step 3: Create views (must be done after tables)
	db.Logger.Info("Creating views")
	if err := db.initViews(ctx); err != nil {
		return fmt.Errorf("failed to create views: %w", err)
	}

	// Step 4: Create functions (must be done after tables)
	db.Logger.Info("Creating functions")
	if err := db.initFunctions(ctx); err != nil {
		return fmt.Errorf("failed to create functions: %w", err)
	}

	db.Logger.Info("Crosschain database initialized successfully",
		zap.String("database", db.Name),
		zap.Duration("duration", time.Since(initStart)))

	return nil
}

// initTables creates all tables in parallel
func (db *DB) initTables(ctx context.Context) error {
	tableOps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"blocks", db.initBlocks},
		{"block_summaries", db.initBlockSummaries},
		{"txs", db.initTransactions},
		{"events", db.initEvents},
		{"accounts", db.initAccounts},
		{"validators", db.initValidators},
		{"validator_non_signing_info", db.initValidatorNonSigningInfo},
		{"validator_double_signing_info", db.initValidatorDoubleSigningInfo},
		{"committees", db.initCommittees},
		{"committee_validators", db.initCommitteeValidators},
		{"committee_payments", db.initCommitteePayments},
		{"pools", db.initPools},
		{"pool_points_by_holder", db.initPoolPointsByHolder},
		{"orders", db.initOrders},
		{"dex_orders", db.initDexOrders},
		{"dex_deposits", db.initDexDeposits},
		{"dex_withdrawals", db.initDexWithdrawals},
		{"dex_prices", db.initDexPrices},
		{"params", db.initParams},
		{"supply", db.initSupply},
		{"tvl_snapshots", db.initTvlSnapshots},
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(tableOps))

	for _, op := range tableOps {
		wg.Add(1)
		go func(name string, fn func(context.Context) error) {
			defer wg.Done()
			db.Logger.Debug("Creating table",
				zap.String("table", name),
				zap.String("database", db.Name),
			)
			if err := fn(ctx); err != nil {
				enhancedErr := fmt.Errorf("create table %s failed in crosschain database (db: %s): %w",
					name, db.Name, err)
				errChan <- enhancedErr
			}
		}(op.name, op.fn)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

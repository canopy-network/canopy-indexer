package chain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"go.uber.org/zap"
)

// DB represents a PostgreSQL database connection for handling chain operations
type DB struct {
	postgres.Client
	Name    string // Database name (e.g., "chain_1")
	ChainID uint64
}

// NewWithPoolConfig creates and initializes a chain database instance with custom pool configuration
func NewWithPoolConfig(ctx context.Context, logger *zap.Logger, chainID uint64, poolConfig postgres.PoolConfig) (*DB, error) {
	name := postgres.SanitizeName(fmt.Sprintf("chain_%d", chainID))

	client, err := postgres.New(ctx, logger.With(
		zap.String("db", name),
		zap.Uint64("chain_id", chainID),
		zap.String("component", poolConfig.Component),
	), name, &poolConfig)
	if err != nil {
		return nil, err
	}

	chainDB := &DB{
		Client:  client,
		Name:    name,
		ChainID: chainID,
	}

	if err := chainDB.InitializeDB(ctx); err != nil {
		return nil, err
	}

	return chainDB, nil
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

// DatabaseName returns the name of the chain database
func (db *DB) DatabaseName() string {
	return db.Name
}

// ChainKey returns the chain ID as a string
func (db *DB) ChainKey() string {
	return fmt.Sprintf("%d", db.ChainID)
}

// InitializeDB ensures the required database and tables exist
// Creates all tables in parallel for efficiency
func (db *DB) InitializeDB(ctx context.Context) error {
	initStart := time.Now()

	db.Logger.Info("Initializing chain database",
		zap.String("database", db.Name),
		zap.Uint64("chain_id", db.ChainID))

	if err := db.CreateDbIfNotExists(ctx, db.Name); err != nil {
		return fmt.Errorf("failed to create database %s: %w", db.Name, err)
	}
	db.Logger.Info("Database created successfully", zap.String("database", db.Name))

	// Create all tables in parallel (no staging tables for Postgres)
	initOps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"blocks", db.initBlocks},
		{"block_summaries", db.initBlockSummaries},
		{"transactions", db.initTransactions},
		{"accounts", db.initAccounts},
		{"events", db.initEvents},
		{"pools", db.initPools},
		{"orders", db.initOrders},
		{"dex_prices", db.initDexPrices},
		{"dex_orders", db.initDexOrders},
		{"dex_deposits", db.initDexDeposits},
		{"dex_withdrawals", db.initDexWithdrawals},
		{"pool_points_by_holder", db.initPoolPointsByHolder},
		{"validators", db.initValidators},
		{"validator_non_signing_info", db.initValidatorNonSigningInfo},
		{"validator_double_signing_info", db.initValidatorDoubleSigningInfo},
		{"params", db.initParams},
		{"committees", db.initCommittees},
		{"committee_validators", db.initCommitteeValidators},
		{"committee_payments", db.initCommitteePayments},
		{"poll_snapshots", db.initPollSnapshots},
		{"proposal_snapshots", db.initProposalSnapshots},
		{"supply", db.initSupply},
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(initOps))

	// Launch all init operations in parallel
	for _, op := range initOps {
		wg.Add(1)
		go func(name string, fn func(context.Context) error) {
			defer wg.Done()
			db.Logger.Debug("Initializing table",
				zap.String("table", name),
				zap.String("database", db.Name),
				zap.Uint64("chain_id", db.ChainID),
			)
			if err := fn(ctx); err != nil {
				enhancedErr := fmt.Errorf("init table %s failed in chain database (chain_id: %d, db: %s): %w",
					name, db.ChainID, db.Name, err)
				errChan <- enhancedErr
			}
		}(op.name, op.fn)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	db.Logger.Info("Chain database initialized successfully",
		zap.String("database", db.Name),
		zap.Uint64("chain_id", db.ChainID),
		zap.Duration("duration", time.Since(initStart)))

	return nil
}

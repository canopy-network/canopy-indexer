package admin

import (
	"context"
	"fmt"

	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// DB represents a PostgreSQL database connection for handling admin operations
type DB struct {
	postgres.Client
	Name string
}

// NewWithPoolConfig creates and initializes an admin database instance with custom pool configuration
func NewWithPoolConfig(ctx context.Context, logger *zap.Logger, name string, poolConfig postgres.PoolConfig) (*DB, error) {
	client, err := postgres.New(ctx, logger.With(
		zap.String("db", name),
		zap.String("component", poolConfig.Component),
	), name, &poolConfig)
	if err != nil {
		return nil, err
	}

	adminDB := &DB{
		Client: client,
		Name:   name,
	}

	if err := adminDB.InitializeDB(ctx); err != nil {
		return nil, err
	}

	return adminDB, nil
}

// Close terminates the underlying PostgreSQL connection
func (db *DB) Close() error {
	db.Pool.Close()
	return nil
}

// GetConnection returns the underlying connection pool
// Note: PostgreSQL uses pgx.Conn, but we return the pool for compatibility
func (db *DB) GetConnection() interface{} {
	return db.Pool
}

// DatabaseName returns the name of the admin database
func (db *DB) DatabaseName() string {
	return db.Name
}

// DropChainDatabase drops the chain-specific database
func (db *DB) DropChainDatabase(ctx context.Context, chainID uint64) error {
	dbName := postgres.SanitizeName(fmt.Sprintf("chain_%d", chainID))

	// Cannot use parameterized query for DROP DATABASE
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgx.Identifier{dbName}.Sanitize())
	if err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("drop chain database %s: %w", dbName, err)
	}
	return nil
}

// InitializeDB ensures the required database and tables exist
func (db *DB) InitializeDB(ctx context.Context) error {
	db.Logger.Info("Initializing admin database", zap.String("database", db.Name))

	if err := db.CreateDbIfNotExists(ctx, db.Name); err != nil {
		return fmt.Errorf("failed to create database %s: %w", db.Name, err)
	}
	db.Logger.Info("Database created successfully", zap.String("database", db.Name))

	db.Logger.Info("Initialize chains table", zap.String("database", db.Name))
	if err := db.initChains(ctx); err != nil {
		return err
	}

	db.Logger.Info("Initialize index_progress table", zap.String("database", db.Name))
	if err := db.initIndexProgress(ctx); err != nil {
		return err
	}

	db.Logger.Debug("Initialize reindex_requests table", zap.String("database", db.Name))
	if err := db.initReindexRequests(ctx); err != nil {
		return err
	}

	db.Logger.Debug("Initialize rpc_endpoints table", zap.String("database", db.Name))
	if err := db.initRPCEndpoints(ctx); err != nil {
		return err
	}

	return nil
}

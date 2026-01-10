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
	Name   string // Database name (e.g., "indexer")
	Schema string // Schema name (e.g., "admin")
}

// NewWithPoolConfig creates and initializes an admin database instance with custom pool configuration
func NewWithPoolConfig(ctx context.Context, logger *zap.Logger, databaseName string, poolConfig postgres.PoolConfig) (*DB, error) {
	schemaName := "admin"

	client, err := postgres.New(ctx, logger.With(
		zap.String("db", databaseName),
		zap.String("schema", schemaName),
		zap.String("component", poolConfig.Component),
	), databaseName, &poolConfig)
	if err != nil {
		return nil, err
	}

	adminDB := &DB{
		Client: client,
		Name:   databaseName,
		Schema: schemaName,
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

// SchemaTable returns a schema-qualified table name
func (db *DB) SchemaTable(tableName string) string {
	return fmt.Sprintf("%s.%s", db.Schema, tableName)
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
	db.Logger.Info("Initializing admin database",
		zap.String("database", db.Name),
		zap.String("schema", db.Schema))

	if err := db.CreateSchemaIfNotExists(ctx, db.Schema); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", db.Schema, err)
	}
	db.Logger.Info("Schema created successfully", zap.String("schema", db.Schema))

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

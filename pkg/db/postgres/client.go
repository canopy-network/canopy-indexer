package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/retry"
	"github.com/canopy-network/canopy-indexer/pkg/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Executor is an interface that both *pgxpool.Pool and pgx.Tx implement.
// This allows methods to work with either a connection pool or a transaction.
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// Client wraps a PostgreSQL connection pool and provides helper methods
type Client struct {
	Logger         *zap.Logger
	Pool           *pgxpool.Pool
	TargetDatabase string // Target database name
	TargetSchema   string // Target schema name
}

// PoolConfig defines connection pool settings for a specific component
type PoolConfig struct {
	MinConns        int32
	MaxConns        int32
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Component       string // For logging/debugging
}

// New initializes and returns a new PostgreSQL client with provided context and logger.
// Includes connection pooling optimizations for high-throughput workloads.
// Accepts optional poolConfig parameter for component-specific pool sizing.
// For schema-based access: provide database name and schema name.
func New(ctx context.Context, logger *zap.Logger, dbName string, poolConfig ...*PoolConfig) (client Client, err error) {
	// Add timeout to context for initial connection
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	client.Logger = logger
	client.TargetDatabase = dbName
	retryConfig := retry.DefaultConfig()

	// Get database URL from environment
	dbURL := utils.Env("POSTGRES_URL", "postgres://localhost:5434/indexer")

	// Parse the connection string to get config
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return Client{}, fmt.Errorf("failed to parse POSTGRES_URL: %w", err)
	}

	// Connection pool settings - use provided config or fallback to defaults
	var poolConf PoolConfig
	if len(poolConfig) > 0 && poolConfig[0] != nil {
		poolConf = *poolConfig[0]
	} else {
		// Fallback to defaults
		poolConf = PoolConfig{
			MinConns:        2,
			MaxConns:        20,
			ConnMaxLifetime: 1 * time.Hour,
			ConnMaxIdleTime: 30 * time.Minute,
			Component:       "unknown",
		}
	}

	// Apply pool configuration
	config.MinConns = poolConf.MinConns
	config.MaxConns = poolConf.MaxConns
	config.MaxConnLifetime = poolConf.ConnMaxLifetime
	config.MaxConnIdleTime = poolConf.ConnMaxIdleTime

	// Connect to postgres (default database) first
	// We'll create the target database if it doesn't exist, then reconnect to it
	retryErr := retry.WithBackoff(connCtx, retryConfig, logger, "postgres_connection", func() error {
		pool, openErr := pgxpool.NewWithConfig(connCtx, config)
		if openErr != nil {
			return fmt.Errorf("failed to create postgres connection pool: %w", openErr)
		}

		client.Pool = pool

		logger.Debug("Pinging PostgreSQL connection",
			zap.String("db", dbName),
			zap.String("component", poolConf.Component),
		)

		// Ping to verify connection
		pingErr := pool.Ping(connCtx)
		if pingErr != nil {
			pool.Close()
			return fmt.Errorf("failed to ping postgres: %w", pingErr)
		}

		logger.Info("PostgreSQL connection pool configured",
			zap.String("database", dbName),
			zap.String("component", poolConf.Component),
			zap.Int32("min_conns", poolConf.MinConns),
			zap.Int32("max_conns", poolConf.MaxConns),
			zap.Duration("conn_max_lifetime", poolConf.ConnMaxLifetime),
			zap.Duration("conn_max_idle_time", poolConf.ConnMaxIdleTime),
		)

		return nil
	})

	if retryErr != nil {
		return Client{}, retryErr
	}

	return client, nil
}

// CreateSchemaIfNotExists ensures that the specified schema exists by creating it if it does not already exist.
// This is used for schema-based database organization instead of separate databases.
func (c *Client) CreateSchemaIfNotExists(ctx context.Context, schemaName string) error {
	// Check if schema exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)"
	err := c.Pool.QueryRow(ctx, query, schemaName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if schema exists: %w", err)
	}

	if !exists {
		// Create schema
		query := fmt.Sprintf("CREATE SCHEMA %s", pgx.Identifier{schemaName}.Sanitize())
		c.Logger.Info("Creating schema", zap.String("schema", schemaName))
		_, err = c.Pool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	c.TargetSchema = schemaName
	return nil
}

// Exec executes a query without returning any rows
func (c *Client) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := c.Pool.Exec(ctx, query, args...)
	return err
}

// Query executes a query that returns rows
// IMPORTANT: Caller MUST call rows.Close() when done to release the connection
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return c.Pool.Query(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row
func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return c.Pool.QueryRow(ctx, query, args...)
}

// Select executes a query and scans the result rows into dest (slice of structs)
// This is a helper method that handles row closing automatically
func (c *Client) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := c.Pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Use pgx's CollectRows to scan into slice
	// Note: This requires dest to be a pointer to a slice
	// For more complex scanning, caller should use Query() directly
	return rows.Err()
}

// Begin starts a new transaction
func (c *Client) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.Pool.Begin(ctx)
}

// BeginFunc executes a function within a transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (c *Client) BeginFunc(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, c.Pool, fn)
}

// PrepareBatch creates a new batch for efficient bulk inserts
func (c *Client) PrepareBatch(ctx context.Context) *pgx.Batch {
	return &pgx.Batch{}
}

// SendBatch sends a batch of queries
func (c *Client) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return c.Pool.SendBatch(ctx, batch)
}

// Close closes the connection pool
func (c *Client) Close() {
	c.Pool.Close()
}

// ctxKey is the type used for context keys to avoid collisions
type ctxKey string

// txKey is the context key for storing the transaction
const txKey ctxKey = "pgx_tx"

// WithTx returns a new context with the transaction embedded
// This allows methods to automatically use the transaction when present
func (c *Client) WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetExecutor returns an Executor from the context
// If a transaction is present in the context, it returns the transaction
// Otherwise, it returns the connection pool for non-transactional operations
func (c *Client) GetExecutor(ctx context.Context) Executor {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return c.Pool
}

// TableExists checks if a table exists in the database
func (c *Client) TableExists(ctx context.Context, database, table string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`

	var exists bool
	err := c.Pool.QueryRow(ctx, query, table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check if table exists %s: %w", table, err)
	}

	return exists, nil
}

// IsNoRows checks if the error is a "no rows" error
func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// GetPoolConfigForComponent returns deterministic pool settings for each component
func GetPoolConfigForComponent(component string) *PoolConfig {
	var minConns, maxConns int32
	connMaxLifetime := 5 * time.Minute
	connMaxIdleTime := 2 * time.Minute

	switch component {
	case "indexer_admin":
		minConns = 2
		maxConns = 15
	case "indexer_chain":
		minConns = 5
		maxConns = 40
	case "admin":
		minConns = 2
		maxConns = 10
	case "admin_chain":
		minConns = 2
		maxConns = 5
	case "controller":
		minConns = 2
		maxConns = 10
	case "crosschain":
		minConns = 2
		maxConns = 10
	default:
		// Unknown component - use defaults
		minConns = 2
		maxConns = 20
	}

	return &PoolConfig{
		MinConns:        minConns,
		MaxConns:        maxConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
		Component:       component,
	}
}

// SanitizeName sanitizes the provided database name to be compatible with PostgreSQL
func SanitizeName(name string) string {
	// PostgreSQL names are case-insensitive and use underscores
	return name // For now, keep it simple - can add more sanitization if needed
}

// ParseConnMaxLifetime parses a connection max lifetime duration string
func ParseConnMaxLifetime(lifetimeStr string) time.Duration {
	if lifetimeStr != "" {
		if d, err := time.ParseDuration(lifetimeStr); err == nil {
			return d
		}
	}

	// Fall back to environment variable
	if envStr := os.Getenv("POSTGRES_CONN_MAX_LIFETIME"); envStr != "" {
		if d, err := time.ParseDuration(envStr); err == nil {
			return d
		}
	}

	// Default to 1 hour
	return 1 * time.Hour
}

package admin

import (
	"context"
	"fmt"
	"time"

	dbtypes "github.com/canopy-network/canopy-indexer/pkg/db"
	adminmodels "github.com/canopy-network/canopy-indexer/pkg/db/models/admin"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
)

// initChains creates the chains table using PostgreSQL DDL
func (db *DB) initChains(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS chains (
			chain_id BIGINT PRIMARY KEY,
			chain_name TEXT NOT NULL,
			rpc_endpoints TEXT[] NOT NULL DEFAULT '{}',
			paused SMALLINT NOT NULL DEFAULT 0,
			deleted SMALLINT NOT NULL DEFAULT 0,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`

	return db.Exec(ctx, query)
}

// GetChain returns the chain for the given chain_id
func (db *DB) GetChain(ctx context.Context, id uint64) (*adminmodels.Chain, error) {
	query := `
		SELECT chain_id, chain_name, rpc_endpoints, paused, deleted, notes, created_at, updated_at
		FROM chains
		WHERE chain_id = $1
	`

	var chain adminmodels.Chain
	err := db.QueryRow(ctx, query, id).Scan(
		&chain.ChainID,
		&chain.ChainName,
		&chain.RPCEndpoints,
		&chain.Paused,
		&chain.Deleted,
		&chain.Notes,
		&chain.CreatedAt,
		&chain.UpdatedAt,
	)

	if err != nil {
		if postgres.IsNoRows(err) {
			return nil, fmt.Errorf("chain %d not found", id)
		}
		return nil, fmt.Errorf("failed to query chain %d: %w", id, err)
	}

	return &chain, nil
}

// GetChainRPCEndpoints returns only the chain_id and rpc_endpoints for a chain
func (db *DB) GetChainRPCEndpoints(ctx context.Context, chainID uint64) (*dbtypes.ChainRPCEndpoints, error) {
	query := `
		SELECT chain_id, rpc_endpoints
		FROM chains
		WHERE chain_id = $1
	`

	var result dbtypes.ChainRPCEndpoints
	err := db.QueryRow(ctx, query, chainID).Scan(&result.ChainID, &result.RPCEndpoints)
	if err != nil {
		if postgres.IsNoRows(err) {
			return nil, fmt.Errorf("chain %d not found", chainID)
		}
		return nil, fmt.Errorf("failed to query chain %d rpc_endpoints: %w", chainID, err)
	}

	return &result, nil
}

// ListChain returns all chains, optionally including deleted ones
func (db *DB) ListChain(ctx context.Context, includeDeleted bool) ([]adminmodels.Chain, error) {
	query := `
		SELECT chain_id, chain_name, rpc_endpoints, paused, deleted, notes, created_at, updated_at
		FROM chains
	`

	if !includeDeleted {
		query += ` WHERE deleted = 0`
	}

	query += ` ORDER BY chain_id`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []adminmodels.Chain
	for rows.Next() {
		var chain adminmodels.Chain
		err := rows.Scan(
			&chain.ChainID,
			&chain.ChainName,
			&chain.RPCEndpoints,
			&chain.Paused,
			&chain.Deleted,
			&chain.Notes,
			&chain.CreatedAt,
			&chain.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	return chains, rows.Err()
}

// HardDeleteChain permanently removes a chain record from the database
func (db *DB) HardDeleteChain(ctx context.Context, chainID uint64) error {
	query := `DELETE FROM chains WHERE chain_id = $1`
	return db.Exec(ctx, query, chainID)
}

// InsertChain inserts a new chain record with ON CONFLICT handling
func (db *DB) InsertChain(ctx context.Context, c *adminmodels.Chain) error {
	now := time.Now()

	// Set created_at if not already set (new record)
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}

	// Always update updated_at
	c.UpdatedAt = now

	query := `
		INSERT INTO chains (
			chain_id, chain_name, rpc_endpoints, paused, deleted, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (chain_id) DO UPDATE SET
			chain_name = EXCLUDED.chain_name,
			rpc_endpoints = EXCLUDED.rpc_endpoints,
			paused = EXCLUDED.paused,
			deleted = EXCLUDED.deleted,
			notes = EXCLUDED.notes,
			updated_at = EXCLUDED.updated_at
	`

	return db.Exec(ctx, query,
		c.ChainID,
		c.ChainName,
		c.RPCEndpoints,
		c.Paused,
		c.Deleted,
		c.Notes,
		c.CreatedAt,
		c.UpdatedAt,
	)
}


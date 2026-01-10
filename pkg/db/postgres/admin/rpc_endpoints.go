package admin

import (
	"context"
	"fmt"
	"time"

	adminmodels "github.com/canopy-network/canopy-indexer/pkg/db/models/admin"
)

// initRPCEndpoints creates the rpc_endpoints table
func (db *DB) initRPCEndpoints(ctx context.Context) error {
	rpcEndpointsTable := db.SchemaTable("rpc_endpoints")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			chain_id BIGINT NOT NULL,
			endpoint TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'unknown',
			height BIGINT NOT NULL DEFAULT 0,
			latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, endpoint)
		)
	`, rpcEndpointsTable)

	return db.Exec(ctx, query)
}

// UpsertEndpointHealth inserts or updates RPC endpoint health status
func (db *DB) UpsertEndpointHealth(ctx context.Context, ep *adminmodels.RPCEndpoint) error {
	rpcEndpointsTable := db.SchemaTable("rpc_endpoints")
	query := fmt.Sprintf(`
		INSERT INTO %s (chain_id, endpoint, status, height, latency_ms, error, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chain_id, endpoint) DO UPDATE SET
			status = EXCLUDED.status,
			height = EXCLUDED.height,
			latency_ms = EXCLUDED.latency_ms,
			error = EXCLUDED.error,
			updated_at = EXCLUDED.updated_at
	`, rpcEndpointsTable)

	if ep.UpdatedAt.IsZero() {
		ep.UpdatedAt = time.Now()
	}

	return db.Exec(ctx, query,
		ep.ChainID,
		ep.Endpoint,
		ep.Status,
		ep.Height,
		ep.LatencyMs,
		ep.Error,
		ep.UpdatedAt,
	)
}

// GetEndpointsForChain returns all RPC endpoints for a chain
func (db *DB) GetEndpointsForChain(ctx context.Context, chainID uint64) ([]adminmodels.RPCEndpoint, error) {
	rpcEndpointsTable := db.SchemaTable("rpc_endpoints")
	query := fmt.Sprintf(`
		SELECT chain_id, endpoint, status, height, latency_ms, error, updated_at
		FROM %s
		WHERE chain_id = $1
		ORDER BY endpoint
	`, rpcEndpointsTable)

	rows, err := db.Query(ctx, query, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []adminmodels.RPCEndpoint
	for rows.Next() {
		var ep adminmodels.RPCEndpoint
		if err := rows.Scan(&ep.ChainID, &ep.Endpoint, &ep.Status, &ep.Height, &ep.LatencyMs, &ep.Error, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, rows.Err()
}

// GetEndpointsWithMinHeight returns RPC endpoints at or above a minimum height
func (db *DB) GetEndpointsWithMinHeight(ctx context.Context, chainID uint64, minHeight uint64) ([]adminmodels.RPCEndpoint, error) {
	rpcEndpointsTable := db.SchemaTable("rpc_endpoints")
	query := fmt.Sprintf(`
		SELECT chain_id, endpoint, status, height, latency_ms, error, updated_at
		FROM %s
		WHERE chain_id = $1
		  AND height >= $2
		ORDER BY height DESC, latency_ms ASC
	`, rpcEndpointsTable)

	rows, err := db.Query(ctx, query, chainID, minHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []adminmodels.RPCEndpoint
	for rows.Next() {
		var ep adminmodels.RPCEndpoint
		if err := rows.Scan(&ep.ChainID, &ep.Endpoint, &ep.Status, &ep.Height, &ep.LatencyMs, &ep.Error, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, rows.Err()
}

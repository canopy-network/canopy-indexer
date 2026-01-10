package admin

import (
	"context"
)

// initReindexRequests creates the reindex_requests table
func (db *DB) initReindexRequests(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS reindex_requests (
			chain_id BIGINT NOT NULL,
			from_height BIGINT NOT NULL,
			to_height BIGINT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMP,
			error TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (chain_id, from_height, to_height)
		)
	`

	return db.Exec(ctx, query)
}

// DeleteReindexRequestsForChain deletes all reindex requests for a chain
func (db *DB) DeleteReindexRequestsForChain(ctx context.Context, chainID uint64) error {
	query := `DELETE FROM reindex_requests WHERE chain_id = $1`
	return db.Exec(ctx, query, chainID)
}

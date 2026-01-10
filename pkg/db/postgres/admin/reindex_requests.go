package admin

import (
	"context"
	"fmt"
)

// initReindexRequests creates the reindex_requests table
func (db *DB) initReindexRequests(ctx context.Context) error {
	reindexTable := db.SchemaTable("reindex_requests")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
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
	`, reindexTable)

	return db.Exec(ctx, query)
}

// DeleteReindexRequestsForChain deletes all reindex requests for a chain
func (db *DB) DeleteReindexRequestsForChain(ctx context.Context, chainID uint64) error {
	reindexTable := db.SchemaTable("reindex_requests")
	query := fmt.Sprintf(`DELETE FROM %s WHERE chain_id = $1`, reindexTable)
	return db.Exec(ctx, query, chainID)
}

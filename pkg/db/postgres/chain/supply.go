package chain

import (
	"context"
)

// initSupply creates the supply table matching indexer.Supply
// This matches pkg/db/models/indexer/supply.go:30-39 (5 fields)
// Only inserted when supply changes
func (db *DB) initSupply(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS supply (
			total BIGINT NOT NULL DEFAULT 0,
			staked BIGINT NOT NULL DEFAULT 0,
			delegated_only BIGINT NOT NULL DEFAULT 0,
			height BIGINT PRIMARY KEY,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL
		);
	`

	return db.Exec(ctx, query)
}

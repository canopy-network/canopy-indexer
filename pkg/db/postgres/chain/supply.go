package chain

import (
	"context"
	"fmt"
)

// initSupply creates the supply table matching indexer.Supply
// This matches pkg/db/models/indexer/supply.go:30-39 (5 fields)
// Only inserted when supply changes
func (db *DB) initSupply(ctx context.Context) error {
	supplyTable := db.SchemaTable("supply")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			total BIGINT NOT NULL DEFAULT 0,
			staked BIGINT NOT NULL DEFAULT 0,
			delegated_only BIGINT NOT NULL DEFAULT 0,
			height BIGINT PRIMARY KEY,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL
		);
	`, supplyTable)

	return db.Exec(ctx, query)
}

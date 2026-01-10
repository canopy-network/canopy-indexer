package chain

import (
	"context"
)

// initAccounts creates the accounts table
func (db *DB) initAccounts(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS accounts (
			address TEXT NOT NULL,
			height BIGINT NOT NULL,
			amount BIGINT NOT NULL DEFAULT 0,
			rewards BIGINT NOT NULL DEFAULT 0,
			slashes BIGINT NOT NULL DEFAULT 0,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_accounts_height ON accounts(height);
	`

	return db.Exec(ctx, query)
}

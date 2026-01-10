package crosschain

import (
	"context"
)

// initEnums creates all enum types used by the crosschain database
func (db *DB) initEnums(ctx context.Context) error {
	query := `
		-- Validator status enum
		DO $$ BEGIN
			CREATE TYPE validator_status AS ENUM ('active', 'paused', 'unstaking');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		-- Order status enum
		DO $$ BEGIN
			CREATE TYPE order_status AS ENUM ('open', 'complete', 'canceled');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		-- DEX order state enum
		DO $$ BEGIN
			CREATE TYPE dex_order_state AS ENUM ('future', 'locked', 'complete');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		-- DEX deposit state enum
		DO $$ BEGIN
			CREATE TYPE dex_deposit_state AS ENUM ('pending', 'locked', 'complete');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		-- DEX withdrawal state enum
		DO $$ BEGIN
			CREATE TYPE dex_withdrawal_state AS ENUM ('pending', 'locked', 'complete');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
	`

	return db.Exec(ctx, query)
}

package chain

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// initCommittees creates the committees table matching indexer.Committee
// This matches pkg/db/models/indexer/committee.go:32-48 (8 fields)
func (db *DB) initCommittees(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS %s.committees (
			chain_id SMALLINT NOT NULL,                    -- UInt16 -> SMALLINT (renamed to chain_id, crosschain uses committee_chain_id)
			last_root_height_updated BIGINT NOT NULL DEFAULT 0,
			last_chain_height_updated BIGINT NOT NULL DEFAULT 0,
			number_of_samples BIGINT NOT NULL DEFAULT 0,
			subsidized BOOLEAN NOT NULL DEFAULT false,     -- UInt8 (0/1) -> BOOLEAN
			retired BOOLEAN NOT NULL DEFAULT false,        -- UInt8 (0/1) -> BOOLEAN
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (chain_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_committees_height ON committees(height);
	`

	return db.Exec(ctx, query)
}

// initCommitteeValidators creates the committee_validators table matching indexer.CommitteeValidator
// This matches pkg/db/models/indexer/committee_validator.go:48-67 (10 fields)
func (db *DB) initCommitteeValidators(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS %s.committee_validators (
			committee_id BIGINT NOT NULL,                  -- UInt64 -> BIGINT
			validator_address TEXT NOT NULL,               -- CrossChainRename: address
			staked_amount BIGINT NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',         -- LowCardinality(String)
			delegate BOOLEAN NOT NULL DEFAULT false,
			compound BOOLEAN NOT NULL DEFAULT false,
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			subsidized BOOLEAN NOT NULL DEFAULT false,     -- NEW: Denormalized from Committee
			retired BOOLEAN NOT NULL DEFAULT false,        -- NEW: Denormalized from Committee
			PRIMARY KEY (committee_id, validator_address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_committee_validators_height ON committee_validators(height);
		CREATE INDEX IF NOT EXISTS idx_committee_validators_address ON committee_validators(validator_address);
	`

	db.Logger.Debug("Executing SQL for committee_validators table",
		zap.String("table", "committee_validators"),
		zap.String("database", db.Name),
		zap.Uint64("chain_id", db.ChainID),
		zap.String("sql", query),
	)

	if err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("create committee_validators table in chain database (chain_id: %d, db: %s): SQL execution failed: %w", db.ChainID, db.Name, err)
	}

	return nil
}

// initCommitteePayments creates the committee_payments table matching indexer.CommitteePayment
// This matches pkg/db/models/indexer/committee_payment.go:25-31 (5 fields)
func (db *DB) initCommitteePayments(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS %s.committee_payments (
			committee_id BIGINT NOT NULL,                  -- UInt64 -> BIGINT
			address TEXT NOT NULL,
			percent BIGINT NOT NULL DEFAULT 0,             -- UInt64 -> BIGINT
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (committee_id, address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_committee_payments_height ON committee_payments(height);
		CREATE INDEX IF NOT EXISTS idx_committee_payments_address ON committee_payments(address);
	`

	return db.Exec(ctx, query)
}

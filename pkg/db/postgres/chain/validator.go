package chain

import (
	"context"
)

// initValidators creates the validators table
func (db *DB) initValidators(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS validators (
			address TEXT NOT NULL,
			height BIGINT NOT NULL,
			public_key TEXT NOT NULL,
			net_address TEXT NOT NULL,
			staked_amount BIGINT NOT NULL DEFAULT 0,
			max_paused_height BIGINT NOT NULL DEFAULT 0,
			unstaking_height BIGINT NOT NULL DEFAULT 0,
			output TEXT NOT NULL,
			delegate BOOLEAN NOT NULL DEFAULT false,
			compound BOOLEAN NOT NULL DEFAULT false,
			status TEXT NOT NULL DEFAULT 'active', -- active, paused, unstaking
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_validators_height ON validators(height);
		CREATE INDEX IF NOT NULL idx_validators_status ON validators(status);
	`

	return db.Exec(ctx, query)
}

// initValidatorNonSigningInfo creates the validator_non_signing_info table
func (db *DB) initValidatorNonSigningInfo(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS validator_non_signing_info (
			address TEXT NOT NULL,
			height BIGINT NOT NULL,
			missed_blocks_count BIGINT NOT NULL DEFAULT 0,
			last_signed_height BIGINT NOT NULL DEFAULT 0,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_validator_non_signing_info_height ON validator_non_signing_info(height);
	`

	return db.Exec(ctx, query)
}

// initValidatorDoubleSigningInfo creates the validator_double_signing_info table
func (db *DB) initValidatorDoubleSigningInfo(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS validator_double_signing_info (
			address TEXT NOT NULL,
			height BIGINT NOT NULL,
			evidence_count BIGINT NOT NULL DEFAULT 0,
			first_evidence_height BIGINT NOT NULL DEFAULT 0,
			last_evidence_height BIGINT NOT NULL DEFAULT 0,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_validator_double_signing_info_height ON validator_double_signing_info(height);
	`

	return db.Exec(ctx, query)
}

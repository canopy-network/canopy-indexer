package chain

import (
	"context"
	"fmt"
)

// initValidators creates the validators table
func (db *DB) initValidators(ctx context.Context) error {
	validatorsTable := db.SchemaTable("validators")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
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

		CREATE INDEX IF NOT EXISTS idx_validators_height ON %s(height);
		CREATE INDEX IF NOT EXISTS idx_validators_status ON %s(status);
	`, validatorsTable, validatorsTable, validatorsTable)

	return db.Exec(ctx, query)
}

// initValidatorNonSigningInfo creates the validator_non_signing_info table
func (db *DB) initValidatorNonSigningInfo(ctx context.Context) error {
	validatorNonSigningInfoTable := db.SchemaTable("validator_non_signing_info")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			address TEXT NOT NULL,
			height BIGINT NOT NULL,
			missed_blocks_count BIGINT NOT NULL DEFAULT 0,
			last_signed_height BIGINT NOT NULL DEFAULT 0,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_validator_non_signing_info_height ON %s(height);
	`, validatorNonSigningInfoTable, validatorNonSigningInfoTable)

	return db.Exec(ctx, query)
}

// initValidatorDoubleSigningInfo creates the validator_double_signing_info table
func (db *DB) initValidatorDoubleSigningInfo(ctx context.Context) error {
	validatorDoubleSigningInfoTable := db.SchemaTable("validator_double_signing_info")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			address TEXT NOT NULL,
			height BIGINT NOT NULL,
			evidence_count BIGINT NOT NULL DEFAULT 0,
			first_evidence_height BIGINT NOT NULL DEFAULT 0,
			last_evidence_height BIGINT NOT NULL DEFAULT 0,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (address, height)
		);

		CREATE INDEX IF NOT EXISTS idx_validator_double_signing_info_height ON %s(height);
	`, validatorDoubleSigningInfoTable, validatorDoubleSigningInfoTable)

	return db.Exec(ctx, query)
}

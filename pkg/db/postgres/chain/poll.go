package chain

import (
	"context"
)

// initPollSnapshots creates the poll_snapshots table matching indexer.PollSnapshot
// This matches pkg/db/models/indexer/poll_snapshot.go:47-76 (17 fields)
// NOTE: Poll snapshots are time-based, not height-based
func (db *DB) initPollSnapshots(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS poll_snapshots (
			proposal_hash TEXT NOT NULL,
			proposal_url TEXT DEFAULT '',
			accounts_approve_tokens BIGINT NOT NULL DEFAULT 0,
			accounts_reject_tokens BIGINT NOT NULL DEFAULT 0,
			accounts_total_voted_tokens BIGINT NOT NULL DEFAULT 0,
			accounts_total_tokens BIGINT NOT NULL DEFAULT 0,
			accounts_approve_percentage BIGINT NOT NULL DEFAULT 0,
			accounts_reject_percentage BIGINT NOT NULL DEFAULT 0,
			accounts_voted_percentage BIGINT NOT NULL DEFAULT 0,
			validators_approve_tokens BIGINT NOT NULL DEFAULT 0,
			validators_reject_tokens BIGINT NOT NULL DEFAULT 0,
			validators_total_voted_tokens BIGINT NOT NULL DEFAULT 0,
			validators_total_tokens BIGINT NOT NULL DEFAULT 0,
			validators_approve_percentage BIGINT NOT NULL DEFAULT 0,
			validators_reject_percentage BIGINT NOT NULL DEFAULT 0,
			validators_voted_percentage BIGINT NOT NULL DEFAULT 0,
			snapshot_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (proposal_hash, snapshot_time)
		);

		CREATE INDEX IF NOT EXISTS idx_poll_snapshots_time ON poll_snapshots(snapshot_time);
	`

	return db.Exec(ctx, query)
}

// initProposalSnapshots creates the proposal_snapshots table matching indexer.ProposalSnapshot
// This matches pkg/db/models/indexer/proposal_snapshot.go:50-76 (13 fields)
// NOTE: Proposal snapshots are time-based, not height-based
func (db *DB) initProposalSnapshots(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS proposal_snapshots (
			proposal_hash TEXT NOT NULL,
			proposal TEXT NOT NULL,                        -- JSON proposal data
			approve BOOLEAN NOT NULL DEFAULT false,
			snapshot_time TIMESTAMP WITH TIME ZONE NOT NULL,
			proposal_type TEXT DEFAULT '',                 -- LowCardinality(String)
			signer TEXT DEFAULT '',
			start_height BIGINT NOT NULL DEFAULT 0,
			end_height BIGINT NOT NULL DEFAULT 0,
			parameter_space TEXT DEFAULT '',
			parameter_key TEXT DEFAULT '',
			parameter_value TEXT DEFAULT '',
			dao_transfer_address TEXT DEFAULT '',
			dao_transfer_amount BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (proposal_hash, snapshot_time)
		);

		CREATE INDEX IF NOT EXISTS idx_proposal_snapshots_time ON proposal_snapshots(snapshot_time);
		CREATE INDEX IF NOT EXISTS idx_proposal_snapshots_type ON proposal_snapshots(proposal_type);
		CREATE INDEX IF NOT EXISTS idx_proposal_snapshots_signer ON proposal_snapshots(signer);
	`

	return db.Exec(ctx, query)
}

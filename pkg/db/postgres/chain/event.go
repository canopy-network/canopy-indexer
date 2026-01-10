package chain

import (
	"context"
)

// initEvents creates the events table
func (db *DB) initEvents(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS events (
			height BIGINT NOT NULL,
			chain_id SMALLINT NOT NULL,
			address TEXT NOT NULL,
			reference TEXT NOT NULL,
			event_type TEXT NOT NULL,
			block_height BIGINT NOT NULL,
			amount BIGINT DEFAULT 0,
			sold_amount BIGINT DEFAULT 0,
			bought_amount BIGINT DEFAULT 0,
			local_amount BIGINT DEFAULT 0,
			remote_amount BIGINT DEFAULT 0,
			success BOOLEAN DEFAULT false,
			local_origin BOOLEAN DEFAULT false,
			order_id TEXT DEFAULT '',
			points_received BIGINT DEFAULT 0,
			points_burned BIGINT DEFAULT 0,
			data TEXT DEFAULT '',
			seller_receive_address TEXT DEFAULT '',
			buyer_send_address TEXT DEFAULT '',
			sellers_send_address TEXT DEFAULT '',
			msg TEXT NOT NULL, -- JSON message data
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY (height, event_type, chain_id, address, reference)
		);

		CREATE INDEX IF NOT EXISTS idx_events_address ON events(address);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
		CREATE INDEX IF NOT EXISTS idx_events_time ON events(height_time);
	`

	return db.Exec(ctx, query)
}

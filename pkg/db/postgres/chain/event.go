package chain

import (
	"context"
	"fmt"
)

// initEvents creates the events table
func (db *DB) initEvents(ctx context.Context) error {
	eventsTable := db.SchemaTable("events")
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			height BIGINT NOT NULL,
			chain_id BIGINT NOT NULL,
			event_chain_id BIGINT NOT NULL,
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
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height, event_chain_id, address, reference, event_type)
		);

		CREATE INDEX IF NOT EXISTS idx_events_address ON %s(address);
		CREATE INDEX IF NOT EXISTS idx_events_type ON %s(event_type);
		CREATE INDEX IF NOT EXISTS idx_events_time ON %s(height_time);
	`, eventsTable, eventsTable, eventsTable, eventsTable)

	return db.Exec(ctx, query)
}

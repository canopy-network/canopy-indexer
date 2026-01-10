package chain

import (
	"context"
)

// initOrders creates the orders table matching indexer.Order
// This matches pkg/db/models/indexer/order.go:47-69 (12 fields)
func (db *DB) initOrders(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS orders (
			order_id TEXT NOT NULL,
			height BIGINT NOT NULL,
			height_time TIMESTAMP WITH TIME ZONE NOT NULL,
			committee SMALLINT NOT NULL,                 -- UInt16 -> SMALLINT
			data TEXT DEFAULT '',
			amount_for_sale BIGINT NOT NULL DEFAULT 0,
			requested_amount BIGINT NOT NULL DEFAULT 0,
			seller_receive_address TEXT DEFAULT '',
			buyer_send_address TEXT DEFAULT '',
			buyer_receive_address TEXT DEFAULT '',
			buyer_chain_deadline BIGINT NOT NULL DEFAULT 0,
			sellers_send_address TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',         -- LowCardinality(String): open, complete, canceled
			PRIMARY KEY (order_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_orders_height ON orders(height);
		CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
		CREATE INDEX IF NOT EXISTS idx_orders_committee ON orders(committee);
	`

	return db.Exec(ctx, query)
}

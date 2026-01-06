package indexer

import (
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeEvents(batch *pgx.Batch, data *BlockData) {
	for _, event := range data.Events {
		msgJSON, _ := json.Marshal(event)

		// Extract common fields based on event type
		var address string
		var eventType string
		var amount *uint64

		// TODO: Extract fields from event based on its type
		// This requires inspecting the oneof field in lib.Event

		batch.Queue(`
			INSERT INTO events (
				chain_id, height, height_time, event_chain_id, address,
				reference, event_type, block_height, amount, msg
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			data.ChainID,
			data.Height,
			data.BlockTime,
			data.ChainID, // event_chain_id
			address,      // extract from event
			"",           // reference
			eventType,    // extract from event type
			data.Height,
			amount,
			msgJSON,
		)
	}
}

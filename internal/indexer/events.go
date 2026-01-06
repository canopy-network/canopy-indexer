package indexer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) indexEvents(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	events, err := rpc.EventsByHeight(ctx, height)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, event := range events {
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
			chainID,
			height,
			blockTime,
			chainID,   // event_chain_id
			address,   // extract from event
			"",        // reference
			eventType, // extract from event type
			height,
			amount,
			msgJSON,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range events {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

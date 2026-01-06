package indexer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) indexTransactions(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	txs, err := rpc.TxsByHeight(ctx, height)
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for i, tx := range txs {
		msgJSON, _ := json.Marshal(tx)

		batch.Queue(`
			INSERT INTO txs (
				chain_id, height, height_time, tx_hash, tx_index, tx_time,
				created_height, network_id, message_type, signer, fee, msg
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (chain_id, height, tx_hash) DO NOTHING
		`,
			chainID,
			height,
			blockTime,
			tx.TxHash,
			i,
			blockTime, // tx_time
			height,    // created_height
			0,         // network_id
			tx.MessageType,
			transform.BytesToHex(tx.Sender),
			0, // fee (extract from tx if available)
			msgJSON,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range txs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}
